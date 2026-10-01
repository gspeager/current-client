package git

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	"github.com/gspeager/current-client/core/internal/gittest"
)

func TestParseBisectOutputInProgress(t *testing.T) {
	output := "Bisecting: 4 revisions left to test after this (roughly 2 steps)\n[1437b714519357be57d710c37e59e0a283bdf964] commit 5\n"
	got, err := parseBisectOutput(output)
	if err != nil {
		t.Fatalf("parseBisectOutput: %v", err)
	}
	want := BisectStatus{
		Active:         true,
		CurrentSHA:     "1437b714519357be57d710c37e59e0a283bdf964",
		RevisionsLeft:  4,
		StepsRemaining: 2,
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %+v, want %+v", got, want)
	}
}

func TestParseBisectOutputDone(t *testing.T) {
	output := "ffd275c6ea7be51d1f28a60488db66c249bba9a4 is the first bad commit\n" +
		"commit ffd275c6ea7be51d1f28a60488db66c249bba9a4\nAuthor: a <a@b.com>\n\n    commit 2\n"
	got, err := parseBisectOutput(output)
	if err != nil {
		t.Fatalf("parseBisectOutput: %v", err)
	}
	want := BisectStatus{Active: true, Done: true, FoundSHA: "ffd275c6ea7be51d1f28a60488db66c249bba9a4"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %+v, want %+v", got, want)
	}
}

func TestParseBisectOutputMalformed(t *testing.T) {
	if _, err := parseBisectOutput("nonsense"); err == nil {
		t.Fatal("expected an error for unrecognized bisect output")
	}
}

func TestBisectActiveReflectsSessionState(t *testing.T) {
	dir := gittest.InitRepo(t)
	gittest.CommitFile(t, dir, "file.txt", "v1", "v1")
	first := gittest.Run(t, dir, "rev-parse", "HEAD")
	gittest.CommitFile(t, dir, "file.txt", "v2", "v2")
	head := gittest.Run(t, dir, "rev-parse", "HEAD")

	if active, err := BisectActive(context.Background(), dir); err != nil || active {
		t.Fatalf("BisectActive = %v, %v; want false before any bisect started", active, err)
	}

	if _, err := BisectStart(context.Background(), dir, head, first); err != nil {
		t.Fatalf("BisectStart: %v", err)
	}
	if active, err := BisectActive(context.Background(), dir); err != nil || !active {
		t.Fatalf("BisectActive = %v, %v; want true mid-bisect", active, err)
	}

	if err := BisectReset(context.Background(), dir); err != nil {
		t.Fatalf("BisectReset: %v", err)
	}
	if active, err := BisectActive(context.Background(), dir); err != nil || active {
		t.Fatalf("BisectActive = %v, %v; want false after reset", active, err)
	}
}

func TestBisectConvergesOnKnownBadCommit(t *testing.T) {
	dir := gittest.InitRepo(t)

	const n = 10
	const firstBadIndex = 6 // 1-based; commits 1-5 good, 6-10 bad
	var shas []string
	for i := 1; i <= n; i++ {
		gittest.CommitFile(t, dir, "file.txt", fmt.Sprintf("commit %d\n", i), fmt.Sprintf("commit %d", i))
		shas = append(shas, gittest.Run(t, dir, "rev-parse", "HEAD"))
	}
	indexOf := make(map[string]int, n)
	for i, sha := range shas {
		indexOf[sha] = i + 1
	}

	status, err := BisectStart(context.Background(), dir, shas[n-1], shas[0])
	if err != nil {
		t.Fatalf("BisectStart: %v", err)
	}

	steps := 0
	for !status.Done {
		steps++
		if steps > n {
			t.Fatalf("bisect did not converge within %d steps", n)
		}
		idx, ok := indexOf[status.CurrentSHA]
		if !ok {
			t.Fatalf("bisect landed on unknown commit %q", status.CurrentSHA)
		}
		verdict := "good"
		if idx >= firstBadIndex {
			verdict = "bad"
		}
		status, err = BisectMark(context.Background(), dir, verdict)
		if err != nil {
			t.Fatalf("BisectMark(%s): %v", verdict, err)
		}
	}

	if status.FoundSHA != shas[firstBadIndex-1] {
		t.Fatalf("FoundSHA = %q, want %q (commit %d)", status.FoundSHA, shas[firstBadIndex-1], firstBadIndex)
	}

	if err := BisectReset(context.Background(), dir); err != nil {
		t.Fatalf("BisectReset: %v", err)
	}
	if head := gittest.Run(t, dir, "rev-parse", "HEAD"); head != shas[n-1] {
		t.Fatalf("HEAD after reset = %q, want back at %q", head, shas[n-1])
	}
}
