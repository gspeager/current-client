package identity

import "fmt"

// Color overrides are a map of IdentityKey to hex color. The functions return
// a new map rather than changing the caller's.

// SetColor rejects colors that fail WCAG contrast against BadgeInk.
func SetColor(overrides map[string]string, name, email, color string) (map[string]string, error) {
	key := IdentityKey(name, email)
	if key == "" {
		return overrides, fmt.Errorf("cannot set a color for an identity with no name or email")
	}
	if _, _, _, ok := hexToRGB(color); !ok {
		return overrides, fmt.Errorf("invalid color %q: must be a 6-digit hex color", color)
	}
	if ratio := ContrastRatio(color, BadgeInk); ratio < MinContrastRatio {
		return overrides, fmt.Errorf("color %q has insufficient contrast against badge text (%.2f:1, need at least %.1f:1)", color, ratio, MinContrastRatio)
	}

	colors := make(map[string]string, len(overrides)+1)
	for k, v := range overrides {
		colors[k] = v
	}
	colors[key] = color
	return colors, nil
}

func ResolveColor(overrides map[string]string, name, email string) string {
	if color, ok := overrides[IdentityKey(name, email)]; ok {
		return color
	}
	return DefaultColor(name, email)
}

func RemoveColorOverride(overrides map[string]string, key string) map[string]string {
	if _, ok := overrides[key]; !ok {
		return overrides
	}
	colors := make(map[string]string, len(overrides)-1)
	for k, v := range overrides {
		if k != key {
			colors[k] = v
		}
	}
	return colors
}
