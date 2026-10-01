package git

import (
	"context"
	"reflect"
	"strings"
	"testing"

	"github.com/gspeager/current-client/core/internal/gittest"
)

func TestParseTags(t *testing.T) {
	output := "v1.0.0\tabc123\t\t2024-01-15T10:30:00-08:00\n" +
		"v2.0.0\tdef456\t789xyz\t2024-02-20T09:00:00-08:00\n"
	got := parseTags(output)
	want := []Tag{
		{Name: "v1.0.0", SHA: "abc123", Annotated: false, Date: "2024-01-15T10:30:00-08:00"},
		{Name: "v2.0.0", SHA: "789xyz", Annotated: true, Date: "2024-02-20T09:00:00-08:00"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %+v, want %+v", got, want)
	}
}

func TestParseTagsEmpty(t *testing.T) {
	if got := parseTags(""); len(got) != 0 {
		t.Fatalf("got %+v, want empty", got)
	}
}

func initTagRepo(t *testing.T) (dir string) {
	t.Helper()
	dir = gittest.InitRepo(t)
	gittest.CommitFile(t, dir, "file.txt", "v1", "initial")
	return dir
}

func TestCreateTagLightweight(t *testing.T) {
	dir := initTagRepo(t)

	if err := CreateTag(context.Background(), dir, "v1.0.0", "", ""); err != nil {
		t.Fatalf("CreateTag: %v", err)
	}

	tags, err := ListTags(context.Background(), dir)
	if err != nil {
		t.Fatalf("ListTags: %v", err)
	}
	if len(tags) != 1 || tags[0].Name != "v1.0.0" || tags[0].Annotated {
		t.Fatalf("got %+v, want a single lightweight tag v1.0.0", tags)
	}
}

func TestCreateTagAnnotatedWithMessage(t *testing.T) {
	dir := initTagRepo(t)

	if err := CreateTag(context.Background(), dir, "v1.0.0", "Release 1.0.0", ""); err != nil {
		t.Fatalf("CreateTag: %v", err)
	}

	tags, err := ListTags(context.Background(), dir)
	if err != nil {
		t.Fatalf("ListTags: %v", err)
	}
	if len(tags) != 1 || !tags[0].Annotated {
		t.Fatalf("got %+v, want a single annotated tag", tags)
	}

	subject, body, err := TagMessage(context.Background(), dir, "v1.0.0")
	if err != nil {
		t.Fatalf("TagMessage: %v", err)
	}
	if subject != "Release 1.0.0" {
		t.Fatalf("subject = %q, want %q", subject, "Release 1.0.0")
	}
	if body != "" {
		t.Fatalf("body = %q, want empty for a single-line message", body)
	}
}

func TestCreateTagAtSpecificTarget(t *testing.T) {
	dir := initTagRepo(t)
	firstCommit := strings.TrimSpace(gittest.Run(t, dir, "rev-parse", "HEAD"))
	gittest.CommitFile(t, dir, "file.txt", "v2", "second")

	if err := CreateTag(context.Background(), dir, "v1.0.0", "", firstCommit); err != nil {
		t.Fatalf("CreateTag: %v", err)
	}

	tags, err := ListTags(context.Background(), dir)
	if err != nil {
		t.Fatalf("ListTags: %v", err)
	}
	if len(tags) != 1 || tags[0].SHA != firstCommit {
		t.Fatalf("got %+v, want the tag to point at %q", tags, firstCommit)
	}
}

// Lightweight tags have no message of their own; callers check Tag.Annotated.
func TestTagMessageFallsBackToCommitMessageForLightweightTag(t *testing.T) {
	dir := initTagRepo(t)
	gittest.Run(t, dir, "tag", "v1.0.0")

	subject, _, err := TagMessage(context.Background(), dir, "v1.0.0")
	if err != nil {
		t.Fatalf("TagMessage: %v", err)
	}
	if subject != "initial" {
		t.Fatalf("subject = %q, want the underlying commit's subject %q", subject, "initial")
	}
}

func TestDeleteTagRemovesIt(t *testing.T) {
	dir := initTagRepo(t)
	gittest.Run(t, dir, "tag", "v1.0.0")

	if err := DeleteTag(context.Background(), dir, "v1.0.0"); err != nil {
		t.Fatalf("DeleteTag: %v", err)
	}

	tags, err := ListTags(context.Background(), dir)
	if err != nil {
		t.Fatalf("ListTags: %v", err)
	}
	if len(tags) != 0 {
		t.Fatalf("got %+v, want no tags after delete", tags)
	}
}

func TestPushTagPublishesToRemote(t *testing.T) {
	remoteDir := t.TempDir() + "/remote.git"
	gittest.Run(t, "", "init", "--bare", "-b", "main", remoteDir)

	dir := initTagRepo(t)
	gittest.Run(t, dir, "remote", "add", "origin", remoteDir)
	gittest.Run(t, dir, "push", "origin", "main")
	gittest.Run(t, dir, "tag", "v1.0.0")

	if err := PushTag(context.Background(), dir, "origin", "v1.0.0"); err != nil {
		t.Fatalf("PushTag: %v", err)
	}

	remoteTags := gittest.Run(t, "", "ls-remote", "--tags", remoteDir, "v1.0.0")
	if remoteTags == "" {
		t.Fatal("expected v1.0.0 to exist on the remote after push")
	}
}

func TestTagsInRange(t *testing.T) {
	dir := gittest.InitRepo(t)
	gittest.CommitFile(t, dir, "a.txt", "1", "first")
	gittest.Run(t, dir, "tag", "v0.1.0")
	gittest.CommitFile(t, dir, "a.txt", "2", "second")
	gittest.Run(t, dir, "tag", "-a", "v0.2.0", "-m", "Release 0.2.0")
	gittest.CommitFile(t, dir, "a.txt", "3", "third")
	gittest.Run(t, dir, "checkout", "-q", "-b", "side")
	gittest.CommitFile(t, dir, "b.txt", "1", "side work")
	gittest.Run(t, dir, "tag", "side-only")
	gittest.Run(t, dir, "checkout", "-q", "main")

	names := func(from string) string {
		t.Helper()
		tags, err := TagsInRange(context.Background(), dir, from, "main")
		if err != nil {
			t.Fatalf("TagsInRange: %v", err)
		}
		var got []string
		for _, tag := range tags {
			if len(tag.Date) != len("2006-01-02") {
				t.Errorf("tag %s date = %q", tag.Name, tag.Date)
			}
			got = append(got, tag.Name)
		}
		return strings.Join(got, ",")
	}
	if got := names(""); got != "v0.2.0,v0.1.0" {
		t.Errorf("all history = %s, want v0.2.0,v0.1.0 (newest first, side branch excluded)", got)
	}
	if got := names("v0.1.0"); got != "v0.2.0" {
		t.Errorf("after v0.1.0 = %s, want v0.2.0", got)
	}
}
