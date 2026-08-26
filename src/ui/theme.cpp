#include "ui/theme.h"

#include "imgui.h"

namespace bit
{
	namespace
	{
		void ApplyShape(ImGuiStyle& style)
		{
			style.WindowPadding = ImVec2(14.0f, 12.0f);
			style.FramePadding = ImVec2(10.0f, 6.0f);
			style.ItemSpacing = ImVec2(10.0f, 8.0f);
			style.ItemInnerSpacing = ImVec2(8.0f, 6.0f);
			style.ScrollbarSize = 12.0f;
			style.GrabMinSize = 10.0f;

			style.WindowRounding = 0.0f;
			style.ChildRounding = 6.0f;
			style.FrameRounding = 6.0f;
			style.PopupRounding = 6.0f;
			style.ScrollbarRounding = 8.0f;
			style.GrabRounding = 6.0f;
			style.TabRounding = 6.0f;

			style.WindowBorderSize = 0.0f;
			style.ChildBorderSize = 1.0f;
			style.FrameBorderSize = 0.0f;

			style.SeparatorTextBorderSize = 1.0f;
			style.SeparatorTextPadding = ImVec2(16.0f, 6.0f);
		}
	}

	void ApplyTheme(bool dark)
	{
		ImGuiStyle& style = ImGui::GetStyle();

		if (dark)
			ImGui::StyleColorsDark();
		else
			ImGui::StyleColorsLight();

		ApplyShape(style);

		ImVec4* colors = style.Colors;

		// One accent for both themes so the tool keeps its identity when the
		// theme flips; everything else is derived from the background.
		const ImVec4 accent      = dark ? ImVec4(0.26f, 0.59f, 0.98f, 1.00f) : ImVec4(0.16f, 0.45f, 0.85f, 1.00f);
		const ImVec4 accentHover = dark ? ImVec4(0.33f, 0.66f, 1.00f, 1.00f) : ImVec4(0.24f, 0.54f, 0.92f, 1.00f);

		if (dark)
		{
			colors[ImGuiCol_WindowBg]        = ImVec4(0.09f, 0.09f, 0.11f, 1.00f);
			colors[ImGuiCol_ChildBg]         = ImVec4(0.12f, 0.12f, 0.15f, 1.00f);
			colors[ImGuiCol_PopupBg]         = ImVec4(0.12f, 0.12f, 0.15f, 0.98f);
			colors[ImGuiCol_FrameBg]         = ImVec4(0.17f, 0.17f, 0.21f, 1.00f);
			colors[ImGuiCol_FrameBgHovered]  = ImVec4(0.22f, 0.22f, 0.27f, 1.00f);
			colors[ImGuiCol_FrameBgActive]   = ImVec4(0.25f, 0.25f, 0.31f, 1.00f);
			colors[ImGuiCol_Border]          = ImVec4(0.24f, 0.24f, 0.29f, 1.00f);
			colors[ImGuiCol_Text]            = ImVec4(0.90f, 0.90f, 0.93f, 1.00f);
			colors[ImGuiCol_TextDisabled]    = ImVec4(0.48f, 0.48f, 0.54f, 1.00f);
		}
		else
		{
			colors[ImGuiCol_WindowBg]        = ImVec4(0.96f, 0.96f, 0.97f, 1.00f);
			colors[ImGuiCol_ChildBg]         = ImVec4(1.00f, 1.00f, 1.00f, 1.00f);
			colors[ImGuiCol_PopupBg]         = ImVec4(1.00f, 1.00f, 1.00f, 0.98f);
			colors[ImGuiCol_FrameBg]         = ImVec4(0.90f, 0.90f, 0.92f, 1.00f);
			colors[ImGuiCol_FrameBgHovered]  = ImVec4(0.85f, 0.86f, 0.90f, 1.00f);
			colors[ImGuiCol_FrameBgActive]   = ImVec4(0.80f, 0.82f, 0.88f, 1.00f);
			colors[ImGuiCol_Border]          = ImVec4(0.78f, 0.78f, 0.81f, 1.00f);
			colors[ImGuiCol_Text]            = ImVec4(0.10f, 0.10f, 0.12f, 1.00f);
			colors[ImGuiCol_TextDisabled]    = ImVec4(0.52f, 0.52f, 0.56f, 1.00f);
		}

		colors[ImGuiCol_Button]              = accent;
		colors[ImGuiCol_ButtonHovered]       = accentHover;
		colors[ImGuiCol_ButtonActive]        = accent;
		colors[ImGuiCol_CheckMark]           = accent;
		colors[ImGuiCol_SliderGrab]          = accent;
		colors[ImGuiCol_SliderGrabActive]    = accentHover;
		colors[ImGuiCol_Header]              = ImVec4(accent.x, accent.y, accent.z, 0.35f);
		colors[ImGuiCol_HeaderHovered]       = ImVec4(accent.x, accent.y, accent.z, 0.55f);
		colors[ImGuiCol_HeaderActive]        = accent;
		colors[ImGuiCol_SeparatorHovered]    = accent;
		colors[ImGuiCol_SeparatorActive]     = accentHover;
		colors[ImGuiCol_PlotHistogram]       = accent;   // what ProgressBar draws with
	}
}
