// Package site provides functionality for generating the static site.
package site

import (
	"bytes"
	"context"
	"encoding/xml"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/mazrean/go-proposal-review-meeting/internal/content"
	"github.com/mazrean/go-proposal-review-meeting/internal/parser"
)

// RSS is the root element of an RSS 2.0 feed.
type RSS struct {
	XMLName xml.Name   `xml:"rss"`
	Version string     `xml:"version,attr"`
	Channel RSSChannel `xml:"channel"`
}

// RSSChannel represents the channel element in an RSS feed.
type RSSChannel struct {
	Title       string    `xml:"title"`
	Link        string    `xml:"link"`
	Description string    `xml:"description"`
	Items       []RSSItem `xml:"item"`
}

// RSSItem represents an item in an RSS feed.
type RSSItem struct {
	Title       string `xml:"title"`
	Link        string `xml:"link"`
	Description string `xml:"description"`
	PubDate     string `xml:"pubDate"`
	GUID        string `xml:"guid"`
}

func TestFeedGenerator_GenerateFeed_EmptyContent(t *testing.T) {
	fg := NewFeedGenerator(WithSiteURL("https://example.com"))

	data, err := fg.GenerateFeed(context.Background(), nil)
	if err != nil {
		t.Fatalf("GenerateFeed() error = %v", err)
	}

	// Should return valid RSS even with no items
	var rss RSS
	if err := xml.Unmarshal(data, &rss); err != nil {
		t.Fatalf("Failed to parse RSS: %v", err)
	}

	if rss.Version != "2.0" {
		t.Errorf("RSS version = %q, want %q", rss.Version, "2.0")
	}

	if len(rss.Channel.Items) != 0 {
		t.Errorf("Expected 0 items, got %d", len(rss.Channel.Items))
	}
}

func TestFeedGenerator_GenerateFeed_SingleWeek(t *testing.T) {
	fg := NewFeedGenerator(WithSiteURL("https://example.com"))

	weeks := []*content.WeeklyContent{
		{
			Year:      2026,
			Week:      5,
			CreatedAt: time.Date(2026, 1, 30, 12, 0, 0, 0, time.UTC),
			Proposals: []content.ProposalContent{
				{
					IssueNumber:    12345,
					Title:          "proposal: add new feature",
					PreviousStatus: parser.StatusDiscussions,
					CurrentStatus:  parser.StatusAccepted,
					Summary:        "This proposal was accepted because...",
					ChangedAt:      time.Date(2026, 1, 30, 12, 0, 0, 0, time.UTC),
				},
			},
		},
	}

	data, err := fg.GenerateFeed(context.Background(), weeks)
	if err != nil {
		t.Fatalf("GenerateFeed() error = %v", err)
	}

	var rss RSS
	if err := xml.Unmarshal(data, &rss); err != nil {
		t.Fatalf("Failed to parse RSS: %v", err)
	}

	// Check RSS version
	if rss.Version != "2.0" {
		t.Errorf("RSS version = %q, want %q", rss.Version, "2.0")
	}

	// Check channel metadata
	if !strings.Contains(rss.Channel.Title, "Go Proposal") {
		t.Errorf("Channel title = %q, should contain 'Go Proposal'", rss.Channel.Title)
	}

	// Check we have 1 item (one week)
	if len(rss.Channel.Items) != 1 {
		t.Fatalf("Expected 1 item, got %d", len(rss.Channel.Items))
	}

	item := rss.Channel.Items[0]
	// Check item contains week information
	if !strings.Contains(item.Title, "2026") || !strings.Contains(item.Title, "5") {
		t.Errorf("Item title = %q, should contain year 2026 and week 5", item.Title)
	}

	// Check description contains proposal info
	if !strings.Contains(item.Description, "12345") {
		t.Errorf("Item description should contain proposal number 12345")
	}
}

