package diff

import (
	"context"
	"fmt"
	"strings"

	"github.com/gspeager/current-client/core/gitexec"
)

// LineSelection picks changed lines in a hunk: added lines by their new line
// number, removed lines by their old one.
type LineSelection struct {
	Added   []int
	Removed []int
}

func StageLines(ctx context.Context, repoPath, path, hunkText string, sel LineSelection) error {
	return applyLines(ctx, repoPath, path, hunkText, sel, applyOptions{cached: true})
}

func UnstageLines(ctx context.Context, repoPath, path, hunkText string, sel LineSelection) error {
	return applyLines(ctx, repoPath, path, hunkText, sel, applyOptions{cached: true, reverse: true})
}

// DiscardLines reverts the selected lines in the working tree only.
func DiscardLines(ctx context.Context, repoPath, path, hunkText string, sel LineSelection) error {
	return applyLines(ctx, repoPath, path, hunkText, sel, applyOptions{reverse: true})
}

func applyLines(ctx context.Context, repoPath, path, hunkText string, sel LineSelection, opts applyOptions) error {
	partial, err := partialHunk(hunkText, sel, opts.reverse)
	if err != nil {
		return err
	}
	return applyHunk(ctx, repoPath, path, partial, opts)
}

// partialHunk keeps only the selected changes. The unselected ones have to
// match the file the patch is applied to: applied forward (to the old side),
// an unselected removal is still there and an unselected addition isn't;
// applied in reverse (to the new side), the other way round.
func partialHunk(hunkText string, sel LineSelection, reverse bool) (string, error) {
	lines := strings.Split(strings.TrimSuffix(hunkText, "\n"), "\n")
	m := hunkHeaderPattern.FindStringSubmatch(lines[0])
	if m == nil {
		return "", fmt.Errorf("not a hunk: %q", lines[0])
	}
	oldStart, newStart := atoi(m[1]), atoi(m[3])
	added, removed := toSet(sel.Added), toSet(sel.Removed)

	var body []string
	oldLine, newLine, oldCount, newCount := oldStart, newStart, 0, 0
	changed, previousKept := false, false
	keep := func(line string, inOld, inNew bool) {
		body = append(body, line)
		if inOld {
			oldCount++
		}
		if inNew {
			newCount++
		}
		previousKept = true
	}
	for _, line := range lines[1:] {
		if strings.HasPrefix(line, `\`) {
			// "\ No newline at end of file" belongs to the line before it.
			if previousKept {
				body = append(body, line)
			}
			continue
		}
		text := ""
		if line != "" {
			text = line[1:]
		}
		switch {
		case strings.HasPrefix(line, "+"):
			switch {
			case added[newLine]:
				keep(line, false, true)
				changed = true
			case reverse:
				keep(" "+text, true, true)
			default:
				previousKept = false
			}
			newLine++
		case strings.HasPrefix(line, "-"):
			switch {
			case removed[oldLine]:
				keep(line, true, false)
				changed = true
			case reverse:
				previousKept = false
			default:
				keep(" "+text, true, true)
			}
			oldLine++
		default:
			keep(" "+text, true, true)
			oldLine++
			newLine++
		}
	}
	if !changed {
		return "", &gitexec.AppError{Message: "Select a changed line in this hunk."}
	}
	return fmt.Sprintf("@@ -%d,%d +%d,%d @@\n%s\n", oldStart, oldCount, newStart, newCount, strings.Join(body, "\n")), nil
}

func toSet(numbers []int) map[int]bool {
	set := make(map[int]bool, len(numbers))
	for _, n := range numbers {
		set[n] = true
	}
	return set
}
