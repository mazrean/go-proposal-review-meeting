package parser_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/mazrean/go-proposal-review-meeting/internal/parser"
)

func TestStateManager_LoadState(t *testing.T) {
	t.Parallel()

	tests := []struct {
		setupFile        func(t *testing.T, path string)
		validateFunc     func(t *testing.T, state *parser.State)
		name             string
		wantErr          bool
		wantNilState     bool
		requirePOSIXPerm bool
	}{
		{
			name:      "file not exists - returns default state",
			setupFile: nil,
			wantErr:   false,
			validateFunc: func(t *testing.T, state *parser.State) {
				t.Helper()
				now := time.Now()
				oneMonthAgo := now.AddDate(0, -1, 0)

				// 1日程度の誤差は許容
				if state.LastProcessedAt.Before(oneMonthAgo.Add(-24*time.Hour)) ||
					state.LastProcessedAt.After(oneMonthAgo.Add(24*time.Hour)) {
					t.Errorf("LastProcessedAt should be around 1 month ago, got %v", state.LastProcessedAt)
				}

				if state.LastCommentID != "" {
					t.Errorf("LastCommentID should be empty for initial state, got %q", state.LastCommentID)
				}
			},
		},
		{
			name: "valid json file - loads state correctly",
			setupFile: func(t *testing.T, path string) {
				t.Helper()
				content := `{"lastProcessedAt":"2026-01-30T12:00:00Z","lastCommentId":"issuecomment-1234567890"}`
				if err := os.WriteFile(path, []byte(content), 0644); err != nil {
					t.Fatalf("failed to write test file: %v", err)
				}
			},
			wantErr: false,
			validateFunc: func(t *testing.T, state *parser.State) {
				t.Helper()
				expectedTime := time.Date(2026, 1, 30, 12, 0, 0, 0, time.UTC)
				if !state.LastProcessedAt.Equal(expectedTime) {
					t.Errorf("LastProcessedAt = %v, want %v", state.LastProcessedAt, expectedTime)
				}
				if state.LastCommentID != "issuecomment-1234567890" {
					t.Errorf("LastCommentID = %q, want %q", state.LastCommentID, "issuecomment-1234567890")
				}
			},
		},
		{
			name: "invalid json - returns error and nil state",
			setupFile: func(t *testing.T, path string) {
				t.Helper()
				if err := os.WriteFile(path, []byte("invalid json"), 0644); err != nil {
					t.Fatalf("failed to write test file: %v", err)
				}
			},
			wantErr:      true,
			wantNilState: true,
			validateFunc: nil,
		},
		{
			name: "empty json object - loads with zero values",
			setupFile: func(t *testing.T, path string) {
				t.Helper()
				if err := os.WriteFile(path, []byte("{}"), 0644); err != nil {
					t.Fatalf("failed to write test file: %v", err)
				}
			},
			wantErr: false,
			validateFunc: func(t *testing.T, state *parser.State) {
				t.Helper()
				if !state.LastProcessedAt.IsZero() {
					t.Errorf("LastProcessedAt should be zero, got %v", state.LastProcessedAt)
				}
				if state.LastCommentID != "" {
					t.Errorf("LastCommentID should be empty, got %q", state.LastCommentID)
				}
			},
		},
		{
			name: "unreadable file - returns error",
			setupFile: func(t *testing.T, path string) {
				t.Helper()
				content := `{"lastProcessedAt":"2026-01-30T12:00:00Z","lastCommentId":"test"}`
				if err := os.WriteFile(path, []byte(content), 0000); err != nil {
					t.Fatalf("failed to write test file: %v", err)
				}
			},
			wantErr:          true,
			wantNilState:     true,
			requirePOSIXPerm: true,
			validateFunc:     nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if tt.requirePOSIXPerm {
				skipIfNoPermissionEnforcement(t)
			}

			dir := t.TempDir()
			path := filepath.Join(dir, "state.json")

			if tt.setupFile != nil {
				tt.setupFile(t, path)
			}

			sm := parser.NewStateManager(path)
			state, err := sm.LoadState()

			if (err != nil) != tt.wantErr {
				t.Fatalf("LoadState() error = %v, wantErr %v", err, tt.wantErr)
			}

			if tt.wantNilState && state != nil {
				t.Errorf("LoadState() state should be nil on error, got %+v", state)
			}

			if tt.validateFunc != nil && state != nil {
				tt.validateFunc(t, state)
			}
		})
	}
}

