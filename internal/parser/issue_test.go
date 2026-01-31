package parser_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/mazrean/go-proposal-review-meeting/internal/parser"
)

// mockComment represents a mock GitHub comment for testing.
type mockComment struct {
	CreatedAt time.Time
	UpdatedAt time.Time
	Body      string
	HTMLURL   string
	ID        int64
}

// serverConfig configures the mock server behavior.
type serverConfig struct {
	etag        string
	comments    []mockComment
	statusCode  int
	invalidJSON bool
}

// setupMockServer creates a test server with the given configuration.
func setupMockServer(t *testing.T, config serverConfig) *httptest.Server {
	t.Helper()

	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Handle ETag caching
		if config.etag != "" {
			if ifNoneMatch := r.Header.Get("If-None-Match"); ifNoneMatch == config.etag {
				w.WriteHeader(http.StatusNotModified)
				return
			}
			w.Header().Set("ETag", config.etag)
		}

		// Handle error responses
		if config.statusCode != 0 && config.statusCode != http.StatusOK {
			w.WriteHeader(config.statusCode)
			_, _ = w.Write([]byte("Error"))
			return
		}

		// Handle invalid JSON
		if config.invalidJSON {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte("invalid json{"))
			return
		}

		// Build response
		var comments []map[string]any
		for _, c := range config.comments {
			updatedAt := c.UpdatedAt
			if updatedAt.IsZero() {
				updatedAt = c.CreatedAt
			}
			comments = append(comments, map[string]any{
				"id":         c.ID,
				"body":       c.Body,
				"created_at": c.CreatedAt.Format(time.RFC3339),
				"updated_at": updatedAt.Format(time.RFC3339),
				"html_url":   c.HTMLURL,
			})
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(comments)
	}))
}

func TestNewIssueParser(t *testing.T) {
	t.Parallel()

	tests := []struct {
		wantErr error
		config  parser.IssueParserConfig
		name    string
	}{
		{
			name: "正常系: StateManagerあり",
			config: parser.IssueParserConfig{
				StateManager: parser.NewStateManager(filepath.Join(t.TempDir(), "state.json")),
			},
			wantErr: nil,
		},
		{
			name: "異常系: StateManagerなし",
			config: parser.IssueParserConfig{
				StateManager: nil,
			},
			wantErr: parser.ErrNilStateManager,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ip, err := parser.NewIssueParser(tt.config)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("expected error %v, got %v", tt.wantErr, err)
				}
				if ip != nil {
					t.Error("expected nil IssueParser on error")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if ip == nil {
					t.Error("expected non-nil IssueParser")
				}
			}
		})
	}
}

