package git

import (
	"context"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/gspeager/current-client/core/internal/gittest"
)

func TestParseLocalBranches(t *testing.T) {
	output := "*\tfeature\torigin/feature\t[ahead 2]\t2024-01-15T10:30:00-08:00\n" +
		" \tmain\torigin/main\t\t2024-01-10T08:00:00-08:00\n" +
		" \trelease\t\t\t2024-01-05T12:00:00-08:00\n"
	got := parseLocalBranches(output)
	want := []Branch{
		{Name: "feature", Current: true, Upstream: "origin/feature", Ahead: 2, LastCommitDate: "2024-01-15T10:30:00-08:00"},
		{Name: "main", Current: false, Upstream: "origin/main", LastCommitDate: "2024-01-10T08:00:00-08:00"},
		{Name: "release", Current: false, LastCommitDate: "2024-01-05T12:00:00-08:00"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %+v, want %+v", got, want)
	}
}

func TestParseUpstreamTrack(t *testing.T) {
	tests := []struct {
		track      string
		wantAhead  int
		wantBehind int
	}{
		{"", 0, 0},
		{"[gone]", 0, 0},
		{"[ahead 3]", 3, 0},
		{"[behind 5]", 0, 5},
		{"[ahead 2, behind 1]", 2, 1},
	}
	for _, tt := range tests {
		ahead, behind := parseUpstreamTrack(tt.track)
		if ahead != tt.wantAhead || behind != tt.wantBehind {
			t.Errorf("parseUpstreamTrack(%q) = (%d, %d), want (%d, %d)", tt.track, ahead, behind, tt.wantAhead, tt.wantBehind)
		}
	}
}

func TestParseLocalBranchesEmpty(t *testing.T) {
	got := parseLocalBranches("")
	if len(got) != 0 {
		t.Fatalf("got %+v, want none", got)
	}
}

func TestParseRemoteBranches(t *testing.T) {
	output := "refs/remotes/origin/HEAD\torigin\n" +
		"refs/remotes/origin/main\torigin/main\n" +
		"refs/remotes/origin/feature\torigin/feature\n"
	got := parseRemoteBranches(output)
	want := []string{"origin/main", "origin/feature"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %+v, want %+v (symbolic origin/HEAD entry should be filtered out)", got, want)
	}
}

func TestListLocalRealRepo(t *testing.T) {
	dir := gittest.InitRepo(t)
	gittest.CommitFile(t, dir, "file.txt", "v1", "initial")
	gittest.Run(t, dir, "branch", "feature")

	branches, err := ListBranches(context.Background(), dir)
	if err != nil {
		t.Fatalf("ListBranches: %v", err)
	}
	byName := make(map[string]bool)
	for _, b := range branches {
		byName[b.Name] = b.Current
	}
	if len(branches) != 2 {
		t.Fatalf("got %+v, want 2 branches", branches)
	}
	if !byName["main"] {
		t.Fatalf("got %+v, want main marked current", branches)
	}
	if byName["feature"] {
		t.Fatalf("got %+v, want feature not current", branches)
	}
}

func TestListLocalReportsAheadBehindAgainstUpstream(t *testing.T) {
	remoteDir := t.TempDir() + "/remote.git"
	gittest.Run(t, "", "init", "--bare", "-b", "main", remoteDir)

	dir := gittest.InitRepo(t)
	gittest.Run(t, dir, "remote", "add", "origin", remoteDir)
	gittest.CommitFile(t, dir, "file.txt", "v1", "initial")
	gittest.Run(t, dir, "push", "-u", "origin", "main")

	// main is ahead by a local commit and behind a fetched remote commit.
	clone := t.TempDir()
	gittest.Run(t, "", "clone", "-q", remoteDir, clone)
	gittest.Run(t, clone, "config", "user.email", "test@example.com")
	gittest.Run(t, clone, "config", "user.name", "Test")
	gittest.WriteFile(t, clone, "file.txt", "v2")
	gittest.Run(t, clone, "add", "file.txt")
	gittest.Run(t, clone, "commit", "-m", "from clone")
	gittest.Run(t, clone, "push", "-q", "origin", "main")

	gittest.CommitFile(t, dir, "local-only.txt", "unpushed", "local only")
	gittest.Run(t, dir, "fetch", "-q", "origin")

	branches, err := ListBranches(context.Background(), dir)
	if err != nil {
		t.Fatalf("ListBranches: %v", err)
	}
	var main *Branch
	for i := range branches {
		if branches[i].Name == "main" {
			main = &branches[i]
		}
	}
	if main == nil {
		t.Fatalf("got %+v, want a main branch", branches)
	}
	if main.Upstream != "origin/main" {
		t.Fatalf("main.Upstream = %q, want origin/main", main.Upstream)
	}
	if main.Ahead != 1 || main.Behind != 1 {
		t.Fatalf("main ahead/behind = %d/%d, want 1/1", main.Ahead, main.Behind)
	}
	if main.LastCommitDate == "" {
		t.Fatalf("main.LastCommitDate is empty, want the tip commit's date")
	}
}

func TestCreateBranchDoesNotSwitch(t *testing.T) {
	dir := gittest.InitRepo(t)
	gittest.CommitFile(t, dir, "file.txt", "v1", "initial")

	if err := CreateBranch(context.Background(), dir, "feature"); err != nil {
		t.Fatalf("CreateBranch: %v", err)
	}

	branches, err := ListBranches(context.Background(), dir)
	if err != nil {
		t.Fatalf("ListBranches: %v", err)
	}
	byName := make(map[string]bool)
	for _, b := range branches {
		byName[b.Name] = b.Current
	}
	if _, ok := byName["feature"]; !ok {
		t.Fatalf("got %+v, want feature to exist", branches)
	}
	if !byName["main"] {
		t.Fatalf("got %+v, want main still current after CreateBranch", branches)
	}
}

func TestCheckoutBranchSwitchesHeadAndWorkingTree(t *testing.T) {
	dir := gittest.InitRepo(t)
	gittest.CommitFile(t, dir, "file.txt", "on main\n", "initial")
	gittest.Run(t, dir, "checkout", "-q", "-b", "feature")
	gittest.CommitFile(t, dir, "file.txt", "on feature\n", "feature change")
	gittest.Run(t, dir, "checkout", "-q", "main")

	if err := CheckoutBranch(context.Background(), dir, "feature"); err != nil {
		t.Fatalf("CheckoutBranch: %v", err)
	}

	branches, err := ListBranches(context.Background(), dir)
	if err != nil {
		t.Fatalf("ListBranches: %v", err)
	}
	byName := make(map[string]bool)
	for _, b := range branches {
		byName[b.Name] = b.Current
	}
	if !byName["feature"] {
		t.Fatalf("got %+v, want feature current after CheckoutBranch", branches)
	}

	content, err := os.ReadFile(dir + "/file.txt")
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(content) != "on feature\n" {
		t.Fatalf("working tree content = %q, want %q (checkout should update the working tree)", content, "on feature\n")
	}
}

func TestCreateBranchAtPointsToGivenCommit(t *testing.T) {
	dir := gittest.InitRepo(t)
	gittest.CommitFile(t, dir, "file.txt", "v1", "first")
	firstSHA := gittest.Run(t, dir, "rev-parse", "HEAD")
	gittest.CommitFile(t, dir, "file.txt", "v2", "second")

	if err := CreateBranchAt(context.Background(), dir, "from-first", firstSHA); err != nil {
		t.Fatalf("CreateBranchAt: %v", err)
	}

	got := gittest.Run(t, dir, "log", "-1", "--format=%H", "from-first")
	if got != firstSHA {
		t.Fatalf("from-first points to %q, want %q", got, firstSHA)
	}

	branches, err := ListBranches(context.Background(), dir)
	if err != nil {
		t.Fatalf("ListBranches: %v", err)
	}
	byName := make(map[string]bool)
	for _, b := range branches {
		byName[b.Name] = b.Current
	}
	if byName["from-first"] {
		t.Fatalf("got %+v, want from-first not current", branches)
	}
	if !byName["main"] {
		t.Fatalf("got %+v, want main still current after CreateBranchAt", branches)
	}
}

func TestRenameBranch(t *testing.T) {
	dir := gittest.InitRepo(t)
	gittest.CommitFile(t, dir, "file.txt", "v1", "initial")
	gittest.Run(t, dir, "branch", "feature")

	if err := RenameBranch(context.Background(), dir, "feature", "renamed"); err != nil {
		t.Fatalf("RenameBranch: %v", err)
	}

	branches, err := ListBranches(context.Background(), dir)
	if err != nil {
		t.Fatalf("ListBranches: %v", err)
	}
	byName := make(map[string]bool)
	for _, b := range branches {
		byName[b.Name] = true
	}
	if byName["feature"] {
		t.Fatalf("got %+v, want feature gone after rename", branches)
	}
	if !byName["renamed"] {
		t.Fatalf("got %+v, want renamed to exist", branches)
	}
}

func TestDeleteBranchRemovesMergedBranch(t *testing.T) {
	dir := gittest.InitRepo(t)
	gittest.CommitFile(t, dir, "file.txt", "v1", "initial")
	gittest.Run(t, dir, "branch", "feature")

	if err := DeleteBranch(context.Background(), dir, "feature", false); err != nil {
		t.Fatalf("DeleteBranch: %v", err)
	}

	branches, err := ListBranches(context.Background(), dir)
	if err != nil {
		t.Fatalf("ListBranches: %v", err)
	}
	for _, b := range branches {
		if b.Name == "feature" {
			t.Fatalf("got %+v, want feature deleted", branches)
		}
	}
}

func TestDeleteBranchSafeFailsForUnmergedBranch(t *testing.T) {
	dir := gittest.InitRepo(t)
	gittest.CommitFile(t, dir, "file.txt", "v1", "initial")
	gittest.Run(t, dir, "checkout", "-q", "-b", "feature")
	gittest.CommitFile(t, dir, "file.txt", "v2", "unmerged change")
	gittest.Run(t, dir, "checkout", "-q", "main")

	if err := DeleteBranch(context.Background(), dir, "feature", false); err == nil {
		t.Fatal("expected error deleting an unmerged branch without force")
	}

	if err := DeleteBranch(context.Background(), dir, "feature", true); err != nil {
		t.Fatalf("force DeleteBranch: %v", err)
	}
	branches, err := ListBranches(context.Background(), dir)
	if err != nil {
		t.Fatalf("ListBranches: %v", err)
	}
	for _, b := range branches {
		if b.Name == "feature" {
			t.Fatalf("got %+v, want feature deleted after force", branches)
		}
	}
}

func TestListRemoteRealRepo(t *testing.T) {
	remoteDir := t.TempDir() + "/remote.git"
	gittest.Run(t, "", "init", "--bare", "-b", "main", remoteDir)

	dir := gittest.InitRepo(t)
	gittest.Run(t, dir, "remote", "add", "origin", remoteDir)
	gittest.CommitFile(t, dir, "file.txt", "v1", "initial")
	gittest.Run(t, dir, "push", "-u", "origin", "main")

	branches, err := ListRemote(context.Background(), dir)
	if err != nil {
		t.Fatalf("ListRemote: %v", err)
	}
	if len(branches) != 1 || branches[0] != "origin/main" {
		t.Fatalf("got %+v, want [\"origin/main\"]", branches)
	}
}

func TestCurrentBranchStatusNoUpstream(t *testing.T) {
	dir := gittest.InitRepo(t)
	gittest.CommitFile(t, dir, "file.txt", "v1", "initial")

	status, err := CurrentBranchStatus(context.Background(), dir)
	if err != nil {
		t.Fatalf("CurrentBranchStatus: %v", err)
	}
	want := BranchStatus{Current: "main"}
	if status != want {
		t.Fatalf("got %+v, want %+v", status, want)
	}
}

func TestCurrentBranchStatusDivergedUpstream(t *testing.T) {
	remoteDir := t.TempDir() + "/remote.git"
	gittest.Run(t, "", "init", "--bare", "-b", "main", remoteDir)

	dir := gittest.InitRepo(t)
	gittest.Run(t, dir, "remote", "add", "origin", remoteDir)
	gittest.CommitFile(t, dir, "file.txt", "v1", "initial")
	gittest.Run(t, dir, "push", "-u", "origin", "main")

	clone := t.TempDir()
	gittest.Run(t, "", "clone", "-q", remoteDir, clone)
	gittest.Run(t, clone, "config", "user.email", "test@example.com")
	gittest.Run(t, clone, "config", "user.name", "Test")
	gittest.WriteFile(t, clone, "file.txt", "from clone")
	gittest.Run(t, clone, "add", "file.txt")
	gittest.Run(t, clone, "commit", "-m", "remote change")
	gittest.Run(t, clone, "push", "-q", "origin", "main")

	gittest.CommitFile(t, dir, "other.txt", "v1", "local change 1")
	gittest.CommitFile(t, dir, "other.txt", "v2", "local change 2")
	gittest.Run(t, dir, "fetch", "-q", "origin")

	status, err := CurrentBranchStatus(context.Background(), dir)
	if err != nil {
		t.Fatalf("CurrentBranchStatus: %v", err)
	}
	want := BranchStatus{Current: "main", Upstream: "origin/main", Ahead: 2, Behind: 1}
	if status != want {
		t.Fatalf("got %+v, want %+v", status, want)
	}
}

func TestDefaultBranchPrefersRemoteHead(t *testing.T) {
	branches := []Branch{{Name: "main"}, {Name: "trunk"}}
	if got := defaultBranch(branches, "trunk"); got != "trunk" {
		t.Fatalf("defaultBranch = %q, want trunk", got)
	}
}

func TestDefaultBranchFallsBackToMain(t *testing.T) {
	branches := []Branch{{Name: "feature"}, {Name: "main"}}
	if got := defaultBranch(branches, ""); got != "main" {
		t.Fatalf("defaultBranch = %q, want main", got)
	}
}

func TestDefaultBranchFallsBackToMaster(t *testing.T) {
	branches := []Branch{{Name: "feature"}, {Name: "master"}}
	if got := defaultBranch(branches, "trunk"); got != "master" {
		t.Fatalf("defaultBranch = %q, want master (trunk has no local branch)", got)
	}
}

func TestDefaultBranchNoneFound(t *testing.T) {
	branches := []Branch{{Name: "feature"}}
	if got := defaultBranch(branches, ""); got != "" {
		t.Fatalf("defaultBranch = %q, want empty", got)
	}
}

func TestDefaultBranchReadsOriginHead(t *testing.T) {
	remoteDir := t.TempDir() + "/remote.git"
	gittest.Run(t, "", "init", "--bare", "-b", "trunk", remoteDir)
	seed := gittest.InitRepo(t)
	gittest.Run(t, seed, "checkout", "-q", "-b", "trunk")
	gittest.CommitFile(t, seed, "file.txt", "v1", "initial")
	gittest.Run(t, seed, "push", "-q", remoteDir, "trunk")

	clone := t.TempDir()
	gittest.Run(t, "", "clone", "-q", remoteDir, clone)
	gittest.Run(t, clone, "branch", "main")

	got, err := DefaultBranch(context.Background(), clone)
	if err != nil {
		t.Fatalf("DefaultBranch: %v", err)
	}
	if got != "trunk" {
		t.Fatalf("DefaultBranch = %q, want trunk from origin/HEAD over a local main", got)
	}
}

func TestCheckoutCommitDetachesHeadAndStatusSaysWhere(t *testing.T) {
	ctx := context.Background()
	dir := gittest.InitRepo(t)
	gittest.CommitFile(t, dir, "file.txt", "v1", "first")
	first := gittest.Run(t, dir, "rev-parse", "HEAD")
	gittest.CommitFile(t, dir, "file.txt", "v2", "second")

	if err := CheckoutCommit(ctx, dir, first); err != nil {
		t.Fatalf("CheckoutCommit: %v", err)
	}

	if head := gittest.Run(t, dir, "rev-parse", "HEAD"); head != first {
		t.Fatalf("HEAD = %q, want %q", head, first)
	}
	status, err := CurrentBranchStatus(ctx, dir)
	if err != nil {
		t.Fatalf("CurrentBranchStatus: %v", err)
	}
	if status.Current != "HEAD" || status.DetachedAt == "" || !strings.HasPrefix(first, status.DetachedAt) {
		t.Fatalf("status = %+v, want detached at a short form of %q", status, first)
	}

	if err := CheckoutBranch(ctx, dir, "main"); err != nil {
		t.Fatalf("CheckoutBranch: %v", err)
	}
	if status, _ := CurrentBranchStatus(ctx, dir); status.DetachedAt != "" || status.Current != "main" {
		t.Fatalf("back on main, status = %+v", status)
	}
}
