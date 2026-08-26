#include "core/compiler.h"

#include "core/strings.h"

#include <windows.h>

#include <chrono>
#include <format>
#include <system_error>
#include <thread>

namespace fs = std::filesystem;

namespace bit
{
	std::mutex& SharedVMATCompileMutex()
	{
		static std::mutex mutex;
		return mutex;
	}

	fs::path ResourceCompilerPath(const fs::path& gameRoot)
	{
		return gameRoot / "game" / "bin" / "win64" / "resourcecompiler.exe";
	}

	bool RunResourceCompilerOnce(const fs::path& compiler, const fs::path& target,
		std::string& output, DWORD& exitCode)
	{
		output.clear();
		exitCode = 0;

		SECURITY_ATTRIBUTES sa = {};
		sa.nLength = sizeof(sa);
		sa.bInheritHandle = TRUE;

		HANDLE readPipe = nullptr;
		HANDLE writePipe = nullptr;
		if (!::CreatePipe(&readPipe, &writePipe, &sa, 0))
			return false;

		// Only the write end may be inherited, otherwise the read end never sees EOF.
		::SetHandleInformation(readPipe, HANDLE_FLAG_INHERIT, 0);

		STARTUPINFOW si = {};
		si.cb = sizeof(si);
		si.dwFlags = STARTF_USESTDHANDLES | STARTF_USESHOWWINDOW;
		si.wShowWindow = SW_HIDE;
		si.hStdOutput = writePipe;
		si.hStdError = writePipe;

		// resourcecompiler -i <path> -f
		std::wstring commandLine = std::format(L"\"{}\" -i \"{}\" -f",
			compiler.wstring(), target.wstring());

		PROCESS_INFORMATION pi = {};
		const BOOL started = ::CreateProcessW(nullptr, commandLine.data(), nullptr, nullptr,
			TRUE, CREATE_NO_WINDOW, nullptr, nullptr, &si, &pi);

		::CloseHandle(writePipe);

		if (!started)
		{
			::CloseHandle(readPipe);
			return false;
		}

		char buffer[4096];
		DWORD read = 0;
		while (::ReadFile(readPipe, buffer, sizeof(buffer), &read, nullptr) && read > 0)
			output.append(buffer, read);

		::CloseHandle(readPipe);

		::WaitForSingleObject(pi.hProcess, INFINITE);
		::GetExitCodeProcess(pi.hProcess, &exitCode);

		::CloseHandle(pi.hProcess);
		::CloseHandle(pi.hThread);

		return true;
	}

	CompileResult CompileTargets(const fs::path& gameRoot, bool lockShared, bool retry,
		const std::vector<fs::path>& targets)
	{
		CompileResult result;

		const fs::path compiler = ResourceCompilerPath(gameRoot);

		std::error_code ec;
		if (!fs::exists(compiler, ec))
		{
			result.log = std::format(
				"resourcecompiler.exe was not found at:\r\n{}\r\n\r\nMake sure CS2 Workshop Tools are installed.",
				compiler.string());
			return result;
		}

		for (const fs::path& target : targets)
		{
			const bool locked = lockShared && EqualsNoCase(target.extension().string(), ".vmat");

			std::unique_lock<std::mutex> guard(SharedVMATCompileMutex(), std::defer_lock);
			if (locked)
				guard.lock();

			std::string output;
			DWORD exitCode = 0;
			bool started = RunResourceCompilerOnce(compiler, target, output, exitCode);

			result.log += std::format("=== {} ===\r\n", target.filename().string());
			result.log += output;

			if (output.empty())
				result.log += "(no console output)\r\n";

			bool failed = !started || exitCode != 0;

			if (failed && retry)
			{
				result.log += "\r\n--- Automatic retry 1/1 after compiler error ---\r\n";
				std::this_thread::sleep_for(std::chrono::milliseconds(180));

				std::string retryOutput;
				DWORD retryExit = 0;
				const bool retryStarted = RunResourceCompilerOnce(compiler, target, retryOutput, retryExit);

				result.log += retryOutput;
				if (retryOutput.empty())
					result.log += "(no console output on retry)\r\n";

				if (retryStarted && retryExit == 0)
				{
					result.log += "\r\nRETRY RESULT: OK\r\n";
					failed = false;
				}
				else
				{
					result.log += std::format("\r\nRETRY ERROR: exit code {}\r\n", retryExit);
					exitCode = retryExit;
				}
			}

			if (locked)
				guard.unlock();

			if (failed)
			{
				result.log += std::format("\r\nERROR: resourcecompiler exited with {}\r\n", exitCode);
				return result;
			}

			result.log += "\r\n";
		}

		result.ok = true;
		return result;
	}
}
