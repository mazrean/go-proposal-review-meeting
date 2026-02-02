// Package content provides functionality for managing weekly proposal digest content.
package content

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/mazrean/go-proposal-review-meeting/internal/parser"
)

// Integration tests for Content domain.
// These tests verify the end-to-end flow from content generation to Markdown output,
// including summary integration and fallback processing.

// TestIntegration_FullContentWorkflow tests the complete workflow:
// ProposalChange → PrepareContent → IntegrateSummaries → WriteContent → ReadExistingContent
// This covers Requirements 2.1, 2.2, 2.3.
func TestIntegration_FullContentWorkflow(t *testing.T) {
	t.Parallel()

	baseTime := time.Date(2026, 1, 30, 12, 0, 0, 0, time.UTC)

	// Create temporary directories for content and summaries
	tmpDir := t.TempDir()
	contentDir := filepath.Join(tmpDir, "content")
	summariesDir := filepath.Join(tmpDir, "summaries")

	if err := os.MkdirAll(summariesDir, 0o755); err != nil {
		t.Fatalf("failed to create summaries dir: %v", err)
	}

	// Step 1: Simulate parser output - ProposalChange from parsed minutes
	changes := []parser.ProposalChange{
		{
			IssueNumber:    12345,
			Title:          "proposal: add new feature for testing",
			PreviousStatus: parser.StatusDiscussions,
			CurrentStatus:  parser.StatusAccepted,
			ChangedAt:      baseTime,
			CommentURL:     "https://github.com/golang/go/issues/33502#issuecomment-123",
			RelatedIssues:  []int{67890, 11111},
		},
		{
			IssueNumber:    22222,
			Title:          "proposal: improve error handling",
			PreviousStatus: parser.StatusActive,
			CurrentStatus:  parser.StatusDeclined,
			ChangedAt:      baseTime.Add(time.Hour),
			CommentURL:     "https://github.com/golang/go/issues/33502#issuecomment-456",
			RelatedIssues:  nil,
		},
	}

	// Step 2: Prepare AI-generated summary files
	summary1 := `このproposalは新機能を追加するものです。

**理由**: 既存のAPIでは複雑な操作が困難でした。
**背景**: Go 1.21からジェネリクスが導入され、より柔軟な実装が可能になりました。

詳細は[関連issue](https://github.com/golang/go/issues/99999)を参照してください。`

	summary2 := `このproposalはエラーハンドリングの改善を提案していましたが、既存の実装で十分と判断されました。

**理由**: 現在のerrorインターフェースで十分な機能が提供されています。
**背景**: errors.Is()とerrors.As()により、エラー処理は改善されています。`

	if err := os.WriteFile(filepath.Join(summariesDir, "12345.md"), []byte(summary1), 0o644); err != nil {
		t.Fatalf("failed to write summary file: %v", err)
	}
	if err := os.WriteFile(filepath.Join(summariesDir, "22222.md"), []byte(summary2), 0o644); err != nil {
		t.Fatalf("failed to write summary file: %v", err)
	}

	// Step 3: Initialize manager and process content
	mgr := NewManager(
		WithBaseDir(contentDir),
		WithSummariesDir(summariesDir),
	)

	// Step 4: Prepare content from changes
	content := mgr.PrepareContent(changes)

	// Verify PrepareContent output
	if content.Year != 2026 {
		t.Errorf("Year = %d, want 2026", content.Year)
	}
	if content.Week != 5 {
		t.Errorf("Week = %d, want 5", content.Week)
	}
	if len(content.Proposals) != 2 {
		t.Fatalf("len(Proposals) = %d, want 2", len(content.Proposals))
	}

	// Step 5: Read summaries and integrate them
	summaries, err := mgr.ReadSummaries()
	if err != nil {
		t.Fatalf("ReadSummaries() error = %v", err)
	}
	if len(summaries) != 2 {
		t.Errorf("len(summaries) = %d, want 2", len(summaries))
	}

	if err := mgr.IntegrateSummaries(content, summaries); err != nil {
		t.Fatalf("IntegrateSummaries() error = %v", err)
	}

	// Verify summaries were integrated
	for _, p := range content.Proposals {
		if p.Summary == "" {
			t.Errorf("Proposal[%d].Summary should not be empty after IntegrateSummaries", p.IssueNumber)
		}
	}

	// Step 6: Write content to disk
	if err := mgr.WriteContent(content); err != nil {
		t.Fatalf("WriteContent() error = %v", err)
	}

	// Step 7: Verify directory structure (Requirement 2.1)
	expectedDir := filepath.Join(contentDir, "2026/W05")
	if _, err := os.Stat(expectedDir); os.IsNotExist(err) {
		t.Errorf("Weekly directory %s was not created (Requirement 2.1)", expectedDir)
	}

	// Step 8: Verify individual proposal files were created (Requirement 2.2)
	expectedFiles := []string{"proposal-12345.md", "proposal-22222.md"}
	for _, filename := range expectedFiles {
		filePath := filepath.Join(expectedDir, filename)
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			t.Errorf("Proposal file %s was not created (Requirement 2.2)", filePath)
		}
	}

	// Step 9: Read back and verify content persistence (Requirement 2.3)
	readBack, err := mgr.ReadExistingContent(2026, 5)
	if err != nil {
		t.Fatalf("ReadExistingContent() error = %v", err)
	}
	if readBack == nil {
		t.Fatal("ReadExistingContent() returned nil")
	}

	if len(readBack.Proposals) != 2 {
		t.Fatalf("ReadExistingContent() len(Proposals) = %d, want 2", len(readBack.Proposals))
	}

	// Verify metadata was preserved in files (Requirement 2.3)
	for _, p := range readBack.Proposals {
		// Check required fields were persisted
		if p.IssueNumber == 0 {
			t.Error("IssueNumber was not persisted")
		}
		if p.Title == "" {
			t.Error("Title was not persisted")
		}
		if p.PreviousStatus == "" {
			t.Error("PreviousStatus was not persisted")
		}
		if p.CurrentStatus == "" {
			t.Error("CurrentStatus was not persisted")
		}
		if p.CommentURL == "" {
			t.Error("CommentURL was not persisted")
		}
		if p.ChangedAt.IsZero() {
			t.Error("ChangedAt was not persisted")
		}

		// Check summary was persisted
		if p.Summary == "" {
			t.Errorf("Proposal[%d].Summary was not persisted", p.IssueNumber)
		}
	}
}

