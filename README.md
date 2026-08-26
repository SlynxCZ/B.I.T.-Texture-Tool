# B.I.T. Texture Tool

Windows utility for importing custom textures into Source 2 games — image in,
compiled Hammer-ready material out.

Written in C++20 with a [Dear ImGui](https://github.com/ocornut/imgui) interface
on a Win32 + D3D11 backend.

## Building

### Prerequisites

| | |
| --- | --- |
| Visual Studio 2022 | with the **Desktop development with C++** workload |
| Windows 10 SDK | 10.0.19041 or newer (pulled in by that workload) |
| CMake | 3.18 or newer |
| Git | for the ImGui submodule |

Windows x64 only. The tool drives `resourcecompiler.exe` and renders through
D3D11, so there is nothing to cross-compile.

### Clone

The ImGui submodule is not optional — the build stops with a clear error if it
is missing.

```
git clone --recursive https://github.com/tabo420/B.I.T.-Texture-Tool.git
```

Already cloned without `--recursive`:

```
git submodule update --init --recursive
```

### Configure and build

```
cmake -S . -B build -A x64
cmake --build build --config Release
```

The executable lands at `build/bin/Release/BIT_Texture_Tool.exe`. Swap
`Release` for `Debug` for a debug build.

### Ninja instead of MSBuild

From a *x64 Native Tools Command Prompt for VS 2022*:

```
cmake -S . -B build -G Ninja -DCMAKE_BUILD_TYPE=Release
cmake --build build
```

### IDE

Open the folder directly in Visual Studio 2022 or CLion — both read
`CMakeLists.txt` and need no generated solution. The `bit_texture_tool` target
is the app; `imgui` is the vendored static library.

### Version stamping

`SEMVER` and `GITHUB_SHA_SHORT` are read from the environment at configure time
and compiled in; both default to `Local` when unset.

## Layout

```
makefiles/     cmake fragments: shared, imgui, windows.base
resource/      app.rc, manifest, and the UI artwork compiled into the binary
src/core/      game detection, material generation, resourcecompiler — no UI
src/ui/        ImGui frame
src/main.cpp   Win32 + D3D11 bootstrap
vendor/imgui   submodule, docking branch
```

`src/core` deliberately knows nothing about ImGui: the workflow is the part
worth testing, and it shouldn't need a window to run.

UI artwork is `RCDATA` in `resource/app.rc` — PNG bytes compiled into the binary
and decoded through WIC at startup, so the alpha channel survives.

## Status

Done:

| | |
| --- | --- |
| `core/strings` | UTF-8 ↔ UTF-16, trim, case-insensitive compare |
| `core/games` | game profiles, `libraryfolders.vdf` parsing, root detection, addon enumeration |
| `core/material` | name sanitizing, `.vtex` DMX and `.vmat` KV3 generation, all three material modes |
| `core/compiler` | `resourcecompiler.exe` with piped output, the shared-VMAT lock and the one automatic retry |
| `core/image` | WIC decode, alpha detection, centre crop to square, Fant resize, PNG writeback |
| `core/pipeline` | the create-material job: image → PNG → `.vtex` → `.vmat` → compile, with the compile log written next to the output |
| `core/settings` | `settings.json` under `%APPDATA%`, junk folder, same field names as before |
| `core/batch` | Autonomous Production: worker pool, speed presets, overwrite modes, cancellation |
| `i18n` | 9 languages, English fallback per key |
| `ui/theme` | light and dark |
| `ui/texture` | decoded pixels to a D3D11 shader resource view |
| `main.cpp` | window, device, frame loop |
| `ui/app` | the whole workflow — pickers, material settings, batch, threaded jobs, progress, log |

Behaviour matches the previous implementation, including the parts that aren't
obvious from the outside — the VMAT-only compile lock, the 180 ms retry pause,
the `.vmat`/`.vtex`/`.png` suffix stripping in material names, and skipping UNC
paths during Steam library detection so probing can't block on a network timeout.

Both paths work end to end. A single material: pick an image, name it, choose
type and resolution, hit Create. Autonomous Production: point it at a folder and
it processes every image WIC can read, honouring the speed preset and overwrite
mode, with a progress bar and a stop button that lets in-flight items finish.

Jobs run on worker threads, so the window stays responsive while Resource
Compiler runs. Compile output shows in the UI and is written next to the result
as `<name>_compile_log.txt`.

Worker counts follow the original's tuning: Slow and Normal stay single-threaded
as stable reference modes, Fast and Extreme scale with CPU count but are capped
by output resolution — a 4K RGBA buffer is 64 MB per worker, and that spike is
what actually breaks large batches. Resource Compiler gets its own, smaller
budget on top.

The picked image is previewed at the crop the pipeline will actually use, over a
checkerboard so an alpha cutout reads as transparent rather than blending into
the window. The logo swaps with the theme and the language picker carries its
flags, all decoded out of the binary's `RCDATA` at first frame.

Nothing from the original is missing.

## Supported games

Counter-Strike 2 is the primary target. Team Fortress 2, Deadlock and
Half-Life: Alyx have profiles and addon-directory handling; support differs
between them because Valve games use different material systems and folder
layouts.

## License

MIT — see `LICENSE`.

Not affiliated with, sponsored by, or endorsed by Valve Corporation.
Counter-Strike, Source, Hammer and related names are Valve trademarks.
