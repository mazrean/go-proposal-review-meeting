package site

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/mazrean/go-proposal-review-meeting/internal/content"
	"github.com/mazrean/go-proposal-review-meeting/internal/parser"
)

func TestGenerator_Generate(t *testing.T) {
	t.Parallel()

	// Create a temporary directory for test output
	distDir := t.TempDir()

	// Create test weekly content
	weeklyContent := &content.WeeklyContent{
		Year: 2026,
		Week: 5,
		Proposals: []content.ProposalContent{
			{
				IssueNumber:    12345,
				Title:          "proposal: add new feature",
				PreviousStatus: parser.StatusDiscussions,
				CurrentStatus:  parser.StatusAccepted,
				ChangedAt:      time.Date(2026, 1, 30, 12, 0, 0, 0, time.UTC),
				CommentURL:     "https://github.com/golang/go/issues/33502#issuecomment-xxx",
				Summary:        "This proposal adds a new feature to Go.",
				Links: []content.Link{
					{Title: "proposal issue", URL: "https://github.com/golang/go/issues/12345"},
				},
			},
			{
				IssueNumber:    67890,
				Title:          "proposal: improve performance",
				PreviousStatus: parser.StatusActive,
				CurrentStatus:  parser.StatusLikelyAccept,
				ChangedAt:      time.Date(2026, 1, 29, 10, 0, 0, 0, time.UTC),
				CommentURL:     "https://github.com/golang/go/issues/33502#issuecomment-yyy",
				Summary:        "This proposal improves performance.",
				Links: []content.Link{
					{Title: "proposal issue", URL: "https://github.com/golang/go/issues/67890"},
				},
			},
		},
		CreatedAt: time.Now(),
	}

	// Create generator
	gen := NewGenerator(
		WithDistDir(distDir),
	)

	// Generate the site
	ctx := context.Background()
	err := gen.Generate(ctx, []*content.WeeklyContent{weeklyContent})
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	// Verify that index.html was created
	indexPath := filepath.Join(distDir, "index.html")
	if _, err := os.Stat(indexPath); os.IsNotExist(err) {
		t.Errorf("index.html was not created at %s", indexPath)
	}

	// Verify that weekly index was created
	weeklyIndexPath := filepath.Join(distDir, "2026", "w05", "index.html")
	if _, err := os.Stat(weeklyIndexPath); os.IsNotExist(err) {
		t.Errorf("weekly index.html was not created at %s", weeklyIndexPath)
	}

	// Verify that proposal pages were created
	proposal1Path := filepath.Join(distDir, "2026", "w05", "12345.html")
	if _, err := os.Stat(proposal1Path); os.IsNotExist(err) {
		t.Errorf("proposal page was not created at %s", proposal1Path)
	}

	proposal2Path := filepath.Join(distDir, "2026", "w05", "67890.html")
	if _, err := os.Stat(proposal2Path); os.IsNotExist(err) {
		t.Errorf("proposal page was not created at %s", proposal2Path)
	}

	// Verify index.html contains expected content
	indexContent, err := os.ReadFile(indexPath)
	if err != nil {
		t.Fatalf("Failed to read index.html: %v", err)
	}
	if !strings.Contains(string(indexContent), "2026年 第5週") {
		t.Errorf("index.html does not contain expected week reference")
	}

	// Verify weekly index contains proposal links
	weeklyContent2, err := os.ReadFile(weeklyIndexPath)
	if err != nil {
		t.Fatalf("Failed to read weekly index.html: %v", err)
	}
	if !strings.Contains(string(weeklyContent2), "#12345") {
		t.Errorf("weekly index.html does not contain proposal #12345")
	}
	if !strings.Contains(string(weeklyContent2), "#67890") {
		t.Errorf("weekly index.html does not contain proposal #67890")
	}

	// Verify proposal page contains expected content
	proposalContent, err := os.ReadFile(proposal1Path)
	if err != nil {
		t.Fatalf("Failed to read proposal page: %v", err)
	}
	if !strings.Contains(string(proposalContent), "proposal: add new feature") {
		t.Errorf("proposal page does not contain expected title")
	}
	if !strings.Contains(string(proposalContent), "This proposal adds a new feature to Go.") {
		t.Errorf("proposal page does not contain expected summary")
	}
}

func TestGenerator_GenerateEmpty(t *testing.T) {
	t.Parallel()

	distDir := t.TempDir()

	gen := NewGenerator(
		WithDistDir(distDir),
	)

	ctx := context.Background()
	err := gen.Generate(ctx, nil)
	if err != nil {
		t.Fatalf("Generate() with nil content should not error = %v", err)
	}

	// Should still create index.html even with empty content
	indexPath := filepath.Join(distDir, "index.html")
	if _, err := os.Stat(indexPath); os.IsNotExist(err) {
		t.Errorf("index.html was not created at %s", indexPath)
	}
}

