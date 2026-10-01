package app

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestLargeRepoTiming times the service calls the UI makes against a generated
// 50,000-commit, 20,000-file repository with a large working-tree change.
// Opt-in: LARGE_REPO_TIMING=1 go test -run TestLargeRepoTiming -v ./app
func TestLargeRepoTiming(t *testing.T) {
	if os.Getenv("LARGE_REPO_TIMING") == "" {
		t.Skip("set LARGE_REPO_TIMING=1")
	}
	const files, commits, modified, bigLines = 20000, 50000, 5000, 20000
	const tagEvery = 500
	// Topic branches off older commits; every other one edits a file main changes later, so it conflicts.
	const topics = 50
	subjectPrefixes := []string{"feat: ", "fix(diff): ", "chore: ", "", "docs: ", "feat(history)!: ", "refactor: "}
	// Services such as OpenRepository save to the settings file, so point the
	// user config folder somewhere disposable instead of the real one.
	configHome := t.TempDir()
	t.Setenv("APPDATA", configHome)
	t.Setenv("XDG_CONFIG_HOME", configHome)
	t.Setenv("HOME", configHome)
	dir := t.TempDir()
	run := func(stdin *os.File, args ...string) {
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		if stdin != nil {
			cmd.Stdin = stdin
		}
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	run(nil, "init", "-q", "-b", "main")

	start := time.Now()
	script := filepath.Join(t.TempDir(), "large-repo.fi")
	f, _ := os.Create(script)
	w := bufio.NewWriterSize(f, 1<<20)
	exts := []string{"go", "ts", "tsx", "md", "scss", "json"}
	path := func(i int) string { return fmt.Sprintf("dir%03d/file%05d.%s", i%100, i, exts[i%len(exts)]) }
	data := func(s string) { fmt.Fprintf(w, "data %d\n%s\n", len(s), s) }
	begin := time.Now().Add(-730 * 24 * time.Hour).Unix()
	for c := 0; c < commits; c++ {
		author := fmt.Sprintf("Author %02d <a%02d@example.com>", c%30, c%30)
		fmt.Fprintf(w, "commit refs/heads/main\nmark :%d\ncommitter %s %d +0000\n", c+1, author, begin+int64(c)*730*86400/commits)
		// Mostly Conventional Commits, some plain, as in a project that adopted the format.
		data(fmt.Sprintf("%scommit %d", subjectPrefixes[c%len(subjectPrefixes)], c))
		if c > 0 {
			fmt.Fprintf(w, "from :%d\n", c)
		}
		if c == 0 {
			for i := 0; i < files; i++ {
				fmt.Fprintf(w, "M 100644 inline %s\n", path(i))
				data(fmt.Sprintf("line one of %d\nline two\n", i))
			}
			var big strings.Builder
			for l := 0; l < bigLines; l++ {
				fmt.Fprintf(&big, "original line %d\n", l)
			}
			fmt.Fprintf(w, "M 100644 inline big.txt\n")
			data(big.String())
		} else {
			fmt.Fprintf(w, "M 100644 inline %s\n", path(c%files))
			data(fmt.Sprintf("line one of %d\nchanged in %d\n", c%files, c))
		}
		if (c+1)%tagEvery == 0 {
			fmt.Fprintf(w, "reset refs/tags/v0.%d.0\nfrom :%d\n\n", (c+1)/tagEvery, c+1)
		}
	}
	for b := 0; b < topics; b++ {
		base := commits - 100 - b*97
		fmt.Fprintf(w, "commit refs/heads/topic-%02d\nmark :%d\ncommitter Topic <topic@example.com> %d +0000\n", b, commits+1+b, begin+730*86400)
		data(fmt.Sprintf("feat: topic %d", b))
		fmt.Fprintf(w, "from :%d\n", base)
		file := fmt.Sprintf("topic-%02d.txt", b)
		if b%2 == 0 {
			file = path((base + 5) % files)
		}
		fmt.Fprintf(w, "M 100644 inline %s\n", file)
		data(fmt.Sprintf("topic %d\n", b))
	}
	w.Flush()
	f.Close()
	in, _ := os.Open(script)
	run(in, "fast-import", "--quiet")
	in.Close()
	run(nil, "checkout", "-q", "-f", "main")
	for i := 0; i < modified; i++ {
		os.WriteFile(filepath.Join(dir, path(i)), []byte("modified in working tree\n"), 0o644)
	}
	var big strings.Builder
	for l := 0; l < bigLines; l++ {
		fmt.Fprintf(&big, "rewritten line %d\n", l)
	}
	os.WriteFile(filepath.Join(dir, "big.txt"), []byte(big.String()), 0o644)
	t.Logf("setup: %s (%d commits, %d files, %d modified, big.txt %d lines rewritten)", time.Since(start).Round(time.Second), commits, files+1, modified, bigLines)

	timeIt := func(name string, fn func() (int, error)) {
		s := time.Now()
		n, err := fn()
		t.Logf("%-40s %8s  items=%-6d err=%v", name, time.Since(s).Round(time.Millisecond), n, err)
	}
	history := &HistoryService{}
	var firstPage []CommitInfo
	timeIt("OpenRepository", func() (int, error) { _, err := (&RepositoryService{}).OpenRepository(dir); return 1, err })
	timeIt("GetStatus (5,001 modified)", func() (int, error) { r, err := (&StatusService{}).GetStatus(dir); return len(r), err })
	timeIt("GetHistory first page (200)", func() (int, error) {
		r, err := history.GetHistory(dir, 200, 0, HistoryFilterInfo{})
		firstPage = r
		return len(r), err
	})
	timeIt("GetHistory page at 40,000", func() (int, error) {
		r, err := history.GetHistory(dir, 200, 40000, HistoryFilterInfo{})
		return len(r), err
	})
	timeIt("ComputeGraphLayout (200)", func() (int, error) {
		refs := make([]CommitRef, len(firstPage))
		for i, c := range firstPage {
			refs[i] = CommitRef{SHA: c.SHA, ParentSHAs: c.ParentSHAs}
		}
		return len(history.ComputeGraphLayout(refs)), nil
	})
	timeIt("GetRefs", func() (int, error) { r, err := history.GetRefs(dir); return len(r), err })
	timeIt("CurrentBranchStatus", func() (int, error) { _, err := (&BranchService{}).CurrentBranchStatus(dir); return 1, err })
	timeIt("GetWorkingTreeDiff big.txt (40k lines)", func() (int, error) {
		d, err := (&DiffService{}).GetWorkingTreeDiff(dir, "big.txt", false, false)
		n := 0
		for _, h := range d.Hunks {
			n += len(h.Lines)
		}
		return n, err
	})
	timeIt("ListFiles (quick switch)", func() (int, error) { r, err := (&StatusService{}).ListFiles(dir); return len(r), err })
	timeIt("SearchHistory", func() (int, error) { r, err := history.SearchHistory(dir, "commit 4999", 50); return len(r), err })
	timeIt("GetFileHistory", func() (int, error) { r, err := history.GetFileHistory(dir, path(7), 200, 0); return len(r), err })
	timeIt("GetRepoStats", func() (int, error) { _, err := (&DashboardService{}).GetRepoStats(dir); return 1, err })
	timeIt("GetFileChurn", func() (int, error) { r, err := (&DashboardService{}).GetFileChurn(dir); return len(r), err })
	timeIt("GetLanguageBreakdown", func() (int, error) { r, err := (&DashboardService{}).GetLanguageBreakdown(dir); return len(r), err })
	timeIt("GetRepoDiskUsage", func() (int, error) { _, err := (&DashboardService{}).GetRepoDiskUsage(dir); return 1, err })
	timeIt("GetCommitActivity (365 days)", func() (int, error) { r, err := (&ActivityService{}).GetCommitActivity(dir, 365); return len(r), err })
	timeIt("GetContributors (1 year)", func() (int, error) {
		r, err := (&ContributorsService{}).GetContributors(dir, time.Now().AddDate(-1, 0, 0).Format("2006-01-02"), "")
		return len(r), err
	})
	timeIt("GetRepoSummaries (tab dots)", func() (int, error) { return len((&RepositoryService{}).GetRepoSummaries([]string{dir})), nil })
	timeIt("ListRecentAuthors", func() (int, error) { r, err := (&CommitService{}).ListRecentAuthors(dir); return len(r), err })
	timeIt("OverlapService.Predict (50 topics, cold)", func() (int, error) { r, err := (&OverlapService{}).Predict(dir); return len(r.Overlaps), err })
	timeIt("OverlapService.Predict (cached)", func() (int, error) { r, err := (&OverlapService{}).Predict(dir); return len(r.Overlaps), err })

	timeIt("FollowsConventionalCommits", func() (int, error) { _, err := history.FollowsConventionalCommits(dir); return 1, err })
	timeIt("GetHistory type fix, first page", func() (int, error) {
		r, err := history.GetHistory(dir, 200, 0, HistoryFilterInfo{Type: "fix"})
		return len(r), err
	})
	changelog := &ChangelogService{}
	entries := func(sections []ChangelogReleaseSection) int {
		n := 0
		for _, s := range sections {
			n += len(s.Entries)
		}
		return n
	}
	var all []ChangelogReleaseSection
	timeIt("Changelog LatestTag", func() (int, error) { _, err := changelog.LatestTag(dir, "HEAD"); return 1, err })
	timeIt("Changelog Build last tag..HEAD", func() (int, error) {
		r, err := changelog.Build(dir, "v0.99.0", "HEAD", "", false)
		return entries(r), err
	})
	timeIt("Changelog Build first commit..HEAD", func() (int, error) {
		r, err := changelog.Build(dir, "", "HEAD", "", false)
		all = r
		return entries(r), err
	})
	timeIt("Changelog Build split (100 releases)", func() (int, error) {
		r, err := changelog.Build(dir, "", "HEAD", "", true)
		return len(r), err
	})
	var markdown string
	timeIt("Changelog Markdown (all commits)", func() (int, error) {
		sections := make([]ChangelogMarkdownSection, len(all))
		for i, s := range all {
			sections[i] = ChangelogMarkdownSection{Version: "1.0.0", Date: "2026-01-01", Entries: s.Entries}
		}
		markdown = changelog.Markdown(sections, ChangelogMarkdownOptions{Authors: true, Dates: true})
		return len(markdown), nil
	})
	timeIt("Changelog RenderHTML (all commits)", func() (int, error) { r, err := changelog.RenderHTML(markdown); return len(r), err })
}
