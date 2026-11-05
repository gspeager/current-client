package gitexec

import (
	"strings"
	"testing"
)

func TestRedactSecrets(t *testing.T) {
	cases := []struct{ in, want string }{
		{"https://deploy:s3cret@git.example.com/a.git", "https://<token>@git.example.com/a.git"},
		{"https://s3cret@git.example.com/a.git", "https://<token>@git.example.com/a.git"},
		{"ssh://git:s3cret@git.example.com/a.git", "ssh://git:<token>@git.example.com/a.git"},
		{"ssh://git@git.example.com/a.git", "ssh://git@git.example.com/a.git"},
		{"git@git.example.com:a.git", "git@git.example.com:a.git"},
		{"AUTHORIZATION: basic eC1hY2Nlc3M6c2VjcmV0", "<token>"},
		{"Authorization: s3cret", "<token>"},
		{"answered 401 for token=abc", "answered 401 for <token>"},
		{"glpat-AbCdEfGhIjKlMnOpQrSt", "<token>"},
		{"basic authentication failed", "basic authentication failed"},
		{"https://git.example.com/a.git", "https://git.example.com/a.git"},
	}
	for _, c := range cases {
		if got := RedactSecrets(c.in); got != c.want {
			t.Errorf("RedactSecrets(%q) = %q, want %q", c.in, got, c.want)
		}
		if strings.Contains(RedactSecrets(c.in), "s3cret") {
			t.Errorf("RedactSecrets(%q) leaked the secret", c.in)
		}
	}
}
