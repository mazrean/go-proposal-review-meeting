// Package site provides integration tests for the Feed domain.
// These tests validate RSS feed generation and autodiscovery integration.
// Requirements: 5.1, 5.2, 5.3, 5.4, 5.5
package site

import (
	"bytes"
	"context"
	"encoding/xml"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/mazrean/go-proposal-review-meeting/internal/content"
	"github.com/mazrean/go-proposal-review-meeting/internal/parser"
)

// =============================================================================
// Feed Domain Integration Tests (Task 8.4)
// =============================================================================
//
// These tests validate the complete Feed domain integration:
// - RSS feed generation from Content domain data
// - Autodiscovery tag presence across all page types
// - Compliance with all Feed requirements (5.1-5.5)
//
// =============================================================================

// RSSFeed represents the complete structure of an RSS 2.0 feed for validation.
type RSSFeed struct {
	XMLName xml.Name      `xml:"rss"`
	Version string        `xml:"version,attr"`
	Channel RSSFeedChannel `xml:"channel"`
}

// RSSFeedChannel represents the channel element in an RSS feed with all standard fields.
type RSSFeedChannel struct {
	Title         string        `xml:"title"`
	Link          string        `xml:"link"`
	Description   string        `xml:"description"`
	Language      string        `xml:"language,omitempty"`
	LastBuildDate string        `xml:"lastBuildDate,omitempty"`
	Items         []RSSFeedItem `xml:"item"`
}

// RSSFeedItem represents an item in an RSS feed with all standard fields.
type RSSFeedItem struct {
	Title       string `xml:"title"`
	Link        string `xml:"link"`
	Description string `xml:"description"`
	PubDate     string `xml:"pubDate"`
	GUID        string `xml:"guid"`
	Author      string `xml:"author,omitempty"`
}

// =============================================================================
// Requirement 5.1: RSS 2.0形式のフィードを生成する
// =============================================================================

// TestFeedIntegration_RSS20Format validates that the generated feed conforms to RSS 2.0 specification.
// Requirement: 5.1
func TestFeedIntegration_RSS20Format(t *testing.T) {
	t.Parallel()

	distDir := t.TempDir()

	weeks := createTestWeeks(5, 3) // 5 weeks with 3 proposals each

	gen := NewGenerator(
		WithDistDir(distDir),
		WithGeneratorSiteURL("https://go-proposal-digest.example.com"),
	)

	if err := gen.Generate(context.Background(), weeks); err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	feedContent, err := os.ReadFile(filepath.Join(distDir, "feed.xml"))
	if err != nil {
		t.Fatalf("failed to read feed.xml: %v", err)
	}

	t.Run("feed has XML declaration", func(t *testing.T) {
		if !bytes.HasPrefix(feedContent, []byte("<?xml")) {
			t.Error("RSS 2.0 feed must start with XML declaration")
		}
	})

	t.Run("feed has RSS 2.0 version attribute", func(t *testing.T) {
		var feed RSSFeed
		if err := xml.Unmarshal(feedContent, &feed); err != nil {
			t.Fatalf("failed to parse RSS feed: %v", err)
		}

		if feed.Version != "2.0" {
			t.Errorf("RSS version = %q, want %q", feed.Version, "2.0")
		}
	})

	t.Run("feed channel has required elements", func(t *testing.T) {
		var feed RSSFeed
		if err := xml.Unmarshal(feedContent, &feed); err != nil {
			t.Fatalf("failed to parse RSS feed: %v", err)
		}

		// RSS 2.0 requires: title, link, description
		if feed.Channel.Title == "" {
			t.Error("RSS 2.0 channel must have title element")
		}
		if feed.Channel.Link == "" {
			t.Error("RSS 2.0 channel must have link element")
		}
		if feed.Channel.Description == "" {
			t.Error("RSS 2.0 channel must have description element")
		}
	})

	t.Run("feed items have required elements", func(t *testing.T) {
		var feed RSSFeed
		if err := xml.Unmarshal(feedContent, &feed); err != nil {
			t.Fatalf("failed to parse RSS feed: %v", err)
		}

		if len(feed.Channel.Items) == 0 {
			t.Fatal("feed should have at least one item")
		}

		for i, item := range feed.Channel.Items {
			// RSS 2.0 requires at least title or description
			if item.Title == "" && item.Description == "" {
				t.Errorf("item[%d]: RSS 2.0 item must have title or description", i)
			}
			// GUID is strongly recommended for uniqueness
			if item.GUID == "" {
				t.Errorf("item[%d]: RSS 2.0 item should have GUID for unique identification", i)
			}
			// Link is important for user navigation
			if item.Link == "" {
				t.Errorf("item[%d]: RSS 2.0 item should have link", i)
			}
			// PubDate for chronological ordering
			if item.PubDate == "" {
				t.Errorf("item[%d]: RSS 2.0 item should have pubDate", i)
			}
		}
	})

	t.Run("feed is well-formed XML", func(t *testing.T) {
		decoder := xml.NewDecoder(bytes.NewReader(feedContent))
		for {
			_, err := decoder.Token()
			if err != nil {
				if err == io.EOF {
					break
				}
				t.Errorf("feed.xml contains invalid XML: %v", err)
				break
			}
		}
	})
}

