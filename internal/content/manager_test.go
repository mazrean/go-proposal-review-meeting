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

func TestManager_PrepareContent(t *testing.T) {
	t.Parallel()

	baseTime := time.Date(2026, 1, 30, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name     string
		changes  []parser.ProposalChange
		wantYear int
		wantWeek int
		wantLen  int
	}{
		{
			name:     "empty changes",
			changes:  []parser.ProposalChange{},
			wantYear: 0,
			wantWeek: 0,
			wantLen:  0,
		},
		{
			name: "single change",
			changes: []parser.ProposalChange{
				{
					IssueNumber:    12345,
					Title:          "proposal: add new feature",
					PreviousStatus: parser.StatusDiscussions,
					CurrentStatus:  parser.StatusAccepted,
					ChangedAt:      baseTime,
					CommentURL:     "https://github.com/golang/go/issues/33502#issuecomment-xxx",
					RelatedIssues:  []int{67890},
				},
			},
			wantYear: 2026,
			wantWeek: 5,
			wantLen:  1,
		},
		{
			name: "multiple changes",
			changes: []parser.ProposalChange{
				{
					IssueNumber:    12345,
					Title:          "proposal: feature one",
					PreviousStatus: parser.StatusDiscussions,
					CurrentStatus:  parser.StatusAccepted,
					ChangedAt:      baseTime,
					CommentURL:     "https://github.com/golang/go/issues/33502#issuecomment-1",
					RelatedIssues:  []int{11111},
				},
				{
					IssueNumber:    67890,
					Title:          "proposal: feature two",
					PreviousStatus: parser.StatusActive,
					CurrentStatus:  parser.StatusDeclined,
					ChangedAt:      baseTime.Add(time.Hour),
					CommentURL:     "https://github.com/golang/go/issues/33502#issuecomment-2",
					RelatedIssues:  nil,
				},
			},
			wantYear: 2026,
			wantWeek: 5,
			wantLen:  2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mgr := NewManager()
			content := mgr.PrepareContent(tt.changes)

			if content.Year != tt.wantYear {
				t.Errorf("Year = %d, want %d", content.Year, tt.wantYear)
			}
			if content.Week != tt.wantWeek {
				t.Errorf("Week = %d, want %d", content.Week, tt.wantWeek)
			}
			if len(content.Proposals) != tt.wantLen {
				t.Errorf("len(Proposals) = %d, want %d", len(content.Proposals), tt.wantLen)
			}

			// Verify proposal content mapping
			for i, change := range tt.changes {
				if i >= len(content.Proposals) {
					break
				}
				p := content.Proposals[i]
				if p.IssueNumber != change.IssueNumber {
					t.Errorf("Proposals[%d].IssueNumber = %d, want %d", i, p.IssueNumber, change.IssueNumber)
				}
				if p.Title != change.Title {
					t.Errorf("Proposals[%d].Title = %q, want %q", i, p.Title, change.Title)
				}
				if p.PreviousStatus != change.PreviousStatus {
					t.Errorf("Proposals[%d].PreviousStatus = %q, want %q", i, p.PreviousStatus, change.PreviousStatus)
				}
				if p.CurrentStatus != change.CurrentStatus {
					t.Errorf("Proposals[%d].CurrentStatus = %q, want %q", i, p.CurrentStatus, change.CurrentStatus)
				}
			}
		})
	}
}

func TestManager_WriteContent(t *testing.T) {
	t.Parallel()

	baseTime := time.Date(2026, 1, 30, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name    string
		content *WeeklyContent
		wantDir string
	}{
		{
			name: "single proposal",
			content: &WeeklyContent{
				Year: 2026,
				Week: 5,
				Proposals: []ProposalContent{
					{
						IssueNumber:    12345,
						Title:          "proposal: add new feature",
						PreviousStatus: parser.StatusDiscussions,
						CurrentStatus:  parser.StatusAccepted,
						ChangedAt:      baseTime,
						CommentURL:     "https://github.com/golang/go/issues/33502#issuecomment-xxx",
						Summary:        "",
						Links: []Link{
							{Title: "proposal issue", URL: "https://github.com/golang/go/issues/12345"},
						},
					},
				},
				CreatedAt: baseTime,
			},
			wantDir: "2026/W05",
		},
		{
			name: "week 52",
			content: &WeeklyContent{
				Year: 2025,
				Week: 52,
				Proposals: []ProposalContent{
					{
						IssueNumber:    99999,
						Title:          "proposal: year end feature",
						PreviousStatus: parser.StatusActive,
						CurrentStatus:  parser.StatusHold,
						ChangedAt:      time.Date(2025, 12, 25, 12, 0, 0, 0, time.UTC),
						CommentURL:     "https://github.com/golang/go/issues/33502#issuecomment-yyy",
						Summary:        "",
						Links:          nil,
					},
				},
				CreatedAt: time.Date(2025, 12, 25, 12, 0, 0, 0, time.UTC),
			},
			wantDir: "2025/W52",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Create temp directory
			tmpDir := t.TempDir()
			mgr := NewManager(WithBaseDir(tmpDir))

			err := mgr.WriteContent(tt.content)
			if err != nil {
				t.Fatalf("WriteContent() error = %v", err)
			}

			// Verify directory was created
			expectedDir := filepath.Join(tmpDir, tt.wantDir)
			if _, err := os.Stat(expectedDir); os.IsNotExist(err) {
				t.Errorf("Expected directory %s was not created", expectedDir)
			}

			// Verify files were created
			for _, p := range tt.content.Proposals {
				expectedFile := filepath.Join(expectedDir, proposalFilename(p.IssueNumber))
				if _, err := os.Stat(expectedFile); os.IsNotExist(err) {
					t.Errorf("Expected file %s was not created", expectedFile)
				}
			}
		})
	}
}

