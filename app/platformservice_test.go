package app

import (
	"errors"
	"os/exec"
	"testing"

	"github.com/gspeager/current-client/internal/platform"
)

func TestMissingEditorIsExplained(t *testing.T) {
	err := editorError(platform.OpenInEditor(t.TempDir(), "definitely-not-a-real-editor"), "definitely-not-a-real-editor")
	if err == nil || err.Error() != "The editor definitely-not-a-real-editor wasn't found. Choose another in Settings." {
		t.Fatalf("err = %v", err)
	}
	notFound := &exec.Error{Name: "code", Err: exec.ErrNotFound}
	if got := editorError(notFound, "").Error(); got != "VS Code's code command isn't installed. Choose an editor in Settings." {
		t.Fatalf("default editor message = %q", got)
	}
	other := errors.New("permission denied")
	if got := editorError(other, ""); got != other {
		t.Fatalf("other errors should pass through, got %v", got)
	}
	if editorError(nil, "") != nil {
		t.Fatal("no error should stay no error")
	}
}

func TestOpenWebPageOnlyTakesWebAddresses(t *testing.T) {
	for _, u := range []string{"https://github.com/owner/repo/issues/1", "http://git.example.com:8080/o/r/issues/2"} {
		if err := checkWebPageURL(u); err != nil {
			t.Errorf("checkWebPageURL(%q) = %v", u, err)
		}
	}
	for _, u := range []string{"", "file:///etc/passwd", "javascript:alert(1)", "ftp://example.com/x", "owner/repo", "https://"} {
		if err := checkWebPageURL(u); err == nil {
			t.Errorf("checkWebPageURL(%q) accepted it", u)
		}
	}
}
