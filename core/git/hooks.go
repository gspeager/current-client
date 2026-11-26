package git

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"

	"github.com/gspeager/current-client/core/gitexec"
)

type Hooks struct {
	// The hooks directory in use.
	Dir string
	// Where core.hooksPath set Dir, "" when it isn't set; "global" or "system"
	// applies to every repository.
	Scope string
	// Dir is in the repository, so turning a hook off is a change to commit.
	Tracked bool
	// How many worktrees share Dir.
	Worktrees int
	// Hooks are set in config and this Git doesn't run them.
	ConfigUnsupported bool
	List              []Hook
}

// Hook is a file in the hooks directory, or a hook set in config.
type Hook struct {
	Event string
	// A file's Name is its Event. A config hook has a name of its own, a
	// Command and the Scope it's set in.
	Name    string
	Path    string
	Command string
	Scope   string
	On      bool
	// The tool that writes the hook, where that can be told.
	Manager string
}

var hookEvents = []string{
	"applypatch-msg", "pre-applypatch", "post-applypatch", "pre-commit", "pre-merge-commit",
	"prepare-commit-msg", "commit-msg", "post-commit", "pre-rebase", "post-checkout", "post-merge",
	"pre-push", "pre-receive", "update", "proc-receive", "post-receive", "post-update",
	"reference-transaction", "push-to-checkout", "pre-auto-gc", "post-rewrite", "sendemail-validate",
	"fsmonitor-watchman", "p4-changelist", "p4-prepare-changelist", "p4-post-changelist",
	"p4-pre-submit", "post-index-change",
}

// Windows has no executable bit, so a hook there is turned off by this suffix.
const hookOff = ".off"

var (
	ErrHookManaged = &gitexec.AppError{Message: "This hook is written by a tool. Turn it off there."}
	errHookGone    = &gitexec.AppError{Message: "This hook isn't there any more."}
	errHookTaken   = &gitexec.AppError{Message: "A hook of that name is already on."}
)

// ReadHooks lists the hooks Git would run, and those turned off. It runs none.
func ReadHooks(ctx context.Context, repoPath string) (Hooks, error) {
	// All relative to repoPath, so they compare: --path-format=absolute needs Git 2.31.
	result, err := gitexec.RunIn(ctx, repoPath, "rev-parse", "--git-path", "hooks", "--git-common-dir", "--show-cdup")
	if err != nil {
		return Hooks{}, err
	}
	// --show-cdup is an empty line at the top.
	lines := strings.Split(result.Stdout, "\n")
	if len(lines) < 3 {
		return Hooks{}, &gitexec.AppError{Message: "Hooks can only be listed in a repository with a working tree."}
	}
	hooks := Hooks{Dir: filepath.Clean(absoluteIn(repoPath, lines[0])), Worktrees: 1}
	common, top := absoluteIn(repoPath, lines[1]), filepath.Clean(filepath.Join(repoPath, lines[2]))
	if scoped, err := gitexec.RunIn(ctx, repoPath, "config", "--show-scope", "--get", "core.hooksPath"); err == nil {
		hooks.Scope, _, _ = strings.Cut(scoped.Stdout, "\t")
	}
	// A folder in the checkout is each worktree's own; any other is shared.
	if inside(top, hooks.Dir) && !inside(common, hooks.Dir) {
		rel, _ := filepath.Rel(top, hooks.Dir)
		tracked, err := gitexec.RunIn(ctx, repoPath, "ls-files", "--", filepath.ToSlash(rel))
		hooks.Tracked = err == nil && strings.TrimSpace(tracked.Stdout) != ""
	} else if worktrees, err := gitexec.RunIn(ctx, repoPath, "worktree", "list", "--porcelain"); err == nil {
		hooks.Worktrees = strings.Count("\n"+worktrees.Stdout, "\nworktree ")
	}
	if hooks.List, err = directoryHooks(hooks.Dir); err != nil {
		return Hooks{}, err
	}
	configured, supported, err := configHooks(ctx, repoPath)
	if err != nil {
		return Hooks{}, err
	}
	hooks.ConfigUnsupported = len(configured) > 0 && !supported
	if supported {
		hooks.List = append(hooks.List, configured...)
	}
	slices.SortStableFunc(hooks.List, func(a, b Hook) int {
		return slices.Index(hookEvents, a.Event) - slices.Index(hookEvents, b.Event)
	})
	return hooks, nil
}

