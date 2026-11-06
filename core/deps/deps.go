// Package deps compares two versions of a dependency manifest or lockfile and
// says which packages were added, removed or bumped. It only reads the bytes
// it's given: no registry or advisory lookup.
package deps

import (
	"encoding/json"
	"fmt"
	"path"
	"regexp"
	"slices"
	"sort"
	"strings"

	"github.com/BurntSushi/toml"
	"golang.org/x/mod/modfile"
)

type ChangeKind string

const (
	Added   ChangeKind = "added"
	Removed ChangeKind = "removed"
	Bumped  ChangeKind = "bumped"
)

type Change struct {
	Name string
	Kind ChangeKind
	From string
	To   string
}

// versions holds each package's versions. A lockfile can hold several of one
// package, and a manifest can list a package with no version.
type versions map[string][]string

var parsers = map[string]func([]byte) (versions, error){
	"package.json":        parsePackageJSON,
	"package-lock.json":   parsePackageLock,
	"npm-shrinkwrap.json": parsePackageLock,
	"go.mod":              parseGoMod,
	"go.sum":              parseGoSum,
	"Cargo.toml":          parseCargoToml,
	"Cargo.lock":          parseCargoLock,
	"pyproject.toml":      parsePyproject,
}

func parserFor(filePath string) func([]byte) (versions, error) {
	name := path.Base(filePath)
	if parser, ok := parsers[name]; ok {
		return parser
	}
	if strings.HasPrefix(name, "requirements") && strings.HasSuffix(name, ".txt") {
		return parseRequirements
	}
	return nil
}

// locks lists each manifest's possible lockfiles, by file name.
var locks = map[string][]string{
	"package.json":   {"package-lock.json", "npm-shrinkwrap.json", "yarn.lock", "pnpm-lock.yaml"},
	"go.mod":         {"go.sum"},
	"Cargo.toml":     {"Cargo.lock"},
	"pyproject.toml": {"poetry.lock", "uv.lock"},
}

// LocksOf returns the lockfile names a manifest of this name can have beside it.
func LocksOf(manifest string) []string {
	return locks[manifest]
}

// ManifestOf returns the manifest name a lockfile of this name belongs to, or "".
func ManifestOf(lock string) string {
	for manifest, names := range locks {
		if slices.Contains(names, lock) {
			return manifest
		}
	}
	return ""
}

// Recognized says whether Summarize can read a file of this name.
func Recognized(filePath string) bool {
	return parserFor(filePath) != nil
}

// Summarize compares two versions of the file, sorted by package name. An empty
// version is a file that doesn't exist on that side. A file it can't parse is
// an error, so the caller can show the raw diff.
func Summarize(filePath string, before, after []byte) ([]Change, error) {
	parser := parserFor(filePath)
	if parser == nil {
		return nil, fmt.Errorf("%s isn't a recognized manifest or lockfile", filePath)
	}
	old, err := parseUnlessEmpty(parser, before)
	if err != nil {
		return nil, err
	}
	updated, err := parseUnlessEmpty(parser, after)
	if err != nil {
		return nil, err
	}
	return compare(old, updated), nil
}

func parseUnlessEmpty(parser func([]byte) (versions, error), data []byte) (versions, error) {
	if len(strings.TrimSpace(string(data))) == 0 {
		return versions{}, nil
	}
	return parser(data)
}

func compare(old, updated versions) []Change {
	names := map[string]bool{}
	for name := range old {
		names[name] = true
	}
	for name := range updated {
		names[name] = true
	}
	changes := []Change{}
	for name := range names {
		gone := without(old[name], updated[name])
		added := without(updated[name], old[name])
		for len(gone) > 0 && len(added) > 0 {
			changes = append(changes, Change{Name: name, Kind: Bumped, From: gone[0], To: added[0]})
			gone, added = gone[1:], added[1:]
		}
		for _, version := range gone {
			changes = append(changes, Change{Name: name, Kind: Removed, From: version})
		}
		for _, version := range added {
			changes = append(changes, Change{Name: name, Kind: Added, To: version})
		}
	}
	sort.Slice(changes, func(i, j int) bool {
		if changes[i].Name != changes[j].Name {
			return changes[i].Name < changes[j].Name
		}
		return changes[i].From+changes[i].To < changes[j].From+changes[j].To
	})
	return changes
}

