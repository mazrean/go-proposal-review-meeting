package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/mazrean/go-proposal-review-meeting/internal/parser"
)

func TestRun(t *testing.T) {
	t.Parallel()

	now := time.Now().Truncate(time.Second)

	tests := []struct {
		name              string
		comments          []map[string]any
		apiError          bool
		wantErr           bool
		wantHasChanges    bool
		wantChangesCount  int
	}{
		{
			name: "正常系: 変更あり",
			comments: []map[string]any{
				{
					"id":         int64(12345),
					"body":       "**2026-01-30** / **@rsc**\n\n- [#12345](https://github.com/golang/go/issues/12345) **proposal: add new feature**\n  - **accepted**\n",
					"created_at": now.Format(time.RFC3339),
					"updated_at": now.Format(time.RFC3339),
					"html_url":   "https://github.com/golang/go/issues/33502#issuecomment-12345",
				},
			},
			wantErr:          false,
			wantHasChanges:   true,
			wantChangesCount: 1,
		},
		{
			name:             "正常系: 変更なし",
			comments:         []map[string]any{},
			wantErr:          false,
			wantHasChanges:   false,
			wantChangesCount: 0,
		},
		{
			name: "正常系: 複数の変更",
			comments: []map[string]any{
				{
					"id":         int64(11111),
					"body":       "**2026-01-30** / **@rsc**\n\n- #11111 **proposal: feature A**\n  - **accepted**\n\n- #22222 **proposal: feature B**\n  - **declined**\n",
					"created_at": now.Format(time.RFC3339),
					"updated_at": now.Format(time.RFC3339),
					"html_url":   "https://github.com/golang/go/issues/33502#issuecomment-11111",
				},
			},
			wantErr:          false,
			wantHasChanges:   true,
			wantChangesCount: 2,
		},
		{
			name:     "異常系: APIエラー",
			apiError: true,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Setup mock server
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if tt.apiError {
					w.WriteHeader(http.StatusInternalServerError)
					return
				}
				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(tt.comments)
			}))
			defer server.Close()

			// Setup temp directory
			tmpDir := t.TempDir()
			statePath := filepath.Join(tmpDir, "state.json")
			changesPath := filepath.Join(tmpDir, "changes.json")

			// Capture stdout
			var stdout bytes.Buffer

			// Run the parse function
			config := parseConfig{
				statePath:   statePath,
				changesPath: changesPath,
				baseURL:     server.URL,
				token:       "test-token",
				stdout:      &stdout,
			}

			err := runParse(context.Background(), config)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			// Check stdout output for has_changes
			output := stdout.String()
			if tt.wantHasChanges {
				if !strings.Contains(output, "has_changes=true") {
					t.Errorf("expected has_changes=true in output, got: %s", output)
				}
			} else {
				if !strings.Contains(output, "has_changes=false") {
					t.Errorf("expected has_changes=false in output, got: %s", output)
				}
			}

			// Check changes.json content
			data, err := os.ReadFile(changesPath)
			if err != nil {
				t.Fatalf("failed to read changes.json: %v", err)
			}

			var changesOutput parser.ChangesOutput
			if err := json.Unmarshal(data, &changesOutput); err != nil {
				t.Fatalf("failed to unmarshal changes.json: %v", err)
			}

			if len(changesOutput.Changes) != tt.wantChangesCount {
				t.Errorf("expected %d changes, got %d", tt.wantChangesCount, len(changesOutput.Changes))
			}
		})
	}
}

func TestRunParse_StateFileUpdate(t *testing.T) {
	t.Parallel()

	now := time.Now().Truncate(time.Second)

	// Setup mock server with one comment
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		comments := []map[string]any{
			{
				"id":         int64(99999),
				"body":       "**2026-01-30** / **@rsc**\n\n- #99999 **proposal: test**\n  - **accepted**\n",
				"created_at": now.Format(time.RFC3339),
				"updated_at": now.Format(time.RFC3339),
				"html_url":   "https://github.com/golang/go/issues/33502#issuecomment-99999",
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(comments)
	}))
	defer server.Close()

	tmpDir := t.TempDir()
	statePath := filepath.Join(tmpDir, "state.json")
	changesPath := filepath.Join(tmpDir, "changes.json")

	var stdout bytes.Buffer
	config := parseConfig{
		statePath:   statePath,
		changesPath: changesPath,
		baseURL:     server.URL,
		token:       "test-token",
		stdout:      &stdout,
	}

	// First run
	if err := runParse(context.Background(), config); err != nil {
		t.Fatalf("first run failed: %v", err)
	}

	// Verify state file was created and updated
	data, err := os.ReadFile(statePath)
	if err != nil {
		t.Fatalf("failed to read state.json: %v", err)
	}

	var state parser.State
	if err := json.Unmarshal(data, &state); err != nil {
		t.Fatalf("failed to unmarshal state.json: %v", err)
	}

	if state.LastCommentID != "99999" {
		t.Errorf("expected lastCommentId to be 99999, got %s", state.LastCommentID)
	}
}

func TestRunParse_OutputFormat(t *testing.T) {
	t.Parallel()

	now := time.Now().Truncate(time.Second)

	// Setup mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		comments := []map[string]any{
			{
				"id":         int64(12345),
				"body":       "**2026-01-30** / **@rsc**\n\n- [#12345](https://github.com/golang/go/issues/12345) **proposal: test**\n  - **accepted**\n",
				"created_at": now.Format(time.RFC3339),
				"updated_at": now.Format(time.RFC3339),
				"html_url":   "https://github.com/golang/go/issues/33502#issuecomment-12345",
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(comments)
	}))
	defer server.Close()

	tmpDir := t.TempDir()
	statePath := filepath.Join(tmpDir, "state.json")
	changesPath := filepath.Join(tmpDir, "changes.json")

	var stdout bytes.Buffer
	config := parseConfig{
		statePath:   statePath,
		changesPath: changesPath,
		baseURL:     server.URL,
		token:       "test-token",
		stdout:      &stdout,
	}

	if err := runParse(context.Background(), config); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := stdout.String()
	lines := strings.Split(strings.TrimSpace(output), "\n")

	// Should have at least one line with has_changes
	found := false
	for _, line := range lines {
		if strings.HasPrefix(line, "has_changes=") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected has_changes= in output, got: %s", output)
	}
}