func TestFeedGenerator_GenerateFeed_MultipleWeeks(t *testing.T) {
	fg := NewFeedGenerator(WithSiteURL("https://example.com"))

	weeks := make([]*content.WeeklyContent, 3)
	for i := range 3 {
		weeks[i] = &content.WeeklyContent{
			Year:      2026,
			Week:      i + 1,
			CreatedAt: time.Date(2026, 1, 1+i*7, 12, 0, 0, 0, time.UTC),
			Proposals: []content.ProposalContent{
				{
					IssueNumber:    10000 + i,
					Title:          "proposal: feature " + string(rune('A'+i)),
					PreviousStatus: parser.StatusDiscussions,
					CurrentStatus:  parser.StatusAccepted,
					Summary:        "Summary for week " + string(rune('1'+i)),
					ChangedAt:      time.Date(2026, 1, 1+i*7, 12, 0, 0, 0, time.UTC),
				},
			},
		}
	}

	data, err := fg.GenerateFeed(context.Background(), weeks)
	if err != nil {
		t.Fatalf("GenerateFeed() error = %v", err)
	}

	var rss RSS
	if err := xml.Unmarshal(data, &rss); err != nil {
		t.Fatalf("Failed to parse RSS: %v", err)
	}

	// Should have 3 items (one per week)
	if len(rss.Channel.Items) != 3 {
		t.Errorf("Expected 3 items, got %d", len(rss.Channel.Items))
	}
}

func TestFeedGenerator_GenerateFeed_MaxItems(t *testing.T) {
	fg := NewFeedGenerator(WithSiteURL("https://example.com"))

	// Create 25 weeks (more than max 20)
	weeks := make([]*content.WeeklyContent, 25)
	for i := range 25 {
		weeks[i] = &content.WeeklyContent{
			Year:      2026,
			Week:      i + 1,
			CreatedAt: time.Date(2026, 1, 1+i*7, 12, 0, 0, 0, time.UTC),
			Proposals: []content.ProposalContent{
				{
					IssueNumber:    10000 + i,
					Title:          "proposal: feature",
					PreviousStatus: parser.StatusDiscussions,
					CurrentStatus:  parser.StatusAccepted,
					ChangedAt:      time.Date(2026, 1, 1+i*7, 12, 0, 0, 0, time.UTC),
				},
			},
		}
	}

	data, err := fg.GenerateFeed(context.Background(), weeks)
	if err != nil {
		t.Fatalf("GenerateFeed() error = %v", err)
	}

	var rss RSS
	if err := xml.Unmarshal(data, &rss); err != nil {
		t.Fatalf("Failed to parse RSS: %v", err)
	}

	// Should be limited to 20 items
	if len(rss.Channel.Items) != 20 {
		t.Errorf("Expected 20 items (max limit), got %d", len(rss.Channel.Items))
	}
}

func TestFeedGenerator_GenerateFeed_ContainsProposalDetails(t *testing.T) {
	fg := NewFeedGenerator(WithSiteURL("https://example.com"))

	weeks := []*content.WeeklyContent{
		{
			Year:      2026,
			Week:      5,
			CreatedAt: time.Date(2026, 1, 30, 12, 0, 0, 0, time.UTC),
			Proposals: []content.ProposalContent{
				{
					IssueNumber:    12345,
					Title:          "proposal: add generics",
					PreviousStatus: parser.StatusDiscussions,
					CurrentStatus:  parser.StatusAccepted,
					Summary:        "This proposal was accepted due to strong community support.",
					ChangedAt:      time.Date(2026, 1, 30, 12, 0, 0, 0, time.UTC),
				},
				{
					IssueNumber:    67890,
					Title:          "proposal: improve error handling",
					PreviousStatus: parser.StatusDiscussions,
					CurrentStatus:  parser.StatusDeclined,
					Summary:        "This proposal was declined because...",
					ChangedAt:      time.Date(2026, 1, 30, 14, 0, 0, 0, time.UTC),
				},
			},
		},
	}

	data, err := fg.GenerateFeed(context.Background(), weeks)
	if err != nil {
		t.Fatalf("GenerateFeed() error = %v", err)
	}

	// Convert to string for easier checking
	content := string(data)

	// Check that proposal titles are included
	if !strings.Contains(content, "add generics") {
		t.Error("Feed should contain first proposal title 'add generics'")
	}
	if !strings.Contains(content, "improve error handling") {
		t.Error("Feed should contain second proposal title 'improve error handling'")
	}

	// Check that status changes are included
	if !strings.Contains(content, "accepted") || !strings.Contains(content, "discussions") {
		t.Error("Feed should contain status change information")
	}
}

