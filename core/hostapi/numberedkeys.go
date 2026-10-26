package hostapi

import (
	"regexp"
	"strconv"
)

// NumberedKeys is the reference syntax of a host whose issues are numbers:
// "#123", with the key "123".
var NumberedKeys = regexp.MustCompile(`#(\d+)\b`)

var numberedBranch = regexp.MustCompile(`^(\d+)-`)

// IssueNumber parses a numbered host's key; a key that isn't a number is a
// 404, as an issue the host doesn't have would be.
func IssueNumber(key string) (int, error) {
	n, err := strconv.Atoi(key)
	if err != nil || n <= 0 {
		return 0, &HostError{Status: 404}
	}
	return n, nil
}

// NumberedBranchKey reads "123-fix-login" as issue 123.
func NumberedBranchKey(branch string) string {
	if m := numberedBranch.FindStringSubmatch(branch); m != nil {
		return m[1]
	}
	return ""
}

func NumberedReference(key string) string {
	return "#" + key
}
