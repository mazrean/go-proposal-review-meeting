// Package parser provides integration tests for the Parser domain.
// These tests verify the full flow of GitHub API comment retrieval and parsing.
package parser

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// TestIntegration_FetchChangesFlow tests the complete flow of fetching changes
// from GitHub API and producing output.
// Requirements: 1.1
func TestIntegration_FetchChangesFlow(t *testing.T) {
	t.Parallel()

	now := time.Now().Truncate(time.Second)

	// Create a realistic mock server that mimics GitHub API behavior
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request headers match GitHub API expectations
		accept := r.Header.Get("Accept")
		if accept != "application/vnd.github+json" {
			t.Errorf("expected Accept header 'application/vnd.github+json', got %q", accept)
		}

		apiVersion := r.Header.Get("X-GitHub-Api-Version")
		if apiVersion != "2022-11-28" {
			t.Errorf("expected X-GitHub-Api-Version header '2022-11-28', got %q", apiVersion)
		}

		// Verify Authorization header is set when token is provided
		auth := r.Header.Get("Authorization")
		if auth == "" {
			t.Error("expected Authorization header to be set")
		}

		// Return realistic GitHub API response
		comments := []map[string]any{
			{
				"id":         int64(12345678),
				"body":       "**2026-01-30** / **@rsc**\n\n- [#12345](https://github.com/golang/go/issues/12345) **proposal: add generics support**\n  - **accepted**\n\n- [#67890](https://github.com/golang/go/issues/67890) **proposal: improve error handling**\n  - **likely accept**\n",
				"created_at": now.Format(time.RFC3339),
				"updated_at": now.Format(time.RFC3339),
				"html_url":   "https://github.com/golang/go/issues/33502#issuecomment-12345678",
			},
		}

		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("ETag", `"abc123"`)
		_ = json.NewEncoder(w).Encode(comments)
	}))
	defer server.Close()

	tmpDir := t.TempDir()
	statePath := filepath.Join(tmpDir, "state.json")
	sm := NewStateManager(statePath)

	ip, err := NewIssueParser(IssueParserConfig{
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

	// Verify changes were extracted
	if len(changes) != 2 {
		t.Errorf("expected 2 changes, got %d", len(changes))
	}

	// Verify change content
	if len(changes) >= 1 {
		if changes[0].IssueNumber != 12345 {
			t.Errorf("expected first change issue number 12345, got %d", changes[0].IssueNumber)
		}
		if changes[0].CurrentStatus != StatusAccepted {
			t.Errorf("expected first change status Accepted, got %s", changes[0].CurrentStatus)
		}
		if changes[0].CommentURL != "https://github.com/golang/go/issues/33502#issuecomment-12345678" {
			t.Errorf("expected comment URL to be set, got %q", changes[0].CommentURL)
		}
	}

	if len(changes) >= 2 {
		if changes[1].IssueNumber != 67890 {
			t.Errorf("expected second change issue number 67890, got %d", changes[1].IssueNumber)
		}
		if changes[1].CurrentStatus != StatusLikelyAccept {
			t.Errorf("expected second change status LikelyAccept, got %s", changes[1].CurrentStatus)
		}
	}

	// Verify state was updated
	state, err := sm.LoadState()
	if err != nil {
		t.Fatalf("failed to load state: %v", err)
	}
	if state.LastCommentID != "12345678" {
		t.Errorf("expected state LastCommentID to be updated to '12345678', got %q", state.LastCommentID)
	}

	// Write changes to JSON and verify
	outputPath := filepath.Join(tmpDir, "changes.json")
	if err := ip.WriteChangesJSON(changes, outputPath); err != nil {
		t.Fatalf("WriteChangesJSON() error = %v", err)
	}

	// Verify output file content
	data, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("failed to read output file: %v", err)
	}

	var output ChangesOutput
	if err := json.Unmarshal(data, &output); err != nil {
		t.Fatalf("failed to unmarshal output: %v", err)
	}

	if len(output.Changes) != 2 {
		t.Errorf("expected 2 changes in output, got %d", len(output.Changes))
	}
}