func TestManager_WriteContent_Frontmatter(t *testing.T) {
	t.Parallel()

	baseTime := time.Date(2026, 1, 30, 12, 0, 0, 0, time.UTC)

	content := &WeeklyContent{
		Year: 2026,
		Week: 5,
		Proposals: []ProposalContent{
			{
				IssueNumber:    12345,
				Title:          "proposal: add new feature",
				PreviousStatus: parser.StatusDiscussions,
				CurrentStatus:  parser.StatusAccepted,
				ChangedAt:      baseTime,
				CommentURL:     "https://github.com/golang/go/issues/33502#issuecomment-xxx",
				Summary:        "",
				Links: []Link{
					{Title: "proposal issue", URL: "https://github.com/golang/go/issues/12345"},
					{Title: "related discussion", URL: "https://github.com/golang/go/issues/67890"},
				},
			},
		},
		CreatedAt: baseTime,
	}

	tmpDir := t.TempDir()
	mgr := NewManager(WithBaseDir(tmpDir))

	err := mgr.WriteContent(content)
	if err != nil {
		t.Fatalf("WriteContent() error = %v", err)
	}

	// Read the generated file
	expectedFile := filepath.Join(tmpDir, "2026/W05", proposalFilename(12345))
	data, err := os.ReadFile(expectedFile)
	if err != nil {
		t.Fatalf("Failed to read generated file: %v", err)
	}

	fileContent := string(data)

	// Verify frontmatter structure
	if !strings.HasPrefix(fileContent, "---\n") {
		t.Error("File should start with frontmatter delimiter")
	}

	// Verify required frontmatter fields
	expectedFields := []string{
		"issue_number: 12345",
		"title: \"proposal: add new feature\"",
		"previous_status: discussions",
		"current_status: accepted",
		"changed_at: 2026-01-30T12:00:00Z",
		"comment_url: https://github.com/golang/go/issues/33502#issuecomment-xxx",
	}

	for _, field := range expectedFields {
		if !strings.Contains(fileContent, field) {
			t.Errorf("File should contain %q", field)
		}
	}

	// Verify related_issues section
	if !strings.Contains(fileContent, "related_issues:") {
		t.Error("File should contain related_issues section")
	}
}

func TestManager_WeekDirPath(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		year int
		week int
		want string
	}{
		{
			name: "normal week",
			year: 2026,
			week: 5,
			want: "2026/W05",
		},
		{
			name: "single digit week",
			year: 2026,
			week: 1,
			want: "2026/W01",
		},
		{
			name: "double digit week",
			year: 2025,
			week: 52,
			want: "2025/W52",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := weekDirPath(tt.year, tt.week)
			if got != tt.want {
				t.Errorf("weekDirPath(%d, %d) = %q, want %q", tt.year, tt.week, got, tt.want)
			}
		})
	}
}

func TestProposalFilename(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		issueNumber int
		want        string
	}{
		{
			name:        "normal issue number",
			issueNumber: 12345,
			want:        "proposal-12345.md",
		},
		{
			name:        "small issue number",
			issueNumber: 1,
			want:        "proposal-1.md",
		},
		{
			name:        "large issue number",
			issueNumber: 999999,
			want:        "proposal-999999.md",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := proposalFilename(tt.issueNumber)
			if got != tt.want {
				t.Errorf("proposalFilename(%d) = %q, want %q", tt.issueNumber, got, tt.want)
			}
		})
	}
}

func TestLink(t *testing.T) {
	t.Parallel()

	link := Link{
		Title: "proposal issue",
		URL:   "https://github.com/golang/go/issues/12345",
	}

	if link.Title != "proposal issue" {
		t.Errorf("Link.Title = %q, want %q", link.Title, "proposal issue")
	}
	if link.URL != "https://github.com/golang/go/issues/12345" {
		t.Errorf("Link.URL = %q, want %q", link.URL, "https://github.com/golang/go/issues/12345")
	}
}
