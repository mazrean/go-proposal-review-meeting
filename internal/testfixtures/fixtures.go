// Package testfixtures provides E2E test fixtures and domain-specific test helpers
// for testing the Go Proposal Weekly Digest system.
//
// This package provides:
// - TenProposalChanges: 10 proposal change mock data with various statuses
// - GenerateMinutesComment: Creates parseable minutes comments
// - GenerateTestSummary: Creates Japanese test summaries
// - MockGitHubAPIHandler: HTTP handler for mocking GitHub API
// - SetupSummariesDir: Creates summary files for Content domain tests
//
// Task 8.7: E2E test fixtures and domain test helpers
// Requirements: 1.1, 2.1, 4.1
package testfixtures

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/mazrean/go-proposal-review-meeting/internal/content"
	"github.com/mazrean/go-proposal-review-meeting/internal/parser"
	"github.com/mazrean/go-proposal-review-meeting/internal/site"
)

// DefaultBaseTime is the default time used for proposal changes.
var DefaultBaseTime = time.Date(2026, 2, 5, 12, 0, 0, 0, time.UTC)

// TenProposalChanges returns 10 proposal changes with various statuses.
// This is the primary fixture for E2E tests.
func TenProposalChanges() []parser.ProposalChange {
	return TenProposalChangesAt(DefaultBaseTime)
}

// TenProposalChangesAt returns 10 proposal changes at the specified time.
func TenProposalChangesAt(baseTime time.Time) []parser.ProposalChange {
	// Use a variety of statuses that are parseable by MinutesParser
	// StatusActive is excluded as it has no matching pattern in the parser
	proposals := []struct {
		issueNumber    int
		title          string
		currentStatus  parser.Status
		previousStatus parser.Status
	}{
		{60001, "proposal: add structured concurrency", parser.StatusAccepted, parser.StatusLikelyAccept},
		{60002, "proposal: generic type aliases", parser.StatusDeclined, parser.StatusDiscussions},
		{60003, "proposal: range over functions", parser.StatusLikelyAccept, parser.StatusDiscussions},
		{60004, "proposal: error handling improvements", parser.StatusLikelyDecline, parser.StatusDiscussions},
		{60005, "proposal: extended context API", parser.StatusHold, parser.StatusLikelyAccept},
		{60006, "proposal: new testing framework", parser.StatusDiscussions, parser.StatusDiscussions},
		{60007, "proposal: io/v2 package", parser.StatusAccepted, parser.StatusLikelyAccept},
		{60008, "proposal: memory model updates", parser.StatusLikelyAccept, parser.StatusDiscussions},
		{60009, "proposal: build constraints revision", parser.StatusDeclined, parser.StatusLikelyDecline},
		{60010, "proposal: net/http/v2", parser.StatusLikelyDecline, parser.StatusDiscussions},
	}

	changes := make([]parser.ProposalChange, len(proposals))
	for i, p := range proposals {
		changes[i] = parser.ProposalChange{
			IssueNumber:    p.issueNumber,
			Title:          p.title,
			PreviousStatus: p.previousStatus,
			CurrentStatus:  p.currentStatus,
			ChangedAt:      baseTime,
			CommentURL:     fmt.Sprintf("https://github.com/golang/go/issues/33502#issuecomment-%d", 90000000+i),
			RelatedIssues:  []int{},
		}
	}

	return changes
}

// GenerateMinutesComment creates a minutes comment in the format expected by MinutesParser.
func GenerateMinutesComment(changes []parser.ProposalChange) string {
	if len(changes) == 0 {
		return ""
	}

	var b strings.Builder

	// Date header
	b.WriteString(fmt.Sprintf("**%s** / **@rsc**\n\n", changes[0].ChangedAt.Format("2006-01-02")))

	for _, change := range changes {
		// Proposal line: - [#NNNNN](URL) **title**
		b.WriteString(fmt.Sprintf("- [#%d](https://github.com/golang/go/issues/%d) **%s**\n",
			change.IssueNumber, change.IssueNumber, change.Title))

		// Status line
		b.WriteString(fmt.Sprintf("  - **%s**\n", statusToMinutesFormat(change.CurrentStatus)))
		b.WriteString("\n")
	}

	return b.String()
}

