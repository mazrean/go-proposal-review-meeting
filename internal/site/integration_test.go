// Package site provides integration tests for the static site generation pipeline.
// These tests verify that templ templates, UnoCSS, and esbuild work together correctly.
package site

import (
	"bytes"
	"context"
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/mazrean/go-proposal-review-meeting/internal/content"
	"github.com/mazrean/go-proposal-review-meeting/internal/parser"
)

// TestIntegration_FullSiteGeneration tests the complete site generation pipeline.
// It verifies that all components (templates, CSS, JS) are properly generated.
// Requirements: 4.1, 4.3
func TestIntegration_FullSiteGeneration(t *testing.T) {
	t.Parallel()

	distDir := t.TempDir()

	// Create comprehensive test data covering multiple scenarios
	weeks := []*content.WeeklyContent{
		{
			Year:      2026,
			Week:      5,
			CreatedAt: time.Date(2026, 1, 30, 12, 0, 0, 0, time.UTC),
			Proposals: []content.ProposalContent{
				{
					IssueNumber:    12345,
					Title:          "proposal: add generics support",
					PreviousStatus: parser.StatusDiscussions,
					CurrentStatus:  parser.StatusAccepted,
					ChangedAt:      time.Date(2026, 1, 30, 12, 0, 0, 0, time.UTC),
					CommentURL:     "https://github.com/golang/go/issues/33502#issuecomment-xxx",
					Summary:        "ジェネリクスのサポートを追加するproposalが承認されました。",
					Links: []content.Link{
						{Title: "proposal issue", URL: "https://github.com/golang/go/issues/12345"},
						{Title: "design doc", URL: "https://go.dev/design/12345"},
					},
				},
				{
					IssueNumber:    67890,
					Title:          "proposal: improve error handling",
					PreviousStatus: parser.StatusActive,
					CurrentStatus:  parser.StatusLikelyAccept,
					ChangedAt:      time.Date(2026, 1, 29, 10, 0, 0, 0, time.UTC),
					CommentURL:     "https://github.com/golang/go/issues/33502#issuecomment-yyy",
					Summary:        "エラーハンドリングの改善が検討されています。",
					Links: []content.Link{
						{Title: "proposal issue", URL: "https://github.com/golang/go/issues/67890"},
					},
				},
			},
		},
		{
			Year:      2026,
			Week:      4,
			CreatedAt: time.Date(2026, 1, 23, 12, 0, 0, 0, time.UTC),
			Proposals: []content.ProposalContent{
				{
					IssueNumber:    11111,
					Title:          "proposal: add new API",
					PreviousStatus: parser.StatusDiscussions,
					CurrentStatus:  parser.StatusDeclined,
					ChangedAt:      time.Date(2026, 1, 23, 12, 0, 0, 0, time.UTC),
					CommentURL:     "https://github.com/golang/go/issues/33502#issuecomment-zzz",
					Summary:        "新しいAPIの提案は却下されました。",
					Links:          []content.Link{},
				},
			},
		},
	}

	gen := NewGenerator(
		WithDistDir(distDir),
		WithGeneratorSiteURL("https://go-proposal-digest.example.com"),
	)

	ctx := context.Background()
	err := gen.Generate(ctx, weeks)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	// Verify all expected files are generated
	t.Run("generates all required files", func(t *testing.T) {
		expectedFiles := []string{
			"index.html",
			"feed.xml",
			"2026/w05/index.html",
			"2026/w05/12345.html",
			"2026/w05/67890.html",
			"2026/w04/index.html",
			"2026/w04/11111.html",
		}

		for _, file := range expectedFiles {
			path := filepath.Join(distDir, file)
			if _, err := os.Stat(path); os.IsNotExist(err) {
				t.Errorf("expected file %s was not created", file)
			}
		}
	})

	// Requirement 4.3: Home page links to each weekly index
	t.Run("home page links to weekly indexes", func(t *testing.T) {
		indexContent, err := os.ReadFile(filepath.Join(distDir, "index.html"))
		if err != nil {
			t.Fatalf("failed to read index.html: %v", err)
		}

		html := string(indexContent)

		// Should link to week 5
		if !strings.Contains(html, "2026/w05") && !strings.Contains(html, "/2026/w05") {
			t.Error("home page should link to week 5 index")
		}
		// Should link to week 4
		if !strings.Contains(html, "2026/w04") && !strings.Contains(html, "/2026/w04") {
			t.Error("home page should link to week 4 index")
		}
	})
}

// TestIntegration_HTMLStructureValidation validates that generated HTML has proper structure.
// Requirements: 4.1, 4.4, 4.5, 4.6
func TestIntegration_HTMLStructureValidation(t *testing.T) {
	t.Parallel()

	distDir := t.TempDir()

	week := &content.WeeklyContent{
		Year:      2026,
		Week:      5,
		CreatedAt: time.Now(),
		Proposals: []content.ProposalContent{
			{
				IssueNumber:    12345,
				Title:          "proposal: test feature",
				PreviousStatus: parser.StatusDiscussions,
				CurrentStatus:  parser.StatusAccepted,
				ChangedAt:      time.Now(),
				Summary:        "テスト用のproposal要約です。",
			},
		},
	}

	gen := NewGenerator(WithDistDir(distDir))

	if err := gen.Generate(context.Background(), []*content.WeeklyContent{week}); err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	t.Run("index.html has valid HTML5 structure", func(t *testing.T) {
		content, err := os.ReadFile(filepath.Join(distDir, "index.html"))
		if err != nil {
			t.Fatalf("failed to read index.html: %v", err)
		}

		html := string(content)

		// Validate HTML5 doctype
		if !strings.HasPrefix(html, "<!doctype html>") && !strings.HasPrefix(html, "<!DOCTYPE html>") {
			t.Error("index.html should start with HTML5 doctype")
		}

		// Validate essential HTML elements
		requiredElements := []string{
			"<html",
			"lang=\"ja\"",
			"<head>",
			"<meta charset=\"UTF-8\"",
			"<title>",
			"</head>",
			"<body",
			"</body>",
			"</html>",
		}

		for _, elem := range requiredElements {
			if !strings.Contains(html, elem) {
				t.Errorf("index.html missing required element: %s", elem)
			}
		}
	})

	// Requirement 4.4: UnoCSS stylesheet reference
	t.Run("HTML includes stylesheet reference for UnoCSS", func(t *testing.T) {
		content, err := os.ReadFile(filepath.Join(distDir, "index.html"))
		if err != nil {
			t.Fatalf("failed to read index.html: %v", err)
		}

		html := string(content)

		// Must have stylesheet link
		if !strings.Contains(html, "stylesheet") {
			t.Error("index.html should include stylesheet link")
		}
		if !strings.Contains(html, "styles.css") {
			t.Error("index.html should reference styles.css")
		}
	})

	// Requirement 4.5, 4.6: esbuild bundled Lit components
	t.Run("HTML includes ESM script for Lit components", func(t *testing.T) {
		content, err := os.ReadFile(filepath.Join(distDir, "index.html"))
		if err != nil {
			t.Fatalf("failed to read index.html: %v", err)
		}

		html := string(content)

		// Must have script tag with type="module" for ESM
		if !strings.Contains(html, `type="module"`) {
			t.Error("index.html should include script with type='module' for ESM")
		}
		if !strings.Contains(html, "components.js") {
			t.Error("index.html should reference components.js")
		}
	})

	t.Run("HTML includes RSS autodiscovery", func(t *testing.T) {
		content, err := os.ReadFile(filepath.Join(distDir, "index.html"))
		if err != nil {
			t.Fatalf("failed to read index.html: %v", err)
		}

		html := string(content)

		if !strings.Contains(html, `type="application/rss+xml"`) {
			t.Error("index.html should include RSS autodiscovery link")
		}
		if !strings.Contains(html, "feed.xml") {
			t.Error("index.html should reference feed.xml")
		}
	})
}

