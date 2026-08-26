#include "core/batch.h"

#include "core/image.h"
#include "core/material.h"
#include "core/games.h"
#include "core/strings.h"

#include <algorithm>
#include <array>
#include <semaphore>
#include <format>
#include <mutex>
#include <system_error>
#include <thread>

namespace fs = std::filesystem;

namespace bit
{
	namespace
	{
		int HardwareThreads()
		{
			const unsigned n = std::thread::hardware_concurrency();
			return n > 0 ? static_cast<int>(n) : 1;
		}

		int ClampCustomWorkers(int n)
		{
			return std::clamp(n, 1, 32);
		}
	}

	int AutonomousWorkerCount(SpeedMode mode, int resolutionClass, int customWorkers)
	{
		const int cpu = HardwareThreads();

		// Custom gives explicit control and, unlike Fast/Extreme, is not
		// silently reduced based on resolution.
		if (mode == SpeedMode::Custom)
			return ClampCustomWorkers(customWorkers);

		if (mode == SpeedMode::Slow || mode == SpeedMode::Normal)
			return 1;

		if (mode == SpeedMode::Fast)
		{
			int n = std::max(cpu / 2, 4);

			int cap = 8;
			if (resolutionClass >= 4096)      cap = 2;
			else if (resolutionClass >= 2048) cap = 4;
			else if (resolutionClass >= 1024) cap = 6;

			return std::min(n, cap);
		}

		int n = std::max(cpu, 8);

		int cap = 16;
		if (resolutionClass >= 4096)      cap = 4;
		else if (resolutionClass >= 2048) cap = 8;
		else if (resolutionClass >= 1024) cap = 12;

		return std::min(n, cap);
	}

	int AutonomousCompilerSlots(SpeedMode mode, int resolutionClass, int workers)
	{
		const int cpu = HardwareThreads();

		switch (mode)
		{
		case SpeedMode::Custom:
			return std::clamp(workers / 4, 1, 6);

		case SpeedMode::Fast:
			if (resolutionClass >= 4096) return 1;
			if (resolutionClass >= 2048) return 2;
			return std::max(std::min(3, cpu), 1);

		case SpeedMode::Extreme:
		{
			int cap = 6;
			if (resolutionClass >= 4096)      cap = 2;
			else if (resolutionClass >= 2048) cap = 3;
			else if (resolutionClass >= 1024) cap = 4;

			return std::min(std::max(cpu / 2, 2), cap);
		}

		default:
			return 1;
		}
	}

	int DetectResolutionClass(const std::vector<fs::path>& files)
	{
		std::vector<int> dims;
		dims.reserve(files.size());

		for (const fs::path& file : files)
		{
			auto image = LoadImageFile(file);
			if (!image)
				continue;

			dims.push_back(std::max(image->width, image->height));
		}

		if (dims.empty())
			return 1024;   // conservative fallback

		std::sort(dims.begin(), dims.end());

		size_t index = (dims.size() * 3) / 4;
		if (index >= dims.size())
			index = dims.size() - 1;

		const int d = dims[index];

		if (d <= 512)  return 512;
		if (d <= 1024) return 1024;
		if (d <= 2048) return 2048;
		return 4096;
	}

	std::vector<fs::path> CollectImages(const fs::path& folder)
	{
		static constexpr std::array<std::string_view, 7> kExtensions = {
			".png", ".jpg", ".jpeg", ".bmp", ".gif", ".tif", ".tiff"
		};

		std::vector<fs::path> files;
		std::error_code ec;

		for (const fs::directory_entry& entry : fs::directory_iterator(folder, ec))
		{
			if (!entry.is_regular_file(ec))
				continue;

			const std::string ext = entry.path().extension().string();

			const bool supported = std::any_of(kExtensions.begin(), kExtensions.end(),
				[&ext](std::string_view e) { return EqualsNoCase(ext, e); });

			if (supported)
				files.push_back(entry.path());
		}

		std::sort(files.begin(), files.end(), [](const fs::path& a, const fs::path& b) {
			return ToLower(a.filename().string()) < ToLower(b.filename().string());
		});

		return files;
	}

	BatchResult RunAutonomousProduction(const BatchRequest& request,
		const ProgressFn& progress, const std::atomic<bool>& cancel)
	{
		BatchResult result;
		result.total = static_cast<int>(request.files.size());

		if (request.files.empty())
			return result;

		const int resolutionClass = request.base.quality > 0
			? request.base.quality
			: DetectResolutionClass(request.files);

		const int workers = AutonomousWorkerCount(request.speed, resolutionClass, request.customWorkers);

		// Compiler slots are a separate, smaller budget: image work is CPU and
		// RAM bound, resourcecompiler is disk and shared-resource bound.
		const int compilerSlots = AutonomousCompilerSlots(request.speed, resolutionClass, workers);

		std::counting_semaphore<32> compileGate(std::max(compilerSlots, 1));

		std::atomic<size_t> next{ 0 };
		std::atomic<int> done{ 0 };

		std::mutex resultMutex;
		result.items.resize(request.files.size());

		const auto run = [&]() {
			// Each worker owns its own COM apartment for WIC.
			const ComScope com;

			for (;;)
			{
				const size_t index = next.fetch_add(1);
				if (index >= request.files.size())
					return;

				if (cancel.load())
					return;

				const fs::path& source = request.files[index];

				BatchItemResult item;
				item.source = source;

				auto name = CleanMaterialName(source.stem().string());
				if (!name)
				{
					item.message = name.error();

					std::scoped_lock lock(resultMutex);
					result.items[index] = std::move(item);
					continue;
				}

				CreateRequest job = request.base;
				job.imagePath = source;
				job.materialName = *name;

				// Skip mode leaves an existing material alone rather than
				// rebuilding it; Replace and Ask both fall through to a rebuild,
				// Ask having already been answered before the batch started.
				if (request.overwrite == OverwriteMode::Skip)
				{
					const fs::path addonContent = GameAddonDirectory(job.gameRoot, job.gameKey) / job.addon;
					const fs::path vmat = addonContent / "materials" / (*name + ".vmat");

					std::error_code ec;
					if (fs::exists(vmat, ec))
					{
						item.skipped = true;
						item.message = "already exists";

						std::scoped_lock lock(resultMutex);
						result.items[index] = std::move(item);

						const int n = done.fetch_add(1) + 1;
						if (progress)
							progress(std::format("{} / {} — skipped {}", n, result.total, source.filename().string()));
						continue;
					}
				}

				compileGate.acquire();
				const CreateResult created = CreateMaterial(job, nullptr);
				compileGate.release();

				item.ok = created.ok;
				item.message = created.ok ? created.status : created.error;

				{
					std::scoped_lock lock(resultMutex);
					result.items[index] = std::move(item);
				}

				const int n = done.fetch_add(1) + 1;
				if (progress)
				{
					progress(std::format("{} / {} — {} {}", n, result.total,
						created.ok ? "done" : "failed", source.filename().string()));
				}
			}
		};

		std::vector<std::thread> pool;
		pool.reserve(static_cast<size_t>(workers));

		for (int i = 0; i < workers; ++i)
			pool.emplace_back(run);

		for (std::thread& t : pool)
			t.join();

		result.cancelled = cancel.load();

		for (const BatchItemResult& item : result.items)
		{
			if (item.skipped)      ++result.skipped;
			else if (item.ok)      ++result.succeeded;
			else if (!item.source.empty()) ++result.failed;
		}

		return result;
	}
}
