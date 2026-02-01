package templates

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

func TestRenderMarkdown(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		contains []string
	}{
		{
			name:  "plain text",
			input: "Hello, World!",
			contains: []string{
				"<p>Hello, World!</p>",
			},
		},
		{
			name:  "bold text",
			input: "This is **bold** text",
			contains: []string{
				"<strong>bold</strong>",
			},
		},
		{
			name:  "italic text",
			input: "This is *italic* text",
			contains: []string{
				"<em>italic</em>",
			},
		},
		{
			name:  "code inline",
			input: "Use the `fmt.Println` function",
			contains: []string{
				"<code>fmt.Println</code>",
			},
		},
		{
			name: "go code block",
			input: "```go\nfunc main() {\n    fmt.Println(\"Hello\")\n}\n```",
			contains: []string{
				"<pre",
				"chroma",
				"func",
				"main",
				"fmt",
				"Println",
			},
		},
		{
			name:  "link",
			input: "[Go](https://golang.org)",
			contains: []string{
				`<a href="https://golang.org"`,
				">Go</a>",
			},
		},
		{
			name: "unordered list",
			input: "- Item 1\n- Item 2\n- Item 3",
			contains: []string{
				"<ul>",
				"<li>Item 1</li>",
				"<li>Item 2</li>",
				"<li>Item 3</li>",
				"</ul>",
			},
		},
		{
			name:  "heading h2",
			input: "## Section Title",
			contains: []string{
				"<h2",
				"Section Title</h2>",
			},
		},
		{
			name:  "heading h3",
			input: "### Subsection",
			contains: []string{
				"<h3",
				"Subsection</h3>",
			},
		},
		{
			name: "blockquote",
			input: "> This is a quote",
			contains: []string{
				"<blockquote>",
				"This is a quote",
				"</blockquote>",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			component := RenderMarkdown(tt.input)

			var buf bytes.Buffer
			err := component.Render(context.Background(), &buf)
			if err != nil {
				t.Fatalf("Failed to render: %v", err)
			}

			output := buf.String()
			for _, want := range tt.contains {
				if !strings.Contains(output, want) {
					t.Errorf("Output should contain %q, but got:\n%s", want, output)
				}
			}
		})
	}
}

func TestRenderMarkdownGoCodeHighlighting(t *testing.T) {
	input := "```go\npackage main\n\nimport \"fmt\"\n\nfunc main() {\n    fmt.Println(\"Hello, World!\")\n}\n```"

	component := RenderMarkdown(input)

	var buf bytes.Buffer
	err := component.Render(context.Background(), &buf)
	if err != nil {
		t.Fatalf("Failed to render: %v", err)
	}

	output := buf.String()

	// Check that syntax highlighting classes are present
	highlightChecks := []string{
		"chroma",    // Chroma wrapper class
		"<pre",      // Pre element
		"<code",     // Code element
	}

	for _, check := range highlightChecks {
		if !strings.Contains(output, check) {
			t.Errorf("Output should contain %q for syntax highlighting, but got:\n%s", check, output)
		}
	}

	// Check that Go code elements are present
	codeChecks := []string{
		"package",
		"main",
		"import",
		"fmt",
		"func",
		"Println",
	}

	for _, check := range codeChecks {
		if !strings.Contains(output, check) {
			t.Errorf("Output should contain Go code element %q, but got:\n%s", check, output)
		}
	}
}