func TestFeedGenerator_GenerateFeed_ValidXML(t *testing.T) {
	fg := NewFeedGenerator(WithSiteURL("https://example.com"))

	weeks := []*content.WeeklyContent{
		{
			Year:      2026,
			Week:      5,
			CreatedAt: time.Date(2026, 1, 30, 12, 0, 0, 0, time.UTC),
			Proposals: []content.ProposalContent{
				{
					IssueNumber:    12345,
					Title:          "proposal: test <special> & \"chars\"",
					PreviousStatus: parser.StatusDiscussions,
					CurrentStatus:  parser.StatusAccepted,
					Summary:        "Summary with <html> & special chars",
					ChangedAt:      time.Date(2026, 1, 30, 12, 0, 0, 0, time.UTC),
				},
			},
		},
	}

	data, err := fg.GenerateFeed(context.Background(), weeks)
	if err != nil {
		t.Fatalf("GenerateFeed() error = %v", err)
	}

	// Verify it's valid XML by parsing
	var rss RSS
	if err := xml.Unmarshal(data, &rss); err != nil {
		t.Fatalf("Generated RSS should be valid XML: %v", err)
	}

	// Check XML declaration
	if !bytes.HasPrefix(data, []byte("<?xml")) {
		t.Error("RSS should start with XML declaration")
	}
}

func TestFeedGenerator_GenerateFeed_ContextCancellation(t *testing.T) {
	fg := NewFeedGenerator(WithSiteURL("https://example.com"))

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	weeks := []*content.WeeklyContent{
		{
			Year: 2026,
			Week: 5,
			Proposals: []content.ProposalContent{
				{IssueNumber: 12345, Title: "test"},
			},
		},
	}

	_, err := fg.GenerateFeed(ctx, weeks)
	if err == nil {
		t.Error("Expected error when context is cancelled")
	}
}

func TestFeedGenerator_GenerateFeed_CorrectLinks(t *testing.T) {
	siteURL := "https://go-proposal-digest.example.com"
	fg := NewFeedGenerator(WithSiteURL(siteURL))

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
					ChangedAt:      time.Date(2026, 1, 30, 12, 0, 0, 0, time.UTC),
				},
			},
		},
	}

	data, err := fg.GenerateFeed(context.Background(), weeks)
	if err != nil {
		t.Fatalf("GenerateFeed() error = %v", err)
	}

	var rss RSS
	if err := xml.Unmarshal(data, &rss); err != nil {
		t.Fatalf("Failed to parse RSS: %v", err)
	}

	// Check channel link
	if rss.Channel.Link != siteURL {
		t.Errorf("Channel link = %q, want %q", rss.Channel.Link, siteURL)
	}

	// Check item link contains the site URL
	if len(rss.Channel.Items) > 0 {
		item := rss.Channel.Items[0]
		if !strings.HasPrefix(item.Link, siteURL) {
			t.Errorf("Item link = %q, should start with %q", item.Link, siteURL)
		}
	}
}

