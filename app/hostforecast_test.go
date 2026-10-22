//go:build !nohost

package app

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gspeager/current-client/core/hostapi"
	"github.com/gspeager/current-client/core/hostapi/hostapitest"
	"github.com/gspeager/current-client/internal/config"
)

func writeAndCommit(t *testing.T, repo, name, content, message string) string {
	t.Helper()
	if err := os.WriteFile(filepath.Join(repo, name), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	gitIn(t, repo, "add", name)
	gitIn(t, repo, "commit", "-q", "-m", message)
	return strings.TrimSpace(string(mustOutput(t, repo, "rev-parse", "HEAD")))
}

func TestForecastNamesTheConflictingChangeAndSkipsTheRest(t *testing.T) {
	useTempSettings(t)
	swapHostClient(t, hostapitest.Forbid(t))
	repo, hostname, _ := hostedRemote(t)
	writeAndCommit(t, repo, "shared.txt", "base\n", "base")
	gitIn(t, repo, "push", "-q", "origin", "main")

	gitIn(t, repo, "switch", "-q", "-c", "conflicting")
	conflicting := writeAndCommit(t, repo, "shared.txt", "theirs\n", "theirs")
	gitIn(t, repo, "push", "-q", "origin", "HEAD:refs/merge-requests/1/head")
	gitIn(t, repo, "switch", "-q", "-c", "clean", "main")
	clean := writeAndCommit(t, repo, "other.txt", "new\n", "other")
	gitIn(t, repo, "push", "-q", "origin", "HEAD:refs/merge-requests/2/head")
	gitIn(t, repo, "switch", "-q", "-c", "elsewhere", "main")
	elsewhere := writeAndCommit(t, repo, "shared.txt", "also theirs\n", "other base")
	gitIn(t, repo, "push", "-q", "origin", "HEAD:refs/merge-requests/3/head")
	gitIn(t, repo, "switch", "-q", "main")
	writeAndCommit(t, repo, "shared.txt", "ours\n", "ours")
	gitIn(t, repo, "branch", "-q", "-D", "conflicting", "clean", "elsewhere")
	gitIn(t, repo, "symbolic-ref", "refs/remotes/origin/HEAD", "refs/remotes/origin/main")

	if err := updateConfig(func(cfg *config.Config) {
		cfg.Hosts = map[string]config.HostConfig{hostname: {Provider: "gitlab", Enabled: true}}
	}); err != nil {
		t.Fatal(err)
	}
	var s HostService
	before, _ := s.Forecast(repo)
	if !before.Enabled || len(before.Conflicts) != 0 || before.Checked != 0 {
		t.Fatalf("before a refresh: %+v", before)
	}

	now := time.Now()
	changes := []hostapi.Change{
		{Number: 1, Title: "Change shared", Author: hostapi.User{Name: "Ada"}, SourceBranch: "conflicting", TargetBranch: "main", HeadSHA: conflicting, UpdatedAt: now, WebURL: "https://h/1"},
		{Number: 2, Title: "Add other", Author: hostapi.User{Name: "Bob"}, SourceBranch: "clean", TargetBranch: "main", HeadSHA: clean, UpdatedAt: now.Add(-time.Hour)},
		{Number: 3, Title: "Targets release", SourceBranch: "elsewhere", TargetBranch: "release", HeadSHA: elsewhere, UpdatedAt: now},
	}
	hosts := configOrDefault().Hosts
	info := infoFor(hostByName(hostname, hosts), hosts)
	info.Project = "group/app"
	refreshForecast(context.Background(), info, repo, "origin", changes)

	report, err := s.Forecast(repo)
	if err != nil {
		t.Fatal(err)
	}
	if report.Branch != "main" || report.Base != "main" || report.Total != 2 || report.Checked != 2 || len(report.Conflicts) != 1 {
		t.Fatalf("Forecast = %+v", report)
	}
	if c := report.Conflicts[0]; c.Number != 1 || c.Author != "Ada" || len(c.Files) != 1 || c.Files[0] != "shared.txt" || c.Prefix != "!" {
		t.Fatalf("conflict = %+v", c)
	}
	if out := strings.TrimSpace(string(mustOutput(t, repo, "for-each-ref", "--format=%(refname)", "refs/current-client"))); !strings.Contains(out, "mr/1") || !strings.Contains(out, "mr/2") || strings.Contains(out, "mr/3") {
		t.Fatalf("fetched heads = %q", out)
	}
	if out := strings.TrimSpace(string(mustOutput(t, repo, "branch", "-a"))); strings.Contains(out, "mr/") {
		t.Fatalf("fetched heads show as branches:\n%s", out)
	}
}
