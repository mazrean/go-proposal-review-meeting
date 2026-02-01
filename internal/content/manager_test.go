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
		want string
		year int
		week int
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
		want        string
		issueNumber int
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

func TestManager_MergeContent(t *testing.T) {
	t.Parallel()

	baseTime := time.Date(2026, 1, 30, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		existing     *WeeklyContent
		newContent   *WeeklyContent
		wantStatuses map[int]parser.Status
		name         string
		wantLen      int
	}{
		{
			name:     "merge with no existing content",
			existing: nil,
			newContent: &WeeklyContent{
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
						Summary:        "AI generated summary",
						Links: []Link{
							{Title: "proposal issue", URL: "https://github.com/golang/go/issues/12345"},
						},
					},
				},
				CreatedAt: baseTime,
			},
			wantLen: 1,
			wantStatuses: map[int]parser.Status{
				12345: parser.StatusAccepted,
			},
		},
		{
			name: "merge new proposal into existing week",
			existing: &WeeklyContent{
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
						Summary:        "Existing summary",
						Links: []Link{
							{Title: "proposal issue", URL: "https://github.com/golang/go/issues/12345"},
						},
					},
				},
				CreatedAt: baseTime,
			},
			newContent: &WeeklyContent{
				Year: 2026,
				Week: 5,
				Proposals: []ProposalContent{
					{
						IssueNumber:    67890,
						Title:          "proposal: another feature",
						PreviousStatus: parser.StatusActive,
						CurrentStatus:  parser.StatusDeclined,
						ChangedAt:      baseTime.Add(time.Hour),
						CommentURL:     "https://github.com/golang/go/issues/33502#issuecomment-yyy",
						Summary:        "New summary",
						Links: []Link{
							{Title: "proposal issue", URL: "https://github.com/golang/go/issues/67890"},
						},
					},
				},
				CreatedAt: baseTime.Add(time.Hour),
			},
			wantLen: 2,
			wantStatuses: map[int]parser.Status{
				12345: parser.StatusAccepted,
				67890: parser.StatusDeclined,
			},
		},
		{
			name: "update existing proposal status - preserve older status as previous",
			existing: &WeeklyContent{
				Year: 2026,
				Week: 5,
				Proposals: []ProposalContent{
					{
						IssueNumber:    12345,
						Title:          "proposal: add new feature",
						PreviousStatus: parser.StatusDiscussions,
						CurrentStatus:  parser.StatusLikelyAccept,
						ChangedAt:      baseTime,
						CommentURL:     "https://github.com/golang/go/issues/33502#issuecomment-xxx",
						Summary:        "Existing summary",
						Links: []Link{
							{Title: "proposal issue", URL: "https://github.com/golang/go/issues/12345"},
						},
					},
				},
				CreatedAt: baseTime,
			},
			newContent: &WeeklyContent{
				Year: 2026,
				Week: 5,
				Proposals: []ProposalContent{
					{
						IssueNumber:    12345,
						Title:          "proposal: add new feature",
						PreviousStatus: parser.StatusLikelyAccept,
						CurrentStatus:  parser.StatusAccepted,
						ChangedAt:      baseTime.Add(2 * time.Hour),
						CommentURL:     "https://github.com/golang/go/issues/33502#issuecomment-zzz",
						Summary:        "Updated summary",
						Links: []Link{
							{Title: "proposal issue", URL: "https://github.com/golang/go/issues/12345"},
						},
					},
				},
				CreatedAt: baseTime.Add(2 * time.Hour),
			},
			wantLen: 1,
			wantStatuses: map[int]parser.Status{
				12345: parser.StatusAccepted,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mgr := NewManager()
			merged := mgr.MergeContent(tt.existing, tt.newContent)

			if merged == nil {
				t.Fatal("MergeContent() returned nil")
			}

			if len(merged.Proposals) != tt.wantLen {
				t.Errorf("len(Proposals) = %d, want %d", len(merged.Proposals), tt.wantLen)
			}

			// Verify statuses
			for _, p := range merged.Proposals {
				wantStatus, ok := tt.wantStatuses[p.IssueNumber]
				if !ok {
					t.Errorf("Unexpected proposal in merged content: %d", p.IssueNumber)
					continue
				}
				if p.CurrentStatus != wantStatus {
					t.Errorf("Proposals[%d].CurrentStatus = %q, want %q", p.IssueNumber, p.CurrentStatus, wantStatus)
				}
			}
		})
	}
}

func TestManager_MergeContent_PreservesSummary(t *testing.T) {
	t.Parallel()

	baseTime := time.Date(2026, 1, 30, 12, 0, 0, 0, time.UTC)

	existing := &WeeklyContent{
		Year: 2026,
		Week: 5,
		Proposals: []ProposalContent{
			{
				IssueNumber:    12345,
				Title:          "proposal: add new feature",
				PreviousStatus: parser.StatusDiscussions,
				CurrentStatus:  parser.StatusLikelyAccept,
				ChangedAt:      baseTime,
				CommentURL:     "https://github.com/golang/go/issues/33502#issuecomment-xxx",
				Summary:        "Existing summary that should be preserved",
				Links: []Link{
					{Title: "proposal issue", URL: "https://github.com/golang/go/issues/12345"},
				},
			},
		},
		CreatedAt: baseTime,
	}

	newContent := &WeeklyContent{
		Year: 2026,
		Week: 5,
		Proposals: []ProposalContent{
			{
				IssueNumber:    12345,
				Title:          "proposal: add new feature",
				PreviousStatus: parser.StatusLikelyAccept,
				CurrentStatus:  parser.StatusAccepted,
				ChangedAt:      baseTime.Add(2 * time.Hour),
				CommentURL:     "https://github.com/golang/go/issues/33502#issuecomment-zzz",
				Summary:        "", // New update has no summary
				Links: []Link{
					{Title: "proposal issue", URL: "https://github.com/golang/go/issues/12345"},
				},
			},
		},
		CreatedAt: baseTime.Add(2 * time.Hour),
	}

	mgr := NewManager()
	merged := mgr.MergeContent(existing, newContent)

	if merged == nil {
		t.Fatal("MergeContent() returned nil")
	}

	if len(merged.Proposals) != 1 {
		t.Fatalf("len(Proposals) = %d, want 1", len(merged.Proposals))
	}

	// Should preserve existing summary when new summary is empty
	if merged.Proposals[0].Summary != "Existing summary that should be preserved" {
		t.Errorf("Summary = %q, want existing summary preserved", merged.Proposals[0].Summary)
	}
}