// TestIntegration_AccessibilityFeatures validates accessibility features in generated HTML.
// Requirements: 4.1
func TestIntegration_AccessibilityFeatures(t *testing.T) {
	t.Parallel()

	distDir := t.TempDir()

	week := &content.WeeklyContent{
		Year:      2026,
		Week:      5,
		CreatedAt: time.Now(),
		Proposals: []content.ProposalContent{
			{
				IssueNumber:    12345,
				Title:          "proposal: accessibility test",
				PreviousStatus: parser.StatusDiscussions,
				CurrentStatus:  parser.StatusAccepted,
				ChangedAt:      time.Now(),
			},
		},
	}

	gen := NewGenerator(WithDistDir(distDir))

	if err := gen.Generate(context.Background(), []*content.WeeklyContent{week}); err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	t.Run("includes skip link", func(t *testing.T) {
		content, err := os.ReadFile(filepath.Join(distDir, "index.html"))
		if err != nil {
			t.Fatalf("failed to read index.html: %v", err)
		}

		html := string(content)

		if !strings.Contains(html, `href="#main-content"`) {
			t.Error("index.html should include skip link to main content")
		}
	})

	t.Run("main content has proper landmark", func(t *testing.T) {
		content, err := os.ReadFile(filepath.Join(distDir, "index.html"))
		if err != nil {
			t.Fatalf("failed to read index.html: %v", err)
		}

		html := string(content)

		if !strings.Contains(html, `id="main-content"`) {
			t.Error("index.html should have main content landmark")
		}
		if !strings.Contains(html, "<main") {
			t.Error("index.html should use semantic main element")
		}
	})

	t.Run("navigation has proper landmark", func(t *testing.T) {
		content, err := os.ReadFile(filepath.Join(distDir, "index.html"))
		if err != nil {
			t.Fatalf("failed to read index.html: %v", err)
		}

		html := string(content)

		if !strings.Contains(html, "<nav") {
			t.Error("index.html should use semantic nav element")
		}
	})
}

// TestIntegration_ProposalPageContent validates individual proposal pages.
// Requirements: 4.1, 4.2, 4.3
func TestIntegration_ProposalPageContent(t *testing.T) {
	t.Parallel()

	distDir := t.TempDir()

	week := &content.WeeklyContent{
		Year:      2026,
		Week:      5,
		CreatedAt: time.Now(),
		Proposals: []content.ProposalContent{
			{
				IssueNumber:    12345,
				Title:          "proposal: important feature",
				PreviousStatus: parser.StatusDiscussions,
				CurrentStatus:  parser.StatusAccepted,
				ChangedAt:      time.Date(2026, 1, 30, 12, 0, 0, 0, time.UTC),
				CommentURL:     "https://github.com/golang/go/issues/33502#issuecomment-123",
				Summary:        "重要な機能に関するproposalが承認されました。",
				Links: []content.Link{
					{Title: "Proposal Issue", URL: "https://github.com/golang/go/issues/12345"},
				},
			},
		},
	}

	gen := NewGenerator(WithDistDir(distDir))

	if err := gen.Generate(context.Background(), []*content.WeeklyContent{week}); err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	t.Run("proposal page contains expected content", func(t *testing.T) {
		content, err := os.ReadFile(filepath.Join(distDir, "2026", "w05", "12345.html"))
		if err != nil {
			t.Fatalf("failed to read proposal page: %v", err)
		}

		html := string(content)

		// Check proposal title
		if !strings.Contains(html, "proposal: important feature") {
			t.Error("proposal page should contain the proposal title")
		}

		// Check summary
		if !strings.Contains(html, "重要な機能に関するproposalが承認されました") {
			t.Error("proposal page should contain the summary")
		}

		// Check links
		if !strings.Contains(html, "https://github.com/golang/go/issues/12345") {
			t.Error("proposal page should contain related links")
		}
	})

	// Requirement 4.2: Status change is visually distinct
	t.Run("proposal page shows status change with both statuses", func(t *testing.T) {
		content, err := os.ReadFile(filepath.Join(distDir, "2026", "w05", "12345.html"))
		if err != nil {
			t.Fatalf("failed to read proposal page: %v", err)
		}

		html := string(content)
		htmlLower := strings.ToLower(html)

		// Should show current status
		if !strings.Contains(htmlLower, "accepted") {
			t.Error("proposal page should display current status 'Accepted'")
		}

		// Should show previous status for status change context
		if !strings.Contains(htmlLower, "discussions") {
			t.Error("proposal page should display previous status 'Discussions' for status change")
		}
	})

	// Requirement 4.2: Status badge styling
	t.Run("proposal page has status badge with styling class", func(t *testing.T) {
		content, err := os.ReadFile(filepath.Join(distDir, "2026", "w05", "12345.html"))
		if err != nil {
			t.Fatalf("failed to read proposal page: %v", err)
		}

		html := string(content)

		// Must have status badge styling classes (rounded for pill shape, color classes)
		if !strings.Contains(html, "rounded") {
			t.Error("proposal page should have rounded class for status badge styling")
		}
		// Should have color styling for accepted status
		if !strings.Contains(html, "bg-green") {
			t.Error("proposal page should have green color class for accepted status badge")
		}
	})
}

// TestIntegration_WeeklyIndexContent validates weekly index pages.
// Requirements: 4.2, 4.3, 4.5
func TestIntegration_WeeklyIndexContent(t *testing.T) {
	t.Parallel()

	distDir := t.TempDir()

	week := &content.WeeklyContent{
		Year:      2026,
		Week:      5,
		CreatedAt: time.Now(),
		Proposals: []content.ProposalContent{
			{
				IssueNumber:    12345,
				Title:          "proposal: feature one",
				PreviousStatus: parser.StatusDiscussions,
				CurrentStatus:  parser.StatusAccepted,
				ChangedAt:      time.Now(),
			},
			{
				IssueNumber:    67890,
				Title:          "proposal: feature two",
				PreviousStatus: parser.StatusActive,
				CurrentStatus:  parser.StatusDeclined,
				ChangedAt:      time.Now(),
			},
		},
	}

	gen := NewGenerator(WithDistDir(distDir))

	if err := gen.Generate(context.Background(), []*content.WeeklyContent{week}); err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	t.Run("weekly index lists all proposals", func(t *testing.T) {
		content, err := os.ReadFile(filepath.Join(distDir, "2026", "w05", "index.html"))
		if err != nil {
			t.Fatalf("failed to read weekly index: %v", err)
		}

		html := string(content)

		if !strings.Contains(html, "12345") {
			t.Error("weekly index should list proposal #12345")
		}
		if !strings.Contains(html, "67890") {
			t.Error("weekly index should list proposal #67890")
		}
	})

	// Requirement 4.3: Links to individual proposal pages
	t.Run("weekly index has links to individual proposals", func(t *testing.T) {
		content, err := os.ReadFile(filepath.Join(distDir, "2026", "w05", "index.html"))
		if err != nil {
			t.Fatalf("failed to read weekly index: %v", err)
		}

		html := string(content)

		// Should have links to individual proposal pages
		if !strings.Contains(html, "12345.html") && !strings.Contains(html, "12345\"") {
			t.Error("weekly index should link to proposal 12345 page")
		}
		if !strings.Contains(html, "67890.html") && !strings.Contains(html, "67890\"") {
			t.Error("weekly index should link to proposal 67890 page")
		}
	})

	// Requirement 4.3: Navigation back to home
	t.Run("weekly index links back to home", func(t *testing.T) {
		content, err := os.ReadFile(filepath.Join(distDir, "2026", "w05", "index.html"))
		if err != nil {
			t.Fatalf("failed to read weekly index: %v", err)
		}

		html := string(content)

		// Should have link to home page
		if !strings.Contains(html, `href="/"`) && !strings.Contains(html, `href="../"`) && !strings.Contains(html, `href="../../"`) {
			t.Error("weekly index should have navigation link to home page")
		}
	})

	// Requirement 4.2: Status badges with UnoCSS classes
	t.Run("weekly index shows status badges with classes", func(t *testing.T) {
		content, err := os.ReadFile(filepath.Join(distDir, "2026", "w05", "index.html"))
		if err != nil {
			t.Fatalf("failed to read weekly index: %v", err)
		}

		html := string(content)

		// Should have status badge styling classes (rounded for pill shape)
		if !strings.Contains(html, "rounded") {
			t.Error("weekly index should have rounded class for status badge styling")
		}
	})
}

