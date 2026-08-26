#include "ui/app.h"

#include "core/batch.h"
#include "core/games.h"
#include "core/image.h"
#include "core/pipeline.h"
#include "core/settings.h"
#include "core/strings.h"
#include "i18n/translations.h"
#include "ui/texture.h"
#include "ui/theme.h"

#include "resource.h"

#include "imgui.h"

#include <windows.h>
#include <shobjidl.h>
#include <wrl/client.h>

#include <algorithm>
#include <array>
#include <format>
#include <fstream>
#include <iterator>
#include <optional>

using Microsoft::WRL::ComPtr;
namespace fs = std::filesystem;

namespace bit
{
	AppState::~AppState()
	{
		if (worker && worker->joinable())
			worker->join();
	}

	namespace
	{
		void SetStatus(AppState& state, std::string line)
		{
			std::scoped_lock lock(state.mutex);
			state.statusLine = std::move(line);
		}

		// IFileOpenDialog for both files and folders -- one API, and it is the
		// picker users already know from every other Windows app.
		std::optional<fs::path> RunPicker(bool folders, const std::wstring& title, const fs::path& startAt)
		{
			ComPtr<IFileOpenDialog> dialog;
			if (FAILED(::CoCreateInstance(CLSID_FileOpenDialog, nullptr, CLSCTX_INPROC_SERVER, IID_PPV_ARGS(&dialog))))
				return std::nullopt;

			if (folders)
			{
				DWORD options = 0;
				dialog->GetOptions(&options);
				dialog->SetOptions(options | FOS_PICKFOLDERS);
			}
			else
			{
				static constexpr COMDLG_FILTERSPEC kFilters[] = {
					{ L"Images", L"*.png;*.jpg;*.jpeg;*.bmp;*.gif;*.tif;*.tiff" },
					{ L"All files", L"*.*" },
				};
				dialog->SetFileTypes(static_cast<UINT>(std::size(kFilters)), kFilters);
			}

			dialog->SetTitle(title.c_str());

			if (!startAt.empty())
			{
				ComPtr<IShellItem> folder;
				if (SUCCEEDED(::SHCreateItemFromParsingName(startAt.c_str(), nullptr, IID_PPV_ARGS(&folder))))
					dialog->SetFolder(folder.Get());
			}

			if (FAILED(dialog->Show(nullptr)))
				return std::nullopt;

			ComPtr<IShellItem> item;
			if (FAILED(dialog->GetResult(&item)))
				return std::nullopt;

			PWSTR raw = nullptr;
			if (FAILED(item->GetDisplayName(SIGDN_FILESYSPATH, &raw)))
				return std::nullopt;

			fs::path path(raw);
			::CoTaskMemFree(raw);

			return path;
		}

		CreateRequest BuildRequest(const AppState& state)
		{
			CreateRequest request;
			request.gameRoot = state.gameRoot;
			request.gameKey = state.settings.game;
			request.addon = state.selectedAddon >= 0
				? state.addons[static_cast<size_t>(state.selectedAddon)]
				: "";
			request.mode = static_cast<MaterialMode>(state.settings.materialMode);
			request.alphaRef = state.alphaRef;
			request.quality = state.settings.quality;
			request.compilerLock = state.settings.compilerLock;
			request.retryCompile = state.settings.retryCompile;
			return request;
		}
	}

	namespace
	{
		// Language code -> the flag RCDATA id in resource/app.rc.
		const std::unordered_map<std::string, int>& FlagResources()
		{
			static const std::unordered_map<std::string, int> flags = {
				{ "en", IDR_FLAG_GB }, { "ru", IDR_FLAG_RU }, { "cs", IDR_FLAG_CZ },
				{ "pt-BR", IDR_FLAG_BR }, { "fr", IDR_FLAG_FR }, { "de", IDR_FLAG_DE },
				{ "es", IDR_FLAG_ES }, { "pl", IDR_FLAG_PL }, { "tr", IDR_FLAG_TR },
			};
			return flags;
		}

		void LoadArtwork(AppState& state)
		{
			if (state.artworkLoaded)
				return;

			state.artworkLoaded = true;

			if (auto light = LoadImageResource(IDR_LOGO_LIGHT))
				state.logoLight.Upload(*light);

			if (auto dark = LoadImageResource(IDR_LOGO_DARK))
				state.logoDark.Upload(*dark);

			for (const auto& [code, id] : FlagResources())
			{
				if (auto flag = LoadImageResource(id))
				{
					GpuTexture texture;
					if (texture.Upload(*flag))
						state.flags.emplace(code, std::move(texture));
				}
			}
		}