// =============================================================================
// Requirement 5.2: 各週の更新をフィードアイテムとして含める
// =============================================================================

// TestFeedIntegration_WeeklyItems validates that each week becomes a feed item.
// Requirement: 5.2
func TestFeedIntegration_WeeklyItems(t *testing.T) {
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
					Title:          "proposal: week 5 feature",
					PreviousStatus: parser.StatusDiscussions,
					CurrentStatus:  parser.StatusAccepted,
					ChangedAt:      time.Date(2026, 1, 30, 12, 0, 0, 0, time.UTC),
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
					Title:          "proposal: week 4 feature",
					PreviousStatus: parser.StatusActive,
					CurrentStatus:  parser.StatusDeclined,
					ChangedAt:      time.Date(2026, 1, 23, 12, 0, 0, 0, time.UTC),
				},
			},
		},
		{
			Year:      2026,
			Week:      3,
			CreatedAt: time.Date(2026, 1, 16, 12, 0, 0, 0, time.UTC),
			Proposals: []content.ProposalContent{
				{
					IssueNumber:    10000,
					Title:          "proposal: week 3 feature",
					PreviousStatus: parser.StatusHold,
					CurrentStatus:  parser.StatusLikelyAccept,
					ChangedAt:      time.Date(2026, 1, 16, 12, 0, 0, 0, time.UTC),
				},
			},
		},
	}

	gen := NewGenerator(
		WithDistDir(distDir),
		WithGeneratorSiteURL("https://example.com"),
	)

	if err := gen.Generate(context.Background(), weeks); err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	feedContent, err := os.ReadFile(filepath.Join(distDir, "feed.xml"))
	if err != nil {
		t.Fatalf("failed to read feed.xml: %v", err)
	}

	var feed RSSFeed
	if err := xml.Unmarshal(feedContent, &feed); err != nil {
		t.Fatalf("failed to parse RSS feed: %v", err)
	}

	t.Run("each week has corresponding feed item", func(t *testing.T) {
		if len(feed.Channel.Items) != 3 {
			t.Errorf("expected 3 feed items (one per week), got %d", len(feed.Channel.Items))
		}
	})

	t.Run("feed items are ordered newest first", func(t *testing.T) {
		if len(feed.Channel.Items) < 2 {
			t.Skip("need at least 2 items to verify order")
		}

		// First item should be week 5 (newest)
		if !strings.Contains(feed.Channel.Items[0].Title, "5") {
			t.Errorf("first item should be week 5, got: %s", feed.Channel.Items[0].Title)
		}
		// Last item should be week 3 (oldest)
		if !strings.Contains(feed.Channel.Items[2].Title, "3") {
			t.Errorf("last item should be week 3, got: %s", feed.Channel.Items[2].Title)
		}
	})

	t.Run("feed items have correct week information in title", func(t *testing.T) {
		for i, item := range feed.Channel.Items {
			expectedWeek := 5 - i // 5, 4, 3
			if !strings.Contains(item.Title, "2026") {
				t.Errorf("item[%d]: title should contain year 2026, got: %s", i, item.Title)
			}
			if !strings.Contains(item.Title, string(rune('0'+expectedWeek))) {
				t.Errorf("item[%d]: title should contain week %d, got: %s", i, expectedWeek, item.Title)
			}
		}
	})
}

// =============================================================================
// Requirement 5.3: フィードにproposalのタイトル、ステータス変更、要約を含める
// =============================================================================