func TestFeedGenerator_GenerateFeed_MaxItemsKeepsNewest(t *testing.T) {
	fg := NewFeedGenerator(WithSiteURL("https://example.com"))

	// Create 25 weeks in random order (not sorted by date)
	weeks := make([]*content.WeeklyContent, 25)
	for i := range 25 {
		// Shuffle: week 25 at index 0, week 24 at index 1, etc.
		weekNum := 25 - i
		weeks[i] = &content.WeeklyContent{
			Year:      2026,
			Week:      weekNum,
			CreatedAt: time.Date(2026, 1, 1+weekNum*7, 12, 0, 0, 0, time.UTC),
			Proposals: []content.ProposalContent{
				{
					IssueNumber:    10000 + weekNum,
					Title:          "proposal: feature",
					PreviousStatus: parser.StatusDiscussions,
					CurrentStatus:  parser.StatusAccepted,
					ChangedAt:      time.Date(2026, 1, 1+weekNum*7, 12, 0, 0, 0, time.UTC),
				},
			},
		}
	}

	data, err := fg.GenerateFeed(context.Background(), weeks)
	if err != nil {
		t.Fatalf("GenerateFeed() error = %v", err)
	}

	var rss RSS
	if err := xml.Unmarshal(data, &rss); err != nil {
		t.Fatalf("Failed to parse RSS: %v", err)
	}

	// Should be limited to 20 items
	if len(rss.Channel.Items) != 20 {
		t.Fatalf("Expected 20 items (max limit), got %d", len(rss.Channel.Items))
	}

	// First item should be the newest week (week 25)
	if !strings.Contains(rss.Channel.Items[0].Title, "25") {
		t.Errorf("First item should be week 25, got: %s", rss.Channel.Items[0].Title)
	}

	// Last item should be week 6 (25-20+1=6)
	if !strings.Contains(rss.Channel.Items[19].Title, "6") {
		t.Errorf("Last item should be week 6, got: %s", rss.Channel.Items[19].Title)
	}
}

func TestFeedGenerator_GenerateFeed_PubDateFormat(t *testing.T) {
	fg := NewFeedGenerator(WithSiteURL("https://example.com"))

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
					ChangedAt:      time.Date(2026, 1, 30, 12, 0, 0, 0, time.UTC),
				},
			},
		},
	}

	data, err := fg.GenerateFeed(context.Background(), weeks)
	if err != nil {
		t.Fatalf("GenerateFeed() error = %v", err)
	}

	var rss RSS
	if err := xml.Unmarshal(data, &rss); err != nil {
		t.Fatalf("Failed to parse RSS: %v", err)
	}

	// Check pubDate is present and valid format (RFC 1123)
	if len(rss.Channel.Items) == 0 {
		t.Fatal("Expected at least 1 item")
	}

	pubDate := rss.Channel.Items[0].PubDate
	if pubDate == "" {
		t.Error("Item pubDate should not be empty")
	}

	// Parse the pubDate to verify format
	_, err = time.Parse(time.RFC1123Z, pubDate)
	if err != nil {
		// Try RFC1123 as well
		_, err = time.Parse(time.RFC1123, pubDate)
		if err != nil {
			t.Errorf("Item pubDate %q is not valid RFC1123 format: %v", pubDate, err)
		}
	}
}

func TestFeedGenerator_GenerateFeed_MultibyteCharTruncation(t *testing.T) {
	fg := NewFeedGenerator(WithSiteURL("https://example.com"))

	// Create a long Japanese summary that would need truncation
	// This should not break in the middle of a multibyte character
	longSummary := strings.Repeat("日本語テスト", 50) // 300 characters (50 * 6 chars)

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
					Summary:        longSummary,
					ChangedAt:      time.Date(2026, 1, 30, 12, 0, 0, 0, time.UTC),
				},
			},
		},
	}

	data, err := fg.GenerateFeed(context.Background(), weeks)
	if err != nil {
		t.Fatalf("GenerateFeed() error = %v", err)
	}

	// Verify it's still valid UTF-8 and valid XML
	if !isValidUTF8(data) {
		t.Error("Generated feed contains invalid UTF-8")
	}

	var rss RSS
	if err := xml.Unmarshal(data, &rss); err != nil {
		t.Fatalf("Generated feed is not valid XML: %v", err)
	}
}

func isValidUTF8(data []byte) bool {
	for len(data) > 0 {
		r, size := utf8.DecodeRune(data)
		if r == utf8.RuneError && size == 1 {
			return false
		}
		data = data[size:]
	}
	return true
}

