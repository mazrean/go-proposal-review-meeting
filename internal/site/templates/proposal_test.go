package templates_test

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"

	"github.com/mazrean/go-proposal-review-meeting/internal/content"
	"github.com/mazrean/go-proposal-review-meeting/internal/parser"
	"github.com/mazrean/go-proposal-review-meeting/internal/site/templates"
)

func TestProposalDetail(t *testing.T) {
	t.Parallel()

	tests := []struct {
		checkFunc      func(t *testing.T, html string)
		name           string
		wantContains   []string
		wantNotContain []string
		proposal       templates.ProposalDetailData
	}{
		{
			name: "renders proposal title and issue number",
			proposal: templates.ProposalDetailData{
				IssueNumber:    12345,
				Title:          "proposal: add generics",
				CurrentStatus:  parser.StatusAccepted,
				PreviousStatus: parser.StatusDiscussions,
				IssueURL:       "https://github.com/golang/go/issues/12345",
				CommentURL:     "https://github.com/golang/go/issues/33502#issuecomment-123",
				ChangedAt:      time.Date(2026, 1, 30, 12, 0, 0, 0, time.UTC),
			},
			wantContains: []string{
				"#12345",
				"proposal: add generics",
				"https://github.com/golang/go/issues/12345",
			},
		},
		{
			name: "displays status badge",
			proposal: templates.ProposalDetailData{
				IssueNumber:   12345,
				Title:         "test",
				CurrentStatus: parser.StatusDeclined,
				IssueURL:      "https://github.com/golang/go/issues/12345",
			},
			wantContains: []string{
				"declined",
			},
		},
		{
			name: "shows full summary",
			proposal: templates.ProposalDetailData{
				IssueNumber:   12345,
				Title:         "test",
				CurrentStatus: parser.StatusAccepted,
				Summary:       "This is a detailed summary of the proposal change explaining the technical background and reasons.",
				IssueURL:      "https://github.com/golang/go/issues/12345",
			},
			wantContains: []string{
				"This is a detailed summary of the proposal change explaining the technical background and reasons.",
			},
		},
		{
			name: "displays status change information",
			proposal: templates.ProposalDetailData{
				IssueNumber:    12345,
				Title:          "test",
				PreviousStatus: parser.StatusDiscussions,
				CurrentStatus:  parser.StatusAccepted,
				IssueURL:       "https://github.com/golang/go/issues/12345",
			},
			wantContains: []string{
				"discussions",
				"accepted",
			},
		},
		{
			name: "shows related links",
			proposal: templates.ProposalDetailData{
				IssueNumber:   12345,
				Title:         "test",
				CurrentStatus: parser.StatusAccepted,
				IssueURL:      "https://github.com/golang/go/issues/12345",
				Links: []templates.LinkData{
					{Title: "proposal issue", URL: "https://github.com/golang/go/issues/12345"},
					{Title: "review comment", URL: "https://github.com/golang/go/issues/33502#issuecomment-456"},
				},
			},
			wantContains: []string{
				"proposal issue",
				"review comment",
				"https://github.com/golang/go/issues/12345",
				"https://github.com/golang/go/issues/33502#issuecomment-456",
			},
		},
		{
			name: "shows review comment link",
			proposal: templates.ProposalDetailData{
				IssueNumber:   12345,
				Title:         "test",
				CurrentStatus: parser.StatusAccepted,
				IssueURL:      "https://github.com/golang/go/issues/12345",
				CommentURL:    "https://github.com/golang/go/issues/33502#issuecomment-123",
			},
			wantContains: []string{
				"https://github.com/golang/go/issues/33502#issuecomment-123",
			},
		},
		{
			name: "omits summary section when empty",
			proposal: templates.ProposalDetailData{
				IssueNumber:   12345,
				Title:         "test",
				CurrentStatus: parser.StatusAccepted,
				IssueURL:      "https://github.com/golang/go/issues/12345",
				Summary:       "",
			},
			wantNotContain: []string{
				">要約<",
			},
		},
		{
			name: "omits review comment link when empty",
			proposal: templates.ProposalDetailData{
				IssueNumber:   12345,
				Title:         "test",
				CurrentStatus: parser.StatusAccepted,
				IssueURL:      "https://github.com/golang/go/issues/12345",
				CommentURL:    "",
			},
			wantNotContain: []string{
				"Review Comment",
			},
		},
		{
			name: "omits extra links when none",
			proposal: templates.ProposalDetailData{
				IssueNumber:   12345,
				Title:         "test",
				CurrentStatus: parser.StatusAccepted,
				IssueURL:      "https://github.com/golang/go/issues/12345",
				Links:         nil,
			},
			wantContains: []string{
				"Proposal Issue",
			},
			checkFunc: func(t *testing.T, html string) {
				t.Helper()
				liCount := strings.Count(html, "<li>")
				if liCount != 1 {
					t.Errorf("expected exactly 1 list item (Proposal Issue), got %d", liCount)
				}
			},
		},
		{
			name: "omits date when zero",
			proposal: templates.ProposalDetailData{
				IssueNumber:   12345,
				Title:         "test",
				CurrentStatus: parser.StatusAccepted,
				IssueURL:      "https://github.com/golang/go/issues/12345",
				ChangedAt:     time.Time{},
			},
			wantNotContain: []string{
				"更新日時",
			},
		},
		{
			name: "formats date correctly",
			proposal: templates.ProposalDetailData{
				IssueNumber:   12345,
				Title:         "test",
				CurrentStatus: parser.StatusAccepted,
				IssueURL:      "https://github.com/golang/go/issues/12345",
				ChangedAt:     time.Date(2026, 1, 30, 12, 0, 0, 0, time.UTC),
			},
			wantContains: []string{
				"2026年1月30日",
				"2026-01-30T12:00:00Z",
			},
		},
		{
			name: "omits status change when same",
			proposal: templates.ProposalDetailData{
				IssueNumber:    12345,
				Title:          "test",
				PreviousStatus: parser.StatusAccepted,
				CurrentStatus:  parser.StatusAccepted,
				IssueURL:       "https://github.com/golang/go/issues/12345",
			},
			wantNotContain: []string{
				"ステータス変更",
			},
		},
		{
			name: "omits status change when previous status empty",
			proposal: templates.ProposalDetailData{
				IssueNumber:    12345,
				Title:          "test",
				PreviousStatus: "",
				CurrentStatus:  parser.StatusAccepted,
				IssueURL:       "https://github.com/golang/go/issues/12345",
			},
			wantNotContain: []string{
				"ステータス変更",
			},
		},
		{
			name: "uses h1 for main title",
			proposal: templates.ProposalDetailData{
				IssueNumber:   12345,
				Title:         "test proposal",
				CurrentStatus: parser.StatusAccepted,
				IssueURL:      "https://github.com/golang/go/issues/12345",
			},
			wantContains: []string{
				"<h1",
			},
			wantNotContain: []string{
				`<h2 class="text-3xl`,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var buf bytes.Buffer
			err := templates.ProposalDetail(tt.proposal).Render(context.Background(), &buf)
			if err != nil {
				t.Fatalf("failed to render: %v", err)
			}

			html := buf.String()

			for _, want := range tt.wantContains {
				if !strings.Contains(html, want) {
					t.Errorf("expected HTML to contain %q, got:\n%s", want, html)
				}
			}

			for _, notWant := range tt.wantNotContain {
				if strings.Contains(html, notWant) {
					t.Errorf("expected HTML to NOT contain %q, got:\n%s", notWant, html)
				}
			}

			if tt.checkFunc != nil {
				tt.checkFunc(t, html)
			}
		})
	}
}

