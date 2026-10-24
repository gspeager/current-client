package hostlink

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func fakeSSH(t *testing.T, hostnames map[string]string, err error) *[]string {
	t.Helper()
	var asked []string
	previous := sshHostname
	sshHostname = func(_ context.Context, alias string) (string, error) {
		asked = append(asked, alias)
		return hostnames[alias], err
	}
	t.Cleanup(func() { sshHostname = previous })
	return &asked
}

func TestResolveReplacesSSHAliases(t *testing.T) {
	asked := fakeSSH(t, map[string]string{"work": "GitLab.example.com", "github.com": "github.com"}, nil)
	tests := []struct {
		url  string
		want Host
	}{
		{"git@work:group/repo.git", Host{"https://gitlab.example.com/", "group/repo", GitLab}},
		{"ssh://git@work:2222/group/repo.git", Host{"https://gitlab.example.com/", "group/repo", GitLab}},
		{"git@github.com:owner/repo.git", Host{"https://github.com/", "owner/repo", GitHub}},
		{"https://work/owner/repo.git", Host{"https://work/", "owner/repo", GitHub}},
		{"git://work/owner/repo.git", Host{"https://work/", "owner/repo", GitHub}},
	}
	for _, tt := range tests {
		got, ok := Resolve(context.Background(), tt.url)
		if !ok || got != tt.want {
			t.Errorf("Resolve(%q) = %+v, %v; want %+v", tt.url, got, ok, tt.want)
		}
	}
	if want := []string{"work", "work", "github.com"}; len(*asked) != len(want) || (*asked)[0] != "work" || (*asked)[2] != "github.com" {
		t.Errorf("ssh asked for %v, want %v", *asked, want)
	}
	if _, ok := Resolve(context.Background(), "/srv/git/repo.git"); ok {
		t.Error("a local path resolved to a host")
	}
}

func TestResolveKeepsTheAliasWhenSSHFails(t *testing.T) {
	fakeSSH(t, nil, errors.New("no ssh"))
	got, ok := Resolve(context.Background(), "git@work:group/repo.git")
	if want := (Host{"https://work/", "group/repo", GitHub}); !ok || got != want {
		t.Errorf("Resolve = %+v, %v; want %+v", got, ok, want)
	}
}

func TestSSHConfigHostnameReadsTheConfigFile(t *testing.T) {
	if _, err := exec.LookPath("ssh"); err != nil {
		t.Skip("ssh not installed")
	}
	cfg := filepath.Join(t.TempDir(), "config")
	if err := os.WriteFile(cfg, []byte("Host work\n\tHostName gitlab.example.com\n\tUser git\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	got, err := sshConfigHostname(context.Background(), "work", "-F", cfg)
	if err != nil || got != "gitlab.example.com" {
		t.Fatalf("sshConfigHostname = %q, %v", got, err)
	}
}

func TestParseSSHHostname(t *testing.T) {
	output := "user git\nhostname gitlab.example.com\nport 22\nhostkeyalias work\n"
	if got := parseSSHHostname(output); got != "gitlab.example.com" {
		t.Errorf("parseSSHHostname = %q", got)
	}
	if got := parseSSHHostname("user git\n"); got != "" {
		t.Errorf("parseSSHHostname without hostname = %q", got)
	}
}

func TestAPI(t *testing.T) {
	tests := []struct {
		url  string
		want string
	}{
		{"git@github.com:owner/repo.git", "https://api.github.com"},
		{"https://gitlab.com/group/sub/repo.git", "https://gitlab.com/api/v4"},
		{"https://github.example.com/owner/repo.git", "https://github.example.com/api/v3"},
		{"https://gitlab.example.com:8443/group/repo.git", "https://gitlab.example.com:8443/api/v4"},
		{"http://git.example.com:8080/group/sub/repo.git", "http://git.example.com:8080/api/v4"},
	}
	for _, tt := range tests {
		host, ok := Parse(tt.url)
		if !ok {
			t.Fatalf("Parse(%q) found no host", tt.url)
		}
		if got := host.API(); got != tt.want {
			t.Errorf("API(%q) = %q, want %q", tt.url, got, tt.want)
		}
	}
	if got := (Host{Root: "https://git.example.com:8443/", Style: GitHub}).Hostname(); got != "git.example.com:8443" {
		t.Errorf("Hostname = %q", got)
	}
}