		// Flags are wider than they are tall; scale to a fixed height so the
		// rows line up regardless of the source artwork's size.
		void DrawFlag(const AppState& state, std::string_view code, float height)
		{
			const auto it = state.flags.find(std::string(code));
			if (it == state.flags.end() || !it->second.Valid())
			{
				ImGui::Dummy(ImVec2(height * 1.5f, height));
				return;
			}

			const GpuTexture& texture = it->second;
            const float width = height * static_cast<float>(texture.width())
                / static_cast<float>(std::max(texture.height(), 1));

			ImGui::Image((ImTextureID)(intptr_t)texture.Handle(), ImVec2(width, height));
		}
	}

	void InitApp(AppState& state)
	{
		state.settings = LoadSettings();

		SetLanguage(state.settings.language);
		state.themeDirty = true;

		if (!ValidGameKey(state.settings.game))
			state.settings.game = "cs2";

		// A remembered root beats re-probing every Steam library on startup.
		if (!state.settings.cs2Root.empty() && ValidGameRoot(state.settings.cs2Root, state.settings.game))
		{
			state.gameRoot = state.settings.cs2Root;
			state.gameRootValid = true;
			state.addons = EnumerateAddons(state.gameRoot, state.settings.game);
		}
		else
		{
			RefreshGameRoot(state);
		}

		if (!state.settings.lastAddon.empty())
		{
			const auto it = std::find(state.addons.begin(), state.addons.end(), state.settings.lastAddon);
			if (it != state.addons.end())
				state.selectedAddon = static_cast<int>(std::distance(state.addons.begin(), it));
		}

		if (state.selectedAddon < 0 && !state.addons.empty())
			state.selectedAddon = 0;
	}

	void ShutdownApp(AppState& state)
	{
		state.cancelBatch = true;

		if (state.worker && state.worker->joinable())
			state.worker->join();

		// Drop the GPU resources while the device is still alive.
		state.preview.Reset();
		state.logoLight.Reset();
		state.logoDark.Reset();
		state.flags.clear();

		state.settings.cs2Root = state.gameRoot.string();
		state.settings.lastAddon = state.selectedAddon >= 0
			? state.addons[static_cast<size_t>(state.selectedAddon)]
			: "";
		state.settings.language = std::string(CurrentLanguage());

		SaveSettings(state.settings);
	}

	void RefreshGameRoot(AppState& state)
	{
		state.gameRoot = DetectGameRoot(state.settings.game);
		state.gameRootValid = !state.gameRoot.empty();

		state.addons = state.gameRootValid
			? EnumerateAddons(state.gameRoot, state.settings.game)
			: std::vector<std::string>{};

		state.selectedAddon = state.addons.empty() ? -1 : 0;

		SetStatus(state, state.gameRootValid
			? std::format("{}: {}", tr("detected"), state.gameRoot.string())
			: tr("not_found"));
	}

	void ChooseImage(AppState& state)
	{
		const auto picked = RunPicker(false, Widen(tr("choose_image")), state.settings.lastImageDir);
		if (!picked)
			return;

		state.settings.lastImageDir = picked->parent_path().string();

		auto loaded = LoadImageFile(*picked);
		if (!loaded)
		{
			SetStatus(state, std::format("{}: {}", picked->filename().string(), loaded.error()));
			return;
		}

		state.imagePath = *picked;
		state.sourceWidth = loaded->width;
		state.sourceHeight = loaded->height;
		state.sourceHasAlpha = HasAlpha(*loaded);

		// The preview is what the material will actually be built from, so it
		// gets the same centre crop the pipeline applies.
		state.preview.Upload(CropToSquare(*loaded));

		// A transparent source almost always wants an alpha shader, so nudge
		// the mode rather than silently producing an opaque material.
		if (state.sourceHasAlpha && state.settings.materialMode == static_cast<int>(MaterialMode::Opaque))
			state.settings.materialMode = static_cast<int>(MaterialMode::Cutout);

		if (state.materialName.empty())
		{
			if (auto cleaned = CleanMaterialName(picked->stem().string()))
				state.materialName = *cleaned;
		}

		SetStatus(state, std::format("{} — {} x {} — {}",
			picked->filename().string(), state.sourceWidth, state.sourceHeight,
			state.sourceHasAlpha ? tr("alpha_found") : tr("alpha_none")));
	}

