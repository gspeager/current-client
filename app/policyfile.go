package app

import "github.com/gspeager/current-client/internal/policy"

// UsePolicyFile reads the organization policy from path instead of the
// platform's fixed place, for an embedding app that keeps its own or a test
// of one. It replaces what Startup loaded.
func UsePolicyFile(path string) error {
	policy.SetPath(path)
	return policy.Load()
}
