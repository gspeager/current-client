package repository

import (
	"reflect"
	"testing"
)

func TestAddRecentPrependsNewPath(t *testing.T) {
	got := AddRecent([]string{"/b", "/c"}, "/a")

	want := []string{"/a", "/b", "/c"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestAddRecentMovesExistingPathToFront(t *testing.T) {
	got := AddRecent([]string{"/a", "/b", "/c"}, "/b")

	want := []string{"/b", "/a", "/c"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestAddRecentCapsAtMax(t *testing.T) {
	var recent []string
	for i := 0; i < maxRecentRepositories+5; i++ {
		recent = AddRecent(recent, string(rune('a'+i)))
	}

	if len(recent) != maxRecentRepositories {
		t.Fatalf("len = %d, want %d", len(recent), maxRecentRepositories)
	}
}

func TestRemoveRecentDropsOnlyThatPath(t *testing.T) {
	got := RemoveRecent([]string{"/a", "/b", "/c"}, "/b")

	want := []string{"/a", "/c"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}
