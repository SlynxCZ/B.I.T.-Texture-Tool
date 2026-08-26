// B.I.T. Texture Tool -- C++ port of Tabo's Go original.
// Original: https://github.com/SlynxCZ/B.I.T.-Texture-Tool (MIT)

#include "ui/app.h"
#include "ui/texture.h"

#include "imgui.h"
#include "imgui_impl_dx11.h"
#include "imgui_impl_win32.h"

#include <d3d11.h>
#include <windows.h>

#include <cstdio>

namespace
{
	ID3D11Device* g_pDevice = nullptr;
	ID3D11DeviceContext* g_pContext = nullptr;
	IDXGISwapChain* g_pSwapChain = nullptr;
	ID3D11RenderTargetView* g_pRenderTarget = nullptr;

	UINT g_ResizeWidth = 0;
	UINT g_ResizeHeight = 0;

	void CreateRenderTarget()
	{
		ID3D11Texture2D* pBackBuffer = nullptr;
		g_pSwapChain->GetBuffer(0, IID_PPV_ARGS(&pBackBuffer));
		if (!pBackBuffer)
			return;

		g_pDevice->CreateRenderTargetView(pBackBuffer, nullptr, &g_pRenderTarget);
		pBackBuffer->Release();
	}

	void CleanupRenderTarget()
	{
		if (g_pRenderTarget)
		{
			g_pRenderTarget->Release();
			g_pRenderTarget = nullptr;
		}
	}

	bool CreateDeviceD3D(HWND hWnd)
	{
		DXGI_SWAP_CHAIN_DESC desc = {};
		desc.BufferCount = 2;
		desc.BufferDesc.Format = DXGI_FORMAT_R8G8B8A8_UNORM;
		desc.BufferDesc.RefreshRate.Numerator = 60;
		desc.BufferDesc.RefreshRate.Denominator = 1;
		desc.Flags = DXGI_SWAP_CHAIN_FLAG_ALLOW_MODE_SWITCH;
		desc.BufferUsage = DXGI_USAGE_RENDER_TARGET_OUTPUT;
		desc.OutputWindow = hWnd;
		desc.SampleDesc.Count = 1;
		desc.Windowed = TRUE;
		desc.SwapEffect = DXGI_SWAP_EFFECT_DISCARD;

		constexpr D3D_FEATURE_LEVEL levels[] = { D3D_FEATURE_LEVEL_11_0, D3D_FEATURE_LEVEL_10_0 };
		D3D_FEATURE_LEVEL obtained = {};

		HRESULT hr = ::D3D11CreateDeviceAndSwapChain(nullptr, D3D_DRIVER_TYPE_HARDWARE, nullptr, 0,
			levels, static_cast<UINT>(std::size(levels)), D3D11_SDK_VERSION,
			&desc, &g_pSwapChain, &g_pDevice, &obtained, &g_pContext);

		// Machines without a usable GPU driver still get a window through WARP.
		if (hr == DXGI_ERROR_UNSUPPORTED)
		{
			hr = ::D3D11CreateDeviceAndSwapChain(nullptr, D3D_DRIVER_TYPE_WARP, nullptr, 0,
				levels, static_cast<UINT>(std::size(levels)), D3D11_SDK_VERSION,
				&desc, &g_pSwapChain, &g_pDevice, &obtained, &g_pContext);
		}

		if (FAILED(hr))
			return false;

		CreateRenderTarget();
		return true;
	}

	void CleanupDeviceD3D()
	{
		CleanupRenderTarget();

		if (g_pSwapChain) { g_pSwapChain->Release(); g_pSwapChain = nullptr; }
		if (g_pContext)   { g_pContext->Release();   g_pContext = nullptr; }
		if (g_pDevice)    { g_pDevice->Release();    g_pDevice = nullptr; }
	}
}

extern IMGUI_IMPL_API LRESULT ImGui_ImplWin32_WndProcHandler(HWND hWnd, UINT msg, WPARAM wParam, LPARAM lParam);

static LRESULT WINAPI WndProc(HWND hWnd, UINT msg, WPARAM wParam, LPARAM lParam)
{
	if (ImGui_ImplWin32_WndProcHandler(hWnd, msg, wParam, lParam))
		return true;

	switch (msg)
	{
	case WM_SIZE:
		if (wParam == SIZE_MINIMIZED)
			return 0;
		g_ResizeWidth = static_cast<UINT>(LOWORD(lParam));
		g_ResizeHeight = static_cast<UINT>(HIWORD(lParam));
		return 0;

	case WM_SYSCOMMAND:
		// Swallow the Alt-menu activation, it steals keyboard focus from ImGui.
		if ((wParam & 0xfff0) == SC_KEYMENU)
			return 0;
		break;

	case WM_DESTROY:
		::PostQuitMessage(0);
		return 0;

	default:
		break;
	}

	return ::DefWindowProcW(hWnd, msg, wParam, lParam);
}