func TestFeedGenerator_GenerateFeed_EmptyAuthorEmail(t *testing.T) {
	// When author email is empty, the feed should still be valid
	fg := NewFeedGenerator(
		WithSiteURL("https://example.com"),
		WithAuthor("Go Proposal Digest", ""), // Empty email
	)

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
					ChangedAt:      time.Date(2026, 1, 30, 12, 0, 0, 0, time.UTC),
				},
			},
		},
	}

	data, err := fg.GenerateFeed(context.Background(), weeks)
	if err != nil {
		t.Fatalf("GenerateFeed() error = %v", err)
	}

	// Should be valid XML
	var rss RSS
	if err := xml.Unmarshal(data, &rss); err != nil {
		t.Fatalf("Generated feed is not valid XML: %v", err)
	}

	// The managingEditor field should either be absent or have valid format
	// (not just " (Name)" without email)
	content := string(data)
	if strings.Contains(content, "<managingEditor> (") {
		t.Error("managingEditor should not contain empty email format ' (Name)'")
	}
}

func TestFeedGenerator_GenerateFeed_HTMLEscaping(t *testing.T) {
	fg := NewFeedGenerator(WithSiteURL("https://example.com"))

	weeks := []*content.WeeklyContent{
		{
			Year:      2026,
			Week:      5,
			CreatedAt: time.Date(2026, 1, 30, 12, 0, 0, 0, time.UTC),
			Proposals: []content.ProposalContent{
				{
					IssueNumber:    12345,
					Title:          "proposal: test's \"special\" <chars> & more",
					PreviousStatus: parser.StatusDiscussions,
					CurrentStatus:  parser.StatusAccepted,
					Summary:        "Summary with 'single quotes' and <script>alert('xss')</script>",
					ChangedAt:      time.Date(2026, 1, 30, 12, 0, 0, 0, time.UTC),
				},
			},
		},
	}

	data, err := fg.GenerateFeed(context.Background(), weeks)
	if err != nil {
		t.Fatalf("GenerateFeed() error = %v", err)
	}

	// Should be valid XML (escaping works)
	var rss RSS
	if err := xml.Unmarshal(data, &rss); err != nil {
		t.Fatalf("Generated feed is not valid XML: %v", err)
	}

	// Raw <script> tags should not appear (should be escaped)
	content := string(data)
	if strings.Contains(content, "<script>") {
		t.Error("HTML content should be escaped, but found raw <script> tag")
	}
}

func TestFeedGenerator_GenerateFeed_WeekWithNoProposals(t *testing.T) {
	fg := NewFeedGenerator(WithSiteURL("https://example.com"))

	weeks := []*content.WeeklyContent{
		{
			Year:      2026,
			Week:      5,
			CreatedAt: time.Date(2026, 1, 30, 12, 0, 0, 0, time.UTC),
			Proposals: []content.ProposalContent{}, // No proposals
		},
	}

	data, err := fg.GenerateFeed(context.Background(), weeks)
	if err != nil {
		t.Fatalf("GenerateFeed() error = %v", err)
	}

	var rss RSS
	if err := xml.Unmarshal(data, &rss); err != nil {
		t.Fatalf("Generated feed is not valid XML: %v", err)
	}

	// Should still have 1 item for the week
	if len(rss.Channel.Items) != 1 {
		t.Errorf("Expected 1 item for week with no proposals, got %d", len(rss.Channel.Items))
	}

	// Description should indicate no updates
	if !strings.Contains(rss.Channel.Items[0].Description, "ありません") {
		t.Error("Week with no proposals should indicate no updates in description")
	}
}