func TestManager_MergeContent_MergesLinks(t *testing.T) {
	t.Parallel()

	baseTime := time.Date(2026, 1, 30, 12, 0, 0, 0, time.UTC)

	existing := &WeeklyContent{
		Year: 2026,
		Week: 5,
		Proposals: []ProposalContent{
			{
				IssueNumber:    12345,
				Title:          "proposal: add new feature",
				PreviousStatus: parser.StatusDiscussions,
				CurrentStatus:  parser.StatusLikelyAccept,
				ChangedAt:      baseTime,
				CommentURL:     "https://github.com/golang/go/issues/33502#issuecomment-xxx",
				Summary:        "Summary",
				Links: []Link{
					{Title: "proposal issue", URL: "https://github.com/golang/go/issues/12345"},
					{Title: "existing link", URL: "https://github.com/golang/go/issues/11111"},
				},
			},
		},
		CreatedAt: baseTime,
	}

	newContent := &WeeklyContent{
		Year: 2026,
		Week: 5,
		Proposals: []ProposalContent{
			{
				IssueNumber:    12345,
				Title:          "proposal: add new feature",
				PreviousStatus: parser.StatusLikelyAccept,
				CurrentStatus:  parser.StatusAccepted,
				ChangedAt:      baseTime.Add(2 * time.Hour),
				CommentURL:     "https://github.com/golang/go/issues/33502#issuecomment-zzz",
				Summary:        "Updated summary",
				Links: []Link{
					{Title: "proposal issue", URL: "https://github.com/golang/go/issues/12345"},
					{Title: "new link", URL: "https://github.com/golang/go/issues/22222"},
				},
			},
		},
		CreatedAt: baseTime.Add(2 * time.Hour),
	}

	mgr := NewManager()
	merged := mgr.MergeContent(existing, newContent)

	if merged == nil {
		t.Fatal("MergeContent() returned nil")
	}

	// Should have merged links (deduplicated)
	if len(merged.Proposals[0].Links) < 2 {
		t.Errorf("len(Links) = %d, want at least 2 (merged)", len(merged.Proposals[0].Links))
	}

	// Verify all links are present (deduplicated by URL)
	urlSet := make(map[string]bool)
	for _, link := range merged.Proposals[0].Links {
		urlSet[link.URL] = true
	}

	expectedURLs := []string{
		"https://github.com/golang/go/issues/12345",
		"https://github.com/golang/go/issues/11111",
		"https://github.com/golang/go/issues/22222",
	}
	for _, url := range expectedURLs {
		if !urlSet[url] {
			t.Errorf("Missing expected link URL: %s", url)
		}
	}
}

func TestManager_WriteContentWithMerge(t *testing.T) {
	t.Parallel()

	baseTime := time.Date(2026, 1, 30, 12, 0, 0, 0, time.UTC)

	tmpDir := t.TempDir()
	mgr := NewManager(WithBaseDir(tmpDir))

	// First write
	content1 := &WeeklyContent{
		Year: 2026,
		Week: 5,
		Proposals: []ProposalContent{
			{
				IssueNumber:    12345,
				Title:          "proposal: add new feature",
				PreviousStatus: parser.StatusDiscussions,
				CurrentStatus:  parser.StatusLikelyAccept,
				ChangedAt:      baseTime,
				CommentURL:     "https://github.com/golang/go/issues/33502#issuecomment-xxx",
				Summary:        "First summary",
				Links: []Link{
					{Title: "proposal issue", URL: "https://github.com/golang/go/issues/12345"},
				},
			},
		},
		CreatedAt: baseTime,
	}

	err := mgr.WriteContentWithMerge(content1)
	if err != nil {
		t.Fatalf("WriteContentWithMerge() error = %v", err)
	}

	// Second write with update to same proposal
	content2 := &WeeklyContent{
		Year: 2026,
		Week: 5,
		Proposals: []ProposalContent{
			{
				IssueNumber:    12345,
				Title:          "proposal: add new feature",
				PreviousStatus: parser.StatusLikelyAccept,
				CurrentStatus:  parser.StatusAccepted,
				ChangedAt:      baseTime.Add(2 * time.Hour),
				CommentURL:     "https://github.com/golang/go/issues/33502#issuecomment-zzz",
				Summary:        "", // Empty summary should preserve existing
				Links: []Link{
					{Title: "proposal issue", URL: "https://github.com/golang/go/issues/12345"},
				},
			},
		},
		CreatedAt: baseTime.Add(2 * time.Hour),
	}

	err = mgr.WriteContentWithMerge(content2)
	if err != nil {
		t.Fatalf("WriteContentWithMerge() second call error = %v", err)
	}

	// Read and verify content
	filePath := filepath.Join(tmpDir, "2026/W05", proposalFilename(12345))
	data, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}

	fileContent := string(data)

	// Should have current_status: accepted
	if !strings.Contains(fileContent, "current_status: accepted") {
		t.Error("File should contain updated status: accepted")
	}

	// Should preserve original previous_status from first update (discussions, not likely_accept)
	if !strings.Contains(fileContent, "previous_status: discussions") {
		t.Error("File should contain original previous_status: discussions")
	}

	// Should preserve first summary
	if !strings.Contains(fileContent, "First summary") {
		t.Error("File should preserve the first summary")
	}
}