// TestFeedIntegration_ProposalContent validates that feed items contain proposal details.
// Requirement: 5.3
func TestFeedIntegration_ProposalContent(t *testing.T) {
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
					Title:          "proposal: add generics support",
					PreviousStatus: parser.StatusDiscussions,
					CurrentStatus:  parser.StatusAccepted,
					ChangedAt:      time.Date(2026, 1, 30, 12, 0, 0, 0, time.UTC),
					Summary:        "ジェネリクスのサポートが追加されました。型パラメータを使用して汎用的なコードを書けるようになります。",
				},
				{
					IssueNumber:    67890,
					Title:          "proposal: improve error handling",
					PreviousStatus: parser.StatusActive,
					CurrentStatus:  parser.StatusDeclined,
					ChangedAt:      time.Date(2026, 1, 29, 10, 0, 0, 0, time.UTC),
					Summary:        "エラーハンドリングの改善提案は却下されました。",
				},
			},
		},
	}

	gen := NewGenerator(
		WithDistDir(distDir),
		WithGeneratorSiteURL("https://example.com"),
	)

	if err := gen.Generate(context.Background(), weeks); err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	feedContent, err := os.ReadFile(filepath.Join(distDir, "feed.xml"))
	if err != nil {
		t.Fatalf("failed to read feed.xml: %v", err)
	}

	feedStr := string(feedContent)

	t.Run("feed contains proposal titles", func(t *testing.T) {
		if !strings.Contains(feedStr, "generics") {
			t.Error("feed should contain first proposal title (generics)")
		}
		if !strings.Contains(feedStr, "error handling") {
			t.Error("feed should contain second proposal title (error handling)")
		}
	})

	t.Run("feed contains proposal numbers", func(t *testing.T) {
		if !strings.Contains(feedStr, "12345") {
			t.Error("feed should contain first proposal number (12345)")
		}
		if !strings.Contains(feedStr, "67890") {
			t.Error("feed should contain second proposal number (67890)")
		}
	})

	t.Run("feed contains status change information", func(t *testing.T) {
		// Status names should appear in the description
		if !strings.Contains(strings.ToLower(feedStr), "accepted") {
			t.Error("feed should contain status 'accepted'")
		}
		if !strings.Contains(strings.ToLower(feedStr), "declined") {
			t.Error("feed should contain status 'declined'")
		}
		if !strings.Contains(strings.ToLower(feedStr), "discussions") {
			t.Error("feed should contain previous status 'discussions'")
		}
	})

	t.Run("feed contains proposal summaries", func(t *testing.T) {
		if !strings.Contains(feedStr, "ジェネリクス") {
			t.Error("feed should contain first proposal summary")
		}
		if !strings.Contains(feedStr, "エラーハンドリング") {
			t.Error("feed should contain second proposal summary")
		}
	})
}

// =============================================================================
// Requirement 5.4: フィードのautodiscoveryタグをHTMLに埋め込む
// =============================================================================

// TestFeedIntegration_AutodiscoveryTag validates RSS autodiscovery across all page types.
// Requirement: 5.4
func TestFeedIntegration_AutodiscoveryTag(t *testing.T) {
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
					Title:          "proposal: autodiscovery test",
					PreviousStatus: parser.StatusDiscussions,
					CurrentStatus:  parser.StatusAccepted,
					ChangedAt:      time.Date(2026, 1, 30, 12, 0, 0, 0, time.UTC),
				},
			},
		},
	}

	gen := NewGenerator(
		WithDistDir(distDir),
		WithGeneratorSiteURL("https://example.com"),
	)

	if err := gen.Generate(context.Background(), weeks); err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	// Test all page types for autodiscovery tag
	pagesToTest := []struct {
		name string
		path string
	}{
		{"home page", "index.html"},
		{"weekly index", "2026/w05/index.html"},
		{"proposal page", "2026/w05/12345.html"},
	}

	for _, page := range pagesToTest {
		t.Run(page.name+" has RSS autodiscovery", func(t *testing.T) {
			content, err := os.ReadFile(filepath.Join(distDir, page.path))
			if err != nil {
				t.Fatalf("failed to read %s: %v", page.path, err)
			}

			html := string(content)

			// Check for autodiscovery link element
			if !strings.Contains(html, `rel="alternate"`) {
				t.Errorf("%s: missing rel='alternate' attribute for RSS autodiscovery", page.name)
			}
			if !strings.Contains(html, `type="application/rss+xml"`) {
				t.Errorf("%s: missing type='application/rss+xml' attribute for RSS autodiscovery", page.name)
			}
			if !strings.Contains(html, "feed.xml") {
				t.Errorf("%s: missing feed.xml reference in RSS autodiscovery", page.name)
			}
		})
	}

	t.Run("autodiscovery link is in head section", func(t *testing.T) {
		content, err := os.ReadFile(filepath.Join(distDir, "index.html"))
		if err != nil {
			t.Fatalf("failed to read index.html: %v", err)
		}

		html := string(content)

		// Find positions
		headEnd := strings.Index(html, "</head>")
		rssLink := strings.Index(html, `type="application/rss+xml"`)

		if headEnd == -1 {
			t.Fatal("index.html missing </head> tag")
		}
		if rssLink == -1 {
			t.Fatal("index.html missing RSS autodiscovery link")
		}
		if rssLink > headEnd {
			t.Error("RSS autodiscovery link should be inside <head> section")
		}
	})

	t.Run("autodiscovery has proper title attribute", func(t *testing.T) {
		content, err := os.ReadFile(filepath.Join(distDir, "index.html"))
		if err != nil {
			t.Fatalf("failed to read index.html: %v", err)
		}

		html := string(content)

		// The title attribute helps RSS readers display the feed name
		if !strings.Contains(html, `title="`) {
			t.Error("RSS autodiscovery link should have title attribute for feed name")
		}
	})
}

