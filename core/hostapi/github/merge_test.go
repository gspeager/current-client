package github

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/gspeager/current-client/core/hostapi"
)

func TestMergeOptionsFollowTheRepositorySettings(t *testing.T) {
	p, _ := newProvider(t, map[string]string{"GET " + v3: `{"allow_merge_commit": false, "allow_squash_merge": true, "allow_rebase_merge": true, "delete_branch_on_merge": true}`})
	options, err := p.MergeOptions(context.Background(), project)
	if err != nil || !reflect.DeepEqual(options.Methods, []string{"squash", "rebase"}) || !options.DeleteBranch || !options.DeleteBranchFixed {
		t.Fatalf("MergeOptions = %+v, %v", options, err)
	}
}

func TestMergeSendsMethodAndHead(t *testing.T) {
	p, f := newProvider(t, map[string]string{"PUT " + v3 + "/pulls/7/merge": `{"merged": true}`})
	if err := p.Merge(context.Background(), project, 7, hostapi.MergeRequest{Method: hostapi.MergeMethodSquash, HeadSHA: "abc"}); err != nil {
		t.Fatal(err)
	}
	var sent map[string]any
	json.Unmarshal([]byte(f.requests[0].Body), &sent)
	if want := map[string]any{"merge_method": "squash", "sha": "abc"}; !reflect.DeepEqual(sent, want) {
		t.Errorf("sent %v, want %v", sent, want)
	}
}

func TestMergeRefusalsAreSaidInPlainWords(t *testing.T) {
	for status, reason := range map[string]string{"405": "isn't mergeable yet", "409": "head moved", "422": "isn't allowed"} {
		p, _ := newProvider(t, map[string]string{"PUT " + v3 + "/pulls/7/merge": status})
		err := p.Merge(context.Background(), project, 7, hostapi.MergeRequest{Method: hostapi.MergeMethodMerge})
		var refused *hostapi.MergeRefusedError
		if !errors.As(err, &refused) || !strings.Contains(refused.Reason, reason) {
			t.Errorf("%s: err = %v, want a refusal saying %q", status, err, reason)
		}
	}
}
