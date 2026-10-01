package git

import (
	"context"
	"path/filepath"
	"sort"
	"strings"
)

type LanguageStat struct {
	Extension string
	Count     int
}

const otherExtensionLabel = "other"

// LanguageBreakdown folds extensionless files and anything past limit into
// "other".
func LanguageBreakdown(ctx context.Context, repoPath string, limit int) ([]LanguageStat, error) {
	result, err := runResult(ctx, repoPath, "ls-files")
	if err != nil {
		return nil, err
	}

	counts := make(map[string]int)
	var order []string
	for _, line := range strings.Split(result.Stdout, "\n") {
		path := strings.TrimSpace(line)
		if path == "" {
			continue
		}
		ext := strings.ToLower(strings.TrimPrefix(filepath.Ext(path), "."))
		if ext == "" {
			ext = otherExtensionLabel
		}
		if counts[ext] == 0 {
			order = append(order, ext)
		}
		counts[ext]++
	}

	stats := make([]LanguageStat, len(order))
	for i, ext := range order {
		stats[i] = LanguageStat{Extension: ext, Count: counts[ext]}
	}
	sort.SliceStable(stats, func(i, j int) bool { return stats[i].Count > stats[j].Count })

	if len(stats) <= limit {
		return stats, nil
	}

	kept := make([]LanguageStat, 0, limit+1)
	otherCount := 0
	for i, s := range stats {
		if i < limit && s.Extension != otherExtensionLabel {
			kept = append(kept, s)
			continue
		}
		otherCount += s.Count
	}
	if otherCount > 0 {
		kept = append(kept, LanguageStat{Extension: otherExtensionLabel, Count: otherCount})
	}
	return kept, nil
}
