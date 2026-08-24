# B.I.T. Texture Tool

**B.I.T. Texture Tool** is a free and open-source Windows utility made for Valve mappers and content creators.

Its main focus is **Counter-Strike 2**, but it also supports workflows for:

- Counter-Strike 2
- Team Fortress 2
- Deadlock
- Dota 2

B.I.T. is designed to make importing and preparing custom textures much faster by automating repetitive conversion, material creation, compilation, and batch-processing tasks.

> Made by Tabo using ChatGPT

---

## What does it do?

Importing custom textures into Valve games can involve several manual steps.

B.I.T. simplifies the process by handling as much of the workflow as possible automatically.

For Source 2 games, the workflow can include:

**Image → VTEX → VMAT → Resource Compiler → Hammer-ready material**

The exact workflow depends on the selected game.

---

## Supported Games

### Counter-Strike 2

**Primary supported game.**

B.I.T. can prepare and compile materials for CS2 / Source 2 Hammer projects.

### Team Fortress 2

B.I.T. also includes support for Team Fortress 2 texture workflows.

### Deadlock

B.I.T. supports Deadlock project and texture workflows.

### Dota 2

B.I.T. supports Dota 2 project and texture workflows.

Support may differ between games because Valve games use different material systems, tools, and folder structures.

---

## Features

- Import PNG, JPG, JPEG, BMP, GIF and TGA images
- Automatically convert images when needed
- Automatically crop non-square images
- Keep original image resolution or resize to:
  - 512
  - 1024
  - 2048
  - 4096
- Automatic material naming from the image filename
- Automatic material/source file generation
- Resource Compiler integration for supported Source 2 games
- Automatic game/tool path detection
- Addon/project selection
- Existing material replacement
- Compile logs
- Temporary/Junk file management

---

## Material Types

Depending on the selected game and workflow, B.I.T. provides material options such as:

- Normal / Opaque
- Cutout Transparency
- See-through Transparency

---

## Autonomous Production

**Autonomous Production** allows B.I.T. to process an entire folder of textures automatically.

This is useful when importing dozens or hundreds of textures for a map or project.

Available speed modes:

- Experimental Slow
- Normal
- Experimental Fast
- Experimental Extreme
- Custom

### Custom workers

Advanced users can manually select the number of workers used during batch processing.

B.I.T. can also automatically adjust worker usage depending on texture resolution and selected speed mode.

Using more workers does not always mean better performance, so the built-in presets are designed to balance speed, CPU usage, RAM usage, disk access, and compiler stability.

---

## Existing Materials

When B.I.T. detects materials with the same name, you can choose how they should be handled:

- **Ask**
- **Skip existing**
- **Replace existing**

This makes it easy to reprocess a texture using a different image, quality setting, or material type without manually cleaning up the old files.

---

## Reliability Features

B.I.T. includes several systems intended to make large texture batches more reliable:

- Automatic compile retry
- Compiler locking for shared resources
- Hammer-safe material replacement
- Existing material detection
- Batch failure handling
- Compile logging
- Worker/concurrency control

A failed texture does not have to stop an entire Autonomous Production batch.

---

## Interface

B.I.T. includes:

- Light mode
- Dark mode
- Multiple interface languages
- Automatic game detection
- Addon/project selection
- Output quality settings
- Material type settings
- Autonomous speed settings
- Custom worker control
- Junk folder controls
- Compile logs

---

## Basic Usage

1. Launch B.I.T. Texture Tool.
2. Select your game.
3. Detect or select the game's tools/project directory.
4. Select your addon or project.
5. Choose an image.
6. Choose the material type.
7. Choose the output quality.
8. Create the material.
9. Use the generated texture/material in your mapping tools.

For large texture collections, use **Autonomous Production** and select an entire folder.

---

## Building From Source

B.I.T. Texture Tool is written in **Go (Golang)** and uses the native Windows **Win32 API** for its interface.

To build a 64-bit Windows GUI executable:

```bash
GOOS=windows GOARCH=amd64 go build -ldflags="-H=windowsgui" -o BIT_Texture_Tool.exe main.go
