package git

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/gspeager/current-client/core/internal/gittest"
)

func TestFilesAtReadsManyFilesInOneGo(t *testing.T) {
	dir := gittest.InitRepo(t)
	gittest.CommitFile(t, dir, "a.txt", "first\n", "first")
	if err := os.MkdirAll(filepath.Join(dir, "dir"), 0o755); err != nil {
		t.Fatal(err)
	}
	gittest.WriteFile(t, dir, "dir/b.txt", "b\n")
	gittest.WriteFile(t, dir, "with space.txt", "spaced")
	gittest.WriteFile(t, dir, "bin.dat", "\x00\x01\n\x02")
	gittest.WriteFile(t, dir, "a.txt", "second\n")
	gittest.Run(t, dir, "add", ".")
	gittest.Run(t, dir, "commit", "-q", "-m", "second")
	gittest.WriteFile(t, dir, "a.txt", "staged\n")
	gittest.Run(t, dir, "add", "a.txt")

	files := []RevPath{
		{"HEAD~1", "a.txt"}, {"HEAD", "a.txt"}, {"", "a.txt"},
		{"HEAD", "dir/b.txt"}, {"HEAD", "with space.txt"}, {"HEAD", "bin.dat"},
		{"HEAD", "missing.txt"}, {"HEAD~1", "dir/b.txt"}, {"nope", "a.txt"}, {"HEAD", "dir"},
	}
	got, err := FilesAt(context.Background(), dir, files)
	if err != nil {
		t.Fatal(err)
	}
	want := map[RevPath][]byte{
		{"HEAD~1", "a.txt"}: []byte("first\n"), {"HEAD", "a.txt"}: []byte("second\n"), {"", "a.txt"}: []byte("staged\n"),
		{"HEAD", "dir/b.txt"}: []byte("b\n"), {"HEAD", "with space.txt"}: []byte("spaced"), {"HEAD", "bin.dat"}: []byte("\x00\x01\n\x02"),
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("FilesAt = %q, want %q", got, want)
	}

	data, found, err := FileAt(context.Background(), dir, "HEAD~1", "a.txt")
	if err != nil || !found || string(data) != "first\n" {
		t.Errorf("FileAt = %q, %v, %v", data, found, err)
	}
	if _, found, err := FileAt(context.Background(), dir, "HEAD", "missing.txt"); err != nil || found {
		t.Errorf("missing: found = %v, %v", found, err)
	}
	if _, err := FilesAt(context.Background(), dir, []RevPath{{"HEAD", "a\nb"}}); err == nil {
		t.Error("a line break in the name should be refused")
	}
}