// TestIntegration_RSSFeedValidity validates RSS feed structure and content.
func TestIntegration_RSSFeedValidity(t *testing.T) {
	t.Parallel()

	distDir := t.TempDir()

	weeks := []*content.WeeklyContent{
		{
			Year:      2026,
			Week:      5,
			CreatedAt: time.Date(2026, 1, 30, 12, 0, 0, 0, time.UTC),
			Proposals: []content.ProposalContent{
				{
					IssueNumber:    12345,
					Title:          "proposal: RSS test",
					PreviousStatus: parser.StatusDiscussions,
					CurrentStatus:  parser.StatusAccepted,
					ChangedAt:      time.Date(2026, 1, 30, 12, 0, 0, 0, time.UTC),
					Summary:        "RSSフィードのテスト用proposal。",
				},
			},
		},
	}

	gen := NewGenerator(
		WithDistDir(distDir),
		WithGeneratorSiteURL("https://go-proposal-digest.example.com"),
	)

	if err := gen.Generate(context.Background(), weeks); err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	t.Run("RSS feed is valid XML", func(t *testing.T) {
		content, err := os.ReadFile(filepath.Join(distDir, "feed.xml"))
		if err != nil {
			t.Fatalf("failed to read feed.xml: %v", err)
		}

		decoder := xml.NewDecoder(bytes.NewReader(content))
		for {
			_, err := decoder.Token()
			if err != nil {
				if err.Error() == "EOF" {
					break
				}
				t.Errorf("feed.xml is not valid XML: %v", err)
				break
			}
		}
	})

	t.Run("RSS feed has required elements", func(t *testing.T) {
		content, err := os.ReadFile(filepath.Join(distDir, "feed.xml"))
		if err != nil {
			t.Fatalf("failed to read feed.xml: %v", err)
		}

		feedStr := string(content)

		requiredElements := []string{
			"<?xml",
			"<rss",
			"version=\"2.0\"",
			"<channel>",
			"<title>",
			"<link>",
			"<description>",
			"<item>",
		}

		for _, elem := range requiredElements {
			if !strings.Contains(feedStr, elem) {
				t.Errorf("feed.xml missing required element: %s", elem)
			}
		}
	})

	t.Run("RSS feed contains proposal information", func(t *testing.T) {
		content, err := os.ReadFile(filepath.Join(distDir, "feed.xml"))
		if err != nil {
			t.Fatalf("failed to read feed.xml: %v", err)
		}

		feedStr := string(content)

		if !strings.Contains(feedStr, "12345") {
			t.Error("feed.xml should contain proposal number")
		}
		if !strings.Contains(feedStr, "RSS test") {
			t.Error("feed.xml should contain proposal title")
		}
	})
}

// TestIntegration_StatusBadgeClasses validates that status badges have correct CSS classes.
// Requirements: 4.2, 4.4
func TestIntegration_StatusBadgeClasses(t *testing.T) {
	t.Parallel()

	distDir := t.TempDir()

	// Test all status types to ensure each has appropriate badge styling
	week := &content.WeeklyContent{
		Year:      2026,
		Week:      5,
		CreatedAt: time.Now(),
		Proposals: []content.ProposalContent{
			{
				IssueNumber:    1,
				Title:          "proposal: accepted test",
				PreviousStatus: parser.StatusDiscussions,
				CurrentStatus:  parser.StatusAccepted,
				ChangedAt:      time.Now(),
			},
			{
				IssueNumber:    2,
				Title:          "proposal: declined test",
				PreviousStatus: parser.StatusDiscussions,
				CurrentStatus:  parser.StatusDeclined,
				ChangedAt:      time.Now(),
			},
			{
				IssueNumber:    3,
				Title:          "proposal: likely accept test",
				PreviousStatus: parser.StatusDiscussions,
				CurrentStatus:  parser.StatusLikelyAccept,
				ChangedAt:      time.Now(),
			},
			{
				IssueNumber:    4,
				Title:          "proposal: hold test",
				PreviousStatus: parser.StatusDiscussions,
				CurrentStatus:  parser.StatusHold,
				ChangedAt:      time.Now(),
			},
		},
	}

	gen := NewGenerator(WithDistDir(distDir))

	if err := gen.Generate(context.Background(), []*content.WeeklyContent{week}); err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	// Requirement 4.2: Each status type should have badge styling
	t.Run("status badges use pill styling classes", func(t *testing.T) {
		indexContent, err := os.ReadFile(filepath.Join(distDir, "2026", "w05", "index.html"))
		if err != nil {
			t.Fatalf("failed to read weekly index: %v", err)
		}

		html := string(indexContent)

		// Must have rounded class for pill-shaped badges
		if !strings.Contains(html, "rounded") {
			t.Error("weekly index must have rounded class for status badge styling (requirement 4.2)")
		}
		// Must have inline-flex for badge layout
		if !strings.Contains(html, "inline-flex") {
			t.Error("weekly index must have inline-flex class for status badge layout (requirement 4.2)")
		}
	})

	// Requirement 4.2, 4.4: Weekly index has status-specific color classes for all statuses
	t.Run("weekly index has status-specific color classes", func(t *testing.T) {
		indexContent, err := os.ReadFile(filepath.Join(distDir, "2026", "w05", "index.html"))
		if err != nil {
			t.Fatalf("failed to read weekly index: %v", err)
		}

		html := string(indexContent)

		// Each status type should have its specific color in the weekly index
		expectedColors := []struct {
			status string
			color  string
		}{
			{"Accepted", "bg-green-100"},
			{"Declined", "bg-red-100"},
			{"Likely Accept", "bg-emerald-100"},
			{"Hold", "bg-yellow-100"},
		}

		for _, ec := range expectedColors {
			if !strings.Contains(html, ec.color) {
				t.Errorf("weekly index should have %s color class for %s status badge", ec.color, ec.status)
			}
		}
	})

	// Requirement 4.2: Verify each status is displayed
	t.Run("all status types are displayed", func(t *testing.T) {
		indexContent, err := os.ReadFile(filepath.Join(distDir, "2026", "w05", "index.html"))
		if err != nil {
			t.Fatalf("failed to read weekly index: %v", err)
		}

		htmlLower := strings.ToLower(string(indexContent))

		statuses := []string{"accepted", "declined", "likely", "hold"}
		for _, status := range statuses {
			if !strings.Contains(htmlLower, status) {
				t.Errorf("weekly index should display status: %s", status)
			}
		}
	})

	// Requirement 4.2: Individual proposal pages show status change
	t.Run("proposal pages show status transition", func(t *testing.T) {
		// Check accepted proposal page
		acceptedContent, err := os.ReadFile(filepath.Join(distDir, "2026", "w05", "1.html"))
		if err != nil {
			t.Fatalf("failed to read proposal 1 page: %v", err)
		}

		htmlLower := strings.ToLower(string(acceptedContent))

		// Should show both previous (discussions) and current (accepted) status
		if !strings.Contains(htmlLower, "accepted") {
			t.Error("proposal page should show current status 'accepted'")
		}
		if !strings.Contains(htmlLower, "discussions") {
			t.Error("proposal page should show previous status 'discussions' for status change context")
		}
	})

	// Requirement 4.2, 4.4: Each status type has its distinct color styling
	t.Run("each status type has specific color class", func(t *testing.T) {
		// Test status-color mapping for each proposal page
		// Colors from statusBadgeClass in weekly.templ:
		// - Accepted: bg-green-100
		// - Declined: bg-red-100
		// - LikelyAccept: bg-emerald-100
		// - Hold: bg-yellow-100
		tests := []struct {
			issueNum      int
			status        string
			expectedColor string
		}{
			{1, "accepted", "bg-green-100"},
			{2, "declined", "bg-red-100"},
			{3, "likely accept", "bg-emerald-100"},
			{4, "hold", "bg-yellow-100"},
		}

		for _, tc := range tests {
			filePath := filepath.Join(distDir, "2026", "w05", fmt.Sprintf("%d.html", tc.issueNum))
			content, err := os.ReadFile(filePath)
			if err != nil {
				t.Fatalf("failed to read proposal %d page: %v", tc.issueNum, err)
			}

			html := string(content)

			// Verify status-specific color class exists
			if !strings.Contains(html, tc.expectedColor) {
				t.Errorf("proposal %d (%s status) should have %s color class, but not found in HTML",
					tc.issueNum, tc.status, tc.expectedColor)
			}
		}
	})
}