// statusToMinutesFormat converts a status to the format used in minutes comments.
func statusToMinutesFormat(s parser.Status) string {
	switch s {
	case parser.StatusAccepted:
		return "accepted**"
	case parser.StatusDeclined:
		return "declined**"
	case parser.StatusLikelyAccept:
		return "likely accept**"
	case parser.StatusLikelyDecline:
		return "likely decline**"
	case parser.StatusHold:
		return "put on hold"
	case parser.StatusDiscussions:
		return "discussion ongoing"
	default:
		return string(s)
	}
}

// GenerateTestSummary creates a Japanese test summary for a proposal change.
func GenerateTestSummary(change parser.ProposalChange) string {
	return fmt.Sprintf(`このproposal「%s」は%sとなりました。

**理由**: このproposalは技術的な実現可能性と既存APIとの互換性を考慮して慎重に検討されました。Go言語の設計原則に沿った判断が行われています。

**背景**: Go言語の発展に関連する重要な変更提案です。パフォーマンス、後方互換性、開発者体験の観点から総合的に評価されました。

詳細は[関連issue](https://github.com/golang/go/issues/%d)を参照してください。`,
		change.Title, statusToJapanese(change.CurrentStatus), change.IssueNumber)
}

// statusToJapanese converts a status to Japanese text.
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

// MockGitHubAPIHandler returns an HTTP handler that simulates the GitHub API
// returning comments containing the given proposal changes.
func MockGitHubAPIHandler(changes []parser.ProposalChange) http.HandlerFunc {
	comment := GenerateMinutesComment(changes)
	var commentedAt time.Time
	if len(changes) > 0 {
		commentedAt = changes[0].ChangedAt
	} else {
		commentedAt = DefaultBaseTime
	}

	return func(w http.ResponseWriter, r *http.Request) {
		// Verify GitHub API headers
		if r.Header.Get("Accept") != "application/vnd.github+json" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		comments := []map[string]any{
			{
				"id":         int64(99999999),
				"body":       comment,
				"created_at": commentedAt.Format(time.RFC3339),
				"updated_at": commentedAt.Format(time.RFC3339),
				"html_url":   "https://github.com/golang/go/issues/33502#issuecomment-99999999",
			},
		}

		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("ETag", `"test-etag-fixture"`)
		_ = json.NewEncoder(w).Encode(comments)
	}
}

// SetupSummariesDir creates a summaries directory with AI-generated summary files
// for each proposal change. Returns the path to the summaries directory.
func SetupSummariesDir(tmpDir string, changes []parser.ProposalChange) (string, error) {
	summariesDir := filepath.Join(tmpDir, "summaries")
	if err := os.MkdirAll(summariesDir, 0o755); err != nil {
		return "", fmt.Errorf("failed to create summaries directory: %w", err)
	}

	for _, change := range changes {
		summary := GenerateTestSummary(change)
		summaryPath := filepath.Join(summariesDir, fmt.Sprintf("%d.md", change.IssueNumber))
		if err := os.WriteFile(summaryPath, []byte(summary), 0o644); err != nil {
			return "", fmt.Errorf("failed to write summary for issue %d: %w", change.IssueNumber, err)
		}
	}

	return summariesDir, nil
}

// TestingT is an interface for testing.T to avoid import cycle.
type TestingT interface {
	Helper()
	Fatalf(format string, args ...any)
	TempDir() string
}

// ParserTestSetup contains all components needed for Parser domain tests.
type ParserTestSetup struct {
	Server       *httptest.Server
	StateManager *parser.StateManager
	IssueParser  *parser.IssueParser
	StatePath    string
	Cleanup      func()
}

