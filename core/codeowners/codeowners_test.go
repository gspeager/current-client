package codeowners

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestGitHubRules(t *testing.T) {
	f := Parse(`# comment
*       @global-owner1 @global-owner2
*.js    @js-owner
/build/out/ @doctocat
docs/*  docs@example.com
apps/   @octocat
/scripts/ @doctocat @octocat
**/logs @octocat
/apps/ @octocat
/apps/github
\#weird @hash
`, GitHub)
	cases := map[string][]string{
		"README.md":                      {"@global-owner1", "@global-owner2"},
		"src/index.js":                   {"@js-owner"},
		"build/out/today.txt":            {"@doctocat"},
		"build/logs/today.txt":           {"@octocat"},
		"docs/getting-started.md":        {"docs@example.com"},
		"docs/build-app/troubleshoot.md": {"@global-owner1", "@global-owner2"},
		"any/apps/x.go":                  {"@octocat"},
		"scripts/run.sh":                 {"@doctocat", "@octocat"},
		"a/b/logs/x":                     {"@octocat"},
		"apps/x":                         {"@octocat"},
		"apps/github/x":                  nil,
		"#weird":                         {"@hash"},
	}
	for path, want := range cases {
		if got := f.Owners(path); !reflect.DeepEqual(got, want) {
			t.Errorf("%s: got %v, want %v", path, got, want)
		}
	}
}

func TestGitLabSections(t *testing.T) {
	f := Parse(`*.rb @ruby-owner
[Documentation]
docs/ @docs-team
README.md @tech-writer

^[Database][2] @dba-lead @dba-two
db/ @database-team
model/db/

[documentation]
docs/api/ @api-docs

[Config] @config-default
config/
config/secrets.yml @security
path\ with\ space.txt @spacey
`, GitLab)
	if len(f.Sections) != 3 {
		t.Fatalf("sections = %d, want 3 (documentation merged by name)", len(f.Sections))
	}
	db := f.Sections[1]
	if db.Name != "Database" || !db.Optional || db.Approvals != 2 || !reflect.DeepEqual(db.Default, []string{"@dba-lead", "@dba-two"}) {
		t.Fatalf("database section = %+v", db)
	}

	got := f.Match("docs/api/users.md")
	want := []Owner{{Name: "@api-docs", Section: "Documentation"}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("docs/api: got %+v, want %+v", got, want)
	}

	got = f.Match("db/schema.rb")
	want = []Owner{
		{Name: "@ruby-owner"},
		{Name: "@database-team", Section: "Database", Optional: true, Approvals: 2},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("db/schema.rb: got %+v, want %+v", got, want)
	}

	if got := f.Owners("model/db/x.go"); !reflect.DeepEqual(got, []string{"@dba-lead", "@dba-two"}) {
		t.Fatalf("section default owners: got %v", got)
	}
	if got := f.Owners("config/app.yml"); !reflect.DeepEqual(got, []string{"@config-default"}) {
		t.Fatalf("config default: got %v", got)
	}
	if got := f.Owners("config/secrets.yml"); !reflect.DeepEqual(got, []string{"@security"}) {
		t.Fatalf("last match in a section wins: got %v", got)
	}
	if got := f.Owners("path with space.txt"); !reflect.DeepEqual(got, []string{"@spacey"}) {
		t.Fatalf("escaped space: got %v", got)
	}
	if got := f.Owners("README.md"); !reflect.DeepEqual(got, []string{"@tech-writer"}) {
		t.Fatalf("README: got %v", got)
	}
}

func TestGitHubHasNoSections(t *testing.T) {
	f := Parse("[Docs]\ndocs/ @a\n", GitHub)
	if len(f.Sections) != 0 {
		t.Fatalf("GitHub files have no sections: %+v", f.Sections)
	}
	if got := f.Owners("docs/x"); !reflect.DeepEqual(got, []string{"@a"}) {
		t.Fatalf("got %v", got)
	}
}

func TestLoadPrefersTheHostsOwnLocation(t *testing.T) {
	root := t.TempDir()
	write := func(rel, content string) {
		t.Helper()
		path := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if f, path, err := Load(root, GitHub); f != nil || path != "" || err != nil {
		t.Fatalf("empty repo: %v %q %v", f, path, err)
	}
	write("docs/CODEOWNERS", "* @docs\n")
	write(".gitlab/CODEOWNERS", "* @gitlab\n")
	write(".github/CODEOWNERS", "* @github\n")
	for style, want := range map[Style]string{GitHub: ".github/CODEOWNERS", GitLab: ".gitlab/CODEOWNERS"} {
		f, path, err := Load(root, style)
		if err != nil || path != want || f.Style != style {
			t.Fatalf("%s: got %q %v (style %q)", style, path, err, f.Style)
		}
	}
}