// TestIntegration_FallbackProcessing tests the fallback flow when AI summaries are not available.
// This covers Requirement 3.4.
func TestIntegration_FallbackProcessing(t *testing.T) {
	t.Parallel()

	baseTime := time.Date(2026, 1, 30, 12, 0, 0, 0, time.UTC)

	tmpDir := t.TempDir()
	contentDir := filepath.Join(tmpDir, "content")
	summariesDir := filepath.Join(tmpDir, "summaries")

	// Create empty summaries directory (simulating AI generation failure)
	if err := os.MkdirAll(summariesDir, 0o755); err != nil {
		t.Fatalf("failed to create summaries dir: %v", err)
	}

	changes := []parser.ProposalChange{
		{
			IssueNumber:    33333,
			Title:          "proposal: add context.Context to API",
			PreviousStatus: parser.StatusDiscussions,
			CurrentStatus:  parser.StatusLikelyAccept,
			ChangedAt:      baseTime,
			CommentURL:     "https://github.com/golang/go/issues/33502#issuecomment-789",
			RelatedIssues:  nil,
		},
	}

	mgr := NewManager(
		WithBaseDir(contentDir),
		WithSummariesDir(summariesDir),
	)

	// Prepare content
	content := mgr.PrepareContent(changes)

	// Read summaries (will be empty)
	summaries, err := mgr.ReadSummaries()
	if err != nil {
		t.Fatalf("ReadSummaries() error = %v", err)
	}
	if len(summaries) != 0 {
		t.Errorf("Expected empty summaries map, got %d entries", len(summaries))
	}

	// Integrate (no-op since no summaries)
	if err := mgr.IntegrateSummaries(content, summaries); err != nil {
		t.Fatalf("IntegrateSummaries() error = %v", err)
	}

	// Verify summary is still empty before fallback
	if content.Proposals[0].Summary != "" {
		t.Error("Summary should be empty before ApplyFallback")
	}

	// Apply fallback (Requirement 3.4)
	if err := mgr.ApplyFallback(content); err != nil {
		t.Fatalf("ApplyFallback() error = %v", err)
	}

	// Verify fallback was applied
	p := content.Proposals[0]
	if p.Summary == "" {
		t.Error("Summary should not be empty after ApplyFallback (Requirement 3.4)")
	}

	// Verify fallback contains basic information
	expectedStrings := []string{
		"33333",                             // Issue number
		"proposal: add context.Context to API", // Title
		"discussions",                       // Previous status
		"likely_accept",                     // Current status
	}
	for _, s := range expectedStrings {
		if !strings.Contains(p.Summary, s) {
			t.Errorf("Fallback summary should contain %q, got %q", s, p.Summary)
		}
	}

	// Write content with fallback summary
	if err := mgr.WriteContent(content); err != nil {
		t.Fatalf("WriteContent() error = %v", err)
	}

	// Read back and verify fallback was persisted
	readBack, err := mgr.ReadExistingContent(2026, 5)
	if err != nil {
		t.Fatalf("ReadExistingContent() error = %v", err)
	}
	if readBack == nil {
		t.Fatal("ReadExistingContent() returned nil")
	}

	if readBack.Proposals[0].Summary == "" {
		t.Error("Fallback summary was not persisted")
	}
	if !strings.Contains(readBack.Proposals[0].Summary, "33333") {
		t.Error("Persisted fallback should contain issue number")
	}
}

