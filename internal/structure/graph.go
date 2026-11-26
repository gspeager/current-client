// Package structure reads what each file of a repository uses, by its language's
// imports or by the names it defines, and what the parts of a folder use of each
// other. It completes, edits and explains nothing.
package structure

import (
	"bytes"
	"cmp"
	"context"
	"path"
	"runtime"
	"slices"
	"strings"
	"sync"
	"unique"

	"github.com/gspeager/current-client/core/filekind"
	"github.com/gspeager/current-client/core/git"
)

// Graph is a repository's files at one point, and which use which.
type Graph struct {
	// Sorted by path.
	Files []string
	// Indexes into Files, each pair once: the first file uses the second.
	Links [][2]int32
	// The languages read, the one with the most files first.
	Languages []string
	// Files of code in a language that isn't read.
	Unread int
	// A folder's description from its package comment, package.json or README.
	Docs map[string]string
	// Every folder that holds a file, of any kind.
	Folders map[string]bool
	// Line counts of the files read, for a partial clone, where Git can't count
	// every file without fetching it.
	Lines map[string]int
}

// facts are what one file says without the others, so they're kept by its content.
type facts struct {
	imports []imported
	// Names other files can use.
	defines []string
	// Names used that aren't the file's own.
	uses []string
	// Go's "package.Name".
	selected []string
	// A Go package, a C# namespace.
	scope string
	// The namespaces or modules named by using or import, for languages that don't
	// name files.
	sees  []string
	doc   string
	lines int
	// A go.mod's module or a package.json's name.
	module string
	// A tsconfig's paths.
	aliases []alias
	// A tsconfig's baseUrl or the folder of a package.json's source.
	base string
}

type imported struct {
	path string
	// The name it's used by, where that isn't the path's own.
	alias string
	// What's taken from it, in Python.
	names []string
	// How many folders up a relative Python import starts.
	up int
}

type language struct {
	name string
	read func(src string) *facts
	link func(l *linker, files []int32)
}

var languages = map[string]*language{}

func register(lang *language, extensions ...string) {
	for _, ext := range extensions {
		languages[ext] = lang
	}
}

func languageOf(file string) *language {
	if strings.HasSuffix(file, ".d.ts") {
		return nil
	}
	return languages[path.Ext(file)]
}

// Extensions of code in languages that aren't read, so the view can say so.
var unreadCode = map[string]bool{
	".java": true, ".kt": true, ".kts": true, ".scala": true, ".rs": true, ".c": true, ".h": true, ".cc": true, ".cpp": true,
	".hpp": true, ".cxx": true, ".m": true, ".mm": true, ".php": true, ".rb": true, ".dart": true, ".ex": true, ".exs": true,
	".erl": true, ".hs": true, ".ml": true, ".clj": true, ".fs": true, ".vb": true, ".lua": true, ".pl": true, ".r": true,
	".jl": true, ".zig": true, ".nim": true, ".vue": true, ".svelte": true, ".groovy": true, ".elm": true,
}

type linker struct {
	files []string
	facts []*facts
	index map[string]int32
	links map[[2]int32]bool
	// Files that describe modules and aliases, by their own path.
	config map[string]*facts
}

func (l *linker) add(from, to int32) {
	if from != to {
		l.links[[2]int32{from, to}] = true
	}
}

func (l *linker) find(paths ...string) (int32, bool) {
	for _, p := range paths {
		if i, ok := l.index[p]; ok {
			return i, true
		}
	}
	return 0, false
}

const (
	// Blobs asked of one git process, so the content held at once stays small.
	filesAtOnce = 300
	// Bytes; a larger file is taken for data, not code, and not read.
	largestRead = 2 << 20
	// Files' facts kept across builds. Past this the cache starts over, so it stays
	// bounded however many points are read.
	mostFacts = 120_000
)

var kept = struct {
	sync.Mutex
	facts map[string]*facts
}{facts: map[string]*facts{}}

