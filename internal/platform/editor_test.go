package platform

import (
	"reflect"
	"testing"
)

func TestOpenInEditorReturnsErrorForMissingExecutable(t *testing.T) {
	if err := OpenInEditor(t.TempDir(), "definitely-not-a-real-editor-binary"); err == nil {
		t.Fatal("expected an error for a nonexistent editor executable")
	}
}

func TestEditorFileArgs(t *testing.T) {
	tests := []struct {
		editor string
		want   []string
	}{
		{"code", []string{"/repo", "-g", "/repo/a.go:12"}},
		{"/usr/local/bin/cursor", []string{"/repo", "-g", "/repo/a.go:12"}},
		{`C:\Programs\VS Code\bin\Code.cmd`, []string{"/repo", "-g", "/repo/a.go:12"}},
		{"subl", []string{"/repo/a.go"}},
	}
	for _, tt := range tests {
		if got := editorFileArgs(tt.editor, "/repo", "/repo/a.go", 12); !reflect.DeepEqual(got, tt.want) {
			t.Errorf("editorFileArgs(%q) = %q, want %q", tt.editor, got, tt.want)
		}
	}
}
