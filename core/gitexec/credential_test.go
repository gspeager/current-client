package gitexec

import (
	"context"
	"net/http"
	"net/http/cgi"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// gitServer serves a repository over smart HTTP, as git http-backend behind
// Basic auth for alice / s3cret.
func gitServer(t *testing.T) string {
	t.Helper()
	execPath, err := exec.Command("git", "--exec-path").Output()
	if err != nil {
		t.Skipf("git --exec-path: %v", err)
	}
	backend := filepath.Join(strings.TrimSpace(string(execPath)), "git-http-backend")
	if _, err := os.Stat(backend); err != nil {
		if _, err := os.Stat(backend + ".exe"); err != nil {
			t.Skip("git-http-backend not available")
		}
		backend += ".exe"
	}

	root := t.TempDir()
	work := filepath.Join(t.TempDir(), "work")
	for _, args := range [][]string{
		{"init", "-q", work},
		{"-C", work, "-c", "user.name=T", "-c", "user.email=t@example.com", "commit", "-q", "--allow-empty", "-m", "first"},
		{"clone", "-q", "--bare", work, filepath.Join(root, "repo.git")},
	} {
		if out, err := exec.Command("git", args...).CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}

	cgiHandler := &cgi.Handler{
		Path: backend,
		Env:  []string{"GIT_PROJECT_ROOT=" + root, "GIT_HTTP_EXPORT_ALL=1"},
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if user, pass, ok := r.BasicAuth(); !ok || user != "alice" || pass != "s3cret" {
			w.Header().Set("WWW-Authenticate", `Basic realm="test"`)
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		cgiHandler.ServeHTTP(w, r)
	}))
	t.Cleanup(server.Close)
	return server.URL + "/repo.git"
}

// isolateGitConfig keeps the machine's helpers (such as the system-wide
// osxkeychain) out of the test, and gives it a store helper in its own file.
func isolateGitConfig(t *testing.T) (storeFile string) {
	t.Helper()
	home := t.TempDir()
	storeFile = filepath.Join(home, "credentials")
	config := "[credential]\n\thelper = store --file=" + filepath.ToSlash(storeFile) + "\n"
	if err := os.WriteFile(filepath.Join(home, ".gitconfig"), []byte(config), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", home)
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	return storeFile
}

func cloneWith(ctx context.Context, t *testing.T, url string) (Result, error) {
	t.Helper()
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	return NewExecutor("").Run(ctx, Command{Args: []string{"clone", "-q", url, filepath.Join(t.TempDir(), "clone")}})
}

func TestCredentialIsOfferedAndStoredByTheConfiguredHelper(t *testing.T) {
	url := gitServer(t)
	storeFile := isolateGitConfig(t)

	result, err := cloneWith(context.Background(), t, url)
	if err != nil || result.ExitCode == 0 || !strings.Contains(result.Stderr, "terminal prompts disabled") {
		t.Fatalf("without a credential: err=%v exit=%d stderr=%q, want it to need credentials", err, result.ExitCode, result.Stderr)
	}

	ctx := WithCredential(context.Background(), Credential{Username: "alice", Password: "s3cret"})
	result, err = cloneWith(ctx, t, url)
	if err != nil || result.ExitCode != 0 {
		t.Fatalf("with a credential: err=%v exit=%d stderr=%q", err, result.ExitCode, result.Stderr)
	}

	stored, err := os.ReadFile(storeFile)
	if err != nil || !strings.Contains(string(stored), "alice:s3cret@") {
		t.Fatalf("store helper file = %q (%v), want Git to have saved the credential", stored, err)
	}

	// Remembered: no credential needed the next time.
	if result, err := cloneWith(context.Background(), t, url); err != nil || result.ExitCode != 0 {
		t.Fatalf("after storing: err=%v stderr=%q", err, result.Stderr)
	}
}

func TestConfiguredHelperIsAskedBeforeTheOfferedCredential(t *testing.T) {
	url := gitServer(t)
	storeFile := isolateGitConfig(t)
	host := strings.TrimPrefix(strings.SplitN(url, "/repo.git", 2)[0], "http://")
	if err := os.WriteFile(storeFile, []byte("http://alice:s3cret@"+host+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	ctx := WithCredential(context.Background(), Credential{Username: "alice", Password: "wrong"})
	if result, err := cloneWith(ctx, t, url); err != nil || result.ExitCode != 0 {
		t.Fatalf("err=%v stderr=%q, want the stored credential to be used", err, result.Stderr)
	}
}

func TestWrongCredentialFailsAsAuthentication(t *testing.T) {
	url := gitServer(t)
	isolateGitConfig(t)

	ctx := WithCredential(context.Background(), Credential{Username: "alice", Password: "wrong"})
	result, err := cloneWith(ctx, t, url)
	if err != nil || result.ExitCode == 0 {
		t.Fatalf("err=%v exit=%d, want a failed clone", err, result.ExitCode)
	}
	if got := WrapResult(result).Message; got != authFailedMessage {
		t.Errorf("message = %q, want %q (stderr %q)", got, authFailedMessage, result.Stderr)
	}
}

func TestCredentialStaysOffTheCommandLine(t *testing.T) {
	args, env := credentialArgs(Credential{Username: "alice", Password: "s3cret"})
	if strings.Contains(strings.Join(args, " "), "s3cret") || strings.Contains(strings.Join(args, " "), "alice") {
		t.Errorf("args %q contain the credential", args)
	}
	if strings.Join(env, "\n") != usernameEnv+"=alice\n"+passwordEnv+"=s3cret" {
		t.Errorf("env = %q", env)
	}
}

func TestValidateCredential(t *testing.T) {
	for _, cred := range []Credential{{"", "x"}, {"a", ""}, {"a\nb", "x"}, {"a", "x\ry"}, {"a", "x\x00"}} {
		if ValidateCredential(cred) == nil {
			t.Errorf("%q: expected an error", cred)
		}
	}
	if err := ValidateCredential(Credential{"alice", "ghp_token"}); err != nil {
		t.Errorf("valid credential: %v", err)
	}
}
