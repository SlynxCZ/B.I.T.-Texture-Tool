#pragma once

#include <filesystem>
#include <string>

namespace bit
{
	// Mirrors the original settings.json field for field, so an existing file
	// from the Go build is picked up rather than ignored.
	struct AppSettings
	{
		std::string language = "en";
		std::string game = "cs2";
		bool darkMode = true;
		std::string cs2Root;
		std::string lastAddon;
		std::string lastImageDir;
		int materialMode = 0;
		int quality = 0;
		int autoMode = 1;
		bool retryCompile = true;
		bool compilerLock = true;
		int overwriteMode = 0;
		int customWorkers = 0;
	};

	// %APPDATA%\BIT_Texture_Tool
	std::filesystem::path AppDataDir();
	std::filesystem::path SettingsPath();
	std::filesystem::path JunkDir();

	// Missing or malformed file leaves the defaults in place; this never fails
	// loudly, because losing settings should not stop the tool from starting.
	AppSettings LoadSettings();
	bool SaveSettings(const AppSettings& settings);

	// Deletes everything in the junk folder. Returns how many files went.
	int ClearJunkFolder();
}
