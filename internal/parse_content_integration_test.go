// Package internal_test provides integration tests that span multiple domains.
// This file contains Parse→Content integration tests (Task 8.5).
package internal_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/mazrean/go-proposal-review-meeting/internal/content"
	"github.com/mazrean/go-proposal-review-meeting/internal/parser"
)

// TestIntegration_ParseToContent_TenProposals tests the complete flow from
// parsing GitHub API comments to generating content with 10 proposal changes.
// Requirements: 1.1, 1.2, 1.3, 1.4, 2.1, 2.2, 2.3
func TestIntegration_ParseToContent_TenProposals(t *testing.T) {
	t.Parallel()

	now := time.Now().Truncate(time.Second)

	// Generate 10 proposals with different statuses
	proposals := generateTenProposals(now)

	// Create mock GitHub API server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify GitHub API headers
		if r.Header.Get("Accept") != "application/vnd.github+json" {
			t.Errorf("expected Accept header 'application/vnd.github+json', got %q", r.Header.Get("Accept"))
		}

		// Generate mock GitHub API response with all 10 proposals in a single minutes comment
		comment := generateMinutesComment(proposals, now)

		comments := []map[string]any{
			{
				"id":         int64(99999999),
				"body":       comment,
				"created_at": now.Format(time.RFC3339),
				"updated_at": now.Format(time.RFC3339),
				"html_url":   "https://github.com/golang/go/issues/33502#issuecomment-99999999",
			},
		}

		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("ETag", `"test-etag-123"`)
		_ = json.NewEncoder(w).Encode(comments)
	}))
	defer server.Close()

	// Setup temporary directories
	tmpDir := t.TempDir()
	statePath := filepath.Join(tmpDir, "state.json")
	contentDir := filepath.Join(tmpDir, "content")
	summariesDir := filepath.Join(tmpDir, "summaries")

	// Create summaries directory and add AI-generated summaries for all proposals
	if err := os.MkdirAll(summariesDir, 0o755); err != nil {
		t.Fatalf("failed to create summaries dir: %v", err)
	}

	for _, p := range proposals {
		summary := generateTestSummary(p)
		summaryPath := filepath.Join(summariesDir, fmt.Sprintf("%d.md", p.IssueNumber))
		if err := os.WriteFile(summaryPath, []byte(summary), 0o644); err != nil {
			t.Fatalf("failed to write summary for issue %d: %v", p.IssueNumber, err)
		}
	}

	// Step 1: Parse - Create IssueParser and fetch changes
	sm := parser.NewStateManager(statePath)
	ip, err := parser.NewIssueParser(parser.IssueParserConfig{
		StateManager: sm,
		BaseURL:      server.URL,
		Token:        "test-token",
	})
	if err != nil {
		t.Fatalf("failed to create IssueParser: %v", err)
	}

	ctx := context.Background()
	changes, err := ip.FetchChanges(ctx)
	if err != nil {
		t.Fatalf("FetchChanges() error = %v", err)
	}

	// Verify we got 10 changes (Requirement 1.1, 1.2, 1.3)
	if len(changes) != 10 {
		t.Fatalf("expected 10 changes, got %d", len(changes))
	}

	// Verify each change has required fields (Requirement 1.4)
	for i, change := range changes {
		if change.IssueNumber == 0 {
			t.Errorf("change[%d] IssueNumber should not be 0", i)
		}
		if change.Title == "" {
			t.Errorf("change[%d] Title should not be empty", i)
		}
		if change.CurrentStatus == "" {
			t.Errorf("change[%d] CurrentStatus should not be empty", i)
		}
		if change.CommentURL == "" {
			t.Errorf("change[%d] CommentURL should not be empty", i)
		}
	}

	// Note: MinutesParser does not extract PreviousStatus from minutes comments.
	// In a real workflow, PreviousStatus would come from historical state tracking.
	// For this integration test, we set a default PreviousStatus to simulate this.
	for i := range changes {
		if changes[i].PreviousStatus == "" {
			changes[i].PreviousStatus = parser.StatusDiscussions
		}
	}

	// Step 2: Content - Create ContentManager and process content
	mgr := content.NewManager(
		content.WithBaseDir(contentDir),
		content.WithSummariesDir(summariesDir),
	)

	// Prepare content from changes
	weeklyContent := mgr.PrepareContent(changes)

	// Verify content structure (Requirement 2.1)
	if weeklyContent.Year == 0 || weeklyContent.Week == 0 {
		t.Error("WeeklyContent should have valid Year and Week")
	}

	if len(weeklyContent.Proposals) != 10 {
		t.Fatalf("expected 10 proposals in content, got %d", len(weeklyContent.Proposals))
	}

	// Read and integrate summaries
	summaries, err := mgr.ReadSummaries()
	if err != nil {
		t.Fatalf("ReadSummaries() error = %v", err)
	}

	if len(summaries) != 10 {
		t.Errorf("expected 10 summaries, got %d", len(summaries))
	}

	if err := mgr.IntegrateSummaries(weeklyContent, summaries); err != nil {
		t.Fatalf("IntegrateSummaries() error = %v", err)
	}

	// Verify all proposals have summaries
	for _, p := range weeklyContent.Proposals {
		if p.Summary == "" {
			t.Errorf("proposal %d should have a summary after integration", p.IssueNumber)
		}
	}

	// Write content to filesystem
	if err := mgr.WriteContent(weeklyContent); err != nil {
		t.Fatalf("WriteContent() error = %v", err)
	}

	// Step 3: Verify output (Requirements 2.2, 2.3)
	// Check directory structure
	year, week := now.ISOWeek()
	expectedDir := filepath.Join(contentDir, fmt.Sprintf("%d/W%02d", year, week))
	if _, err := os.Stat(expectedDir); os.IsNotExist(err) {
		t.Errorf("Weekly directory %s was not created (Requirement 2.1)", expectedDir)
	}

	// Check all 10 proposal files were created
	for _, p := range proposals {
		filename := fmt.Sprintf("proposal-%d.md", p.IssueNumber)
		filePath := filepath.Join(expectedDir, filename)
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			t.Errorf("Proposal file %s was not created (Requirement 2.2)", filePath)
		}
	}

	// Read back and verify content persistence (Requirement 2.3)
	readBack, err := mgr.ReadExistingContent(year, week)
	if err != nil {
		t.Fatalf("ReadExistingContent() error = %v", err)
	}
	if readBack == nil {
		t.Fatal("ReadExistingContent() returned nil")
	}

	if len(readBack.Proposals) != 10 {
		t.Fatalf("ReadExistingContent() returned %d proposals, want 10", len(readBack.Proposals))
	}

	// Verify all required metadata was persisted (Requirement 2.3)
	for _, p := range readBack.Proposals {
		if p.IssueNumber == 0 {
			t.Error("IssueNumber should be persisted")
		}
		if p.Title == "" {
			t.Error("Title should be persisted")
		}
		if p.PreviousStatus == "" {
			t.Error("PreviousStatus should be persisted")
		}
		if p.CurrentStatus == "" {
			t.Error("CurrentStatus should be persisted")
		}
		if p.CommentURL == "" {
			t.Error("CommentURL should be persisted")
		}
		if p.ChangedAt.IsZero() {
			t.Error("ChangedAt should be persisted")
		}
		if p.Summary == "" {
			t.Errorf("Summary should be persisted for proposal %d", p.IssueNumber)
		}
		if len(p.Links) == 0 {
			t.Errorf("Links should be persisted for proposal %d", p.IssueNumber)
		}
	}
}

