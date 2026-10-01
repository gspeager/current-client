package platform

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestDiagnosticsHidesRepositoryPathsAndHomeFolder(t *testing.T) {
	home := filepath.Join("C:", "Users", "ada")
	repo := filepath.Join(home, "src", "secret-project")
	nested := filepath.Join(repo, "vendor", "lib")

	got := Diagnostics(DiagnosticsInput{
		AppVersion:      "1.2.3",
		GitVersion:      "2.45.1.windows.1",
		GitPath:         filepath.Join(home, "git", "cmd", "git.exe"),
		SettingsVersion: 1,
		HomeDir:         home,
		RepoPaths:       []string{repo, nested},
		RecentErrors: []RecentError{
			{At: "2026-09-25T10:00:00Z", Message: "open " + filepath.Join(nested, "a.txt") + ": access denied"},
			{At: "2026-09-25T09:00:00Z", Message: "fatal: '" + filepath.ToSlash(repo) + "' is locked"},
		},
	})

	for _, leaked := range []string{"secret-project", "ada"} {
		if strings.Contains(got, leaked) {
			t.Fatalf("diagnostics leak %q:\n%s", leaked, got)
		}
	}
	for _, want := range []string{
		"App version: 1.2.3",
		"Git: 2.45.1.windows.1 at " + filepath.Join("~", "git", "cmd", "git.exe"),
		"Settings format: 1",
		"open " + filepath.Join("<repository>", "a.txt") + ": access denied",
		"'<repository>' is locked",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("diagnostics missing %q:\n%s", want, got)
		}
	}
}

func TestDiagnosticsReportsAGitProblemAndNoErrors(t *testing.T) {
	got := Diagnostics(DiagnosticsInput{AppVersion: "dev", GitProblem: "Git not found.", SettingsVersion: 1})

	if !strings.Contains(got, "Git: Git not found.") || !strings.Contains(got, "Recent errors:\n  none") {
		t.Fatalf("diagnostics =\n%s", got)
	}
}
