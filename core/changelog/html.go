package changelog

import (
	"bytes"
	"html"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
)

// goldmark escapes raw HTML and drops unsafe link targets by default, so the
// output is safe to show in the app.
var renderer = goldmark.New(goldmark.WithExtensions(extension.GFM))

func HTML(markdown string) (string, error) {
	var b bytes.Buffer
	if err := renderer.Convert([]byte(markdown), &b); err != nil {
		return "", err
	}
	return b.String(), nil
}

// HTMLDocument is a standalone page with its styles inline, so it opens the
// same anywhere without fetching anything.
func HTMLDocument(title, markdown string) (string, error) {
	body, err := HTML(markdown)
	if err != nil {
		return "", err
	}
	return `<!doctype html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>` + html.EscapeString(title) + `</title>
<style>
body { max-width: 760px; margin: 40px auto; padding: 0 16px; font: 15px/1.6 system-ui, sans-serif; color: #1f2328; }
h1, h2, h3 { line-height: 1.25; }
h2 { margin-top: 32px; padding-bottom: 6px; border-bottom: 1px solid #d1d9e0; }
code { font: 13px ui-monospace, monospace; }
</style>
</head>
<body>
` + body + `</body>
</html>
`, nil
}