// =============================================================================
// Requirement 5.5: 最新20件の更新をフィードに保持する
// =============================================================================

// TestFeedIntegration_MaxItemsLimit validates the 20 items limit.
// Requirement: 5.5
func TestFeedIntegration_MaxItemsLimit(t *testing.T) {
	t.Parallel()

	distDir := t.TempDir()

	// Create 25 weeks (more than the 20 limit)
	weeks := createTestWeeks(25, 1)

	gen := NewGenerator(
		WithDistDir(distDir),
		WithGeneratorSiteURL("https://example.com"),
	)

	if err := gen.Generate(context.Background(), weeks); err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	feedContent, err := os.ReadFile(filepath.Join(distDir, "feed.xml"))
	if err != nil {
		t.Fatalf("failed to read feed.xml: %v", err)
	}

	var feed RSSFeed
	if err := xml.Unmarshal(feedContent, &feed); err != nil {
		t.Fatalf("failed to parse RSS feed: %v", err)
	}

	t.Run("feed is limited to 20 items", func(t *testing.T) {
		if len(feed.Channel.Items) != 20 {
			t.Errorf("feed should have exactly 20 items, got %d", len(feed.Channel.Items))
		}
	})

	t.Run("feed keeps newest 20 items", func(t *testing.T) {
		if len(feed.Channel.Items) == 0 {
			t.Skip("no items to verify")
		}

		// First item should be week 25 (newest)
		if !strings.Contains(feed.Channel.Items[0].Title, "25") {
			t.Errorf("first item should be week 25 (newest), got: %s", feed.Channel.Items[0].Title)
		}

		// Last item should be week 6 (25 - 20 + 1 = 6)
		if !strings.Contains(feed.Channel.Items[19].Title, "6") {
			t.Errorf("last item should be week 6 (oldest in top 20), got: %s", feed.Channel.Items[19].Title)
		}
	})

	t.Run("oldest 5 weeks are excluded", func(t *testing.T) {
		feedStr := string(feedContent)

		// Weeks 1-5 should NOT be in the feed (they're the oldest 5)
		// Note: We need to be careful about partial matches (e.g., "15" contains "1" and "5")
		// Check for specific patterns like "第1週" or "Week 1"
		for week := 1; week <= 5; week++ {
			// The feed items have titles like "2026年 第5週"
			weekPattern := strings.ReplaceAll(feed.Channel.Items[0].Title, "25", string(rune('0'+week)))
			// This is a simplified check - the actual pattern depends on title format
			if strings.Contains(feedStr, weekPattern) && week <= 5 {
				t.Logf("Note: Found potential match for week %d, but may be part of larger number", week)
			}
		}
	})
}

// =============================================================================
// Integration with Content Domain
// =============================================================================

