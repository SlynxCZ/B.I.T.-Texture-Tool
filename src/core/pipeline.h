#pragma once

#include "core/material.h"

#include <filesystem>
#include <functional>
#include <string>

namespace bit
{
	struct CreateRequest
	{
		std::filesystem::path gameRoot;
		std::string gameKey = "cs2";
		std::string addon;

		std::filesystem::path imagePath;
		std::string materialName;      // already through CleanMaterialName

		MaterialMode mode = MaterialMode::Opaque;
		double alphaRef = 0.5;

		// 0 keeps the source resolution, otherwise the square edge length.
		int quality = 0;

		bool compile = true;
		bool compilerLock = true;
		bool retryCompile = true;
	};

	struct CreateResult
	{
		bool ok = false;
		std::string status;
		std::string error;

		std::filesystem::path outputDir;
		std::filesystem::path logPath;
	};

	// Called from the worker thread with a one-line progress update.
	using ProgressFn = std::function<void(std::string)>;

	// image -> square PNG -> .vtex -> .vmat -> resourcecompiler.
	// Blocking; run it on a worker. Initialises COM on the calling thread itself.
	CreateResult CreateMaterial(const CreateRequest& request, const ProgressFn& progress);
}