// TestIntegration_ParseToContent_StatusVariety verifies that all proposal
// statuses are correctly parsed and passed through to content.
// Requirements: 1.3 (status extraction)
// Note: StatusActive is not tested as it has no matching pattern in MinutesParser.
func TestIntegration_ParseToContent_StatusVariety(t *testing.T) {
	t.Parallel()

	now := time.Now().Truncate(time.Second)

	// Create proposals with each status type that has a matching pattern
	// StatusActive is excluded as it has no matching pattern in MinutesParser
	statusProposals := []struct {
		issueNumber int
		title       string
		status      parser.Status
	}{
		{10001, "proposal: accepted feature", parser.StatusAccepted},
		{10002, "proposal: declined feature", parser.StatusDeclined},
		{10003, "proposal: likely accept", parser.StatusLikelyAccept},
		{10004, "proposal: likely decline", parser.StatusLikelyDecline},
		{10005, "proposal: on hold", parser.StatusHold},
		{10006, "proposal: under discussion", parser.StatusDiscussions},
	}

	// Generate minutes comment with all status types
	var minutesBuilder strings.Builder
	minutesBuilder.WriteString(fmt.Sprintf("**%s** / **@rsc**\n\n", now.Format("2006-01-02")))
	for _, p := range statusProposals {
		minutesBuilder.WriteString(fmt.Sprintf("- [#%d](https://github.com/golang/go/issues/%d) **%s**\n",
			p.issueNumber, p.issueNumber, p.title))
		minutesBuilder.WriteString(fmt.Sprintf("  - **%s**\n", statusToMinutesFormat(p.status)))
		minutesBuilder.WriteString("\n")
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		comments := []map[string]any{
			{
				"id":         int64(88888888),
				"body":       minutesBuilder.String(),
				"created_at": now.Format(time.RFC3339),
				"updated_at": now.Format(time.RFC3339),
				"html_url":   "https://github.com/golang/go/issues/33502#issuecomment-88888888",
			},
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(comments)
	}))
	defer server.Close()

	tmpDir := t.TempDir()
	statePath := filepath.Join(tmpDir, "state.json")
	contentDir := filepath.Join(tmpDir, "content")

	sm := parser.NewStateManager(statePath)
	ip, err := parser.NewIssueParser(parser.IssueParserConfig{
		StateManager: sm,
		BaseURL:      server.URL,
		Token:        "test-token",
	})
	if err != nil {
		t.Fatalf("failed to create IssueParser: %v", err)
	}

	ctx := context.Background()
	changes, err := ip.FetchChanges(ctx)
	if err != nil {
		t.Fatalf("FetchChanges() error = %v", err)
	}

	if len(changes) != len(statusProposals) {
		t.Fatalf("expected %d changes, got %d", len(statusProposals), len(changes))
	}

	// Set default PreviousStatus (see note in TenProposals test)
	for i := range changes {
		if changes[i].PreviousStatus == "" {
			changes[i].PreviousStatus = parser.StatusDiscussions
		}
	}

	// Create a map for easier lookup
	changeMap := make(map[int]parser.ProposalChange)
	for _, c := range changes {
		changeMap[c.IssueNumber] = c
	}

	// Verify each status was correctly parsed
	for _, expected := range statusProposals {
		change, ok := changeMap[expected.issueNumber]
		if !ok {
			t.Errorf("change for issue %d not found", expected.issueNumber)
			continue
		}
		if change.CurrentStatus != expected.status {
			t.Errorf("issue %d: expected status %q, got %q",
				expected.issueNumber, expected.status, change.CurrentStatus)
		}
	}

	// Pass through ContentManager and verify status preservation
	mgr := content.NewManager(content.WithBaseDir(contentDir))
	weeklyContent := mgr.PrepareContent(changes)
	mgr.ApplyFallback(weeklyContent)

	if err := mgr.WriteContent(weeklyContent); err != nil {
		t.Fatalf("WriteContent() error = %v", err)
	}

	year, week := now.ISOWeek()
	readBack, err := mgr.ReadExistingContent(year, week)
	if err != nil {
		t.Fatalf("ReadExistingContent() error = %v", err)
	}

	// Verify statuses are preserved in content
	contentMap := make(map[int]content.ProposalContent)
	for _, p := range readBack.Proposals {
		contentMap[p.IssueNumber] = p
	}

	for _, expected := range statusProposals {
		proposal, ok := contentMap[expected.issueNumber]
		if !ok {
			t.Errorf("content for issue %d not found", expected.issueNumber)
			continue
		}
		if proposal.CurrentStatus != expected.status {
			t.Errorf("issue %d in content: expected status %q, got %q",
				expected.issueNumber, expected.status, proposal.CurrentStatus)
		}
	}
}