// TestFeedGenerator_GenerateFeed_ContainsSummary verifies that proposal summaries are included in the feed.
// This is a specific test for Requirement 5.3: フィードにproposalのタイトル、ステータス変更、要約を含める
func TestFeedGenerator_GenerateFeed_ContainsSummary(t *testing.T) {
	fg := NewFeedGenerator(WithSiteURL("https://example.com"))

	expectedSummary := "このproposalはジェネリクスの実装方針について議論された結果、承認されました。"

	weeks := []*content.WeeklyContent{
		{
			Year:      2026,
			Week:      5,
			CreatedAt: time.Date(2026, 1, 30, 12, 0, 0, 0, time.UTC),
			Proposals: []content.ProposalContent{
				{
					IssueNumber:    12345,
					Title:          "proposal: add generics",
					PreviousStatus: parser.StatusDiscussions,
					CurrentStatus:  parser.StatusAccepted,
					Summary:        expectedSummary,
					ChangedAt:      time.Date(2026, 1, 30, 12, 0, 0, 0, time.UTC),
				},
			},
		},
	}

	data, err := fg.GenerateFeed(context.Background(), weeks)
	if err != nil {
		t.Fatalf("GenerateFeed() error = %v", err)
	}

	var rss RSS
	if err := xml.Unmarshal(data, &rss); err != nil {
		t.Fatalf("Failed to parse RSS: %v", err)
	}

	if len(rss.Channel.Items) == 0 {
		t.Fatal("Expected at least 1 item")
	}

	// Requirement 5.3: Summary must be included in the feed item description
	if !strings.Contains(rss.Channel.Items[0].Description, "ジェネリクス") {
		t.Error("Feed item description should contain the proposal summary (Requirement 5.3)")
	}
}

// TestFeedGenerator_GenerateFeed_RSS20RequiredChannelElements verifies RSS 2.0 channel requirements.
// RSS 2.0 spec requires: title, link, description for channel element.
func TestFeedGenerator_GenerateFeed_RSS20RequiredChannelElements(t *testing.T) {
	fg := NewFeedGenerator(
		WithSiteURL("https://go-proposal-digest.example.com"),
		WithSiteTitle("Go Proposal Weekly Digest"),
		WithSiteDescription("週次Go proposal更新情報"),
	)

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
					ChangedAt:      time.Date(2026, 1, 30, 12, 0, 0, 0, time.UTC),
				},
			},
		},
	}

	data, err := fg.GenerateFeed(context.Background(), weeks)
	if err != nil {
		t.Fatalf("GenerateFeed() error = %v", err)
	}

	var rss RSS
	if err := xml.Unmarshal(data, &rss); err != nil {
		t.Fatalf("Failed to parse RSS: %v", err)
	}

	// RSS 2.0 required channel elements
	if rss.Channel.Title == "" {
		t.Error("RSS 2.0 requires channel title")
	}
	if rss.Channel.Link == "" {
		t.Error("RSS 2.0 requires channel link")
	}
	if rss.Channel.Description == "" {
		t.Error("RSS 2.0 requires channel description")
	}
}

// TestFeedGenerator_GenerateFeed_RSS20RequiredItemElements verifies RSS 2.0 item requirements.
// Each item should have at least title or description, plus recommended GUID.
func TestFeedGenerator_GenerateFeed_RSS20RequiredItemElements(t *testing.T) {
	fg := NewFeedGenerator(WithSiteURL("https://example.com"))

	weeks := []*content.WeeklyContent{
		{
			Year:      2026,
			Week:      5,
			CreatedAt: time.Date(2026, 1, 30, 12, 0, 0, 0, time.UTC),
			Proposals: []content.ProposalContent{
				{
					IssueNumber:    12345,
					Title:          "proposal: test item elements",
					PreviousStatus: parser.StatusDiscussions,
					CurrentStatus:  parser.StatusAccepted,
					ChangedAt:      time.Date(2026, 1, 30, 12, 0, 0, 0, time.UTC),
				},
			},
		},
	}

	data, err := fg.GenerateFeed(context.Background(), weeks)
	if err != nil {
		t.Fatalf("GenerateFeed() error = %v", err)
	}

	var rss RSS
	if err := xml.Unmarshal(data, &rss); err != nil {
		t.Fatalf("Failed to parse RSS: %v", err)
	}

	if len(rss.Channel.Items) == 0 {
		t.Fatal("Expected at least 1 item")
	}

	item := rss.Channel.Items[0]

	// Item must have title
	if item.Title == "" {
		t.Error("RSS 2.0 item should have title")
	}

	// Item must have description
	if item.Description == "" {
		t.Error("RSS 2.0 item should have description")
	}

	// Item should have GUID for uniqueness
	if item.GUID == "" {
		t.Error("RSS 2.0 item should have GUID for unique identification")
	}

	// Item should have link
	if item.Link == "" {
		t.Error("RSS 2.0 item should have link")
	}

	// Item should have pubDate
	if item.PubDate == "" {
		t.Error("RSS 2.0 item should have pubDate")
	}
}

