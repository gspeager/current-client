//go:build !darwin

package platform

// UseLoginShellPath only has work to do on macOS; elsewhere a desktop
// session already gives apps the user's PATH.
func UseLoginShellPath() {}