// TestIntegration_PartialSummaryFallback tests mixed scenario where some proposals have
// AI summaries and others need fallback.
func TestIntegration_PartialSummaryFallback(t *testing.T) {
	t.Parallel()

	baseTime := time.Date(2026, 1, 30, 12, 0, 0, 0, time.UTC)

	tmpDir := t.TempDir()
	contentDir := filepath.Join(tmpDir, "content")
	summariesDir := filepath.Join(tmpDir, "summaries")

	if err := os.MkdirAll(summariesDir, 0o755); err != nil {
		t.Fatalf("failed to create summaries dir: %v", err)
	}

	changes := []parser.ProposalChange{
		{
			IssueNumber:    44444,
			Title:          "proposal: with AI summary",
			PreviousStatus: parser.StatusDiscussions,
			CurrentStatus:  parser.StatusAccepted,
			ChangedAt:      baseTime,
			CommentURL:     "https://github.com/golang/go/issues/33502#issuecomment-111",
			RelatedIssues:  nil,
		},
		{
			IssueNumber:    55555,
			Title:          "proposal: needs fallback",
			PreviousStatus: parser.StatusActive,
			CurrentStatus:  parser.StatusHold,
			ChangedAt:      baseTime.Add(time.Hour),
			CommentURL:     "https://github.com/golang/go/issues/33502#issuecomment-222",
			RelatedIssues:  nil,
		},
	}

	// Only create summary for first proposal
	aiSummary := "このproposalは素晴らしい新機能を追加します。技術的に優れた提案であり、コミュニティからの支持も得られました。"
	if err := os.WriteFile(filepath.Join(summariesDir, "44444.md"), []byte(aiSummary), 0o644); err != nil {
		t.Fatalf("failed to write summary file: %v", err)
	}

	mgr := NewManager(
		WithBaseDir(contentDir),
		WithSummariesDir(summariesDir),
	)

	content := mgr.PrepareContent(changes)

	// Integrate summaries
	summaries, err := mgr.ReadSummaries()
	if err != nil {
		t.Fatalf("ReadSummaries() error = %v", err)
	}
	if err := mgr.IntegrateSummaries(content, summaries); err != nil {
		t.Fatalf("IntegrateSummaries() error = %v", err)
	}

	// Apply fallback for proposals without AI summary
	if err := mgr.ApplyFallback(content); err != nil {
		t.Fatalf("ApplyFallback() error = %v", err)
	}

	// Verify both proposals have summaries
	var proposalWithAI, proposalWithFallback *ProposalContent
	for i := range content.Proposals {
		if content.Proposals[i].IssueNumber == 44444 {
			proposalWithAI = &content.Proposals[i]
		} else if content.Proposals[i].IssueNumber == 55555 {
			proposalWithFallback = &content.Proposals[i]
		}
	}

	if proposalWithAI == nil || proposalWithFallback == nil {
		t.Fatal("Could not find both proposals")
	}

	// AI summary should be preserved
	if proposalWithAI.Summary != aiSummary {
		t.Errorf("AI summary should be preserved, got %q", proposalWithAI.Summary)
	}

	// Fallback should be applied to second proposal
	if proposalWithFallback.Summary == "" {
		t.Error("Fallback should be applied to proposal without AI summary")
	}
	if !strings.Contains(proposalWithFallback.Summary, "55555") {
		t.Error("Fallback should contain issue number")
	}

	// Write and verify persistence
	if err := mgr.WriteContent(content); err != nil {
		t.Fatalf("WriteContent() error = %v", err)
	}

	readBack, err := mgr.ReadExistingContent(2026, 5)
	if err != nil {
		t.Fatalf("ReadExistingContent() error = %v", err)
	}

	for _, p := range readBack.Proposals {
		if p.Summary == "" {
			t.Errorf("Proposal[%d].Summary should not be empty after read back", p.IssueNumber)
		}
	}
}