func TestManager_ReadExistingContent(t *testing.T) {
	t.Parallel()

	baseTime := time.Date(2026, 1, 30, 12, 0, 0, 0, time.UTC)

	tmpDir := t.TempDir()
	mgr := NewManager(WithBaseDir(tmpDir))

	// Write initial content
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
				Summary:        "Test summary",
				Links: []Link{
					{Title: "proposal issue", URL: "https://github.com/golang/go/issues/12345"},
				},
			},
		},
		CreatedAt: baseTime,
	}

	err := mgr.WriteContent(content)
	if err != nil {
		t.Fatalf("WriteContent() error = %v", err)
	}

	// Read existing content
	existing, err := mgr.ReadExistingContent(2026, 5)
	if err != nil {
		t.Fatalf("ReadExistingContent() error = %v", err)
	}

	if existing == nil {
		t.Fatal("ReadExistingContent() returned nil")
	}

	if len(existing.Proposals) != 1 {
		t.Fatalf("len(Proposals) = %d, want 1", len(existing.Proposals))
	}

	p := existing.Proposals[0]
	if p.IssueNumber != 12345 {
		t.Errorf("IssueNumber = %d, want 12345", p.IssueNumber)
	}
	if p.CurrentStatus != parser.StatusAccepted {
		t.Errorf("CurrentStatus = %q, want %q", p.CurrentStatus, parser.StatusAccepted)
	}
	if p.Summary != "Test summary" {
		t.Errorf("Summary = %q, want %q", p.Summary, "Test summary")
	}
}

func TestManager_ReadExistingContent_NotExists(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	mgr := NewManager(WithBaseDir(tmpDir))

	// Try to read non-existent content
	existing, err := mgr.ReadExistingContent(2026, 5)
	if err != nil {
		t.Fatalf("ReadExistingContent() error = %v", err)
	}

	// Should return nil for non-existent content
	if existing != nil {
		t.Errorf("ReadExistingContent() = %+v, want nil", existing)
	}
}

func TestManager_IntegrateSummaries(t *testing.T) {
	t.Parallel()

	baseTime := time.Date(2026, 1, 30, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		content      *WeeklyContent
		summaries    map[int]string
		wantSummary  map[int]string
		wantLinkURLs map[int][]string
		name         string
	}{
		{
			name: "integrate single summary",
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
			summaries: map[int]string{
				12345: "このproposalは新機能を追加するためのものです。技術的な背景として、既存のAPIを拡張する必要がありました。",
			},
			wantSummary: map[int]string{
				12345: "このproposalは新機能を追加するためのものです。技術的な背景として、既存のAPIを拡張する必要がありました。",
			},
			wantLinkURLs: nil,
		},
		{
			name: "integrate summary with links - extracts links from markdown",
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
			summaries: map[int]string{
				12345: "このproposalは新機能を追加します。関連する議論は[#67890](https://github.com/golang/go/issues/67890)と[#11111](https://github.com/golang/go/issues/11111)を参照してください。",
			},
			wantSummary: map[int]string{
				12345: "このproposalは新機能を追加します。関連する議論は[#67890](https://github.com/golang/go/issues/67890)と[#11111](https://github.com/golang/go/issues/11111)を参照してください。",
			},
			wantLinkURLs: map[int][]string{
				12345: {
					"https://github.com/golang/go/issues/12345", // Original link
					"https://github.com/golang/go/issues/67890", // From summary
					"https://github.com/golang/go/issues/11111", // From summary
				},
			},
		},
		{
			name: "integrate multiple summaries",
			content: &WeeklyContent{
				Year: 2026,
				Week: 5,
				Proposals: []ProposalContent{
					{
						IssueNumber:    12345,
						Title:          "proposal: feature one",
						PreviousStatus: parser.StatusDiscussions,
						CurrentStatus:  parser.StatusAccepted,
						ChangedAt:      baseTime,
						CommentURL:     "https://github.com/golang/go/issues/33502#issuecomment-1",
						Summary:        "",
						Links:          nil,
					},
					{
						IssueNumber:    67890,
						Title:          "proposal: feature two",
						PreviousStatus: parser.StatusActive,
						CurrentStatus:  parser.StatusDeclined,
						ChangedAt:      baseTime.Add(time.Hour),
						CommentURL:     "https://github.com/golang/go/issues/33502#issuecomment-2",
						Summary:        "",
						Links:          nil,
					},
				},
				CreatedAt: baseTime,
			},
			summaries: map[int]string{
				12345: "最初のproposalの要約です。",
				67890: "2番目のproposalの要約です。",
			},
			wantSummary: map[int]string{
				12345: "最初のproposalの要約です。",
				67890: "2番目のproposalの要約です。",
			},
			wantLinkURLs: nil,
		},
		{
			name: "partial summaries - some proposals without summary",
			content: &WeeklyContent{
				Year: 2026,
				Week: 5,
				Proposals: []ProposalContent{
					{
						IssueNumber:    12345,
						Title:          "proposal: feature one",
						PreviousStatus: parser.StatusDiscussions,
						CurrentStatus:  parser.StatusAccepted,
						ChangedAt:      baseTime,
						CommentURL:     "https://github.com/golang/go/issues/33502#issuecomment-1",
						Summary:        "",
						Links:          nil,
					},
					{
						IssueNumber:    67890,
						Title:          "proposal: feature two",
						PreviousStatus: parser.StatusActive,
						CurrentStatus:  parser.StatusDeclined,
						ChangedAt:      baseTime.Add(time.Hour),
						CommentURL:     "https://github.com/golang/go/issues/33502#issuecomment-2",
						Summary:        "",
						Links:          nil,
					},
				},
				CreatedAt: baseTime,
			},
			summaries: map[int]string{
				12345: "最初のproposalの要約です。",
				// 67890 has no summary
			},
			wantSummary: map[int]string{
				12345: "最初のproposalの要約です。",
				67890: "", // Should remain empty
			},
			wantLinkURLs: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mgr := NewManager()
			err := mgr.IntegrateSummaries(tt.content, tt.summaries)
			if err != nil {
				t.Fatalf("IntegrateSummaries() error = %v", err)
			}

			for _, p := range tt.content.Proposals {
				// Check summary
				if want, ok := tt.wantSummary[p.IssueNumber]; ok {
					if p.Summary != want {
						t.Errorf("Proposal[%d].Summary = %q, want %q", p.IssueNumber, p.Summary, want)
					}
				}

				// Check extracted links
				if wantURLs, ok := tt.wantLinkURLs[p.IssueNumber]; ok {
					urlSet := make(map[string]bool)
					for _, link := range p.Links {
						urlSet[link.URL] = true
					}
					for _, url := range wantURLs {
						if !urlSet[url] {
							t.Errorf("Proposal[%d] missing expected link URL: %s", p.IssueNumber, url)
						}
					}
				}
			}
		})
	}
}