func without(from, remove []string) []string {
	kept := []string{}
	for _, version := range from {
		if !slices.Contains(remove, version) {
			kept = append(kept, version)
		}
	}
	sort.Strings(kept)
	return kept
}

func (v versions) add(name, version string) {
	if !slices.Contains(v[name], version) {
		v[name] = append(v[name], version)
	}
}

func parsePackageJSON(data []byte) (versions, error) {
	var manifest map[string]json.RawMessage
	if err := json.Unmarshal(data, &manifest); err != nil {
		return nil, err
	}
	found := versions{}
	for _, section := range []string{"dependencies", "devDependencies", "peerDependencies", "optionalDependencies"} {
		raw, ok := manifest[section]
		if !ok {
			continue
		}
		var entries map[string]string
		if err := json.Unmarshal(raw, &entries); err != nil {
			return nil, err
		}
		for name, version := range entries {
			found.add(name, version)
		}
	}
	return found, nil
}

type packageLockEntry struct {
	Version      string                      `json:"version"`
	Dependencies map[string]packageLockEntry `json:"dependencies"`
}

// Lockfile v2 and v3 key packages by their path in node_modules; v1 nests them.
func parsePackageLock(data []byte) (versions, error) {
	var lock struct {
		Packages     map[string]packageLockEntry `json:"packages"`
		Dependencies map[string]packageLockEntry `json:"dependencies"`
	}
	if err := json.Unmarshal(data, &lock); err != nil {
		return nil, err
	}
	found := versions{}
	if lock.Packages != nil {
		const marker = "node_modules/"
		for key, entry := range lock.Packages {
			at := strings.LastIndex(key, marker)
			if at >= 0 && entry.Version != "" {
				found.add(key[at+len(marker):], entry.Version)
			}
		}
		return found, nil
	}
	var walk func(map[string]packageLockEntry)
	walk = func(entries map[string]packageLockEntry) {
		for name, entry := range entries {
			found.add(name, entry.Version)
			walk(entry.Dependencies)
		}
	}
	walk(lock.Dependencies)
	return found, nil
}

func parseGoMod(data []byte) (versions, error) {
	// Parse, not ParseLax, which skips replace lines.
	file, err := modfile.Parse("go.mod", data, nil)
	if err != nil {
		return nil, err
	}
	found := versions{}
	for _, require := range file.Require {
		found.add(require.Mod.Path, require.Mod.Version)
	}
	for _, replace := range file.Replace {
		found.add(replace.Old.Path+" => "+replace.New.Path, replace.New.Version)
	}
	return found, nil
}

func parseGoSum(data []byte) (versions, error) {
	found := versions{}
	for number, line := range strings.Split(string(data), "\n") {
		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}
		if len(fields) != 3 || !strings.HasPrefix(fields[2], "h1:") {
			return nil, fmt.Errorf("go.sum line %d isn't a checksum", number+1)
		}
		found.add(fields[0], strings.TrimSuffix(fields[1], "/go.mod"))
	}
	return found, nil
}

func parseCargoToml(data []byte) (versions, error) {
	var manifest struct {
		Dependencies      map[string]any `toml:"dependencies"`
		DevDependencies   map[string]any `toml:"dev-dependencies"`
		BuildDependencies map[string]any `toml:"build-dependencies"`
		Workspace         struct {
			Dependencies map[string]any `toml:"dependencies"`
		} `toml:"workspace"`
		Target map[string]struct {
			Dependencies      map[string]any `toml:"dependencies"`
			DevDependencies   map[string]any `toml:"dev-dependencies"`
			BuildDependencies map[string]any `toml:"build-dependencies"`
		} `toml:"target"`
	}
	if err := toml.Unmarshal(data, &manifest); err != nil {
		return nil, err
	}
	found := versions{}
	for _, section := range []map[string]any{manifest.Dependencies, manifest.DevDependencies, manifest.BuildDependencies, manifest.Workspace.Dependencies} {
		addTomlDependencies(found, section)
	}
	for _, target := range manifest.Target {
		addTomlDependencies(found, target.Dependencies)
		addTomlDependencies(found, target.DevDependencies)
		addTomlDependencies(found, target.BuildDependencies)
	}
	return found, nil
}

