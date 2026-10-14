package git

import (
	"context"
	"strings"

	"github.com/gspeager/current-client/core/gitexec"
)

type UserIdentity struct {
	Name  string
	Email string
}

func CurrentUser(ctx context.Context, repoPath string) UserIdentity {
	return userIdentity(ctx, repoPath)
}

func GlobalUser(ctx context.Context) UserIdentity {
	return userIdentity(ctx, "", "--global")
}

// userIdentity reads both keys in one process. Like --get, the last value
// wins when several scopes set one.
func userIdentity(ctx context.Context, repoPath string, scope ...string) UserIdentity {
	args := append([]string{"config", "-z"}, scope...)
	result, err := gitexec.NewExecutor("").Run(ctx, gitexec.Command{
		Dir:  repoPath,
		Args: append(args, "--get-regexp", `^user\.(name|email)$`),
	})
	var user UserIdentity
	if err != nil || result.ExitCode != 0 {
		return user
	}
	for _, entry := range strings.Split(result.Stdout, "\x00") {
		key, value, _ := strings.Cut(entry, "\n")
		switch key {
		case "user.name":
			user.Name = value
		case "user.email":
			user.Email = value
		}
	}
	return user
}

func SetGlobalUser(ctx context.Context, name, email string) error {
	return setUser(ctx, "", "--global", name, email)
}

func SetRepoUser(ctx context.Context, repoPath, name, email string) error {
	return setUser(ctx, repoPath, "--local", name, email)
}

func setUser(ctx context.Context, repoPath, scope, name, email string) error {
	if _, err := runResult(ctx, repoPath, "config", scope, "user.name", name); err != nil {
		return err
	}
	_, err := runResult(ctx, repoPath, "config", scope, "user.email", email)
	return err
}

// configValue returns "" for an unset key rather than an error.
func configValue(ctx context.Context, repoPath string, args ...string) string {
	result, err := gitexec.NewExecutor("").Run(ctx, gitexec.Command{
		Dir:  repoPath,
		Args: append([]string{"config", "--get"}, args...),
	})
	if err != nil || result.ExitCode != 0 {
		return ""
	}
	return strings.TrimSpace(result.Stdout)
}

// CredentialHelper returns the credential helper Git will store sign-ins with
// (the last one configured), or "" if there is none.
func CredentialHelper(ctx context.Context, repoPath string) string {
	return configValue(ctx, repoPath, "credential.helper")
}
