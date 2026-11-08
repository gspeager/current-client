package hostapi

import (
	"regexp"
	"strconv"
)

var (
	numberedKey    = regexp.MustCompile(`#(\d+)\b`)
	numberedBranch = regexp.MustCompile(`^(\d+)-`)
)

// NumberedKeys is the IssueTracker key syntax of a host whose issues are
// numbers: "#123", with the key "123". A provider embeds it.
type NumberedKeys struct{}

func (NumberedKeys) KeyPattern() *regexp.Regexp {
	return numberedKey
}

// KeyInBranch reads "123-fix-login" as issue 123.
func (NumberedKeys) KeyInBranch(branch string) string {
	if m := numberedBranch.FindStringSubmatch(branch); m != nil {
		return m[1]
	}
	return ""
}

func (NumberedKeys) Reference(key string) string {
	return "#" + key
}

// IssueNumber parses a numbered host's key; a key that isn't a number is a
// 404, as an issue the host doesn't have would be.
func IssueNumber(key string) (int, error) {
	n, err := strconv.Atoi(key)
	if err != nil || n <= 0 {
		return 0, &HostError{Status: 404}
	}
	return n, nil
}
