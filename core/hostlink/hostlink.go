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
// host syntax. Commit+sha, Tree+branch, Tag+name and Blob+sha+"/"+path
// (+"#L<n>") are the host's pages for those; Project is the repository's
// home page.
type Links struct {
	Issue        string `json:"issue"`
	MergeRequest string `json:"mergeRequest"`
	Root         string `json:"root"`
	IssuePath    string `json:"issuePath"`
	Project      string `json:"project"`
	Commit       string `json:"commit"`
	Tree         string `json:"tree"`
	Tag          string `json:"tag"`
	Blob         string `json:"blob"`
	Style        Style  `json:"style"`
}

// NewChange is the host's pre-filled form for a merge request / pull
// request from branch into base: no API, no token. For a pull request from
// a fork, base lives in the upstream project and the head is "owner:branch".
func (h Host) NewChange(branch, base string) string {
	project := h.Root + h.Project
	if h.Style == GitLab {
		q := url.Values{"merge_request[source_branch]": {branch}}
		if base != "" {
			q.Set("merge_request[target_branch]", base)
		}
		return project + "/-/merge_requests/new?" + q.Encode()
	}
	if base == "" {
		base = "main"
	}
	return project + "/compare/" + escapeRef(base) + "..." + escapeRef(branch) + "?expand=1"
}

// NewChangeFromFork is NewChange for a GitHub-style fork: the form lives in
// upstream's project and names the fork's owner on the head. GitLab's form
// is the fork's own, where the target project is chosen.
func (h Host) NewChangeFromFork(upstream Host, branch, base string) string {
	if h.Style == GitLab {
		return h.NewChange(branch, base)
	}
	owner, _, _ := strings.Cut(h.Project, "/")
	return upstream.NewChange(owner+":"+branch, base)
}

func escapeRef(ref string) string {
	parts := strings.Split(ref, "/")
	for i, part := range parts {
		parts[i] = url.PathEscape(part)
	}
	return strings.Join(parts, "/")
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

// DefaultStyle is the guess for a host entered by hand, with no project
// path to go on.
func DefaultStyle(hostname string) Style {
	return styleFor(hostname, "")
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

func (h Host) Hostname() string {
	u, err := url.Parse(h.Root)
	if err != nil {
		return ""
	}
	return u.Host
}

// API is the host's REST API address: GitLab serves it under the web root,
// github.com on its own host, GitHub Enterprise Server under /api/v3.
func (h Host) API() string {
	switch {
	case h.Style == GitLab:
		return h.Root + "api/v4"
	case h.Hostname() == "github.com":
		return "https://api.github.com"
	default:
		return h.Root + "api/v3"
	}
}

// Links builds the host's page addresses in its style: GitLab keeps its
// pages under "-/", GitHub doesn't, and GitHub's tag page is a release.
func (h Host) Links() Links {
	project := h.Root + h.Project
	issuePath, pages, tag, mergeRequest := "/issues/", project+"/", project+"/releases/tag/", ""
	if h.Style == GitLab {
		issuePath, pages, tag = "/-/issues/", project+"/-/", project+"/-/tags/"
		mergeRequest = pages + "merge_requests/"
	}
	return Links{
		Issue:        project + issuePath,
		MergeRequest: mergeRequest,
		Root:         h.Root,
		IssuePath:    issuePath,
		Project:      project,
		Commit:       pages + "commit/",
		Tree:         pages + "tree/",
		Tag:          tag,
		Blob:         pages + "blob/",
		Style:        h.Style,
	}
}
