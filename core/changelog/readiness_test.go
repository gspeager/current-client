package changelog

import "testing"

func TestVersionPatternMatchesTheWholeVersion(t *testing.T) {
	p := versionPattern("0.5.0")
	for text, want := range map[string]bool{"0.5.0": true, `"0.5.0.0"`: true, "v0.5.0,": true, "10.5.0": false, "1.0.5.0": false, "0.5.01": false} {
		if p.MatchString(text) != want {
			t.Errorf("%q: %v", text, !want)
		}
	}
}

func TestNextVersionRaisesTheLastNumber(t *testing.T) {
	for tag, want := range map[string]string{"v1.1": "1.2", "7": "8", "v0.4.9": "0.4.10", "release-1": "", "": ""} {
		if got := nextVersion(tag); got != want {
			t.Errorf("nextVersion(%q) = %q, want %q", tag, got, want)
		}
	}
}
