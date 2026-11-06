// Package tasks finds a project's own commands and runs them as local processes.
// Finding only reads files: nothing from a repository runs until the user has seen
// and approved the command.
package tasks

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

const (
	SourceTaskfile = "Taskfile"
	SourceMake     = "Makefile"
	SourceNpm      = "package.json"
	SourceGo       = "go.mod"
	SourceCustom   = "Custom"
)

type Task struct {
	ID      string
	Source  string
	Name    string
	Command string
	// File is what the command comes from, relative to the repository.
	File string
	// Note is what approving should know beyond the command itself.
	Note string
}

type Custom struct {
	Name    string `json:"name"`
	Command string `json:"command"`
}

// Discover reads the repository's root for commands. A file that can't be read
// or parsed just offers nothing.
func Discover(repoPath string, custom []Custom) []Task {
	var found []Task
	found = append(found, taskfileTasks(repoPath)...)
	found = append(found, makeTasks(repoPath)...)
	found = append(found, npmTasks(repoPath)...)
	found = append(found, goTasks(repoPath)...)
	for _, item := range custom {
		found = append(found, Task{ID: "custom:" + item.Name, Source: SourceCustom, Name: item.Name, Command: item.Command})
	}
	return found
}

func Find(repoPath string, custom []Custom, id string) (Task, bool) {
	for _, task := range Discover(repoPath, custom) {
		if task.ID == id {
			return task, true
		}
	}
	return Task{}, false
}

// Fingerprint changes when what would run might have. It covers the whole source
// file, since `make test` reads the same after a pull changes its recipe.
func Fingerprint(repoPath string, task Task) string {
	hash := sha256.New()
	hash.Write([]byte(task.Command + "\x00"))
	if task.File != "" {
		if data, err := os.ReadFile(filepath.Join(repoPath, task.File)); err == nil {
			hash.Write(data)
		}
	}
	return hex.EncodeToString(hash.Sum(nil))
}

var (
	taskfileKey = regexp.MustCompile(`^(\s+)([A-Za-z0-9_][A-Za-z0-9_:.-]*)\s*:`)
	makeTarget  = regexp.MustCompile(`^([A-Za-z0-9_][A-Za-z0-9_.-]*)\s*:([^=]|$)`)
)

func taskfileTasks(repoPath string) []Task {
	for _, name := range []string{"Taskfile.yml", "Taskfile.yaml", "taskfile.yml", "taskfile.yaml"} {
		file, err := os.Open(filepath.Join(repoPath, name))
		if err != nil {
			continue
		}
		defer file.Close()
		var found []Task
		inTasks, indent := false, ""
		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			line := scanner.Text()
			if strings.TrimSpace(line) == "" || strings.HasPrefix(strings.TrimSpace(line), "#") {
				continue
			}
			if !strings.HasPrefix(line, " ") && !strings.HasPrefix(line, "\t") {
				inTasks = strings.HasPrefix(line, "tasks:")
				continue
			}
			match := taskfileKey.FindStringSubmatch(line)
			if !inTasks || match == nil {
				continue
			}
			if indent == "" {
				indent = match[1]
			}
			if match[1] == indent {
				found = append(found, Task{ID: "task:" + match[2], Source: SourceTaskfile, Name: match[2], Command: "task " + match[2], File: name})
			}
		}
		return found
	}
	return nil
}

func makeTasks(repoPath string) []Task {
	for _, name := range []string{"Makefile", "makefile", "GNUmakefile"} {
		data, err := os.ReadFile(filepath.Join(repoPath, name))
		if err != nil {
			continue
		}
		seen := map[string]bool{}
		var found []Task
		for _, line := range strings.Split(string(data), "\n") {
			match := makeTarget.FindStringSubmatch(line)
			if match == nil || seen[match[1]] {
				continue
			}
			seen[match[1]] = true
			found = append(found, Task{ID: "make:" + match[1], Source: SourceMake, Name: match[1], Command: "make " + match[1], File: name})
		}
		return found
	}
	return nil
}

func npmTasks(repoPath string) []Task {
	data, err := os.ReadFile(filepath.Join(repoPath, "package.json"))
	if err != nil {
		return nil
	}
	var manifest struct {
		Scripts map[string]string `json:"scripts"`
	}
	if json.Unmarshal(data, &manifest) != nil {
		return nil
	}
	runner := "npm run "
	for lock, command := range map[string]string{"pnpm-lock.yaml": "pnpm run ", "yarn.lock": "yarn "} {
		if _, err := os.Stat(filepath.Join(repoPath, lock)); err == nil {
			runner = command
		}
	}
	names := make([]string, 0, len(manifest.Scripts))
	for name := range manifest.Scripts {
		names = append(names, name)
	}
	sort.Strings(names)
	found := make([]Task, 0, len(names))
	for _, name := range names {
		found = append(found, Task{ID: "npm:" + name, Source: SourceNpm, Name: name, Command: runner + name, File: "package.json", Note: manifest.Scripts[name]})
	}
	return found
}

func goTasks(repoPath string) []Task {
	if _, err := os.Stat(filepath.Join(repoPath, "go.mod")); err != nil {
		return nil
	}
	return []Task{{
		ID:      "go:test",
		Source:  SourceGo,
		Name:    "test",
		Command: "go test ./...",
		File:    "go.mod",
		Note:    "This runs the repository's own code, which can change with any commit.",
	}}
}
