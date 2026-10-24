package hostcontent

import (
	"strings"
	"testing"
)

const hostile = "Fixes the login.\n\n" +
	"<script>alert(1)</script>\n\n" +
	"<img src=x onerror=\"alert(2)\">\n\n" +
	"[run](javascript:alert(3)) and [open](https://example.com/page) and " +
	"[note](/group/app/-/issues/4) and [file](file:///etc/passwd)\n\n" +
	"![screenshot](https://cdn.example.com/shot.png) ![](https://cdn.example.com/plain.png) ![inline](data:image/png;base64,AAAA)\n\n" +
	"Also <b onclick=\"x()\">bold</b> and https://example.com/auto and javascript:alert(4)\n"

func TestRenderMarkdownLeavesNothingActive(t *testing.T) {
	html := RenderMarkdown(hostile, "https://gitlab.example.com/")
	for _, forbidden := range []string{"<script", "onerror", "onclick", `href="javascript`, "alert(3)", "<img", "file:", "data:", "<a href=\"\">"} {
		if strings.Contains(html, forbidden) {
			t.Errorf("output contains %q:\n%s", forbidden, html)
		}
	}
	for _, want := range []string{
		"Fixes the login.",
		"run",
		`<a href="https://example.com/page">open</a>`,
		`<a href="https://gitlab.example.com/group/app/-/issues/4">note</a>`,
		"file",
		`<a href="https://cdn.example.com/shot.png">[screenshot]</a>`,
		`<a href="https://cdn.example.com/plain.png">[image]</a>`,
		"[inline]",
		"bold",
		`<a href="https://example.com/auto">https://example.com/auto</a>`,
		"and javascript:alert(4)</p>",
	} {
		if !strings.Contains(html, want) {
			t.Errorf("output lacks %q:\n%s", want, html)
		}
	}
	if strings.Contains(html, "&lt;script") {
		t.Errorf("raw HTML should be dropped, not shown:\n%s", html)
	}
}

func TestRenderMarkdownWithoutAWebRootDropsRelativeLinks(t *testing.T) {
	html := RenderMarkdown("[note](/group/app/-/issues/4) and [abs](https://example.com/)", "")
	if strings.Contains(html, "issues/4") || !strings.Contains(html, "<p>note and") || !strings.Contains(html, `<a href="https://example.com/">abs</a>`) {
		t.Errorf("got:\n%s", html)
	}
}

func TestRenderMarkdownKeepsOrdinaryMarkdown(t *testing.T) {
	html := RenderMarkdown("## Steps\n\n- [x] one\n- two\n\n```go\nfmt.Println()\n```\n\n| a | b |\n|---|---|\n| 1 | 2 |\n", "")
	for _, want := range []string{"<h2>Steps</h2>", "<li>", "<pre><code", "<table>"} {
		if !strings.Contains(html, want) {
			t.Errorf("output lacks %q:\n%s", want, html)
		}
	}
}
