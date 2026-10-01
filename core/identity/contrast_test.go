package identity

import "testing"

func TestContrastRatioBlackOnWhite(t *testing.T) {
	if got := ContrastRatio("#000000", "#ffffff"); got < 20.9 || got > 21.1 {
		t.Fatalf("black/white contrast = %.2f, want ~21", got)
	}
}

func TestContrastRatioSameColor(t *testing.T) {
	if got := ContrastRatio("#4cd7f6", "#4cd7f6"); got < 0.99 || got > 1.01 {
		t.Fatalf("same-color contrast = %.2f, want 1", got)
	}
}

func TestContrastRatioOrderIndependent(t *testing.T) {
	a := ContrastRatio("#0a0e16", "#4cd7f6")
	b := ContrastRatio("#4cd7f6", "#0a0e16")
	if a != b {
		t.Fatalf("contrast ratio depends on argument order: %.4f vs %.4f", a, b)
	}
}

func TestContrastRatioInvalidColor(t *testing.T) {
	if got := ContrastRatio("not-a-color", "#ffffff"); got != 0 {
		t.Fatalf("got %.2f, want 0 for an invalid color", got)
	}
}

func TestDefaultColorAlwaysMeetsContrastRequirement(t *testing.T) {
	// Every generated hue must stay readable against BadgeInk.
	for i := 0; i < 256; i++ {
		hue := float64(i) * 360 / 256
		color := hslToHex(hue, 0.60, 0.66)
		if ratio := ContrastRatio(color, BadgeInk); ratio < MinContrastRatio {
			t.Fatalf("hue %.1f -> %s has contrast %.2f:1 against ink, want >= %.1f:1", hue, color, ratio, MinContrastRatio)
		}
	}
}
