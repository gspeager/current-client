package git

import (
	"context"
	"reflect"
	"testing"

	"github.com/gspeager/current-client/core/internal/gittest"
)

func TestParseSubmoduleStatus(t *testing.T) {
	output := " aaa lib/core (v1.0)\n-bbb vendor/with space\n+ccc tools (heads/main)\nUddd conflicted\n"
	want := []Submodule{
		{Path: "lib/core", Commit: "aaa", Initialized: true},
		{Path: "vendor/with space", Commit: "bbb"},
		{Path: "tools", Commit: "ccc", Initialized: true, CommitChanged: true},
		{Path: "conflicted", Commit: "ddd", Initialized: true, Conflicted: true},
	}
	if got := parseSubmoduleStatus(output); !reflect.DeepEqual(got, want) {
		t.Fatalf("got %+v, want %+v", got, want)
	}
}

// Local submodules need protocol.file.allow since Git 2.38.1.
func gitAllowFile(t *testing.T, dir string, args ...string) string {
	t.Helper()
	return gittest.Run(t, dir, append([]string{"-c", "protocol.file.allow=always"}, args...)...)
}

func initSuperproject(t *testing.T) (super, lib, first, second string) {
	lib = gittest.InitRepo(t)
	gittest.CommitFile(t, lib, "lib.txt", "v1", "lib: first")
	first = gittest.Run(t, lib, "rev-parse", "HEAD")
	gittest.CommitFile(t, lib, "lib.txt", "v2", "lib: second")
	second = gittest.Run(t, lib, "rev-parse", "HEAD")

	super = gittest.InitRepo(t)
	gittest.CommitFile(t, super, "app.txt", "app", "initial")
	gitAllowFile(t, super, "submodule", "add", "-q", lib, "vendor/lib")
	gittest.Run(t, super+"/vendor/lib", "checkout", "-q", first)
	gittest.Run(t, super, "add", "vendor/lib")
	gittest.Run(t, super, "commit", "-q", "-m", "add lib at first")
	return super, lib, first, second
}

func TestSubmoduleStatusListAndCommits(t *testing.T) {
	ctx := context.Background()
	super, _, first, second := initSuperproject(t)

	gittest.Run(t, super+"/vendor/lib", "checkout", "-q", second)
	gittest.WriteFile(t, super+"/vendor/lib", "lib.txt", "local edit")

	statuses, err := GetStatus(ctx, super)
	if err != nil {
		t.Fatalf("GetStatus: %v", err)
	}
	if len(statuses) != 1 || statuses[0].Path != "vendor/lib" ||
		!reflect.DeepEqual(statuses[0].Submodule, &SubmoduleStatus{CommitChanged: true, Modified: true}) {
		t.Fatalf("status = %+v (submodule %+v), want vendor/lib with a changed commit and modifications", statuses, statuses[0].Submodule)
	}

	submodules, err := ListSubmodules(ctx, super)
	if err != nil {
		t.Fatalf("ListSubmodules: %v", err)
	}
	if len(submodules) != 1 || submodules[0].Path != "vendor/lib" || !submodules[0].CommitChanged || submodules[0].Commit != second {
		t.Fatalf("submodules = %+v", submodules)
	}

	commits, err := SubmoduleCommits(ctx, super, "vendor/lib", first, second)
	if err != nil {
		t.Fatalf("SubmoduleCommits: %v", err)
	}
	if len(commits) != 1 || commits[0].Subject != "lib: second" || !commits[0].Added {
		t.Fatalf("commits = %+v, want lib: second added", commits)
	}
	back, _ := SubmoduleCommits(ctx, super, "vendor/lib", second, first)
	if len(back) != 1 || back[0].Added {
		t.Fatalf("moving back = %+v, want lib: second dropped", back)
	}
}

func TestUpdateSubmodulesInitialisesAClone(t *testing.T) {
	ctx := context.Background()
	super, _, first, _ := initSuperproject(t)
	clone := t.TempDir()
	gittest.Run(t, "", "clone", "-q", super, clone)

	submodules, _ := ListSubmodules(ctx, clone)
	if len(submodules) != 1 || submodules[0].Initialized {
		t.Fatalf("before updating = %+v, want one uninitialised submodule", submodules)
	}

	t.Setenv("GIT_CONFIG_COUNT", "1")
	t.Setenv("GIT_CONFIG_KEY_0", "protocol.file.allow")
	t.Setenv("GIT_CONFIG_VALUE_0", "always")
	if err := UpdateSubmodules(ctx, clone, nil); err != nil {
		t.Fatalf("UpdateSubmodules: %v", err)
	}
	if got := gittest.Run(t, clone+"/vendor/lib", "rev-parse", "HEAD"); got != first {
		t.Fatalf("submodule HEAD = %q, want the recorded %q", got, first)
	}
	if submodules, _ := ListSubmodules(ctx, clone); !submodules[0].Initialized || submodules[0].CommitChanged {
		t.Fatalf("after updating = %+v", submodules)
	}
}
