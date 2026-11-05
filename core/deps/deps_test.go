package deps

import (
	"reflect"
	"testing"
)

func TestSummarizeEachFormat(t *testing.T) {
	bumped := Change{Name: "bumped", Kind: Bumped, From: "1.0.0", To: "1.1.0"}
	removed := Change{Name: "removed", Kind: Removed, From: "1.0.0"}
	added := Change{Name: "added", Kind: Added, To: "2.0.0"}
	want := []Change{added, bumped, removed}

	cases := []struct{ path, before, after string }{
		{
			"package.json",
			`{"dependencies":{"bumped":"1.0.0","same":"1.0.0"},"devDependencies":{"removed":"1.0.0"},"scripts":{"a":"b"}}`,
			`{"dependencies":{"bumped":"1.1.0","same":"1.0.0"},"devDependencies":{"added":"2.0.0"},"scripts":{"a":"c"}}`,
		},
		{
			"web/package-lock.json",
			`{"lockfileVersion":3,"packages":{"":{"version":"9"},"node_modules/bumped":{"version":"1.0.0"},"node_modules/removed":{"version":"1.0.0"}}}`,
			`{"lockfileVersion":3,"packages":{"":{"version":"9"},"node_modules/bumped":{"version":"1.1.0"},"node_modules/added":{"version":"2.0.0"}}}`,
		},
		{
			"npm-shrinkwrap.json",
			`{"lockfileVersion":1,"dependencies":{"bumped":{"version":"1.0.0"},"same":{"version":"1.0.0","dependencies":{"removed":{"version":"1.0.0"}}}}}`,
			`{"lockfileVersion":1,"dependencies":{"bumped":{"version":"1.1.0"},"same":{"version":"1.0.0","dependencies":{"added":{"version":"2.0.0"}}}}}`,
		},
		{
			"go.mod",
			"module m\n\ngo 1.25\n\nrequire (\n\tbumped v1.0.0\n\tremoved v1.0.0 // indirect\n)\n",
			"module m\n\ngo 1.25\n\nrequire (\n\tbumped v1.1.0\n\tadded/v2 v2.0.0\n)\n",
		},
		{
			"go.sum",
			"bumped v1.0.0 h1:a=\nbumped v1.0.0/go.mod h1:b=\nremoved v1.0.0 h1:c=\n",
			"bumped v1.1.0 h1:d=\nbumped v1.1.0/go.mod h1:e=\nadded v2.0.0 h1:f=\n",
		},
		{
			"Cargo.toml",
			"[dependencies]\nbumped = \"1.0.0\"\n\n[dev-dependencies]\nremoved = { version = \"1.0.0\" }\n",
			"[dependencies]\nbumped = \"1.1.0\"\n\n[build-dependencies]\nadded = { version = \"2.0.0\", features = [\"x\"] }\n",
		},
		{
			"Cargo.lock",
			"[[package]]\nname = \"bumped\"\nversion = \"1.0.0\"\n\n[[package]]\nname = \"removed\"\nversion = \"1.0.0\"\n",
			"[[package]]\nname = \"bumped\"\nversion = \"1.1.0\"\n\n[[package]]\nname = \"added\"\nversion = \"2.0.0\"\n",
		},
		{
			"pyproject.toml",
			"[project]\ndependencies = [\"bumped==1.0.0\"]\n\n[project.optional-dependencies]\nx = [\"removed==1.0.0\"]\n",
			"[project]\ndependencies = [\"bumped==1.1.0\"]\n\n[tool.poetry.dependencies]\npython = \"^3.12\"\nadded = \"2.0.0\"\n",
		},
		{
			"requirements-dev.txt",
			"# pinned\nbumped==1.0.0\nremoved==1.0.0 ; python_version > '3'\n-r base.txt\n",
			"bumped==1.1.0 \\\n  --hash=sha256:abc\nadded==2.0.0 # new\n-r base.txt\n",
		},
	}
	for _, c := range cases {
		t.Run(c.path, func(t *testing.T) {
			got, err := Summarize(c.path, []byte(c.before), []byte(c.after))
			if err != nil {
				t.Fatal(err)
			}
			wantHere := want
			switch c.path {
			case "go.mod", "go.sum":
				wantHere = []Change{
					{Name: "added", Kind: Added, To: "v2.0.0"},
					{Name: "bumped", Kind: Bumped, From: "v1.0.0", To: "v1.1.0"},
					{Name: "removed", Kind: Removed, From: "v1.0.0"},
				}
				if c.path == "go.mod" {
					wantHere[0].Name = "added/v2"
				}
			case "pyproject.toml", "requirements-dev.txt":
				wantHere = []Change{
					{Name: "added", Kind: Added, To: "==2.0.0"},
					{Name: "bumped", Kind: Bumped, From: "==1.0.0", To: "==1.1.0"},
					{Name: "removed", Kind: Removed, From: "==1.0.0"},
				}
				if c.path == "pyproject.toml" {
					wantHere[0].To = "2.0.0"
				}
			}
			if !reflect.DeepEqual(got, wantHere) {
				t.Fatalf("got  %+v\nwant %+v", got, wantHere)
			}
		})
	}
}

