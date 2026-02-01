package templates

import (
	"bytes"
	"sync"

	"github.com/a-h/templ"
	chromahtml "github.com/alecthomas/chroma/v2/formatters/html"
	"github.com/yuin/goldmark"
	highlighting "github.com/yuin/goldmark-highlighting/v2"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer/html"
)

var (
	mdOnce     sync.Once
	mdRenderer goldmark.Markdown
)

// getMarkdownRenderer returns the singleton goldmark renderer instance.
func getMarkdownRenderer() goldmark.Markdown {
	mdOnce.Do(func() {
		mdRenderer = goldmark.New(
			goldmark.WithExtensions(
				extension.GFM,
				highlighting.NewHighlighting(
					highlighting.WithStyle("github"),
					highlighting.WithFormatOptions(
						chromahtml.WithClasses(true),
					),
				),
			),
			goldmark.WithParserOptions(
				parser.WithAutoHeadingID(),
			),
			goldmark.WithRendererOptions(
				html.WithHardWraps(),
				html.WithXHTML(),
			),
		)
	})
	return mdRenderer
}

// RenderMarkdown converts markdown text to HTML.
// Returns safe HTML that can be used with templ.Raw().
func RenderMarkdown(markdown string) templ.Component {
	var buf bytes.Buffer
	md := getMarkdownRenderer()
	if err := md.Convert([]byte(markdown), &buf); err != nil {
		// If conversion fails, return the original text escaped
		return templ.Raw("<p>" + templ.EscapeString(markdown) + "</p>")
	}
	return templ.Raw(buf.String())
}
