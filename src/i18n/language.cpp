#include "i18n/translations.h"

#include <algorithm>

namespace bit
{
	namespace
	{
		const LanguageDef* g_pActive = nullptr;
		const LanguageDef* g_pEnglish = nullptr;

		const LanguageDef* Find(std::string_view code)
		{
			const auto& languages = Languages();

			const auto it = std::find_if(languages.begin(), languages.end(),
				[code](const LanguageDef& l) { return l.code == code; });

			return it != languages.end() ? &*it : nullptr;
		}

		const LanguageDef* English()
		{
			if (!g_pEnglish)
				g_pEnglish = Find("en");

			return g_pEnglish;
		}
	}

	void SetLanguage(std::string_view code)
	{
		const LanguageDef* found = Find(code);
		g_pActive = found ? found : English();
	}

	std::string_view CurrentLanguage()
	{
		if (!g_pActive)
			g_pActive = English();

		return g_pActive ? g_pActive->code : "en";
	}

	int CurrentLanguageIndex()
	{
		const auto& languages = Languages();
		const std::string_view code = CurrentLanguage();

		for (size_t i = 0; i < languages.size(); ++i)
		{
			if (languages[i].code == code)
				return static_cast<int>(i);
		}
		return 0;
	}

	const char* tr(std::string_view key)
	{
		if (!g_pActive)
			g_pActive = English();

		// string_view keys point at the generated tables, which are static, so
		// handing back .data() is safe -- every literal there is NUL terminated.
		if (g_pActive && g_pActive->table)
		{
			if (const auto it = g_pActive->table->find(key); it != g_pActive->table->end())
				return it->second.data();
		}

		if (const LanguageDef* en = English(); en && en->table)
		{
			if (const auto it = en->table->find(key); it != en->table->end())
				return it->second.data();
		}

		return key.data();
	}
}