// TestIntegration_ParseToContent_WithFallback tests the flow when AI summaries
// are not available and fallback is applied.
// Requirements: 1.1, 2.1, 2.2, 2.3
func TestIntegration_ParseToContent_WithFallback(t *testing.T) {
	t.Parallel()

	now := time.Now().Truncate(time.Second)

	proposals := generateTenProposals(now)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		comment := generateMinutesComment(proposals, now)
		comments := []map[string]any{
			{
				"id":         int64(77777777),
				"body":       comment,
				"created_at": now.Format(time.RFC3339),
				"updated_at": now.Format(time.RFC3339),
				"html_url":   "https://github.com/golang/go/issues/33502#issuecomment-77777777",
			},
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(comments)
	}))
	defer server.Close()

	tmpDir := t.TempDir()
	statePath := filepath.Join(tmpDir, "state.json")
	contentDir := filepath.Join(tmpDir, "content")
	summariesDir := filepath.Join(tmpDir, "summaries") // Empty - no AI summaries

	// Create empty summaries directory (simulating AI generation failure)
	if err := os.MkdirAll(summariesDir, 0o755); err != nil {
		t.Fatalf("failed to create summaries dir: %v", err)
	}

	// Parse
	sm := parser.NewStateManager(statePath)
	ip, err := parser.NewIssueParser(parser.IssueParserConfig{
		StateManager: sm,
		BaseURL:      server.URL,
		Token:        "test-token",
	})
	if err != nil {
		t.Fatalf("failed to create IssueParser: %v", err)
	}

	ctx := context.Background()
	changes, err := ip.FetchChanges(ctx)
	if err != nil {
		t.Fatalf("FetchChanges() error = %v", err)
	}

	if len(changes) != 10 {
		t.Fatalf("expected 10 changes, got %d", len(changes))
	}

	// Set default PreviousStatus (see note in TenProposals test)
	for i := range changes {
		if changes[i].PreviousStatus == "" {
			changes[i].PreviousStatus = parser.StatusDiscussions
		}
	}

	// Content with fallback
	mgr := content.NewManager(
		content.WithBaseDir(contentDir),
		content.WithSummariesDir(summariesDir),
	)

	weeklyContent := mgr.PrepareContent(changes)

	summaries, err := mgr.ReadSummaries()
	if err != nil {
		t.Fatalf("ReadSummaries() error = %v", err)
	}

	// Should be empty
	if len(summaries) != 0 {
		t.Errorf("expected 0 summaries, got %d", len(summaries))
	}

	// Integrate (no-op) and apply fallback
	_ = mgr.IntegrateSummaries(weeklyContent, summaries)
	_ = mgr.ApplyFallback(weeklyContent)

	// Verify fallback was applied
	for _, p := range weeklyContent.Proposals {
		if p.Summary == "" {
			t.Errorf("proposal %d should have fallback summary", p.IssueNumber)
		}
		// Fallback should contain issue number
		if !strings.Contains(p.Summary, fmt.Sprintf("%d", p.IssueNumber)) {
			t.Errorf("fallback summary should contain issue number %d", p.IssueNumber)
		}
	}

	// Write and verify persistence
	if err := mgr.WriteContent(weeklyContent); err != nil {
		t.Fatalf("WriteContent() error = %v", err)
	}

	year, week := now.ISOWeek()
	readBack, err := mgr.ReadExistingContent(year, week)
	if err != nil {
		t.Fatalf("ReadExistingContent() error = %v", err)
	}

	if len(readBack.Proposals) != 10 {
		t.Fatalf("expected 10 proposals, got %d", len(readBack.Proposals))
	}

	// Verify fallback was persisted
	for _, p := range readBack.Proposals {
		if p.Summary == "" {
			t.Errorf("fallback summary should be persisted for proposal %d", p.IssueNumber)
		}
	}
}

