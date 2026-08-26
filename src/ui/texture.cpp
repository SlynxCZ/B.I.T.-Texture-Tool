#include "ui/texture.h"

#include <d3d11.h>

#include <utility>

namespace bit
{
	namespace
	{
		ID3D11Device* g_pDevice = nullptr;
	}

	void SetRenderDevice(ID3D11Device* device)
	{
		g_pDevice = device;
	}

	GpuTexture::~GpuTexture()
	{
		Reset();
	}

	GpuTexture::GpuTexture(GpuTexture&& other) noexcept
		: m_pView(std::exchange(other.m_pView, nullptr))
		, m_width(std::exchange(other.m_width, 0))
		, m_height(std::exchange(other.m_height, 0))
	{
	}

	GpuTexture& GpuTexture::operator=(GpuTexture&& other) noexcept
	{
		if (this != &other)
		{
			Reset();
			m_pView = std::exchange(other.m_pView, nullptr);
			m_width = std::exchange(other.m_width, 0);
			m_height = std::exchange(other.m_height, 0);
		}
		return *this;
	}

	void GpuTexture::Reset()
	{
		if (m_pView)
		{
			m_pView->Release();
			m_pView = nullptr;
		}
		m_width = 0;
		m_height = 0;
	}

	bool GpuTexture::Upload(const Image& image)
	{
		Reset();

		if (!g_pDevice || !image.Valid())
			return false;

		D3D11_TEXTURE2D_DESC desc = {};
		desc.Width = static_cast<UINT>(image.width);
		desc.Height = static_cast<UINT>(image.height);
		desc.MipLevels = 1;
		desc.ArraySize = 1;
		desc.Format = DXGI_FORMAT_R8G8B8A8_UNORM;
		desc.SampleDesc.Count = 1;
		desc.Usage = D3D11_USAGE_IMMUTABLE;
		desc.BindFlags = D3D11_BIND_SHADER_RESOURCE;

		D3D11_SUBRESOURCE_DATA data = {};
		data.pSysMem = image.pixels.data();
		data.SysMemPitch = static_cast<UINT>(image.width) * 4;

		ID3D11Texture2D* texture = nullptr;
		if (FAILED(g_pDevice->CreateTexture2D(&desc, &data, &texture)) || !texture)
			return false;

		D3D11_SHADER_RESOURCE_VIEW_DESC viewDesc = {};
		viewDesc.Format = desc.Format;
		viewDesc.ViewDimension = D3D11_SRV_DIMENSION_TEXTURE2D;
		viewDesc.Texture2D.MipLevels = 1;

		const HRESULT hr = g_pDevice->CreateShaderResourceView(texture, &viewDesc, &m_pView);

		// The view holds its own reference; the texture object itself is no
		// longer needed here.
		texture->Release();

		if (FAILED(hr))
		{
			m_pView = nullptr;
			return false;
		}

		m_width = image.width;
		m_height = image.height;
		return true;
	}
}
