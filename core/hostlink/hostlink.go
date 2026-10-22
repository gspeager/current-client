// Package hostlink turns a remote URL into links to the host's web pages,
// without talking to the host.
package hostlink

import (
	"net/url"
	"regexp"
	"strings"
)

type Style string

const (
	GitHub Style = "github"
	GitLab Style = "gitlab"
)

// Host is where a repository's web pages live.
type Host struct {
	// Root is the host's web origin with a trailing slash, such as "https://gitlab.example.com/".
	Root    string
	Project string
	Style   Style
}

// Links are the URL prefixes a reference completes: Issue+"123",
// MergeRequest+"45", and Root+"group/project"+IssuePath+"12" for a
// reference to another project. MergeRequest is empty where "!45" isn't
// host syntax.
type Links struct {
	Issue        string `json:"issue"`
	MergeRequest string `json:"mergeRequest"`
	Root         string `json:"root"`
	IssuePath    string `json:"issuePath"`
}

// scpLike matches "git@host:path"; a path can't contain "://", and Windows
// drive letters are kept out by requiring more than one character before ":".
var scpLike = regexp.MustCompile(`^(?:[^@/:]+@)?([^@/:]{2,}):(.+)$`)

// Parse reads the host and project out of a remote URL in any form Git
// accepts: HTTPS, ssh://, git://, or scp-like "git@host:group/repo.git".
// Local paths and URLs with no project path report false.
func Parse(remoteURL string) (Host, bool) {
	remoteURL = strings.TrimSpace(remoteURL)
	var scheme, host, path string
	switch {
	case strings.Contains(remoteURL, "://"):
		u, err := url.Parse(remoteURL)
		if err != nil || u.Hostname() == "" {
			return Host{}, false
		}
		switch u.Scheme {
		case "http", "https":
			scheme, host = u.Scheme, u.Host
		case "ssh", "git", "git+ssh", "ssh+git":
			scheme, host = "https", u.Hostname()
		default:
			return Host{}, false
		}
		path = u.Path
	default:
		m := scpLike.FindStringSubmatch(remoteURL)
		if m == nil || strings.HasPrefix(m[2], `\`) {
			return Host{}, false
		}
		scheme, host, path = "https", m[1], m[2]
	}
	project := strings.Trim(path, "/")
	project = strings.TrimSuffix(project, ".git")
	project = strings.TrimSuffix(project, "/")
	if strings.Count(project, "/") < 1 || strings.Contains(project, "//") {
		return Host{}, false
	}
	host = strings.ToLower(host)
	return Host{Root: scheme + "://" + host + "/", Project: project, Style: styleFor(host, project)}, true
}

// GitHub and GitLab are recognized by name; any other host that nests groups
// is taken for GitLab, since GitHub-style hosts have exactly owner/repo.
func styleFor(host, project string) Style {
	hostname := strings.Split(host, ":")[0]
	switch {
	case hostname == "github.com":
		return GitHub
	case strings.Contains(hostname, "gitlab"), strings.Count(project, "/") > 1:
		return GitLab
	}
	return GitHub
}

func (h Host) Links() Links {
	issuePath := "/issues/"
	mergeRequest := ""
	if h.Style == GitLab {
		issuePath = "/-/issues/"
		mergeRequest = h.Root + h.Project + "/-/merge_requests/"
	}
	return Links{
		Issue:        h.Root + h.Project + issuePath,
		MergeRequest: mergeRequest,
		Root:         h.Root,
		IssuePath:    issuePath,
	}
}