// Build reads the files of a commit or tree that aren't tests, generated or
// lockfiles. What a file says is kept by its content, so another point that shares
// most of its files reads only the rest.
func Build(ctx context.Context, repoPath string, env []string, rev string) (*Graph, error) {
	files, err := git.TreeFiles(ctx, repoPath, env, rev)
	if err != nil {
		return nil, err
	}
	folders := map[string]bool{}
	for _, f := range files {
		for dir := folderOf(f.Path); dir != "" && !folders[dir]; dir = folderOf(dir) {
			folders[dir] = true
		}
	}
	files = slices.DeleteFunc(files, func(f git.TreeFile) bool { return filekind.Of(f.Path) != filekind.Code })
	known, err := readFacts(ctx, repoPath, env, files)
	if err != nil {
		return nil, err
	}
	l := &linker{index: make(map[string]int32, len(files)), links: map[[2]int32]bool{}, config: map[string]*facts{}}
	byLanguage := map[*language][]int32{}
	graph := &Graph{Docs: map[string]string{}, Lines: map[string]int{}, Folders: folders}
	for i, f := range files {
		l.files = append(l.files, f.Path)
		l.facts = append(l.facts, known[i])
		l.index[f.Path] = int32(i)
		if known[i] != nil {
			graph.Lines[f.Path] = known[i].lines
		}
		switch lang := languageOf(f.Path); {
		case lang != nil:
			byLanguage[lang] = append(byLanguage[lang], int32(i))
		case known[i] != nil:
			l.config[f.Path] = known[i]
		case unreadCode[path.Ext(f.Path)]:
			graph.Unread++
		}
	}
	for lang, indexes := range byLanguage {
		lang.link(l, indexes)
		graph.Languages = append(graph.Languages, lang.name)
	}
	count := map[string]int{}
	for lang, indexes := range byLanguage {
		count[lang.name] = len(indexes)
	}
	slices.SortFunc(graph.Languages, func(a, b string) int { return cmp.Or(count[b]-count[a], strings.Compare(a, b)) })
	graph.Files = l.files
	for link := range l.links {
		graph.Links = append(graph.Links, link)
	}
	slices.SortFunc(graph.Links, func(a, b [2]int32) int { return slices.Compare(a[:], b[:]) })
	l.describe(graph.Docs)
	return graph, nil
}

// readerFor is nil for a file that says nothing about the structure.
func readerFor(file string) func(src string) *facts {
	if lang := languageOf(file); lang != nil {
		return lang.read
	}
	return configReaders(file)
}

func readFacts(ctx context.Context, repoPath string, env []string, files []git.TreeFile) ([]*facts, error) {
	known := make([]*facts, len(files))
	key := func(i int) string {
		if languageOf(files[i].Path) != nil {
			return files[i].Blob + path.Ext(files[i].Path)
		}
		return files[i].Blob + path.Base(files[i].Path)
	}
	var missing []int
	kept.Lock()
	for i, f := range files {
		if readerFor(f.Path) == nil {
			continue
		}
		if known[i] = kept.facts[key(i)]; known[i] == nil {
			missing = append(missing, i)
		}
	}
	kept.Unlock()

	// What a partial clone left out is left unread. Git before 2.45 would fetch it.
	env = append(slices.Clone(env), "GIT_NO_LAZY_FETCH=1")
	var mu sync.Mutex
	var failed error
	slots := make(chan struct{}, max(2, runtime.NumCPU()/2))
	var wg sync.WaitGroup
	for batch := range slices.Chunk(missing, filesAtOnce) {
		slots <- struct{}{}
		wg.Go(func() {
			defer func() { <-slots }()
			names := make([]string, len(batch))
			for k, i := range batch {
				names[k] = files[i].Blob
			}
			blobs, err := git.Blobs(ctx, repoPath, env, names)
			if err != nil {
				mu.Lock()
				failed = err
				mu.Unlock()
				return
			}
			for k, i := range batch {
				known[i] = &facts{}
				if blob := blobs[k]; len(blob) <= largestRead {
					known[i] = readerFor(files[i].Path)(string(blob))
					known[i].lines = bytes.Count(blob, []byte("\n"))
				}
			}
		})
	}
	wg.Wait()
	if failed != nil {
		return nil, failed
	}
	kept.Lock()
	defer kept.Unlock()
	if len(kept.facts)+len(missing) > mostFacts {
		clear(kept.facts)
	}
	for _, i := range missing {
		kept.facts[key(i)] = known[i]
	}
	return known, nil
}

// sorted interns the names, copying them out of the source they were sliced from, so
// keeping a file's facts doesn't keep its content.
func sorted(names map[string]bool) []string {
	list := make([]string, 0, len(names))
	for name := range names {
		list = append(list, intern(name))
	}
	slices.Sort(list)
	return list
}

func intern(name string) string {
	return unique.Make(name).Value()
}
