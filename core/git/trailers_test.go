package git

import (
	"context"
	"reflect"
	"testing"

	"github.com/gspeager/current-client/core/internal/gittest"
)

func TestParseTrailers(t *testing.T) {
	tests := []struct {
		name string
		body string
		want []Trailer
	}{
		{"none", "", nil},
		{"prose only", "Explains the change.", nil},
		{
			"several co-authors",
			"Explains the change.\n\nCo-authored-by: Ada Lovelace <ada@example.com>\nCo-authored-by: Grace Hopper <grace@example.com>",
			[]Trailer{{"Co-authored-by", "Ada Lovelace <ada@example.com>"}, {"Co-authored-by", "Grace Hopper <grace@example.com>"}},
		},
		{
			"mixed trailers",
			"Signed-off-by: Ada Lovelace <ada@example.com>\nRefs: #12\nCo-authored-by: Grace Hopper <grace@example.com>",
			[]Trailer{{"Signed-off-by", "Ada Lovelace <ada@example.com>"}, {"Refs", "#12"}, {"Co-authored-by", "Grace Hopper <grace@example.com>"}},
		},
		{
			"trailer-like line mid-body",
			"Note: this is prose.\n\nMore prose after it.",
			nil,
		},
		{
			"last paragraph mixes prose and a trailer",
			"Fixes the thing.\nRefs: #12",
			nil,
		},
		{
			"continuation line",
			"Body.\n\nBREAKING-CHANGE: the old flag\n  is gone\nRefs: #3",
			[]Trailer{{"BREAKING-CHANGE", "the old flag is gone"}, {"Refs", "#3"}},
		},
		{"CRLF", "Body.\r\n\r\nRefs: #4\r\n", []Trailer{{"Refs", "#4"}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ParseTrailers(tt.body); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ParseTrailers = %#v, want %#v", got, tt.want)
			}
		})
	}
}

func TestCoAuthors(t *testing.T) {
	trailers := []Trailer{
		{"Signed-off-by", "Ada Lovelace <ada@example.com>"},
		{"co-authored-by", "Grace Hopper <grace@example.com>"},
		{"Co-authored-by", "Just A Name"},
	}
	want := []Author{{Name: "Grace Hopper", Email: "grace@example.com"}, {Name: "Just A Name"}}
	if got := CoAuthors(trailers); !reflect.DeepEqual(got, want) {
		t.Errorf("CoAuthors = %#v, want %#v", got, want)
	}
}

func TestHistoryReadsTrailers(t *testing.T) {
	dir := gittest.InitRepo(t)
	gittest.WriteFile(t, dir, "a.txt", "one\n")
	gittest.Run(t, dir, "add", "a.txt")
	gittest.Run(t, dir, "commit", "-m", "feat: pair work", "-m", "Co-authored-by: Grace Hopper <grace@example.com>")

	entries, err := History(context.Background(), dir, 1, 0, HistoryFilter{})
	if err != nil {
		t.Fatalf("History: %v", err)
	}
	want := []Trailer{{"Co-authored-by", "Grace Hopper <grace@example.com>"}}
	if len(entries) != 1 || !reflect.DeepEqual(entries[0].Trailers, want) {
		t.Errorf("Trailers = %#v, want %#v", entries, want)
	}
}
