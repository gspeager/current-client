package git

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/gspeager/current-client/core/internal/gittest"
)

func TestCurrentUserReadsConfiguredIdentity(t *testing.T) {
	dir := t.TempDir()
	gittest.Run(t, dir, "init", "-q", "-b", "main")
	gittest.Run(t, dir, "config", "user.name", "Test User")
	gittest.Run(t, dir, "config", "user.email", "test@example.com")

	got := CurrentUser(context.Background(), dir)
	want := UserIdentity{Name: "Test User", Email: "test@example.com"}
	if got != want {
		t.Fatalf("got %+v, want %+v", got, want)
	}
}

func TestConfigValueHandlesUnsetKey(t *testing.T) {
	dir := t.TempDir()
	gittest.Run(t, dir, "init", "-q", "-b", "main")

	if got := configValue(context.Background(), dir, "definitely.not.a.real.key"); got != "" {
		t.Fatalf("got %q, want empty string for an unset key", got)
	}
}

// GIT_CONFIG_GLOBAL keeps these tests away from the real ~/.gitconfig.
func TestGlobalUserRoundTrip(t *testing.T) {
	t.Setenv("GIT_CONFIG_GLOBAL", filepath.Join(t.TempDir(), "gitconfig"))

	if err := SetGlobalUser(context.Background(), "Global Name", "global@example.com"); err != nil {
		t.Fatalf("SetGlobalUser: %v", err)
	}

	got := GlobalUser(context.Background())
	want := UserIdentity{Name: "Global Name", Email: "global@example.com"}
	if got != want {
		t.Fatalf("got %+v, want %+v", got, want)
	}
}

func TestGlobalUserEmptyWhenUnset(t *testing.T) {
	t.Setenv("GIT_CONFIG_GLOBAL", filepath.Join(t.TempDir(), "gitconfig"))

	if got := GlobalUser(context.Background()); got != (UserIdentity{}) {
		t.Fatalf("got %+v, want an empty identity", got)
	}
}

func TestSetRepoUserOverridesGlobalInThatRepo(t *testing.T) {
	t.Setenv("GIT_CONFIG_GLOBAL", filepath.Join(t.TempDir(), "gitconfig"))
	dir := t.TempDir()
	gittest.Run(t, dir, "init", "-q", "-b", "main")
	if err := SetGlobalUser(context.Background(), "Global Name", "global@example.com"); err != nil {
		t.Fatalf("SetGlobalUser: %v", err)
	}

	if err := SetRepoUser(context.Background(), dir, "Repo Name", "repo@example.com"); err != nil {
		t.Fatalf("SetRepoUser: %v", err)
	}

	want := UserIdentity{Name: "Repo Name", Email: "repo@example.com"}
	if got := CurrentUser(context.Background(), dir); got != want {
		t.Fatalf("CurrentUser = %+v, want %+v", got, want)
	}
	if got := GlobalUser(context.Background()); got.Name != "Global Name" {
		t.Fatalf("GlobalUser = %+v, want the global identity untouched", got)
	}
}
