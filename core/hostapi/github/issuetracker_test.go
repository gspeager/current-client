package github

import (
	"context"
	"errors"
	"testing"

	"github.com/gspeager/current-client/core/hostapi"
)

func TestIssueTrackerKeysAreNumbers(t *testing.T) {
	var p Provider
	if got := p.KeyPattern().FindAllStringSubmatch("Fix #12, see group/x#7 and v1.2#3x", -1); len(got) != 2 || got[0][1] != "12" || got[1][1] != "7" {
		t.Fatalf("KeyPattern matches = %v", got)
	}
	_, err := p.IssueByKey(context.Background(), "g/r", "PROJ-7")
	var hostErr *hostapi.HostError
	if !errors.As(err, &hostErr) || hostErr.Status != 404 {
		t.Fatalf("a non-numeric key should be a 404, got %v", err)
	}
}