func TestProposalDetailPage(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		wantContains []string
		proposal     templates.ProposalDetailData
	}{
		{
			name: "renders full page with layout",
			proposal: templates.ProposalDetailData{
				IssueNumber:    12345,
				Title:          "test proposal",
				CurrentStatus:  parser.StatusAccepted,
				PreviousStatus: parser.StatusDiscussions,
				Summary:        "This is a test summary.",
				IssueURL:       "https://github.com/golang/go/issues/12345",
				CommentURL:     "https://github.com/golang/go/issues/33502#issuecomment-123",
				Year:           2026,
				Week:           5,
				ChangedAt:      time.Date(2026, 1, 30, 12, 0, 0, 0, time.UTC),
				Links: []templates.LinkData{
					{Title: "proposal issue", URL: "https://github.com/golang/go/issues/12345"},
				},
			},
			wantContains: []string{
				"<!doctype html>",
				"<title>",
				"#12345",
				"test proposal",
				"<header",
				"<nav",
				"<main",
				"<footer",
				"This is a test summary.",
				"</html>",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var buf bytes.Buffer
			err := templates.ProposalDetailPage(tt.proposal).Render(context.Background(), &buf)
			if err != nil {
				t.Fatalf("failed to render: %v", err)
			}

			html := buf.String()

			for _, want := range tt.wantContains {
				if !strings.Contains(html, want) {
					t.Errorf("expected HTML to contain %q, got:\n%s", want, html)
				}
			}
		})
	}
}

