#pragma once

#include <string>
#include <string_view>

namespace bit
{
	// Win32 is UTF-16 everywhere; everything above the platform layer stays UTF-8.
	std::wstring Widen(std::string_view utf8);
	std::string Narrow(std::wstring_view utf16);

	std::string TrimSpace(std::string_view s);
	bool EqualsNoCase(std::string_view a, std::string_view b);
	bool EndsWithNoCase(std::string_view s, std::string_view suffix);
	std::string ToLower(std::string_view s);
}
