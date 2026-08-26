#pragma once

#include <cstdint>
#include <expected>
#include <filesystem>
#include <string>
#include <vector>

namespace bit
{
	// Straight RGBA8, top-down. Wide enough for a D3D11 preview texture upload
	// and for everything the material pipeline needs to do to a texture.
	struct Image
	{
		int width = 0;
		int height = 0;
		std::vector<uint8_t> pixels;   // width * height * 4

		bool Valid() const { return width > 0 && height > 0 && pixels.size() == static_cast<size_t>(width) * height * 4; }
		bool Square() const { return width == height; }
	};

	// WIC handles PNG, JPEG, BMP, GIF and TIFF out of the box. TGA is not one of
	// them on older Windows builds, so that case is reported rather than guessed at.
	std::expected<Image, std::string> LoadImageFile(const std::filesystem::path& path);

	// Decodes one of the RCDATA blobs from resource/app.rc. Same PNG bytes the
	// Go build carried through //go:embed, just read out of the module image.
	std::expected<Image, std::string> LoadImageResource(int resourceId);

	// True if any pixel is not fully opaque. Drives the material-mode hint.
	bool HasAlpha(const Image& image);

	// Centre crop to the shorter edge. Returns the input untouched if already square.
	Image CropToSquare(const Image& image);

	std::expected<Image, std::string> Resize(const Image& image, int width, int height);

	std::expected<void, std::string> SavePNG(const Image& image, const std::filesystem::path& path);

	// COM has to be live on whichever thread touches WIC. The pipeline runs on a
	// worker, so it initialises COM there itself; this is the RAII for that.
	class ComScope
	{
	public:
		ComScope();
		~ComScope();

		ComScope(const ComScope&) = delete;
		ComScope& operator=(const ComScope&) = delete;

		bool ok() const { return m_initialized; }

	private:
		bool m_initialized = false;
	};
}