func TestConvertToProposalDetailData(t *testing.T) {
	t.Parallel()

	changedAt := time.Date(2026, 1, 30, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		input       *content.WeeklyContent
		checkFunc   func(t *testing.T, data *templates.ProposalDetailData)
		name        string
		issueNumber int
		wantNil     bool
	}{
		{
			name:        "nil input returns nil",
			input:       nil,
			issueNumber: 12345,
			wantNil:     true,
		},
		{
			name: "not found returns nil",
			input: &content.WeeklyContent{
				Year: 2026,
				Week: 5,
				Proposals: []content.ProposalContent{
					{IssueNumber: 12345, Title: "proposal: test"},
				},
			},
			issueNumber: 99999,
			wantNil:     true,
		},
		{
			name: "converts valid content with all fields",
			input: &content.WeeklyContent{
				Year: 2026,
				Week: 5,
				Proposals: []content.ProposalContent{
					{
						IssueNumber:    12345,
						Title:          "proposal: test feature",
						PreviousStatus: parser.StatusDiscussions,
						CurrentStatus:  parser.StatusAccepted,
						ChangedAt:      changedAt,
						CommentURL:     "https://github.com/golang/go/issues/33502#issuecomment-123",
						Summary:        "This is a test summary",
						Links: []content.Link{
							{Title: "proposal issue", URL: "https://github.com/golang/go/issues/12345"},
							{Title: "related discussion", URL: "https://github.com/golang/go/issues/67890"},
						},
					},
				},
			},
			issueNumber: 12345,
			wantNil:     false,
			checkFunc: func(t *testing.T, data *templates.ProposalDetailData) {
				t.Helper()
				if data.IssueNumber != 12345 {
					t.Errorf("expected IssueNumber 12345, got %d", data.IssueNumber)
				}
				if data.Title != "proposal: test feature" {
					t.Errorf("expected Title 'proposal: test feature', got %q", data.Title)
				}
				if data.CurrentStatus != parser.StatusAccepted {
					t.Errorf("expected CurrentStatus accepted, got %s", data.CurrentStatus)
				}
				if data.PreviousStatus != parser.StatusDiscussions {
					t.Errorf("expected PreviousStatus discussions, got %s", data.PreviousStatus)
				}
				if data.Year != 2026 {
					t.Errorf("expected Year 2026, got %d", data.Year)
				}
				if data.Week != 5 {
					t.Errorf("expected Week 5, got %d", data.Week)
				}
				if data.IssueURL != "https://github.com/golang/go/issues/12345" {
					t.Errorf("expected IssueURL set correctly, got %q", data.IssueURL)
				}
				if data.CommentURL != "https://github.com/golang/go/issues/33502#issuecomment-123" {
					t.Errorf("expected CommentURL set correctly, got %q", data.CommentURL)
				}
				if len(data.Links) != 2 {
					t.Fatalf("expected 2 links, got %d", len(data.Links))
				}
				if data.Links[0].Title != "proposal issue" {
					t.Errorf("expected first link title 'proposal issue', got %q", data.Links[0].Title)
				}
				if !data.ChangedAt.Equal(changedAt) {
					t.Errorf("expected ChangedAt %v, got %v", changedAt, data.ChangedAt)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			data := templates.ConvertToProposalDetailData(tt.input, tt.issueNumber)

			if tt.wantNil {
				if data != nil {
					t.Errorf("expected nil, got %v", data)
				}
				return
			}

			if data == nil {
				t.Fatal("expected non-nil data")
			}

			if tt.checkFunc != nil {
				tt.checkFunc(t, data)
			}
		})
	}
}