// TestIntegration_RateLimitHandling tests handling of GitHub API rate limit errors.
// Requirements: 1.5
func TestIntegration_RateLimitHandling(t *testing.T) {
	t.Parallel()

	var logBuffer bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&logBuffer, &slog.HandlerOptions{Level: slog.LevelInfo}))

	// Create mock server that returns 429 rate limit error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-RateLimit-Limit", "5000")
		w.Header().Set("X-RateLimit-Remaining", "0")
		w.Header().Set("X-RateLimit-Reset", "1706630400") // Unix timestamp
		w.Header().Set("Retry-After", "60")
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte(`{"message": "API rate limit exceeded"}`))
	}))
	defer server.Close()

	tmpDir := t.TempDir()
	statePath := filepath.Join(tmpDir, "state.json")
	sm := NewStateManager(statePath)

	ip, err := NewIssueParser(IssueParserConfig{
		StateManager: sm,
		BaseURL:      server.URL,
		Token:        "test-token",
		Logger:       logger,
	})
	if err != nil {
		t.Fatalf("failed to create IssueParser: %v", err)
	}

	ctx := context.Background()
	_, err = ip.FetchChanges(ctx)

	// Should return an error
	if err == nil {
		t.Error("expected error due to rate limit, got nil")
	}

	// Verify error message contains status code
	if !strings.Contains(err.Error(), "429") {
		t.Errorf("expected error message to contain status code 429, got: %v", err)
	}

	// Verify error was logged
	logOutput := logBuffer.String()
	if !strings.Contains(logOutput, "error") {
		t.Error("expected error to be logged")
	}
}

// TestIntegration_ForbiddenHandling tests handling of GitHub API forbidden errors.
// Requirements: 1.5
func TestIntegration_ForbiddenHandling(t *testing.T) {
	t.Parallel()

	var logBuffer bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&logBuffer, &slog.HandlerOptions{Level: slog.LevelInfo}))

	// Create mock server that returns 403 forbidden error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"message": "Resource not accessible by integration"}`))
	}))
	defer server.Close()

	tmpDir := t.TempDir()
	statePath := filepath.Join(tmpDir, "state.json")
	sm := NewStateManager(statePath)

	ip, err := NewIssueParser(IssueParserConfig{
		StateManager: sm,
		BaseURL:      server.URL,
		Token:        "test-token",
		Logger:       logger,
	})
	if err != nil {
		t.Fatalf("failed to create IssueParser: %v", err)
	}

	ctx := context.Background()
	_, err = ip.FetchChanges(ctx)

	// Should return an error
	if err == nil {
		t.Error("expected error due to forbidden access, got nil")
	}

	// Verify error message contains status code
	if !strings.Contains(err.Error(), "403") {
		t.Errorf("expected error message to contain status code 403, got: %v", err)
	}

	// Verify error was logged
	logOutput := logBuffer.String()
	if !strings.Contains(logOutput, "error") {
		t.Error("expected error to be logged")
	}
}

// TestIntegration_ServiceUnavailableHandling tests handling of GitHub API service unavailable errors.
// Requirements: 1.5
func TestIntegration_ServiceUnavailableHandling(t *testing.T) {
	t.Parallel()

	var logBuffer bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&logBuffer, &slog.HandlerOptions{Level: slog.LevelInfo}))

	// Create mock server that returns 503 service unavailable error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = w.Write([]byte(`{"message": "Service temporarily unavailable"}`))
	}))
	defer server.Close()

	tmpDir := t.TempDir()
	statePath := filepath.Join(tmpDir, "state.json")
	sm := NewStateManager(statePath)

	ip, err := NewIssueParser(IssueParserConfig{
		StateManager: sm,
		BaseURL:      server.URL,
		Token:        "test-token",
		Logger:       logger,
	})
	if err != nil {
		t.Fatalf("failed to create IssueParser: %v", err)
	}

	ctx := context.Background()
	_, err = ip.FetchChanges(ctx)

	// Should return an error
	if err == nil {
		t.Error("expected error due to service unavailable, got nil")
	}

	// Verify error message contains status code
	if !strings.Contains(err.Error(), "503") {
		t.Errorf("expected error message to contain status code 503, got: %v", err)
	}

	// Verify error was logged
	logOutput := logBuffer.String()
	if !strings.Contains(logOutput, "error") {
		t.Error("expected error to be logged")
	}
}