func TestGenerator_GenerateMultipleWeeks(t *testing.T) {
	t.Parallel()

	distDir := t.TempDir()

	// Create multiple weekly contents
	weeks := []*content.WeeklyContent{
		{
			Year: 2026,
			Week: 5,
			Proposals: []content.ProposalContent{
				{
					IssueNumber:    12345,
					Title:          "proposal: week 5",
					PreviousStatus: parser.StatusDiscussions,
					CurrentStatus:  parser.StatusAccepted,
					ChangedAt:      time.Date(2026, 1, 30, 12, 0, 0, 0, time.UTC),
				},
			},
		},
		{
			Year: 2026,
			Week: 4,
			Proposals: []content.ProposalContent{
				{
					IssueNumber:    11111,
					Title:          "proposal: week 4",
					PreviousStatus: parser.StatusActive,
					CurrentStatus:  parser.StatusDeclined,
					ChangedAt:      time.Date(2026, 1, 23, 12, 0, 0, 0, time.UTC),
				},
			},
		},
	}

	gen := NewGenerator(
		WithDistDir(distDir),
	)

	ctx := context.Background()
	err := gen.Generate(ctx, weeks)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	// Verify both weeks were generated
	week5Path := filepath.Join(distDir, "2026", "w05", "index.html")
	if _, err := os.Stat(week5Path); os.IsNotExist(err) {
		t.Errorf("week 5 index.html was not created")
	}

	week4Path := filepath.Join(distDir, "2026", "w04", "index.html")
	if _, err := os.Stat(week4Path); os.IsNotExist(err) {
		t.Errorf("week 4 index.html was not created")
	}

	// Verify home page lists both weeks
	indexContent, err := os.ReadFile(filepath.Join(distDir, "index.html"))
	if err != nil {
		t.Fatalf("Failed to read index.html: %v", err)
	}
	if !strings.Contains(string(indexContent), "第5週") {
		t.Errorf("index.html does not list week 5")
	}
	if !strings.Contains(string(indexContent), "第4週") {
		t.Errorf("index.html does not list week 4")
	}
}

func TestGenerator_GenerateContextCancellation(t *testing.T) {
	t.Parallel()

	distDir := t.TempDir()

	weeklyContent := &content.WeeklyContent{
		Year: 2026,
		Week: 5,
		Proposals: []content.ProposalContent{
			{
				IssueNumber:    12345,
				Title:          "test proposal",
				CurrentStatus:  parser.StatusAccepted,
				PreviousStatus: parser.StatusDiscussions,
			},
		},
	}

	gen := NewGenerator(
		WithDistDir(distDir),
	)

	// Create a cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := gen.Generate(ctx, []*content.WeeklyContent{weeklyContent})
	if err == nil {
		t.Error("Generate() with cancelled context should return error")
	}
}

func TestGenerator_GenerateWithRSS(t *testing.T) {
	t.Parallel()

	distDir := t.TempDir()

	weeklyContent := &content.WeeklyContent{
		Year: 2026,
		Week: 5,
		Proposals: []content.ProposalContent{
			{
				IssueNumber:    12345,
				Title:          "proposal: add new feature",
				PreviousStatus: parser.StatusDiscussions,
				CurrentStatus:  parser.StatusAccepted,
				ChangedAt:      time.Date(2026, 1, 30, 12, 0, 0, 0, time.UTC),
				Summary:        "This proposal adds a new feature.",
			},
		},
		CreatedAt: time.Now(),
	}

	// Create generator with site URL
	gen := NewGenerator(
		WithDistDir(distDir),
		WithGeneratorSiteURL("https://example.com"),
	)

	ctx := context.Background()
	err := gen.Generate(ctx, []*content.WeeklyContent{weeklyContent})
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	// Verify feed.xml was created
	feedPath := filepath.Join(distDir, "feed.xml")
	if _, err := os.Stat(feedPath); os.IsNotExist(err) {
		t.Errorf("feed.xml was not created at %s", feedPath)
	}

	// Verify feed.xml contains RSS content
	feedContent, err := os.ReadFile(feedPath)
	if err != nil {
		t.Fatalf("Failed to read feed.xml: %v", err)
	}

	feedStr := string(feedContent)
	if !strings.Contains(feedStr, "<?xml") {
		t.Error("feed.xml should contain XML declaration")
	}
	if !strings.Contains(feedStr, "<rss") {
		t.Error("feed.xml should contain RSS element")
	}
	if !strings.Contains(feedStr, "Go Proposal") {
		t.Error("feed.xml should contain site title")
	}
	if !strings.Contains(feedStr, "12345") {
		t.Error("feed.xml should contain proposal number")
	}
}

func TestGenerator_GenerateRSSWithMaxItems(t *testing.T) {
	t.Parallel()

	distDir := t.TempDir()

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
					Title:          "proposal: test",
					PreviousStatus: parser.StatusDiscussions,
					CurrentStatus:  parser.StatusAccepted,
					ChangedAt:      time.Date(2026, 1, 1+i*7, 12, 0, 0, 0, time.UTC),
				},
			},
		}
	}

	gen := NewGenerator(
		WithDistDir(distDir),
		WithGeneratorSiteURL("https://example.com"),
	)

	ctx := context.Background()
	err := gen.Generate(ctx, weeks)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	// Verify feed.xml was created
	feedPath := filepath.Join(distDir, "feed.xml")
	feedContent, err := os.ReadFile(feedPath)
	if err != nil {
		t.Fatalf("Failed to read feed.xml: %v", err)
	}

	// Count the number of <item> tags
	itemCount := strings.Count(string(feedContent), "<item>")
	if itemCount > 20 {
		t.Errorf("feed.xml should contain at most 20 items, got %d", itemCount)
	}
}