func TestManager_ApplyFallback(t *testing.T) {
	t.Parallel()

	baseTime := time.Date(2026, 1, 30, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		content             *WeeklyContent
		wantHasFallback     map[int]bool
		wantContainsStrings map[int][]string
		name                string
	}{
		{
			name: "apply fallback to empty summary with basic info",
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
						Links:          nil,
					},
				},
				CreatedAt: baseTime,
			},
			wantHasFallback: map[int]bool{
				12345: true,
			},
			wantContainsStrings: map[int][]string{
				12345: {"12345", "proposal: add new feature", "discussions", "accepted"},
			},
		},
		{
			name: "do not apply fallback to existing summary",
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
						Summary:        "既存の要約です。",
						Links:          nil,
					},
				},
				CreatedAt: baseTime,
			},
			wantHasFallback: map[int]bool{
				12345: false,
			},
			wantContainsStrings: nil,
		},
		{
			name: "mixed - some with summary, some without",
			content: &WeeklyContent{
				Year: 2026,
				Week: 5,
				Proposals: []ProposalContent{
					{
						IssueNumber:    12345,
						Title:          "proposal: feature one",
						PreviousStatus: parser.StatusDiscussions,
						CurrentStatus:  parser.StatusAccepted,
						ChangedAt:      baseTime,
						CommentURL:     "https://github.com/golang/go/issues/33502#issuecomment-1",
						Summary:        "既存の要約です。",
						Links:          nil,
					},
					{
						IssueNumber:    67890,
						Title:          "proposal: feature two",
						PreviousStatus: parser.StatusActive,
						CurrentStatus:  parser.StatusDeclined,
						ChangedAt:      baseTime.Add(time.Hour),
						CommentURL:     "https://github.com/golang/go/issues/33502#issuecomment-2",
						Summary:        "", // Empty - needs fallback
						Links:          nil,
					},
				},
				CreatedAt: baseTime,
			},
			wantHasFallback: map[int]bool{
				12345: false,
				67890: true,
			},
			wantContainsStrings: map[int][]string{
				67890: {"67890", "proposal: feature two", "active", "declined"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mgr := NewManager()
			err := mgr.ApplyFallback(tt.content)
			if err != nil {
				t.Fatalf("ApplyFallback() error = %v", err)
			}

			for _, p := range tt.content.Proposals {
				wantFallback, ok := tt.wantHasFallback[p.IssueNumber]
				if !ok {
					continue
				}

				if wantFallback {
					// Should have fallback text (not empty)
					if p.Summary == "" {
						t.Errorf("Proposal[%d].Summary should have fallback text", p.IssueNumber)
					}
					// Check for expected strings in fallback
					if wantStrings, ok := tt.wantContainsStrings[p.IssueNumber]; ok {
						for _, s := range wantStrings {
							if !strings.Contains(p.Summary, s) {
								t.Errorf("Proposal[%d].Summary should contain %q, got %q", p.IssueNumber, s, p.Summary)
							}
						}
					}
				} else if p.Summary != "既存の要約です。" {
					// Should preserve existing summary
					t.Errorf("Proposal[%d].Summary should preserve existing: got %q", p.IssueNumber, p.Summary)
				}
			}
		})
	}
}

func TestManager_ReadSummaries(t *testing.T) {
	t.Parallel()

	tests := []struct {
		setupDir      func(t *testing.T) string
		wantSummaries map[int]string
		name          string
		wantLen       int
	}{
		{
			name: "read multiple summary files",
			setupDir: func(t *testing.T) string {
				t.Helper()
				tmpDir := t.TempDir()
				summariesDir := filepath.Join(tmpDir, "summaries")
				if err := os.MkdirAll(summariesDir, 0o755); err != nil {
					t.Fatalf("Failed to create summaries dir: %v", err)
				}
				if err := os.WriteFile(filepath.Join(summariesDir, "12345.md"), []byte("このproposalは新機能を追加します。"), 0o644); err != nil {
					t.Fatalf("Failed to write summary file: %v", err)
				}
				if err := os.WriteFile(filepath.Join(summariesDir, "67890.md"), []byte("2番目のproposalの要約です。"), 0o644); err != nil {
					t.Fatalf("Failed to write summary file: %v", err)
				}
				return summariesDir
			},
			wantLen: 2,
			wantSummaries: map[int]string{
				12345: "このproposalは新機能を追加します。",
				67890: "2番目のproposalの要約です。",
			},
		},
		{
			name: "empty directory returns empty map",
			setupDir: func(t *testing.T) string {
				t.Helper()
				tmpDir := t.TempDir()
				summariesDir := filepath.Join(tmpDir, "summaries")
				if err := os.MkdirAll(summariesDir, 0o755); err != nil {
					t.Fatalf("Failed to create summaries dir: %v", err)
				}
				return summariesDir
			},
			wantLen:       0,
			wantSummaries: nil,
		},
		{
			name: "non-existent directory returns empty map",
			setupDir: func(t *testing.T) string {
				t.Helper()
				tmpDir := t.TempDir()
				return filepath.Join(tmpDir, "non-existent")
			},
			wantLen:       0,
			wantSummaries: nil,
		},
		{
			name: "ignores non-matching files",
			setupDir: func(t *testing.T) string {
				t.Helper()
				tmpDir := t.TempDir()
				summariesDir := filepath.Join(tmpDir, "summaries")
				if err := os.MkdirAll(summariesDir, 0o755); err != nil {
					t.Fatalf("Failed to create summaries dir: %v", err)
				}
				// Valid file
				if err := os.WriteFile(filepath.Join(summariesDir, "12345.md"), []byte("有効な要約"), 0o644); err != nil {
					t.Fatalf("Failed to write summary file: %v", err)
				}
				// Invalid files (should be ignored)
				if err := os.WriteFile(filepath.Join(summariesDir, "readme.md"), []byte("README"), 0o644); err != nil {
					t.Fatalf("Failed to write readme file: %v", err)
				}
				if err := os.WriteFile(filepath.Join(summariesDir, "abc.md"), []byte("non-numeric"), 0o644); err != nil {
					t.Fatalf("Failed to write abc file: %v", err)
				}
				return summariesDir
			},
			wantLen: 1,
			wantSummaries: map[int]string{
				12345: "有効な要約",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			summariesDir := tt.setupDir(t)
			mgr := NewManager(WithSummariesDir(summariesDir))
			summaries, err := mgr.ReadSummaries()
			if err != nil {
				t.Fatalf("ReadSummaries() error = %v", err)
			}

			if len(summaries) != tt.wantLen {
				t.Errorf("len(summaries) = %d, want %d", len(summaries), tt.wantLen)
			}

			for issueNum, wantContent := range tt.wantSummaries {
				if got := summaries[issueNum]; got != wantContent {
					t.Errorf("summaries[%d] = %q, want %q", issueNum, got, wantContent)
				}
			}
		})
	}
}

