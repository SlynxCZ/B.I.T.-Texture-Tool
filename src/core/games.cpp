#include "core/games.h"

#include "core/strings.h"

#include <windows.h>

#include <algorithm>
#include <fstream>
#include <regex>
#include <system_error>

namespace fs = std::filesystem;

namespace bit
{
	const std::vector<GameProfile>& GameProfiles()
	{
		static const std::vector<GameProfile> profiles = {
			{ "cs2",      "CS2",              "CS2",      "Counter-Strike Global Offensive" },
			{ "tf2",      "Team Fortress 2",  "TF2",      "Team Fortress 2" },
			{ "deadlock", "Deadlock",         "Deadlock", "Deadlock" },
			{ "hla",      "Half-Life: Alyx",  "HLA",      "Half-Life Alyx" },
		};
		return profiles;
	}

	bool ValidGameKey(std::string_view key)
	{
		const auto& profiles = GameProfiles();
		return std::any_of(profiles.begin(), profiles.end(),
			[key](const GameProfile& g) { return g.key == key; });
	}

	const GameProfile& GameProfileForKey(std::string_view key)
	{
		const auto& profiles = GameProfiles();

		const auto it = std::find_if(profiles.begin(), profiles.end(),
			[key](const GameProfile& g) { return g.key == key; });

		return it != profiles.end() ? *it : profiles.front();
	}

	int GameIndexForKey(std::string_view key)
	{
		const auto& profiles = GameProfiles();

		for (size_t i = 0; i < profiles.size(); ++i)
		{
			if (profiles[i].key == key)
				return static_cast<int>(i);
		}
		return 0;
	}

	fs::path GameAddonDirectory(const fs::path& root, std::string_view gameKey)
	{
		if (gameKey == "tf2")
			return root / "tf" / "custom";

		if (gameKey == "hla")
			return root / "content" / "hlvr_addons";

		// Deadlock's internal Source 2 project name is citadel. If the folder
		// isn't there, the addon list simply comes back empty.
		if (gameKey == "deadlock")
			return root / "content" / "citadel_addons";

		return root / "content" / "csgo_addons";
	}

	bool ValidGameRoot(const fs::path& root, std::string_view gameKey)
	{
		std::error_code ec;

		if (root.empty() || !fs::is_directory(root, ec))
			return false;

		if (gameKey == "cs2")
		{
			return fs::is_directory(root / "content" / "csgo_addons", ec)
				&& fs::is_directory(root / "game", ec);
		}

		return true;
	}

	namespace
	{
		std::string EnvVar(const wchar_t* name)
		{
			wchar_t buffer[MAX_PATH * 2] = {};

			const DWORD len = ::GetEnvironmentVariableW(name, buffer, static_cast<DWORD>(std::size(buffer)));
			if (len == 0 || len >= std::size(buffer))
				return {};

			return Narrow(std::wstring_view(buffer, len));
		}

		// Steam records extra library drives in libraryfolders.vdf as
		//   "path"    "D:\\SteamLibrary"
		// The doubled backslashes are VDF escaping, not real path separators.
		void AppendLibrariesFromVdf(const fs::path& steamRoot, std::vector<fs::path>& out)
		{
			std::ifstream file(steamRoot / "steamapps" / "libraryfolders.vdf");
			if (!file)
				return;

			static const std::regex pathLine(R"("path"\s+"([^"]+)")");

			std::string line;
			while (std::getline(file, line))
			{
				std::smatch match;
				if (!std::regex_search(line, match, pathLine))
					continue;

				std::string value = match[1].str();

				size_t pos = 0;
				while ((pos = value.find("\\\\", pos)) != std::string::npos)
				{
					value.replace(pos, 2, "\\");
					pos += 1;
				}

				out.emplace_back(value);
			}
		}
	}

	fs::path DetectGameRoot(std::string_view gameKey)
	{
		const GameProfile& profile = GameProfileForKey(gameKey);

		std::vector<fs::path> candidates;

		if (const std::string pf86 = EnvVar(L"ProgramFiles(x86)"); !pf86.empty())
			candidates.emplace_back(fs::path(pf86) / "Steam");

		if (const std::string pf = EnvVar(L"ProgramFiles"); !pf.empty())
			candidates.emplace_back(fs::path(pf) / "Steam");

		candidates.emplace_back(R"(C:\Program Files (x86)\Steam)");
		candidates.emplace_back(R"(C:\Program Files\Steam)");

		std::vector<std::string> seen;
		std::vector<fs::path> libraryRoots;
		std::error_code ec;

		for (fs::path steam : candidates)
		{
			steam = steam.lexically_normal();

			const std::string key = ToLower(steam.string());
			if (std::find(seen.begin(), seen.end(), key) != seen.end())
				continue;

			seen.push_back(key);

			if (!fs::exists(steam, ec))
				continue;

			libraryRoots.push_back(steam);
			AppendLibrariesFromVdf(steam, libraryRoots);
		}

		for (const fs::path& library : libraryRoots)
		{
			// Skip UNC paths: probing those can block for the network timeout.
			const std::string normalized = library.lexically_normal().string();
			if (normalized.rfind("\\\\", 0) == 0)
				continue;

			const fs::path root = library / "steamapps" / "common" / std::string(profile.steamFolder);
			if (ValidGameRoot(root, gameKey))
				return root;
		}

		return {};
	}

	std::vector<std::string> EnumerateAddons(const fs::path& root, std::string_view gameKey)
	{
		std::vector<std::string> addons;
		std::error_code ec;

		const fs::path dir = GameAddonDirectory(root, gameKey);
		if (!fs::is_directory(dir, ec))
			return addons;

		for (const fs::directory_entry& entry : fs::directory_iterator(dir, ec))
		{
			if (entry.is_directory(ec))
				addons.push_back(entry.path().filename().string());
		}

		std::sort(addons.begin(), addons.end(), [](const std::string& a, const std::string& b) {
			return ToLower(a) < ToLower(b);
		});

		return addons;
	}
}
