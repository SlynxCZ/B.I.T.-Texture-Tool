#pragma once

#include "core/image.h"

#include <cstdint>

struct ID3D11Device;
struct ID3D11ShaderResourceView;

namespace bit
{
	// main.cpp hands the device over once the swap chain is up; the UI layer
	// only ever needs it to turn decoded pixels into something ImGui can draw.
	void SetRenderDevice(ID3D11Device* device);

	// Owns one shader resource view. Copyable would mean refcount juggling for
	// no gain, so it moves only.
	class GpuTexture
	{
	public:
		GpuTexture() = default;
		~GpuTexture();

		GpuTexture(const GpuTexture&) = delete;
		GpuTexture& operator=(const GpuTexture&) = delete;

		GpuTexture(GpuTexture&& other) noexcept;
		GpuTexture& operator=(GpuTexture&& other) noexcept;

		// Replaces whatever was here. False leaves the texture empty rather
		// than half-built, so callers can just check Valid().
		bool Upload(const Image& image);
		void Reset();

		bool Valid() const { return m_pView != nullptr; }
		int width() const { return m_width; }
		int height() const { return m_height; }

		// For ImGui::Image.
		uint64_t Handle() const { return reinterpret_cast<uint64_t>(m_pView); }

	private:
		ID3D11ShaderResourceView* m_pView = nullptr;
		int m_width = 0;
		int m_height = 0;
	};
}
