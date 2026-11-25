package structure

import (
	"encoding/json"
	"path"
	"regexp"
	"slices"
	"strings"
)

// configReaders are for files that say something about a folder: its modules and
// aliases, or what it's for.
func configReaders(file string) func(src string) *facts {
	base := path.Base(file)
	switch {
	case base == "go.mod":
		return readGoMod
	case base == "package.json":
		return readPackageJSON
	case tsconfigName.MatchString(base):
		return readTSConfig
	case base == "Package.swift":
		return func(string) *facts { return &facts{} }
	case slices.Contains([]string{"readme.md", "readme", "readme.txt", "readme.rst"}, strings.ToLower(base)):
		return func(src string) *facts { return &facts{doc: readmeDoc(src)} }
	}
	return nil
}

var (
	tsconfigName = regexp.MustCompile(`^(ts|js)config[\w.-]*\.json$`)
	goModule     = regexp.MustCompile(`(?m)^module\s+"?([^\s"]+)`)
	jsonComment  = regexp.MustCompile(`(?m)("(?:\\.|[^"\\\n])*")|//[^\n]*|/\*(?s:.*?)\*/`)
	jsonTrailing = regexp.MustCompile(`,(\s*[}\]])`)
)

func readGoMod(src string) *facts {
	f := &facts{}
	if match := goModule.FindStringSubmatch(src); match != nil {
		f.module = intern(match[1])
	}
	return f
}

func readPackageJSON(src string) *facts {
	var manifest struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		Source      string `json:"source"`
	}
	if json.Unmarshal([]byte(src), &manifest) != nil {
		return &facts{}
	}
	f := &facts{module: manifest.Name, doc: firstSentence(manifest.Description)}
	if manifest.Source != "" {
		f.base = path.Dir(manifest.Source)
	}
	return f
}

// tsconfig files have comments and trailing commas.
func readTSConfig(src string) *facts {
	src = jsonTrailing.ReplaceAllString(jsonComment.ReplaceAllString(src, "$1"), "$1")
	var config struct {
		CompilerOptions struct {
			BaseURL string              `json:"baseUrl"`
			Paths   map[string][]string `json:"paths"`
		} `json:"compilerOptions"`
	}
	if json.Unmarshal([]byte(src), &config) != nil {
		return &facts{}
	}
	f := &facts{base: config.CompilerOptions.BaseURL}
	for from, to := range config.CompilerOptions.Paths {
		a := alias{from: strings.TrimSuffix(strings.TrimSuffix(from, "*"), "/")}
		for _, target := range to {
			a.to = append(a.to, strings.TrimSuffix(strings.TrimSuffix(target, "*"), "/"))
		}
		f.aliases = append(f.aliases, a)
	}
	return f
}

// The first line of prose: not a heading, a badge, a list or markup.
func readmeDoc(src string) string {
	var paragraph []string
	for line := range strings.Lines(src) {
		line = strings.TrimSpace(line)
		prose := line != "" && !strings.ContainsAny(line[:1], "#<[!|>=-*_`+") && !strings.HasPrefix(line, "..")
		if prose {
			paragraph = append(paragraph, line)
		} else if len(paragraph) > 0 {
			break
		}
	}
	return firstSentence(markdownLink.ReplaceAllString(strings.Join(paragraph, " "), "$1"))
}

var markdownLink = regexp.MustCompile(`\[([^\]]*)\]\([^)]*\)`)

const longestDoc = 160

func firstSentence(text string) string {
	text = strings.Join(strings.Fields(strings.NewReplacer("**", "", "`", "").Replace(text)), " ")
	if sentence, _, ok := strings.Cut(text, ". "); ok {
		text = sentence
	}
	text = strings.TrimSuffix(text, ".")
	if runes := []rune(text); len(runes) > longestDoc {
		text = strings.TrimSpace(string(runes[:longestDoc])) + "…"
	}
	return text
}

// describe takes a folder's words from its Go package comment, then its package.json,
// then its README.
func (l *linker) describe(docs map[string]string) {
	rank := map[string]int{}
	say := func(file, doc string, as int) {
		dir := path.Dir(file)
		if dir == "." {
			dir = ""
		}
		if doc != "" && as > rank[dir] {
			docs[dir], rank[dir] = doc, as
		}
	}
	for i, file := range l.files {
		f := l.facts[i]
		switch {
		case f == nil:
		case path.Ext(file) == ".go":
			say(file, f.doc, 3)
		case path.Base(file) == "package.json":
			say(file, f.doc, 2)
		default:
			say(file, f.doc, 1)
		}
	}
}