// TestIntegration_EmptyWeekHandling validates handling of weeks with no proposals.
func TestIntegration_EmptyWeekHandling(t *testing.T) {
	t.Parallel()

	distDir := t.TempDir()

	gen := NewGenerator(WithDistDir(distDir))

	// Empty content should still generate a valid site
	if err := gen.Generate(context.Background(), nil); err != nil {
		t.Fatalf("Generate() with nil content should not error: %v", err)
	}

	t.Run("generates index even with no content", func(t *testing.T) {
		if _, err := os.Stat(filepath.Join(distDir, "index.html")); os.IsNotExist(err) {
			t.Error("index.html should be created even with no content")
		}
	})

	t.Run("generates valid RSS feed with no content", func(t *testing.T) {
		content, err := os.ReadFile(filepath.Join(distDir, "feed.xml"))
		if err != nil {
			t.Fatalf("failed to read feed.xml: %v", err)
		}

		// Should still be valid XML
		decoder := xml.NewDecoder(bytes.NewReader(content))
		for {
			_, err := decoder.Token()
			if err != nil {
				if err.Error() == "EOF" {
					break
				}
				t.Errorf("feed.xml is not valid XML: %v", err)
				break
			}
		}
	})

	// Requirement 4.4, 4.5, 4.6: Assets still referenced in empty site
	t.Run("empty site still references CSS and JS assets", func(t *testing.T) {
		content, err := os.ReadFile(filepath.Join(distDir, "index.html"))
		if err != nil {
			t.Fatalf("failed to read index.html: %v", err)
		}

		html := string(content)

		if !strings.Contains(html, "styles.css") {
			t.Error("empty site should still reference styles.css")
		}
		if !strings.Contains(html, "components.js") {
			t.Error("empty site should still reference components.js")
		}
	})
}

// TestIntegration_LargeDatasetFileCount tests file generation with many proposals.
// Uses deterministic file count assertion instead of timing-based checks.
func TestIntegration_LargeDatasetFileCount(t *testing.T) {
	t.Parallel()

	distDir := t.TempDir()

	// Create 10 weeks with 5 proposals each (50 total)
	weeks := make([]*content.WeeklyContent, 10)
	for w := range 10 {
		proposals := make([]content.ProposalContent, 5)
		for p := range 5 {
			proposals[p] = content.ProposalContent{
				IssueNumber:    w*1000 + p,
				Title:          "proposal: performance test",
				PreviousStatus: parser.StatusDiscussions,
				CurrentStatus:  parser.StatusAccepted,
				ChangedAt:      time.Now(),
				Summary:        "パフォーマンステスト用のproposal。",
			}
		}
		weeks[w] = &content.WeeklyContent{
			Year:      2026,
			Week:      w + 1,
			CreatedAt: time.Now(),
			Proposals: proposals,
		}
	}

	gen := NewGenerator(WithDistDir(distDir))

	if err := gen.Generate(context.Background(), weeks); err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	t.Run("generates correct number of HTML files", func(t *testing.T) {
		// Count HTML files
		var htmlCount int
		err := filepath.Walk(distDir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if !info.IsDir() && strings.HasSuffix(path, ".html") {
				htmlCount++
			}
			return nil
		})
		if err != nil {
			t.Fatalf("failed to walk dist directory: %v", err)
		}

		// Expected: 1 index + 10 weekly indexes + 50 proposal pages = 61
		expectedCount := 1 + 10 + 50
		if htmlCount != expectedCount {
			t.Errorf("expected %d HTML files, got %d", expectedCount, htmlCount)
		}
	})

	t.Run("generates RSS feed", func(t *testing.T) {
		feedPath := filepath.Join(distDir, "feed.xml")
		if _, err := os.Stat(feedPath); os.IsNotExist(err) {
			t.Error("feed.xml should be created")
		}
	})
}

// TestIntegration_NavigationLinks validates navigation between pages.
// Requirements: 4.3
func TestIntegration_NavigationLinks(t *testing.T) {
	t.Parallel()

	distDir := t.TempDir()

	weeks := []*content.WeeklyContent{
		{
			Year:      2026,
			Week:      5,
			CreatedAt: time.Date(2026, 1, 30, 12, 0, 0, 0, time.UTC),
			Proposals: []content.ProposalContent{
				{
					IssueNumber:    12345,
					Title:          "proposal: test",
					PreviousStatus: parser.StatusDiscussions,
					CurrentStatus:  parser.StatusAccepted,
					ChangedAt:      time.Now(),
				},
			},
		},
		{
			Year:      2026,
			Week:      4,
			CreatedAt: time.Date(2026, 1, 23, 12, 0, 0, 0, time.UTC),
			Proposals: []content.ProposalContent{
				{
					IssueNumber:    11111,
					Title:          "proposal: older test",
					PreviousStatus: parser.StatusActive,
					CurrentStatus:  parser.StatusDeclined,
					ChangedAt:      time.Now(),
				},
			},
		},
	}

	gen := NewGenerator(WithDistDir(distDir))

	if err := gen.Generate(context.Background(), weeks); err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	// Requirement 4.3: Home page lists weeks in order (newest first)
	t.Run("home page lists weeks with week 5 before week 4", func(t *testing.T) {
		content, err := os.ReadFile(filepath.Join(distDir, "index.html"))
		if err != nil {
			t.Fatalf("failed to read index.html: %v", err)
		}

		html := string(content)

		// Week 5 should appear before Week 4 in the HTML (newest first)
		week5Pos := strings.Index(html, "w05")
		week4Pos := strings.Index(html, "w04")

		if week5Pos == -1 {
			t.Error("home page should contain link to week 5")
		}
		if week4Pos == -1 {
			t.Error("home page should contain link to week 4")
		}
		if week5Pos != -1 && week4Pos != -1 && week5Pos > week4Pos {
			t.Error("home page should list week 5 before week 4 (newest first)")
		}
	})

	// Requirement 4.3: Weekly pages have navigation
	t.Run("weekly pages have home navigation", func(t *testing.T) {
		week5Content, err := os.ReadFile(filepath.Join(distDir, "2026", "w05", "index.html"))
		if err != nil {
			t.Fatalf("failed to read week 5 index: %v", err)
		}

		html := string(week5Content)

		// Should have link back to home
		if !strings.Contains(html, `href="/"`) {
			t.Error("weekly index should have link to home page")
		}
	})

	// Requirement 4.3: Proposal pages link back to weekly index
	t.Run("proposal pages have weekly index navigation", func(t *testing.T) {
		proposalContent, err := os.ReadFile(filepath.Join(distDir, "2026", "w05", "12345.html"))
		if err != nil {
			t.Fatalf("failed to read proposal page: %v", err)
		}

		html := string(proposalContent)

		// Should have navigation (either via header/nav or breadcrumb)
		if !strings.Contains(html, "<nav") {
			t.Error("proposal page should have navigation element")
		}
	})
}

