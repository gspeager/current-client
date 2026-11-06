package app

import (
	"net/url"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/gspeager/current-client/internal/config"
	"github.com/wailsapp/wails/v3/pkg/application"
)

func link(path string) string {
	return "current-client://open?path=" + url.QueryEscape(path)
}

func TestParseOpenArg(t *testing.T) {
	useTempSettings(t)
	abs := t.TempDir()
	known := filepath.Join(abs, "known")
	if err := updateConfig(func(cfg *config.Config) { cfg.RecentRepositories = []string{known} }); err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		name, arg string
		want      OpenRequestInfo
	}{
		{"an absolute path", abs, OpenRequestInfo{Path: abs}},
		{"a path relative to where it was run", "sub/repo", OpenRequestInfo{Path: filepath.Join(abs, "sub", "repo")}},
		{"a file manager's file URL", "file://" + filepath.ToSlash(abs), OpenRequestInfo{Path: abs}},
		{"a link to a known repository", link(known), OpenRequestInfo{Path: known}},
		{"a link to an unknown repository asks", link(filepath.Join(abs, "other")), OpenRequestInfo{Path: filepath.Join(abs, "other"), Confirm: true}},
		{"a link's path is cleaned before it's matched", link(known + "/x/.."), OpenRequestInfo{Path: known}},
		{"a network path by argument is the user's own choice", `\\host\share\repo`, OpenRequestInfo{Path: `\\host\share\repo`}},
	}
	if runtime.GOOS != "windows" {
		cases[6].want.Path = filepath.Clean(`\\host\share\repo`)
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, ok := parseOpenArg(c.arg, abs)
			if !ok || got != c.want {
				t.Fatalf("parseOpenArg(%q) = %+v, %v, want %+v", c.arg, got, ok, c.want)
			}
		})
	}
}

func TestParseOpenArgRefusesWhatALinkMayNotDo(t *testing.T) {
	useTempSettings(t)
	problems := []string{
		"current-client://",
		"current-client://clone?url=https://example.com/a.git",
		"current-client://run?command=x",
		"current-client://open",
		"current-client://open?path=relative/path",
		"current-client://open?path=%zz",
		"current-client:open",
		link(`\\attacker\share\repo`),
		link("//attacker/share/repo"),
	}
	for _, arg := range problems {
		got, ok := parseOpenArg(arg, "/work")
		if !ok || got.Problem == "" || got.Path != "" {
			t.Errorf("parseOpenArg(%q) = %+v, %v", arg, got, ok)
		}
	}
	if got, _ := parseOpenArg(link(`\\attacker\share`), "/work"); !strings.Contains(got.Problem, "network") {
		t.Errorf("network path problem = %q", got.Problem)
	}
}

func TestParseOpenArgIgnoresFlags(t *testing.T) {
	for _, arg := range []string{"", "--help", "-psn_0_12345"} {
		if _, ok := parseOpenArg(arg, "/work"); ok {
			t.Errorf("%q should ask for nothing", arg)
		}
	}
}

func TestOpenRequestsQueueOnceAndAreTakenOnce(t *testing.T) {
	useTempSettings(t)
	abs := t.TempDir()
	var s RepositoryService
	s.TakeOpenRequests()

	QueueOpenArgs([]string{abs, "--flag", abs}, "/work")
	taken := s.TakeOpenRequests()
	if len(taken) != 1 || taken[0].Path != abs {
		t.Fatalf("taken = %+v", taken)
	}
	if again := s.TakeOpenRequests(); again == nil || len(again) != 0 {
		t.Fatalf("second take = %+v", again)
	}
}

func TestSingleInstanceHandsOverArgumentsWithoutTheProgramName(t *testing.T) {
	useTempSettings(t)
	abs := t.TempDir()
	var s RepositoryService
	s.TakeOpenRequests()
	options := SingleInstance()
	if !strings.HasPrefix(options.UniqueID, "io.github.gspeager.current-client.") {
		t.Fatalf("UniqueID = %q", options.UniqueID)
	}
	options.OnSecondInstanceLaunch(application.SecondInstanceData{Args: []string{"current-client", abs}, WorkingDir: "/work"})
	if taken := s.TakeOpenRequests(); len(taken) != 1 || taken[0].Path != abs {
		t.Fatalf("taken = %+v", taken)
	}
	options.OnSecondInstanceLaunch(application.SecondInstanceData{Args: []string{"current-client"}})
	if taken := s.TakeOpenRequests(); len(taken) != 0 {
		t.Fatalf("a launch with no arguments asks for nothing: %+v", taken)
	}
	first := options.UniqueID
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", "")
	if SingleInstance().UniqueID == first {
		t.Fatal("another HOME should run its own instance")
	}
}