// TestIntegration_ParseToContent_WriteChangesJSON tests the changes.json output
// for passing data between workflow steps.
// Requirements: 1.4
func TestIntegration_ParseToContent_WriteChangesJSON(t *testing.T) {
	t.Parallel()

	now := time.Now().Truncate(time.Second)

	proposals := generateTenProposals(now)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		comment := generateMinutesComment(proposals, now)
		comments := []map[string]any{
			{
				"id":         int64(66666666),
				"body":       comment,
				"created_at": now.Format(time.RFC3339),
				"updated_at": now.Format(time.RFC3339),
				"html_url":   "https://github.com/golang/go/issues/33502#issuecomment-66666666",
			},
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(comments)
	}))
	defer server.Close()

	tmpDir := t.TempDir()
	statePath := filepath.Join(tmpDir, "state.json")
	changesPath := filepath.Join(tmpDir, "changes.json")

	sm := parser.NewStateManager(statePath)
	ip, err := parser.NewIssueParser(parser.IssueParserConfig{
		StateManager: sm,
		BaseURL:      server.URL,
		Token:        "test-token",
	})
	if err != nil {
		t.Fatalf("failed to create IssueParser: %v", err)
	}

	ctx := context.Background()
	changes, err := ip.FetchChanges(ctx)
	if err != nil {
		t.Fatalf("FetchChanges() error = %v", err)
	}

	// Write changes to JSON file
	if err := ip.WriteChangesJSON(changes, changesPath); err != nil {
		t.Fatalf("WriteChangesJSON() error = %v", err)
	}

	// Verify JSON file exists and is valid
	data, err := os.ReadFile(changesPath)
	if err != nil {
		t.Fatalf("failed to read changes.json: %v", err)
	}

	var output parser.ChangesOutput
	if err := json.Unmarshal(data, &output); err != nil {
		t.Fatalf("failed to unmarshal changes.json: %v", err)
	}

	// Verify all 10 changes are in the output
	if len(output.Changes) != 10 {
		t.Errorf("expected 10 changes in JSON, got %d", len(output.Changes))
	}

	// Verify week format
	if output.Week == "" {
		t.Error("Week should be set in changes.json")
	}

	// Verify each change has required fields
	for _, c := range output.Changes {
		if c.IssueNumber == 0 {
			t.Error("IssueNumber should not be 0")
		}
		if c.Title == "" {
			t.Error("Title should not be empty")
		}
		if c.CurrentStatus == "" {
			t.Error("CurrentStatus should not be empty")
		}
		if c.CommentURL == "" {
			t.Error("CommentURL should not be empty")
		}
	}
}