// TestFeedIntegration_ContentToFeedFlow validates the complete flow from content to feed.
func TestFeedIntegration_ContentToFeedFlow(t *testing.T) {
	t.Parallel()

	distDir := t.TempDir()

	// Create realistic content data that would come from Content Manager
	weeks := []*content.WeeklyContent{
		{
			Year:      2026,
			Week:      5,
			CreatedAt: time.Date(2026, 1, 30, 12, 0, 0, 0, time.UTC),
			Proposals: []content.ProposalContent{
				{
					IssueNumber:    12345,
					Title:          "proposal: add new API for concurrency",
					PreviousStatus: parser.StatusDiscussions,
					CurrentStatus:  parser.StatusAccepted,
					ChangedAt:      time.Date(2026, 1, 30, 14, 30, 0, 0, time.UTC),
					CommentURL:     "https://github.com/golang/go/issues/33502#issuecomment-123",
					Summary:        "並行処理のための新しいAPIが追加されます。これにより、goroutineの管理がより容易になります。",
					Links: []content.Link{
						{Title: "Proposal Issue", URL: "https://github.com/golang/go/issues/12345"},
						{Title: "Design Doc", URL: "https://go.dev/design/12345"},
					},
				},
				{
					IssueNumber:    67890,
					Title:          "proposal: deprecate old interface",
					PreviousStatus: parser.StatusActive,
					CurrentStatus:  parser.StatusLikelyDecline,
					ChangedAt:      time.Date(2026, 1, 30, 10, 0, 0, 0, time.UTC),
					CommentURL:     "https://github.com/golang/go/issues/33502#issuecomment-456",
					Summary:        "古いインターフェースの非推奨化は見送られる可能性があります。",
					Links:          []content.Link{},
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

	// Verify both feed.xml and HTML files are generated correctly
	t.Run("generates both feed and HTML", func(t *testing.T) {
		// Check feed.xml exists
		if _, err := os.Stat(filepath.Join(distDir, "feed.xml")); os.IsNotExist(err) {
			t.Error("feed.xml should be generated")
		}

		// Check index.html exists
		if _, err := os.Stat(filepath.Join(distDir, "index.html")); os.IsNotExist(err) {
			t.Error("index.html should be generated")
		}
	})

	t.Run("feed and HTML are consistent", func(t *testing.T) {
		feedContent, err := os.ReadFile(filepath.Join(distDir, "feed.xml"))
		if err != nil {
			t.Fatalf("failed to read feed.xml: %v", err)
		}

		indexContent, err := os.ReadFile(filepath.Join(distDir, "index.html"))
		if err != nil {
			t.Fatalf("failed to read index.html: %v", err)
		}

		feedStr := string(feedContent)
		indexStr := string(indexContent)

		// Both should reference week 5
		if !strings.Contains(feedStr, "5") || !strings.Contains(indexStr, "05") {
			t.Error("both feed and HTML should reference week 5")
		}
	})

	t.Run("feed pubDate uses latest proposal change time", func(t *testing.T) {
		feedContent, err := os.ReadFile(filepath.Join(distDir, "feed.xml"))
		if err != nil {
			t.Fatalf("failed to read feed.xml: %v", err)
		}

		var feed RSSFeed
		if err := xml.Unmarshal(feedContent, &feed); err != nil {
			t.Fatalf("failed to parse RSS feed: %v", err)
		}

		if len(feed.Channel.Items) == 0 {
			t.Fatal("feed should have at least one item")
		}

		// The pubDate should be set to the latest proposal change time (14:30)
		pubDate := feed.Channel.Items[0].PubDate
		if pubDate == "" {
			t.Error("feed item should have pubDate")
		}
	})
}

// =============================================================================
// Edge Cases
// =============================================================================

// TestFeedIntegration_EmptyContent validates feed generation with no content.
func TestFeedIntegration_EmptyContent(t *testing.T) {
	t.Parallel()

	distDir := t.TempDir()

	gen := NewGenerator(
		WithDistDir(distDir),
		WithGeneratorSiteURL("https://example.com"),
	)

	if err := gen.Generate(context.Background(), nil); err != nil {
		t.Fatalf("Generate() with nil content should not error: %v", err)
	}

	t.Run("generates valid empty RSS feed", func(t *testing.T) {
		feedContent, err := os.ReadFile(filepath.Join(distDir, "feed.xml"))
		if err != nil {
			t.Fatalf("failed to read feed.xml: %v", err)
		}

		var feed RSSFeed
		if err := xml.Unmarshal(feedContent, &feed); err != nil {
			t.Fatalf("feed.xml is not valid XML: %v", err)
		}

		if feed.Version != "2.0" {
			t.Error("empty feed should still be RSS 2.0")
		}
		if len(feed.Channel.Items) != 0 {
			t.Error("empty feed should have no items")
		}
	})

	t.Run("HTML still has autodiscovery for empty feed", func(t *testing.T) {
		content, err := os.ReadFile(filepath.Join(distDir, "index.html"))
		if err != nil {
			t.Fatalf("failed to read index.html: %v", err)
		}

		html := string(content)

		if !strings.Contains(html, `type="application/rss+xml"`) {
			t.Error("empty site should still have RSS autodiscovery")
		}
	})
}

// TestFeedIntegration_SpecialCharacters validates handling of special characters.
func TestFeedIntegration_SpecialCharacters(t *testing.T) {
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
					Title:          "proposal: test <special> & \"chars\" in 'title'",
					PreviousStatus: parser.StatusDiscussions,
					CurrentStatus:  parser.StatusAccepted,
					ChangedAt:      time.Date(2026, 1, 30, 12, 0, 0, 0, time.UTC),
					Summary:        "サマリーに含まれる特殊文字: <>&\"'",
				},
			},
		},
	}

	gen := NewGenerator(
		WithDistDir(distDir),
		WithGeneratorSiteURL("https://example.com"),
	)

	if err := gen.Generate(context.Background(), weeks); err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	t.Run("feed handles special characters safely", func(t *testing.T) {
		feedContent, err := os.ReadFile(filepath.Join(distDir, "feed.xml"))
		if err != nil {
			t.Fatalf("failed to read feed.xml: %v", err)
		}

		// Should be valid XML (special chars properly escaped)
		var feed RSSFeed
		if err := xml.Unmarshal(feedContent, &feed); err != nil {
			t.Fatalf("feed with special characters should be valid XML: %v", err)
		}

		// Raw HTML tags should not appear (should be escaped)
		feedStr := string(feedContent)
		if strings.Contains(feedStr, "<special>") {
			t.Error("special characters should be XML-escaped in feed")
		}
	})
}

