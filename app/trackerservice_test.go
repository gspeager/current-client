package app

import (
	"os"
	"reflect"
	"regexp"
	"testing"

	"github.com/gspeager/current-client/internal/config"
	"github.com/gspeager/current-client/internal/policy"
)

func TestTrackersAddAssignAndRemove(t *testing.T) {
	useTempSettings(t)
	updateConfig(func(cfg *config.Config) { cfg.OpenTabs = []string{"/r/app", "/r/api"} })
	var s TrackerService

	if s.Links("/r/app") != nil {
		t.Fatal("no tracker yet")
	}
	if _, err := s.Add("example.atlassian.net", "jira-cloud", nil); err == nil {
		t.Fatal("an address without a scheme was kept")
	}
	if _, err := s.Add("https://example.atlassian.net", "linear", nil); err == nil {
		t.Fatal("an unknown kind was kept")
	}
	cloud, err := s.Add("https://Example.atlassian.net/", "jira-cloud", []string{"proj", "utf"})
	if err != nil || cloud.Site != "https://example.atlassian.net/" || !reflect.DeepEqual(cloud.Keys, []string{"PROJ", "UTF"}) {
		t.Fatalf("Add = %+v, %v", cloud, err)
	}
	if _, err := s.Add("https://example.atlassian.net", "jira-cloud", nil); err == nil {
		t.Fatal("the same site twice")
	}
	dc, _ := s.Add("https://jira.example.com/jira", "jira-data-center", nil)
	s.SetKeys(cloud.Site, []string{"PROJ"})
	s.SetRepos(cloud.Site, []string{"/r/app", "/r/api"})
	s.SetRepos(dc.Site, []string{"/r/api"})

	list := s.List()
	if !reflect.DeepEqual(list.OpenRepos, []string{"/r/app", "/r/api"}) || len(list.Trackers) != 2 || list.Default != nil {
		t.Fatalf("List = %+v", list)
	}
	if !reflect.DeepEqual(list.Trackers[0].Repos, []string{"/r/app"}) || !reflect.DeepEqual(list.Trackers[1].Repos, []string{"/r/api"}) {
		t.Fatalf("a repository should move to the tracker it was last given: %+v", list.Trackers)
	}
	links := s.Links("/r/app")
	if links == nil || links.Browse != "https://example.atlassian.net/browse/" || links.Managed {
		t.Fatalf("Links = %+v", links)
	}
	if m := regexp.MustCompile(links.KeyPattern).FindStringSubmatch("fix: login (PROJ-12)"); m == nil || m[1] != "PROJ-12" {
		t.Fatalf("KeyPattern %q missed PROJ-12", links.KeyPattern)
	}
	if regexp.MustCompile(links.KeyPattern).MatchString("UTF-8") {
		t.Fatal("keys should limit what links")
	}
	if links := s.Links("/r/api"); links == nil || links.Site != dc.Site {
		t.Fatalf("api's tracker = %+v", links)
	}

	if err := s.Remove(cloud.Site); err != nil {
		t.Fatal(err)
	}
	if s.Links("/r/app") != nil || len(s.List().Trackers) != 1 {
		t.Fatal("removing a tracker unlinks its repositories")
	}
}

func TestPolicyTrackerIsTheDefault(t *testing.T) {
	useTempSettings(t)
	os.WriteFile(policy.Path(), []byte(`{"tracker": {"site": "https://corp.atlassian.net", "kind": "jira-cloud", "keys": ["OPS"]}}`), 0o644)
	if err := policy.Load(); err != nil {
		t.Fatal(err)
	}
	var s TrackerService
	list := s.List()
	if list.Default == nil || !list.Default.Managed || list.Default.Site != "https://corp.atlassian.net/" || !reflect.DeepEqual(list.Default.Keys, []string{"OPS"}) {
		t.Fatalf("Default = %+v", list.Default)
	}
	if links := s.Links("/r/any"); links == nil || !links.Managed || links.Site != "https://corp.atlassian.net/" {
		t.Fatalf("a repository without its own tracker uses the default: %+v", links)
	}
	own, _ := s.Add("https://team.atlassian.net", "jira-cloud", nil)
	s.SetRepos(own.Site, []string{"/r/mine"})
	if links := s.Links("/r/mine"); links.Site != own.Site || links.Managed {
		t.Fatalf("a repository's own tracker wins: %+v", links)
	}
}