// Helper types and functions

type testProposal struct {
	IssueNumber int
	Title       string
	Status      parser.Status
}

func generateTenProposals(baseTime time.Time) []testProposal {
	// Use only statuses that have matching patterns in MinutesParser
	// StatusActive is not included as it has no matching pattern in the parser
	statuses := []parser.Status{
		parser.StatusAccepted,
		parser.StatusDeclined,
		parser.StatusLikelyAccept,
		parser.StatusLikelyDecline,
		parser.StatusHold,
		parser.StatusDiscussions,
		parser.StatusAccepted,
		parser.StatusLikelyAccept,
		parser.StatusDeclined,
		parser.StatusLikelyDecline,
	}

	proposals := make([]testProposal, 10)
	for i := 0; i < 10; i++ {
		proposals[i] = testProposal{
			IssueNumber: 50000 + i,
			Title:       fmt.Sprintf("proposal: test feature %d", i+1),
			Status:      statuses[i],
		}
	}
	return proposals
}

func generateMinutesComment(proposals []testProposal, baseTime time.Time) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("**%s** / **@rsc**\n\n", baseTime.Format("2006-01-02")))

	for _, p := range proposals {
		b.WriteString(fmt.Sprintf("- [#%d](https://github.com/golang/go/issues/%d) **%s**\n",
			p.IssueNumber, p.IssueNumber, p.Title))
		b.WriteString(fmt.Sprintf("  - **%s**\n", statusToMinutesFormat(p.Status)))
		b.WriteString("\n")
	}

	return b.String()
}

func statusToMinutesFormat(s parser.Status) string {
	// Generate patterns that match MinutesParser's statusPatterns regex
	switch s {
	case parser.StatusAccepted:
		return "accepted**" // matches `**accepted**`
	case parser.StatusDeclined:
		return "declined**" // matches `**declined**`
	case parser.StatusLikelyAccept:
		return "likely accept**" // matches `**likely accept`
	case parser.StatusLikelyDecline:
		return "likely decline**" // matches `**likely decline`
	case parser.StatusHold:
		return "put on hold" // matches `put on hold`
	case parser.StatusDiscussions:
		return "discussion ongoing" // matches `discussion ongoing`
	default:
		return string(s)
	}
}

func generateTestSummary(p testProposal) string {
	return fmt.Sprintf(`このproposal「%s」は%sとなりました。

**理由**: テスト用の理由説明です。技術的な背景と判断理由を含みます。

**背景**: Go言語の発展に関連する技術的背景です。既存のAPI設計との整合性や、
パフォーマンスへの影響、互換性の観点から検討されました。

詳細は[関連issue](https://github.com/golang/go/issues/%d)を参照してください。`,
		p.Title, statusToJapanese(p.Status), p.IssueNumber)
}

func statusToJapanese(s parser.Status) string {
	switch s {
	case parser.StatusAccepted:
		return "承認"
	case parser.StatusDeclined:
		return "却下"
	case parser.StatusLikelyAccept:
		return "承認見込み"
	case parser.StatusLikelyDecline:
		return "却下見込み"
	case parser.StatusActive:
		return "活発な議論中"
	case parser.StatusHold:
		return "保留"
	case parser.StatusDiscussions:
		return "議論中"
	default:
		return string(s)
	}
}
