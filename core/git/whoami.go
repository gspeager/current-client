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
	return UserIdentity{
		Name:  configValue(ctx, repoPath, "user.name"),
		Email: configValue(ctx, repoPath, "user.email"),
	}
}

func GlobalUser(ctx context.Context) UserIdentity {
	return UserIdentity{
		Name:  configValue(ctx, "", "--global", "user.name"),
		Email: configValue(ctx, "", "--global", "user.email"),
	}
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
