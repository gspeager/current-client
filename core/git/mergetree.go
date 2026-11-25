package git

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/gspeager/current-client/core/gitexec"
)

// ErrMergeTreeUnsupported means Git is older than 2.38, which added
// merge-tree --write-tree.
var ErrMergeTreeUnsupported = errors.New("predicting conflicts needs Git 2.38 or newer")

// ErrMergeTreeStdinUnsupported means Git is older than 2.40, which added
// merge-tree --stdin; MergeConflicts still works one merge at a time.
var ErrMergeTreeStdinUnsupported = errors.New("merging in one batch needs Git 2.40 or newer")

// ErrUnrelatedHistories means a commit shares no history with the other, such
// as an orphan gh-pages branch, so merge-tree refuses to merge them.
var ErrUnrelatedHistories = errors.New("these commits share no history")

// mergeTreeError names the failures callers act on.
func mergeTreeError(result gitexec.Result) error {
	switch {
	case result.ExitCode == 129:
		return ErrMergeTreeUnsupported
	case strings.Contains(result.Stderr, "unrelated histories"):
		return ErrUnrelatedHistories
	}
	return gitexec.WrapResult(result)
}

// MergeResult is a merge of two commits in memory: the tree it would make,
// and the files that would conflict when it isn't clean. Unrelated means the
// commits share no history, so there is no merge.
type MergeResult struct {
	Tree      string
	Clean     bool
	Files     []string
	Unrelated bool
}

// MergeConflicts merges two commits in memory, without touching the working
// tree or index, and returns the files that would conflict.
func MergeConflicts(ctx context.Context, repoPath, ours, theirs string) ([]string, error) {
	env, discard := ScratchObjects(ctx, repoPath)
	defer discard()
	merge, err := mergeTree(ctx, repoPath, env, ours, theirs)
	return merge.Files, err
}

// MergeInMemory merges two commits where env says, without following renames.
// The result's tree is among env's objects.
func MergeInMemory(ctx context.Context, repoPath string, env []string, ours, theirs string) (MergeResult, error) {
	env = append([]string{"GIT_CONFIG_COUNT=1", "GIT_CONFIG_KEY_0=merge.renames", "GIT_CONFIG_VALUE_0=false"}, env...)
	return mergeTree(ctx, repoPath, env, ours, theirs)
}

func mergeTree(ctx context.Context, repoPath string, env []string, args ...string) (MergeResult, error) {
	result, err := gitexec.NewExecutor("").Run(ctx, gitexec.Command{
		Dir:  repoPath,
		Env:  env,
		Args: append([]string{"merge-tree", "--write-tree", "--name-only", "--no-messages", "-z"}, args...),
	})
	if err != nil {
		return MergeResult{}, gitexec.WrapRunError(err)
	}
	// "<tree>\0<file>\0<file>\0"
	fields := strings.Split(strings.TrimRight(result.Stdout, "\x00\n"), "\x00")
	switch result.ExitCode {
	case 0:
		return MergeResult{Tree: fields[0], Clean: true}, nil
	case 1:
		return MergeResult{Tree: fields[0], Files: fields[1:]}, nil
	}
	return MergeResult{}, mergeTreeError(result)
}

// MergeConflictsMany merges ours with each of theirs in one process, as
// MergeConflicts does, and returns their conflicting files in the same order.
func MergeConflictsMany(ctx context.Context, repoPath, ours string, theirs []string) ([][]string, error) {
	pairs := make([][2]string, len(theirs))
	for i, tip := range theirs {
		pairs[i] = [2]string{ours, tip}
	}
	return MergeConflictsBatch(ctx, repoPath, pairs)
}

// MergeConflictsBatch is MergeConflictsPairs in one process or not at all: it
// returns ErrMergeTreeStdinUnsupported on Git before 2.40, and
// ErrUnrelatedHistories when a pair shares no history.
func MergeConflictsBatch(ctx context.Context, repoPath string, pairs [][2]string) ([][]string, error) {
	env, discard := ScratchObjects(ctx, repoPath)
	defer discard()
	merges, err := mergeTreeBatch(ctx, repoPath, env, pairs)
	if err != nil {
		return nil, err
	}
	return conflictsOf(merges), nil
}

// MergePairs merges each pair of commits in memory and answers in the same
// order: one process on Git 2.40 or newer. A pair sharing no history is
// Unrelated; it costs the batch its one process, as Git before 2.40 does.
func MergePairs(ctx context.Context, repoPath string, pairs [][2]string) ([]MergeResult, error) {
	return mergePairs(ctx, repoPath, nil, pairs)
}