int WINAPI wWinMain(HINSTANCE hInstance, HINSTANCE, LPWSTR, int)
{
	// IFileOpenDialog and WIC both need an apartment on whichever thread calls
	// them; the pipeline worker initialises its own.
	const HRESULT comInit = ::CoInitializeEx(nullptr, COINIT_APARTMENTTHREADED | COINIT_DISABLE_OLE1DDE);

	WNDCLASSEXW wc = {};
	wc.cbSize = sizeof(wc);
	wc.style = CS_CLASSDC;
	wc.lpfnWndProc = WndProc;
	wc.hInstance = hInstance;
	wc.hCursor = ::LoadCursorW(nullptr, IDC_ARROW);
	wc.lpszClassName = L"BITTextureToolWindow";
	::RegisterClassExW(&wc);

	const HWND hWnd = ::CreateWindowW(wc.lpszClassName, L"B.I.T. Texture Tool",
		WS_OVERLAPPEDWINDOW, 100, 100, 1100, 760,
		nullptr, nullptr, hInstance, nullptr);

	if (!CreateDeviceD3D(hWnd))
	{
		CleanupDeviceD3D();
		::UnregisterClassW(wc.lpszClassName, hInstance);
		::MessageBoxW(nullptr, L"Failed to create a Direct3D 11 device.", L"B.I.T. Texture Tool", MB_ICONERROR);

		if (SUCCEEDED(comInit))
			::CoUninitialize();

		return 1;
	}

	::ShowWindow(hWnd, SW_SHOWDEFAULT);
	::UpdateWindow(hWnd);

	IMGUI_CHECKVERSION();
	ImGui::CreateContext();

	ImGuiIO& io = ImGui::GetIO();
	io.ConfigFlags |= ImGuiConfigFlags_NavEnableKeyboard;
	io.ConfigFlags |= ImGuiConfigFlags_DockingEnable;
	io.IniFilename = nullptr;

	ImGui_ImplWin32_Init(hWnd);
	ImGui_ImplDX11_Init(g_pDevice, g_pContext);

	bit::SetRenderDevice(g_pDevice);

	// Theme and language come out of settings.json; DrawApp applies the style
	// on the first frame and again whenever the toggle flips.
	bit::AppState state;
	bit::InitApp(state);

	constexpr float kClear[4] = { 0.09f, 0.09f, 0.11f, 1.00f };

	bool running = true;
	while (running)
	{
		MSG msg;
		while (::PeekMessageW(&msg, nullptr, 0, 0, PM_REMOVE))
		{
			::TranslateMessage(&msg);
			::DispatchMessageW(&msg);

			if (msg.message == WM_QUIT)
				running = false;
		}
		if (!running)
			break;

		if (g_ResizeWidth != 0 && g_ResizeHeight != 0)
		{
			CleanupRenderTarget();
			g_pSwapChain->ResizeBuffers(0, g_ResizeWidth, g_ResizeHeight, DXGI_FORMAT_UNKNOWN, 0);
			g_ResizeWidth = 0;
			g_ResizeHeight = 0;
			CreateRenderTarget();
		}

		ImGui_ImplDX11_NewFrame();
		ImGui_ImplWin32_NewFrame();
		ImGui::NewFrame();

		bit::DrawApp(state);

		ImGui::Render();

		g_pContext->OMSetRenderTargets(1, &g_pRenderTarget, nullptr);
		g_pContext->ClearRenderTargetView(g_pRenderTarget, kClear);
		ImGui_ImplDX11_RenderDrawData(ImGui::GetDrawData());

		g_pSwapChain->Present(1, 0);
	}

	// Textures have to go before the device they were created on.
	bit::ShutdownApp(state);
	bit::SetRenderDevice(nullptr);

	ImGui_ImplDX11_Shutdown();
	ImGui_ImplWin32_Shutdown();
	ImGui::DestroyContext();

	CleanupDeviceD3D();
	::DestroyWindow(hWnd);
	::UnregisterClassW(wc.lpszClassName, hInstance);

	if (SUCCEEDED(comInit))
		::CoUninitialize();

	return 0;
}
