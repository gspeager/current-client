package app

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/cgi"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gspeager/current-client/core/git"
)

const signedInURL = "https://deploy:s3cret@git.example.com/a.git"

func TestSplitRemoteBranch(t *testing.T) {
	remotes := []git.Remote{{Name: "origin"}, {Name: "team"}, {Name: "team/origin"}}
	tests := []struct {
		ref, remote, branch string
		ok                  bool
	}{
		{"origin/feature", "origin", "feature", true},
		{"origin/user/feature", "origin", "user/feature", true},
		{"team/origin/feature", "team/origin", "feature", true},
		{"team/feature", "team", "feature", true},
		{"upstream/feature", "", "", false},
		{"origin/", "", "", false},
	}
	for _, tt := range tests {
		remote, branch, ok := splitRemoteBranch(remotes, tt.ref)
		if remote != tt.remote || branch != tt.branch || ok != tt.ok {
			t.Errorf("splitRemoteBranch(%q) = %q, %q, %v; want %q, %q, %v", tt.ref, remote, branch, ok, tt.remote, tt.branch, tt.ok)
		}
	}
}

func TestHostLinksPrefersOrigin(t *testing.T) {
	remotes := []git.Remote{
		{Name: "fork", FetchURL: "git@github.com:me/repo.git"},
		{Name: "origin", FetchURL: "", PushURL: "https://gitlab.com/group/repo.git"},
	}
	links := hostLinksFor(remotes, nil)
	if links == nil || links.Issue != "https://gitlab.com/group/repo/-/issues/" {
		t.Fatalf("links = %+v", links)
	}
	if links := hostLinksFor(remotes[:1], nil); links == nil || links.Issue != "https://github.com/me/repo/issues/" {
		t.Fatalf("first remote links = %+v", links)
	}
	if links := hostLinksFor([]git.Remote{{Name: "origin", FetchURL: "/srv/git/repo.git"}}, nil); links != nil {
		t.Fatalf("local remote links = %+v", links)
	}
	if hostLinksFor(nil, nil) != nil {
		t.Fatal("no remotes should give no links")
	}
}

func TestListHidesASignInFromTheFrontend(t *testing.T) {
	repo := t.TempDir()
	gitIn(t, repo, "init", "-q")
	gitIn(t, repo, "remote", "add", "origin", signedInURL)
	gitIn(t, repo, "remote", "set-url", "--push", "origin", "ssh://git:s3cret@git.example.com/a.git")

	remotes, err := new(RemoteService).List(repo)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	want := RemoteInfo{Name: "origin", FetchURL: "https://<token>@git.example.com/a.git", PushURL: "ssh://git:<token>@git.example.com/a.git"}
	if len(remotes) != 1 || remotes[0] != want {
		t.Fatalf("List = %+v, want %+v", remotes, want)
	}
	sent, err := json.Marshal(remotes)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(sent), "s3cret") {
		t.Fatalf("the frontend would receive the password: %s", sent)
	}
	if got := gitOut(t, repo, "remote", "get-url", "origin"); got != signedInURL {
		t.Fatalf("Git's URL = %q, want %q", got, signedInURL)
	}
}

func TestEditRemoteNeverGivesGitThePlaceholder(t *testing.T) {
	repo := t.TempDir()
	gitIn(t, repo, "init", "-q")
	gitIn(t, repo, "remote", "add", "origin", signedInURL)
	s := new(RemoteService)
	remotes, err := s.List(repo)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	gitURL := func() string { return gitOut(t, repo, "remote", "get-url", "origin") }

	shown := remotes[0].FetchURL
	for _, url := range []string{shown, strings.Replace(shown, "a.git", "b.git", 1)} {
		if err := s.EditRemote(repo, "origin", url); err == nil {
			t.Errorf("EditRemote(%q) succeeded, want it refused", url)
		}
		if got := gitURL(); got != signedInURL {
			t.Fatalf("after EditRemote(%q), Git's URL = %q, want %q", url, got, signedInURL)
		}
	}

	for _, url := range []string{"https://git.example.com/b.git", "https://deploy:n3w@git.example.com/b.git"} {
		if err := s.EditRemote(repo, "origin", url); err != nil {
			t.Fatalf("EditRemote(%q): %v", url, err)
		}
		if got := gitURL(); got != url {
			t.Fatalf("Git's URL = %q, want %q", got, url)
		}
	}
}

