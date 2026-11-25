// Package filekind says from a file's path whether it's a test, a generated
// file or a lockfile, so every view agrees. It never reads the file.
package filekind

import (
	"path"
	"regexp"
	"slices"
)

type Kind string

const (
	Code      Kind = ""
	Test      Kind = "test"
	Generated Kind = "generated"
	Lockfile  Kind = "lockfile"
)

var lockfiles = []string{
	"package-lock.json", "npm-shrinkwrap.json", "yarn.lock", "pnpm-lock.yaml", "bun.lock", "bun.lockb", "deno.lock",
	"go.sum", "go.work.sum", "Cargo.lock", "poetry.lock", "uv.lock", "pdm.lock", "Pipfile.lock", "Gemfile.lock",
	"composer.lock", "mix.lock", "pubspec.lock", "Podfile.lock", "Package.resolved", "gradle.lockfile",
	"packages.lock.json", "flake.lock", "Chart.lock", ".terraform.lock.hcl",
}

var generatedPath = regexp.MustCompile(`(^|/)(dist|build|vendor|node_modules|generated|__generated__|target)/|\.min\.(js|css)$|\.pb\.go$|[._]generated\.[a-z]+$`)

var testPath = regexp.MustCompile(`(^|/)(tests?|__tests__|specs?|testdata|__snapshots__|e2e)/` +
	`|(^|/)test_[^/]+\.py$|(^|/)conftest\.py$` +
	`|[^/]_(test|spec)\.[a-z]+$|[^/]\.(test|spec)\.[a-z]+$` +
	`|[^/](Test|Tests|Spec)\.(java|kt|kts|scala|groovy|cs|fs|vb|swift|php|m|mm)$`)

func IsGenerated(file string) bool {
	return generatedPath.MatchString(file)
}

func Of(file string) Kind {
	switch {
	case slices.Contains(lockfiles, path.Base(file)):
		return Lockfile
	case IsGenerated(file):
		return Generated
	case testPath.MatchString(file):
		return Test
	}
	return Code
}
