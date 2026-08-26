cmake_minimum_required(VERSION 3.18)
project(bit_texture_tool C CXX)

set(CMAKE_MSVC_RUNTIME_LIBRARY "MultiThreaded$<$<CONFIG:Debug>:Debug>")

if (NOT WIN32)
    message(FATAL_ERROR "B.I.T. Texture Tool targets Win32 + D3D11; Windows is the only supported target.")
endif ()

include(makefiles/shared.cmake)
include(makefiles/imgui.cmake)

file(GLOB_RECURSE SOURCE_FILES
    src/*.cpp
    src/*.h
    src/*.hpp
)

add_executable(${PROJECT_NAME} WIN32
    ${SOURCE_FILES}
    resource/app.rc
)

find_package(Git QUIET)
if (GIT_FOUND)
    execute_process(
        COMMAND ${GIT_EXECUTABLE} rev-parse --short HEAD
        WORKING_DIRECTORY ${CMAKE_SOURCE_DIR}
        OUTPUT_VARIABLE GIT_SHA
        OUTPUT_STRIP_TRAILING_WHITESPACE
        ERROR_QUIET
    )
else ()
    set(GIT_SHA "nogit")
endif ()

include(${CMAKE_SOURCE_DIR}/makefiles/windows.base.cmake)

set_target_properties(${PROJECT_NAME} PROPERTIES
    OUTPUT_NAME "BIT_Texture_Tool"
    RUNTIME_OUTPUT_DIRECTORY "${CMAKE_BINARY_DIR}/bin"
)

target_link_libraries(${PROJECT_NAME} imgui ${LINK_LIBRARIES})