	void ChooseBatchFolder(AppState& state)
	{
		const auto picked = RunPicker(true, Widen(tr("autonomous")), state.settings.lastImageDir);
		if (!picked)
			return;

		state.batchFolder = *picked;
		state.batchFiles = CollectImages(*picked);
		state.batchFinished = false;

		SetStatus(state, std::format("{}: {} images", picked->filename().string(), state.batchFiles.size()));
	}

	namespace
	{
		void JoinPreviousWorker(AppState& state)
		{
			if (state.worker && state.worker->joinable())
				state.worker->join();
		}
	}

	void StartCreate(AppState& state)
	{
		if (state.busy.exchange(true))
			return;

		auto cleaned = CleanMaterialName(state.materialName);
		if (!cleaned)
		{
			SetStatus(state, cleaned.error());
			state.busy = false;
			return;
		}

		CreateRequest request = BuildRequest(state);
		request.imagePath = state.imagePath;
		request.materialName = *cleaned;

		JoinPreviousWorker(state);

		{
			std::scoped_lock lock(state.mutex);
			state.compileLog.clear();
		}

		state.progressTotal = 1;
		state.progressDone = 0;
		state.batchFinished = false;

		state.worker = std::make_unique<std::thread>([&state, request]() {
			const CreateResult result = CreateMaterial(request,
				[&state](std::string line) { SetStatus(state, std::move(line)); });

			{
				std::scoped_lock lock(state.mutex);
				state.statusLine = result.status;
				state.lastOutputDir = result.outputDir;
				state.compileLog = result.error;

				if (!result.logPath.empty())
				{
					std::ifstream log(result.logPath, std::ios::binary);
					if (log)
					{
						state.compileLog.assign(std::istreambuf_iterator<char>(log),
							std::istreambuf_iterator<char>());
					}
				}
			}

			state.progressDone = 1;
			state.busy = false;
		});
	}

	void StartBatch(AppState& state)
	{
		if (state.batchFiles.empty())
			return;

		if (state.busy.exchange(true))
			return;

		BatchRequest request;
		request.base = BuildRequest(state);
		request.files = state.batchFiles;
		request.speed = static_cast<SpeedMode>(state.settings.autoMode);
		request.overwrite = static_cast<OverwriteMode>(state.settings.overwriteMode);
		request.customWorkers = state.settings.customWorkers > 0 ? state.settings.customWorkers : 4;

		JoinPreviousWorker(state);

		state.cancelBatch = false;
		state.batchFinished = false;
		state.progressTotal = static_cast<int>(state.batchFiles.size());
		state.progressDone = 0;

		{
			std::scoped_lock lock(state.mutex);
			state.compileLog.clear();
		}

		state.worker = std::make_unique<std::thread>([&state, request]() {
			const BatchResult result = RunAutonomousProduction(request,
				[&state](std::string line) {
					state.progressDone.fetch_add(1);
					SetStatus(state, std::move(line));
				},
				state.cancelBatch);

			{
				std::scoped_lock lock(state.mutex);
				state.batchResult = result;
				state.statusLine = std::format("{} ok, {} skipped, {} failed{}",
					result.succeeded, result.skipped, result.failed,
					result.cancelled ? " (stopped)" : "");

				std::string log;
				for (const BatchItemResult& item : result.items)
				{
					if (item.source.empty())
						continue;

					log += std::format("{:<40} {}\r\n", item.source.filename().string(),
						item.skipped ? "skipped" : (item.ok ? "ok" : item.message));
				}
				state.compileLog = std::move(log);
			}

			state.batchFinished = true;
			state.busy = false;
		});
	}

