package identity

import "testing"

func TestSetColorAndResolve(t *testing.T) {
	colors, err := SetColor(nil, "Alex Rivera", "alex@example.com", "#4cd7f6")
	if err != nil {
		t.Fatalf("SetColor: %v", err)
	}
	if got := ResolveColor(colors, "Alex Rivera", "alex@example.com"); got != "#4cd7f6" {
		t.Fatalf("ResolveColor = %q, want override %q", got, "#4cd7f6")
	}
}

func TestResolveColorFallsBackToDefault(t *testing.T) {
	want := DefaultColor("Someone Else", "someone@example.com")
	if got := ResolveColor(nil, "Someone Else", "someone@example.com"); got != want {
		t.Fatalf("ResolveColor = %q, want default %q", got, want)
	}
}

func TestSetColorRejectsLowContrast(t *testing.T) {
	// Near-black on the badge's near-black ink: unreadable.
	if _, err := SetColor(nil, "Alex Rivera", "alex@example.com", "#0a0e16"); err == nil {
		t.Fatal("expected an error for a low-contrast color, got nil")
	}
}

func TestSetColorRejectsInvalidHex(t *testing.T) {
	if _, err := SetColor(nil, "Alex Rivera", "alex@example.com", "cyan"); err == nil {
		t.Fatal("expected an error for a non-hex color, got nil")
	}
}

func TestSetColorDoesNotMutateCallersMap(t *testing.T) {
	original, err := SetColor(nil, "Alex Rivera", "alex@example.com", "#4cd7f6")
	if err != nil {
		t.Fatalf("SetColor: %v", err)
	}

	updated, err := SetColor(original, "Mara Chen", "mara@example.com", "#c0c1ff")
	if err != nil {
		t.Fatalf("SetColor: %v", err)
	}
	if _, ok := original["mara@example.com"]; ok {
		t.Fatal("SetColor mutated the caller's map in place")
	}
	if len(updated) != 2 {
		t.Fatalf("updated map has %d colors, want 2", len(updated))
	}
}

func TestRemoveColorOverrideRevertsToDefault(t *testing.T) {
	colors, err := SetColor(nil, "Alex Rivera", "alex@example.com", "#4cd7f6")
	if err != nil {
		t.Fatalf("SetColor: %v", err)
	}

	colors = RemoveColorOverride(colors, IdentityKey("Alex Rivera", "alex@example.com"))

	want := DefaultColor("Alex Rivera", "alex@example.com")
	if got := ResolveColor(colors, "Alex Rivera", "alex@example.com"); got != want {
		t.Fatalf("ResolveColor after removal = %q, want default %q", got, want)
	}
}

func TestRemoveColorOverrideOfUnknownKeyIsANoop(t *testing.T) {
	colors, err := SetColor(nil, "Alex Rivera", "alex@example.com", "#4cd7f6")
	if err != nil {
		t.Fatalf("SetColor: %v", err)
	}

	if updated := RemoveColorOverride(colors, "nobody@example.com"); len(updated) != 1 {
		t.Fatalf("removing an unknown key changed the map: %+v", updated)
	}
}
