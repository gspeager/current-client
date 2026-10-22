package git

import (
	"context"
	"os"
	"strings"

	"github.com/gspeager/current-client/core/gitexec"
)

// HTTPConfig is what Git itself would use for an HTTPS URL: the http.*
// settings that match it, with Git's environment overrides applied.
type HTTPConfig struct {
	CAInfo    string
	CAPath    string
	Proxy     string
	SSLVerify bool
}

// HTTPConfigFor reads `git config --get-urlmatch http <url>` from the global
// and system configuration; repoPath may be empty.
func HTTPConfigFor(ctx context.Context, repoPath, rawURL string) (HTTPConfig, error) {
	result, err := gitexec.NewExecutor("").Run(ctx, gitexec.Command{
		Dir:  repoPath,
		Args: []string{"config", "-z", "--get-urlmatch", "http", rawURL},
	})
	if err != nil {
		return HTTPConfig{}, err
	}
	cfg := parseHTTPConfig(result.Stdout)
	if v := os.Getenv("GIT_SSL_CAINFO"); v != "" {
		cfg.CAInfo = v
	}
	if v := os.Getenv("GIT_SSL_CAPATH"); v != "" {
		cfg.CAPath = v
	}
	if v := os.Getenv("GIT_SSL_NO_VERIFY"); v != "" {
		cfg.SSLVerify = false
	}
	return cfg, nil
}

func parseHTTPConfig(output string) HTTPConfig {
	cfg := HTTPConfig{SSLVerify: true}
	for _, record := range strings.Split(output, "\x00") {
		key, value, _ := strings.Cut(record, "\n")
		switch key {
		case "http.sslcainfo":
			cfg.CAInfo = value
		case "http.sslcapath":
			cfg.CAPath = value
		case "http.proxy":
			cfg.Proxy = value
		case "http.sslverify":
			cfg.SSLVerify = gitBool(value)
		}
	}
	return cfg
}

func gitBool(value string) bool {
	switch strings.ToLower(value) {
	case "false", "no", "off", "0", "":
		return false
	}
	return true
}
