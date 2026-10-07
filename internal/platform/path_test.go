package platform

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMergePathKeepsTheFirstOrderAndAddsWhatsMissing(t *testing.T) {
	sep := string(os.PathListSeparator)
	first := strings.Join([]string{"/opt/homebrew/bin", "/usr/bin"}, sep)
	second := strings.Join([]string{"/usr/bin", "/bin", ""}, sep)
	want := strings.Join([]string{"/opt/homebrew/bin", "/usr/bin", "/bin"}, sep)
	if got := mergePath(first, second); got != want {
		t.Fatalf("mergePath = %q, want %q", got, want)
	}
}

func TestUseLoginShellPathKeepsWhatTheAppHad(t *testing.T) {
	t.Setenv("PATH", "/usr/bin:/bin:/usr/sbin:/sbin")
	UseLoginShellPath()
	dirs := filepath.SplitList(os.Getenv("PATH"))
	for _, want := range []string{"/usr/bin", "/bin", "/usr/sbin", "/sbin"} {
		found := false
		for _, dir := range dirs {
			found = found || dir == want
		}
		if !found {
			t.Errorf("PATH %q lost %s", os.Getenv("PATH"), want)
		}
	}
	if strings.Contains(os.Getenv("PATH"), pathMarker) {
		t.Errorf("PATH %q contains the marker", os.Getenv("PATH"))
	}
}
