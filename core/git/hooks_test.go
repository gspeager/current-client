//go:build !windows

package git

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gspeager/current-client/core/internal/gittest"
)

func writeHook(t *testing.T, dir, name, script string) string {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	return path
}

func readHooks(t *testing.T, repo string) Hooks {
	t.Helper()
	hooks, err := ReadHooks(context.Background(), repo)
	if err != nil {
		t.Fatal(err)
	}
	return hooks
}

func hookFor(t *testing.T, hooks Hooks, event string) Hook {
	t.Helper()
	for _, hook := range hooks.List {
		if hook.Event == event {
			return hook
		}
	}
	t.Fatalf("no %s hook in %+v", event, hooks.List)
	return Hook{}
}

// Commits, and says whether the pre-commit hook ran.
func hookRan(t *testing.T, repo string) bool {
	t.Helper()
	ran := filepath.Join(repo, "..", filepath.Base(repo)+"-ran")
	_ = os.Remove(ran)
	gittest.Run(t, repo, "commit", "--allow-empty", "-q", "-m", "c")
	_, err := os.Stat(ran)
	return err == nil
}

func ranScript(repo string) string {
	return "#!/bin/sh\ntouch '" + filepath.Join(filepath.Dir(repo), filepath.Base(repo)+"-ran") + "'\n"
}

func TestTurningAHookOffAndOn(t *testing.T) {
	repo, _ := isolatedConfigRepo(t, "")
	gittest.CommitFile(t, repo, "a.txt", "a", "first")
	dir := filepath.Join(repo, ".git", "hooks")
	writeHook(t, dir, "pre-commit", ranScript(repo))
	writeHook(t, dir, "commit-msg.sample", "#!/bin/sh\n")

	hooks := readHooks(t, repo)
	if hooks.Dir != filepath.Join(gittest.Run(t, repo, "rev-parse", "--path-format=absolute", "--git-common-dir"), "hooks") ||
		hooks.Scope != "" || hooks.Tracked || hooks.Worktrees != 1 {
		t.Fatalf("the default directory: %+v", hooks)
	}
	if len(hooks.List) != 1 {
		t.Fatalf("a .sample file is listed: %+v", hooks.List)
	}
	hook := hookFor(t, hooks, "pre-commit")
	if !hook.On || hook.Manager != "" || !hookRan(t, repo) {
		t.Fatalf("an executable hook: %+v", hook)
	}

	if err := SetHookOn(context.Background(), repo, hook, false); err != nil {
		t.Fatal(err)
	}
	if hookFor(t, readHooks(t, repo), "pre-commit").On || hookRan(t, repo) {
		t.Fatal("the hook still runs")
	}
	if content, _ := os.ReadFile(hook.Path); string(content) != ranScript(repo) {
		t.Fatal("turning it off changed the file")
	}
	if err := SetHookOn(context.Background(), repo, hook, true); err != nil {
		t.Fatal(err)
	}
	if !hookFor(t, readHooks(t, repo), "pre-commit").On || !hookRan(t, repo) {
		t.Fatal("the hook doesn't run again")
	}
}

func TestAHookTurnedOffOnWindows(t *testing.T) {
	repo, _ := isolatedConfigRepo(t, "")
	gittest.CommitFile(t, repo, "a.txt", "a", "first")
	writeHook(t, filepath.Join(repo, ".git", "hooks"), "pre-push"+hookOff, "#!/bin/sh\n")
	hook := hookFor(t, readHooks(t, repo), "pre-push")
	if hook.On {
		t.Fatal("a hook with .off is taken as on")
	}
	if err := SetHookOn(context.Background(), repo, hook, true); err != nil {
		t.Fatal(err)
	}
	if on := hookFor(t, readHooks(t, repo), "pre-push"); !on.On || strings.HasSuffix(on.Path, hookOff) {
		t.Fatalf("turned on: %+v", on)
	}
}

func TestHooksFromAGlobalHooksPath(t *testing.T) {
	shared, _ := filepath.EvalSymlinks(t.TempDir())
	repo, _ := isolatedConfigRepo(t, "[core]\n\thooksPath = "+shared+"\n")
	writeHook(t, shared, "pre-push", "#!/bin/sh\n")
	hooks := readHooks(t, repo)
	if hooks.Dir != shared || hooks.Scope != "global" || len(hooks.List) != 1 {
		t.Fatalf("a global hooks path: %+v", hooks)
	}
}

