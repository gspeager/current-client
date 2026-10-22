package hostlink

import (
	"context"
	"os/exec"
	"strings"
)

// Resolve is Parse with an SSH host alias replaced by the hostname ssh would
// connect to, read from ~/.ssh/config through `ssh -G`, which prints the
// effective configuration without connecting. HTTPS remotes are never
// resolved, and a failed lookup keeps the alias.
func Resolve(ctx context.Context, remoteURL string) (Host, bool) {
	host, ok := Parse(remoteURL)
	if !ok || !isSSH(remoteURL) {
		return host, ok
	}
	alias := host.Hostname()
	hostname, err := sshHostname(ctx, alias)
	if err != nil || hostname == "" || strings.EqualFold(hostname, alias) {
		return host, true
	}
	hostname = strings.ToLower(hostname)
	host.Root = "https://" + hostname + "/"
	host.Style = styleFor(hostname, host.Project)
	return host, true
}

func isSSH(remoteURL string) bool {
	scheme, _, found := strings.Cut(strings.TrimSpace(remoteURL), "://")
	if !found {
		return true
	}
	switch scheme {
	case "ssh", "git+ssh", "ssh+git":
		return true
	}
	return false
}

// sshHostname is replaced in tests.
var sshHostname = func(ctx context.Context, alias string) (string, error) {
	return sshConfigHostname(ctx, alias)
}

// sshConfigHostname passes the alias after "--" so a hostname from a remote
// URL can never be read as an ssh option.
func sshConfigHostname(ctx context.Context, alias string, sshArgs ...string) (string, error) {
	args := append([]string{"-G"}, sshArgs...)
	args = append(args, "--", alias)
	out, err := exec.CommandContext(ctx, "ssh", args...).Output()
	if err != nil {
		return "", err
	}
	return parseSSHHostname(string(out)), nil
}

func parseSSHHostname(output string) string {
	for _, line := range strings.Split(output, "\n") {
		if key, value, ok := strings.Cut(line, " "); ok && key == "hostname" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
