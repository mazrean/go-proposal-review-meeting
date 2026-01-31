package parser_test

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"testing"
	"time"

	"github.com/mazrean/go-proposal-review-meeting/internal/parser"
)

type goldenEntry struct {
	CommentedAt string                  `json:"commented_at"`
	Changes     []parser.ProposalChange `json:"changes"`
	CommentID   int64                   `json:"comment_id"`
}

func TestMinutesParser_Golden(t *testing.T) {
	t.Parallel()

	// Load golden file
	goldenPath := filepath.Join("testdata", "golden.json")
	goldenData, err := os.ReadFile(goldenPath)
	if err != nil {
		t.Fatalf("failed to read golden file: %v", err)
	}

	var entries []goldenEntry
	if err := json.Unmarshal(goldenData, &entries); err != nil {
		t.Fatalf("failed to parse golden file: %v", err)
	}

	// Load comment files directory
	commentsDir := filepath.Join("testdata", "comments")

	// Run tests in parallel for each entry
	for _, entry := range entries {
		t.Run(fmt.Sprintf("comment_%d", entry.CommentID), func(t *testing.T) {
			t.Parallel()

			// Load comment body
			commentPath := filepath.Join(commentsDir, fmt.Sprintf("%d.txt", entry.CommentID))
			commentBody, err := os.ReadFile(commentPath)
			if err != nil {
				t.Fatalf("failed to read comment file: %v", err)
			}

			// Parse comment timestamp
			commentedAt, err := time.Parse(time.RFC3339, entry.CommentedAt)
			if err != nil {
				t.Fatalf("failed to parse comment timestamp: %v", err)
			}

			// Parse the comment
			p := parser.NewMinutesParser()
			got, err := p.Parse(string(commentBody), commentedAt)
			if err != nil {
				t.Fatalf("Parse() error: %v", err)
			}

			// Sort both for comparison
			sortChanges(got)
			want := make([]parser.ProposalChange, len(entry.Changes))
			copy(want, entry.Changes)
			sortChanges(want)

			// Compare results
			if len(got) != len(want) {
				t.Errorf("got %d changes, want %d", len(got), len(want))
				return
			}

			for i := range want {
				if got[i].IssueNumber != want[i].IssueNumber {
					t.Errorf("change[%d].IssueNumber = %d, want %d", i, got[i].IssueNumber, want[i].IssueNumber)
				}
				if got[i].Title != want[i].Title {
					t.Errorf("change[%d].Title = %q, want %q", i, got[i].Title, want[i].Title)
				}
				if got[i].CurrentStatus != want[i].CurrentStatus {
					t.Errorf("change[%d].CurrentStatus = %s, want %s", i, got[i].CurrentStatus, want[i].CurrentStatus)
				}
				if !got[i].ChangedAt.Equal(want[i].ChangedAt) {
					t.Errorf("change[%d].ChangedAt = %v, want %v", i, got[i].ChangedAt, want[i].ChangedAt)
				}
			}
		})
	}
}

func sortChanges(changes []parser.ProposalChange) {
	sort.Slice(changes, func(i, j int) bool {
		return changes[i].IssueNumber < changes[j].IssueNumber
	})
}
