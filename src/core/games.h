#pragma once

#include <filesystem>
#include <string>
#include <string_view>
#include <vector>

namespace bit
{
	struct GameProfile
	{
		std::string_view key;
		std::string_view name;
		std::string_view shortName;
		std::string_view steamFolder;
	};

	const std::vector<GameProfile>& GameProfiles();

	bool ValidGameKey(std::string_view key);
	const GameProfile& GameProfileForKey(std::string_view key);
	int GameIndexForKey(std::string_view key);

	// content/<game>_addons for the Source 2 titles, tf/custom for TF2.
	std::filesystem::path GameAddonDirectory(const std::filesystem::path& root, std::string_view gameKey);

	// CS2 is the one we can check properly: it needs both content/csgo_addons
	// and game/. For the others, "is a directory" is as far as we can go.
	bool ValidGameRoot(const std::filesystem::path& root, std::string_view gameKey);

	// Walks the Steam library folders and returns the first install that passes
	// ValidGameRoot, or an empty path.
	std::filesystem::path DetectGameRoot(std::string_view gameKey);

	// Every immediate subdirectory of GameAddonDirectory(), sorted.
	std::vector<std::string> EnumerateAddons(const std::filesystem::path& root, std::string_view gameKey);
}