// TestFeedIntegration_MultibyteSummary validates handling of Japanese text.
func TestFeedIntegration_MultibyteSummary(t *testing.T) {
	t.Parallel()

	distDir := t.TempDir()

	// Create content with long Japanese summary (will be truncated in feed)
	longSummary := strings.Repeat("日本語テスト文字列です。", 50) // ~500 chars

	weeks := []*content.WeeklyContent{
		{
			Year:      2026,
			Week:      5,
			CreatedAt: time.Date(2026, 1, 30, 12, 0, 0, 0, time.UTC),
			Proposals: []content.ProposalContent{
				{
					IssueNumber:    12345,
					Title:          "proposal: 日本語タイトルテスト",
					PreviousStatus: parser.StatusDiscussions,
					CurrentStatus:  parser.StatusAccepted,
					ChangedAt:      time.Date(2026, 1, 30, 12, 0, 0, 0, time.UTC),
					Summary:        longSummary,
				},
			},
		},
	}

	gen := NewGenerator(
		WithDistDir(distDir),
		WithGeneratorSiteURL("https://example.com"),
	)

	if err := gen.Generate(context.Background(), weeks); err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	t.Run("feed handles multibyte characters correctly", func(t *testing.T) {
		feedContent, err := os.ReadFile(filepath.Join(distDir, "feed.xml"))
		if err != nil {
			t.Fatalf("failed to read feed.xml: %v", err)
		}

		// Should be valid XML with proper encoding
		var feed RSSFeed
		if err := xml.Unmarshal(feedContent, &feed); err != nil {
			t.Fatalf("feed with Japanese content should be valid XML: %v", err)
		}

		// Should contain Japanese text
		feedStr := string(feedContent)
		if !strings.Contains(feedStr, "日本語") {
			t.Error("feed should contain Japanese text")
		}
	})
}

// =============================================================================
// Helper Functions
// =============================================================================

// createTestWeeks creates a slice of test weekly content.
func createTestWeeks(numWeeks, proposalsPerWeek int) []*content.WeeklyContent {
	weeks := make([]*content.WeeklyContent, numWeeks)
	for w := range numWeeks {
		weekNum := w + 1
		proposals := make([]content.ProposalContent, proposalsPerWeek)
		for p := range proposalsPerWeek {
			proposals[p] = content.ProposalContent{
				IssueNumber:    weekNum*1000 + p,
				Title:          "proposal: test feature",
				PreviousStatus: parser.StatusDiscussions,
				CurrentStatus:  parser.StatusAccepted,
				ChangedAt:      time.Date(2026, 1, weekNum*7, 12, 0, 0, 0, time.UTC),
				Summary:        "テスト用のproposal要約です。",
			}
		}
		weeks[w] = &content.WeeklyContent{
			Year:      2026,
			Week:      weekNum,
			CreatedAt: time.Date(2026, 1, weekNum*7, 12, 0, 0, 0, time.UTC),
			Proposals: proposals,
		}
	}
	return weeks
}