// mergePairsDiscarding is MergePairs for a caller that only compares the
// trees: they aren't kept.
func mergePairsDiscarding(ctx context.Context, repoPath string, pairs [][2]string) ([]MergeResult, error) {
	env, discard := ScratchObjects(ctx, repoPath)
	defer discard()
	return mergePairs(ctx, repoPath, env, pairs)
}

func mergePairs(ctx context.Context, repoPath string, env []string, pairs [][2]string) ([]MergeResult, error) {
	if len(pairs) == 0 {
		return nil, nil
	}
	merges, err := mergeTreeBatch(ctx, repoPath, env, pairs)
	if errors.Is(err, ErrMergeTreeStdinUnsupported) || errors.Is(err, ErrUnrelatedHistories) {
		return mergePairsEach(ctx, repoPath, env, pairs)
	}
	return merges, err
}

// ScratchObjects is an environment in which Git writes its objects to a folder
// that discard removes. Merging every branch of a large repository in memory
// writes hundreds of megabytes that Git would otherwise keep until it prunes
// them. A partial clone keeps them, since what it fetches for the merge would be
// thrown away too.
func ScratchObjects(ctx context.Context, repoPath string) (env []string, discard func()) {
	discard = func() {}
	objects, err := objectsDir(ctx, repoPath)
	if err != nil {
		return nil, discard
	}
	if hasPromisorPack(objects) {
		return nil, discard
	}
	return scratchBeside(objects)
}

// ScratchWrites is ScratchObjects for a caller that fetches nothing, so a partial
// clone has nothing to keep.
func ScratchWrites(ctx context.Context, repoPath string) (env []string, discard func()) {
	objects, err := objectsDir(ctx, repoPath)
	if err != nil {
		return nil, func() {}
	}
	return scratchBeside(objects)
}

