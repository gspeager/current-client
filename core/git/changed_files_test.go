package git

import (
	"context"
	"reflect"
	"testing"

	"github.com/gspeager/current-client/core/internal/gittest"
)

func TestParseChangedFilesFixture(t *testing.T) {
	output := "M\x00main.txt\x00A\x00new.txt\x00"
	files, err := parseChangedFiles(output)
	if err != nil {
		t.Fatalf("parseChangedFiles: %v", err)
	}
	want := []ChangedFile{
		{Status: "M", Path: "main.txt"},
		{Status: "A", Path: "new.txt"},
	}
	if !reflect.DeepEqual(files, want) {
		t.Fatalf("got %+v, want %+v", files, want)
	}
}

func TestParseChangedFilesRename(t *testing.T) {
	output := "R100\x00old.txt\x00new.txt\x00"
	files, err := parseChangedFiles(output)
	if err != nil {
		t.Fatalf("parseChangedFiles: %v", err)
	}
	want := []ChangedFile{
		{Status: "R100", Path: "new.txt", OrigPath: "old.txt"},
	}
	if !reflect.DeepEqual(files, want) {
		t.Fatalf("got %+v, want %+v", files, want)
	}
}

func TestParseChangedFilesEmpty(t *testing.T) {
	files, err := parseChangedFiles("")
	if err != nil {
		t.Fatalf("parseChangedFiles: %v", err)
	}
	if len(files) != 0 {
		t.Fatalf("got %+v, want empty", files)
	}
}

func TestChangedFilesRealRepo(t *testing.T) {
	dir := gittest.InitRepo(t)
	commit := func(name, content, message string) string {
		gittest.WriteFile(t, dir, name, content)
		gittest.Run(t, dir, "add", ".")
		gittest.Run(t, dir, "commit", "-q", "-m", message)
		return gittest.Run(t, dir, "rev-parse", "HEAD")
	}
	root := commit("root.txt", "root\n", "root commit")
	regular := commit("main.txt", "main1\n", "main commit")
	gittest.Run(t, dir, "checkout", "-qb", "feature")
	commit("feature.txt", "feat\n", "feature commit")
	gittest.Run(t, dir, "checkout", "-q", "main")
	commit("main.txt", "main1\nmainedit\n", "main edit")
	gittest.Run(t, dir, "merge", "--no-edit", "-q", "feature")
	merge := gittest.Run(t, dir, "rev-parse", "HEAD")
	ctx := context.Background()

	rootFiles, err := ChangedFiles(ctx, dir, root)
	if err != nil {
		t.Fatalf("ChangedFiles(root): %v", err)
	}
	if want := []ChangedFile{{Status: "A", Path: "root.txt"}}; !reflect.DeepEqual(rootFiles, want) {
		t.Fatalf("root commit: got %+v, want %+v", rootFiles, want)
	}

	regularFiles, err := ChangedFiles(ctx, dir, regular)
	if err != nil {
		t.Fatalf("ChangedFiles(regular): %v", err)
	}
	if want := []ChangedFile{{Status: "A", Path: "main.txt"}}; !reflect.DeepEqual(regularFiles, want) {
		t.Fatalf("regular commit: got %+v, want %+v", regularFiles, want)
	}

	mergeFiles, err := ChangedFiles(ctx, dir, merge)
	if err != nil {
		t.Fatalf("ChangedFiles(merge): %v", err)
	}
	if want := []ChangedFile{{Status: "A", Path: "feature.txt"}}; !reflect.DeepEqual(mergeFiles, want) {
		t.Fatalf("merge commit: got %+v, want what it brought in from feature, %+v", mergeFiles, want)
	}
	mergeStats, err := CommitNumstat(ctx, dir, merge)
	if err != nil {
		t.Fatalf("CommitNumstat(merge): %v", err)
	}
	if len(mergeStats) != 1 || mergeStats[0].Path != "feature.txt" || mergeStats[0].Added != 1 {
		t.Fatalf("merge numstat = %+v, want feature.txt +1", mergeStats)
	}
}

func TestRefRangeChangedFilesTwoBranches(t *testing.T) {
	dir := gittest.InitRepo(t)
	gittest.CommitFile(t, dir, "base.txt", "v1", "base commit")
	gittest.Run(t, dir, "checkout", "-q", "-b", "feature")
	gittest.CommitFile(t, dir, "feature.txt", "feature content, unrelated to main.txt below", "feature commit")
	gittest.Run(t, dir, "checkout", "-q", "main")
	gittest.CommitFile(t, dir, "main.txt", "main content, unrelated to feature.txt above", "main commit")

	files, err := RefRangeChangedFiles(context.Background(), dir, "feature", "main")
	if err != nil {
		t.Fatalf("RefRangeChangedFiles: %v", err)
	}
	want := []ChangedFile{
		{Status: "D", Path: "feature.txt"},
		{Status: "A", Path: "main.txt"},
	}
	if !reflect.DeepEqual(files, want) {
		t.Fatalf("got %+v, want %+v", files, want)
	}
}
