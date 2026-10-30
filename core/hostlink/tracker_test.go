package hostlink

import (
	"reflect"
	"testing"
)

func TestTrackerSite(t *testing.T) {
	for raw, want := range map[string]string{
		"https://Example.atlassian.net":    "https://example.atlassian.net/",
		" https://jira.example.com/jira/ ": "https://jira.example.com/jira/",
		"http://10.0.0.5:8080/?x=1#frag":   "http://10.0.0.5:8080/",
		"https://example.atlassian.net/":   "https://example.atlassian.net/",
	} {
		if got, err := TrackerSite(raw); err != nil || got != want {
			t.Errorf("TrackerSite(%q) = %q, %v; want %q", raw, got, err, want)
		}
	}
	for _, raw := range []string{"example.atlassian.net", "ftp://x", "https://", "https://user:pw@x.net", "javascript:alert(1)"} {
		if _, err := TrackerSite(raw); err == nil {
			t.Errorf("TrackerSite(%q) accepted", raw)
		}
	}
}

func TestTrackerKeys(t *testing.T) {
	if got := ProjectKeys([]string{" proj", "OPS", "proj", "9X", "", "a-b"}); !reflect.DeepEqual(got, []string{"PROJ", "OPS"}) {
		t.Fatalf("ProjectKeys = %v", got)
	}
	any := Tracker{Site: "https://example.atlassian.net/"}
	only := Tracker{Site: any.Site, Keys: []string{"PROJ"}}
	for text, want := range map[string][2]string{
		"PROJ-123 fix login":      {"PROJ-123", "PROJ-123"},
		"feat: forms (OPS-7)":     {"OPS-7", ""},
		"feature/PROJ-9-x":        {"", ""},
		"UTF-8 handling":          {"UTF-8", ""},
		"see https://x/PROJ-1":    {"", ""},
		"MYPROJ-12":               {"MYPROJ-12", ""},
		"PROJ-0 and lower proj-5": {"", ""},
	} {
		for i, tracker := range []Tracker{any, only} {
			got := ""
			if m := tracker.KeyPattern().FindStringSubmatch(text); m != nil {
				got = m[1]
			}
			if got != want[i] {
				t.Errorf("%q with keys %v = %q, want %q", text, tracker.Keys, got, want[i])
			}
		}
	}
	for branch, want := range map[string]string{"PROJ-12-fix-login": "PROJ-12", "feature/OPS-3": "OPS-3", "12-fix": "", "xPROJ-1-a": ""} {
		got := ""
		if m := any.BranchKeyPattern().FindStringSubmatch(branch); m != nil {
			got = m[1]
		}
		if got != want {
			t.Errorf("branch %q = %q, want %q", branch, got, want)
		}
	}
	if url := any.IssueURL("PROJ-1"); url != "https://example.atlassian.net/browse/PROJ-1" {
		t.Errorf("IssueURL = %q", url)
	}
}
