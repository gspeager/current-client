package git

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gspeager/current-client/core/internal/gittest"
)

// isolatedConfigRepo has no settings of its own beyond what the test adds, and
// a HOME that holds only globalConfig.
func isolatedConfigRepo(t *testing.T, globalConfig string) (repo, home string) {
	t.Helper()
	home = t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", "")
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	t.Setenv("GIT_CONFIG_GLOBAL", "")
	os.Unsetenv("GIT_CONFIG_GLOBAL")
	if err := os.WriteFile(filepath.Join(home, ".gitconfig"), []byte(globalConfig), 0o644); err != nil {
		t.Fatal(err)
	}
	repo = t.TempDir()
	gittest.Run(t, repo, "init", "-b", "main")
	return repo, home
}

func inspect(t *testing.T, repo string) ConfigReport {
	t.Helper()
	report, err := InspectConfig(context.Background(), repo)
	if err != nil {
		t.Fatal(err)
	}
	return report
}

func flagFor(report ConfigReport, key string) string {
	for _, flag := range report.Flags {
		if flag.Key == key {
			return flag.Message
		}
	}
	return ""
}

const namedUser = "[user]\n\tname = Ada\n\temail = ada@example.com\n"

func TestInspectConfigNamesScopeAndOrigin(t *testing.T) {
	repo, _ := isolatedConfigRepo(t, namedUser)
	gittest.Run(t, repo, "config", "core.quotepath", "off")
	gittest.Run(t, repo, "config", "multi.line", "one\ntwo")
	gittest.Run(t, repo, "config", "--add", "core.editor", "vim")

	report := inspect(t, repo)
	global := lastConfig(report.Entries, "user.name")
	if global == nil || global.Scope != "global" || global.Value != "Ada" || !strings.HasSuffix(global.Origin, ".gitconfig") {
		t.Fatalf("user.name = %+v", global)
	}
	local := lastConfig(report.Entries, "core.quotepath")
	if local == nil || local.Scope != "local" || local.Origin != ".git/config" {
		t.Fatalf("core.quotepath = %+v", local)
	}
	if multi := lastConfig(report.Entries, "multi.line"); multi == nil || multi.Value != "one\ntwo" {
		t.Fatalf("multi.line = %+v", multi)
	}
}

func TestInspectConfigChangesNothing(t *testing.T) {
	repo, _ := isolatedConfigRepo(t, namedUser)
	path := filepath.Join(repo, ".git", "config")
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	inspect(t, repo)
	after, err := os.ReadFile(path)
	if err != nil || string(after) != string(before) {
		t.Fatalf("config changed: %v\n%s", err, after)
	}
}

