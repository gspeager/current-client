package identity

import (
	"regexp"
	"testing"
)

func TestInitialsFullName(t *testing.T) {
	if got := Initials("Alex Rivera", "alex.rivera@pulse-engine.io"); got != "AR" {
		t.Fatalf("got %q, want %q", got, "AR")
	}
}

func TestInitialsFullNameMiddleNameIgnored(t *testing.T) {
	if got := Initials("Mara Ellen Chen", "mchen@example.com"); got != "MC" {
		t.Fatalf("got %q, want %q (first + last word, middle skipped)", got, "MC")
	}
}

func TestInitialsEmailOnly(t *testing.T) {
	if got := Initials("", "mara.chen@example.com"); got != "MC" {
		t.Fatalf("got %q, want %q", got, "MC")
	}
}

func TestInitialsSingleWordName(t *testing.T) {
	if got := Initials("Madonna", ""); got != "MA" {
		t.Fatalf("got %q, want %q", got, "MA")
	}
}

func TestInitialsSingleWordEmailLocalPart(t *testing.T) {
	if got := Initials("", "renovate@bots.example.com"); got != "RE" {
		t.Fatalf("got %q, want %q", got, "RE")
	}
}

func TestInitialsEmptyIdentity(t *testing.T) {
	if got := Initials("", ""); got != "?" {
		t.Fatalf("got %q, want %q", got, "?")
	}
}

var hexColorPattern = regexp.MustCompile(`^#[0-9a-f]{6}$`)

func TestDefaultColorDeterministic(t *testing.T) {
	a := DefaultColor("Alex Rivera", "alex.rivera@pulse-engine.io")
	b := DefaultColor("Alex Rivera", "alex.rivera@pulse-engine.io")
	if a != b {
		t.Fatalf("same identity produced different colors: %q vs %q", a, b)
	}
	if !hexColorPattern.MatchString(a) {
		t.Fatalf("color %q is not a 6-digit hex color", a)
	}
}

func TestDefaultColorVariesByIdentity(t *testing.T) {
	a := DefaultColor("Alex Rivera", "alex.rivera@pulse-engine.io")
	b := DefaultColor("Mara Chen", "mara.chen@example.com")
	if a == b {
		t.Fatalf("distinct identities produced the same color: %q", a)
	}
}

func TestDefaultColorEmptyIdentityIsStable(t *testing.T) {
	a := DefaultColor("", "")
	b := DefaultColor("", "")
	if a != b || !hexColorPattern.MatchString(a) {
		t.Fatalf("empty identity color = %q / %q, want equal valid hex colors", a, b)
	}
}