// A dependency is a version string, or a table that may hold one.
func addTomlDependencies(found versions, section map[string]any) {
	for name, value := range section {
		switch dependency := value.(type) {
		case string:
			found.add(name, dependency)
		case map[string]any:
			version, _ := dependency["version"].(string)
			found.add(name, version)
		}
	}
}

func parseCargoLock(data []byte) (versions, error) {
	var lock struct {
		Package []struct {
			Name    string `toml:"name"`
			Version string `toml:"version"`
		} `toml:"package"`
	}
	if err := toml.Unmarshal(data, &lock); err != nil {
		return nil, err
	}
	found := versions{}
	for _, entry := range lock.Package {
		found.add(entry.Name, entry.Version)
	}
	return found, nil
}

func parsePyproject(data []byte) (versions, error) {
	var project struct {
		Project struct {
			Dependencies         []string            `toml:"dependencies"`
			OptionalDependencies map[string][]string `toml:"optional-dependencies"`
		} `toml:"project"`
		DependencyGroups map[string][]any `toml:"dependency-groups"`
		Tool             struct {
			Poetry struct {
				Dependencies    map[string]any `toml:"dependencies"`
				DevDependencies map[string]any `toml:"dev-dependencies"`
				Group           map[string]struct {
					Dependencies map[string]any `toml:"dependencies"`
				} `toml:"group"`
			} `toml:"poetry"`
		} `toml:"tool"`
	}
	if err := toml.Unmarshal(data, &project); err != nil {
		return nil, err
	}
	found := versions{}
	lists := [][]string{project.Project.Dependencies}
	for _, list := range project.Project.OptionalDependencies {
		lists = append(lists, list)
	}
	for _, group := range project.DependencyGroups {
		var list []string
		for _, entry := range group {
			// The other form is a table including another group.
			if requirement, ok := entry.(string); ok {
				list = append(list, requirement)
			}
		}
		lists = append(lists, list)
	}
	for _, list := range lists {
		for _, requirement := range list {
			if err := addRequirement(found, requirement); err != nil {
				return nil, err
			}
		}
	}
	poetry := project.Tool.Poetry
	addTomlDependencies(found, poetry.Dependencies)
	addTomlDependencies(found, poetry.DevDependencies)
	for _, group := range poetry.Group {
		addTomlDependencies(found, group.Dependencies)
	}
	delete(found, "python")
	return found, nil
}

func parseRequirements(data []byte) (versions, error) {
	found := versions{}
	text := strings.ReplaceAll(strings.ReplaceAll(string(data), "\r\n", "\n"), "\\\n", " ")
	for _, line := range strings.Split(text, "\n") {
		if at := strings.Index(line, " #"); at >= 0 {
			line = line[:at]
		}
		line = strings.TrimSpace(line)
		// Options and includes aren't packages; #egg= names a package by URL, with no version to compare.
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "-") || strings.Contains(line, "://") {
			continue
		}
		if err := addRequirement(found, line); err != nil {
			return nil, err
		}
	}
	return found, nil
}

var (
	requirementPattern = regexp.MustCompile(`^([A-Za-z0-9][A-Za-z0-9._-]*)\s*(\[[^\]]*\])?\s*(.*)$`)
	nameSeparators     = regexp.MustCompile(`[-_.]+`)
)

// addRequirement reads a PEP 508 requirement. The specifier is kept as written,
// with any environment marker and hash options left out.
func addRequirement(found versions, requirement string) error {
	match := requirementPattern.FindStringSubmatch(strings.TrimSpace(requirement))
	if match == nil {
		return fmt.Errorf("%q isn't a package requirement", requirement)
	}
	specifier, _, _ := strings.Cut(match[3], ";")
	specifier, _, _ = strings.Cut(specifier, " --")
	specifier = strings.Trim(strings.TrimSpace(specifier), "()")
	if specifier != "" && !strings.ContainsAny(specifier[:1], "=<>!~@") {
		return fmt.Errorf("%q isn't a package requirement", requirement)
	}
	found.add(strings.ToLower(nameSeparators.ReplaceAllString(match[1], "-")), strings.ReplaceAll(specifier, " ", ""))
	return nil
}