	namespace
	{
		void DrawTopBar(AppState& state)
		{
			const GpuTexture& logo = state.settings.darkMode ? state.logoDark : state.logoLight;
			if (logo.Valid())
			{
				constexpr float kLogoHeight = 44.0f;
				const float width = kLogoHeight * static_cast<float>(logo.width())
					/ static_cast<float>(std::max(logo.height(), 1));

				ImGui::Image((ImTextureID)(intptr_t)logo.Handle(), ImVec2(width, kLogoHeight));
				ImGui::SameLine();
			}

			const auto& languages = Languages();
			const int langIndex = CurrentLanguageIndex();
			const float flagHeight = ImGui::GetTextLineHeight();

			ImGui::SetNextItemWidth(200.0f);
			if (ImGui::BeginCombo(tr("language"), std::string(languages[langIndex].name).c_str()))
			{
				for (size_t i = 0; i < languages.size(); ++i)
				{
					const bool selected = static_cast<int>(i) == langIndex;

					ImGui::PushID(static_cast<int>(i));
					DrawFlag(state, languages[i].code, flagHeight);
					ImGui::SameLine();

					if (ImGui::Selectable(std::string(languages[i].name).c_str(), selected))
					{
						SetLanguage(languages[i].code);
						state.settings.language = std::string(languages[i].code);
					}
					if (selected)
						ImGui::SetItemDefaultFocus();
					ImGui::PopID();
				}
				ImGui::EndCombo();
			}

			ImGui::SameLine();
			DrawFlag(state, languages[langIndex].code, flagHeight);

			ImGui::SameLine();
			if (ImGui::Checkbox(tr(state.settings.darkMode ? "dark" : "light"), &state.settings.darkMode))
				state.themeDirty = true;

			ImGui::SameLine();
			if (ImGui::SmallButton(tr("clear_junk")))
			{
				const int removed = ClearJunkFolder();
				SetStatus(state, removed > 0 ? tr("junk_cleared") : tr("junk_empty"));
			}
		}

		void DrawGameSection(AppState& state)
		{
			ImGui::SeparatorText(tr("step_cs2"));

			const auto& profiles = GameProfiles();
			const int index = GameIndexForKey(state.settings.game);

			ImGui::SetNextItemWidth(220.0f);
			if (ImGui::BeginCombo(tr("game"), std::string(profiles[index].name).c_str()))
			{
				for (size_t i = 0; i < profiles.size(); ++i)
				{
					const bool selected = static_cast<int>(i) == index;
					if (ImGui::Selectable(std::string(profiles[i].name).c_str(), selected))
					{
						state.settings.game = std::string(profiles[i].key);
						RefreshGameRoot(state);
					}
					if (selected)
						ImGui::SetItemDefaultFocus();
				}
				ImGui::EndCombo();
			}

			ImGui::SameLine();
			if (ImGui::Button(tr("detect_cs2")))
				RefreshGameRoot(state);

			ImGui::SameLine();
			if (ImGui::Button(tr("browse")))
			{
				if (const auto picked = RunPicker(true, Widen(tr("detect_cs2")), state.gameRoot))
				{
					state.gameRoot = *picked;
					state.gameRootValid = ValidGameRoot(*picked, state.settings.game);
					state.addons = state.gameRootValid ? EnumerateAddons(*picked, state.settings.game) : std::vector<std::string>{};
					state.selectedAddon = state.addons.empty() ? -1 : 0;

					SetStatus(state, state.gameRootValid ? tr("detected") : tr("invalid_root"));
				}
			}

			if (state.gameRootValid)
				ImGui::TextDisabled("%s", state.gameRoot.string().c_str());
			else
				ImGui::TextColored(ImVec4(0.90f, 0.45f, 0.35f, 1.0f), "%s", tr("not_found"));

			// Addon
			if (state.addons.empty())
			{
				ImGui::TextDisabled("%s", tr("no_addons"));
				return;
			}

			const char* preview = state.selectedAddon >= 0
				? state.addons[static_cast<size_t>(state.selectedAddon)].c_str()
				: "";

			ImGui::SetNextItemWidth(220.0f);
			if (ImGui::BeginCombo(tr("addon"), preview))
			{
				for (size_t i = 0; i < state.addons.size(); ++i)
				{
					const bool selected = static_cast<int>(i) == state.selectedAddon;
					if (ImGui::Selectable(state.addons[i].c_str(), selected))
						state.selectedAddon = static_cast<int>(i);
					if (selected)
						ImGui::SetItemDefaultFocus();
				}
				ImGui::EndCombo();
			}

			ImGui::SameLine();
			if (ImGui::Button(tr("refresh")))
			{
				state.addons = EnumerateAddons(state.gameRoot, state.settings.game);
				if (state.selectedAddon >= static_cast<int>(state.addons.size()))
					state.selectedAddon = state.addons.empty() ? -1 : 0;
			}
		}

