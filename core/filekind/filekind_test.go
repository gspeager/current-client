package filekind

import (
	"testing"

	"github.com/gspeager/current-client/core/deps"
)

func TestOf(t *testing.T) {
	for file, want := range map[string]Kind{
		"main.go":                        Code,
		"src/app.ts":                     Code,
		"src/latest.ts":                  Code,
		"src/contest.py":                 Code,
		"internal/testing.go":            Code,
		"docs/protest/plan.md":           Code,
		"builder/a.go":                   Code,
		"Manifest.java":                  Code,
		"core/git/branch_test.go":        Test,
		"src/App.test.tsx":               Test,
		"src/app.spec.ts":                Test,
		"src/__tests__/app.ts":           Test,
		"tests/test_views.py":            Test,
		"app/test_views.py":              Test,
		"app/conftest.py":                Test,
		"spec/models/user_spec.rb":       Test,
		"src/test/java/UserService.java": Test,
		"src/UserServiceTest.java":       Test,
		"AppTests.swift":                 Test,
		"core/diff/testdata/a.patch":     Test,
		"src/__snapshots__/a.snap":       Test,
		"go.sum":                         Lockfile,
		"frontend/package-lock.json":     Lockfile,
		"vendor/github.com/x/go.sum":     Lockfile,
		"frontend/dist/index.js":         Generated,
		"vendor/github.com/x/y_test.go":  Generated,
		"api/service.pb.go":              Generated,
		"lib/jquery.min.js":              Generated,
		"src/schema.generated.ts":        Generated,
	} {
		if got := Of(file); got != want {
			t.Errorf("Of(%q) = %q, want %q", file, got, want)
		}
	}
}

// The risk report pairs a manifest with its lockfiles; both name the same ones.
func TestEveryLockfileTheDependencySummaryKnowsIsOne(t *testing.T) {
	for _, manifest := range []string{"package.json", "go.mod", "Cargo.toml", "pyproject.toml"} {
		for _, lock := range deps.LocksOf(manifest) {
			if Of(lock) != Lockfile {
				t.Errorf("%s is %q", lock, Of(lock))
			}
		}
	}
}
