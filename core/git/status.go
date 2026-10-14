package git

import (
	"context"
	"fmt"
	"strings"
)

type Status struct {
	Path           string
	OrigPath       string
	IndexStatus    byte
	WorktreeStatus byte
	Conflicted     bool
	Submodule      *SubmoduleStatus
}

// SubmoduleStatus is what changed inside a submodule, from porcelain v2's
// "S<c><m><u>" field.
type SubmoduleStatus struct {
	CommitChanged bool
	Modified      bool
	Untracked     bool
}

func parseSubmoduleField(field string) *SubmoduleStatus {
	if len(field) != 4 || field[0] != 'S' {
		return nil
	}
	return &SubmoduleStatus{CommitChanged: field[1] == 'C', Modified: field[2] == 'M', Untracked: field[3] == 'U'}
}

// GetStatus lists each untracked file, rather than one entry for a new folder.
func GetStatus(ctx context.Context, repoPath string) ([]Status, error) {
	result, err := runResult(ctx, repoPath, "status", "--porcelain=v2", "-z", "--untracked-files=all")
	if err != nil {
		return nil, err
	}
	return ParseStatus(result.Stdout)
}

// StatusSummary is the current branch and how many files changed, from one
// git process.
type StatusSummary struct {
	// Branch is "HEAD" when HEAD is detached.
	Branch   string
	Upstream string
	Ahead    int
	Behind   int
	Changes  int
}

func SummarizeStatus(ctx context.Context, repoPath string) (StatusSummary, error) {
	result, err := runResult(ctx, repoPath, "status", "--porcelain=v2", "--branch", "-z", "--untracked-files=all")
	if err != nil {
		return StatusSummary{}, err
	}
	statuses, err := ParseStatus(result.Stdout)
	if err != nil {
		return StatusSummary{}, err
	}
	summary := StatusSummary{Changes: len(statuses)}
	for _, record := range strings.Split(result.Stdout, "\x00") {
		key, value, _ := strings.Cut(strings.TrimPrefix(record, "# "), " ")
		switch key {
		case "branch.head":
			summary.Branch = value
			if value == "(detached)" {
				summary.Branch = "HEAD"
			}
		case "branch.upstream":
			summary.Upstream = value
		case "branch.ab":
			_, _ = fmt.Sscanf(value, "+%d -%d", &summary.Ahead, &summary.Behind)
		}
	}
	return summary, nil
}

// ParseStatus skips the "# branch.…" headers that --branch adds.
func ParseStatus(output string) ([]Status, error) {
	records := strings.Split(output, "\x00")
	var statuses []Status

	for i := 0; i < len(records); i++ {
		record := records[i]
		if record == "" || record[0] == '#' {
			continue
		}

		switch record[0] {
		case '1':
			fields, err := splitStatusFields(record, 9)
			if err != nil {
				return nil, err
			}
			statuses = append(statuses, Status{
				Path:           fields[8],
				IndexStatus:    fields[1][0],
				WorktreeStatus: fields[1][1],
				Submodule:      parseSubmoduleField(fields[2]),
			})
		case '2':
			fields, err := splitStatusFields(record, 10)
			if err != nil {
				return nil, err
			}
			i++
			if i >= len(records) {
				return nil, fmt.Errorf("rename entry missing original path: %q", record)
			}
			statuses = append(statuses, Status{
				Path:           fields[9],
				OrigPath:       records[i],
				IndexStatus:    fields[1][0],
				WorktreeStatus: fields[1][1],
				Submodule:      parseSubmoduleField(fields[2]),
			})
		case 'u':
			fields, err := splitStatusFields(record, 11)
			if err != nil {
				return nil, err
			}
			statuses = append(statuses, Status{
				Path:           fields[10],
				IndexStatus:    fields[1][0],
				WorktreeStatus: fields[1][1],
				Conflicted:     true,
				Submodule:      parseSubmoduleField(fields[2]),
			})
		case '?', '!':
			fields, err := splitStatusFields(record, 2)
			if err != nil {
				return nil, err
			}
			statuses = append(statuses, Status{Path: fields[1], IndexStatus: record[0], WorktreeStatus: record[0]})
		default:
			return nil, fmt.Errorf("unrecognized status entry: %q", record)
		}
	}

	return statuses, nil
}

func splitStatusFields(record string, n int) ([]string, error) {
	fields := strings.SplitN(record, " ", n)
	if len(fields) != n {
		return nil, fmt.Errorf("malformed status entry: %q", record)
	}
	return fields, nil
}
