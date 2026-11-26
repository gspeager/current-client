//go:build !nohost

package app

import (
	"context"
	"regexp"
	"strconv"
	"strings"
)

// JobLog is the end of a CI job's log, or all of it up to the cap, with
// terminal escape sequences removed and invalid bytes replaced.
type JobLog struct {
	Text      string `json:"text"`
	Truncated bool   `json:"truncated"`
	Full      bool   `json:"full"`
}

const (
	jobLogTailLines = 50
	jobLogTailBytes = 64 << 10
	jobLogFullBytes = 2 << 20
)

// JobLog fetches a job's log when asked, never in advance: the last
// jobLogTailLines by default, or up to jobLogFullBytes with full. Each is
// fetched once per job until a refresh.
func (s *HostService) JobLog(repoPath string, jobID int64, full bool) (JobLog, error) {
	_, info, ok, err := enabledHost(repoPath)
	if err != nil || !ok {
		return JobLog{}, err
	}
	provider := providerFor(info)
	if provider == nil {
		return JobLog{}, nil
	}
	key := strconv.FormatInt(jobID, 10)
	if full {
		key += ":full"
	}
	if cached, _, ok := cacheJobLog.get(info.Hostname, repoPath, key); ok {
		return cached, nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), slowHostTimeout)
	defer cancel()
	limit := jobLogTailBytes
	if full {
		limit = jobLogFullBytes
	}
	raw, truncated, err := provider.JobLog(ctx, info.Project, jobID, limit)
	if err != nil {
		return JobLog{}, err
	}
	log := JobLog{Text: cleanLog(raw), Truncated: truncated, Full: full}
	if !full {
		log.Text, log.Truncated = tailLines(log.Text, jobLogTailLines, truncated)
	}
	cacheJobLog.put(info.Hostname, repoPath, key, log)
	return log, nil
}

// Terminal escape sequences: CSI (colors, cursor moves), OSC (titles,
// hyperlinks) and the single-character ones GitLab's section markers use.
var escapes = regexp.MustCompile(`\x1b\[[0-9;?]*[ -/]*[@-~]|\x1b\][^\x07\x1b]*(?:\x07|\x1b\\)|\x1b[@-Z\\-_]|\r`)

func cleanLog(raw string) string {
	return strings.ToValidUTF8(escapes.ReplaceAllString(raw, ""), "�")
}

func tailLines(text string, n int, alreadyCut bool) (string, bool) {
	lines := strings.Split(strings.TrimRight(text, "\n"), "\n")
	if len(lines) <= n {
		return strings.Join(lines, "\n"), alreadyCut
	}
	return strings.Join(lines[len(lines)-n:], "\n"), true
}
