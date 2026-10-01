package platform

import "testing"

func TestOpenInEditorReturnsErrorForMissingExecutable(t *testing.T) {
	if err := OpenInEditor(t.TempDir(), "definitely-not-a-real-editor-binary"); err == nil {
		t.Fatal("expected an error for a nonexistent editor executable")
	}
}
