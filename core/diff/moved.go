package diff

import (
	"slices"
	"strings"
)

// A single matching line (a brace, a blank line) is too weak to call a move.
const minMovedBlockLines = 2

type lineRef struct {
	hunkIdx int
	lineIdx int
}

// markMovedLines matches whole removed and added blocks with identical
// content, avoiding false positives from scattered matching lines.
func markMovedLines(fd *FileDiff) {
	var removedBlocks, addedBlocks [][]lineRef

	for hi := range fd.Hunks {
		lines := fd.Hunks[hi].Lines
		for i := 0; i < len(lines); {
			kind := lines[i].Kind
			if kind != LineRemoved && kind != LineAdded {
				i++
				continue
			}
			start := i
			for i < len(lines) && lines[i].Kind == kind {
				i++
			}
			// git may attach a shared blank line to either block.
			block := trimBlankEdges(fd, hi, start, i)
			if len(block) == 0 {
				continue
			}
			if kind == LineRemoved {
				removedBlocks = append(removedBlocks, block)
			} else {
				addedBlocks = append(addedBlocks, block)
			}
		}
	}

	content := func(fd *FileDiff, block []lineRef) []string {
		lines := make([]string, len(block))
		for i, ref := range block {
			lines[i] = fd.Hunks[ref.hunkIdx].Lines[ref.lineIdx].Content
		}
		return lines
	}

	matchedAdded := make([]bool, len(addedBlocks))
	for _, rb := range removedBlocks {
		if len(rb) < minMovedBlockLines {
			continue
		}
		removedContent := content(fd, rb)
		for ai, ab := range addedBlocks {
			if matchedAdded[ai] || len(ab) != len(rb) {
				continue
			}
			if !slices.Equal(content(fd, ab), removedContent) {
				continue
			}
			markBlock(fd, rb)
			markBlock(fd, ab)
			matchedAdded[ai] = true
			break
		}
	}
}

func trimBlankEdges(fd *FileDiff, hunkIdx, start, end int) []lineRef {
	lines := fd.Hunks[hunkIdx].Lines
	for start < end && strings.TrimSpace(lines[start].Content) == "" {
		start++
	}
	for end > start && strings.TrimSpace(lines[end-1].Content) == "" {
		end--
	}
	block := make([]lineRef, 0, end-start)
	for j := start; j < end; j++ {
		block = append(block, lineRef{hunkIdx: hunkIdx, lineIdx: j})
	}
	return block
}

func markBlock(fd *FileDiff, block []lineRef) {
	for _, ref := range block {
		fd.Hunks[ref.hunkIdx].Lines[ref.lineIdx].Moved = true
	}
}
