#include "core/pipeline.h"

#include "core/compiler.h"
#include "core/games.h"
#include "core/image.h"
#include "core/strings.h"

#include <format>
#include <fstream>
#include <system_error>
#include <vector>

namespace fs = std::filesystem;

namespace bit
{
	namespace
	{
		void Report(const ProgressFn& progress, std::string line)
		{
			if (progress)
				progress(std::move(line));
		}

		bool WriteTextFile(const fs::path& path, std::string_view contents, std::string& error)
		{
			std::ofstream file(path, std::ios::binary | std::ios::trunc);
			if (!file)
			{
				error = std::format("could not open {} for writing", path.string());
				return false;
			}

			file.write(contents.data(), static_cast<std::streamsize>(contents.size()));
			if (!file)
			{
				error = std::format("could not write {}", path.string());
				return false;
			}

			return true;
		}

		CreateResult Fail(std::string status, std::string error)
		{
			CreateResult result;
			result.status = std::move(status);
			result.error = std::move(error);
			return result;
		}
	}

	CreateResult CreateMaterial(const CreateRequest& request, const ProgressFn& progress)
	{
		// WIC lives on this thread for the duration of the job.
		const ComScope com;
		if (!com.ok())
			return Fail("Could not start the imaging system.", "CoInitializeEx failed on the worker thread.");

		if (request.addon.empty())
			return Fail("No addon selected.", "Pick the addon the material should land in.");

		const std::string addon = SanitizeAddon(request.addon);
		if (addon.empty())
			return Fail("Invalid addon name.", "The selected addon name contains characters that cannot appear in a path.");

		// materials/<dirs...>/<leaf>
		std::vector<std::string> parts;
		for (size_t start = 0; start <= request.materialName.size();)
		{
			const size_t slash = request.materialName.find('/', start);
			const size_t end = slash == std::string::npos ? request.materialName.size() : slash;

			if (end > start)
				parts.push_back(request.materialName.substr(start, end - start));

			if (slash == std::string::npos)
				break;

			start = slash + 1;
		}

		if (parts.empty())
			return Fail("No material name.", "Enter a name for the material.");

		const std::string leaf = parts.back();
		parts.pop_back();

		const fs::path addonContent = GameAddonDirectory(request.gameRoot, request.gameKey) / addon;

		std::error_code ec;
		if (!fs::is_directory(addonContent, ec))
		{
			return Fail("Addon not found.", std::format(
				"Addon not found in Workshop Tools:\n{}\n\nOpen or create it in the Workshop Tools first.",
				addonContent.string()));
		}

		fs::path destDir = addonContent / "materials";
		std::string relPrefix = "materials";

		for (const std::string& part : parts)
		{
			destDir /= part;
			relPrefix += "/" + part;
		}

		fs::create_directories(destDir, ec);
		if (ec)
			return Fail("Could not create the material folder.", std::format("{}:\n{}", destDir.string(), ec.message()));

		Report(progress, std::format("Loading {}...", request.imagePath.filename().string()));

		auto loaded = LoadImageFile(request.imagePath);
		if (!loaded)
			return Fail("Could not read the image.", loaded.error());

		Image image = std::move(*loaded);

		if (!image.Square())
		{
			Report(progress, std::format("Cropping {} x {} to square...", image.width, image.height));
			image = CropToSquare(image);
		}

		if (request.quality > 0 && request.quality != image.width)
		{
			Report(progress, std::format("Resizing to {0} x {0}...", request.quality));

			auto resized = Resize(image, request.quality, request.quality);
			if (!resized)
				return Fail("Could not resize the image.", resized.error());

			image = std::move(*resized);
		}

		const fs::path pngPath = destDir / (leaf + "_color.png");
		const fs::path vtexPath = destDir / (leaf + "_color.vtex");
		const fs::path vmatPath = destDir / (leaf + ".vmat");

		Report(progress, std::format("Saving {0} x {0} PNG...", image.width));

		if (auto saved = SavePNG(image, pngPath); !saved)
			return Fail("Could not save the texture.", saved.error());

		// Free the pixels before the compiler runs -- a 4096 RGBA buffer is 64 MB.
		image = {};

		const std::string pngResource = std::format("{}/{}", relPrefix, pngPath.filename().string());

		// The VMAT has to reference the *compiled* texture (.vtex_c), not the
		// .vtex descriptor. Resource Compiler writes the .vtex_c into
		// game/<addon> at the matching resource path.
		const std::string vtexResource = std::format("{}/{}_color.vtex_c", relPrefix, leaf);

		std::string error;

		Report(progress, "Writing VTEX...");
		if (!WriteTextFile(vtexPath, MakeVTEX(pngResource), error))
			return Fail("Could not write the VTEX.", error);

		Report(progress, "Writing VMAT...");
		if (!WriteTextFile(vmatPath, MakeVMAT(request.mode, request.alphaRef, vtexResource), error))
			return Fail("Could not write the VMAT.", error);

		CreateResult result;
		result.outputDir = destDir;

		if (!request.compile)
		{
			result.ok = true;
			result.status = "Done. Material source files created; compilation was skipped.";
			return result;
		}

		Report(progress, "Compiling with Workshop Tools...");

		const CompileResult compiled = CompileTargets(request.gameRoot,
			request.compilerLock, request.retryCompile, { vtexPath, vmatPath });

		result.logPath = destDir / (leaf + "_compile_log.txt");

		std::string logError;
		WriteTextFile(result.logPath, compiled.log, logError);

		if (!compiled.ok)
		{
			result.status = "Compile failed. See the compile log.";
			result.error = std::format(
				"The PNG/VTEX/VMAT source files were written, but Resource Compiler returned an error.\n\nLog:\n{}",
				result.logPath.string());
			return result;
		}

		result.ok = true;
		result.status = "Done. Material compiled successfully.";
		return result;
	}
}
