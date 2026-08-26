add_definitions(
    -DWIN32 -D_WINDOWS -DUNICODE -D_UNICODE
    -DWIN32_LEAN_AND_MEAN -DNOMINMAX
    -D_CRT_SECURE_NO_WARNINGS=1 -D_CRT_SECURE_NO_DEPRECATE=1 -D_CRT_NONSTDC_NO_DEPRECATE=1
)

set(CMAKE_CXX_FLAGS_RELEASE "${CMAKE_CXX_FLAGS_RELEASE} /Zi")
set(CMAKE_CXX_FLAGS "${CMAKE_CXX_FLAGS} /W3 /permissive- /utf-8 /MP /EHsc")
set(CMAKE_EXE_LINKER_FLAGS_RELEASE "${CMAKE_EXE_LINKER_FLAGS_RELEASE} /OPT:REF /OPT:ICF")
set(CMAKE_EXE_LINKER_FLAGS_DEBUG "${CMAKE_EXE_LINKER_FLAGS_DEBUG} /NODEFAULTLIB:libcmt")

# comctl32/comdlg32: common controls and the file/folder pickers.
# windowscodecs: WIC, the image decoder that replaces Go's image/* packages.
# shlwapi/shell32/ole32: path helpers, ShellExecute, COM init for WIC.
set(LINK_LIBRARIES
    d3d11
    dxgi
    comctl32
    comdlg32
    windowscodecs
    shlwapi
    shell32
    ole32
    user32
    gdi32
    advapi32
)
