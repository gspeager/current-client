package hostapi

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// storeHelper gives the test Git's store helper in its own file, keeping the
// machine's helpers out.
func storeHelper(t *testing.T) (storeFile string) {
	t.Helper()
	home := isolateGitConfig(t)
	storeFile = filepath.Join(home, "credentials")
	writeGitConfig(t, home, "[credential]\n\thelper = store --file="+filepath.ToSlash(storeFile)+"\n")
	return storeFile
}

func gitFill(t *testing.T, input string) string {
	t.Helper()
	cmd := exec.Command("git", "credential", "fill")
	cmd.Stdin = strings.NewReader(input)
	cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")
	out, _ := cmd.Output()
	return string(out)
}

func TestSavedTokenNeverAnswersAPushLookup(t *testing.T) {
	storeFile := storeHelper(t)
	os.WriteFile(storeFile, []byte("https://alice:s3cret@gitlab.example.com\n"), 0o600)
	creds := &GitCredentials{Saved: func(string) bool { return true }}
	ctx := context.Background()

	if err := creds.Save(ctx, "gitlab.example.com", " glpat-123 \n"); err != nil {
		t.Fatalf("Save: %v", err)
	}
	stored, _ := os.ReadFile(storeFile)
	if !strings.Contains(string(stored), "https://current-client-api:glpat-123@api.gitlab.example.com") || !strings.Contains(string(stored), "alice:s3cret@gitlab.example.com") {
		t.Fatalf("store file = %q", stored)
	}

	// What git push asks for: the host, no username.
	if got := gitFill(t, "protocol=https\nhost=gitlab.example.com\n\n"); !strings.Contains(got, "password=s3cret") || strings.Contains(got, "glpat") {
		t.Fatalf("a push lookup got %q, want Git's own credential", got)
	}
	if token, err := creds.Token(ctx, "gitlab.example.com"); err != nil || token != "glpat-123" {
		t.Fatalf("Token = %q, %v", token, err)
	}

	if err := creds.Remove(ctx, "gitlab.example.com"); err != nil {
		t.Fatalf("Remove: %v", err)
	}
	stored, _ = os.ReadFile(storeFile)
	if strings.Contains(string(stored), "glpat") || !strings.Contains(string(stored), "alice:s3cret") {
		t.Fatalf("after Remove, store file = %q", stored)
	}
	if token, _ := creds.Token(ctx, "gitlab.example.com"); token != "s3cret" {
		t.Fatalf("after Remove, Token = %q, want Git's own credential", token)
	}
}

func TestTokenIsGitsOwnCredentialOrNothing(t *testing.T) {
	storeFile := storeHelper(t)
	creds := &GitCredentials{}
	if token, err := creds.Token(context.Background(), "gitlab.example.com"); err != nil || token != "" {
		t.Fatalf("with nothing stored: %q, %v", token, err)
	}
	creds.Forget("gitlab.example.com")
	os.WriteFile(storeFile, []byte("https://alice:s3cret@gitlab.example.com\n"), 0o600)
	if token, _ := creds.Token(context.Background(), "gitlab.example.com"); token != "s3cret" {
		t.Fatalf("with Git's credential stored: %q", token)
	}
}

func TestSavedGatesTheTokenKeyLookup(t *testing.T) {
	home := isolateGitConfig(t)
	log := filepath.Join(home, "asked")
	writeGitConfig(t, home, "[credential]\n\thelper = \"!f() { test \\\"$1\\\" = get && cat >> "+filepath.ToSlash(log)+"; }; f\"\n")
	ctx := context.Background()

	notSaved := &GitCredentials{Saved: func(string) bool { return false }}
	notSaved.Token(ctx, "gitlab.example.com")
	asked, _ := os.ReadFile(log)
	if strings.Contains(string(asked), "api.gitlab.example.com") || !strings.Contains(string(asked), "host=gitlab.example.com") {
		t.Fatalf("without a saved token the helper was asked:\n%s", asked)
	}

	os.Remove(log)
	saved := &GitCredentials{Saved: func(string) bool { return true }}
	saved.Token(ctx, "gitlab.example.com")
	asked, _ = os.ReadFile(log)
	if !strings.HasPrefix(string(asked), "protocol=https\nhost=api.gitlab.example.com\nusername=current-client-api\n") {
		t.Fatalf("with a saved token the key should be asked first:\n%s", asked)
	}
}

func TestSaveRejectsEmptyAndMultilineTokens(t *testing.T) {
	storeHelper(t)
	creds := &GitCredentials{}
	for _, token := range []string{"", "  ", "glpat\nmore"} {
		if err := creds.Save(context.Background(), "gitlab.example.com", token); err == nil {
			t.Errorf("Save(%q) accepted", token)
		}
	}
}

func TestTokenStaysOffTheCommandLine(t *testing.T) {
	key := tokenKey("gitlab.example.com:8443")
	key.password = "glpat-123"
	if got := key.input(); got != "protocol=https\nhost=api.gitlab.example.com:8443\nusername=current-client-api\npassword=glpat-123\n\n" {
		t.Fatalf("input = %q", got)
	}
}
