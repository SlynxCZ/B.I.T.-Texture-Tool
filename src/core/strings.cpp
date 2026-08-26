#include "core/strings.h"

#include <windows.h>

#include <algorithm>
#include <cctype>

namespace bit
{
	std::wstring Widen(std::string_view utf8)
	{
		if (utf8.empty())
			return {};

		const int len = ::MultiByteToWideChar(CP_UTF8, 0, utf8.data(), static_cast<int>(utf8.size()), nullptr, 0);
		if (len <= 0)
			return {};

		std::wstring out(static_cast<size_t>(len), L'\0');
		::MultiByteToWideChar(CP_UTF8, 0, utf8.data(), static_cast<int>(utf8.size()), out.data(), len);
		return out;
	}

	std::string Narrow(std::wstring_view utf16)
	{
		if (utf16.empty())
			return {};

		const int len = ::WideCharToMultiByte(CP_UTF8, 0, utf16.data(), static_cast<int>(utf16.size()), nullptr, 0, nullptr, nullptr);
		if (len <= 0)
			return {};

		std::string out(static_cast<size_t>(len), '\0');
		::WideCharToMultiByte(CP_UTF8, 0, utf16.data(), static_cast<int>(utf16.size()), out.data(), len, nullptr, nullptr);
		return out;
	}

	std::string TrimSpace(std::string_view s)
	{
		const auto notSpace = [](unsigned char c) { return !std::isspace(c); };

		auto begin = std::find_if(s.begin(), s.end(), notSpace);
		auto end = std::find_if(s.rbegin(), s.rend(), notSpace).base();

		return begin < end ? std::string(begin, end) : std::string();
	}

	std::string ToLower(std::string_view s)
	{
		std::string out(s);
		std::transform(out.begin(), out.end(), out.begin(),
			[](unsigned char c) { return static_cast<char>(std::tolower(c)); });
		return out;
	}

	bool EqualsNoCase(std::string_view a, std::string_view b)
	{
		if (a.size() != b.size())
			return false;

		for (size_t i = 0; i < a.size(); ++i)
		{
			if (std::tolower(static_cast<unsigned char>(a[i])) != std::tolower(static_cast<unsigned char>(b[i])))
				return false;
		}
		return true;
	}

	bool EndsWithNoCase(std::string_view s, std::string_view suffix)
	{
		if (suffix.size() > s.size())
			return false;

		return EqualsNoCase(s.substr(s.size() - suffix.size()), suffix);
	}
}
