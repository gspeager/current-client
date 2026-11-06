// Package risk reports objective findings on staged changes: secret-looking
// strings, conflict markers, large files, a lockfile out of step with its
// manifest, and sweeping changes to generated files. It only reads, never
// blocks anything, and never returns a matched secret in full.
//
// gitleaks.toml is gitleaks's rule set, used as it ships, under the license in
// gitleaks.LICENSE. To update it, replace the file from a gitleaks release.
package risk
