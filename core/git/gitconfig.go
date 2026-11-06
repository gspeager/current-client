package git

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/gspeager/current-client/core/gitexec"
)

// ErrConfigScopeUnsupported means Git is older than 2.26, which added --show-scope.
var ErrConfigScopeUnsupported = errors.New("showing where Git settings come from needs Git 2.26 or newer")

// ConfigEntry is one line of Git's settings, in the order Git reads them, so
// the last value of a key is the one that applies. Secrets are already redacted.
type ConfigEntry struct {
	Key    string
	Value  string
	Scope  string
	Origin string
}

// ConfigFlag is a setting that commonly explains surprising behavior. Key is
// the setting it's about.
type ConfigFlag struct {
	Key     string
	Message string
}

type ConfigReport struct {
	Entries []ConfigEntry
	Flags   []ConfigFlag
}

// InspectConfig only reads. It looks for a signing program and key rather than
// signing anything, since that can prompt for a passphrase or a hardware key.
func InspectConfig(ctx context.Context, repoPath string) (ConfigReport, error) {
	result, err := gitexec.NewExecutor("").Run(ctx, gitexec.Command{
		Dir:  repoPath,
		Args: []string{"config", "--list", "-z", "--show-scope", "--show-origin"},
	})
	if err != nil {
		return ConfigReport{}, err
	}
	switch result.ExitCode {
	case 0:
	case 129:
		return ConfigReport{}, ErrConfigScopeUnsupported
	default:
		return ConfigReport{}, gitexec.WrapResult(result)
	}
	entries := parseConfigList(result.Stdout)
	report := ConfigReport{Flags: configFlags(entries)}
	for i := range report.Flags {
		report.Flags[i].Message = gitexec.RedactSecrets(report.Flags[i].Message)
	}
	for _, entry := range entries {
		entry.Key, entry.Value = redactConfig(entry.Key, entry.Value)
		entry.Origin = gitexec.RedactSecrets(entry.Origin)
		report.Entries = append(report.Entries, entry)
	}
	return report, nil
}

// parseConfigList reads repeating scope, origin and key-then-value fields. A key
// with no value, as in "[core] bare", has no newline.
func parseConfigList(output string) []ConfigEntry {
	fields := strings.Split(output, "\x00")
	var entries []ConfigEntry
	for i := 0; i+2 < len(fields); i += 3 {
		key, value, _ := strings.Cut(fields[i+2], "\n")
		entries = append(entries, ConfigEntry{
			Key:    key,
			Value:  value,
			Scope:  fields[i],
			Origin: strings.TrimPrefix(fields[i+1], "file:"),
		})
	}
	return entries
}

var secretConfigKey = regexp.MustCompile(`(?i)(password|passwd|smtppass|token|secret|api-?key|extraheader)$`)

func redactConfig(key, value string) (string, string) {
	inlineHelper := strings.HasPrefix(key, "credential.") && strings.HasSuffix(key, ".helper") && strings.HasPrefix(value, "!")
	if value != "" && (secretConfigKey.MatchString(key) || inlineHelper) {
		return gitexec.RedactSecrets(key), gitexec.RedactedSecret
	}
	return gitexec.RedactSecrets(key), gitexec.RedactSecrets(value)
}

func configFlags(entries []ConfigEntry) []ConfigFlag {
	var flags []ConfigFlag
	flags = append(flags, identityFlags(entries, "user.name", "author name")...)
	flags = append(flags, identityFlags(entries, "user.email", "author email")...)
	if autocrlf := lastConfig(entries, "core.autocrlf"); autocrlf != nil && isTrueOrInput(autocrlf.Value) {
		flags = append(flags, ConfigFlag{"core.autocrlf", fmt.Sprintf("core.autocrlf is %s, from %s, so Git changes line endings.", autocrlf.Value, autocrlf.Origin)})
	}
	if flag, ok := signingFlag(entries); ok {
		flags = append(flags, flag)
	}
	if !hasCredentialHelper(entries) {
		flags = append(flags, ConfigFlag{"credential.helper", "No credential helper is set, so HTTPS remotes can't remember a sign-in."})
	}
	if path := globalIgnoreFile(entries); path != "" {
		flags = append(flags, ConfigFlag{"core.excludesfile", fmt.Sprintf("A global ignore file applies, so files it matches don't show in Changes: %s", path)})
	}
	return flags
}