func TestStateManager_SaveState(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name             string
		setupDir         func(t *testing.T, dir string) string
		lastProcessedAt  time.Time
		lastCommentID    string
		wantErr          bool
		requirePOSIXPerm bool
	}{
		{
			name: "save to valid path",
			setupDir: func(t *testing.T, dir string) string {
				t.Helper()
				return filepath.Join(dir, "state.json")
			},
			lastProcessedAt: time.Date(2026, 1, 30, 12, 0, 0, 0, time.UTC),
			lastCommentID:   "issuecomment-1234567890",
			wantErr:         false,
		},
		{
			name: "save with empty comment id",
			setupDir: func(t *testing.T, dir string) string {
				t.Helper()
				return filepath.Join(dir, "state.json")
			},
			lastProcessedAt: time.Date(2026, 1, 15, 8, 30, 0, 0, time.UTC),
			lastCommentID:   "",
			wantErr:         false,
		},
		{
			name: "save to non-existent directory - returns error",
			setupDir: func(t *testing.T, dir string) string {
				t.Helper()
				return filepath.Join(dir, "nonexistent", "state.json")
			},
			lastProcessedAt: time.Date(2026, 1, 30, 12, 0, 0, 0, time.UTC),
			lastCommentID:   "test",
			wantErr:         true,
		},
		{
			name: "save to read-only directory - returns error",
			setupDir: func(t *testing.T, dir string) string {
				t.Helper()
				roDir := filepath.Join(dir, "readonly")
				if err := os.Mkdir(roDir, 0555); err != nil {
					t.Fatalf("failed to create read-only dir: %v", err)
				}
				return filepath.Join(roDir, "state.json")
			},
			lastProcessedAt:  time.Date(2026, 1, 30, 12, 0, 0, 0, time.UTC),
			lastCommentID:    "test",
			wantErr:          true,
			requirePOSIXPerm: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if tt.requirePOSIXPerm {
				skipIfNoPermissionEnforcement(t)
			}

			dir := t.TempDir()
			path := tt.setupDir(t, dir)

			sm := parser.NewStateManager(path)
			state := &parser.State{
				LastProcessedAt: tt.lastProcessedAt,
				LastCommentID:   tt.lastCommentID,
			}

			err := sm.SaveState(state)

			if (err != nil) != tt.wantErr {
				t.Fatalf("SaveState() error = %v, wantErr %v", err, tt.wantErr)
			}

			if !tt.wantErr {
				// Verify file exists and content is correct
				if _, err := os.Stat(path); os.IsNotExist(err) {
					t.Fatal("state.json should exist after SaveState")
				}

				// Reload and verify
				sm2 := parser.NewStateManager(path)
				loaded, err := sm2.LoadState()
				if err != nil {
					t.Fatalf("LoadState() error = %v", err)
				}

				if !loaded.LastProcessedAt.Equal(tt.lastProcessedAt) {
					t.Errorf("LastProcessedAt = %v, want %v", loaded.LastProcessedAt, tt.lastProcessedAt)
				}

				if loaded.LastCommentID != tt.lastCommentID {
					t.Errorf("LastCommentID = %q, want %q", loaded.LastCommentID, tt.lastCommentID)
				}
			}
		})
	}
}

func TestStateManager_UpdateState(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		setupDir    func(t *testing.T, dir string) string
		initialTime time.Time
		initialID   string
		updateTime  time.Time
		updateID    string
		wantErr     bool
	}{
		{
			name: "update from initial state",
			setupDir: func(t *testing.T, dir string) string {
				t.Helper()
				return filepath.Join(dir, "state.json")
			},
			initialTime: time.Time{},
			initialID:   "",
			updateTime:  time.Date(2026, 1, 31, 15, 30, 0, 0, time.UTC),
			updateID:    "issuecomment-9876543210",
			wantErr:     false,
		},
		{
			name: "update existing state",
			setupDir: func(t *testing.T, dir string) string {
				t.Helper()
				return filepath.Join(dir, "state.json")
			},
			initialTime: time.Date(2026, 1, 30, 12, 0, 0, 0, time.UTC),
			initialID:   "issuecomment-1111111111",
			updateTime:  time.Date(2026, 1, 31, 18, 0, 0, 0, time.UTC),
			updateID:    "issuecomment-2222222222",
			wantErr:     false,
		},
		{
			name: "update to non-existent directory - returns error",
			setupDir: func(t *testing.T, dir string) string {
				t.Helper()
				return filepath.Join(dir, "nonexistent", "state.json")
			},
			initialTime: time.Time{},
			initialID:   "",
			updateTime:  time.Date(2026, 1, 31, 15, 30, 0, 0, time.UTC),
			updateID:    "test",
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			dir := t.TempDir()
			path := tt.setupDir(t, dir)

			sm := parser.NewStateManager(path)

			// Set up initial state if provided
			if !tt.initialTime.IsZero() {
				initial := &parser.State{
					LastProcessedAt: tt.initialTime,
					LastCommentID:   tt.initialID,
				}
				if err := sm.SaveState(initial); err != nil {
					t.Fatalf("SaveState() error = %v", err)
				}
			}

			// Update state
			err := sm.UpdateState(tt.updateTime, tt.updateID)

			if (err != nil) != tt.wantErr {
				t.Fatalf("UpdateState() error = %v, wantErr %v", err, tt.wantErr)
			}

			if !tt.wantErr {
				// Verify with new manager
				sm2 := parser.NewStateManager(path)
				loaded, err := sm2.LoadState()
				if err != nil {
					t.Fatalf("LoadState() error = %v", err)
				}

				if !loaded.LastProcessedAt.Equal(tt.updateTime) {
					t.Errorf("LastProcessedAt = %v, want %v", loaded.LastProcessedAt, tt.updateTime)
				}

				if loaded.LastCommentID != tt.updateID {
					t.Errorf("LastCommentID = %q, want %q", loaded.LastCommentID, tt.updateID)
				}
			}
		})
	}
}