func TestATrackedHooksDirectory(t *testing.T) {
	repo, _ := isolatedConfigRepo(t, "")
	writeHook(t, filepath.Join(repo, ".githooks"), "pre-commit", ranScript(repo))
	gittest.Run(t, repo, "add", ".")
	gittest.Run(t, repo, "commit", "-q", "-m", "hooks")
	gittest.Run(t, repo, "config", "core.hooksPath", ".githooks")

	hooks := readHooks(t, repo)
	if !hooks.Tracked || hooks.Scope != "local" {
		t.Fatalf("a tracked directory: %+v", hooks)
	}
	if err := SetHookOn(context.Background(), repo, hookFor(t, hooks, "pre-commit"), false); err != nil {
		t.Fatal(err)
	}
	if status := gittest.Run(t, repo, "status", "--porcelain"); !strings.Contains(status, ".githooks/pre-commit") {
		t.Fatalf("turning it off isn't a change to commit: %q", status)
	}
}

func TestHooksAToolManages(t *testing.T) {
	repo, _ := isolatedConfigRepo(t, "")
	writeHook(t, filepath.Join(repo, ".husky", "_"), "pre-commit", "#!/usr/bin/env sh\n. \"$(dirname \"$0\")/h\"\n")
	writeHook(t, filepath.Join(repo, ".husky", "_"), "h", "#!/bin/sh\n")
	gittest.Run(t, repo, "config", "core.hooksPath", ".husky/_")
	husky := readHooks(t, repo)
	hook := hookFor(t, husky, "pre-commit")
	if hook.Manager != "Husky" || len(husky.List) != 1 {
		t.Fatalf("Husky: %+v", husky.List)
	}
	if err := SetHookOn(context.Background(), repo, hook, false); err != ErrHookManaged {
		t.Fatalf("turned off a hook Husky writes: %v", err)
	}

	other, _ := isolatedConfigRepo(t, "")
	writeHook(t, filepath.Join(other, ".git", "hooks"), "pre-commit", "#!/usr/bin/env bash\n# File generated by pre-commit: https://pre-commit.com\n")
	if manager := hookFor(t, readHooks(t, other), "pre-commit").Manager; manager != "pre-commit" {
		t.Fatalf("pre-commit: %q", manager)
	}
}

func TestHooksALinkedWorktreeShares(t *testing.T) {
	repo, _ := isolatedConfigRepo(t, "")
	gittest.CommitFile(t, repo, "a.txt", "a", "first")
	writeHook(t, filepath.Join(repo, ".git", "hooks"), "post-checkout", "#!/bin/sh\n")
	worktree := filepath.Join(t.TempDir(), "wt")
	gittest.Run(t, repo, "worktree", "add", "-q", worktree)
	hooks := readHooks(t, worktree)
	if hooks.Worktrees != 2 || len(hooks.List) != 1 || filepath.Dir(hooks.List[0].Path) != hooks.Dir {
		t.Fatalf("from a linked worktree: %+v", hooks)
	}
	if err := SetHookOn(context.Background(), worktree, hooks.List[0], false); err != nil {
		t.Fatal(err)
	}
	if hookFor(t, readHooks(t, repo), "post-checkout").On {
		t.Fatal("turning it off in one worktree didn't for the other")
	}
}

func TestAHookSetInConfig(t *testing.T) {
	repo, _ := isolatedConfigRepo(t, "[hook \"lint\"]\n\tcommand = touch '"+filepath.Join(t.TempDir(), "x")+"'\n\tevent = pre-commit\n\tevent = pre-push\n")
	gittest.CommitFile(t, repo, "a.txt", "a", "first")
	supported, err := configHooksSupported(context.Background(), repo)
	if err != nil {
		t.Fatal(err)
	}
	hooks := readHooks(t, repo)
	if !supported {
		if !hooks.ConfigUnsupported || len(hooks.List) != 0 {
			t.Fatalf("a Git without config hooks: %+v", hooks)
		}
		t.Skip("this Git doesn't run hooks from config")
	}
	hook := hookFor(t, hooks, "pre-push")
	if len(hooks.List) != 2 || hook.Name != "lint" || hook.Scope != "global" || !hook.On || hook.Path != "" {
		t.Fatalf("a config hook: %+v", hooks.List)
	}
	if err := SetHookOn(context.Background(), repo, hook, false); err != nil {
		t.Fatal(err)
	}
	if listed := gittest.Run(t, repo, "hook", "list", "--show-scope", "pre-commit"); !strings.Contains(listed, "disabled") {
		t.Fatalf("Git still runs it: %q", listed)
	}
	if off := readHooks(t, repo); off.List[0].On || off.List[1].On {
		t.Fatalf("turned off: %+v", off.List)
	}
	if err := SetHookOn(context.Background(), repo, hook, true); err != nil {
		t.Fatal(err)
	}
	if on := readHooks(t, repo); !on.List[0].On {
		t.Fatalf("turned on: %+v", on.List)
	}
	if local, _ := os.ReadFile(filepath.Join(repo, ".git", "config")); strings.Contains(string(local), "hook") {
		t.Fatalf("turning it on again left %q", local)
	}
}
