#pragma once

#include "core/pipeline.h"

#include <atomic>
#include <filesystem>
#include <functional>
#include <string>
#include <vector>

namespace bit
{
	enum class SpeedMode
	{
		Slow = 0,
		Normal = 1,
		Fast = 2,
		Extreme = 3,
		Custom = 4,
	};

	enum class OverwriteMode
	{
		Ask = 0,
		Skip = 1,
		Replace = 2,
	};

	// Image worker concurrency. Slow and Normal stay single-threaded on purpose:
	// they are the stable reference modes. Fast and Extreme scale with CPU count
	// but are capped by output resolution, because a 4K RGBA buffer is 64 MB per
	// worker and the spike is what actually breaks these batches.
	int AutonomousWorkerCount(SpeedMode mode, int resolutionClass, int customWorkers);

	// Resource Compiler concurrency, always lower than the image workers.
	int AutonomousCompilerSlots(SpeedMode mode, int resolutionClass, int workers);

	// 75th percentile of the sources' longest edge, rounded to a texture class.
	// Used when quality is "keep original" and the real size isn't known upfront.
	int DetectResolutionClass(const std::vector<std::filesystem::path>& files);

	struct BatchRequest
	{
		CreateRequest base;                             // everything except imagePath/materialName
		std::vector<std::filesystem::path> files;
		SpeedMode speed = SpeedMode::Normal;
		OverwriteMode overwrite = OverwriteMode::Skip;
		int customWorkers = 4;
	};

	struct BatchItemResult
	{
		std::filesystem::path source;
		bool ok = false;
		bool skipped = false;
		std::string message;
	};

	struct BatchResult
	{
		int total = 0;
		int succeeded = 0;
		int skipped = 0;
		int failed = 0;
		bool cancelled = false;
		std::vector<BatchItemResult> items;
	};

	// Every image WIC can read, sorted, non-recursive -- same as the original.
	std::vector<std::filesystem::path> CollectImages(const std::filesystem::path& folder);

	// Blocking; run it on a worker. cancel is polled between items, so the ones
	// already in flight finish rather than leaving half-written materials.
	BatchResult RunAutonomousProduction(const BatchRequest& request,
		const ProgressFn& progress, const std::atomic<bool>& cancel);
}
