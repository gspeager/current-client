package gitexec

import "regexp"

const RedactedSecret = "<token>"

var (
	urlCredentialPattern = regexp.MustCompile(`(?i)(https?://|\w+://[^/\s:@]+:)[^/\s]+@`)
	extraHeaderPattern   = regexp.MustCompile(`(?i)(extraheader\s*[=:]\s*)[^\n]+`)
	tokenPattern         = regexp.MustCompile(`(?i)glpat-[A-Za-z0-9_-]{8,}|gh[pousr]_[A-Za-z0-9]{16,}|github_pat_[A-Za-z0-9_]{16,}|authorization:\s*(?:\S+\s+)?\S+|bearer\s+\S+|private-token:\s*\S+|token[=:]\s*\S+`)
)

// RedactSecrets replaces what looks like a credential: known token shapes,
// authorization headers, an http.extraheader value, anything after "token=",
// and a URL's sign-in. An ssh:// URL keeps its user name, which is usually just git.
func RedactSecrets(text string) string {
	text = urlCredentialPattern.ReplaceAllString(text, "${1}"+RedactedSecret+"@")
	text = extraHeaderPattern.ReplaceAllString(text, "${1}"+RedactedSecret)
	return tokenPattern.ReplaceAllString(text, RedactedSecret)
}

func redactArgs(args []string) []string {
	redacted := make([]string, len(args))
	for i, arg := range args {
		redacted[i] = RedactSecrets(arg)
	}
	return redacted
}
