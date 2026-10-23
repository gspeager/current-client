//go:build !nohost

package app

import (
	"strings"
	"testing"
)

func TestJobLogIsFetchedOnlyWhenAskedAndCleaned(t *testing.T) {
	repo := enabledGitLab(t)
	api := "/api/v4/projects/group%2Fapp"
	var lines []string
	for i := 1; i <= 80; i++ {
		lines = append(lines, "line "+strings.Repeat("x", i))
	}
	trace := "\x1b[0Ksection_start:1:step\r\x1b[0K\x1b[32;1mRunning\x1b[0;m\n" + strings.Join(lines, "\n") + "\n\x1b]8;;https://x\x07link\x1b]8;;\x07 \xff bad byte\n\x1b[31;1mERROR: Job failed: exit code 1\x1b[0;m\n"
	transport := &routeTransport{routes: map[string]string{api + "/jobs/5/trace": trace}, hits: map[string]int{}}
	swapHostClient(t, transport)
	var s HostService

	log, err := s.JobLog(repo, 5, false)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(log.Text, "\x1b") || strings.Contains(log.Text, "\r") || !strings.Contains(log.Text, "ERROR: Job failed: exit code 1") || !strings.Contains(log.Text, "link � bad byte") {
		t.Fatalf("log text not cleaned:\n%q", log.Text)
	}
	if got := strings.Count(log.Text, "\n") + 1; got != 50 || !log.Truncated || log.Full {
		t.Fatalf("tail = %d lines, truncated=%v, full=%v", got, log.Truncated, log.Full)
	}
	if strings.Contains(log.Text, "Running") {
		t.Fatal("the tail should start after the first lines")
	}
	s.JobLog(repo, 5, false)
	if transport.hits[api+"/jobs/5/trace"] != 1 {
		t.Fatalf("the tail was fetched %d times, want once", transport.hits[api+"/jobs/5/trace"])
	}

	full, err := s.JobLog(repo, 5, true)
	if err != nil || !full.Full || full.Truncated || !strings.HasPrefix(full.Text, "section_start:1:stepRunning") {
		t.Fatalf("full log = %+v, %v", full, err)
	}
	if transport.hits[api+"/jobs/5/trace"] != 2 {
		t.Fatalf("the full log should be its own fetch, got %d", transport.hits[api+"/jobs/5/trace"])
	}
}
