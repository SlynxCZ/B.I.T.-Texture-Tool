#pragma once

#include "core/batch.h"
#include "core/material.h"
#include "core/settings.h"
#include "ui/texture.h"

#include <unordered_map>

#include <atomic>
#include <filesystem>
#include <memory>
#include <mutex>
#include <string>
#include <thread>
#include <vector>

namespace bit
{
	// Everything the UI reads and writes. Kept separate from the ImGui drawing
	// code so the workflow stays testable without a window.
	struct AppState
	{
		AppSettings settings;

		std::filesystem::path gameRoot;
		bool gameRootValid = false;

		std::vector<std::string> addons;
		int selectedAddon = -1;

		std::filesystem::path imagePath;
		std::string materialName;
		int sourceWidth = 0;
		int sourceHeight = 0;
		bool sourceHasAlpha = false;
		GpuTexture preview;

		// Decoded once on first use and kept for the process lifetime.
		GpuTexture logoLight;
		GpuTexture logoDark;
		std::unordered_map<std::string, GpuTexture> flags;
		bool artworkLoaded = false;

		float alphaRef = 0.5f;

		// Autonomous Production
		std::filesystem::path batchFolder;
		std::vector<std::filesystem::path> batchFiles;
		BatchResult batchResult;
		bool batchFinished = false;
		std::atomic<bool> cancelBatch{ false };

		// Written by the worker, read by the frame. busy is the flag the UI
		// gates on and is the last thing a job clears.
		std::atomic<bool> busy{ false };
		std::atomic<int> progressDone{ 0 };
		std::atomic<int> progressTotal{ 0 };
		std::mutex mutex;
		std::string statusLine;
		std::string compileLog;
		std::filesystem::path lastOutputDir;

		std::unique_ptr<std::thread> worker;
		bool themeDirty = true;

		~AppState();
	};

	// Reads settings.json, applies language and theme, runs detection.
	void InitApp(AppState& state);
	void ShutdownApp(AppState& state);

	void RefreshGameRoot(AppState& state);
	void ChooseImage(AppState& state);
	void ChooseBatchFolder(AppState& state);

	void StartCreate(AppState& state);
	void StartBatch(AppState& state);

	// One ImGui frame of the whole window.
	void DrawApp(AppState& state);
}