// signedInRemote serves group/app.git over smart HTTP behind Basic auth for
// deploy / s3cret, and returns a clone that signs in only through origin's URL.
func signedInRemote(t *testing.T) (repo, bare string) {
	t.Helper()
	execPath, err := exec.Command("git", "--exec-path").Output()
	if err != nil {
		t.Skipf("git --exec-path: %v", err)
	}
	backend, err := exec.LookPath(filepath.Join(strings.TrimSpace(string(execPath)), "git-http-backend"))
	if err != nil {
		t.Skip("git-http-backend not available")
	}
	// Git offers a working sign-in to the machine's helper, such as osxkeychain.
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", home)
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")

	root := t.TempDir()
	bare = filepath.Join(root, "group", "app.git")
	if err := os.MkdirAll(filepath.Dir(bare), 0o755); err != nil {
		t.Fatal(err)
	}
	gitIn(t, root, "init", "-q", "--bare", "-b", "main", bare)
	backendHandler := &cgi.Handler{Path: backend, Env: []string{"GIT_PROJECT_ROOT=" + root, "GIT_HTTP_EXPORT_ALL=1"}}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if user, pass, ok := r.BasicAuth(); !ok || user != "deploy" || pass != "s3cret" {
			w.Header().Set("WWW-Authenticate", `Basic realm="test"`)
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		backendHandler.ServeHTTP(w, r)
	}))
	t.Cleanup(server.Close)
	gitIn(t, bare, "config", "http.receivepack", "true")

	repo = t.TempDir()
	gitIn(t, repo, "init", "-q", "-b", "main")
	gitIn(t, repo, "config", "user.name", "T")
	gitIn(t, repo, "config", "user.email", "t@example.com")
	gitIn(t, repo, "commit", "-q", "--allow-empty", "-m", "first")
	gitIn(t, repo, "remote", "add", "origin", strings.Replace(server.URL, "http://", "http://deploy:s3cret@", 1)+"/group/app.git")
	return repo, bare
}

func TestFetchAndPushSignInWithTheURLListHides(t *testing.T) {
	useTempSettings(t)
	repo, bare := signedInRemote(t)
	s := new(RemoteService)
	ctx := context.Background()

	remotes, err := s.List(repo)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if strings.Contains(remotes[0].FetchURL, "s3cret") {
		t.Fatalf("List = %+v, shows the password", remotes)
	}

	pushed, err := s.PushSetUpstream(ctx, repo, "origin", "main", nil)
	if err != nil {
		t.Fatalf("PushSetUpstream: %v", err)
	}
	if got, want := gitOut(t, bare, "rev-parse", "main"), gitOut(t, repo, "rev-parse", "HEAD"); got != want {
		t.Fatalf("the remote's main = %s, want %s", got, want)
	}
	if !strings.Contains(pushed.CreateLink, "/group/app") || strings.Contains(pushed.CreateLink, "s3cret") {
		t.Fatalf("create link = %q, want the project without the password", pushed.CreateLink)
	}

	gitIn(t, repo, "commit", "-q", "--allow-empty", "-m", "second")
	if _, err := s.Push(ctx, repo, nil); err != nil {
		t.Fatalf("Push: %v", err)
	}
	gitIn(t, repo, "update-ref", "-d", "refs/remotes/origin/main")
	if err := s.Fetch(ctx, repo, "origin", nil); err != nil {
		t.Fatalf("Fetch: %v", err)
	}
	if got, want := gitOut(t, repo, "rev-parse", "origin/main"), gitOut(t, repo, "rev-parse", "HEAD"); got != want {
		t.Fatalf("origin/main after Fetch = %s, want %s", got, want)
	}
}

func TestHostLinksComeFromASignedInURLWithoutTheSignIn(t *testing.T) {
	links := hostLinksFor([]git.Remote{{Name: "origin", FetchURL: "https://deploy:s3cret@gitlab.example.com/group/repo.git"}}, nil)
	if links == nil || links.Issue != "https://gitlab.example.com/group/repo/-/issues/" {
		t.Fatalf("links = %+v", links)
	}
	if sent, _ := json.Marshal(links); strings.Contains(string(sent), "s3cret") {
		t.Fatalf("links show the password: %s", sent)
	}
}
