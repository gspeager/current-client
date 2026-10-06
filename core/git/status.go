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

func GetStatus(ctx context.Context, repoPath string) ([]Status, error) {
	result, err := runResult(ctx, repoPath, "status", "--porcelain=v2", "-z")
	if err != nil {
		return nil, err
	}
	return ParseStatus(result.Stdout)
}

func ParseStatus(output string) ([]Status, error) {
	records := strings.Split(output, "\x00")
	var statuses []Status

	for i := 0; i < len(records); i++ {
		record := records[i]
		if record == "" {
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
