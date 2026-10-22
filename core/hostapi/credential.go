package hostapi

import (
	"context"
	"errors"
	"strings"
	"sync"

	"github.com/gspeager/current-client/core/gitexec"
)

// GitCredentials gets a host's API token from Git's own credential helpers
// and stores nothing itself. A token pasted in Settings is saved by the
// helper under its own key, https://current-client-api@api.<host>, which no
// fetch or push ever asks for: a helper answers a push's lookup (host only,
// no username) with any entry for that host, so a read-only token stored
// under the host itself would be tried for the push, rejected, and erased.
// Without a saved token, Git's own credential for the host is used, which
// Git Credential Manager already holds for many people.
type GitCredentials struct {
	// Saved says whether a token was saved for the host, so its key is only
	// looked up when there is one: Git Credential Manager opens a sign-in
	// window for a host it has nothing for.
	Saved func(hostName string) bool

	mu     sync.Mutex
	tokens map[string]string
}

const tokenUsername = "current-client-api"

type credential struct {
	protocol, host, username, password string
}

func tokenKey(hostName string) credential {
	return credential{protocol: "https", host: "api." + hostName, username: tokenUsername}
}

// Token is asked once per host and remembered until Save, Remove or Forget;
// helpers can be slow or show a window, so it isn't asked per request.
func (g *GitCredentials) Token(ctx context.Context, hostName string) (string, error) {
	g.mu.Lock()
	token, ok := g.tokens[hostName]
	g.mu.Unlock()
	if ok {
		return token, nil
	}
	if g.Saved != nil && g.Saved(hostName) {
		token = fill(ctx, tokenKey(hostName))
	}
	if token == "" {
		token = fill(ctx, credential{protocol: "https", host: hostName})
	}
	g.remember(hostName, token)
	return token, nil
}

func (g *GitCredentials) Save(ctx context.Context, hostName, token string) error {
	token = strings.TrimSpace(token)
	if token == "" {
		return errors.New("enter a token")
	}
	if strings.ContainsAny(token, "\n\r\x00") {
		return errors.New("the token contains a line break")
	}
	key := tokenKey(hostName)
	key.password = token
	if err := run(ctx, "approve", key); err != nil {
		return err
	}
	g.remember(hostName, token)
	return nil
}

func (g *GitCredentials) Remove(ctx context.Context, hostName string) error {
	if err := run(ctx, "reject", tokenKey(hostName)); err != nil {
		return err
	}
	g.Forget(hostName)
	return nil
}

func (g *GitCredentials) Forget(hostName string) {
	g.mu.Lock()
	delete(g.tokens, hostName)
	g.mu.Unlock()
}

func (g *GitCredentials) remember(hostName, token string) {
	g.mu.Lock()
	if g.tokens == nil {
		g.tokens = map[string]string{}
	}
	g.tokens[hostName] = token
	g.mu.Unlock()
}

// fill returns the password Git's helpers hold for the key, or "" when they
// hold none; with terminal prompts disabled Git fails instead of asking.
func fill(ctx context.Context, key credential) string {
	result, err := gitexec.NewExecutor("").Run(ctx, gitexec.Command{
		Args:  []string{"credential", "fill"},
		Stdin: strings.NewReader(key.input()),
	})
	if err != nil || result.ExitCode != 0 {
		return ""
	}
	for _, line := range strings.Split(result.Stdout, "\n") {
		if password, ok := strings.CutPrefix(line, "password="); ok {
			return password
		}
	}
	return ""
}

func run(ctx context.Context, action string, key credential) error {
	result, err := gitexec.NewExecutor("").Run(ctx, gitexec.Command{
		Args:  []string{"credential", action},
		Stdin: strings.NewReader(key.input()),
	})
	if err != nil {
		return err
	}
	if result.ExitCode != 0 {
		return errors.New("Git's credential helper could not " + action + " the token")
	}
	return nil
}

// input is Git's credential protocol; the token reaches Git on stdin, never
// on the command line.
func (c credential) input() string {
	var b strings.Builder
	b.WriteString("protocol=" + c.protocol + "\nhost=" + c.host + "\n")
	if c.username != "" {
		b.WriteString("username=" + c.username + "\n")
	}
	if c.password != "" {
		b.WriteString("password=" + c.password + "\n")
	}
	b.WriteString("\n")
	return b.String()
}