// SetupParserTest creates a complete test setup for Parser domain tests.
// It includes a mock GitHub API server and configured IssueParser.
func SetupParserTest(t TestingT, changes []parser.ProposalChange) *ParserTestSetup {
	t.Helper()

	tmpDir := t.TempDir()
	statePath := filepath.Join(tmpDir, "state.json")

	// Create mock server
	server := httptest.NewServer(MockGitHubAPIHandler(changes))

	// Create state manager
	sm := parser.NewStateManager(statePath)

	// Create issue parser
	ip, err := parser.NewIssueParser(parser.IssueParserConfig{
		StateManager: sm,
		BaseURL:      server.URL,
		Token:        "test-token",
	})
	if err != nil {
		server.Close()
		t.Fatalf("failed to create IssueParser: %v", err)
	}

	return &ParserTestSetup{
		Server:       server,
		StateManager: sm,
		IssueParser:  ip,
		StatePath:    statePath,
		Cleanup: func() {
			server.Close()
		},
	}
}

// ContentTestSetup contains all components needed for Content domain tests.
type ContentTestSetup struct {
	Manager      *content.Manager
	ContentDir   string
	SummariesDir string
	Cleanup      func()
}

// SetupContentTest creates a complete test setup for Content domain tests.
// If withSummaries is true, it also creates summary files for each proposal.
func SetupContentTest(t TestingT, changes []parser.ProposalChange, withSummaries bool) *ContentTestSetup {
	t.Helper()

	tmpDir := t.TempDir()
	contentDir := filepath.Join(tmpDir, "content")
	summariesDir := filepath.Join(tmpDir, "summaries")

	// Create directories
	if err := os.MkdirAll(contentDir, 0o755); err != nil {
		t.Fatalf("failed to create content directory: %v", err)
	}

	if withSummaries {
		var err error
		summariesDir, err = SetupSummariesDir(tmpDir, changes)
		if err != nil {
			t.Fatalf("failed to setup summaries: %v", err)
		}
	} else {
		if err := os.MkdirAll(summariesDir, 0o755); err != nil {
			t.Fatalf("failed to create summaries directory: %v", err)
		}
	}

	// Create manager
	mgr := content.NewManager(
		content.WithBaseDir(contentDir),
		content.WithSummariesDir(summariesDir),
	)

	return &ContentTestSetup{
		Manager:      mgr,
		ContentDir:   contentDir,
		SummariesDir: summariesDir,
		Cleanup:      func() {}, // tmpDir is cleaned up by t.TempDir()
	}
}

// SiteTestSetup contains all components needed for Site domain tests.
type SiteTestSetup struct {
	Generator *site.Generator
	DistDir   string
	Cleanup   func()
}

// SetupSiteTest creates a complete test setup for Site domain tests.
func SetupSiteTest(t TestingT) *SiteTestSetup {
	t.Helper()

	tmpDir := t.TempDir()
	distDir := filepath.Join(tmpDir, "dist")

	// Create dist directory
	if err := os.MkdirAll(distDir, 0o755); err != nil {
		t.Fatalf("failed to create dist directory: %v", err)
	}

	// Create generator
	gen := site.NewGenerator(
		site.WithDistDir(distDir),
		site.WithGeneratorSiteURL("https://test.example.com"),
	)

	return &SiteTestSetup{
		Generator: gen,
		DistDir:   distDir,
		Cleanup:   func() {}, // tmpDir is cleaned up by t.TempDir()
	}
}

// PrepareWeeklyContent creates WeeklyContent from proposal changes for site tests.
func PrepareWeeklyContent(changes []parser.ProposalChange) *content.WeeklyContent {
	mgr := content.NewManager()
	weeklyContent := mgr.PrepareContent(changes)
	_ = mgr.ApplyFallback(weeklyContent)
	return weeklyContent
}
