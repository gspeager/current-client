package hostlink

import "testing"

func TestParse(t *testing.T) {
	tests := []struct {
		url  string
		want Host
	}{
		{"https://github.com/owner/repo.git", Host{"https://github.com/", "owner/repo", GitHub}},
		{"https://github.com/owner/repo", Host{"https://github.com/", "owner/repo", GitHub}},
		{"https://github.com/owner/repo/", Host{"https://github.com/", "owner/repo", GitHub}},
		{"https://user:token@github.com/owner/repo.git", Host{"https://github.com/", "owner/repo", GitHub}},
		{"http://git.example.com:8080/owner/repo.git", Host{"http://git.example.com:8080/", "owner/repo", GitHub}},
		{"git@github.com:owner/repo.git", Host{"https://github.com/", "owner/repo", GitHub}},
		{"git@GitHub.com:owner/repo.git", Host{"https://github.com/", "owner/repo", GitHub}},
		{"ssh://git@github.com/owner/repo.git", Host{"https://github.com/", "owner/repo", GitHub}},
		{"ssh://git@gitlab.example.com:2222/group/sub/repo.git", Host{"https://gitlab.example.com/", "group/sub/repo", GitLab}},
		{"git@gitlab.com:group/sub/repo.git", Host{"https://gitlab.com/", "group/sub/repo", GitLab}},
		{"https://gitlab.com/owner/repo.git", Host{"https://gitlab.com/", "owner/repo", GitLab}},
		{"https://gitlab.example.com/owner/repo.git", Host{"https://gitlab.example.com/", "owner/repo", GitLab}},
		{"https://git.example.com/group/sub/repo.git", Host{"https://git.example.com/", "group/sub/repo", GitLab}},
		{"https://git.example.com/owner/repo.git", Host{"https://git.example.com/", "owner/repo", GitHub}},
		{"git://git.example.com/owner/repo.git", Host{"https://git.example.com/", "owner/repo", GitHub}},
		{"git+ssh://git@git.example.com/owner/repo.git", Host{"https://git.example.com/", "owner/repo", GitHub}},
		{"  git@github.com:owner/repo.git\n", Host{"https://github.com/", "owner/repo", GitHub}},
	}
	for _, tt := range tests {
		got, ok := Parse(tt.url)
		if !ok || got != tt.want {
			t.Errorf("Parse(%q) = %+v, %v; want %+v", tt.url, got, ok, tt.want)
		}
	}
}

func TestParseRejectsWhatIsNotAHostedProject(t *testing.T) {
	for _, u := range []string{
		"",
		"/Users/me/repo.git",
		"../repo",
		"file:///Users/me/repo.git",
		`C:\repos\app`,
		"https://github.com/",
		"https://github.com/owner",
		"git@github.com:repo.git",
		"https://github.com/owner//repo",
		"ftp://example.com/owner/repo",
	} {
		if got, ok := Parse(u); ok {
			t.Errorf("Parse(%q) = %+v, want no host", u, got)
		}
	}
}

func TestLinks(t *testing.T) {
	tests := []struct {
		url  string
		want Links
	}{
		{"git@github.com:owner/repo.git", Links{
			Issue:     "https://github.com/owner/repo/issues/",
			Root:      "https://github.com/",
			IssuePath: "/issues/",
		}},
		{"https://gitlab.com/group/sub/repo.git", Links{
			Issue:        "https://gitlab.com/group/sub/repo/-/issues/",
			MergeRequest: "https://gitlab.com/group/sub/repo/-/merge_requests/",
			Root:         "https://gitlab.com/",
			IssuePath:    "/-/issues/",
		}},
		{"http://git.example.com:8080/owner/repo", Links{
			Issue:     "http://git.example.com:8080/owner/repo/issues/",
			Root:      "http://git.example.com:8080/",
			IssuePath: "/issues/",
		}},
	}
	for _, tt := range tests {
		host, ok := Parse(tt.url)
		if !ok {
			t.Fatalf("Parse(%q) found no host", tt.url)
		}
		if got := host.Links(); got != tt.want {
			t.Errorf("Links(%q) = %+v, want %+v", tt.url, got, tt.want)
		}
	}
}
