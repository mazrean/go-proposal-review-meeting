package templates_test

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/mazrean/go-proposal-review-meeting/internal/site/templates"
)

func TestBaseLayout(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		title        string
		wantContains []string
	}{
		{
			name:  "renders basic HTML structure",
			title: "Test Page",
			wantContains: []string{
				"<!doctype html>",
				"<html",
				"lang=\"ja\"",
				"<head>",
				"</head>",
				"<body",
				"</body>",
				"</html>",
			},
		},
		{
			name:  "includes page title",
			title: "Go Proposal Weekly Digest",
			wantContains: []string{
				"<title>Go Proposal Weekly Digest</title>",
			},
		},
		{
			name:  "includes meta tags",
			title: "Test",
			wantContains: []string{
				"charset=\"UTF-8\"",
				"viewport",
				"width=device-width",
			},
		},
		{
			name:  "includes stylesheet link",
			title: "Test",
			wantContains: []string{
				"<link",
				"stylesheet",
				"styles.css",
			},
		},
		{
			name:  "includes RSS autodiscovery",
			title: "Test",
			wantContains: []string{
				"application/rss+xml",
				"feed.xml",
			},
		},
	}

	for _, tt := range tests {
		tt := tt // capture loop variable for parallel test
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var buf bytes.Buffer
			// Render with empty child content
			err := templates.BaseLayout(tt.title).Render(context.Background(), &buf)
			if err != nil {
				t.Fatalf("failed to render: %v", err)
			}

			html := buf.String()

			for _, want := range tt.wantContains {
				if !strings.Contains(html, want) {
					t.Errorf("expected HTML to contain %q, got:\n%s", want, html)
				}
			}
		})
	}
}

func TestHeader(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	err := templates.Header().Render(context.Background(), &buf)
	if err != nil {
		t.Fatalf("failed to render: %v", err)
	}

	html := buf.String()

	wantContains := []string{
		"<header",
		"</header>",
		"Go Proposal Weekly Digest",
	}

	for _, want := range wantContains {
		if !strings.Contains(html, want) {
			t.Errorf("expected HTML to contain %q, got:\n%s", want, html)
		}
	}
}

func TestFooter(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	err := templates.Footer().Render(context.Background(), &buf)
	if err != nil {
		t.Fatalf("failed to render: %v", err)
	}

	html := buf.String()

	wantContains := []string{
		"<footer",
		"</footer>",
		"GitHub",
		"golang/go",
	}

	for _, want := range wantContains {
		if !strings.Contains(html, want) {
			t.Errorf("expected HTML to contain %q, got:\n%s", want, html)
		}
	}
}

func TestNavigation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		currentPath  string
		wantContains []string
	}{
		{
			name:        "renders nav element",
			currentPath: "/",
			wantContains: []string{
				"<nav",
				"</nav>",
			},
		},
		{
			name:        "includes home link",
			currentPath: "/2026/w05/",
			wantContains: []string{
				"href=\"/\"",
				"ホーム",
			},
		},
		{
			name:        "includes RSS feed link",
			currentPath: "/",
			wantContains: []string{
				"feed.xml",
				"RSS",
			},
		},
		{
			name:        "includes aria-current for active page",
			currentPath: "/",
			wantContains: []string{
				"aria-current=\"page\"",
			},
		},
	}

	for _, tt := range tests {
		tt := tt // capture loop variable for parallel test
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var buf bytes.Buffer
			err := templates.Navigation(tt.currentPath).Render(context.Background(), &buf)
			if err != nil {
				t.Fatalf("failed to render: %v", err)
			}

			html := buf.String()

			for _, want := range tt.wantContains {
				if !strings.Contains(html, want) {
					t.Errorf("expected HTML to contain %q, got:\n%s", want, html)
				}
			}
		})
	}
}

func TestPageWithLayout(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	// Create a simple page using the layout
	page := templates.PageWithLayout("Test Page", "/", templates.TestContent("Hello World"))
	err := page.Render(context.Background(), &buf)
	if err != nil {
		t.Fatalf("failed to render: %v", err)
	}

	html := buf.String()

	wantContains := []string{
		"<!doctype html>",
		"<title>Test Page</title>",
		"<header",
		"<nav",
		"<main",
		"Hello World",
		"<footer",
		"</html>",
		// Accessibility features
		"href=\"#main-content\"",     // Skip link
		"id=\"main-content\"",        // Main content target
		"メインコンテンツへスキップ", // Skip link text
	}

	for _, want := range wantContains {
		if !strings.Contains(html, want) {
			t.Errorf("expected HTML to contain %q, got:\n%s", want, html)
		}
	}
}

