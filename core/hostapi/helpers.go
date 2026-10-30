package hostapi

import (
	"bytes"
	"errors"
)

// WithScope names the token scope a refused request needed, so a 403 can
// say which one to add.
func WithScope(err error, scope string) error {
	var hostErr *HostError
	if errors.As(err, &hostErr) && hostErr.Scope == "" {
		hostErr.Scope = scope
	}
	return err
}

// TailLog keeps the last maxBytes of a log, cut at a line start; truncated
// says something was dropped. maxBytes of zero keeps everything.
func TailLog(body []byte, maxBytes int) (log string, truncated bool) {
	if maxBytes <= 0 || len(body) <= maxBytes {
		return string(body), false
	}
	cut := body[len(body)-maxBytes:]
	if i := bytes.IndexByte(cut, '\n'); i >= 0 {
		cut = cut[i+1:]
	}
	return string(cut), true
}

// Retryable is whether a job that ended this way is worth running again.
func Retryable(status CIStatus) bool {
	return status == CIFailed || status == CICanceled
}
