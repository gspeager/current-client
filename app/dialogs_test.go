package app

import (
	"errors"
	"testing"
)

func TestDialogResultTellsCancelFromFailure(t *testing.T) {
	if path, err := dialogResult("", errors.New("cancelled by user")); path != "" || err != nil {
		t.Errorf("Windows cancel = %q, %v; want no path and no error", path, err)
	}
	if path, err := dialogResult("", nil); path != "" || err != nil {
		t.Errorf("macOS/Linux cancel = %q, %v; want no path and no error", path, err)
	}
	failure := errors.New("unable to create dialog")
	if _, err := dialogResult("", failure); err != failure {
		t.Errorf("failure = %v, want it passed on", err)
	}
	if path, err := dialogResult("/repos/app", nil); path != "/repos/app" || err != nil {
		t.Errorf("choice = %q, %v", path, err)
	}
}