func TestBaseLayoutWithConfig(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		config       templates.LayoutConfig
		wantContains []string
	}{
		{
			name: "uses custom feed URL in autodiscovery",
			config: templates.LayoutConfig{
				Title:   "Test Page",
				FeedURL: "https://example.com/feed.xml",
			},
			wantContains: []string{
				"application/rss+xml",
				"href=\"https://example.com/feed.xml\"",
			},
		},
		{
			name: "uses custom feed URL with different path",
			config: templates.LayoutConfig{
				Title:   "Test Page",
				FeedURL: "/custom/rss.xml",
			},
			wantContains: []string{
				"href=\"/custom/rss.xml\"",
			},
		},
		{
			name: "uses default feed URL when empty",
			config: templates.LayoutConfig{
				Title:   "Test Page",
				FeedURL: "",
			},
			wantContains: []string{
				"href=\"/feed.xml\"",
			},
		},
	}

	for _, tt := range tests {
		tt := tt // capture range variable for parallel test
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var buf bytes.Buffer
			err := templates.BaseLayoutWithConfig(tt.config).Render(context.Background(), &buf)
			if err != nil {
				t.Fatalf("failed to render: %v", err)
			}

			html := buf.String()

			for _, want := range tt.wantContains {
				if !strings.Contains(html, want) {
					t.Errorf("expected HTML to contain %q, got:\n%s", want, html)
				}
			}
		})
	}
}

func TestNavigationWithFeedURL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		currentPath  string
		feedURL      string
		wantContains []string
	}{
		{
			name:        "uses custom feed URL",
			currentPath: "/",
			feedURL:     "https://example.com/rss.xml",
			wantContains: []string{
				"href=\"https://example.com/rss.xml\"",
				"RSS",
			},
		},
		{
			name:        "uses default feed URL when empty",
			currentPath: "/",
			feedURL:     "",
			wantContains: []string{
				"href=\"/feed.xml\"",
			},
		},
	}

	for _, tt := range tests {
		tt := tt // capture range variable for parallel test
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var buf bytes.Buffer
			err := templates.NavigationWithFeedURL(tt.currentPath, tt.feedURL).Render(context.Background(), &buf)
			if err != nil {
				t.Fatalf("failed to render: %v", err)
			}

			html := buf.String()

			for _, want := range tt.wantContains {
				if !strings.Contains(html, want) {
					t.Errorf("expected HTML to contain %q, got:\n%s", want, html)
				}
			}
		})
	}
}

func TestPageWithLayoutConfig(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name              string
		config            templates.PageConfig
		wantContains      []string
		wantFeedURLCount  int // expected occurrences of the feed URL (head + nav)
	}{
		{
			name: "uses custom feed URL in both head and navigation",
			config: templates.PageConfig{
				Title:       "Test Page",
				CurrentPath: "/",
				FeedURL:     "https://example.com/feed.xml",
			},
			wantContains: []string{
				"<title>Test Page</title>",
				// Autodiscovery in head
				"application/rss+xml",
				// Navigation RSS link
				"RSS",
			},
			wantFeedURLCount: 2, // once in head autodiscovery, once in nav
		},
		{
			name: "uses default feed URL when empty",
			config: templates.PageConfig{
				Title:       "Test Page",
				CurrentPath: "/",
				FeedURL:     "",
			},
			wantContains: []string{
				"href=\"/feed.xml\"",
			},
			wantFeedURLCount: 2, // once in head, once in nav
		},
	}

	for _, tt := range tests {
		tt := tt // capture range variable for parallel test
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var buf bytes.Buffer
			page := templates.PageWithLayoutConfig(tt.config, templates.TestContent("Test Content"))
			err := page.Render(context.Background(), &buf)
			if err != nil {
				t.Fatalf("failed to render: %v", err)
			}

			html := buf.String()

			for _, want := range tt.wantContains {
				if !strings.Contains(html, want) {
					t.Errorf("expected HTML to contain %q, got:\n%s", want, html)
				}
			}

			// Verify feed URL appears in both head and navigation
			feedURL := tt.config.FeedURL
			if feedURL == "" {
				feedURL = "/feed.xml"
			}
			feedURLCount := strings.Count(html, "href=\""+feedURL+"\"")
			if feedURLCount != tt.wantFeedURLCount {
				t.Errorf("expected feed URL %q to appear %d times, got %d times in:\n%s",
					feedURL, tt.wantFeedURLCount, feedURLCount, html)
			}
		})
	}
}
