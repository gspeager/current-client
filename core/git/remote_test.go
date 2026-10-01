package git

import (
	"context"
	"reflect"
	"testing"

	"github.com/gspeager/current-client/core/internal/gittest"
)

func TestParseRemotes(t *testing.T) {
	output := "origin\thttps://example.com/repo.git (fetch)\n" +
		"origin\thttps://example.com/repo.git (push)\n" +
		"upstream\thttps://example.com/upstream.git (fetch)\n" +
		"upstream\thttps://example.com/upstream-push.git (push)\n"

	got := parseRemotes(output)
	want := []Remote{
		{Name: "origin", FetchURL: "https://example.com/repo.git", PushURL: "https://example.com/repo.git"},
		{Name: "upstream", FetchURL: "https://example.com/upstream.git", PushURL: "https://example.com/upstream-push.git"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %+v, want %+v", got, want)
	}
}

func TestParseRemotesEmpty(t *testing.T) {
	got := parseRemotes("")
	if len(got) != 0 {
		t.Fatalf("got %+v, want none", got)
	}
}

func TestListRemotesRealRepo(t *testing.T) {
	remoteDir := t.TempDir() + "/remote.git"
	gittest.Run(t, "", "init", "--bare", "-b", "main", remoteDir)

	dir := t.TempDir()
	gittest.Run(t, dir, "init", "-b", "main")
	gittest.Run(t, dir, "remote", "add", "origin", remoteDir)

	remotes, err := ListRemotes(context.Background(), dir)
	if err != nil {
		t.Fatalf("ListRemotes: %v", err)
	}
	want := []Remote{{Name: "origin", FetchURL: remoteDir, PushURL: remoteDir}}
	if !reflect.DeepEqual(remotes, want) {
		t.Fatalf("got %+v, want %+v", remotes, want)
	}
}

func TestAddRemote(t *testing.T) {
	remoteDir := t.TempDir() + "/remote.git"
	gittest.Run(t, "", "init", "--bare", "-b", "main", remoteDir)

	dir := t.TempDir()
	gittest.Run(t, dir, "init", "-b", "main")

	if err := AddRemote(context.Background(), dir, "origin", remoteDir); err != nil {
		t.Fatalf("AddRemote: %v", err)
	}

	remotes, err := ListRemotes(context.Background(), dir)
	if err != nil {
		t.Fatalf("ListRemotes: %v", err)
	}
	want := []Remote{{Name: "origin", FetchURL: remoteDir, PushURL: remoteDir}}
	if !reflect.DeepEqual(remotes, want) {
		t.Fatalf("got %+v, want %+v", remotes, want)
	}
}

func TestRemoveRemote(t *testing.T) {
	remoteDir := t.TempDir() + "/remote.git"
	gittest.Run(t, "", "init", "--bare", "-b", "main", remoteDir)

	dir := t.TempDir()
	gittest.Run(t, dir, "init", "-b", "main")
	gittest.Run(t, dir, "remote", "add", "origin", remoteDir)

	if err := RemoveRemote(context.Background(), dir, "origin"); err != nil {
		t.Fatalf("RemoveRemote: %v", err)
	}

	remotes, err := ListRemotes(context.Background(), dir)
	if err != nil {
		t.Fatalf("ListRemotes: %v", err)
	}
	if len(remotes) != 0 {
		t.Fatalf("got %+v, want no remotes", remotes)
	}
}

func TestEditRemote(t *testing.T) {
	oldRemoteDir := t.TempDir() + "/old.git"
	gittest.Run(t, "", "init", "--bare", "-b", "main", oldRemoteDir)
	newRemoteDir := t.TempDir() + "/new.git"
	gittest.Run(t, "", "init", "--bare", "-b", "main", newRemoteDir)

	dir := t.TempDir()
	gittest.Run(t, dir, "init", "-b", "main")
	gittest.Run(t, dir, "remote", "add", "origin", oldRemoteDir)

	if err := EditRemote(context.Background(), dir, "origin", newRemoteDir); err != nil {
		t.Fatalf("EditRemote: %v", err)
	}

	remotes, err := ListRemotes(context.Background(), dir)
	if err != nil {
		t.Fatalf("ListRemotes: %v", err)
	}
	want := []Remote{{Name: "origin", FetchURL: newRemoteDir, PushURL: newRemoteDir}}
	if !reflect.DeepEqual(remotes, want) {
		t.Fatalf("got %+v, want %+v", remotes, want)
	}
}
