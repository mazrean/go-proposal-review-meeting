// Package internal_test provides E2E pipeline tests for the Go Proposal Weekly Digest system.
// This file contains E2E pipeline execution tests (Task 8.8) that use the fixtures from Task 8.7.
package internal_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mazrean/go-proposal-review-meeting/internal/parser"
	"github.com/mazrean/go-proposal-review-meeting/internal/testfixtures"
)

// =============================================================================
// Task 8.8: E2Eパイプライン実行
// =============================================================================
//
// This test uses the fixtures from Task 8.7 to execute the full pipeline:
// Parse (mock GitHub API) → Content → Site
// And verifies that all output files are generated correctly.
//
// Requirements covered: 1.2, 1.3, 1.4, 2.2, 2.3
// =============================================================================

// TestE2E_PipelineExecution runs the complete Parse → Content → Site pipeline
// using the 10 proposal fixtures from Task 8.7 and verifies output file generation.
// Requirements: 1.2 (新規コメント識別), 1.3 (ステータス抽出), 1.4 (変更記録),
//
//	2.2 (proposal別MDファイル), 2.3 (MDメタデータ)
func TestE2E_PipelineExecution(t *testing.T) {
	t.Parallel()

	// -------------------------------------------------------------------------
	// Setup: Use fixtures from Task 8.7
	// -------------------------------------------------------------------------
	changes := testfixtures.TenProposalChanges()
	parserSetup := testfixtures.SetupParserTest(t, changes)
	defer parserSetup.Cleanup()

	contentSetup := testfixtures.SetupContentTest(t, changes, true) // with summaries
	defer contentSetup.Cleanup()

	siteSetup := testfixtures.SetupSiteTest(t)
	defer siteSetup.Cleanup()

	// -------------------------------------------------------------------------
	// Step 1: Parse - Fetch changes from mock GitHub API (Requirements 1.2, 1.3)
	// -------------------------------------------------------------------------
	ctx := context.Background()
	fetchedChanges, err := parserSetup.IssueParser.FetchChanges(ctx)
	if err != nil {
		t.Fatalf("FetchChanges() error = %v", err)
	}

	// Requirement 1.2: Should identify new comments
	t.Run("Req 1.2: identifies new comments correctly", func(t *testing.T) {
		if len(fetchedChanges) == 0 {
			t.Error("FetchChanges should return changes from new comments")
		}
	})

	// Requirement 1.3: Should extract statuses
	t.Run("Req 1.3: extracts proposal statuses correctly", func(t *testing.T) {
		for i, change := range fetchedChanges {
			if change.CurrentStatus == "" {
				t.Errorf("change[%d]: CurrentStatus should not be empty", i)
			}
		}
	})

	// Set default PreviousStatus for integration (MinutesParser doesn't track history)
	for i := range fetchedChanges {
		if fetchedChanges[i].PreviousStatus == "" {
			fetchedChanges[i].PreviousStatus = parser.StatusDiscussions
		}
	}

	// -------------------------------------------------------------------------
	// Step 2: Content - Create weekly content and integrate summaries
	// -------------------------------------------------------------------------
	weeklyContent := contentSetup.Manager.PrepareContent(fetchedChanges)

	// Read and integrate summaries
	summaries, err := contentSetup.Manager.ReadSummaries()
	if err != nil {
		t.Fatalf("ReadSummaries() error = %v", err)
	}

	if err := contentSetup.Manager.IntegrateSummaries(weeklyContent, summaries); err != nil {
		t.Fatalf("IntegrateSummaries() error = %v", err)
	}

	// Requirement 2.2: Write content to Markdown files
	if err := contentSetup.Manager.WriteContent(weeklyContent); err != nil {
		t.Fatalf("WriteContent() error = %v", err)
	}

	// Requirement 1.4: Verify changes are recorded
	t.Run("Req 1.4: changes are recorded in content", func(t *testing.T) {
		if len(weeklyContent.Proposals) != len(fetchedChanges) {
			t.Errorf("expected %d proposals in content, got %d", len(fetchedChanges), len(weeklyContent.Proposals))
		}
	})

	// Read content back for verification
	readContent, err := contentSetup.Manager.ListAllWeeks()
	if err != nil {
		t.Fatalf("ListAllWeeks() error = %v", err)
	}

	// Requirement 2.2: Verify MD files are generated for each proposal
	t.Run("Req 2.2: generates MD file for each proposal", func(t *testing.T) {
		year, week := testfixtures.DefaultBaseTime.ISOWeek()
		weekDir := filepath.Join(contentSetup.ContentDir,
			formatYear(year),
			formatWeek(week))

		entries, err := os.ReadDir(weekDir)
		if os.IsNotExist(err) {
			t.Fatalf("week directory should exist: %s", weekDir)
		}
		if err != nil {
			t.Fatalf("failed to read week directory: %v", err)
		}

		mdCount := 0
		for _, entry := range entries {
			if strings.HasSuffix(entry.Name(), ".md") {
				mdCount++
			}
		}

		// Should have at least fetchedChanges count (may have index.md too)
		if mdCount < len(fetchedChanges) {
			t.Errorf("expected at least %d MD files, got %d", len(fetchedChanges), mdCount)
		}
	})

	// Requirement 2.3: Verify MD metadata
	t.Run("Req 2.3: MD files contain required metadata", func(t *testing.T) {
		if len(readContent) == 0 {
			t.Fatal("should have at least one week of content")
		}

		for _, week := range readContent {
			for _, proposal := range week.Proposals {
				if proposal.IssueNumber == 0 {
					t.Error("IssueNumber should be set")
				}
				if proposal.Title == "" {
					t.Error("Title should be set")
				}
				if proposal.CurrentStatus == "" {
					t.Error("CurrentStatus should be set")
				}
				if proposal.PreviousStatus == "" {
					t.Error("PreviousStatus should be set")
				}
				if proposal.ChangedAt.IsZero() {
					t.Error("ChangedAt should be set")
				}
			}
		}
	})

	// -------------------------------------------------------------------------
	// Step 3: Site - Generate static site from content
	// -------------------------------------------------------------------------
	if err := siteSetup.Generator.Generate(ctx, readContent); err != nil {
		t.Fatalf("Generator.Generate() error = %v", err)
	}

	// -------------------------------------------------------------------------
	// Step 4: Verify output files are generated
	// -------------------------------------------------------------------------
	t.Run("generates home page", func(t *testing.T) {
		indexPath := filepath.Join(siteSetup.DistDir, "index.html")
		if _, err := os.Stat(indexPath); os.IsNotExist(err) {
			t.Error("index.html should be generated")
		}
	})

	t.Run("generates weekly index page", func(t *testing.T) {
		year, week := testfixtures.DefaultBaseTime.ISOWeek()
		weeklyIndexPath := filepath.Join(siteSetup.DistDir,
			formatYear(year),
			formatWeekLower(week),
			"index.html")

		if _, err := os.Stat(weeklyIndexPath); os.IsNotExist(err) {
			t.Errorf("weekly index should be generated: %s", weeklyIndexPath)
		}
	})

	t.Run("generates proposal HTML pages for all 10 proposals", func(t *testing.T) {
		year, week := testfixtures.DefaultBaseTime.ISOWeek()
		baseDir := filepath.Join(siteSetup.DistDir,
			formatYear(year),
			formatWeekLower(week))

		for _, change := range changes {
			proposalPath := filepath.Join(baseDir, formatProposalPage(change.IssueNumber))
			if _, err := os.Stat(proposalPath); os.IsNotExist(err) {
				t.Errorf("proposal page should be generated for issue %d: %s",
					change.IssueNumber, proposalPath)
			}
		}
	})

	t.Run("generates RSS feed", func(t *testing.T) {
		feedPath := filepath.Join(siteSetup.DistDir, "feed.xml")
		if _, err := os.Stat(feedPath); os.IsNotExist(err) {
			t.Error("feed.xml should be generated")
		}
	})

	t.Run("total HTML file count is correct", func(t *testing.T) {
		var htmlCount int
		err := filepath.Walk(siteSetup.DistDir, func(path string, info os.FileInfo, err error) error {
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

		// Expected: 1 home + 1 weekly index + 10 proposal pages = 12
		// Note: The actual count depends on how many proposals were parsed
		expectedCount := 1 + 1 + len(changes)
		if htmlCount < expectedCount {
			t.Errorf("expected at least %d HTML files, got %d", expectedCount, htmlCount)
		}
	})
}

// TestE2E_PipelineExecution_WithFallback runs the pipeline without AI summaries
// and verifies fallback text is applied.
// Requirements: 1.2, 1.3, 1.4, 2.2, 2.3
func TestE2E_PipelineExecution_WithFallback(t *testing.T) {
	t.Parallel()

	// Setup with NO summaries (simulating AI generation failure)
	changes := testfixtures.TenProposalChanges()
	parserSetup := testfixtures.SetupParserTest(t, changes)
	defer parserSetup.Cleanup()

	contentSetup := testfixtures.SetupContentTest(t, changes, false) // without summaries
	defer contentSetup.Cleanup()

	siteSetup := testfixtures.SetupSiteTest(t)
	defer siteSetup.Cleanup()

	// Execute pipeline
	ctx := context.Background()
	fetchedChanges, err := parserSetup.IssueParser.FetchChanges(ctx)
	if err != nil {
		t.Fatalf("FetchChanges() error = %v", err)
	}

	// Set default PreviousStatus
	for i := range fetchedChanges {
		if fetchedChanges[i].PreviousStatus == "" {
			fetchedChanges[i].PreviousStatus = parser.StatusDiscussions
		}
	}

	weeklyContent := contentSetup.Manager.PrepareContent(fetchedChanges)

	// Read summaries (should be empty)
	summaries, err := contentSetup.Manager.ReadSummaries()
	if err != nil {
		t.Fatalf("ReadSummaries() error = %v", err)
	}

	// Integrate (no-op) and apply fallback
	_ = contentSetup.Manager.IntegrateSummaries(weeklyContent, summaries)
	_ = contentSetup.Manager.ApplyFallback(weeklyContent)

	// Write and generate
	if err := contentSetup.Manager.WriteContent(weeklyContent); err != nil {
		t.Fatalf("WriteContent() error = %v", err)
	}

	readContent, err := contentSetup.Manager.ListAllWeeks()
	if err != nil {
		t.Fatalf("ListAllWeeks() error = %v", err)
	}

	if err := siteSetup.Generator.Generate(ctx, readContent); err != nil {
		t.Fatalf("Generator.Generate() error = %v", err)
	}

	// Verify fallback was applied
	t.Run("proposals have fallback summaries", func(t *testing.T) {
		for _, week := range readContent {
			for _, proposal := range week.Proposals {
				if proposal.Summary == "" {
					t.Errorf("proposal %d should have fallback summary", proposal.IssueNumber)
				}
			}
		}
	})

	// Verify output files are still generated
	t.Run("generates all HTML files even with fallback", func(t *testing.T) {
		indexPath := filepath.Join(siteSetup.DistDir, "index.html")
		if _, err := os.Stat(indexPath); os.IsNotExist(err) {
			t.Error("index.html should be generated")
		}

		feedPath := filepath.Join(siteSetup.DistDir, "feed.xml")
		if _, err := os.Stat(feedPath); os.IsNotExist(err) {
			t.Error("feed.xml should be generated")
		}
	})
}

// TestE2E_PipelineExecution_WritesChangesJSON verifies that changes.json is written
// for passing data between workflow steps.
// Requirement: 1.4
func TestE2E_PipelineExecution_WritesChangesJSON(t *testing.T) {
	t.Parallel()

	changes := testfixtures.TenProposalChanges()
	parserSetup := testfixtures.SetupParserTest(t, changes)
	defer parserSetup.Cleanup()

	ctx := context.Background()
	fetchedChanges, err := parserSetup.IssueParser.FetchChanges(ctx)
	if err != nil {
		t.Fatalf("FetchChanges() error = %v", err)
	}

	// Write changes.json
	tmpDir := t.TempDir()
	changesPath := filepath.Join(tmpDir, "changes.json")
	if err := parserSetup.IssueParser.WriteChangesJSON(fetchedChanges, changesPath); err != nil {
		t.Fatalf("WriteChangesJSON() error = %v", err)
	}

	// Verify file exists and has content
	info, err := os.Stat(changesPath)
	if os.IsNotExist(err) {
		t.Error("changes.json should be created")
	}
	if info.Size() == 0 {
		t.Error("changes.json should not be empty")
	}
}

// TestE2E_PipelineExecution_StateManagement verifies that state is updated after processing.
// Requirement: 1.2
func TestE2E_PipelineExecution_StateManagement(t *testing.T) {
	t.Parallel()

	changes := testfixtures.TenProposalChanges()
	parserSetup := testfixtures.SetupParserTest(t, changes)
	defer parserSetup.Cleanup()

	// First fetch
	ctx := context.Background()
	fetchedChanges, err := parserSetup.IssueParser.FetchChanges(ctx)
	if err != nil {
		t.Fatalf("FetchChanges() error = %v", err)
	}

	if len(fetchedChanges) == 0 {
		t.Fatal("should get changes on first fetch")
	}

	// Verify state file was updated
	info, err := os.Stat(parserSetup.StatePath)
	if os.IsNotExist(err) {
		t.Error("state.json should be created after fetch")
	}
	if info.Size() == 0 {
		t.Error("state.json should not be empty")
	}
}

// Helper functions for formatting paths

func formatYear(year int) string {
	return fmt.Sprintf("%d", year)
}

func formatWeek(week int) string {
	return fmt.Sprintf("W%02d", week)
}

func formatWeekLower(week int) string {
	return fmt.Sprintf("w%02d", week)
}

func formatProposalPage(issueNumber int) string {
	return fmt.Sprintf("%d.html", issueNumber)
}