func scratchBeside(objects string) (env []string, discard func()) {
	discard = func() {}
	sweepScratch()
	dir, err := os.MkdirTemp("", scratchPrefix)
	if err != nil {
		return nil, discard
	}
	// Quoted, so a path that holds the list separator is still one path. Git
	// doesn't read the \u escapes strconv.Quote writes.
	alternates := `"` + strings.NewReplacer(`\`, `\\`, `"`, `\"`).Replace(filepath.ToSlash(objects)) + `"`
	if inherited := os.Getenv("GIT_ALTERNATE_OBJECT_DIRECTORIES"); inherited != "" {
		alternates += string(os.PathListSeparator) + inherited
	}
	return []string{"GIT_OBJECT_DIRECTORY=" + dir, "GIT_ALTERNATE_OBJECT_DIRECTORIES=" + alternates},
		func() { _ = os.RemoveAll(dir) }
}

// Git marks each pack a partial clone fetched with a .promisor file. One made with
// --reference can have all of its packs in another's folder.
func hasPromisorPack(objects string) bool {
	return promisorPackWithin(objects, 5)
}

// As many alternates of alternates as Git follows.
func promisorPackWithin(objects string, depth int) bool {
	packs, _ := os.ReadDir(filepath.Join(objects, "pack"))
	if slices.ContainsFunc(packs, func(pack os.DirEntry) bool { return strings.HasSuffix(pack.Name(), ".promisor") }) {
		return true
	}
	alternates, _ := os.ReadFile(filepath.Join(objects, "info", "alternates"))
	for line := range strings.Lines(string(alternates)) {
		alternate := strings.TrimSpace(line)
		if alternate == "" || strings.HasPrefix(alternate, "#") || depth == 0 {
			continue
		}
		if !filepath.IsAbs(alternate) {
			alternate = filepath.Join(objects, alternate)
		}
		if promisorPackWithin(alternate, depth-1) {
			return true
		}
	}
	return false
}

// PartialClone says whether reading a file may fetch it.
func PartialClone(ctx context.Context, repoPath string) bool {
	objects, err := objectsDir(ctx, repoPath)
	return err == nil && hasPromisorPack(objects)
}

func objectsDir(ctx context.Context, repoPath string) (string, error) {
	if objects := os.Getenv("GIT_OBJECT_DIRECTORY"); objects != "" {
		return objects, nil
	}
	commonDir, err := CommonDir(ctx, repoPath)
	return filepath.Join(commonDir, "objects"), err
}

const scratchPrefix = "current-client-objects"

// Quitting during a merge leaves its folder behind.
var sweepScratch = sync.OnceFunc(func() { removeScratchOlderThan(24 * time.Hour) })

func removeScratchOlderThan(age time.Duration) {
	dirs, _ := filepath.Glob(filepath.Join(os.TempDir(), scratchPrefix+"*"))
	for _, dir := range dirs {
		if info, err := os.Lstat(dir); err == nil && time.Since(info.ModTime()) > age {
			_ = os.RemoveAll(dir)
		}
	}
}

// MergeConflictsPairs is MergePairs reduced to each pair's conflicting
// files: nil for a clean merge, and for a pair sharing no history.
func MergeConflictsPairs(ctx context.Context, repoPath string, pairs [][2]string) ([][]string, error) {
	env, discard := ScratchObjects(ctx, repoPath)
	defer discard()
	return MergeConflictsIn(ctx, repoPath, env, pairs)
}

// MergeConflictsIn is MergeConflictsPairs where env says, such as ScratchObjects'.
func MergeConflictsIn(ctx context.Context, repoPath string, env []string, pairs [][2]string) ([][]string, error) {
	merges, err := mergePairs(ctx, repoPath, env, pairs)
	if err != nil {
		return nil, err
	}
	return conflictsOf(merges), nil
}

func conflictsOf(merges []MergeResult) [][]string {
	conflicts := make([][]string, len(merges))
	for i, merge := range merges {
		conflicts[i] = merge.Files
	}
	return conflicts
}

func mergePairsEach(ctx context.Context, repoPath string, env []string, pairs [][2]string) ([]MergeResult, error) {
	merges := make([]MergeResult, len(pairs))
	for i, pair := range pairs {
		merge, err := mergeTree(ctx, repoPath, env, pair[0], pair[1])
		if errors.Is(err, ErrUnrelatedHistories) {
			merge = MergeResult{Unrelated: true}
		} else if err != nil {
			return nil, err
		}
		merges[i] = merge
	}
	return merges, nil
}

func mergeTreeBatch(ctx context.Context, repoPath string, env []string, pairs [][2]string) ([]MergeResult, error) {
	var input strings.Builder
	for _, pair := range pairs {
		fmt.Fprintf(&input, "%s %s\n", pair[0], pair[1])
	}
	result, err := gitexec.NewExecutor("").Run(ctx, gitexec.Command{
		Dir:   repoPath,
		Env:   env,
		Args:  []string{"merge-tree", "--write-tree", "--stdin", "--name-only", "--no-messages", "-z"},
		Stdin: strings.NewReader(input.String()),
	})
	if err != nil {
		return nil, gitexec.WrapRunError(err)
	}
	if result.ExitCode == 129 {
		return nil, ErrMergeTreeStdinUnsupported
	}
	if result.ExitCode != 0 {
		return nil, mergeTreeError(result)
	}
	return parseMergeTreeStdin(result.Stdout, len(pairs))
}

// Each merge is "<1 clean, 0 conflicted>\0<tree>\0<file>\0…\0".
func parseMergeTreeStdin(output string, merges int) ([]MergeResult, error) {
	fields := strings.Split(output, "\x00")
	results := make([]MergeResult, 0, merges)
	for i := 0; len(results) < merges; {
		if i+1 >= len(fields) {
			return nil, fmt.Errorf("merge-tree answered %d of %d merges", len(results), merges)
		}
		merge := MergeResult{Clean: fields[i] == "1", Tree: fields[i+1]}
		i += 2
		for ; i < len(fields) && fields[i] != ""; i++ {
			merge.Files = append(merge.Files, fields[i])
		}
		i++
		if merge.Clean {
			merge.Files = nil
		}
		results = append(results, merge)
	}
	return results, nil
}

// SharingHistory returns the commits of the branches under namespaces
// (refs/heads and refs/remotes by default) that share history with ours:
// those that contain one of its root commits.
func SharingHistory(ctx context.Context, repoPath, ours string, namespaces ...string) (map[string]bool, error) {
	if len(namespaces) == 0 {
		namespaces = []string{"refs/heads", "refs/remotes"}
	}
	roots, err := runResult(ctx, repoPath, "rev-list", "--max-parents=0", ours)
	if err != nil {
		return nil, err
	}
	args := []string{"for-each-ref", "--format=%(objectname)"}
	for _, root := range strings.Fields(roots.Stdout) {
		args = append(args, "--contains="+root)
	}
	containing, err := runResult(ctx, repoPath, append(args, namespaces...)...)
	if err != nil {
		return nil, err
	}
	related := map[string]bool{}
	for _, sha := range strings.Fields(containing.Stdout) {
		related[sha] = true
	}
	return related, nil
}

// SparseCheckout says whether only some of HEAD's files are checked out.
func SparseCheckout(ctx context.Context, repoPath string) bool {
	result, err := gitexec.NewExecutor("").Run(ctx, gitexec.Command{Dir: repoPath, Args: []string{"config", "--bool", "core.sparseCheckout"}})
	return err == nil && strings.TrimSpace(result.Stdout) == "true"
}
