#include "core/image.h"

#include "core/strings.h"

#include <windows.h>
#include <wincodec.h>
#include <wrl/client.h>

#include <format>

using Microsoft::WRL::ComPtr;
namespace fs = std::filesystem;

namespace bit
{
	ComScope::ComScope()
	{
		const HRESULT hr = ::CoInitializeEx(nullptr, COINIT_APARTMENTTHREADED | COINIT_DISABLE_OLE1DDE);

		// RPC_E_CHANGED_MODE means someone already initialised this thread with a
		// different model. WIC still works, we just must not uninitialise it.
		m_initialized = SUCCEEDED(hr);
	}

	ComScope::~ComScope()
	{
		if (m_initialized)
			::CoUninitialize();
	}

	namespace
	{
		std::expected<ComPtr<IWICImagingFactory>, std::string> CreateFactory()
		{
			ComPtr<IWICImagingFactory> factory;

			const HRESULT hr = ::CoCreateInstance(CLSID_WICImagingFactory, nullptr,
				CLSCTX_INPROC_SERVER, IID_PPV_ARGS(&factory));

			if (FAILED(hr))
				return std::unexpected(std::format("WIC is unavailable (0x{:08X})", static_cast<uint32_t>(hr)));

			return factory;
		}

		// Everything downstream assumes 32bpp RGBA, so conversion happens once here.
		std::expected<Image, std::string> ToImage(IWICImagingFactory* factory, IWICBitmapSource* source)
		{
			ComPtr<IWICFormatConverter> converter;
			if (FAILED(factory->CreateFormatConverter(&converter)))
				return std::unexpected("could not create a WIC format converter");

			if (FAILED(converter->Initialize(source, GUID_WICPixelFormat32bppRGBA,
				WICBitmapDitherTypeNone, nullptr, 0.0, WICBitmapPaletteTypeCustom)))
			{
				return std::unexpected("could not convert the image to RGBA");
			}

			UINT width = 0;
			UINT height = 0;
			if (FAILED(converter->GetSize(&width, &height)) || width == 0 || height == 0)
				return std::unexpected("the image reports a zero size");

			Image image;
			image.width = static_cast<int>(width);
			image.height = static_cast<int>(height);
			image.pixels.resize(static_cast<size_t>(width) * height * 4);

			const UINT stride = width * 4;
			const HRESULT hr = converter->CopyPixels(nullptr, stride,
				static_cast<UINT>(image.pixels.size()), image.pixels.data());

			if (FAILED(hr))
				return std::unexpected("could not read the image pixels");

			return image;
		}

		std::expected<ComPtr<IWICBitmap>, std::string> ToWicBitmap(IWICImagingFactory* factory, const Image& image)
		{
			ComPtr<IWICBitmap> bitmap;

			const HRESULT hr = factory->CreateBitmapFromMemory(
				static_cast<UINT>(image.width), static_cast<UINT>(image.height),
				GUID_WICPixelFormat32bppRGBA, static_cast<UINT>(image.width) * 4,
				static_cast<UINT>(image.pixels.size()),
				const_cast<BYTE*>(image.pixels.data()), &bitmap);

			if (FAILED(hr))
				return std::unexpected("could not wrap the image for WIC");

			return bitmap;
		}
	}

	std::expected<Image, std::string> LoadImageFile(const fs::path& path)
	{
		auto factory = CreateFactory();
		if (!factory)
			return std::unexpected(factory.error());

		ComPtr<IWICBitmapDecoder> decoder;
		const HRESULT hr = (*factory)->CreateDecoderFromFilename(path.c_str(), nullptr,
			GENERIC_READ, WICDecodeMetadataCacheOnDemand, &decoder);

		if (FAILED(hr))
		{
			if (EqualsNoCase(path.extension().string(), ".tga"))
				return std::unexpected("TGA needs a codec Windows doesn't ship — convert it to PNG first");

			return std::unexpected(std::format("could not decode {} (0x{:08X})",
				path.filename().string(), static_cast<uint32_t>(hr)));
		}

		ComPtr<IWICBitmapFrameDecode> frame;
		if (FAILED(decoder->GetFrame(0, &frame)))
			return std::unexpected("the image has no readable frame");

		return ToImage(factory->Get(), frame.Get());
	}

	std::expected<Image, std::string> LoadImageResource(int resourceId)
	{
		const HMODULE module = ::GetModuleHandleW(nullptr);

		const HRSRC found = ::FindResourceW(module, MAKEINTRESOURCEW(resourceId), RT_RCDATA);
		if (!found)
			return std::unexpected(std::format("resource {} is missing from the binary", resourceId));

		const DWORD size = ::SizeofResource(module, found);
		const HGLOBAL handle = ::LoadResource(module, found);
		const void* bytes = handle ? ::LockResource(handle) : nullptr;

		if (!bytes || size == 0)
			return std::unexpected(std::format("resource {} could not be read", resourceId));

		auto factory = CreateFactory();
		if (!factory)
			return std::unexpected(factory.error());

		ComPtr<IWICStream> stream;
		if (FAILED((*factory)->CreateStream(&stream)))
			return std::unexpected("could not create a WIC stream");

		// The resource stays mapped for the module's lifetime, so handing WIC a
		// pointer into it is safe and saves a copy.
		if (FAILED(stream->InitializeFromMemory(
			static_cast<BYTE*>(const_cast<void*>(bytes)), size)))
		{
			return std::unexpected("could not wrap the resource for WIC");
		}

		ComPtr<IWICBitmapDecoder> decoder;
		if (FAILED((*factory)->CreateDecoderFromStream(stream.Get(), nullptr,
			WICDecodeMetadataCacheOnDemand, &decoder)))
		{
			return std::unexpected(std::format("could not decode resource {}", resourceId));
		}

		ComPtr<IWICBitmapFrameDecode> frame;
		if (FAILED(decoder->GetFrame(0, &frame)))
			return std::unexpected("the embedded image has no readable frame");

		return ToImage(factory->Get(), frame.Get());
	}

