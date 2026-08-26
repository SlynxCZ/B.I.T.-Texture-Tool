#pragma once

#include <filesystem>
#include <mutex>
#include <string>
#include <vector>

namespace bit
{
	struct CompileResult
	{
		bool ok = false;
		std::string log;
	};

	// game/bin/win64/resourcecompiler.exe under the game root.
	std::filesystem::path ResourceCompilerPath(const std::filesystem::path& gameRoot);

	// Runs resourcecompiler once and captures its combined stdout+stderr with no
	// console window. exitCode is the process exit code; false means it never started.
	bool RunResourceCompilerOnce(const std::filesystem::path& compiler,
		const std::filesystem::path& target, std::string& output, DWORD& exitCode);

	// Compiles each path in order, stopping at the first failure.
	//
	// lockShared serializes .vmat compilation only: a VMAT touches shared default
	// normal/AO resources, so two of them at once can collide, while the unique
	// per-texture VTEX files are safe to run in parallel.
	//
	// retry gives one more attempt after a short pause, which clears most of the
	// transient compiler failures seen under load.
	CompileResult CompileTargets(const std::filesystem::path& gameRoot,
		bool lockShared, bool retry, const std::vector<std::filesystem::path>& targets);

	// The mutex CompileTargets uses for the .vmat phase. Exposed because the
	// batch path takes it around its own compile calls.
	std::mutex& SharedVMATCompileMutex();
}
