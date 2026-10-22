package app

import (
	"os/exec"
	"strings"
	"testing"
)

// A build with the nohost tag must link no host API client and no host
// content code; the ordinary build must.
func TestNohostBuildLeavesOutHostIntegration(t *testing.T) {
	hostPackages := []string{"core/hostapi", "core/hostcontent"}
	without := dependencies(t, "-tags", "nohost")
	for _, pkg := range hostPackages {
		if strings.Contains(without, pkg) {
			t.Errorf("nohost build still depends on %s", pkg)
		}
	}
	if with := dependencies(t); !strings.Contains(with, "core/hostapi") {
		t.Error("the ordinary build should depend on core/hostapi")
	}
}

func dependencies(t *testing.T, flags ...string) string {
	t.Helper()
	args := append([]string{"list", "-deps"}, flags...)
	cmd := exec.Command("go", append(args, "github.com/gspeager/current-client")...)
	cmd.Dir = ".."
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("go %v: %v", args, err)
	}
	return string(out)
}
