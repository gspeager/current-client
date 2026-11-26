//go:build !windows

package platform

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func isolatedHome(t *testing.T, path string) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("PATH", path)
	return home
}

func TestInstallingTheTerminalCommand(t *testing.T) {
	home := isolatedHome(t, "/usr/bin:/bin")
	command := filepath.Join(home, ".local", "bin", "current-client")

	before, err := LauncherStatus("/Applications/Current Client.app/Contents/MacOS/current-client")
	if err != nil {
		t.Fatal(err)
	}
	if before.Installed || before.Path != command {
		t.Fatalf("before installing: %+v", before)
	}
	if _, err := os.Stat(command); err == nil {
		t.Fatal("reading the state wrote the command")
	}

	installed, err := InstallLauncher("/Applications/Current Client.app/Contents/MacOS/current-client")
	if err != nil {
		t.Fatal(err)
	}
	if !installed.Installed || installed.OnPath || installed.PathLine == "" {
		t.Fatalf("installed outside the PATH: %+v", installed)
	}
	if info, _ := os.Stat(command); info.Mode().Perm()&0o100 == 0 {
		t.Fatal("the command isn't executable")
	}

	t.Setenv("PATH", filepath.Dir(command)+":/usr/bin:/bin")
	again, err := InstallLauncher("/Applications/Current Client.app/Contents/MacOS/current-client")
	if err != nil {
		t.Fatal(err)
	}
	if !again.Installed || !again.OnPath || again.PathLine != "" || again.ByPackage {
		t.Fatalf("installed again on the PATH: %+v", again)
	}

	moved, err := LauncherStatus("/Users/me/Applications/Current Client.app/Contents/MacOS/current-client")
	if err != nil {
		t.Fatal(err)
	}
	if moved.Installed {
		t.Fatal("a command for another copy of the app is taken for this one's")
	}
	if replaced, err := InstallLauncher("/Users/me/Applications/Current Client.app/Contents/MacOS/current-client"); err != nil || !replaced.Installed {
		t.Fatalf("replacing another copy's command: %+v, %v", replaced, err)
	}

	removed, err := RemoveLauncher("/Users/me/Applications/Current Client.app/Contents/MacOS/current-client")
	if err != nil {
		t.Fatal(err)
	}
	if removed.Installed {
		t.Fatalf("removed: %+v", removed)
	}
	if _, err := os.Stat(command); err == nil {
		t.Fatal("the command is still there")
	}
}

func TestTheTerminalCommandLeavesOthersAlone(t *testing.T) {
	home := isolatedHome(t, "/usr/bin:/bin")
	command := filepath.Join(home, ".local", "bin", "current-client")
	if err := os.MkdirAll(filepath.Dir(command), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(command, []byte("#!/bin/sh\necho mine\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := InstallLauncher("/app"); err == nil {
		t.Fatal("installed over someone else's command")
	}
	if _, err := RemoveLauncher("/app"); err != nil {
		t.Fatal(err)
	}
	if content, _ := os.ReadFile(command); string(content) != "#!/bin/sh\necho mine\n" {
		t.Fatalf("someone else's command changed: %q", content)
	}
}

func TestTheTerminalCommandWhereItCantBeWritten(t *testing.T) {
	home := isolatedHome(t, "/usr/bin:/bin")
	if err := os.Mkdir(filepath.Join(home, ".local"), 0o500); err != nil {
		t.Fatal(err)
	}
	if _, err := InstallLauncher("/app"); err == nil {
		t.Fatal("installed into a folder that can't be written")
	}
}

func TestATerminalCommandAPackageInstalled(t *testing.T) {
	packaged := t.TempDir()
	if err := os.WriteFile(filepath.Join(packaged, "current-client"), []byte("\x7fELF"), 0o755); err != nil {
		t.Fatal(err)
	}
	isolatedHome(t, packaged+":/usr/bin:/bin")
	launcher, err := LauncherStatus("/opt/current-client/current-client")
	if err != nil {
		t.Fatal(err)
	}
	if !launcher.ByPackage || !launcher.Installed || launcher.Path != filepath.Join(packaged, "current-client") {
		t.Fatalf("a packaged command: %+v", launcher)
	}
}

func TestTheTerminalCommandOpensThePathAndReturns(t *testing.T) {
	home := isolatedHome(t, "/usr/bin:/bin")
	opened := filepath.Join(home, "opened")
	app := filepath.Join(home, "it's the app")
	// Writes what it was given once the command has returned, as an app that keeps running would.
	script := "#!/bin/sh\nsleep 0.3\nprintf '%s|' \"$PWD\" \"$@\" > " + shellQuote(opened) + "\n"
	if err := os.WriteFile(app, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	launcher, err := InstallLauncher(app)
	if err != nil {
		t.Fatal(err)
	}
	repo := t.TempDir()
	cmd := exec.Command(launcher.Path, ".", "a path")
	cmd.Dir = repo
	started := time.Now()
	if out, err := cmd.CombinedOutput(); err != nil || len(out) > 0 {
		t.Fatalf("the command: %q, %v", out, err)
	}
	if took := time.Since(started); took > 250*time.Millisecond {
		t.Fatalf("the command waited for the app: %v", took)
	}
	deadline := time.Now().Add(3 * time.Second)
	for {
		content, err := os.ReadFile(opened)
		if err == nil && strings.HasSuffix(string(content), "|") {
			dir, _ := filepath.EvalSymlinks(repo)
			if got := string(content); got != dir+"|.|a path|" && got != repo+"|.|a path|" {
				t.Fatalf("the app was given %q", got)
			}
			return
		}
		if time.Now().After(deadline) {
			t.Fatal("the app never ran")
		}
		time.Sleep(20 * time.Millisecond)
	}
}

func TestEditingAPathList(t *testing.T) {
	sep := string(os.PathListSeparator)
	if got := withDir("/a"+sep+"/b", "/c"); got != "/a"+sep+"/b"+sep+"/c" {
		t.Fatalf("adding: %q", got)
	}
	if got := withDir("/a"+sep+"/c/", "/c"); got != "/a"+sep+"/c/" {
		t.Fatalf("adding one that's there: %q", got)
	}
	if got := withDir("", "/c"); got != "/c" {
		t.Fatalf("adding to nothing: %q", got)
	}
	if got := withoutDir("/a"+sep+"/c"+sep+"/b", "/c"); got != "/a"+sep+"/b" {
		t.Fatalf("removing: %q", got)
	}
}
