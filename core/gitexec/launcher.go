package gitexec

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

func resolveDefault() (string, []string) {
	if runtime.GOOS != "windows" {
		return "git", nil
	}
	result, err := (&Executor{binary: "git"}).Run(context.Background(), Command{Args: []string{"--exec-path"}})
	if err != nil || result.ExitCode != 0 {
		return "git", nil
	}
	if binary, env, ok := gitForWindows(strings.TrimSpace(result.Stdout)); ok {
		return binary, env
	}
	return "git", nil
}

// gitForWindows finds the real git from Git for Windows' exec path,
// <root>/mingw64/libexec/git-core (clangarm64 on Arm64). The git.exe on PATH,
// in <root>/cmd, is a launcher that starts it, doubling every command's
// process cost. The environment is what the launcher would set, so ssh, shell
// scripts, Git LFS and credential helpers are still found.
func gitForWindows(execPath string) (binary string, env []string, ok bool) {
	prefix := filepath.Dir(filepath.Dir(filepath.Clean(execPath)))
	binDir := filepath.Join(prefix, "bin")
	binary = filepath.Join(binDir, "git.exe")
	if info, err := os.Stat(binary); err != nil || info.IsDir() {
		return "", nil, false
	}
	usrBin := filepath.Join(filepath.Dir(prefix), "usr", "bin")
	env = []string{"PATH=" + strings.Join([]string{binDir, usrBin, os.Getenv("PATH")}, string(os.PathListSeparator))}
	if os.Getenv("MSYSTEM") == "" {
		env = append(env, "MSYSTEM="+strings.ToUpper(filepath.Base(prefix)))
	}
	return binary, env, true
}
