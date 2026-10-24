// Package hostcontent renders text from a Git host, which anyone with an
// account there may have written, into HTML that can do nothing: no raw
// HTML, no script, links only to http(s) pages, images shown as links.
package hostcontent

import (
	"bytes"
	"net/url"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"
)

// goldmark drops raw HTML blocks and inline HTML unless told otherwise, and
// the transformer below leaves no link or image the page could fetch from.
var renderer = goldmark.New(
	goldmark.WithExtensions(extension.GFM),
	goldmark.WithParserOptions(parser.WithASTTransformers(util.Prioritized(linkTransformer{}, 100))),
)

// RenderMarkdown returns HTML for source. Relative links are resolved against
// webRoot, the host's web origin for the project, when one is given.
func RenderMarkdown(source, webRoot string) string {
	var b bytes.Buffer
	ctx := parser.NewContext()
	ctx.Set(webRootKey, webRoot)
	if err := renderer.Convert([]byte(source), &b, parser.WithContext(ctx)); err != nil {
		return ""
	}
	return b.String()
}

var webRootKey = parser.NewContextKey()

type linkTransformer struct{}

// Transform turns images into links to their address (nothing is fetched),
// resolves relative links against the web root, and unwraps links that
// aren't http(s) so their text stays and nothing is clickable.
func (linkTransformer) Transform(doc *ast.Document, reader text.Reader, pc parser.Context) {
	webRoot, _ := pc.Get(webRootKey).(string)
	var replace []func()
	ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		switch node := n.(type) {
		case *ast.Image:
			replace = append(replace, func() { replaceImage(node, reader.Source(), webRoot) })
		case *ast.Link:
			if dest := webLink(string(node.Destination), webRoot); dest == "" {
				replace = append(replace, func() { unwrap(node) })
			} else {
				node.Destination = []byte(dest)
			}
		case *ast.AutoLink:
			if webLink(string(node.URL(reader.Source())), webRoot) == "" {
				replace = append(replace, func() { unwrapAutoLink(node, reader.Source()) })
			}
		}
		return ast.WalkContinue, nil
	})
	for _, apply := range replace {
		apply()
	}
}

func replaceImage(image *ast.Image, source []byte, webRoot string) {
	label := textOf(image, source)
	if label == "" {
		label = "image"
	}
	dest := webLink(string(image.Destination), webRoot)
	textNode := ast.NewString([]byte("[" + label + "]"))
	if dest == "" {
		image.Parent().ReplaceChild(image.Parent(), image, textNode)
		return
	}
	link := ast.NewLink()
	link.Destination = []byte(dest)
	link.AppendChild(link, textNode)
	image.Parent().ReplaceChild(image.Parent(), image, link)
}

func textOf(node ast.Node, source []byte) string {
	var b strings.Builder
	for child := node.FirstChild(); child != nil; child = child.NextSibling() {
		if textNode, ok := child.(*ast.Text); ok {
			b.Write(textNode.Segment.Value(source))
		} else {
			b.WriteString(textOf(child, source))
		}
	}
	return b.String()
}

func unwrap(link *ast.Link) {
	parent := link.Parent()
	for child := link.FirstChild(); child != nil; {
		next := child.NextSibling()
		parent.InsertBefore(parent, link, child)
		child = next
	}
	parent.RemoveChild(parent, link)
}

func unwrapAutoLink(link *ast.AutoLink, source []byte) {
	link.Parent().ReplaceChild(link.Parent(), link, ast.NewString(link.Label(source)))
}

// webLink is the absolute http(s) address for dest, or "" for anything else:
// other schemes, and relative paths when there is no web root to resolve
// them against.
func webLink(dest, webRoot string) string {
	dest = strings.TrimSpace(dest)
	u, err := url.Parse(dest)
	if err != nil || dest == "" {
		return ""
	}
	if u.Scheme == "" && webRoot != "" {
		root, err := url.Parse(webRoot)
		if err != nil {
			return ""
		}
		u = root.ResolveReference(u)
	}
	if (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return ""
	}
	return u.String()
}
