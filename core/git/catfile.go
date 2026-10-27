package git

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/gspeager/current-client/core/gitexec"
)

// RevPath is a file at a revision; Rev "" is the index.
type RevPath struct {
	Rev  string
	Path string
}

// FileAt reads path as it is at rev, false when it isn't there. For a file
// that may be large, diff.ReadRevisionFile takes a limit.
func FileAt(ctx context.Context, repoPath, rev, path string) ([]byte, bool, error) {
	files, err := FilesAt(ctx, repoPath, []RevPath{{rev, path}})
	if err != nil {
		return nil, false, err
	}
	data, found := files[RevPath{rev, path}]
	return data, found, nil
}

// FilesAt reads many files in one process (cat-file --batch). A file that
// isn't at its revision, or whose revision doesn't exist, is left out, as
// is a path that is a directory or a submodule there.
func FilesAt(ctx context.Context, repoPath string, files []RevPath) (map[RevPath][]byte, error) {
	found := map[RevPath][]byte{}
	if len(files) == 0 {
		return found, nil
	}
	var input strings.Builder
	for _, f := range files {
		if strings.ContainsAny(f.Rev+f.Path, "\n") {
			return nil, fmt.Errorf("can't read %q at %q: a line break in the name", f.Path, f.Rev)
		}
		input.WriteString(f.Rev + ":" + f.Path + "\n")
	}
	result, err := gitexec.NewExecutor("").RunChecked(ctx, gitexec.Command{
		Dir:   repoPath,
		Args:  []string{"cat-file", "--batch"},
		Stdin: strings.NewReader(input.String()),
	})
	if err != nil {
		return nil, err
	}
	// "<oid> <type> <size>\n<content>\n", or "<spec> missing\n".
	rest := result.Stdout
	for _, f := range files {
		header, after, ok := strings.Cut(rest, "\n")
		if !ok {
			return nil, fmt.Errorf("cat-file stopped before %s", f.Path)
		}
		rest = after
		if strings.HasSuffix(header, " missing") || strings.HasSuffix(header, " ambiguous") {
			continue
		}
		fields := strings.Fields(header)
		size, err := strconv.Atoi(fields[len(fields)-1])
		if len(fields) != 3 || err != nil || len(rest) < size+1 {
			return nil, fmt.Errorf("cat-file's answer for %s can't be read: %q", f.Path, header)
		}
		if fields[1] == "blob" {
			found[f] = []byte(rest[:size])
		}
		rest = rest[size+1:]
	}
	return found, nil
}
