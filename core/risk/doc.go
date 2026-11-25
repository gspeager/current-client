// Package risk reports objective findings on what's staged or on a range of
// commits: secret-looking strings, conflict markers, large files, a lockfile out
// of step with its manifest, sweeping changes to generated files, fewer or
// skipped tests, silenced checkers, CI files, agent instructions, database
// migrations and what packages export. It only reads, never blocks anything, and
// never returns a matched secret in full.
//
// gitleaks.toml is gitleaks's rule set, used as it ships, under the license in
// gitleaks.LICENSE. To update it, replace the file from a gitleaks release.
package risk
