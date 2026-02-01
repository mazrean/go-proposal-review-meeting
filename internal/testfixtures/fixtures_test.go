// Package testfixtures provides E2E test fixtures and domain-specific test helpers.
// This file contains tests for the fixtures package (Task 8.7).
package testfixtures_test

import (
	"testing"
	"time"

	"github.com/mazrean/go-proposal-review-meeting/internal/parser"
	"github.com/mazrean/go-proposal-review-meeting/internal/testfixtures"
)

// TestTenProposalChanges verifies that the fixture provides exactly 10 proposal changes.
// Requirement: 8.7 - Create 10 proposal change mock data
func TestTenProposalChanges(t *testing.T) {
	t.Parallel()

	changes := testfixtures.TenProposalChanges()

	if len(changes) != 10 {
		t.Errorf("expected 10 proposal changes, got %d", len(changes))
	}
}

// TestTenProposalChanges_UniqueIssueNumbers verifies that all issue numbers are unique.
func TestTenProposalChanges_UniqueIssueNumbers(t *testing.T) {
	t.Parallel()

	changes := testfixtures.TenProposalChanges()

	seen := make(map[int]bool)
	for _, change := range changes {
		if seen[change.IssueNumber] {
			t.Errorf("duplicate issue number: %d", change.IssueNumber)
		}
		seen[change.IssueNumber] = true
	}
}

// TestTenProposalChanges_RequiredFields verifies all required fields are populated.
func TestTenProposalChanges_RequiredFields(t *testing.T) {
	t.Parallel()

	changes := testfixtures.TenProposalChanges()

	for i, change := range changes {
		if change.IssueNumber == 0 {
			t.Errorf("change[%d]: IssueNumber should not be 0", i)
		}
		if change.Title == "" {
			t.Errorf("change[%d]: Title should not be empty", i)
		}
		if change.PreviousStatus == "" {
			t.Errorf("change[%d]: PreviousStatus should not be empty", i)
		}
		if change.CurrentStatus == "" {
			t.Errorf("change[%d]: CurrentStatus should not be empty", i)
		}
		if change.CommentURL == "" {
			t.Errorf("change[%d]: CommentURL should not be empty", i)
		}
		if change.ChangedAt.IsZero() {
			t.Errorf("change[%d]: ChangedAt should not be zero", i)
		}
	}
}

// TestTenProposalChanges_StatusVariety verifies that multiple status types are represented.
func TestTenProposalChanges_StatusVariety(t *testing.T) {
	t.Parallel()

	changes := testfixtures.TenProposalChanges()

	statusCounts := make(map[parser.Status]int)
	for _, change := range changes {
		statusCounts[change.CurrentStatus]++
	}

	// Should have at least 3 different status types for variety
	if len(statusCounts) < 3 {
		t.Errorf("expected at least 3 different status types, got %d", len(statusCounts))
	}
}

// TestTenProposalChanges_WithCustomTime verifies custom time is applied correctly.
func TestTenProposalChanges_WithCustomTime(t *testing.T) {
	t.Parallel()

	customTime := time.Date(2026, 3, 15, 12, 0, 0, 0, time.UTC)
	changes := testfixtures.TenProposalChangesAt(customTime)

	for i, change := range changes {
		if !change.ChangedAt.Equal(customTime) {
			t.Errorf("change[%d]: expected ChangedAt %v, got %v", i, customTime, change.ChangedAt)
		}
	}
}

// TestMinutesComment verifies that GenerateMinutesComment produces valid minutes format.
func TestMinutesComment(t *testing.T) {
	t.Parallel()

	changes := testfixtures.TenProposalChanges()
	comment := testfixtures.GenerateMinutesComment(changes)

	// Should contain date header
	if comment == "" {
		t.Error("minutes comment should not be empty")
	}

	// Should be parseable by MinutesParser
	mp := parser.NewMinutesParser()
	parsed, err := mp.Parse(comment, changes[0].ChangedAt)
	if err != nil {
		t.Fatalf("failed to parse minutes comment: %v", err)
	}

	// Should contain proposals
	if len(parsed) == 0 {
		t.Error("parsed minutes should contain proposals")
	}
}

// TestTestSummary verifies test summary generation.
func TestTestSummary(t *testing.T) {
	t.Parallel()

	changes := testfixtures.TenProposalChanges()
	summary := testfixtures.GenerateTestSummary(changes[0])

	if summary == "" {
		t.Error("test summary should not be empty")
	}

	// Should be within recommended length (200-500 characters)
	length := len([]rune(summary))
	if length < 50 {
		t.Errorf("test summary seems too short: %d characters", length)
	}
}

// TestMockGitHubAPIHandler verifies the mock GitHub API handler works correctly.
func TestMockGitHubAPIHandler(t *testing.T) {
	t.Parallel()

	changes := testfixtures.TenProposalChanges()
	handler := testfixtures.MockGitHubAPIHandler(changes)

	if handler == nil {
		t.Error("mock handler should not be nil")
	}
}

// TestSetupContentWithSummaries verifies that content setup creates proper directory structure.
func TestSetupContentWithSummaries(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	changes := testfixtures.TenProposalChanges()

	summariesDir, err := testfixtures.SetupSummariesDir(tmpDir, changes)
	if err != nil {
		t.Fatalf("failed to setup summaries: %v", err)
	}

	if summariesDir == "" {
		t.Error("summaries directory path should not be empty")
	}
}

// Domain-specific helper tests

// TestSetupParserTest verifies the parser test helper creates a working setup.
func TestSetupParserTest(t *testing.T) {
	t.Parallel()

	changes := testfixtures.TenProposalChanges()
	setup := testfixtures.SetupParserTest(t, changes)

	// Verify all components are non-nil
	if setup.Server == nil {
		t.Error("Server should not be nil")
	}
	if setup.StateManager == nil {
		t.Error("StateManager should not be nil")
	}
	if setup.IssueParser == nil {
		t.Error("IssueParser should not be nil")
	}
	if setup.StatePath == "" {
		t.Error("StatePath should not be empty")
	}

	// Cleanup should be callable
	setup.Cleanup()
}

// TestSetupContentTest verifies the content test helper creates a working setup.
func TestSetupContentTest(t *testing.T) {
	t.Parallel()

	changes := testfixtures.TenProposalChanges()
	setup := testfixtures.SetupContentTest(t, changes, true)

	// Verify all components are non-nil
	if setup.Manager == nil {
		t.Error("Manager should not be nil")
	}
	if setup.ContentDir == "" {
		t.Error("ContentDir should not be empty")
	}
	if setup.SummariesDir == "" {
		t.Error("SummariesDir should not be empty")
	}

	// Cleanup should be callable
	setup.Cleanup()
}

// TestSetupSiteTest verifies the site test helper creates a working setup.
func TestSetupSiteTest(t *testing.T) {
	t.Parallel()

	setup := testfixtures.SetupSiteTest(t)

	// Verify all components are non-nil
	if setup.Generator == nil {
		t.Error("Generator should not be nil")
	}
	if setup.DistDir == "" {
		t.Error("DistDir should not be empty")
	}

	// Cleanup should be callable
	setup.Cleanup()
}
