package changelog

import (
	"strings"
	"testing"
)

func TestHTML(t *testing.T) {
	got, err := HTML("## [1.0.0] - 2026-09-28\n\n### Added\n\n- **history:** filter by type\n")
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"<h2>[1.0.0] - 2026-09-28</h2>", "<h3>Added</h3>", "<li><strong>history:</strong> filter by type</li>"} {
		if !strings.Contains(got, want) {
			t.Errorf("HTML missing %q:\n%s", want, got)
		}
	}
}

func TestHTMLEscapesRawHTMLAndUnsafeLinks(t *testing.T) {
	got, err := HTML("- <script>alert(1)</script>\n- [link](javascript:alert(1))\n")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(got, "<script>") || strings.Contains(got, "javascript:") {
		t.Errorf("unsafe content survived:\n%s", got)
	}
}

func TestHTMLDocumentIsStandalone(t *testing.T) {
	got, err := HTMLDocument("Changelog <1.0>", "### Added\n\n- x\n")
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"<!doctype html>", "<title>Changelog &lt;1.0&gt;</title>", "<h3>Added</h3>"} {
		if !strings.Contains(got, want) {
			t.Errorf("document missing %q", want)
		}
	}
	if strings.Contains(got, "http") {
		t.Error("document references something remote")
	}
}
