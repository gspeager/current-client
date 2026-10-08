package platform

import (
	"os"
	"path/filepath"
	"strings"
)

// Login scripts can print to stdout, so the shell's PATH follows this marker.
const pathMarker = "__current_client_path__"

// mergePath keeps first's order and adds the entries of second it lacks.
func mergePath(first, second string) string {
	seen := map[string]bool{}
	var merged []string
	for _, list := range []string{first, second} {
		for _, dir := range filepath.SplitList(list) {
			if dir != "" && !seen[dir] {
				seen[dir] = true
				merged = append(merged, dir)
			}
		}
	}
	return strings.Join(merged, string(os.PathListSeparator))
}