func TestIssueParser_FetchChanges(t *testing.T) {
	t.Parallel()

	now := time.Now().Truncate(time.Second)

	tests := []struct {
		initialState     *time.Time
		initialCommentID string
		name             string
		wantStatus       parser.Status
		serverConfig     serverConfig
		wantChanges      int
		wantIssueNum     int
		wantErr          bool
		wantStateUpdated bool
	}{
		{
			name: "正常系: 新規コメントから変更を抽出",
			serverConfig: serverConfig{
				comments: []mockComment{
					{
						ID:        12345,
						Body:      "**2026-01-30** / **@rsc**\n\n- [#12345](https://github.com/golang/go/issues/12345) **proposal: add new feature**\n  - **accepted**\n",
						CreatedAt: now,
						HTMLURL:   "https://github.com/golang/go/issues/33502#issuecomment-12345",
					},
				},
			},
			wantChanges:      1,
			wantErr:          false,
			wantStatus:       parser.StatusAccepted,
			wantIssueNum:     12345,
			wantStateUpdated: true,
		},
		{
			name: "正常系: 古いコメントはフィルタリングされる",
			serverConfig: serverConfig{
				comments: []mockComment{
					{
						ID:        99999,
						Body:      "**2026-01-20** / **@rsc**\n\n- #99999 **old proposal**\n  - **accepted**\n",
						CreatedAt: now.Add(-48 * time.Hour),
						HTMLURL:   "https://github.com/golang/go/issues/33502#issuecomment-99999",
					},
				},
			},
			initialState: func() *time.Time { t := now.Add(-1 * time.Hour); return &t }(),
			wantChanges:  0,
			wantErr:      false,
		},
		{
			name: "正常系: 同一タイムスタンプでもID大きい方は処理される",
			serverConfig: serverConfig{
				comments: []mockComment{
					{
						ID:        10000,
						Body:      "**2026-01-30** / **@rsc**\n\n- #10000 **old proposal**\n  - **accepted**\n",
						CreatedAt: now,
						HTMLURL:   "https://github.com/golang/go/issues/33502#issuecomment-10000",
					},
					{
						ID:        10001,
						Body:      "**2026-01-30** / **@rsc**\n\n- #10001 **new proposal**\n  - **declined**\n",
						CreatedAt: now,
						HTMLURL:   "https://github.com/golang/go/issues/33502#issuecomment-10001",
					},
				},
			},
			initialState:     &now,
			initialCommentID: "10000",
			wantChanges:      1, // Only 10001 should be processed
			wantErr:          false,
			wantStateUpdated: true,
		},
		{
			name: "正常系: 編集されたコメントは再処理される",
			serverConfig: serverConfig{
				comments: []mockComment{
					{
						ID:        20000,
						Body:      "**2026-01-30** / **@rsc**\n\n- #20000 **edited proposal**\n  - **accepted**\n",
						CreatedAt: now.Add(-24 * time.Hour), // Created yesterday
						UpdatedAt: now,                      // Updated now
						HTMLURL:   "https://github.com/golang/go/issues/33502#issuecomment-20000",
					},
				},
			},
			initialState:     func() *time.Time { t := now.Add(-1 * time.Hour); return &t }(),
			wantChanges:      1, // Should be processed because UpdatedAt > LastProcessedAt
			wantErr:          false,
			wantStateUpdated: true,
		},
		{
			name: "正常系: 複数のproposal変更を抽出",
			serverConfig: serverConfig{
				comments: []mockComment{
					{
						ID:        11111,
						Body:      "**2026-01-30** / **@rsc**\n\n- #11111 **proposal: feature A**\n  - **accepted**\n\n- #22222 **proposal: feature B**\n  - **declined**\n",
						CreatedAt: now,
						HTMLURL:   "https://github.com/golang/go/issues/33502#issuecomment-11111",
					},
				},
			},
			wantChanges:      2,
			wantErr:          false,
			wantStateUpdated: true,
		},
		{
			name: "正常系: コメントなし",
			serverConfig: serverConfig{
				comments: []mockComment{},
			},
			wantChanges: 0,
			wantErr:     false,
		},
		{
			name: "異常系: APIエラー",
			serverConfig: serverConfig{
				statusCode: http.StatusInternalServerError,
			},
			wantChanges: 0,
			wantErr:     true,
		},
		{
			name: "異常系: 認証エラー",
			serverConfig: serverConfig{
				statusCode: http.StatusUnauthorized,
			},
			wantChanges: 0,
			wantErr:     true,
		},
		{
			name: "異常系: JSONデコードエラー",
			serverConfig: serverConfig{
				invalidJSON: true,
			},
			wantChanges: 0,
			wantErr:     true,
		},
		{
			name: "正常系: Minutesフォーマットでないコメントはスキップ",
			serverConfig: serverConfig{
				comments: []mockComment{
					{
						ID:        33333,
						Body:      "This is just a regular comment without minutes format.",
						CreatedAt: now,
						HTMLURL:   "https://github.com/golang/go/issues/33502#issuecomment-33333",
					},
				},
			},
			wantChanges:      0,
			wantErr:          false,
			wantStateUpdated: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			server := setupMockServer(t, tt.serverConfig)
			defer server.Close()

			tmpDir := t.TempDir()
			statePath := filepath.Join(tmpDir, "state.json")
			sm := parser.NewStateManager(statePath)

			// Set initial state if specified
			if tt.initialState != nil {
				commentID := tt.initialCommentID
				if err := sm.UpdateState(*tt.initialState, commentID); err != nil {
					t.Fatalf("failed to set initial state: %v", err)
				}
			}

			ip, err := parser.NewIssueParser(parser.IssueParserConfig{
				StateManager: sm,
				BaseURL:      server.URL,
				Token:        "test-token",
			})
			if err != nil {
				t.Fatalf("failed to create IssueParser: %v", err)
			}

			changes, err := ip.FetchChanges(context.Background())

			// Check error
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			// Check changes count
			if len(changes) != tt.wantChanges {
				t.Errorf("expected %d changes, got %d", tt.wantChanges, len(changes))
			}

			// Check specific values if expected
			if tt.wantChanges > 0 && tt.wantStatus != "" {
				if changes[0].CurrentStatus != tt.wantStatus {
					t.Errorf("expected status %s, got %s", tt.wantStatus, changes[0].CurrentStatus)
				}
			}
			if tt.wantChanges > 0 && tt.wantIssueNum != 0 {
				if changes[0].IssueNumber != tt.wantIssueNum {
					t.Errorf("expected issue number %d, got %d", tt.wantIssueNum, changes[0].IssueNumber)
				}
			}

			// Check state update
			if tt.wantStateUpdated && len(tt.serverConfig.comments) > 0 {
				state, err := sm.LoadState()
				if err != nil {
					t.Fatalf("failed to load state: %v", err)
				}
				if state.LastCommentID == "" && !tt.wantErr {
					t.Error("expected state to be updated with comment ID")
				}
			}
		})
	}
}

