package app

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"testing"

	"github.com/gspeager/current-client/internal/watch"
)

// TestGitProcesses counts the git processes that opening a repository starts,
// making the calls the UI makes when a repository opens on Activity, all at
// once as its components do, so repeated questions show up as extra processes.
func TestGitProcesses(t *testing.T) {
	home := t.TempDir()
	t.Setenv("APPDATA", home)
	t.Setenv("XDG_CONFIG_HOME", home)
	t.Setenv("HOME", home)

	remote := filepath.Join(t.TempDir(), "remote.git")
	repo := filepath.Join(t.TempDir(), "repo")
	run := func(dir string, args ...string) {
		t.Helper()
		cmd := exec.Command("git", append([]string{"-c", "user.name=T", "-c", "user.email=t@example.com"}, args...)...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	run(".", "init", "-q", "--bare", "-b", "main", remote)
	run(".", "init", "-q", "-b", "main", repo)
	if err := os.WriteFile(filepath.Join(repo, "main.go"), []byte("package main\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	run(repo, "add", "main.go")
	run(repo, "commit", "-q", "-m", "feat: first")
	run(repo, "remote", "add", "origin", remote)
	run(repo, "push", "-q", "-u", "origin", "main")
	for i := range 15 {
		run(repo, "branch", fmt.Sprintf("topic-%02d", i))
	}
	run(repo, "tag", "v1.0.0")

	trace := filepath.Join(t.TempDir(), "trace.json")
	t.Setenv("GIT_TRACE2_EVENT", trace)

	var watcher watch.Watcher
	defer watcher.Stop()
	open := []func(){
		func() { _, _ = (&GitService{}).GetGitVersion() },
		func() { _ = (&RepositoryService{}).GetRepoSummaries([]string{repo}) },
		func() { _, _ = (&HistoryService{}).FollowsConventionalCommits(repo) },
		func() { _, _ = (&StatusService{}).GetStatus(repo) },
		func() { _, _ = (&ConflictService{}).GetConflictState(repo) },
		func() { _, _ = (&RemoteService{}).LastFetchTime(repo) },
		func() { _ = watcher.Start(repo, func(bool) {}) },
		// HeaderBar
		func() { _ = (&IdentityService{}).GetCurrentUser(repo) },
		func() { _, _ = (&BranchService{}).CurrentBranchStatus(repo) },
		// StatusBar
		func() { _, _ = (&BranchService{}).CurrentBranchStatus(repo) },
		// Nav pane
		func() { _, _ = (&BranchService{}).ListLocal(repo) },
		func() { _, _ = (&BranchService{}).CurrentBranchStatus(repo) },
		func() { _, _ = (&BranchService{}).DefaultBranch(repo) },
		func() { _, _ = (&OverlapService{}).Predict(repo) },
		func() { _, _ = (&StashService{}).ListStash(repo) },
		func() { _, _ = (&TagService{}).ListTags(repo) },
		func() { _, _ = (&RemoteService{}).List(repo) },
		func() { _, _ = (&RemoteService{}).List(repo) },
		func() { _, _ = (&WorktreeService{}).List(repo) },
		func() { _, _ = (&BranchService{}).ListLocal(repo) },
		func() { _, _ = (&SubmoduleService{}).List(repo) },
		// Activity
		func() { _, _ = (&DashboardService{}).GetRepoStats(repo) },
		func() { _, _ = (&DashboardService{}).GetRepoDiskUsage(repo) },
		func() { _, _ = (&DashboardService{}).GetRepoDiskUsage(repo) },
		func() { _, _ = (&BranchService{}).ListLocal(repo) },
		func() { _, _ = (&BranchService{}).CurrentBranchStatus(repo) },
		func() { _, _ = (&ActivityService{}).GetCommitActivity(repo, 371) },
		func() { _, _ = (&ActivityService{}).GetCommitActivity(repo, 84) },
		func() { _, _ = (&DashboardService{}).GetFileChurn(repo) },
		func() { _, _ = (&ContributorsService{}).GetContributors(repo, "", "") },
		func() { _, _ = (&DashboardService{}).GetLanguageBreakdown(repo) },
	}
	var wg sync.WaitGroup
	for _, call := range open {
		wg.Go(call)
	}
	wg.Wait()

	total, byCommand := countGitProcesses(t, trace)
	t.Logf("opening a repository: %d git processes\n%s", total, byCommand)
	const budget = 67
	if total > budget {
		t.Errorf("opening a repository started %d git processes, want at most %d", total, budget)
	}
}

func countGitProcesses(t *testing.T, trace string) (int, string) {
	t.Helper()
	f, err := os.Open(trace)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	counts := map[string]int{}
	total := 0
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 1<<20), 1<<20)
	for scanner.Scan() {
		var event struct {
			Event string   `json:"event"`
			Argv  []string `json:"argv"`
		}
		if json.Unmarshal(scanner.Bytes(), &event) != nil || event.Event != "start" {
			continue
		}
		total++
		args := event.Argv[1:]
		if len(args) > 3 {
			args = args[:3]
		}
		counts[strings.Join(args, " ")]++
	}
	lines := make([]string, 0, len(counts))
	for command, n := range counts {
		lines = append(lines, fmt.Sprintf("%3d  %s", n, command))
	}
	sort.Sort(sort.Reverse(sort.StringSlice(lines)))
	return total, strings.Join(lines, "\n")
}
