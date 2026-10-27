package hostlink

import (
	"errors"
	"net/url"
	"regexp"
	"strings"
)

type TrackerKind string

const (
	JiraCloud      TrackerKind = "jira-cloud"
	JiraDataCenter TrackerKind = "jira-data-center"
)

// Tracker is an issue tracker beside the code host, such as a team's Jira
// site. Its links are built from the address the user entered; nothing is
// asked of it. Keys limits which project keys link; empty links any.
type Tracker struct {
	Site string      `json:"site"`
	Kind TrackerKind `json:"kind"`
	Keys []string    `json:"keys"`
}

var (
	ErrTrackerSite = errors.New("the site address needs to start with https:// or http://, such as https://example.atlassian.net")
	projectKey     = regexp.MustCompile(`^[A-Z][A-Z0-9_]+$`)
)

// TrackerSite is the site address as the app keeps it: scheme and host
// lowercased, no query or fragment, one trailing slash, so a Data Center
// site under a path ("https://example.com/jira/") keeps its path.
func TrackerSite(raw string) (string, error) {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || (u.Scheme != "https" && u.Scheme != "http") || u.Host == "" || u.User != nil {
		return "", ErrTrackerSite
	}
	return strings.ToLower(u.Scheme) + "://" + strings.ToLower(u.Host) + strings.TrimRight(u.Path, "/") + "/", nil
}

// ProjectKeys uppercases and keeps the entries shaped like Jira project
// keys, each once.
func ProjectKeys(raw []string) []string {
	var keys []string
	seen := map[string]bool{}
	for _, k := range raw {
		k = strings.ToUpper(strings.TrimSpace(k))
		if projectKey.MatchString(k) && !seen[k] {
			seen[k] = true
			keys = append(keys, k)
		}
	}
	return keys
}

// IssueURL is the issue's page; Jira Cloud and Data Center share /browse/.
func (t Tracker) IssueURL(key string) string {
	return t.Site + "browse/" + key
}

// KeyPattern matches one issue key, in group 1, standing on its own: not
// inside a word, a path or a longer key.
func (t Tracker) KeyPattern() *regexp.Regexp {
	return regexp.MustCompile(`(?:^|[^\w/.-])(` + t.issueKey() + `)\b`)
}

// BranchKeyPattern matches the key a branch is named after, in group 1:
// "PROJ-123-fix-login", "feature/PROJ-123".
func (t Tracker) BranchKeyPattern() *regexp.Regexp {
	return regexp.MustCompile(`(?:^|/)(` + t.issueKey() + `)(?:[-_/]|$)`)
}

func (t Tracker) issueKey() string {
	if len(t.Keys) == 0 {
		return `[A-Z][A-Z0-9_]+-[1-9]\d*`
	}
	quoted := make([]string, len(t.Keys))
	for i, k := range t.Keys {
		quoted[i] = regexp.QuoteMeta(k)
	}
	return "(?:" + strings.Join(quoted, "|") + `)-[1-9]\d*`
}