func TestManager_WriteContentWithMerge_PastWeekUnchanged(t *testing.T) {
	t.Parallel()

	baseTimeW4 := time.Date(2026, 1, 23, 12, 0, 0, 0, time.UTC) // W04
	baseTimeW5 := time.Date(2026, 1, 30, 12, 0, 0, 0, time.UTC) // W05

	tmpDir := t.TempDir()
	mgr := NewManager(WithBaseDir(tmpDir))

	// Write W04 content
	contentW4 := &WeeklyContent{
		Year: 2026,
		Week: 4,
		Proposals: []ProposalContent{
			{
				IssueNumber:    11111,
				Title:          "proposal: week 4 feature",
				PreviousStatus: parser.StatusDiscussions,
				CurrentStatus:  parser.StatusAccepted,
				ChangedAt:      baseTimeW4,
				CommentURL:     "https://github.com/golang/go/issues/33502#issuecomment-w4",
				Summary:        "Week 4 summary",
				Links: []Link{
					{Title: "proposal issue", URL: "https://github.com/golang/go/issues/11111"},
				},
			},
		},
		CreatedAt: baseTimeW4,
	}

	err := mgr.WriteContentWithMerge(contentW4)
	if err != nil {
		t.Fatalf("WriteContentWithMerge() W04 error = %v", err)
	}

	// Read W04 file content for comparison
	w4FilePath := filepath.Join(tmpDir, "2026/W04", proposalFilename(11111))
	w4Before, err := os.ReadFile(w4FilePath)
	if err != nil {
		t.Fatalf("Failed to read W04 file: %v", err)
	}

	// Write W05 content (should not affect W04)
	contentW5 := &WeeklyContent{
		Year: 2026,
		Week: 5,
		Proposals: []ProposalContent{
			{
				IssueNumber:    22222,
				Title:          "proposal: week 5 feature",
				PreviousStatus: parser.StatusActive,
				CurrentStatus:  parser.StatusDeclined,
				ChangedAt:      baseTimeW5,
				CommentURL:     "https://github.com/golang/go/issues/33502#issuecomment-w5",
				Summary:        "Week 5 summary",
				Links: []Link{
					{Title: "proposal issue", URL: "https://github.com/golang/go/issues/22222"},
				},
			},
		},
		CreatedAt: baseTimeW5,
	}

	err = mgr.WriteContentWithMerge(contentW5)
	if err != nil {
		t.Fatalf("WriteContentWithMerge() W05 error = %v", err)
	}

	// Verify W04 content is unchanged
	w4After, err := os.ReadFile(w4FilePath)
	if err != nil {
		t.Fatalf("Failed to read W04 file after W05 write: %v", err)
	}

	if string(w4Before) != string(w4After) {
		t.Error("W04 content should not be modified when writing W05")
	}

	// Verify W05 exists
	w5FilePath := filepath.Join(tmpDir, "2026/W05", proposalFilename(22222))
	if _, err := os.Stat(w5FilePath); os.IsNotExist(err) {
		t.Error("W05 file should exist")
	}
}

