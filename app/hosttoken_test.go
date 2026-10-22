package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gspeager/current-client/core/git"
)

func TestSaveTokenRecordsOnlyThatOneExists(t *testing.T) {
	settingsPath := useTempSettings(t)
	home := os.Getenv("HOME")
	storeFile := filepath.Join(home, "credentials")
	gitConfig := "[credential]\n\thelper = store --file=" + filepath.ToSlash(storeFile) + "\n"
	if err := os.WriteFile(filepath.Join(home, ".gitconfig"), []byte(gitConfig), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	var s HostService
	remote := git.Remote{Name: "origin", FetchURL: "git@gitlab.example.com:group/repo.git"}

	if err := s.SaveToken("gitlab.example.com", "glpat-123"); err != nil {
		t.Fatalf("SaveToken: %v", err)
	}
	settings, _ := os.ReadFile(settingsPath)
	if strings.Contains(string(settings), "glpat") || !strings.Contains(string(settings), `"hasToken": true`) {
		t.Fatalf("settings file = %s", settings)
	}
	if info, _ := hostInfoFor(remote, configOrDefault().Hosts); !info.HasToken {
		t.Fatalf("HasToken not reported: %+v", info)
	}
	stored, _ := os.ReadFile(storeFile)
	if !strings.Contains(string(stored), "current-client-api:glpat-123@api.gitlab.example.com") {
		t.Fatalf("store file = %q", stored)
	}

	if err := s.RemoveToken("gitlab.example.com"); err != nil {
		t.Fatalf("RemoveToken: %v", err)
	}
	if hosts := configOrDefault().Hosts; len(hosts) != 0 {
		t.Fatalf("after RemoveToken the entry should be gone: %+v", hosts)
	}
	if stored, _ := os.ReadFile(storeFile); strings.Contains(string(stored), "glpat") {
		t.Fatalf("token still stored: %q", stored)
	}
}
