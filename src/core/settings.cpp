#include "core/settings.h"

#include "core/strings.h"

#include <windows.h>
#include <shlobj.h>

#include <cctype>
#include <fstream>
#include <sstream>
#include <system_error>
#include <unordered_map>

namespace fs = std::filesystem;

namespace bit
{
	namespace
	{
		// The settings file is a flat object of strings, ints and bools, so a
		// full JSON parser would be more machinery than the job needs. This
		// handles exactly that shape and ignores anything it doesn't recognise.
		class FlatJson
		{
		public:
			explicit FlatJson(std::string_view text)
			{
				size_t i = 0;
				while (i < text.size())
				{
					const size_t keyStart = text.find('"', i);
					if (keyStart == std::string_view::npos)
						break;

					const size_t keyEnd = FindClosingQuote(text, keyStart + 1);
					if (keyEnd == std::string_view::npos)
						break;

					const std::string key = Unescape(text.substr(keyStart + 1, keyEnd - keyStart - 1));

					const size_t colon = text.find(':', keyEnd);
					if (colon == std::string_view::npos)
						break;

					size_t v = colon + 1;
					while (v < text.size() && std::isspace(static_cast<unsigned char>(text[v])))
						++v;

					if (v >= text.size())
						break;

					if (text[v] == '"')
					{
						const size_t end = FindClosingQuote(text, v + 1);
						if (end == std::string_view::npos)
							break;

						m_values[key] = Unescape(text.substr(v + 1, end - v - 1));
						i = end + 1;
					}
					else
					{
						size_t end = v;
						while (end < text.size() && text[end] != ',' && text[end] != '}')
							++end;

						m_values[key] = TrimSpace(text.substr(v, end - v));
						i = end;
					}
				}
			}

			std::string Str(const std::string& key, std::string fallback) const
			{
				const auto it = m_values.find(key);
				return it != m_values.end() ? it->second : std::move(fallback);
			}

			int Int(const std::string& key, int fallback) const
			{
				const auto it = m_values.find(key);
				if (it == m_values.end())
					return fallback;

				try { return std::stoi(it->second); }
				catch (...) { return fallback; }
			}

			bool Bool(const std::string& key, bool fallback) const
			{
				const auto it = m_values.find(key);
				if (it == m_values.end())
					return fallback;

				return it->second == "true" || it->second == "1";
			}

		private:
			static size_t FindClosingQuote(std::string_view text, size_t from)
			{
				for (size_t i = from; i < text.size(); ++i)
				{
					if (text[i] == '\\')
					{
						++i;
						continue;
					}
					if (text[i] == '"')
						return i;
				}
				return std::string_view::npos;
			}

			static std::string Unescape(std::string_view s)
			{
				std::string out;
				out.reserve(s.size());

				for (size_t i = 0; i < s.size(); ++i)
				{
					if (s[i] != '\\' || i + 1 >= s.size())
					{
						out.push_back(s[i]);
						continue;
					}

					switch (s[++i])
					{
					case 'n':  out.push_back('\n'); break;
					case 't':  out.push_back('\t'); break;
					case 'r':  out.push_back('\r'); break;
					case '\\': out.push_back('\\'); break;
					case '"':  out.push_back('"');  break;
					default:   out.push_back(s[i]); break;
					}
				}
				return out;
			}

			std::unordered_map<std::string, std::string> m_values;
		};

		std::string Escape(std::string_view s)
		{
			std::string out;
			out.reserve(s.size() + 8);

			for (const char c : s)
			{
				switch (c)
				{
				case '\\': out += "\\\\"; break;
				case '"':  out += "\\\""; break;
				case '\n': out += "\\n";  break;
				case '\r': out += "\\r";  break;
				case '\t': out += "\\t";  break;
				default:   out.push_back(c); break;
				}
			}
			return out;
		}
	}

	fs::path AppDataDir()
	{
		PWSTR raw = nullptr;
		if (FAILED(::SHGetKnownFolderPath(FOLDERID_RoamingAppData, 0, nullptr, &raw)))
			return fs::current_path();

		fs::path dir(raw);
		::CoTaskMemFree(raw);

		dir /= "BIT_Texture_Tool";

		std::error_code ec;
		fs::create_directories(dir, ec);

		return dir;
	}

	fs::path SettingsPath() { return AppDataDir() / "settings.json"; }

	fs::path JunkDir()
	{
		const fs::path dir = AppDataDir() / "junk";

		std::error_code ec;
		fs::create_directories(dir, ec);

		return dir;
	}

	AppSettings LoadSettings()
	{
		AppSettings settings;

		std::ifstream file(SettingsPath(), std::ios::binary);
		if (!file)
			return settings;

		std::ostringstream buffer;
		buffer << file.rdbuf();

		const FlatJson json(buffer.str());

		settings.language      = json.Str("language", settings.language);
		settings.game          = json.Str("game", settings.game);
		settings.darkMode      = json.Bool("dark_mode", settings.darkMode);
		settings.cs2Root       = json.Str("cs2_root", settings.cs2Root);
		settings.lastAddon     = json.Str("last_addon", settings.lastAddon);
		settings.lastImageDir  = json.Str("last_image_dir", settings.lastImageDir);
		settings.materialMode  = json.Int("material_mode", settings.materialMode);
		settings.quality       = json.Int("quality", settings.quality);
		settings.autoMode      = json.Int("auto_mode", settings.autoMode);
		settings.retryCompile  = json.Bool("retry_compile", settings.retryCompile);
		settings.compilerLock  = json.Bool("compiler_lock", settings.compilerLock);
		settings.overwriteMode = json.Int("overwrite_mode", settings.overwriteMode);
		settings.customWorkers = json.Int("custom_workers", settings.customWorkers);

		return settings;
	}

	bool SaveSettings(const AppSettings& settings)
	{
		std::ofstream file(SettingsPath(), std::ios::binary | std::ios::trunc);
		if (!file)
			return false;

		file << "{\n"
			<< "  \"language\": \""       << Escape(settings.language)     << "\",\n"
			<< "  \"game\": \""           << Escape(settings.game)         << "\",\n"
			<< "  \"dark_mode\": "        << (settings.darkMode ? "true" : "false") << ",\n"
			<< "  \"cs2_root\": \""       << Escape(settings.cs2Root)      << "\",\n"
			<< "  \"last_addon\": \""     << Escape(settings.lastAddon)    << "\",\n"
			<< "  \"last_image_dir\": \"" << Escape(settings.lastImageDir) << "\",\n"
			<< "  \"material_mode\": "    << settings.materialMode         << ",\n"
			<< "  \"quality\": "          << settings.quality              << ",\n"
			<< "  \"auto_mode\": "        << settings.autoMode             << ",\n"
			<< "  \"retry_compile\": "    << (settings.retryCompile ? "true" : "false") << ",\n"
			<< "  \"compiler_lock\": "    << (settings.compilerLock ? "true" : "false") << ",\n"
			<< "  \"overwrite_mode\": "   << settings.overwriteMode        << ",\n"
			<< "  \"custom_workers\": "   << settings.customWorkers        << "\n"
			<< "}\n";

		return static_cast<bool>(file);
	}

	int ClearJunkFolder()
	{
		int removed = 0;
		std::error_code ec;

		for (const fs::directory_entry& entry : fs::directory_iterator(JunkDir(), ec))
		{
			if (fs::remove_all(entry.path(), ec) > 0)
				++removed;
		}

		return removed;
	}
}