// TestIntegration_MultiPageFlow tests fetching comments across multiple pages.
// Requirements: 1.1
func TestIntegration_MultiPageFlow(t *testing.T) {
	t.Parallel()

	now := time.Now().Truncate(time.Second)
	var paginationRequests atomic.Int32

	// Create mock server that returns paginated results
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var comments []map[string]any

		// Check if this is a pagination request (has page parameter)
		pageParam := r.URL.Query().Get("page")
		if pageParam == "" || pageParam == "1" {
			// First page request or fetchPreviousComment request (no page param for fetchPreviousComment)
			if pageParam == "1" {
				paginationRequests.Add(1)
			}
			// First page: 100 comments (triggers pagination)
			for i := range 100 {
				comments = append(comments, map[string]any{
					"id":         int64(1000 + i),
					"body":       "Regular comment without minutes format",
					"created_at": now.Format(time.RFC3339),
					"updated_at": now.Format(time.RFC3339),
					"html_url":   "https://github.com/golang/go/issues/33502#issuecomment-" + string(rune(1000+i)),
				})
			}
		} else {
			paginationRequests.Add(1)
			// Second page: 1 comment with minutes format
			comments = append(comments, map[string]any{
				"id":         int64(2001),
				"body":       "**2026-01-30** / **@rsc**\n\n- [#99999](https://github.com/golang/go/issues/99999) **proposal: paginated feature**\n  - **accepted**\n",
				"created_at": now.Format(time.RFC3339),
				"updated_at": now.Format(time.RFC3339),
				"html_url":   "https://github.com/golang/go/issues/33502#issuecomment-2001",
			})
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(comments)
	}))
	defer server.Close()

	tmpDir := t.TempDir()
	statePath := filepath.Join(tmpDir, "state.json")

	// Create existing state file so it doesn't trigger fresh state (latest-only) mode
	oneHourAgo := now.Add(-1 * time.Hour)
	stateContent := fmt.Sprintf(`{"lastProcessedAt":"%s","lastCommentId":"999"}`, oneHourAgo.Format(time.RFC3339))
	if err := os.WriteFile(statePath, []byte(stateContent), 0644); err != nil {
		t.Fatalf("failed to write state file: %v", err)
	}

	sm := NewStateManager(statePath)

	ip, err := NewIssueParser(IssueParserConfig{
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

	// Should have made 2 pagination requests (page 1 and page 2)
	// Note: fetchPreviousComment also makes a request but we only count pagination requests
	if paginationRequests.Load() != 2 {
		t.Errorf("expected 2 pagination requests, got %d", paginationRequests.Load())
	}

	// Should have found 1 change from page 2
	if len(changes) != 1 {
		t.Errorf("expected 1 change, got %d", len(changes))
	}

	if len(changes) > 0 && changes[0].IssueNumber != 99999 {
		t.Errorf("expected issue number 99999, got %d", changes[0].IssueNumber)
	}
}

// TestIntegration_StatePreservation tests that state is correctly preserved across runs.
// Requirements: 1.1
func TestIntegration_StatePreservation(t *testing.T) {
	t.Parallel()

	now := time.Now().Truncate(time.Second)
	var requestCount atomic.Int32

	// Create mock server that returns different comments based on request
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count := requestCount.Add(1)

		var comments []map[string]any

		if count == 1 {
			// First run: return first comment
			comments = append(comments, map[string]any{
				"id":         int64(1001),
				"body":       "**2026-01-30** / **@rsc**\n\n- [#11111](https://github.com/golang/go/issues/11111) **proposal: first run**\n  - **accepted**\n",
				"created_at": now.Format(time.RFC3339),
				"updated_at": now.Format(time.RFC3339),
				"html_url":   "https://github.com/golang/go/issues/33502#issuecomment-1001",
			})
		} else {
			// Second run: return both comments (simulating since parameter)
			comments = append(comments, map[string]any{
				"id":         int64(1001),
				"body":       "**2026-01-30** / **@rsc**\n\n- [#11111](https://github.com/golang/go/issues/11111) **proposal: first run**\n  - **accepted**\n",
				"created_at": now.Format(time.RFC3339),
				"updated_at": now.Format(time.RFC3339),
				"html_url":   "https://github.com/golang/go/issues/33502#issuecomment-1001",
			})
			comments = append(comments, map[string]any{
				"id":         int64(1002),
				"body":       "**2026-01-31** / **@rsc**\n\n- [#22222](https://github.com/golang/go/issues/22222) **proposal: second run**\n  - **declined**\n",
				"created_at": now.Add(time.Hour).Format(time.RFC3339),
				"updated_at": now.Add(time.Hour).Format(time.RFC3339),
				"html_url":   "https://github.com/golang/go/issues/33502#issuecomment-1002",
			})
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(comments)
	}))
	defer server.Close()

	tmpDir := t.TempDir()
	statePath := filepath.Join(tmpDir, "state.json")
	sm := NewStateManager(statePath)

	ip, err := NewIssueParser(IssueParserConfig{
		StateManager: sm,
		BaseURL:      server.URL,
		Token:        "test-token",
	})
	if err != nil {
		t.Fatalf("failed to create IssueParser: %v", err)
	}

	ctx := context.Background()

	// First run
	changes1, err := ip.FetchChanges(ctx)
	if err != nil {
		t.Fatalf("First FetchChanges() error = %v", err)
	}

	if len(changes1) != 1 {
		t.Errorf("expected 1 change in first run, got %d", len(changes1))
	}

	// Second run - should only return new comments
	changes2, err := ip.FetchChanges(ctx)
	if err != nil {
		t.Fatalf("Second FetchChanges() error = %v", err)
	}

	// Only the new comment should be returned (1002), not the old one (1001)
	if len(changes2) != 1 {
		t.Errorf("expected 1 change in second run (only new comment), got %d", len(changes2))
	}

	if len(changes2) > 0 && changes2[0].IssueNumber != 22222 {
		t.Errorf("expected issue number 22222 in second run, got %d", changes2[0].IssueNumber)
	}

	// Verify state was updated correctly
	state, err := sm.LoadState()
	if err != nil {
		t.Fatalf("failed to load state: %v", err)
	}
	if state.LastCommentID != "1002" {
		t.Errorf("expected LastCommentID to be '1002' after second run, got %q", state.LastCommentID)
	}
}