// TestIntegration_MarkdownToHTMLPipeline validates the complete pipeline from Markdown files to HTML output.
// This test creates Markdown content files using ContentManager, reads them back,
// and generates HTML using SiteGenerator to verify the full integration.
// Requirements: 4.1, 4.3 (Task 8.3)
func TestIntegration_MarkdownToHTMLPipeline(t *testing.T) {
	t.Parallel()

	// Create temporary directories for content and dist
	contentDir := t.TempDir()
	distDir := t.TempDir()

	// Create 5 proposals for this test (as specified in task 8.3)
	proposals := []content.ProposalContent{
		{
			IssueNumber:    10001,
			Title:          "proposal: add new feature A",
			PreviousStatus: parser.StatusDiscussions,
			CurrentStatus:  parser.StatusAccepted,
			ChangedAt:      time.Date(2026, 1, 30, 12, 0, 0, 0, time.UTC),
			CommentURL:     "https://github.com/golang/go/issues/33502#issuecomment-001",
			Summary:        "機能Aの追加proposalが承認されました。この機能により、開発者体験が向上します。",
			Links: []content.Link{
				{Title: "proposal issue", URL: "https://github.com/golang/go/issues/10001"},
			},
		},
		{
			IssueNumber:    10002,
			Title:          "proposal: improve feature B",
			PreviousStatus: parser.StatusActive,
			CurrentStatus:  parser.StatusLikelyAccept,
			ChangedAt:      time.Date(2026, 1, 30, 11, 0, 0, 0, time.UTC),
			CommentURL:     "https://github.com/golang/go/issues/33502#issuecomment-002",
			Summary:        "機能Bの改善が前向きに検討されています。パフォーマンス向上が期待されます。",
			Links: []content.Link{
				{Title: "proposal issue", URL: "https://github.com/golang/go/issues/10002"},
			},
		},
		{
			IssueNumber:    10003,
			Title:          "proposal: deprecate feature C",
			PreviousStatus: parser.StatusDiscussions,
			CurrentStatus:  parser.StatusDeclined,
			ChangedAt:      time.Date(2026, 1, 30, 10, 0, 0, 0, time.UTC),
			CommentURL:     "https://github.com/golang/go/issues/33502#issuecomment-003",
			Summary:        "機能Cの廃止proposalは却下されました。後方互換性の観点から維持されます。",
			Links: []content.Link{
				{Title: "proposal issue", URL: "https://github.com/golang/go/issues/10003"},
			},
		},
		{
			IssueNumber:    10004,
			Title:          "proposal: add new API D",
			PreviousStatus: parser.StatusDiscussions,
			CurrentStatus:  parser.StatusHold,
			ChangedAt:      time.Date(2026, 1, 30, 9, 0, 0, 0, time.UTC),
			CommentURL:     "https://github.com/golang/go/issues/33502#issuecomment-004",
			Summary:        "API Dの追加は保留となりました。さらなる議論が必要です。",
			Links: []content.Link{
				{Title: "proposal issue", URL: "https://github.com/golang/go/issues/10004"},
			},
		},
		{
			IssueNumber:    10005,
			Title:          "proposal: refactor module E",
			PreviousStatus: parser.StatusActive,
			CurrentStatus:  parser.StatusAccepted,
			ChangedAt:      time.Date(2026, 1, 30, 8, 0, 0, 0, time.UTC),
			CommentURL:     "https://github.com/golang/go/issues/33502#issuecomment-005",
			Summary:        "モジュールEのリファクタリングが承認されました。コードの保守性が向上します。",
			Links: []content.Link{
				{Title: "proposal issue", URL: "https://github.com/golang/go/issues/10005"},
			},
		},
	}

	weeklyContent := &content.WeeklyContent{
		Year:      2026,
		Week:      5,
		CreatedAt: time.Date(2026, 1, 30, 12, 0, 0, 0, time.UTC),
		Proposals: proposals,
	}

	// Step 1: Write Markdown content files using ContentManager
	contentMgr := content.NewManager(content.WithBaseDir(contentDir))
	if err := contentMgr.WriteContent(weeklyContent); err != nil {
		t.Fatalf("ContentManager.WriteContent() error = %v", err)
	}

	// Step 2: Read content back from Markdown files
	readContent, err := contentMgr.ListAllWeeks()
	if err != nil {
		t.Fatalf("ContentManager.ListAllWeeks() error = %v", err)
	}
	if len(readContent) == 0 {
		t.Fatal("ContentManager.ListAllWeeks() returned no content")
	}

	// Step 3: Generate HTML using SiteGenerator
	gen := NewGenerator(
		WithDistDir(distDir),
		WithGeneratorSiteURL("https://go-proposal-digest.example.com"),
	)

	if err := gen.Generate(context.Background(), readContent); err != nil {
		t.Fatalf("Generator.Generate() error = %v", err)
	}

	// Verify: All 5 proposal HTML files are generated
	t.Run("generates HTML for all 5 proposals", func(t *testing.T) {
		expectedProposalFiles := []string{
			"2026/w05/10001.html",
			"2026/w05/10002.html",
			"2026/w05/10003.html",
			"2026/w05/10004.html",
			"2026/w05/10005.html",
		}

		for _, file := range expectedProposalFiles {
			path := filepath.Join(distDir, file)
			if _, err := os.Stat(path); os.IsNotExist(err) {
				t.Errorf("expected proposal page %s was not created", file)
			}
		}
	})

	// Verify: Weekly index page is generated
	t.Run("generates weekly index page", func(t *testing.T) {
		indexPath := filepath.Join(distDir, "2026", "w05", "index.html")
		if _, err := os.Stat(indexPath); os.IsNotExist(err) {
			t.Error("weekly index page was not created")
		}
	})

	// Verify: Home page is generated with link to the week
	t.Run("generates home page with week link", func(t *testing.T) {
		homePath := filepath.Join(distDir, "index.html")
		htmlContent, err := os.ReadFile(homePath)
		if err != nil {
			t.Fatalf("failed to read home page: %v", err)
		}

		if !strings.Contains(string(htmlContent), "w05") {
			t.Error("home page should contain link to week 5")
		}
	})

	// Verify: Proposal content is correctly rendered in HTML
	t.Run("proposal content is correctly rendered", func(t *testing.T) {
		// Check first proposal page
		proposalPath := filepath.Join(distDir, "2026", "w05", "10001.html")
		htmlContent, err := os.ReadFile(proposalPath)
		if err != nil {
			t.Fatalf("failed to read proposal page: %v", err)
		}

		html := string(htmlContent)

		// Should contain title
		if !strings.Contains(html, "add new feature A") {
			t.Error("proposal page should contain the proposal title")
		}

		// Should contain summary
		if !strings.Contains(html, "機能Aの追加") {
			t.Error("proposal page should contain the summary text")
		}

		// Should contain status
		if !strings.Contains(strings.ToLower(html), "accepted") {
			t.Error("proposal page should contain the status")
		}
	})

	// Verify: Weekly index lists all 5 proposals
	t.Run("weekly index lists all 5 proposals", func(t *testing.T) {
		indexPath := filepath.Join(distDir, "2026", "w05", "index.html")
		htmlContent, err := os.ReadFile(indexPath)
		if err != nil {
			t.Fatalf("failed to read weekly index: %v", err)
		}

		html := string(htmlContent)

		// Check that all 5 proposal numbers are present
		proposalNumbers := []string{"10001", "10002", "10003", "10004", "10005"}
		for _, num := range proposalNumbers {
			if !strings.Contains(html, num) {
				t.Errorf("weekly index should list proposal #%s", num)
			}
		}
	})

	// Verify: RSS feed is generated with proposal information
	t.Run("RSS feed contains proposal information", func(t *testing.T) {
		feedPath := filepath.Join(distDir, "feed.xml")
		feedContent, err := os.ReadFile(feedPath)
		if err != nil {
			t.Fatalf("failed to read RSS feed: %v", err)
		}

		feedStr := string(feedContent)

		// Feed should be valid XML and contain proposal info
		if !strings.Contains(feedStr, "<rss") {
			t.Error("feed.xml should contain RSS element")
		}

		// Should contain at least one proposal reference
		if !strings.Contains(feedStr, "10001") && !strings.Contains(feedStr, "proposal") {
			t.Error("feed.xml should contain proposal information")
		}
	})

	// Verify: Total HTML file count (1 home + 1 weekly index + 5 proposals = 7)
	t.Run("correct total HTML file count", func(t *testing.T) {
		var htmlCount int
		err := filepath.Walk(distDir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if !info.IsDir() && strings.HasSuffix(path, ".html") {
				htmlCount++
			}
			return nil
		})
		if err != nil {
			t.Fatalf("failed to walk dist directory: %v", err)
		}

		expectedCount := 7 // 1 home + 1 weekly index + 5 proposal pages
		if htmlCount != expectedCount {
			t.Errorf("expected %d HTML files, got %d", expectedCount, htmlCount)
		}
	})
}