		void DrawMaterialSection(AppState& state)
		{
			ImGui::SeparatorText(tr("step_image"));

			if (ImGui::Button(tr("choose_image")))
				ChooseImage(state);

			ImGui::SameLine();
			if (state.imagePath.empty())
			{
				ImGui::TextDisabled("%s", tr("no_image"));
			}
			else
			{
				ImGui::Text("%s", state.imagePath.filename().string().c_str());

				if (state.preview.Valid())
				{
					constexpr float kPreview = 128.0f;

					// Checkerboard behind the preview so an alpha cutout reads as
					// transparent instead of blending into the window colour.
					const ImVec2 origin = ImGui::GetCursorScreenPos();
					ImDrawList* draw = ImGui::GetWindowDrawList();

					constexpr float kCell = 8.0f;
					for (int y = 0; y * kCell < kPreview; ++y)
					{
						for (int x = 0; x * kCell < kPreview; ++x)
						{
							const bool dark = ((x + y) & 1) != 0;
							const ImVec2 a(origin.x + x * kCell, origin.y + y * kCell);
							const ImVec2 b(std::min(a.x + kCell, origin.x + kPreview),
								std::min(a.y + kCell, origin.y + kPreview));

							draw->AddRectFilled(a, b, dark ? IM_COL32(80, 80, 86, 255) : IM_COL32(110, 110, 118, 255));
						}
					}

					ImGui::Image((ImTextureID)(intptr_t)state.preview.Handle(), ImVec2(kPreview, kPreview));
					ImGui::SameLine();
				}

				ImGui::BeginGroup();
				ImGui::TextDisabled("%d x %d", state.sourceWidth, state.sourceHeight);
				ImGui::TextDisabled("%s", state.sourceHasAlpha ? tr("alpha_found") : tr("alpha_none"));

				if (state.sourceWidth != state.sourceHeight)
					ImGui::TextDisabled("%s", tr("center"));

				ImGui::EndGroup();
			}

			ImGui::SeparatorText(tr("step_vmat"));

			std::array<char, 260> nameBuffer = {};
			state.materialName.copy(nameBuffer.data(), std::min(state.materialName.size(), nameBuffer.size() - 1));
			ImGui::SetNextItemWidth(360.0f);
			if (ImGui::InputText(tr("create_vmat"), nameBuffer.data(), nameBuffer.size()))
				state.materialName = nameBuffer.data();

			ImGui::SeparatorText(tr("step_type"));

			const std::array<const char*, 3> modes = { tr("mat_opaque"), tr("mat_cutout"), tr("mat_translucent") };

			ImGui::SetNextItemWidth(360.0f);
			ImGui::Combo("##type", &state.settings.materialMode, modes.data(), static_cast<int>(modes.size()));

			if (state.settings.materialMode == static_cast<int>(MaterialMode::Cutout))
			{
				ImGui::SetNextItemWidth(360.0f);
				ImGui::SliderFloat("##alpha", &state.alphaRef, 0.01f, 0.99f, "%.2f");
			}

			ImGui::SeparatorText(tr("step_quality"));

			const std::array<const char*, 5> qualityLabels = {
				tr("quality_original"), tr("quality_low"), tr("quality_mid"),
				tr("quality_high"), tr("quality_hd")
			};
			static constexpr std::array<int, 5> kQualityValues = { 0, 512, 1024, 2048, 4096 };

			int qualityIndex = 0;
			for (size_t i = 0; i < kQualityValues.size(); ++i)
			{
				if (kQualityValues[i] == state.settings.quality)
					qualityIndex = static_cast<int>(i);
			}

			ImGui::SetNextItemWidth(360.0f);
			if (ImGui::Combo("##quality", &qualityIndex, qualityLabels.data(), static_cast<int>(qualityLabels.size())))
				state.settings.quality = kQualityValues[static_cast<size_t>(qualityIndex)];

			ImGui::Checkbox(tr("compile_after"), &state.settings.compilerLock);
			ImGui::SameLine();
			ImGui::Checkbox(tr("retry_compile"), &state.settings.retryCompile);
		}