// TestIntegration_ContentMergeWorkflow tests the merge workflow when updating
// existing weekly content with new changes.
func TestIntegration_ContentMergeWorkflow(t *testing.T) {
	t.Parallel()

	baseTime := time.Date(2026, 1, 30, 12, 0, 0, 0, time.UTC)

	tmpDir := t.TempDir()
	contentDir := filepath.Join(tmpDir, "content")
	summariesDir := filepath.Join(tmpDir, "summaries")

	if err := os.MkdirAll(summariesDir, 0o755); err != nil {
		t.Fatalf("failed to create summaries dir: %v", err)
	}

	mgr := NewManager(
		WithBaseDir(contentDir),
		WithSummariesDir(summariesDir),
	)

	// First batch of changes
	changes1 := []parser.ProposalChange{
		{
			IssueNumber:    66666,
			Title:          "proposal: first batch",
			PreviousStatus: parser.StatusDiscussions,
			CurrentStatus:  parser.StatusLikelyAccept,
			ChangedAt:      baseTime,
			CommentURL:     "https://github.com/golang/go/issues/33502#issuecomment-first",
			RelatedIssues:  nil,
		},
	}

	// Create summary for first batch
	summary1 := "最初のproposalの要約です。詳細な技術的背景を含みます。"
	if err := os.WriteFile(filepath.Join(summariesDir, "66666.md"), []byte(summary1), 0o644); err != nil {
		t.Fatalf("failed to write summary file: %v", err)
	}

	content1 := mgr.PrepareContent(changes1)
	summaries1, _ := mgr.ReadSummaries()
	_ = mgr.IntegrateSummaries(content1, summaries1)
	if err := mgr.WriteContentWithMerge(content1); err != nil {
		t.Fatalf("WriteContentWithMerge() first batch error = %v", err)
	}

	// Second batch of changes - update same proposal and add new one
	changes2 := []parser.ProposalChange{
		{
			IssueNumber:    66666,
			Title:          "proposal: first batch",
			PreviousStatus: parser.StatusLikelyAccept,
			CurrentStatus:  parser.StatusAccepted, // Status updated
			ChangedAt:      baseTime.Add(2 * time.Hour),
			CommentURL:     "https://github.com/golang/go/issues/33502#issuecomment-second",
			RelatedIssues:  nil,
		},
		{
			IssueNumber:    77777,
			Title:          "proposal: second batch",
			PreviousStatus: parser.StatusDiscussions,
			CurrentStatus:  parser.StatusDeclined,
			ChangedAt:      baseTime.Add(3 * time.Hour),
			CommentURL:     "https://github.com/golang/go/issues/33502#issuecomment-new",
			RelatedIssues:  nil,
		},
	}

	// Add summary for new proposal only
	summary2 := "2番目のproposalは却下されました。"
	if err := os.WriteFile(filepath.Join(summariesDir, "77777.md"), []byte(summary2), 0o644); err != nil {
		t.Fatalf("failed to write summary file: %v", err)
	}

	content2 := mgr.PrepareContent(changes2)
	summaries2, _ := mgr.ReadSummaries()
	_ = mgr.IntegrateSummaries(content2, summaries2)
	if err := mgr.WriteContentWithMerge(content2); err != nil {
		t.Fatalf("WriteContentWithMerge() second batch error = %v", err)
	}

	// Read back and verify merge result
	readBack, err := mgr.ReadExistingContent(2026, 5)
	if err != nil {
		t.Fatalf("ReadExistingContent() error = %v", err)
	}
	if readBack == nil {
		t.Fatal("ReadExistingContent() returned nil")
	}

	// Should have 2 proposals after merge
	if len(readBack.Proposals) != 2 {
		t.Errorf("len(Proposals) = %d, want 2 after merge", len(readBack.Proposals))
	}

	// Find merged proposal
	var mergedProposal *ProposalContent
	var newProposal *ProposalContent
	for i := range readBack.Proposals {
		if readBack.Proposals[i].IssueNumber == 66666 {
			mergedProposal = &readBack.Proposals[i]
		} else if readBack.Proposals[i].IssueNumber == 77777 {
			newProposal = &readBack.Proposals[i]
		}
	}

	if mergedProposal == nil {
		t.Fatal("Merged proposal (66666) not found")
	}
	if newProposal == nil {
		t.Fatal("New proposal (77777) not found")
	}

	// Verify merged proposal has updated status and new previous_status
	if mergedProposal.CurrentStatus != parser.StatusAccepted {
		t.Errorf("Merged proposal CurrentStatus = %q, want %q", mergedProposal.CurrentStatus, parser.StatusAccepted)
	}
	if mergedProposal.PreviousStatus != parser.StatusLikelyAccept {
		t.Errorf("Merged proposal PreviousStatus = %q, want %q (new)", mergedProposal.PreviousStatus, parser.StatusLikelyAccept)
	}

	// Verify summary was preserved from first batch
	if !strings.Contains(mergedProposal.Summary, "最初のproposal") {
		t.Errorf("Merged proposal should preserve original summary, got %q", mergedProposal.Summary)
	}

	// Verify new proposal has its summary
	if !strings.Contains(newProposal.Summary, "2番目のproposal") {
		t.Errorf("New proposal should have its summary, got %q", newProposal.Summary)
	}
}