// =============================================================================
// Task 8.6: Content→Site統合テスト
// =============================================================================
//
// This test verifies the complete Content → Site → Feed pipeline integration.
// It uses ContentManager to create and persist 10 proposals, then generates
// the full static site including HTML pages and RSS feed.
//
// Requirements covered: 2.1, 2.2, 2.3, 4.1, 4.3, 5.1
// =============================================================================

// TestIntegration_ContentToSiteToFeed validates the complete Content → Site → Feed pipeline.
// This is a comprehensive integration test that:
// 1. Creates 10 proposals using ContentManager
// 2. Persists them to Markdown files
// 3. Reads them back using ListAllWeeks
// 4. Generates HTML/RSS using SiteGenerator
// 5. Verifies all 10 proposals are correctly rendered
//
// Requirements: 2.1 (週ごとディレクトリ), 2.2 (proposal別MDファイル), 2.3 (MDメタデータ),
//
//	4.1 (HTML生成), 4.3 (ページ生成), 5.1 (RSS生成)
func TestIntegration_ContentToSiteToFeed(t *testing.T) {
	t.Parallel()

	// Create temporary directories
	contentDir := t.TempDir()
	summariesDir := t.TempDir()
	distDir := t.TempDir()

	// Create 10 proposals across 2 weeks for comprehensive testing
	// Week 5: 6 proposals (various statuses)
	// Week 4: 4 proposals (various statuses)
	baseTime := time.Date(2026, 1, 30, 12, 0, 0, 0, time.UTC)

	week5Proposals := []content.ProposalContent{
		{
			IssueNumber:    50001,
			Title:          "proposal: add structured concurrency primitives",
			PreviousStatus: parser.StatusDiscussions,
			CurrentStatus:  parser.StatusAccepted,
			ChangedAt:      baseTime,
			CommentURL:     "https://github.com/golang/go/issues/33502#issuecomment-50001",
			Summary:        "構造化並行処理のプリミティブを追加するproposalが承認されました。これにより、goroutineのライフサイクル管理が容易になります。",
			Links: []content.Link{
				{Title: "proposal issue", URL: "https://github.com/golang/go/issues/50001"},
			},
		},
		{
			IssueNumber:    50002,
			Title:          "proposal: extend context package with new features",
			PreviousStatus: parser.StatusActive,
			CurrentStatus:  parser.StatusLikelyAccept,
			ChangedAt:      baseTime.Add(-1 * time.Hour),
			CommentURL:     "https://github.com/golang/go/issues/33502#issuecomment-50002",
			Summary:        "contextパッケージの機能拡張が前向きに検討されています。新しいキャンセレーション機能が追加される見込みです。",
			Links: []content.Link{
				{Title: "proposal issue", URL: "https://github.com/golang/go/issues/50002"},
			},
		},
		{
			IssueNumber:    50003,
			Title:          "proposal: improve error wrapping semantics",
			PreviousStatus: parser.StatusDiscussions,
			CurrentStatus:  parser.StatusDeclined,
			ChangedAt:      baseTime.Add(-2 * time.Hour),
			CommentURL:     "https://github.com/golang/go/issues/33502#issuecomment-50003",
			Summary:        "エラーラッピングのセマンティクス改善は却下されました。現在のerrors.Is()とerrors.As()で十分と判断されています。",
			Links: []content.Link{
				{Title: "proposal issue", URL: "https://github.com/golang/go/issues/50003"},
			},
		},
		{
			IssueNumber:    50004,
			Title:          "proposal: add iterator protocol to standard library",
			PreviousStatus: parser.StatusDiscussions,
			CurrentStatus:  parser.StatusHold,
			ChangedAt:      baseTime.Add(-3 * time.Hour),
			CommentURL:     "https://github.com/golang/go/issues/33502#issuecomment-50004",
			Summary:        "標準ライブラリへのイテレータプロトコル追加は保留されました。range over funcの導入後に再検討される予定です。",
			Links: []content.Link{
				{Title: "proposal issue", URL: "https://github.com/golang/go/issues/50004"},
			},
		},
		{
			IssueNumber:    50005,
			Title:          "proposal: simplify module dependency resolution",
			PreviousStatus: parser.StatusActive,
			CurrentStatus:  parser.StatusAccepted,
			ChangedAt:      baseTime.Add(-4 * time.Hour),
			CommentURL:     "https://github.com/golang/go/issues/33502#issuecomment-50005",
			Summary:        "モジュール依存関係解決の簡素化が承認されました。go.modの記述がより直感的になります。",
			Links: []content.Link{
				{Title: "proposal issue", URL: "https://github.com/golang/go/issues/50005"},
			},
		},
		{
			IssueNumber:    50006,
			Title:          "proposal: add new sync primitives for channels",
			PreviousStatus: parser.StatusDiscussions,
			CurrentStatus:  parser.StatusLikelyDecline,
			ChangedAt:      baseTime.Add(-5 * time.Hour),
			CommentURL:     "https://github.com/golang/go/issues/33502#issuecomment-50006",
			Summary:        "チャネル用の新しい同期プリミティブは却下される見込みです。既存のselectで十分とされています。",
			Links: []content.Link{
				{Title: "proposal issue", URL: "https://github.com/golang/go/issues/50006"},
			},
		},
	}

	week4Proposals := []content.ProposalContent{
		{
			IssueNumber:    40001,
			Title:          "proposal: enhance testing package with subtests",
			PreviousStatus: parser.StatusDiscussions,
			CurrentStatus:  parser.StatusAccepted,
			ChangedAt:      baseTime.Add(-7 * 24 * time.Hour),
			CommentURL:     "https://github.com/golang/go/issues/33502#issuecomment-40001",
			Summary:        "testingパッケージへのサブテスト機能の強化が承認されました。テストの構造化がより容易になります。",
			Links: []content.Link{
				{Title: "proposal issue", URL: "https://github.com/golang/go/issues/40001"},
			},
		},
		{
			IssueNumber:    40002,
			Title:          "proposal: add JSON streaming API",
			PreviousStatus: parser.StatusActive,
			CurrentStatus:  parser.StatusLikelyAccept,
			ChangedAt:      baseTime.Add(-7*24*time.Hour - 1*time.Hour),
			CommentURL:     "https://github.com/golang/go/issues/33502#issuecomment-40002",
			Summary:        "JSONストリーミングAPIの追加が前向きに検討されています。大規模なJSONデータの効率的な処理が可能になります。",
			Links: []content.Link{
				{Title: "proposal issue", URL: "https://github.com/golang/go/issues/40002"},
			},
		},
		{
			IssueNumber:    40003,
			Title:          "proposal: deprecate ioutil package",
			PreviousStatus: parser.StatusDiscussions,
			CurrentStatus:  parser.StatusAccepted,
			ChangedAt:      baseTime.Add(-7*24*time.Hour - 2*time.Hour),
			CommentURL:     "https://github.com/golang/go/issues/33502#issuecomment-40003",
			Summary:        "ioutilパッケージの非推奨化が承認されました。io、osパッケージへの移行が推奨されます。",
			Links: []content.Link{
				{Title: "proposal issue", URL: "https://github.com/golang/go/issues/40003"},
			},
		},
		{
			IssueNumber:    40004,
			Title:          "proposal: improve reflect package performance",
			PreviousStatus: parser.StatusActive,
			CurrentStatus:  parser.StatusHold,
			ChangedAt:      baseTime.Add(-7*24*time.Hour - 3*time.Hour),
			CommentURL:     "https://github.com/golang/go/issues/33502#issuecomment-40004",
			Summary:        "reflectパッケージのパフォーマンス改善は保留されました。Go 2との互換性を検討中です。",
			Links: []content.Link{
				{Title: "proposal issue", URL: "https://github.com/golang/go/issues/40004"},
			},
		},
	}

	// Step 1: Create ContentManager and write content
	contentMgr := content.NewManager(
		content.WithBaseDir(contentDir),
		content.WithSummariesDir(summariesDir),
	)

	// Create weekly content structures
	week5Content := &content.WeeklyContent{
		Year:      2026,
		Week:      5,
		CreatedAt: baseTime,
		Proposals: week5Proposals,
	}
	week4Content := &content.WeeklyContent{
		Year:      2026,
		Week:      4,
		CreatedAt: baseTime.Add(-7 * 24 * time.Hour),
		Proposals: week4Proposals,
	}

	// Step 2: Write content to disk (Requirement 2.1, 2.2)
	if err := contentMgr.WriteContent(week5Content); err != nil {
		t.Fatalf("WriteContent(week5) error = %v", err)
	}
	if err := contentMgr.WriteContent(week4Content); err != nil {
		t.Fatalf("WriteContent(week4) error = %v", err)
	}

	// Step 3: Read content back using ListAllWeeks (Requirement 2.3)
	weeks, err := contentMgr.ListAllWeeks()
	if err != nil {
		t.Fatalf("ListAllWeeks() error = %v", err)
	}

	// Verify content was persisted correctly
	t.Run("Req 2.1-2.3: content is correctly persisted and readable", func(t *testing.T) {
		if len(weeks) != 2 {
			t.Errorf("expected 2 weeks, got %d", len(weeks))
		}

		// Count total proposals
		var totalProposals int
		for _, w := range weeks {
			totalProposals += len(w.Proposals)
		}
		if totalProposals != 10 {
			t.Errorf("expected 10 proposals total, got %d", totalProposals)
		}

		// Verify week ordering (newest first)
		if weeks[0].Week != 5 || weeks[1].Week != 4 {
			t.Error("weeks should be sorted newest first")
		}
	})

	// Step 4: Generate site using SiteGenerator (Requirement 4.1, 4.3)
	gen := NewGenerator(
		WithDistDir(distDir),
		WithGeneratorSiteURL("https://go-proposal-digest.example.com"),
	)

	ctx := context.Background()
	if err := gen.Generate(ctx, weeks); err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	// Step 5: Verify all generated files (Requirement 4.1, 4.3)
	t.Run("Req 4.1, 4.3: generates all required HTML files", func(t *testing.T) {
		expectedFiles := []string{
			"index.html",
			"feed.xml",
			// Week 5 files
			"2026/w05/index.html",
			"2026/w05/50001.html",
			"2026/w05/50002.html",
			"2026/w05/50003.html",
			"2026/w05/50004.html",
			"2026/w05/50005.html",
			"2026/w05/50006.html",
			// Week 4 files
			"2026/w04/index.html",
			"2026/w04/40001.html",
			"2026/w04/40002.html",
			"2026/w04/40003.html",
			"2026/w04/40004.html",
		}

		for _, file := range expectedFiles {
			path := filepath.Join(distDir, file)
			if _, err := os.Stat(path); os.IsNotExist(err) {
				t.Errorf("expected file %s was not created", file)
			}
		}
	})

	// Verify correct HTML file count
	t.Run("generates correct total HTML file count", func(t *testing.T) {
		var htmlCount int
		err := filepath.Walk(distDir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if !info.IsDir() && strings.HasSuffix(path, ".html") {
				htmlCount++
			}
			return nil
		})
		if err != nil {
			t.Fatalf("failed to walk dist directory: %v", err)
		}

		// Expected: 1 home + 2 weekly indexes + 10 proposal pages = 13
		expectedCount := 1 + 2 + 10
		if htmlCount != expectedCount {
			t.Errorf("expected %d HTML files, got %d", expectedCount, htmlCount)
		}
	})

	// Verify home page content links to both weeks
	t.Run("Req 4.3: home page links to all weeks", func(t *testing.T) {
		homeContent, err := os.ReadFile(filepath.Join(distDir, "index.html"))
		if err != nil {
			t.Fatalf("failed to read index.html: %v", err)
		}

		html := string(homeContent)

		if !strings.Contains(html, "w05") {
			t.Error("home page should link to week 5")
		}
		if !strings.Contains(html, "w04") {
			t.Error("home page should link to week 4")
		}
	})

	// Verify weekly index lists all proposals
	t.Run("Req 4.3: weekly index lists all proposals", func(t *testing.T) {
		week5Index, err := os.ReadFile(filepath.Join(distDir, "2026", "w05", "index.html"))
		if err != nil {
			t.Fatalf("failed to read week 5 index: %v", err)
		}

		html := string(week5Index)

		// Check all 6 proposals are listed
		proposalNumbers := []string{"50001", "50002", "50003", "50004", "50005", "50006"}
		for _, num := range proposalNumbers {
			if !strings.Contains(html, num) {
				t.Errorf("week 5 index should list proposal #%s", num)
			}
		}
	})

	// Verify proposal page content
	t.Run("Req 4.1: proposal pages contain expected content", func(t *testing.T) {
		// Check one proposal page in detail
		proposalPage, err := os.ReadFile(filepath.Join(distDir, "2026", "w05", "50001.html"))
		if err != nil {
			t.Fatalf("failed to read proposal page: %v", err)
		}

		html := string(proposalPage)

		// Should contain title
		if !strings.Contains(html, "structured concurrency") {
			t.Error("proposal page should contain the proposal title")
		}

		// Should contain summary
		if !strings.Contains(html, "構造化並行処理") {
			t.Error("proposal page should contain the summary")
		}

		// Should contain status
		if !strings.Contains(strings.ToLower(html), "accepted") {
			t.Error("proposal page should contain the current status")
		}

		// Should contain previous status for context
		if !strings.Contains(strings.ToLower(html), "discussions") {
			t.Error("proposal page should contain the previous status")
		}
	})

	// Verify RSS feed (Requirement 5.1)
	t.Run("Req 5.1: RSS feed is generated with all weeks", func(t *testing.T) {
		feedContent, err := os.ReadFile(filepath.Join(distDir, "feed.xml"))
		if err != nil {
			t.Fatalf("failed to read feed.xml: %v", err)
		}

		feedStr := string(feedContent)

		// Verify RSS 2.0 format
		if !strings.Contains(feedStr, `version="2.0"`) {
			t.Error("feed should be RSS 2.0 format")
		}

		// Verify both weeks are in feed
		if !strings.Contains(feedStr, "5") {
			t.Error("feed should contain week 5")
		}
		if !strings.Contains(feedStr, "4") {
			t.Error("feed should contain week 4")
		}

		// Verify feed is valid XML
		decoder := xml.NewDecoder(bytes.NewReader(feedContent))
		for {
			_, err := decoder.Token()
			if err != nil {
				if err.Error() == "EOF" {
					break
				}
				t.Errorf("feed.xml is not valid XML: %v", err)
				break
			}
		}
	})

	// Verify RSS autodiscovery on all page types
	t.Run("Req 5.1: HTML pages include RSS autodiscovery", func(t *testing.T) {
		pagesToCheck := []string{
			"index.html",
			"2026/w05/index.html",
			"2026/w05/50001.html",
		}

		for _, pagePath := range pagesToCheck {
			pageContent, err := os.ReadFile(filepath.Join(distDir, pagePath))
			if err != nil {
				t.Fatalf("failed to read %s: %v", pagePath, err)
			}

			html := string(pageContent)

			if !strings.Contains(html, `type="application/rss+xml"`) {
				t.Errorf("%s should include RSS autodiscovery", pagePath)
			}
		}
	})

	// Verify status badges are present with styling
	t.Run("Req 4.1: status badges have styling classes", func(t *testing.T) {
		weeklyIndex, err := os.ReadFile(filepath.Join(distDir, "2026", "w05", "index.html"))
		if err != nil {
			t.Fatalf("failed to read weekly index: %v", err)
		}

		html := string(weeklyIndex)

		// Check for status badge styling classes
		if !strings.Contains(html, "rounded") {
			t.Error("weekly index should have rounded class for status badges")
		}

		// Check for status-specific colors (at least one should be present)
		hasStatusColor := strings.Contains(html, "bg-green") ||
			strings.Contains(html, "bg-red") ||
			strings.Contains(html, "bg-emerald") ||
			strings.Contains(html, "bg-yellow")
		if !hasStatusColor {
			t.Error("weekly index should have status-specific color classes")
		}
	})

	// Verify all status types are represented in output
	t.Run("all status types are represented in generated content", func(t *testing.T) {
		weeklyIndex, err := os.ReadFile(filepath.Join(distDir, "2026", "w05", "index.html"))
		if err != nil {
			t.Fatalf("failed to read weekly index: %v", err)
		}

		htmlLower := strings.ToLower(string(weeklyIndex))

		expectedStatuses := []string{"accepted", "declined", "hold", "likely"}
		for _, status := range expectedStatuses {
			if !strings.Contains(htmlLower, status) {
				t.Errorf("weekly index should contain status: %s", status)
			}
		}
	})

	// Verify navigation links
	t.Run("Req 4.3: navigation links are present", func(t *testing.T) {
		// Check weekly index has home link
		weeklyIndex, err := os.ReadFile(filepath.Join(distDir, "2026", "w05", "index.html"))
		if err != nil {
			t.Fatalf("failed to read weekly index: %v", err)
		}

		if !strings.Contains(string(weeklyIndex), `href="/"`) {
			t.Error("weekly index should have link to home page")
		}

		// Check proposal page has navigation
		proposalPage, err := os.ReadFile(filepath.Join(distDir, "2026", "w05", "50001.html"))
		if err != nil {
			t.Fatalf("failed to read proposal page: %v", err)
		}

		if !strings.Contains(string(proposalPage), "<nav") {
			t.Error("proposal page should have navigation element")
		}
	})
}

