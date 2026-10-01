package identity

import (
	"crypto/sha256"
	"fmt"
	"math"
	"strings"
)

func Initials(name, email string) string {
	if in := initialsFromWords(splitWords(name)); in != "" {
		return in
	}
	local, _, _ := strings.Cut(email, "@")
	if in := initialsFromWords(splitWords(local)); in != "" {
		return in
	}
	return "?"
}

func splitWords(s string) []string {
	return strings.FieldsFunc(s, func(r rune) bool {
		switch r {
		case ' ', '.', '_', '-', '+':
			return true
		}
		return false
	})
}

func initialsFromWords(words []string) string {
	switch len(words) {
	case 0:
		return ""
	case 1:
		return upperPrefix(words[0], 2)
	default:
		return upperPrefix(words[0], 1) + upperPrefix(words[len(words)-1], 1)
	}
}

func upperPrefix(s string, n int) string {
	r := []rune(s)
	if len(r) > n {
		r = r[:n]
	}
	return strings.ToUpper(string(r))
}

// IdentityKey prefers email, which is more stable than a name in git history.
func IdentityKey(name, email string) string {
	if email != "" {
		return email
	}
	return name
}

func DefaultColor(name, email string) string {
	sum := sha256.Sum256([]byte(IdentityKey(name, email)))
	hue := float64(sum[0]) * 360 / 256
	// Tuned so every hue clears MinContrastRatio against BadgeInk.
	return hslToHex(hue, 0.60, 0.66)
}

func hslToHex(h, s, l float64) string {
	c := (1 - math.Abs(2*l-1)) * s
	x := c * (1 - math.Abs(math.Mod(h/60, 2)-1))
	m := l - c/2

	var r, g, b float64
	switch {
	case h < 60:
		r, g, b = c, x, 0
	case h < 120:
		r, g, b = x, c, 0
	case h < 180:
		r, g, b = 0, c, x
	case h < 240:
		r, g, b = 0, x, c
	case h < 300:
		r, g, b = x, 0, c
	default:
		r, g, b = c, 0, x
	}

	toByte := func(v float64) int { return int(math.Round((v + m) * 255)) }
	return fmt.Sprintf("#%02x%02x%02x", toByte(r), toByte(g), toByte(b))
}
