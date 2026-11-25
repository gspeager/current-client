package app

import (
	"fmt"
	"testing"

	"github.com/gspeager/current-client/core/board"
	"github.com/gspeager/current-client/core/git"
)

func TestBoardColumnListsWhatCollidesFirst(t *testing.T) {
	var column board.Column
	for i := range 3 * mostBoardFiles {
		change := git.TreeChange{Added: 2, Removed: 1}
		change.Path, change.Status = fmt.Sprintf("file-%d", i), "M"
		column.Files = append(column.Files, change)
	}
	last := column.Files[len(column.Files)-1].Path
	info := boardColumn(column, map[string]bool{last: true, "committed": true})
	if info.Changed != 300 || len(info.Files) != mostBoardFiles || info.Added != 600 || info.Removed != 300 {
		t.Errorf("%d changed, %d listed, +%d -%d", info.Changed, len(info.Files), info.Added, info.Removed)
	}
	if info.Files[0].Path != last || info.Files[1].Path != "file-0" {
		t.Errorf("listed %v first, want %s", info.Files[:2], last)
	}
	if none := boardColumn(board.Column{}, nil); none.Files == nil || none.Changed != 0 {
		t.Errorf("a clean column = %+v", none)
	}
}