func inside(parent, path string) bool {
	rel, err := filepath.Rel(parent, path)
	return parent != "" && err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

func directoryHooks(dir string) ([]Hook, error) {
	entries, err := os.ReadDir(dir)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	husky := strings.Contains(filepath.ToSlash(dir), ".husky")
	var hooks []Hook
	for _, entry := range entries {
		event := strings.TrimSuffix(entry.Name(), hookOff)
		if entry.IsDir() || !slices.Contains(hookEvents, event) {
			continue
		}
		path := filepath.Join(dir, entry.Name())
		info, err := os.Stat(path)
		if err != nil {
			continue
		}
		hook := Hook{Event: event, Name: event, Path: path, On: event == entry.Name() && runnable(path, info)}
		if husky {
			hook.Manager = "Husky"
		} else {
			hook.Manager = managerOf(path)
		}
		hooks = append(hooks, hook)
	}
	return hooks, nil
}

// Git for Windows has no executable bit, and runs a script with #! or a program.
func runnable(path string, info os.FileInfo) bool {
	if runtime.GOOS != "windows" {
		return info.Mode()&0o111 != 0
	}
	start := fileStart(path, 2)
	return start == "#!" || start == "MZ"
}

func fileStart(path string, size int) string {
	file, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer file.Close()
	start := make([]byte, size)
	n, _ := io.ReadFull(file, start)
	return string(start[:n])
}

func managerOf(path string) string {
	text := fileStart(path, 4096)
	switch {
	case strings.Contains(text, "File generated by pre-commit"):
		return "pre-commit"
	case strings.Contains(text, "husky"):
		return "Husky"
	case strings.Contains(text, "lefthook"):
		return "Lefthook"
	}
	return ""
}

// configHooks reads hook.<name>.* at every scope. supported is false for a Git
// that doesn't run them (before 2.54).
func configHooks(ctx context.Context, repoPath string) (hooks []Hook, supported bool, err error) {
	result, err := gitexec.Run(ctx, gitexec.Command{
		Dir:  repoPath,
		Args: []string{"config", "--show-scope", "-z", "--get-regexp", `^hook\.`},
	})
	if err != nil {
		return nil, false, gitexec.WrapRunError(err)
	}
	type configured struct {
		events  []string
		command string
		scope   string
		on      bool
	}
	byName := map[string]*configured{}
	var names []string
	fields := strings.Split(result.Stdout, "\x00")
	for i := 0; i+1 < len(fields); i += 2 {
		scope := fields[i]
		key, value, _ := strings.Cut(fields[i+1], "\n")
		rest := strings.TrimPrefix(key, "hook.")
		dot := strings.LastIndex(rest, ".")
		if dot <= 0 {
			continue
		}
		name, variable := rest[:dot], rest[dot+1:]
		hook := byName[name]
		if hook == nil {
			hook = &configured{on: true}
			byName[name] = hook
			names = append(names, name)
		}
		switch variable {
		case "event":
			hook.events = append(hook.events, value)
		case "command":
			hook.command, hook.scope = value, scope
		case "enabled":
			hook.on = configTrue(value)
		}
	}
	if len(names) == 0 {
		return nil, true, nil
	}
	var events []string
	for _, name := range names {
		events = append(events, byName[name].events...)
	}
	if len(events) == 0 {
		return nil, true, nil
	}
	if supported, err = configHooksSupported(ctx, repoPath, events[0]); err != nil {
		return nil, false, err
	}
	for _, name := range names {
		hook := byName[name]
		for _, event := range hook.events {
			hooks = append(hooks, Hook{Event: event, Name: name, Command: hook.command, Scope: hook.scope, On: hook.on})
		}
	}
	return hooks, supported, nil
}

func configTrue(value string) bool {
	switch strings.ToLower(value) {
	case "false", "no", "off", "0", "":
		return false
	}
	return true
}

// A Git that runs config hooks lists one for its event. One without them has no
// git hook list --show-scope (exit 129), or before 2.36 no git hook (exit 1).
func configHooksSupported(ctx context.Context, repoPath, event string) (bool, error) {
	result, err := gitexec.Run(ctx, gitexec.Command{
		Dir:  repoPath,
		Args: []string{"hook", "list", "--show-scope", event},
	})
	if err != nil {
		return false, gitexec.WrapRunError(err)
	}
	return result.ExitCode == 0, nil
}

// SetHookOn turns a hook off or on again without deleting it: a file's executable
// bit, or on Windows its name, and a config hook's hook.<name>.enabled in the
// repository's own config. The hook is looked up again, so only one ReadHooks
// lists can be changed.
func SetHookOn(ctx context.Context, repoPath string, hook Hook, on bool) error {
	hooks, err := ReadHooks(ctx, repoPath)
	if err != nil {
		return err
	}
	index := slices.IndexFunc(hooks.List, func(h Hook) bool {
		return h.Name == hook.Name && h.Path == hook.Path && h.Event == hook.Event
	})
	if index < 0 {
		return errHookGone
	}
	found := hooks.List[index]
	if found.Manager != "" {
		return ErrHookManaged
	}
	if found.On == on {
		return nil
	}
	if found.Path == "" {
		return setConfigHookOn(ctx, repoPath, found.Name, on)
	}
	return setFileHookOn(found.Path, on)
}

// A symlink is renamed too: its mode is its target's, which can be a tracked script.
func setFileHookOn(path string, on bool) error {
	link, err := os.Lstat(path)
	if err != nil {
		return err
	}
	if runtime.GOOS == "windows" || link.Mode()&os.ModeSymlink != 0 {
		target := strings.TrimSuffix(path, hookOff)
		if !on {
			target = path + hookOff
		}
		if _, err := os.Stat(target); err == nil {
			return errHookTaken
		}
		return os.Rename(path, target)
	}
	if on && strings.HasSuffix(path, hookOff) {
		target := strings.TrimSuffix(path, hookOff)
		if _, err := os.Stat(target); err == nil {
			return errHookTaken
		}
		if err := os.Rename(path, target); err != nil {
			return err
		}
		path = target
	}
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	mode := info.Mode().Perm() &^ 0o111
	if on {
		// Executable for whoever can read it.
		mode |= (info.Mode().Perm() & 0o444) >> 2
	}
	return os.Chmod(path, mode)
}

func setConfigHookOn(ctx context.Context, repoPath, name string, on bool) error {
	key := "hook." + name + ".enabled"
	if !on {
		_, err := gitexec.RunIn(ctx, repoPath, "config", "--local", key, "false")
		return err
	}
	// It exits 5 when there's nothing to unset.
	unset, err := gitexec.Run(ctx, gitexec.Command{Dir: repoPath, Args: []string{"config", "--local", "--unset-all", key}})
	if err != nil {
		return gitexec.WrapRunError(err)
	}
	if unset.ExitCode != 0 && unset.ExitCode != 5 {
		return gitexec.WrapResult(unset)
	}
	if value, err := gitexec.RunIn(ctx, repoPath, "config", "--get", key); err == nil && !configTrue(strings.TrimSpace(value.Stdout)) {
		_, err := gitexec.RunIn(ctx, repoPath, "config", "--local", key, "true")
		return err
	}
	return nil
}