	bool HasAlpha(const Image& image)
	{
		if (!image.Valid())
			return false;

		for (size_t i = 3; i < image.pixels.size(); i += 4)
		{
			if (image.pixels[i] != 0xFF)
				return true;
		}
		return false;
	}

	Image CropToSquare(const Image& image)
	{
		if (!image.Valid() || image.Square())
			return image;

		const int edge = std::min(image.width, image.height);
		const int offsetX = (image.width - edge) / 2;
		const int offsetY = (image.height - edge) / 2;

		Image out;
		out.width = edge;
		out.height = edge;
		out.pixels.resize(static_cast<size_t>(edge) * edge * 4);

		for (int y = 0; y < edge; ++y)
		{
			const uint8_t* src = image.pixels.data()
				+ (static_cast<size_t>(y + offsetY) * image.width + offsetX) * 4;

			uint8_t* dst = out.pixels.data() + static_cast<size_t>(y) * edge * 4;

			std::memcpy(dst, src, static_cast<size_t>(edge) * 4);
		}

		return out;
	}

	std::expected<Image, std::string> Resize(const Image& image, int width, int height)
	{
		if (!image.Valid())
			return std::unexpected("nothing to resize");

		if (width <= 0 || height <= 0)
			return std::unexpected("invalid resize target");

		if (width == image.width && height == image.height)
			return image;

		auto factory = CreateFactory();
		if (!factory)
			return std::unexpected(factory.error());

		auto bitmap = ToWicBitmap(factory->Get(), image);
		if (!bitmap)
			return std::unexpected(bitmap.error());

		ComPtr<IWICBitmapScaler> scaler;
		if (FAILED((*factory)->CreateBitmapScaler(&scaler)))
			return std::unexpected("could not create a WIC scaler");

		// Fant is WIC's high-quality downsampler -- it averages, so shrinking a
		// 4096 source to 512 doesn't alias the way a bilinear sampler would.
		if (FAILED(scaler->Initialize(bitmap->Get(), static_cast<UINT>(width), static_cast<UINT>(height),
			WICBitmapInterpolationModeFant)))
		{
			return std::unexpected("could not scale the image");
		}

		return ToImage(factory->Get(), scaler.Get());
	}

	std::expected<void, std::string> SavePNG(const Image& image, const fs::path& path)
	{
		if (!image.Valid())
			return std::unexpected("nothing to save");

		auto factory = CreateFactory();
		if (!factory)
			return std::unexpected(factory.error());

		ComPtr<IWICStream> stream;
		if (FAILED((*factory)->CreateStream(&stream)))
			return std::unexpected("could not create a WIC stream");

		if (FAILED(stream->InitializeFromFilename(path.c_str(), GENERIC_WRITE)))
			return std::unexpected(std::format("could not open {} for writing", path.string()));

		ComPtr<IWICBitmapEncoder> encoder;
		if (FAILED((*factory)->CreateEncoder(GUID_ContainerFormatPng, nullptr, &encoder)))
			return std::unexpected("could not create the PNG encoder");

		if (FAILED(encoder->Initialize(stream.Get(), WICBitmapEncoderNoCache)))
			return std::unexpected("could not initialise the PNG encoder");

		ComPtr<IWICBitmapFrameEncode> frame;
		ComPtr<IPropertyBag2> options;
		if (FAILED(encoder->CreateNewFrame(&frame, &options)))
			return std::unexpected("could not create the PNG frame");

		if (FAILED(frame->Initialize(options.Get())))
			return std::unexpected("could not initialise the PNG frame");

		if (FAILED(frame->SetSize(static_cast<UINT>(image.width), static_cast<UINT>(image.height))))
			return std::unexpected("could not set the PNG size");

		WICPixelFormatGUID format = GUID_WICPixelFormat32bppRGBA;
		if (FAILED(frame->SetPixelFormat(&format)))
			return std::unexpected("could not set the PNG pixel format");

		const UINT stride = static_cast<UINT>(image.width) * 4;
		if (FAILED(frame->WritePixels(static_cast<UINT>(image.height), stride,
			static_cast<UINT>(image.pixels.size()), const_cast<BYTE*>(image.pixels.data()))))
		{
			return std::unexpected("could not write the PNG pixels");
		}

		if (FAILED(frame->Commit()) || FAILED(encoder->Commit()))
			return std::unexpected("could not finish writing the PNG");

		return {};
	}
}
