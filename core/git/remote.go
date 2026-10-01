package git

import (
	"context"
	"strings"
)

type Remote struct {
	Name     string
	FetchURL string
	PushURL  string
}

func ListRemotes(ctx context.Context, repoPath string) ([]Remote, error) {
	result, err := runResult(ctx, repoPath, "remote", "-v")
	if err != nil {
		return nil, err
	}
	return parseRemotes(result.Stdout), nil
}

func AddRemote(ctx context.Context, repoPath, name, url string) error {
	_, err := runResult(ctx, repoPath, "remote", "add", name, url)
	return err
}

func RemoveRemote(ctx context.Context, repoPath, name string) error {
	_, err := runResult(ctx, repoPath, "remote", "remove", name)
	return err
}

func EditRemote(ctx context.Context, repoPath, name, url string) error {
	_, err := runResult(ctx, repoPath, "remote", "set-url", name, url)
	return err
}

func parseRemotes(output string) []Remote {
	byName := make(map[string]*Remote)
	var order []string
	for _, line := range strings.Split(output, "\n") {
		fields := strings.Fields(line)
		if len(fields) < 3 {
			continue
		}
		name, url, kind := fields[0], fields[1], strings.Trim(fields[2], "()")
		r, ok := byName[name]
		if !ok {
			r = &Remote{Name: name}
			byName[name] = r
			order = append(order, name)
		}
		switch kind {
		case "fetch":
			r.FetchURL = url
		case "push":
			r.PushURL = url
		}
	}
	remotes := make([]Remote, len(order))
	for i, name := range order {
		remotes[i] = *byName[name]
	}
	return remotes
}