// TestIntegration_LinkExtractionFromSummary tests that links in AI summaries
// are properly extracted and added to the proposal's Links.
func TestIntegration_LinkExtractionFromSummary(t *testing.T) {
	t.Parallel()

	baseTime := time.Date(2026, 1, 30, 12, 0, 0, 0, time.UTC)

	tmpDir := t.TempDir()
	contentDir := filepath.Join(tmpDir, "content")
	summariesDir := filepath.Join(tmpDir, "summaries")

	if err := os.MkdirAll(summariesDir, 0o755); err != nil {
		t.Fatalf("failed to create summaries dir: %v", err)
	}

	changes := []parser.ProposalChange{
		{
			IssueNumber:    88888,
			Title:          "proposal: with related links in summary",
			PreviousStatus: parser.StatusDiscussions,
			CurrentStatus:  parser.StatusAccepted,
			ChangedAt:      baseTime,
			CommentURL:     "https://github.com/golang/go/issues/33502#issuecomment-888",
			RelatedIssues:  []int{99999}, // Original related issue
		},
	}

	// Summary with additional links
	summaryWithLinks := `このproposalは素晴らしい提案です。

関連する議論:
- [#11111](https://github.com/golang/go/issues/11111) - 元々の議論
- [レビューコメント](https://github.com/golang/go/issues/22222#issuecomment-12345) - 詳細なレビュー

詳細は[#33333](https://github.com/golang/go/issues/33333)を参照してください。`

	if err := os.WriteFile(filepath.Join(summariesDir, "88888.md"), []byte(summaryWithLinks), 0o644); err != nil {
		t.Fatalf("failed to write summary file: %v", err)
	}

	mgr := NewManager(
		WithBaseDir(contentDir),
		WithSummariesDir(summariesDir),
	)

	content := mgr.PrepareContent(changes)

	// Original links should be present
	originalLinkCount := len(content.Proposals[0].Links)
	if originalLinkCount < 2 { // proposal issue + related issue 99999
		t.Errorf("Expected at least 2 original links, got %d", originalLinkCount)
	}

	// Integrate summaries (should extract links)
	summaries, _ := mgr.ReadSummaries()
	if err := mgr.IntegrateSummaries(content, summaries); err != nil {
		t.Fatalf("IntegrateSummaries() error = %v", err)
	}

	// Verify links were extracted and added
	p := content.Proposals[0]
	expectedURLs := []string{
		"https://github.com/golang/go/issues/88888",                  // Original proposal link
		"https://github.com/golang/go/issues/99999",                  // Original related issue
		"https://github.com/golang/go/issues/11111",                  // From summary
		"https://github.com/golang/go/issues/22222#issuecomment-12345", // From summary with anchor
		"https://github.com/golang/go/issues/33333",                  // From summary
	}

	urlSet := make(map[string]bool)
	for _, link := range p.Links {
		urlSet[link.URL] = true
	}

	for _, expectedURL := range expectedURLs {
		if !urlSet[expectedURL] {
			t.Errorf("Expected link URL %q not found in proposal links", expectedURL)
		}
	}

	// Write and verify links are persisted
	if err := mgr.WriteContent(content); err != nil {
		t.Fatalf("WriteContent() error = %v", err)
	}

	readBack, err := mgr.ReadExistingContent(2026, 5)
	if err != nil {
		t.Fatalf("ReadExistingContent() error = %v", err)
	}

	readBackURLSet := make(map[string]bool)
	for _, link := range readBack.Proposals[0].Links {
		readBackURLSet[link.URL] = true
	}

	// Verify key links are persisted
	keyURLs := []string{
		"https://github.com/golang/go/issues/88888",
		"https://github.com/golang/go/issues/11111",
	}
	for _, url := range keyURLs {
		if !readBackURLSet[url] {
			t.Errorf("Link URL %q was not persisted", url)
		}
	}
}

// TestIntegration_EmptyChanges tests behavior when no changes are provided.
func TestIntegration_EmptyChanges(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	contentDir := filepath.Join(tmpDir, "content")
	summariesDir := filepath.Join(tmpDir, "summaries")

	mgr := NewManager(
		WithBaseDir(contentDir),
		WithSummariesDir(summariesDir),
	)

	// Empty changes
	changes := []parser.ProposalChange{}
	content := mgr.PrepareContent(changes)

	if content.Year != 0 || content.Week != 0 {
		t.Errorf("Empty changes should result in zero Year/Week, got %d/W%02d", content.Year, content.Week)
	}
	if len(content.Proposals) != 0 {
		t.Errorf("Empty changes should result in no proposals, got %d", len(content.Proposals))
	}

	// WriteContent should handle nil/empty content gracefully
	if err := mgr.WriteContent(content); err != nil {
		t.Errorf("WriteContent() with empty content should not error, got %v", err)
	}

	// Content directory should not be created for empty content
	if _, err := os.Stat(contentDir); !os.IsNotExist(err) {
		t.Error("Content directory should not be created for empty changes")
	}
}
