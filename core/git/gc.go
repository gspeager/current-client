package git

import (
	"context"
	"strconv"
	"strings"

	"github.com/gspeager/current-client/core/gitexec"
)

type RepoDiskUsage struct {
	LooseObjectCount int
	LooseSizeKB      int
	PackCount        int
	PackedSizeKB     int
	TotalSizeKB      int
}

func ComputeRepoDiskUsage(ctx context.Context, repoPath string) (RepoDiskUsage, error) {
	result, err := runResult(ctx, repoPath, "count-objects", "-v")
	if err != nil {
		return RepoDiskUsage{}, err
	}
	fields := make(map[string]int)
	for _, line := range strings.Split(result.Stdout, "\n") {
		key, value, found := strings.Cut(line, ":")
		if !found {
			continue
		}
		n, err := strconv.Atoi(strings.TrimSpace(value))
		if err != nil {
			continue
		}
		fields[strings.TrimSpace(key)] = n
	}
	usage := RepoDiskUsage{
		LooseObjectCount: fields["count"],
		LooseSizeKB:      fields["size"],
		PackCount:        fields["packs"],
		PackedSizeKB:     fields["size-pack"],
	}
	usage.TotalSizeKB = usage.LooseSizeKB + usage.PackedSizeKB
	return usage, nil
}

// FsckCheck returns each reported problem; empty means healthy. It uses the
// unchecked Run because fsck exits non-zero exactly when it has something to
// report, and RunChecked would discard that output.
func FsckCheck(ctx context.Context, repoPath string) ([]string, error) {
	result, err := gitexec.NewExecutor("").Run(ctx, gitexec.Command{Dir: repoPath, Args: []string{"fsck", "--no-dangling"}})
	if err != nil {
		return nil, err
	}
	var issues []string
	for _, line := range strings.Split(result.Stdout+result.Stderr, "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			issues = append(issues, line)
		}
	}
	return issues, nil
}

func RunGC(ctx context.Context, repoPath string) error {
	_, err := runResult(ctx, repoPath, "gc", "--prune")
	return err
}
