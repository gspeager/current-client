package git

import (
	"cmp"
	"context"
	"path/filepath"
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
	files, err := trackedFiles(ctx, repoPath)
	if err != nil {
		return nil, err
	}

	exts := make([]string, len(files))
	for i, path := range files {
		exts[i] = cmp.Or(strings.ToLower(strings.TrimPrefix(filepath.Ext(path), ".")), otherExtensionLabel)
	}
	stats := []LanguageStat{}
	for _, r := range rank(exts) {
		stats = append(stats, LanguageStat{Extension: r.key, Count: r.count})
	}

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
