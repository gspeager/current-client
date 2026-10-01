package repository

import (
	"os"
	"path/filepath"
	"strings"
)

// AddToGitignore does nothing when the pattern is already listed.
func AddToGitignore(repoPath, pattern string) error {
	path := filepath.Join(repoPath, ".gitignore")

	existing, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	lines := strings.Split(string(existing), "\n")
	for _, line := range lines {
		if strings.TrimSpace(line) == pattern {
			return nil
		}
	}

	content := string(existing)
	if content != "" && !strings.HasSuffix(content, "\n") {
		content += "\n"
	}
	content += pattern + "\n"

	return os.WriteFile(path, []byte(content), 0o644)
}