func TestIssueParser_FetchChanges_ETagCaching(t *testing.T) {
	t.Parallel()

	callCount := 0
	etag := "\"abc123\""

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++

		if ifNoneMatch := r.Header.Get("If-None-Match"); ifNoneMatch == etag {
			w.WriteHeader(http.StatusNotModified)
			return
		}

		w.Header().Set("ETag", etag)
		w.Header().Set("Content-Type", "application/json")

		now := time.Now()
		comments := []map[string]any{
			{
				"id":         int64(11111),
				"body":       "**2026-01-30** / **@rsc**\n\n- #11111 **test proposal**\n  - **accepted**\n",
				"created_at": now.Format(time.RFC3339),
				"updated_at": now.Format(time.RFC3339),
				"html_url":   "https://github.com/golang/go/issues/33502#issuecomment-11111",
			},
		}
		_ = json.NewEncoder(w).Encode(comments)
	}))
	defer server.Close()

	tmpDir := t.TempDir()
	statePath := filepath.Join(tmpDir, "state.json")
	sm := parser.NewStateManager(statePath)

	ip, err := parser.NewIssueParser(parser.IssueParserConfig{
		StateManager: sm,
		BaseURL:      server.URL,
		Token:        "test-token",
	})
	if err != nil {
		t.Fatalf("failed to create IssueParser: %v", err)
	}

	tests := []struct {
		name        string
		wantChanges int
	}{
		{name: "初回リクエスト", wantChanges: 1},
		{name: "キャッシュヒット", wantChanges: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			changes, err := ip.FetchChanges(context.Background())
			if err != nil {
				t.Fatalf("FetchChanges failed: %v", err)
			}
			if len(changes) != tt.wantChanges {
				t.Errorf("expected %d changes, got %d", tt.wantChanges, len(changes))
			}
		})
	}

	if callCount != 2 {
		t.Errorf("expected 2 API calls, got %d", callCount)
	}
}

