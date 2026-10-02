package gitexec

import (
	"context"
	"strings"
)

// Credential is a username and password (or access token) for one Git
// operation, typically typed in by the user after Git found none.
type Credential struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type credentialKey struct{}

// WithCredential makes Git commands run with ctx offer cred to the remote.
// Git still asks the user's configured credential helpers first, and on
// success tells them to store cred, so they remember it the usual way;
// nothing is saved here. The values reach Git through its credential protocol
// in the environment, never on the command line.
func WithCredential(ctx context.Context, cred Credential) context.Context {
	return context.WithValue(ctx, credentialKey{}, cred)
}

func credentialFrom(ctx context.Context) (Credential, bool) {
	cred, ok := ctx.Value(credentialKey{}).(Credential)
	return cred, ok
}

// ValidateCredential rejects values Git's line-based credential protocol
// can't carry.
func ValidateCredential(cred Credential) error {
	if cred.Username == "" || cred.Password == "" {
		return &AppError{Message: "Enter a username and a password or token."}
	}
	if strings.ContainsAny(cred.Username+cred.Password, "\n\r\x00") {
		return &AppError{Message: "The username or password contains a line break."}
	}
	return nil
}

const (
	usernameEnv = "CURRENT_CLIENT_GIT_USERNAME"
	passwordEnv = "CURRENT_CLIENT_GIT_PASSWORD"
)

// Added after the configured helpers (-c values come last), so it answers only
// when they have nothing. It ignores store and erase. Git for Windows runs "!"
// helpers with its bundled sh.
const credentialHelper = `credential.helper=!f() { test "$1" = get || return 0; printf 'username=%s\npassword=%s\n' "$` + usernameEnv + `" "$` + passwordEnv + `"; }; f`

func credentialArgs(cred Credential) (args, env []string) {
	return []string{"-c", credentialHelper}, []string{usernameEnv + "=" + cred.Username, passwordEnv + "=" + cred.Password}
}