func TestSummarizeAMalformedFileIsAnError(t *testing.T) {
	cases := []struct{ path, content string }{
		{"package.json", "{not json"},
		{"package-lock.json", "{not json"},
		{"go.mod", "require (\n"},
		{"go.sum", "not a checksum\n"},
		{"Cargo.toml", "[dependencies\n"},
		{"Cargo.lock", "[[package\n"},
		{"pyproject.toml", "[project\n"},
		{"requirements.txt", "not a requirement at all\n"},
	}
	for _, c := range cases {
		if _, err := Summarize(c.path, []byte(c.content), nil); err == nil {
			t.Errorf("%s: no error for malformed content", c.path)
		}
	}
	if _, err := Summarize("README.md", nil, nil); err == nil {
		t.Error("an unrecognized file should be an error")
	}
}

func TestSummarizeATransitiveOnlyLockfileChange(t *testing.T) {
	before := `{"packages":{"node_modules/direct":{"version":"1.0.0"},"node_modules/direct/node_modules/deep":{"version":"1.0.0"}}}`
	after := `{"packages":{"node_modules/direct":{"version":"1.0.0"},"node_modules/direct/node_modules/deep":{"version":"1.0.1"}}}`
	got, err := Summarize("package-lock.json", []byte(before), []byte(after))
	if err != nil {
		t.Fatal(err)
	}
	if want := []Change{{Name: "deep", Kind: Bumped, From: "1.0.0", To: "1.0.1"}}; !reflect.DeepEqual(got, want) {
		t.Fatalf("got %+v", got)
	}
}

func TestSummarizeSeveralVersionsOfOnePackage(t *testing.T) {
	before := "[[package]]\nname = \"a\"\nversion = \"1.0.0\"\n\n[[package]]\nname = \"a\"\nversion = \"2.0.0\"\n"
	after := "[[package]]\nname = \"a\"\nversion = \"2.0.0\"\n\n[[package]]\nname = \"a\"\nversion = \"3.0.0\"\n"
	got, err := Summarize("Cargo.lock", []byte(before), []byte(after))
	if err != nil {
		t.Fatal(err)
	}
	if want := []Change{{Name: "a", Kind: Bumped, From: "1.0.0", To: "3.0.0"}}; !reflect.DeepEqual(got, want) {
		t.Fatalf("got %+v", got)
	}
}

func TestSummarizeAFileOnOneSideOnly(t *testing.T) {
	got, err := Summarize("package.json", nil, []byte(`{"dependencies":{"a":"1"}}`))
	if err != nil || len(got) != 1 || got[0].Kind != Added {
		t.Fatalf("new file = %+v, %v", got, err)
	}
	got, err = Summarize("package.json", []byte(`{"dependencies":{"a":"1"}}`), nil)
	if err != nil || len(got) != 1 || got[0].Kind != Removed {
		t.Fatalf("deleted file = %+v, %v", got, err)
	}
}

func TestSummarizeReportsNothingWhenOnlyOtherFieldsChanged(t *testing.T) {
	got, err := Summarize("package.json", []byte(`{"version":"1","dependencies":{"a":"1"}}`), []byte(`{"version":"2","dependencies":{"a":"1"}}`))
	if err != nil || got == nil || len(got) != 0 {
		t.Fatalf("got %+v, %v", got, err)
	}
}

func TestRequirementNamesAreNormalized(t *testing.T) {
	got, err := Summarize("requirements.txt", []byte("Foo_Bar[extra]==1.0\n"), []byte("foo-bar==2.0\n"))
	if err != nil {
		t.Fatal(err)
	}
	if want := []Change{{Name: "foo-bar", Kind: Bumped, From: "==1.0", To: "==2.0"}}; !reflect.DeepEqual(got, want) {
		t.Fatalf("got %+v", got)
	}
}

func TestRecognized(t *testing.T) {
	for _, path := range []string{"package.json", "a/b/go.mod", "requirements.txt", "requirements-dev.txt", "Cargo.lock"} {
		if !Recognized(path) {
			t.Errorf("%s should be recognized", path)
		}
	}
	for _, path := range []string{"README.md", "package.json.bak", "requirements.md", "yarn.lock"} {
		if Recognized(path) {
			t.Errorf("%s shouldn't be recognized", path)
		}
	}
}
