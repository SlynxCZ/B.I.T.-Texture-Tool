#include "core/material.h"

#include "core/strings.h"

#include <algorithm>
#include <array>
#include <format>
#include <vector>

namespace bit
{
	std::string SanitizeSegment(std::string_view s)
	{
		const std::string trimmed = TrimSpace(s);

		std::string out;
		out.reserve(trimmed.size());

		bool lastUnderscore = false;
		for (const char c : trimmed)
		{
			const unsigned char u = static_cast<unsigned char>(c);
			const bool good = (u >= 'a' && u <= 'z') || (u >= 'A' && u <= 'Z')
				|| (u >= '0' && u <= '9') || c == '_' || c == '-';

			if (good)
			{
				out.push_back(c);
				lastUnderscore = false;
			}
			else if (!lastUnderscore)
			{
				out.push_back('_');
				lastUnderscore = true;
			}
		}

		const size_t first = out.find_first_not_of('_');
		if (first == std::string::npos)
			return {};

		const size_t last = out.find_last_not_of('_');
		return out.substr(first, last - first + 1);
	}

	std::string SanitizeAddon(std::string_view s)
	{
		const std::string trimmed = TrimSpace(s);

		if (trimmed.empty() || trimmed.find("..") != std::string::npos)
			return {};

		if (trimmed.find_first_of(R"(\/:*?"<>|)") != std::string::npos)
			return {};

		return trimmed;
	}

	std::expected<std::string, std::string> CleanMaterialName(std::string_view input)
	{
		std::string s = TrimSpace(input);
		std::replace(s.begin(), s.end(), '\\', '/');

		static constexpr std::array<std::string_view, 3> kStrippable = { ".vmat", ".vtex", ".png" };
		for (const std::string_view ext : kStrippable)
		{
			if (EndsWithNoCase(s, ext))
			{
				s.resize(s.size() - ext.size());
				break;
			}
		}

		std::vector<std::string> parts;

		size_t start = 0;
		while (start <= s.size())
		{
			const size_t slash = s.find('/', start);
			const size_t end = slash == std::string::npos ? s.size() : slash;

			std::string segment = TrimSpace(std::string_view(s).substr(start, end - start));

			if (!segment.empty())
			{
				if (segment == "." || segment == "..")
					return std::unexpected("invalid VMAT path");

				segment = SanitizeSegment(segment);
				if (segment.empty())
					return std::unexpected("enter a valid VMAT name");

				parts.push_back(std::move(segment));
			}

			if (slash == std::string::npos)
				break;

			start = slash + 1;
		}

		if (parts.empty())
			return std::unexpected("enter a VMAT/material name");

		std::string joined = parts.front();
		for (size_t i = 1; i < parts.size(); ++i)
			joined += "/" + parts[i];

		return joined;
	}

	std::string MakeVTEX(std::string_view textureResource)
	{
		return std::format(R"(<!-- dmx encoding keyvalues2_noids 1 format vtex 1 -->
"CDmeVtex"
{{
    "m_inputTextureArray" "element_array"
    [
        "CDmeInputTexture"
        {{
            "m_name" "string" "InputTexture0"
            "m_fileName" "string" "{}"
            "m_colorSpace" "string" "srgb"
            "m_typeString" "string" "2D"
            "m_imageProcessorArray" "element_array"
            [
                "CDmeImageProcessor"
                {{
                    "m_algorithm" "string" "None"
                    "m_stringArg" "string" ""
                    "m_vFloat4Arg" "vector4" "0 0 0 0"
                }}
            ]
        }}
    ]
    "m_outputTypeString" "string" "2D"
    "m_outputFormat" "string" "DXT5"
    "m_outputClearColor" "vector4" "0 0 0 0"
    "m_nOutputMinDimension" "int" "0"
    "m_nOutputMaxDimension" "int" "0"
    "m_textureOutputChannelArray" "element_array"
    [
        "CDmeTextureOutputChannel"
        {{
            "m_inputTextureArray" "string_array" [ "InputTexture0" ]
            "m_srcChannels" "string" "rgba"
            "m_dstChannels" "string" "rgba"
            "m_mipAlgorithm" "CDmeImageProcessor"
            {{
                "m_algorithm" "string" "Box"
                "m_stringArg" "string" ""
                "m_vFloat4Arg" "vector4" "0 0 0 0"
            }}
            "m_outputColorSpace" "string" "srgb"
        }}
    ]
    "m_vClamp" "vector3" "0 0 0"
    "m_bNoLod" "bool" "0"
}}
)", textureResource);
	}

	std::string MakeVMAT(MaterialMode mode, double alphaRef, std::string_view compiledTextureResource)
	{
		static constexpr std::string_view kKV3 =
			R"(<!-- kv3 encoding:text:version{e21c7f3c-8a33-41c5-9977-a76d3a32aa0d} format:generic:version{7412167c-06e9-4698-aff2-e63eb59037e7} -->)";

		std::string_view shader = "csgo_lightmappedgeneric.vfx";
		std::string extra;

		switch (mode)
		{
		case MaterialMode::Cutout:
			// The CS2 Material Editor exposes this as hard-edged transparency
			// plus an Alpha Test Reference value; both are shader features.
			shader = "csgo_complex.vfx";
			alphaRef = std::clamp(alphaRef, 0.01, 0.99);
			extra = std::format("    F_ALPHA_TEST = 1\n    g_flAlphaTestReference = {:.2f}\n", alphaRef);
			break;

		case MaterialMode::Translucent:
			// Keeps partial alpha instead of thresholding it.
			shader = "csgo_complex.vfx";
			extra = "    F_TRANSLUCENT = 1\n";
			break;

		case MaterialMode::Opaque:
			break;
		}

		return std::format(R"({}
{{
    shader = "{}"
    g_tColor = resource:"{}"
{}    g_flRoughness = 0.5
}}
)", kKV3, shader, compiledTextureResource, extra);
	}
}