// TestIntegration_EndToEndWithOutput tests the complete end-to-end flow from API to file output.
// Requirements: 1.1, 1.5
func TestIntegration_EndToEndWithOutput(t *testing.T) {
	t.Parallel()

	now := time.Now().Truncate(time.Second)

	// Create mock server with realistic GitHub API response
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		comments := []map[string]any{
			{
				"id": int64(98765432),
				"body": `**2026-01-30** / **@rsc**

- [#50000](https://github.com/golang/go/issues/50000) **proposal: add range over integers**
  - **accepted**

- [#60000](https://github.com/golang/go/issues/60000) **proposal: structured logging**
  - **likely accept**

- [#70000](https://github.com/golang/go/issues/70000) **proposal: deprecate old API**
  - **declined**
`,
				"created_at": now.Format(time.RFC3339),
				"updated_at": now.Format(time.RFC3339),
				"html_url":   "https://github.com/golang/go/issues/33502#issuecomment-98765432",
			},
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(comments)
	}))
	defer server.Close()

	tmpDir := t.TempDir()
	statePath := filepath.Join(tmpDir, "state.json")
	outputPath := filepath.Join(tmpDir, "changes.json")

	sm := NewStateManager(statePath)
	ip, err := NewIssueParser(IssueParserConfig{
		StateManager: sm,
		BaseURL:      server.URL,
		Token:        "test-token",
	})
	if err != nil {
		t.Fatalf("failed to create IssueParser: %v", err)
	}

	ctx := context.Background()

	// Fetch changes
	changes, err := ip.FetchChanges(ctx)
	if err != nil {
		t.Fatalf("FetchChanges() error = %v", err)
	}

	// Write to output file
	if err := ip.WriteChangesJSON(changes, outputPath); err != nil {
		t.Fatalf("WriteChangesJSON() error = %v", err)
	}

	// Verify output file
	data, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("failed to read output file: %v", err)
	}

	var output ChangesOutput
	if err := json.Unmarshal(data, &output); err != nil {
		t.Fatalf("failed to unmarshal output: %v", err)
	}

	// Verify all changes are in output
	if len(output.Changes) != 3 {
		t.Errorf("expected 3 changes in output, got %d", len(output.Changes))
	}

	// Verify changes are sorted by ChangedAt
	for i := 1; i < len(output.Changes); i++ {
		if output.Changes[i].ChangedAt.Before(output.Changes[i-1].ChangedAt) {
			t.Error("changes should be sorted by ChangedAt")
		}
	}

	// Verify status types
	statusCounts := make(map[Status]int)
	for _, c := range output.Changes {
		statusCounts[c.CurrentStatus]++
	}

	if statusCounts[StatusAccepted] != 1 {
		t.Errorf("expected 1 accepted status, got %d", statusCounts[StatusAccepted])
	}
	if statusCounts[StatusLikelyAccept] != 1 {
		t.Errorf("expected 1 likely accept status, got %d", statusCounts[StatusLikelyAccept])
	}
	if statusCounts[StatusDeclined] != 1 {
		t.Errorf("expected 1 declined status, got %d", statusCounts[StatusDeclined])
	}

	// Verify week format in output
	if output.Week == "" {
		t.Error("expected week to be set in output")
	}

	// Verify state file was created and updated
	if _, err := os.Stat(statePath); os.IsNotExist(err) {
		t.Error("state file should be created")
	}
}

