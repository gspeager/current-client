package identity

import (
	"math"
	"strconv"
	"strings"
)

// BadgeInk must match IdentityBadge's initials color.
const BadgeInk = "#0a0e16"

// MinContrastRatio is WCAG AA for small text.
const MinContrastRatio = 4.5

func ContrastRatio(hexA, hexB string) float64 {
	la, aOK := relativeLuminance(hexA)
	lb, bOK := relativeLuminance(hexB)
	if !aOK || !bOK {
		return 0
	}
	lighter, darker := la, lb
	if lb > la {
		lighter, darker = lb, la
	}
	return (lighter + 0.05) / (darker + 0.05)
}

func relativeLuminance(hex string) (float64, bool) {
	r, g, b, ok := hexToRGB(hex)
	if !ok {
		return 0, false
	}
	lin := func(c float64) float64 {
		c /= 255
		if c <= 0.03928 {
			return c / 12.92
		}
		return math.Pow((c+0.055)/1.055, 2.4)
	}
	return 0.2126*lin(r) + 0.7152*lin(g) + 0.0722*lin(b), true
}

func hexToRGB(hex string) (r, g, b float64, ok bool) {
	hex = strings.TrimPrefix(hex, "#")
	if len(hex) != 6 {
		return 0, 0, 0, false
	}
	v, err := strconv.ParseUint(hex, 16, 32)
	if err != nil {
		return 0, 0, 0, false
	}
	return float64((v >> 16) & 0xff), float64((v >> 8) & 0xff), float64(v & 0xff), true
}