// TestFeedGenerator_GenerateFeed_GUIDUniqueness verifies that each item has a unique GUID.
func TestFeedGenerator_GenerateFeed_GUIDUniqueness(t *testing.T) {
	fg := NewFeedGenerator(WithSiteURL("https://example.com"))

	weeks := make([]*content.WeeklyContent, 5)
	for i := range 5 {
		weeks[i] = &content.WeeklyContent{
			Year:      2026,
			Week:      i + 1,
			CreatedAt: time.Date(2026, 1, 1+i*7, 12, 0, 0, 0, time.UTC),
			Proposals: []content.ProposalContent{
				{
					IssueNumber:    10000 + i,
					Title:          "proposal: test",
					PreviousStatus: parser.StatusDiscussions,
					CurrentStatus:  parser.StatusAccepted,
					ChangedAt:      time.Date(2026, 1, 1+i*7, 12, 0, 0, 0, time.UTC),
				},
			},
		}
	}

	data, err := fg.GenerateFeed(context.Background(), weeks)
	if err != nil {
		t.Fatalf("GenerateFeed() error = %v", err)
	}

	var rss RSS
	if err := xml.Unmarshal(data, &rss); err != nil {
		t.Fatalf("Failed to parse RSS: %v", err)
	}

	// Collect all GUIDs and check for uniqueness
	guids := make(map[string]bool)
	for _, item := range rss.Channel.Items {
		if guids[item.GUID] {
			t.Errorf("Duplicate GUID found: %s", item.GUID)
		}
		guids[item.GUID] = true
	}
}

// TestFeedGenerator_GenerateFeed_CrossYearWeeks tests handling of weeks across year boundaries.
func TestFeedGenerator_GenerateFeed_CrossYearWeeks(t *testing.T) {
	fg := NewFeedGenerator(WithSiteURL("https://example.com"))

	weeks := []*content.WeeklyContent{
		{
			Year:      2027,
			Week:      1, // First week of 2027
			CreatedAt: time.Date(2027, 1, 5, 12, 0, 0, 0, time.UTC),
			Proposals: []content.ProposalContent{
				{
					IssueNumber:    20001,
					Title:          "proposal: 2027 feature",
					PreviousStatus: parser.StatusDiscussions,
					CurrentStatus:  parser.StatusAccepted,
					ChangedAt:      time.Date(2027, 1, 5, 12, 0, 0, 0, time.UTC),
				},
			},
		},
		{
			Year:      2026,
			Week:      52, // Last week of 2026
			CreatedAt: time.Date(2026, 12, 28, 12, 0, 0, 0, time.UTC),
			Proposals: []content.ProposalContent{
				{
					IssueNumber:    10052,
					Title:          "proposal: 2026 feature",
					PreviousStatus: parser.StatusActive,
					CurrentStatus:  parser.StatusDeclined,
					ChangedAt:      time.Date(2026, 12, 28, 12, 0, 0, 0, time.UTC),
				},
			},
		},
	}

	data, err := fg.GenerateFeed(context.Background(), weeks)
	if err != nil {
		t.Fatalf("GenerateFeed() error = %v", err)
	}

	var rss RSS
	if err := xml.Unmarshal(data, &rss); err != nil {
		t.Fatalf("Failed to parse RSS: %v", err)
	}

	// Should have 2 items
	if len(rss.Channel.Items) != 2 {
		t.Fatalf("Expected 2 items, got %d", len(rss.Channel.Items))
	}

	// First item should be from 2027 (newest first)
	if !strings.Contains(rss.Channel.Items[0].Title, "2027") {
		t.Errorf("First item should be from 2027, got: %s", rss.Channel.Items[0].Title)
	}

	// Second item should be from 2026
	if !strings.Contains(rss.Channel.Items[1].Title, "2026") {
		t.Errorf("Second item should be from 2026, got: %s", rss.Channel.Items[1].Title)
	}
}