// TestIntegrateSummaries_WithReasonBackgroundLinks verifies that summaries
// containing reason, background, and related links are properly integrated.
func TestIntegrateSummaries_WithReasonBackgroundLinks(t *testing.T) {
	t.Parallel()

	baseTime := time.Date(2026, 1, 30, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name            string
		summary         string
		wantExtractURLs []string
		wantReason      bool
		wantBackground  bool
	}{
		{
			name: "summary with reason and background",
			summary: `このproposalは新しいAPIを追加するものです。

**理由**: 既存のAPIでは複雑な操作が困難でした。
**背景**: Go 1.21からジェネリクスが導入され、より柔軟な実装が可能になりました。

詳細は[#67890](https://github.com/golang/go/issues/67890)を参照してください。`,
			wantReason:     true,
			wantBackground: true,
			wantExtractURLs: []string{
				"https://github.com/golang/go/issues/67890",
			},
		},
		{
			name: "summary with multiple related links",
			summary: `このproposalはエラーハンドリングを改善します。

関連する議論: [#11111](https://github.com/golang/go/issues/11111)、[#22222](https://github.com/golang/go/issues/22222)

元の提案: [#33333](https://github.com/golang/go/issues/33333)`,
			wantReason:     false,
			wantBackground: false,
			wantExtractURLs: []string{
				"https://github.com/golang/go/issues/11111",
				"https://github.com/golang/go/issues/22222",
				"https://github.com/golang/go/issues/33333",
			},
		},
		{
			name:            "summary without links",
			summary:         "シンプルな要約です。理由と背景の説明はありますが、リンクはありません。",
			wantReason:      false,
			wantBackground:  false,
			wantExtractURLs: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			content := &WeeklyContent{
				Year: 2026,
				Week: 5,
				Proposals: []ProposalContent{
					{
						IssueNumber:    12345,
						Title:          "proposal: test feature",
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
			}

			summaries := map[int]string{
				12345: tt.summary,
			}

			mgr := NewManager()
			err := mgr.IntegrateSummaries(content, summaries)
			if err != nil {
				t.Fatalf("IntegrateSummaries() error = %v", err)
			}

			p := content.Proposals[0]

			// Verify summary was integrated
			if p.Summary != tt.summary {
				t.Errorf("Summary = %q, want %q", p.Summary, tt.summary)
			}

			// Verify reason content (if expected)
			if tt.wantReason && !strings.Contains(p.Summary, "理由") {
				t.Error("Summary should contain 理由 (reason)")
			}

			// Verify background content (if expected)
			if tt.wantBackground && !strings.Contains(p.Summary, "背景") {
				t.Error("Summary should contain 背景 (background)")
			}

			// Verify extracted links
			if len(tt.wantExtractURLs) > 0 {
				urlSet := make(map[string]bool)
				for _, link := range p.Links {
					urlSet[link.URL] = true
				}
				for _, wantURL := range tt.wantExtractURLs {
					if !urlSet[wantURL] {
						t.Errorf("Missing extracted link URL: %s", wantURL)
					}
				}
			}
		})
	}
}

// TestValidateSummaryLength tests the validation of summary character count.
// Summaries should ideally be 200-500 characters as per requirements.
func TestValidateSummaryLength(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		summary    string
		wantReason string
		wantValid  bool
	}{
		{
			name:       "valid summary within range (200-500 chars)",
			summary:    strings.Repeat("あ", 300), // 300 chars
			wantValid:  true,
			wantReason: "",
		},
		{
			name:       "valid summary at minimum (200 chars)",
			summary:    strings.Repeat("あ", 200),
			wantValid:  true,
			wantReason: "",
		},
		{
			name:       "valid summary at maximum (500 chars)",
			summary:    strings.Repeat("あ", 500),
			wantValid:  true,
			wantReason: "",
		},
		{
			name:       "summary too short (under 200 chars)",
			summary:    strings.Repeat("あ", 100),
			wantValid:  false,
			wantReason: "too short",
		},
		{
			name:       "summary too long (over 500 chars)",
			summary:    strings.Repeat("あ", 600),
			wantValid:  false,
			wantReason: "too long",
		},
		{
			name:       "empty summary",
			summary:    "",
			wantValid:  false,
			wantReason: "too short",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			valid, reason := ValidateSummaryLength(tt.summary)
			if valid != tt.wantValid {
				t.Errorf("ValidateSummaryLength() valid = %v, want %v", valid, tt.wantValid)
			}
			if tt.wantReason != "" && !strings.Contains(reason, tt.wantReason) {
				t.Errorf("ValidateSummaryLength() reason = %q, want containing %q", reason, tt.wantReason)
			}
		})
	}
}

// TestExtractLinksFromMarkdown tests the link extraction from markdown text.
func TestExtractLinksFromMarkdown(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		text     string
		wantURLs []string
	}{
		{
			name:     "single link",
			text:     "関連: [#12345](https://github.com/golang/go/issues/12345)",
			wantURLs: []string{"https://github.com/golang/go/issues/12345"},
		},
		{
			name:     "multiple links",
			text:     "[issue1](https://github.com/golang/go/issues/111) and [issue2](https://github.com/golang/go/issues/222)",
			wantURLs: []string{"https://github.com/golang/go/issues/111", "https://github.com/golang/go/issues/222"},
		},
		{
			name:     "no links",
			text:     "This is plain text without any links.",
			wantURLs: nil,
		},
		{
			name:     "non-github links ignored",
			text:     "[external](https://example.com) [github](https://github.com/golang/go/issues/123)",
			wantURLs: []string{"https://github.com/golang/go/issues/123"},
		},
		{
			name:     "link with issuecomment anchor",
			text:     "[review comment](https://github.com/golang/go/issues/33502#issuecomment-1234567890)",
			wantURLs: []string{"https://github.com/golang/go/issues/33502#issuecomment-1234567890"},
		},
		{
			name:     "mixed links with and without anchors",
			text:     "[issue](https://github.com/golang/go/issues/12345) [comment](https://github.com/golang/go/issues/67890#issuecomment-999)",
			wantURLs: []string{"https://github.com/golang/go/issues/12345", "https://github.com/golang/go/issues/67890#issuecomment-999"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			links := extractLinksFromMarkdown(tt.text)

			if len(tt.wantURLs) == 0 && len(links) == 0 {
				return // Both empty, pass
			}

			gotURLs := make([]string, len(links))
			for i, link := range links {
				gotURLs[i] = link.URL
			}

			if len(gotURLs) != len(tt.wantURLs) {
				t.Errorf("extractLinksFromMarkdown() returned %d links, want %d", len(gotURLs), len(tt.wantURLs))
				return
			}

			for i, wantURL := range tt.wantURLs {
				if gotURLs[i] != wantURL {
					t.Errorf("extractLinksFromMarkdown()[%d].URL = %q, want %q", i, gotURLs[i], wantURL)
				}
			}
		})
	}
}