func TestIssueParser_WriteChangesJSON(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		wantWeek  string
		changes   []parser.ProposalChange
		wantCount int
		wantErr   bool
	}{
		{
			name: "正常系: 単一の変更を出力",
			changes: []parser.ProposalChange{
				{
					IssueNumber:   12345,
					Title:         "proposal: add new feature",
					CurrentStatus: parser.StatusAccepted,
					ChangedAt:     time.Date(2026, 1, 30, 12, 0, 0, 0, time.UTC),
					CommentURL:    "https://github.com/golang/go/issues/33502#issuecomment-12345",
				},
			},
			wantWeek:  "2026-W05",
			wantCount: 1,
			wantErr:   false,
		},
		{
			name: "正常系: 複数の変更を出力",
			changes: []parser.ProposalChange{
				{
					IssueNumber:   11111,
					Title:         "proposal: feature A",
					CurrentStatus: parser.StatusAccepted,
					ChangedAt:     time.Date(2026, 1, 30, 12, 0, 0, 0, time.UTC),
				},
				{
					IssueNumber:   22222,
					Title:         "proposal: feature B",
					CurrentStatus: parser.StatusDeclined,
					ChangedAt:     time.Date(2026, 1, 30, 12, 0, 0, 0, time.UTC),
				},
			},
			wantWeek:  "2026-W05",
			wantCount: 2,
			wantErr:   false,
		},
		{
			name:      "正常系: 空の変更リスト",
			changes:   []parser.ProposalChange{},
			wantCount: 0,
			wantErr:   false,
		},
		{
			name: "正常系: 異なる週の変更は最新週を使用",
			changes: []parser.ProposalChange{
				{
					IssueNumber:   11111,
					Title:         "proposal: older",
					CurrentStatus: parser.StatusAccepted,
					ChangedAt:     time.Date(2026, 1, 20, 12, 0, 0, 0, time.UTC), // W04
				},
				{
					IssueNumber:   22222,
					Title:         "proposal: newer",
					CurrentStatus: parser.StatusDeclined,
					ChangedAt:     time.Date(2026, 1, 30, 12, 0, 0, 0, time.UTC), // W05
				},
			},
			wantWeek:  "2026-W05", // Should use latest week
			wantCount: 2,
			wantErr:   false,
		},
		{
			name: "正常系: 出力は日付順にソートされる",
			changes: []parser.ProposalChange{
				{
					IssueNumber:   22222,
					Title:         "proposal: later",
					CurrentStatus: parser.StatusDeclined,
					ChangedAt:     time.Date(2026, 1, 30, 12, 0, 0, 0, time.UTC),
				},
				{
					IssueNumber:   11111,
					Title:         "proposal: earlier",
					CurrentStatus: parser.StatusAccepted,
					ChangedAt:     time.Date(2026, 1, 20, 12, 0, 0, 0, time.UTC),
				},
			},
			wantWeek:  "2026-W05",
			wantCount: 2,
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			tmpDir := t.TempDir()
			statePath := filepath.Join(tmpDir, "state.json")
			outputPath := filepath.Join(tmpDir, "changes.json")

			sm := parser.NewStateManager(statePath)
			ip, err := parser.NewIssueParser(parser.IssueParserConfig{
				StateManager: sm,
				BaseURL:      "https://api.github.com",
				Token:        "test-token",
			})
			if err != nil {
				t.Fatalf("failed to create IssueParser: %v", err)
			}

			err = ip.WriteChangesJSON(tt.changes, outputPath)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			// Read and verify output
			data, err := os.ReadFile(outputPath)
			if err != nil {
				t.Fatalf("failed to read output file: %v", err)
			}

			var output parser.ChangesOutput
			if err := json.Unmarshal(data, &output); err != nil {
				t.Fatalf("failed to unmarshal output: %v", err)
			}

			if len(output.Changes) != tt.wantCount {
				t.Errorf("expected %d changes in output, got %d", tt.wantCount, len(output.Changes))
			}

			if tt.wantWeek != "" && output.Week != tt.wantWeek {
				t.Errorf("expected week %s, got %s", tt.wantWeek, output.Week)
			}

			// Verify sorting for the "出力は日付順にソートされる" test
			if tt.name == "正常系: 出力は日付順にソートされる" && len(output.Changes) >= 2 {
				if output.Changes[0].IssueNumber != 11111 {
					t.Errorf("expected first change to be issue 11111 (earlier), got %d", output.Changes[0].IssueNumber)
				}
				if output.Changes[1].IssueNumber != 22222 {
					t.Errorf("expected second change to be issue 22222 (later), got %d", output.Changes[1].IssueNumber)
				}
			}
		})
	}
}

func TestIssueParser_WriteChangesJSON_WriteError(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	statePath := filepath.Join(tmpDir, "state.json")

	sm := parser.NewStateManager(statePath)
	ip, err := parser.NewIssueParser(parser.IssueParserConfig{
		StateManager: sm,
		BaseURL:      "https://api.github.com",
		Token:        "test-token",
	})
	if err != nil {
		t.Fatalf("failed to create IssueParser: %v", err)
	}

	// Try to write to a non-existent directory
	invalidPath := filepath.Join(tmpDir, "nonexistent", "changes.json")
	changes := []parser.ProposalChange{
		{
			IssueNumber:   12345,
			Title:         "proposal: test",
			CurrentStatus: parser.StatusAccepted,
			ChangedAt:     time.Now(),
		},
	}

	err = ip.WriteChangesJSON(changes, invalidPath)
	if err == nil {
		t.Error("expected error when writing to invalid path, got nil")
	}
}
