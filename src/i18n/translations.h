#pragma once

#include <string>
#include <string_view>
#include <unordered_map>
#include <vector>

namespace bit
{
	using TranslationTable = std::unordered_map<std::string_view, std::string_view>;

	struct LanguageDef
	{
		std::string_view code;      // "en", "pt-BR", ...
		std::string_view name;      // shown in the picker, in its own language
		const TranslationTable* table = nullptr;
	};

	const std::vector<LanguageDef>& Languages();

	// Sets the active language. Unknown codes fall back to English.
	void SetLanguage(std::string_view code);
	std::string_view CurrentLanguage();
	int CurrentLanguageIndex();

	// Looks the key up in the active language, then English, then returns the
	// key itself so a missing string is visible rather than blank.
	const char* tr(std::string_view key);
}