// TestManager_ListAllWeeks tests listing all weekly contents from the content directory.
func TestManager_ListAllWeeks(t *testing.T) {
	t.Parallel()

	baseTime := time.Date(2026, 1, 30, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		setup    func(t *testing.T, dir string)
		validate func(t *testing.T, weeks []*WeeklyContent)
		name     string
		wantLen  int
		wantErr  bool
	}{
		{
			name:    "empty directory returns nil",
			setup:   func(_ *testing.T, _ string) {},
			wantLen: 0,
			wantErr: false,
		},
		{
			name: "single week",
			setup: func(t *testing.T, dir string) {
				t.Helper()
				mgr := NewManager(WithBaseDir(dir))
				content := &WeeklyContent{
					Year: 2026,
					Week: 5,
					Proposals: []ProposalContent{
						{
							IssueNumber:    12345,
							Title:          "test proposal",
							PreviousStatus: parser.StatusDiscussions,
							CurrentStatus:  parser.StatusAccepted,
							ChangedAt:      baseTime,
							CommentURL:     "https://example.com/comment",
						},
					},
				}
				if err := mgr.WriteContent(content); err != nil {
					t.Fatalf("failed to write content: %v", err)
				}
			},
			wantLen: 1,
			wantErr: false,
			validate: func(t *testing.T, weeks []*WeeklyContent) {
				t.Helper()
				if weeks[0].Year != 2026 || weeks[0].Week != 5 {
					t.Errorf("expected 2026-W05, got %d-W%02d", weeks[0].Year, weeks[0].Week)
				}
			},
		},
		{
			name: "multiple weeks sorted newest first",
			setup: func(t *testing.T, dir string) {
				t.Helper()
				mgr := NewManager(WithBaseDir(dir))

				// Write W04
				if err := mgr.WriteContent(&WeeklyContent{
					Year: 2026,
					Week: 4,
					Proposals: []ProposalContent{
						{
							IssueNumber:    11111,
							Title:          "week 4 proposal",
							PreviousStatus: parser.StatusDiscussions,
							CurrentStatus:  parser.StatusAccepted,
							ChangedAt:      baseTime.Add(-7 * 24 * time.Hour),
							CommentURL:     "https://example.com/w4",
						},
					},
				}); err != nil {
					t.Fatalf("failed to write W04 content: %v", err)
				}

				// Write W05
				if err := mgr.WriteContent(&WeeklyContent{
					Year: 2026,
					Week: 5,
					Proposals: []ProposalContent{
						{
							IssueNumber:    12345,
							Title:          "week 5 proposal",
							PreviousStatus: parser.StatusDiscussions,
							CurrentStatus:  parser.StatusAccepted,
							ChangedAt:      baseTime,
							CommentURL:     "https://example.com/w5",
						},
					},
				}); err != nil {
					t.Fatalf("failed to write W05 content: %v", err)
				}
			},
			wantLen: 2,
			wantErr: false,
			validate: func(t *testing.T, weeks []*WeeklyContent) {
				t.Helper()
				// Should be sorted newest first (W05, W04)
				if weeks[0].Week != 5 {
					t.Errorf("first week should be W05, got W%02d", weeks[0].Week)
				}
				if weeks[1].Week != 4 {
					t.Errorf("second week should be W04, got W%02d", weeks[1].Week)
				}
			},
		},
		{
			name: "multiple years sorted correctly",
			setup: func(t *testing.T, dir string) {
				t.Helper()
				mgr := NewManager(WithBaseDir(dir))

				// Write 2025 W52
				if err := mgr.WriteContent(&WeeklyContent{
					Year: 2025,
					Week: 52,
					Proposals: []ProposalContent{
						{
							IssueNumber:    10000,
							Title:          "2025 proposal",
							PreviousStatus: parser.StatusDiscussions,
							CurrentStatus:  parser.StatusAccepted,
							ChangedAt:      baseTime.Add(-30 * 24 * time.Hour),
							CommentURL:     "https://example.com/2025",
						},
					},
				}); err != nil {
					t.Fatalf("failed to write 2025 content: %v", err)
				}

				// Write 2026 W01
				if err := mgr.WriteContent(&WeeklyContent{
					Year: 2026,
					Week: 1,
					Proposals: []ProposalContent{
						{
							IssueNumber:    20000,
							Title:          "2026 proposal",
							PreviousStatus: parser.StatusDiscussions,
							CurrentStatus:  parser.StatusAccepted,
							ChangedAt:      baseTime,
							CommentURL:     "https://example.com/2026",
						},
					},
				}); err != nil {
					t.Fatalf("failed to write 2026 content: %v", err)
				}
			},
			wantLen: 2,
			wantErr: false,
			validate: func(t *testing.T, weeks []*WeeklyContent) {
				t.Helper()
				// Should be sorted newest first (2026-W01, 2025-W52)
				if weeks[0].Year != 2026 {
					t.Errorf("first year should be 2026, got %d", weeks[0].Year)
				}
				if weeks[1].Year != 2025 {
					t.Errorf("second year should be 2025, got %d", weeks[1].Year)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			tmpDir := t.TempDir()
			tt.setup(t, tmpDir)

			mgr := NewManager(WithBaseDir(tmpDir))
			weeks, err := mgr.ListAllWeeks()

			if (err != nil) != tt.wantErr {
				t.Fatalf("ListAllWeeks() error = %v, wantErr %v", err, tt.wantErr)
			}

			if len(weeks) != tt.wantLen {
				t.Errorf("ListAllWeeks() returned %d weeks, want %d", len(weeks), tt.wantLen)
			}

			if tt.validate != nil && len(weeks) > 0 {
				tt.validate(t, weeks)
			}
		})
	}
}

// TestParseProposalFile_InvalidIssueNumber tests that parseProposalFile returns error for invalid issue_number.
func TestParseProposalFile_InvalidIssueNumber(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "proposal-invalid.md")

	// Write a file with invalid issue_number (number too large for int)
	content := `---
issue_number: 99999999999999999999999999999999
title: "test proposal"
previous_status: discussions
current_status: accepted
changed_at: 2026-01-30T12:00:00Z
comment_url: https://example.com
---

## 要約

Test summary
`
	if err := os.WriteFile(filePath, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	_, err := parseProposalFile(filePath)
	if err == nil {
		t.Error("parseProposalFile() should return error for invalid issue_number (overflow)")
	}
	if err != nil && !strings.Contains(err.Error(), "issue_number") {
		t.Errorf("error should mention issue_number, got: %v", err)
	}
}

// TestParseProposalFile_MissingRequiredFields tests that parseProposalFile returns error for missing required fields.
func TestParseProposalFile_MissingRequiredFields(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		content        string
		wantErrContain string
	}{
		{
			name: "missing issue_number",
			content: `---
title: "test proposal"
previous_status: discussions
current_status: accepted
changed_at: 2026-01-30T12:00:00Z
comment_url: https://example.com
---
`,
			wantErrContain: "issue_number",
		},
		{
			name: "missing title",
			content: `---
issue_number: 12345
previous_status: discussions
current_status: accepted
changed_at: 2026-01-30T12:00:00Z
comment_url: https://example.com
---
`,
			wantErrContain: "title",
		},
		{
			name: "missing previous_status",
			content: `---
issue_number: 12345
title: "test proposal"
current_status: accepted
changed_at: 2026-01-30T12:00:00Z
comment_url: https://example.com
---
`,
			wantErrContain: "previous_status",
		},
		{
			name: "missing current_status",
			content: `---
issue_number: 12345
title: "test proposal"
previous_status: discussions
changed_at: 2026-01-30T12:00:00Z
comment_url: https://example.com
---
`,
			wantErrContain: "current_status",
		},
		{
			name: "missing changed_at",
			content: `---
issue_number: 12345
title: "test proposal"
previous_status: discussions
current_status: accepted
comment_url: https://example.com
---
`,
			wantErrContain: "changed_at",
		},
		{
			name: "missing comment_url",
			content: `---
issue_number: 12345
title: "test proposal"
previous_status: discussions
current_status: accepted
changed_at: 2026-01-30T12:00:00Z
---
`,
			wantErrContain: "comment_url",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			tmpDir := t.TempDir()
			filePath := filepath.Join(tmpDir, "proposal-test.md")

			if err := os.WriteFile(filePath, []byte(tt.content), 0o644); err != nil {
				t.Fatalf("failed to write test file: %v", err)
			}

			_, err := parseProposalFile(filePath)
			if err == nil {
				t.Errorf("parseProposalFile() should return error for %s", tt.name)
			}
			if err != nil && !strings.Contains(err.Error(), tt.wantErrContain) {
				t.Errorf("error should contain %q, got: %v", tt.wantErrContain, err)
			}
		})
	}
}

