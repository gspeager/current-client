package gitlab

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/gspeager/current-client/core/hostapi"
)

func TestMergeOptionsFollowTheSquashSetting(t *testing.T) {
	for option, want := range map[string][]string{
		"never":       {"merge"},
		"always":      {"squash"},
		"default_on":  {"squash", "merge"},
		"default_off": {"merge", "squash"},
	} {
		p, _ := newProvider(t, map[string]string{"GET " + api: `{"merge_method": "ff", "squash_option": "` + option + `", "remove_source_branch_after_merge": true}`})
		options, err := p.MergeOptions(context.Background(), project)
		if err != nil || !reflect.DeepEqual(options.Methods, want) || !options.DeleteBranch || options.DeleteBranchFixed {
			t.Errorf("%s: MergeOptions = %+v, %v, want methods %v", option, options, err, want)
		}
	}
}

func TestMergeSendsSquashBranchAndHead(t *testing.T) {
	p, f := newProvider(t, map[string]string{"PUT " + api + "/merge_requests/7/merge": `{"iid": 7, "state": "merged"}`})
	err := p.Merge(context.Background(), project, 7, hostapi.MergeRequest{Method: hostapi.MergeMethodSquash, DeleteBranch: true, HeadSHA: "abc"})
	if err != nil {
		t.Fatal(err)
	}
	var sent map[string]any
	json.Unmarshal([]byte(f.requests[0].Body), &sent)
	if want := map[string]any{"squash": true, "should_remove_source_branch": true, "sha": "abc"}; !reflect.DeepEqual(sent, want) {
		t.Errorf("sent %v, want %v", sent, want)
	}
}

func TestMergeRefusalsAreSaidInPlainWords(t *testing.T) {
	for status, reason := range map[string]string{"401": "may not merge", "405": "can't be merged yet", "406": "conflicts with", "409": "head moved"} {
		p, _ := newProvider(t, map[string]string{"PUT " + api + "/merge_requests/7/merge": status})
		err := p.Merge(context.Background(), project, 7, hostapi.MergeRequest{Method: hostapi.MergeMethodMerge})
		var refused *hostapi.MergeRefusedError
		if !errors.As(err, &refused) || !strings.Contains(refused.Reason, reason) || !strings.Contains(err.Error(), "gitlab.example.com refused the merge") {
			t.Errorf("%s: err = %v, want a refusal saying %q", status, err, reason)
		}
	}
	p, _ := newProvider(t, map[string]string{"PUT " + api + "/merge_requests/7/merge": "403"})
	err := p.Merge(context.Background(), project, 7, hostapi.MergeRequest{Method: hostapi.MergeMethodMerge})
	var hostErr *hostapi.HostError
	if !errors.As(err, &hostErr) || hostErr.Scope != "api" {
		t.Errorf("a 403 names the write scope: %v", err)
	}
}