// TestIntegration_ErrorLogging verifies that errors are properly logged.
// Requirements: 1.5
func TestIntegration_ErrorLogging(t *testing.T) {
	t.Parallel()

	var logBuffer bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&logBuffer, &slog.HandlerOptions{Level: slog.LevelDebug}))

	// Create mock server that returns an error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"message": "Internal server error"}`))
	}))
	defer server.Close()

	tmpDir := t.TempDir()
	statePath := filepath.Join(tmpDir, "state.json")
	sm := NewStateManager(statePath)

	ip, err := NewIssueParser(IssueParserConfig{
		StateManager: sm,
		BaseURL:      server.URL,
		Token:        "test-token",
		Logger:       logger,
	})
	if err != nil {
		t.Fatalf("failed to create IssueParser: %v", err)
	}

	ctx := context.Background()
	_, _ = ip.FetchChanges(ctx)

	// Verify error was logged (fresh state uses "failed to fetch latest comment")
	logOutput := logBuffer.String()
	if !strings.Contains(logOutput, "error") || !strings.Contains(logOutput, "failed to fetch latest comment") {
		t.Errorf("expected error log with 'failed to fetch latest comment', got: %s", logOutput)
	}
}

// TestIntegration_NoChangesFlow tests the flow when no new changes are detected.
// Requirements: 1.1
func TestIntegration_NoChangesFlow(t *testing.T) {
	t.Parallel()

	// Create mock server that returns empty comments
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode([]map[string]any{})
	}))
	defer server.Close()

	tmpDir := t.TempDir()
	statePath := filepath.Join(tmpDir, "state.json")
	sm := NewStateManager(statePath)

	ip, err := NewIssueParser(IssueParserConfig{
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

	// Should return empty changes without error
	if len(changes) != 0 {
		t.Errorf("expected 0 changes, got %d", len(changes))
	}

	// Write empty changes to file
	outputPath := filepath.Join(tmpDir, "changes.json")
	if err := ip.WriteChangesJSON(changes, outputPath); err != nil {
		t.Fatalf("WriteChangesJSON() error = %v", err)
	}

	// Verify output file is valid
	data, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("failed to read output file: %v", err)
	}

	var output ChangesOutput
	if err := json.Unmarshal(data, &output); err != nil {
		t.Fatalf("failed to unmarshal output: %v", err)
	}

	if len(output.Changes) != 0 {
		t.Errorf("expected 0 changes in output, got %d", len(output.Changes))
	}
}