// TestParseProposalFile_InvalidChangedAt tests that parseProposalFile returns error for invalid changed_at.
func TestParseProposalFile_InvalidChangedAt(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "proposal-invalid-date.md")

	// Write a file with invalid changed_at format
	content := `---
issue_number: 12345
title: "test proposal"
previous_status: discussions
current_status: accepted
changed_at: invalid-date-format
comment_url: https://example.com
---

## 要約

Test summary
`
	if err := os.WriteFile(filePath, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	_, err := parseProposalFile(filePath)
	if err == nil {
		t.Error("parseProposalFile() should return error for invalid changed_at")
	}
	if !strings.Contains(err.Error(), "changed_at") {
		t.Errorf("error should mention changed_at, got: %v", err)
	}
}

// TestManager_ListAllWeeks_ErrorOnCorruptedFile tests that ListAllWeeks returns error when file is corrupted.
func TestManager_ListAllWeeks_ErrorOnCorruptedFile(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()

	// Create a valid directory structure but with corrupted file content
	weekDir := filepath.Join(tmpDir, "2026", "W05")
	if err := os.MkdirAll(weekDir, 0o755); err != nil {
		t.Fatalf("failed to create directory: %v", err)
	}

	// Write a corrupted proposal file (invalid changed_at format)
	corruptedContent := `---
issue_number: 12345
title: "corrupted proposal"
previous_status: discussions
current_status: accepted
changed_at: not-a-valid-date
comment_url: https://example.com
---
`
	if err := os.WriteFile(filepath.Join(weekDir, "proposal-12345.md"), []byte(corruptedContent), 0o644); err != nil {
		t.Fatalf("failed to write corrupted file: %v", err)
	}

	mgr := NewManager(WithBaseDir(tmpDir))
	_, err := mgr.ListAllWeeks()
	if err == nil {
		t.Error("ListAllWeeks() should return error when file is corrupted")
	}
}

// TestGenerateFallbackSummary tests the fallback summary generation.
func TestGenerateFallbackSummary(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		wantContains    []string
		wantNotContains []string
		proposal        ProposalContent
	}{
		{
			name: "discussions to accepted",
			proposal: ProposalContent{
				IssueNumber:    12345,
				Title:          "proposal: add new feature",
				PreviousStatus: parser.StatusDiscussions,
				CurrentStatus:  parser.StatusAccepted,
			},
			wantContains: []string{
				"12345",
				"proposal: add new feature",
				"discussions",
				"accepted",
			},
			wantNotContains: nil,
		},
		{
			name: "active to declined",
			proposal: ProposalContent{
				IssueNumber:    67890,
				Title:          "proposal: remove deprecated API",
				PreviousStatus: parser.StatusActive,
				CurrentStatus:  parser.StatusDeclined,
			},
			wantContains: []string{
				"67890",
				"proposal: remove deprecated API",
				"active",
				"declined",
			},
			wantNotContains: nil,
		},
		{
			name: "likely_accept to accepted",
			proposal: ProposalContent{
				IssueNumber:    11111,
				Title:          "proposal: improve error handling",
				PreviousStatus: parser.StatusLikelyAccept,
				CurrentStatus:  parser.StatusAccepted,
			},
			wantContains: []string{
				"11111",
				"proposal: improve error handling",
				"likely_accept",
				"accepted",
			},
			wantNotContains: nil,
		},
		{
			name: "title with special characters",
			proposal: ProposalContent{
				IssueNumber:    99999,
				Title:          "proposal: add `context.Context` to API",
				PreviousStatus: parser.StatusDiscussions,
				CurrentStatus:  parser.StatusHold,
			},
			wantContains: []string{
				"99999",
				"proposal: add `context.Context` to API",
				"discussions",
				"hold",
			},
			wantNotContains: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			summary := generateFallbackSummary(tt.proposal)

			for _, s := range tt.wantContains {
				if !strings.Contains(summary, s) {
					t.Errorf("generateFallbackSummary() should contain %q, got %q", s, summary)
				}
			}

			for _, s := range tt.wantNotContains {
				if strings.Contains(summary, s) {
					t.Errorf("generateFallbackSummary() should not contain %q, got %q", s, summary)
				}
			}
		})
	}
}