// TestIntegration_TemplDataBinding validates that templ correctly binds data across multiple weeks.
// Requirements: 4.1
func TestIntegration_TemplDataBinding(t *testing.T) {
	t.Parallel()

	distDir := t.TempDir()

	weeks := []*content.WeeklyContent{
		{
			Year:      2026,
			Week:      5,
			CreatedAt: time.Date(2026, 1, 30, 12, 0, 0, 0, time.UTC),
			Proposals: []content.ProposalContent{
				{
					IssueNumber:    12345,
					Title:          "proposal: unique title for week 5",
					PreviousStatus: parser.StatusDiscussions,
					CurrentStatus:  parser.StatusAccepted,
					ChangedAt:      time.Date(2026, 1, 30, 12, 0, 0, 0, time.UTC),
					Summary:        "週5のユニークなサマリー",
				},
			},
		},
		{
			Year:      2026,
			Week:      4,
			CreatedAt: time.Date(2026, 1, 23, 12, 0, 0, 0, time.UTC),
			Proposals: []content.ProposalContent{
				{
					IssueNumber:    11111,
					Title:          "proposal: unique title for week 4",
					PreviousStatus: parser.StatusActive,
					CurrentStatus:  parser.StatusDeclined,
					ChangedAt:      time.Date(2026, 1, 23, 12, 0, 0, 0, time.UTC),
					Summary:        "週4のユニークなサマリー",
				},
			},
		},
	}

	gen := NewGenerator(WithDistDir(distDir))

	if err := gen.Generate(context.Background(), weeks); err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	// Verify week 5 content appears only in week 5 pages
	t.Run("week 5 content is correctly isolated", func(t *testing.T) {
		week5Index, err := os.ReadFile(filepath.Join(distDir, "2026", "w05", "index.html"))
		if err != nil {
			t.Fatalf("failed to read week 5 index: %v", err)
		}

		if !strings.Contains(string(week5Index), "unique title for week 5") {
			t.Error("week 5 index should contain week 5 proposal title")
		}
		if strings.Contains(string(week5Index), "unique title for week 4") {
			t.Error("week 5 index should NOT contain week 4 proposal title")
		}
	})

	// Verify week 4 content appears only in week 4 pages
	t.Run("week 4 content is correctly isolated", func(t *testing.T) {
		week4Index, err := os.ReadFile(filepath.Join(distDir, "2026", "w04", "index.html"))
		if err != nil {
			t.Fatalf("failed to read week 4 index: %v", err)
		}

		if !strings.Contains(string(week4Index), "unique title for week 4") {
			t.Error("week 4 index should contain week 4 proposal title")
		}
		if strings.Contains(string(week4Index), "unique title for week 5") {
			t.Error("week 4 index should NOT contain week 5 proposal title")
		}
	})

	// Verify proposal pages have correct content
	t.Run("proposal pages have correct bound data", func(t *testing.T) {
		proposal5, err := os.ReadFile(filepath.Join(distDir, "2026", "w05", "12345.html"))
		if err != nil {
			t.Fatalf("failed to read proposal 12345 page: %v", err)
		}

		if !strings.Contains(string(proposal5), "週5のユニークなサマリー") {
			t.Error("proposal 12345 page should contain its unique summary")
		}

		proposal4, err := os.ReadFile(filepath.Join(distDir, "2026", "w04", "11111.html"))
		if err != nil {
			t.Fatalf("failed to read proposal 11111 page: %v", err)
		}

		if !strings.Contains(string(proposal4), "週4のユニークなサマリー") {
			t.Error("proposal 11111 page should contain its unique summary")
		}
	})
}
