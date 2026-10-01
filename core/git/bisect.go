package git

import (
	"context"
	"fmt"
	"path/filepath"
	"regexp"
	"strconv"
)

type BisectStatus struct {
	Active         bool
	Done           bool
	FoundSHA       string
	CurrentSHA     string
	RevisionsLeft  int
	StepsRemaining int
}

var (
	bisectInProgressPattern = regexp.MustCompile(`(?m)^Bisecting: (\d+) revisions? left to test after this \(roughly (\d+) steps?\)`)
	bisectCandidatePattern  = regexp.MustCompile(`(?m)^\[([0-9a-f]{40})\]`)
	bisectDonePattern       = regexp.MustCompile(`(?m)^([0-9a-f]{40}) is the first bad commit`)
)

func BisectStart(ctx context.Context, repoPath, badRev, goodRev string) (BisectStatus, error) {
	result, err := runResult(ctx, repoPath, "bisect", "start", badRev, goodRev)
	if err != nil {
		return BisectStatus{}, err
	}
	return parseBisectOutput(result.Stdout)
}

// BisectMark runs `git bisect <verdict>`; verdict is "good", "bad", or "skip".
func BisectMark(ctx context.Context, repoPath, verdict string) (BisectStatus, error) {
	result, err := runResult(ctx, repoPath, "bisect", verdict)
	if err != nil {
		return BisectStatus{}, err
	}
	return parseBisectOutput(result.Stdout)
}

func BisectReset(ctx context.Context, repoPath string) error {
	_, err := runResult(ctx, repoPath, "bisect", "reset")
	return err
}

func BisectActive(ctx context.Context, repoPath string) (bool, error) {
	dir, err := GitDir(ctx, repoPath)
	if err != nil {
		return false, err
	}
	return fileExists(filepath.Join(dir, "BISECT_START")), nil
}

func parseBisectOutput(output string) (BisectStatus, error) {
	if m := bisectDonePattern.FindStringSubmatch(output); m != nil {
		return BisectStatus{Active: true, Done: true, FoundSHA: m[1]}, nil
	}
	inProgress := bisectInProgressPattern.FindStringSubmatch(output)
	candidate := bisectCandidatePattern.FindStringSubmatch(output)
	if inProgress == nil || candidate == nil {
		return BisectStatus{}, fmt.Errorf("unrecognized bisect output: %q", output)
	}
	revisionsLeft, err := strconv.Atoi(inProgress[1])
	if err != nil {
		return BisectStatus{}, fmt.Errorf("malformed bisect revisions-left %q: %w", inProgress[1], err)
	}
	stepsRemaining, err := strconv.Atoi(inProgress[2])
	if err != nil {
		return BisectStatus{}, fmt.Errorf("malformed bisect steps-remaining %q: %w", inProgress[2], err)
	}
	return BisectStatus{
		Active:         true,
		CurrentSHA:     candidate[1],
		RevisionsLeft:  revisionsLeft,
		StepsRemaining: stepsRemaining,
	}, nil
}
