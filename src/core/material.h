#pragma once

#include <expected>
#include <string>
#include <string_view>

namespace bit
{
	enum class MaterialMode
	{
		Opaque = 0,       // csgo_lightmappedgeneric
		Cutout = 1,       // csgo_complex + F_ALPHA_TEST
		Translucent = 2,  // csgo_complex + F_TRANSLUCENT
	};

	// Collapses everything that isn't [A-Za-z0-9_-] into a single underscore and
	// trims the result. Empty once sanitized means the caller gave us nothing usable.
	std::string SanitizeSegment(std::string_view s);

	// An addon name is taken as-is or rejected outright -- it has to match a real
	// folder on disk, so silently rewriting it would point at the wrong place.
	std::string SanitizeAddon(std::string_view s);

	// Normalizes a user-entered material path: strips a .vmat/.vtex/.png suffix,
	// splits on / and \, sanitizes each segment, rejects . and .. traversal.
	// Returns the joined path or an error message fit to show the user.
	std::expected<std::string, std::string> CleanMaterialName(std::string_view s);

	// The DMX blob resourcecompiler reads to produce the .vtex_c.
	std::string MakeVTEX(std::string_view textureResource);

	// The KV3 material. alphaRef is clamped to [0.01, 0.99] and only used by Cutout.
	std::string MakeVMAT(MaterialMode mode, double alphaRef, std::string_view compiledTextureResource);
}