func TestInspectConfigFlagsIdentity(t *testing.T) {
	repo, home := isolatedConfigRepo(t, "")
	report := inspect(t, repo)
	if flagFor(report, "user.name") == "" || flagFor(report, "user.email") == "" {
		t.Fatalf("no identity should be flagged: %+v", report.Flags)
	}

	work := filepath.Join(home, "work.gitconfig")
	if err := os.WriteFile(work, []byte("[user]\n\temail = ada@work.example\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	global := namedUser + "[includeIf \"gitdir:**/\"]\n\tpath = work.gitconfig\n"
	if err := os.WriteFile(filepath.Join(home, ".gitconfig"), []byte(global), 0o644); err != nil {
		t.Fatal(err)
	}
	report = inspect(t, repo)
	flag := flagFor(report, "user.email")
	if !strings.Contains(flag, "ada@work.example") || !strings.Contains(flag, "work.gitconfig") || !strings.Contains(flag, "ada@example.com") {
		t.Fatalf("includeIf's identity flag = %q", flag)
	}
	if flagFor(report, "user.name") != "" {
		t.Fatalf("an unchanged name shouldn't be flagged: %q", flagFor(report, "user.name"))
	}
}

func TestInspectConfigFlagsAutocrlf(t *testing.T) {
	repo, _ := isolatedConfigRepo(t, namedUser)
	gittest.Run(t, repo, "config", "core.autocrlf", "false")
	if flag := flagFor(inspect(t, repo), "core.autocrlf"); flag != "" {
		t.Fatalf("autocrlf false flagged: %q", flag)
	}
	for _, value := range []string{"true", "input"} {
		gittest.Run(t, repo, "config", "core.autocrlf", value)
		if flag := flagFor(inspect(t, repo), "core.autocrlf"); !strings.Contains(flag, value) {
			t.Fatalf("autocrlf %s flag = %q", value, flag)
		}
	}
}

func TestInspectConfigFlagsSigning(t *testing.T) {
	repo, home := isolatedConfigRepo(t, namedUser)
	if flag := flagFor(inspect(t, repo), "commit.gpgsign"); flag != "" {
		t.Fatalf("signing off flagged: %q", flag)
	}
	gittest.Run(t, repo, "config", "commit.gpgsign", "true")
	gittest.Run(t, repo, "config", "gpg.program", filepath.Join(home, "missing-gpg"))
	if flag := flagFor(inspect(t, repo), "commit.gpgsign"); !strings.Contains(flag, "missing-gpg") {
		t.Fatalf("missing program flag = %q", flag)
	}

	present := filepath.Join(home, "signer")
	if err := os.WriteFile(present, []byte("#!/bin/sh\ntouch "+filepath.Join(home, "signed")+"\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	gittest.Run(t, repo, "config", "gpg.program", present)
	if flag := flagFor(inspect(t, repo), "commit.gpgsign"); flag != "" {
		t.Fatalf("a program that exists flagged: %q", flag)
	}
	if _, err := os.Stat(filepath.Join(home, "signed")); err == nil {
		t.Fatal("inspecting ran the signing program")
	}

	gittest.Run(t, repo, "config", "gpg.format", "ssh")
	gittest.Run(t, repo, "config", "gpg.ssh.program", present)
	if flag := flagFor(inspect(t, repo), "commit.gpgsign"); !strings.Contains(flag, "user.signingkey") {
		t.Fatalf("no SSH key flag = %q", flag)
	}
	gittest.Run(t, repo, "config", "user.signingkey", "~/missing_key.pub")
	if flag := flagFor(inspect(t, repo), "commit.gpgsign"); !strings.Contains(flag, "missing_key") {
		t.Fatalf("missing SSH key flag = %q", flag)
	}
	gittest.Run(t, repo, "config", "user.signingkey", "key::ssh-ed25519 AAAA")
	if flag := flagFor(inspect(t, repo), "commit.gpgsign"); flag != "" {
		t.Fatalf("a literal SSH key flagged: %q", flag)
	}
}

func TestInspectConfigFlagsNoCredentialHelper(t *testing.T) {
	repo, _ := isolatedConfigRepo(t, namedUser)
	if flagFor(inspect(t, repo), "credential.helper") == "" {
		t.Fatal("no helper should be flagged")
	}
	gittest.Run(t, repo, "config", "credential.https://git.example.com.helper", "store")
	if flag := flagFor(inspect(t, repo), "credential.helper"); flag != "" {
		t.Fatalf("a host's helper flagged: %q", flag)
	}
}

func TestInspectConfigFlagsGlobalIgnoreFile(t *testing.T) {
	repo, home := isolatedConfigRepo(t, namedUser)
	if flag := flagFor(inspect(t, repo), "core.excludesfile"); flag != "" {
		t.Fatalf("no ignore file flagged: %q", flag)
	}

	byDefault := filepath.Join(home, ".config", "git", "ignore")
	if err := os.MkdirAll(filepath.Dir(byDefault), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(byDefault, []byte("*.log\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if flag := flagFor(inspect(t, repo), "core.excludesfile"); !strings.Contains(flag, byDefault) {
		t.Fatalf("default ignore file flag = %q", flag)
	}

	named := filepath.Join(home, "ignores")
	if err := os.WriteFile(named, []byte("*.tmp\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	gittest.Run(t, repo, "config", "core.excludesfile", "~/ignores")
	if flag := flagFor(inspect(t, repo), "core.excludesfile"); !strings.Contains(flag, named) {
		t.Fatalf("named ignore file flag = %q", flag)
	}
}

func TestInspectConfigRedactsCredentials(t *testing.T) {
	repo, _ := isolatedConfigRepo(t, namedUser)
	gittest.Run(t, repo, "config", "remote.origin.url", "https://deploy:s3cret@git.example.com/a.git")
	gittest.Run(t, repo, "config", "http.https://git.example.com/.extraheader", "AUTHORIZATION: basic s3cret")
	gittest.Run(t, repo, "config", "url.https://x:s3cret@git.example.com/.insteadof", "https://git.example.com/")
	gittest.Run(t, repo, "config", "credential.helper", "!f() { echo password=s3cret; }; f")
	gittest.Run(t, repo, "config", "tool.apitoken", "s3cret")
	gittest.Run(t, repo, "config", "alias.lg", `!f() { git log --format="%h %s"; }; f`)

	report := inspect(t, repo)
	for _, entry := range report.Entries {
		if strings.Contains(entry.Key+entry.Value+entry.Origin, "s3cret") {
			t.Fatalf("leaked in %+v", entry)
		}
	}
	for _, flag := range report.Flags {
		if strings.Contains(flag.Message, "s3cret") {
			t.Fatalf("leaked in %+v", flag)
		}
	}
	if origin := lastConfig(report.Entries, "remote.origin.url"); origin == nil || origin.Value != "https://<token>@git.example.com/a.git" {
		t.Fatalf("remote.origin.url = %+v", origin)
	}
	if alias := lastConfig(report.Entries, "alias.lg"); alias == nil || !strings.Contains(alias.Value, `--format="%h %s"`) {
		t.Fatalf("an ordinary alias should be left alone: %+v", alias)
	}
}

func TestInspectConfigReadsAKeyWithNoValue(t *testing.T) {
	entries := parseConfigList("local\x00file:.git/config\x00commit.gpgsign\x00global\x00file:/h/.gitconfig\x00core.editor\nvim\x00")
	if len(entries) != 2 || entries[0].Key != "commit.gpgsign" || entries[0].Value != "" || entries[1].Value != "vim" || entries[1].Origin != "/h/.gitconfig" {
		t.Fatalf("entries = %+v", entries)
	}
	if !isConfigTrue(entries[0].Value) {
		t.Fatal("a key with no value is true")
	}
}
