package diff

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/gspeager/current-client/core/internal/gittest"
)

// Not valid UTF-8, and contains NUL and CR LF, so any text handling would corrupt it.
var pngBytes = []byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n', 0x00, 0xff, 0xfe}

func TestReadRevisionFileAtCommitAndIndex(t *testing.T) {
	ctx := context.Background()
	dir := gittest.InitRepo(t)
	gittest.WriteFile(t, dir, "logo.png", string(pngBytes))
	gittest.Run(t, dir, "add", "logo.png")
	gittest.Run(t, dir, "commit", "-m", "add logo")
	edited := append([]byte{}, pngBytes...)
	edited[len(edited)-1] = 0x01
	gittest.WriteFile(t, dir, "logo.png", string(edited))
	gittest.Run(t, dir, "add", "logo.png")

	got, found, err := ReadRevisionFile(ctx, dir, "HEAD", "logo.png", 1<<20)
	if err != nil || !found || !bytes.Equal(got, pngBytes) {
		t.Fatalf("at HEAD = %v, %v, %v; want the committed bytes", got, found, err)
	}
	got, found, err = ReadRevisionFile(ctx, dir, "", "logo.png", 1<<20)
	if err != nil || !found || !bytes.Equal(got, edited) {
		t.Fatalf("in the index = %v, %v, %v; want the staged bytes", got, found, err)
	}
}

func TestReadRevisionFileMissingAndTooLarge(t *testing.T) {
	ctx := context.Background()
	dir := gittest.InitRepo(t)
	gittest.CommitFile(t, dir, "big.bin", "0123456789", "initial")

	if _, found, err := ReadRevisionFile(ctx, dir, "HEAD", "nope.png", 1<<20); found || err != nil {
		t.Fatalf("missing path = %v, %v; want not found and no error", found, err)
	}
	if _, found, err := ReadRevisionFile(ctx, dir, "4b825dc642cb6eb9a060e54bf8d69288fbee4904", "big.bin", 1<<20); found || err != nil {
		t.Fatalf("path in the empty tree = %v, %v; want not found and no error", found, err)
	}
	if _, found, err := ReadRevisionFile(ctx, dir, "HEAD", "big.bin", 5); !found || !errors.Is(err, ErrFileTooLarge) {
		t.Fatalf("over the limit = %v, %v; want found and ErrFileTooLarge", found, err)
	}
}

func TestReadWorkingFile(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "img"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "img", "logo.png"), pngBytes, 0o644); err != nil {
		t.Fatal(err)
	}

	got, found, err := ReadWorkingFile(dir, "img/logo.png", 1<<20)
	if err != nil || !found || !bytes.Equal(got, pngBytes) {
		t.Fatalf("ReadWorkingFile = %v, %v, %v; want the file's bytes", got, found, err)
	}
	if _, found, err := ReadWorkingFile(dir, "gone.png", 1<<20); found || err != nil {
		t.Fatalf("missing file = %v, %v; want not found and no error", found, err)
	}
	if _, _, err := ReadWorkingFile(dir, "img/logo.png", 3); !errors.Is(err, ErrFileTooLarge) {
		t.Fatalf("over the limit: err = %v, want ErrFileTooLarge", err)
	}
	if _, _, err := ReadWorkingFile(dir, "../outside.png", 1<<20); err == nil {
		t.Fatal("a path outside the repository was read")
	}
}
