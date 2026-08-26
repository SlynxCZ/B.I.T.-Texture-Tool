# Dear ImGui with the Win32 + D3D11 backend -- the standalone-desktop equivalent
# of what CS2ServerGUI does inside the game process. ImGui ships no build files
# of its own, so the target is declared here.

set(IMGUI_DIR ${CMAKE_SOURCE_DIR}/vendor/imgui)

if (NOT EXISTS ${IMGUI_DIR}/imgui.cpp)
    message(FATAL_ERROR "vendor/imgui is empty -- run: git submodule update --init --recursive")
endif ()

add_library(imgui STATIC
    ${IMGUI_DIR}/imgui.cpp
    ${IMGUI_DIR}/imgui_draw.cpp
    ${IMGUI_DIR}/imgui_tables.cpp
    ${IMGUI_DIR}/imgui_widgets.cpp
    ${IMGUI_DIR}/imgui_demo.cpp
    ${IMGUI_DIR}/backends/imgui_impl_win32.cpp
    ${IMGUI_DIR}/backends/imgui_impl_dx11.cpp
)

target_include_directories(imgui PUBLIC
    ${IMGUI_DIR}
    ${IMGUI_DIR}/backends
)

target_compile_features(imgui PUBLIC cxx_std_20)
target_link_libraries(imgui PUBLIC d3d11 dxgi d3dcompiler)

set_target_properties(imgui PROPERTIES FOLDER vendor)
