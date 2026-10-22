package git

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestHTTPConfigForMatchesGitsURLRules(t *testing.T) {
	home := t.TempDir()
	config := `[http "https://gitlab.example.com"]
	sslCAInfo = /etc/ssl/corp.pem
	proxy = http://proxy.example.com:3128
[http "https://gitlab.example.com:8443"]
	sslVerify = false
[http]
	sslCAPath = /etc/ssl/certs
`
	if err := os.WriteFile(filepath.Join(home, ".gitconfig"), []byte(config), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", home)
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	t.Setenv("GIT_SSL_CAINFO", "")
	t.Setenv("GIT_SSL_NO_VERIFY", "")

	tests := []struct {
		url  string
		want HTTPConfig
	}{
		{"https://gitlab.example.com/api/v4", HTTPConfig{CAInfo: "/etc/ssl/corp.pem", CAPath: "/etc/ssl/certs", Proxy: "http://proxy.example.com:3128", SSLVerify: true}},
		{"https://gitlab.example.com:8443/api/v4", HTTPConfig{CAPath: "/etc/ssl/certs", SSLVerify: false}},
		{"https://api.github.com", HTTPConfig{CAPath: "/etc/ssl/certs", SSLVerify: true}},
	}
	for _, tt := range tests {
		got, err := HTTPConfigFor(context.Background(), "", tt.url)
		if err != nil || got != tt.want {
			t.Errorf("HTTPConfigFor(%q) = %+v, %v; want %+v", tt.url, got, err, tt.want)
		}
	}

	t.Setenv("GIT_SSL_NO_VERIFY", "1")
	if got, _ := HTTPConfigFor(context.Background(), "", "https://api.github.com"); got.SSLVerify {
		t.Error("GIT_SSL_NO_VERIFY wasn't honored")
	}
}

func TestParseHTTPConfigBooleans(t *testing.T) {
	for value, want := range map[string]bool{"true": true, "yes": true, "1": true, "false": false, "off": false, "no": false, "0": false} {
		if got := parseHTTPConfig("http.sslverify\n" + value).SSLVerify; got != want {
			t.Errorf("sslverify=%s parsed as %v", value, got)
		}
	}
}