func identityFlags(entries []ConfigEntry, key, name string) []ConfigFlag {
	last := lastConfig(entries, key)
	if last == nil || last.Value == "" {
		return []ConfigFlag{{key, fmt.Sprintf("No %s is set, so committing fails.", name)}}
	}
	for _, entry := range entries {
		if entry.Key == key && entry.Value != last.Value {
			return []ConfigFlag{{key, fmt.Sprintf("The %s is %s, from %s, over %s, from %s.", name, last.Value, last.Origin, entry.Value, entry.Origin)}}
		}
	}
	return nil
}

func signingFlag(entries []ConfigEntry) (ConfigFlag, bool) {
	on := ""
	for _, key := range []string{"commit.gpgsign", "tag.gpgsign"} {
		if setting := lastConfig(entries, key); setting != nil && isConfigTrue(setting.Value) && on == "" {
			on = key
		}
	}
	if on == "" {
		return ConfigFlag{}, false
	}
	format := "openpgp"
	if setting := lastConfig(entries, "gpg.format"); setting != nil && setting.Value != "" {
		format = setting.Value
	}
	program, known := map[string]string{"openpgp": "gpg", "ssh": "ssh-keygen", "x509": "gpgsm"}[format]
	if !known {
		return ConfigFlag{"gpg.format", fmt.Sprintf("Signing is on, but gpg.format isn't one Git knows: %s.", format)}, true
	}
	programKeys := []string{"gpg." + format + ".program"}
	if format == "openpgp" {
		programKeys = append(programKeys, "gpg.program")
	}
	for _, key := range programKeys {
		if setting := lastConfig(entries, key); setting != nil && setting.Value != "" {
			program = setting.Value
			break
		}
	}
	if _, err := gitexec.LookPath(expandHome(program)); err != nil {
		return ConfigFlag{on, fmt.Sprintf("Signing is on, but its program isn't found: %s.", program)}, true
	}
	if format != "ssh" || lastConfig(entries, "gpg.ssh.defaultkeycommand") != nil {
		return ConfigFlag{}, false
	}
	key := lastConfig(entries, "user.signingkey")
	if key == nil || key.Value == "" {
		return ConfigFlag{on, "SSH signing is on, but user.signingkey isn't set."}, true
	}
	if strings.HasPrefix(key.Value, "key::") || strings.HasPrefix(key.Value, "ssh-") {
		return ConfigFlag{}, false
	}
	if _, err := os.Stat(expandHome(key.Value)); err != nil {
		return ConfigFlag{on, fmt.Sprintf("SSH signing is on, but its key isn't found: %s.", key.Value)}, true
	}
	return ConfigFlag{}, false
}

// An empty value clears the helpers set before it. A helper for one host counts,
// since which hosts matter depends on the repository's remotes.
func hasCredentialHelper(entries []ConfigEntry) bool {
	last := map[string]string{}
	for _, entry := range entries {
		if strings.HasPrefix(entry.Key, "credential.") && strings.HasSuffix(entry.Key, ".helper") {
			last[entry.Key] = entry.Value
		}
	}
	for _, value := range last {
		if value != "" {
			return true
		}
	}
	return false
}

// globalIgnoreFile returns the file core.excludesfile names, or Git's default
// one, when it exists.
func globalIgnoreFile(entries []ConfigEntry) string {
	path := ""
	if setting := lastConfig(entries, "core.excludesfile"); setting != nil && setting.Value != "" {
		path = expandHome(setting.Value)
	} else if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		path = filepath.Join(xdg, "git", "ignore")
	} else if home, err := os.UserHomeDir(); err == nil {
		path = filepath.Join(home, ".config", "git", "ignore")
	}
	if info, err := os.Stat(path); path == "" || err != nil || info.IsDir() {
		return ""
	}
	return path
}

func lastConfig(entries []ConfigEntry, key string) *ConfigEntry {
	for i := len(entries) - 1; i >= 0; i-- {
		if entries[i].Key == key {
			return &entries[i]
		}
	}
	return nil
}

// A key with no value, as in "[commit] gpgsign", is true.
func isConfigTrue(value string) bool {
	switch strings.ToLower(value) {
	case "", "true", "yes", "on", "1":
		return true
	}
	return false
}

func isTrueOrInput(value string) bool {
	return isConfigTrue(value) || strings.EqualFold(value, "input")
}

func expandHome(path string) string {
	if rest, ok := strings.CutPrefix(path, "~/"); ok {
		if home, err := os.UserHomeDir(); err == nil {
			return filepath.Join(home, rest)
		}
	}
	return path
}
