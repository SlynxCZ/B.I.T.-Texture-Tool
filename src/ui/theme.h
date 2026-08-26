#pragma once

namespace bit
{
	// The Go build owner-drew every control to get light and dark; with ImGui
	// this is just two colour tables over the same style.
	void ApplyTheme(bool dark);
}