		void DrawBatchSection(AppState& state)
		{
			ImGui::SeparatorText(tr("autonomous"));

			if (ImGui::Button(tr("browse")))
				ChooseBatchFolder(state);

			ImGui::SameLine();
			if (state.batchFolder.empty())
				ImGui::TextDisabled("—");
			else
				ImGui::Text("%s (%zu)", state.batchFolder.filename().string().c_str(), state.batchFiles.size());

			const std::array<const char*, 5> speeds = {
				tr("auto_slow"), tr("auto_normal"), tr("auto_fast"), tr("auto_extreme"), tr("auto_custom")
			};

			ImGui::SetNextItemWidth(240.0f);
			ImGui::Combo(tr("auto_speed"), &state.settings.autoMode, speeds.data(), static_cast<int>(speeds.size()));

			if (state.settings.autoMode == static_cast<int>(SpeedMode::Custom))
			{
				ImGui::SetNextItemWidth(240.0f);
				if (ImGui::SliderInt(tr("custom_workers"), &state.settings.customWorkers, 1, 32))
					state.settings.customWorkers = std::clamp(state.settings.customWorkers, 1, 32);
			}

			const std::array<const char*, 3> overwrites = {
				tr("overwrite_ask"), tr("overwrite_skip"), tr("overwrite_replace")
			};

			ImGui::SetNextItemWidth(240.0f);
			ImGui::Combo(tr("overwrite"), &state.settings.overwriteMode, overwrites.data(), static_cast<int>(overwrites.size()));
		}

		void DrawLogSection(AppState& state)
		{
			ImGui::SeparatorText(tr("output"));

			std::string status;
			std::string log;
			{
				std::scoped_lock lock(state.mutex);
				status = state.statusLine;
				log = state.compileLog;
			}

			const int total = state.progressTotal.load();
			const int done = state.progressDone.load();

			if (state.busy && total > 1)
			{
				ImGui::ProgressBar(static_cast<float>(done) / static_cast<float>(total),
					ImVec2(-1.0f, 0.0f), std::format("{} / {}", done, total).c_str());
			}

			if (!status.empty())
				ImGui::TextWrapped("%s", status.c_str());

			if (!state.lastOutputDir.empty() && !state.busy)
			{
				if (ImGui::SmallButton(tr("open_output")))
					::ShellExecuteW(nullptr, L"open", state.lastOutputDir.c_str(), nullptr, nullptr, SW_SHOWNORMAL);
			}

			ImGui::BeginChild("##log", ImVec2(0.0f, 0.0f), ImGuiChildFlags_Border);
			ImGui::TextUnformatted(log.c_str());
			ImGui::EndChild();
		}
	}

	void DrawApp(AppState& state)
	{
		// Deferred to the first frame: InitApp runs before the D3D11 device is
		// handed over, and these need it.
		LoadArtwork(state);

		if (state.themeDirty)
		{
			ApplyTheme(state.settings.darkMode);
			state.themeDirty = false;
		}

		const ImGuiViewport* viewport = ImGui::GetMainViewport();
		ImGui::SetNextWindowPos(viewport->WorkPos);
		ImGui::SetNextWindowSize(viewport->WorkSize);

		constexpr ImGuiWindowFlags flags = ImGuiWindowFlags_NoDecoration
			| ImGuiWindowFlags_NoMove
			| ImGuiWindowFlags_NoSavedSettings
			| ImGuiWindowFlags_NoBringToFrontOnFocus;

		if (ImGui::Begin("B.I.T. Texture Tool", nullptr, flags))
		{
			const bool busy = state.busy;

			DrawTopBar(state);

			ImGui::BeginDisabled(busy);
			DrawGameSection(state);
			DrawMaterialSection(state);
			DrawBatchSection(state);
			ImGui::EndDisabled();

			ImGui::Spacing();

			const bool haveTarget = state.gameRootValid && state.selectedAddon >= 0;
			const bool canCreate = !busy && haveTarget && !state.imagePath.empty() && !state.materialName.empty();
			const bool canBatch = !busy && haveTarget && !state.batchFiles.empty();

			ImGui::BeginDisabled(!canCreate);
			if (ImGui::Button(tr("create_vmat"), ImVec2(-1.0f, 0.0f)))
				StartCreate(state);
			ImGui::EndDisabled();

			if (busy && state.progressTotal > 1)
			{
				if (ImGui::Button(tr("stop_auto"), ImVec2(-1.0f, 0.0f)))
				{
					state.cancelBatch = true;
					SetStatus(state, tr("stopping"));
				}
			}
			else
			{
				ImGui::BeginDisabled(!canBatch);
				if (ImGui::Button(tr("autonomous"), ImVec2(-1.0f, 0.0f)))
					StartBatch(state);
				ImGui::EndDisabled();
			}

			ImGui::Spacing();

			DrawLogSection(state);
		}
		ImGui::End();
	}
}
