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

func TestWeeklyIndex(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		weeklyData   templates.WeeklyData
		wantContains []string
	}{
		{
			name: "renders week header with year and week number",
			weeklyData: templates.WeeklyData{
				Year: 2026,
				Week: 5,
			},
			wantContains: []string{
				"2026年 第5週",
			},
		},
		{
			name: "renders proposal list",
			weeklyData: templates.WeeklyData{
				Year: 2026,
				Week: 5,
				Proposals: []templates.ProposalData{
					{
						IssueNumber:    12345,
						Title:          "proposal: add new feature",
						CurrentStatus:  parser.StatusAccepted,
						PreviousStatus: parser.StatusDiscussions,
						IssueURL:       "https://github.com/golang/go/issues/12345",
					},
				},
			},
			wantContains: []string{
				"#12345",
				"proposal: add new feature",
				"12345",
			},
		},
		{
			name: "renders multiple proposals",
			weeklyData: templates.WeeklyData{
				Year: 2026,
				Week: 5,
				Proposals: []templates.ProposalData{
					{
						IssueNumber:   12345,
						Title:         "first proposal",
						CurrentStatus: parser.StatusAccepted,
						IssueURL:      "https://github.com/golang/go/issues/12345",
					},
					{
						IssueNumber:   67890,
						Title:         "second proposal",
						CurrentStatus: parser.StatusDeclined,
						IssueURL:      "https://github.com/golang/go/issues/67890",
					},
				},
			},
			wantContains: []string{
				"first proposal",
				"second proposal",
				"#12345",
				"#67890",
			},
		},
		{
			name: "displays status change information",
			weeklyData: templates.WeeklyData{
				Year: 2026,
				Week: 5,
				Proposals: []templates.ProposalData{
					{
						IssueNumber:    12345,
						Title:          "proposal: test",
						PreviousStatus: parser.StatusDiscussions,
						CurrentStatus:  parser.StatusAccepted,
						IssueURL:       "https://github.com/golang/go/issues/12345",
					},
				},
			},
			wantContains: []string{
				"accepted",
			},
		},
		{
			name: "renders empty state when no proposals",
			weeklyData: templates.WeeklyData{
				Year:      2026,
				Week:      5,
				Proposals: nil,
			},
			wantContains: []string{
				"この週には更新がありません",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var buf bytes.Buffer
			err := templates.WeeklyIndex(tt.weeklyData).Render(context.Background(), &buf)
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

func TestWeeklyIndexPage(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		weeklyData   templates.WeeklyData
		wantContains []string
	}{
		{
			name: "renders full page with layout",
			weeklyData: templates.WeeklyData{
				Year: 2026,
				Week: 5,
				Proposals: []templates.ProposalData{
					{
						IssueNumber:   12345,
						Title:         "test proposal",
						CurrentStatus: parser.StatusAccepted,
						IssueURL:      "https://github.com/golang/go/issues/12345",
					},
				},
			},
			wantContains: []string{
				"<!doctype html>",
				"<title>",
				"2026年 第5週",
				"<header",
				"<nav",
				"<main",
				"<footer",
				"test proposal",
				"</html>",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var buf bytes.Buffer
			err := templates.WeeklyIndexPage(tt.weeklyData).Render(context.Background(), &buf)
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

func TestProposalListItem(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		proposal       templates.ProposalData
		wantContains   []string
		wantNotContain []string
	}{
		{
			name: "renders proposal with title and issue number",
			proposal: templates.ProposalData{
				IssueNumber:   12345,
				Title:         "proposal: add generics",
				CurrentStatus: parser.StatusAccepted,
				IssueURL:      "https://github.com/golang/go/issues/12345",
			},
			wantContains: []string{
				"#12345",
				"proposal: add generics",
				"https://github.com/golang/go/issues/12345",
			},
		},
		{
			name: "displays status badge",
			proposal: templates.ProposalData{
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
			name: "shows summary preview when available",
			proposal: templates.ProposalData{
				IssueNumber:   12345,
				Title:         "test",
				CurrentStatus: parser.StatusAccepted,
				Summary:       "This is a summary of the proposal change.",
				IssueURL:      "https://github.com/golang/go/issues/12345",
			},
			wantContains: []string{
				"This is a summary",
			},
		},
		{
			name: "renders detail link when DetailURL is set",
			proposal: templates.ProposalData{
				IssueNumber:   12345,
				Title:         "test",
				CurrentStatus: parser.StatusAccepted,
				IssueURL:      "https://github.com/golang/go/issues/12345",
				DetailURL:     "/2026/w05/12345.html",
			},
			wantContains: []string{
				"/2026/w05/12345.html",
				"詳細を見る",
			},
		},
		{
			name: "renders previous status arrow when status changed",
			proposal: templates.ProposalData{
				IssueNumber:    12345,
				Title:          "test",
				PreviousStatus: parser.StatusDiscussions,
				CurrentStatus:  parser.StatusAccepted,
				IssueURL:       "https://github.com/golang/go/issues/12345",
			},
			wantContains: []string{
				"discussions",
				"M13 7l5 5", // SVG arrow path
				"accepted",
			},
		},
		{
			name: "hides arrow when status unchanged",
			proposal: templates.ProposalData{
				IssueNumber:    12345,
				Title:          "test",
				PreviousStatus: parser.StatusAccepted,
				CurrentStatus:  parser.StatusAccepted,
				IssueURL:       "https://github.com/golang/go/issues/12345",
			},
			wantNotContain: []string{
				"M13 7l5 5", // SVG arrow path
			},
		},
		{
			name: "renders span instead of anchor when IssueURL empty",
			proposal: templates.ProposalData{
				IssueNumber:   12345,
				Title:         "test",
				CurrentStatus: parser.StatusAccepted,
				IssueURL:      "",
			},
			wantContains: []string{
				"#12345",
				"<span class=", // span instead of anchor
			},
			wantNotContain: []string{
				"https://github.com/golang/go/issues/12345",
				"<a href=", // should not be an anchor
			},
		},
		{
			name: "hides detail link when DetailURL empty",
			proposal: templates.ProposalData{
				IssueNumber:   12345,
				Title:         "test",
				CurrentStatus: parser.StatusAccepted,
				IssueURL:      "",
				DetailURL:     "",
			},
			wantNotContain: []string{
				"詳細を見る",
				`href=""`,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var buf bytes.Buffer
			err := templates.ProposalListItem(tt.proposal).Render(context.Background(), &buf)
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
		})
	}
}

func TestConvertToWeeklyData(t *testing.T) {
	t.Parallel()

	changedAt := time.Date(2026, 1, 30, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		input          *content.WeeklyContent
		checkProposals func(t *testing.T, proposals []templates.ProposalData)
		name           string
		wantYear       int
		wantWeek       int
		wantProposals  int
	}{
		{
			name:          "nil input returns empty data",
			input:         nil,
			wantYear:      0,
			wantWeek:      0,
			wantProposals: 0,
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
						},
					},
				},
			},
			wantYear:      2026,
			wantWeek:      5,
			wantProposals: 1,
			checkProposals: func(t *testing.T, proposals []templates.ProposalData) {
				t.Helper()
				p := proposals[0]
				if p.IssueNumber != 12345 {
					t.Errorf("expected IssueNumber 12345, got %d", p.IssueNumber)
				}
				if p.Title != "proposal: test feature" {
					t.Errorf("expected Title 'proposal: test feature', got %q", p.Title)
				}
				if p.CurrentStatus != parser.StatusAccepted {
					t.Errorf("expected CurrentStatus accepted, got %s", p.CurrentStatus)
				}
				if p.IssueURL != "https://github.com/golang/go/issues/12345" {
					t.Errorf("expected IssueURL set correctly, got %q", p.IssueURL)
				}
				if p.Summary != "This is a test summary" {
					t.Errorf("expected Summary preserved, got %q", p.Summary)
				}
			},
		},
		{
			name: "skips proposals with zero issue number",
			input: &content.WeeklyContent{
				Year: 2026,
				Week: 5,
				Proposals: []content.ProposalContent{
					{IssueNumber: 0, Title: "invalid", CurrentStatus: parser.StatusAccepted},
					{IssueNumber: 12345, Title: "valid", CurrentStatus: parser.StatusAccepted},
				},
			},
			wantYear:      2026,
			wantWeek:      5,
			wantProposals: 1,
			checkProposals: func(t *testing.T, proposals []templates.ProposalData) {
				t.Helper()
				if proposals[0].IssueNumber != 12345 {
					t.Errorf("expected valid proposal 12345, got %d", proposals[0].IssueNumber)
				}
			},
		},
		{
			name: "skips proposals with negative issue number",
			input: &content.WeeklyContent{
				Year: 2026,
				Week: 5,
				Proposals: []content.ProposalContent{
					{IssueNumber: -1, Title: "negative", CurrentStatus: parser.StatusAccepted},
					{IssueNumber: 12345, Title: "valid", CurrentStatus: parser.StatusAccepted},
				},
			},
			wantYear:      2026,
			wantWeek:      5,
			wantProposals: 1,
		},
		{
			name: "generates correct URLs for valid year/week",
			input: &content.WeeklyContent{
				Year: 2026,
				Week: 5,
				Proposals: []content.ProposalContent{
					{IssueNumber: 12345, Title: "test", CurrentStatus: parser.StatusAccepted},
				},
			},
			wantYear:      2026,
			wantWeek:      5,
			wantProposals: 1,
			checkProposals: func(t *testing.T, proposals []templates.ProposalData) {
				t.Helper()
				p := proposals[0]
				if p.IssueURL != "https://github.com/golang/go/issues/12345" {
					t.Errorf("expected IssueURL %q, got %q", "https://github.com/golang/go/issues/12345", p.IssueURL)
				}
				if p.DetailURL != "/2026/w05/12345.html" {
					t.Errorf("expected DetailURL %q, got %q", "/2026/w05/12345.html", p.DetailURL)
				}
			},
		},
		{
			name: "empty URLs for zero year",
			input: &content.WeeklyContent{
				Year: 0,
				Week: 5,
				Proposals: []content.ProposalContent{
					{IssueNumber: 12345, Title: "test", CurrentStatus: parser.StatusAccepted},
				},
			},
			wantYear:      0,
			wantWeek:      5,
			wantProposals: 1,
			checkProposals: func(t *testing.T, proposals []templates.ProposalData) {
				t.Helper()
				if proposals[0].IssueURL != "" {
					t.Errorf("expected empty IssueURL, got %q", proposals[0].IssueURL)
				}
				if proposals[0].DetailURL != "" {
					t.Errorf("expected empty DetailURL, got %q", proposals[0].DetailURL)
				}
			},
		},
		{
			name: "empty URLs for zero week",
			input: &content.WeeklyContent{
				Year: 2026,
				Week: 0,
				Proposals: []content.ProposalContent{
					{IssueNumber: 12345, Title: "test", CurrentStatus: parser.StatusAccepted},
				},
			},
			wantYear:      2026,
			wantWeek:      0,
			wantProposals: 1,
			checkProposals: func(t *testing.T, proposals []templates.ProposalData) {
				t.Helper()
				if proposals[0].IssueURL != "" {
					t.Errorf("expected empty IssueURL, got %q", proposals[0].IssueURL)
				}
			},
		},
		{
			name: "empty URLs for negative year",
			input: &content.WeeklyContent{
				Year: -1,
				Week: 5,
				Proposals: []content.ProposalContent{
					{IssueNumber: 12345, Title: "test", CurrentStatus: parser.StatusAccepted},
				},
			},
			wantYear:      -1,
			wantWeek:      5,
			wantProposals: 1,
			checkProposals: func(t *testing.T, proposals []templates.ProposalData) {
				t.Helper()
				if proposals[0].IssueURL != "" {
					t.Errorf("expected empty IssueURL, got %q", proposals[0].IssueURL)
				}
			},
		},
		{
			name: "empty URLs for negative week",
			input: &content.WeeklyContent{
				Year: 2026,
				Week: -1,
				Proposals: []content.ProposalContent{
					{IssueNumber: 12345, Title: "test", CurrentStatus: parser.StatusAccepted},
				},
			},
			wantYear:      2026,
			wantWeek:      -1,
			wantProposals: 1,
			checkProposals: func(t *testing.T, proposals []templates.ProposalData) {
				t.Helper()
				if proposals[0].IssueURL != "" {
					t.Errorf("expected empty IssueURL, got %q", proposals[0].IssueURL)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			data := templates.ConvertToWeeklyData(tt.input)

			if data.Year != tt.wantYear {
				t.Errorf("expected Year %d, got %d", tt.wantYear, data.Year)
			}

			if data.Week != tt.wantWeek {
				t.Errorf("expected Week %d, got %d", tt.wantWeek, data.Week)
			}

			if len(data.Proposals) != tt.wantProposals {
				t.Fatalf("expected %d proposals, got %d", tt.wantProposals, len(data.Proposals))
			}

			if tt.checkProposals != nil {
				tt.checkProposals(t, data.Proposals)
			}
		})
	}
}

func TestStatusBadge(t *testing.T) {
	t.Parallel()

	tests := []struct {
		status       parser.Status
		wantContains string
	}{
		{parser.StatusAccepted, "bg-green-100"},
		{parser.StatusDeclined, "bg-red-100"},
		{parser.StatusLikelyAccept, "bg-emerald-100"},
		{parser.StatusLikelyDecline, "bg-orange-100"},
		{parser.StatusHold, "bg-yellow-100"},
		{parser.StatusActive, "bg-sky-100"},
		{parser.StatusDiscussions, "bg-purple-100"},
	}

	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			t.Parallel()

			var buf bytes.Buffer
			err := templates.StatusBadge(tt.status).Render(context.Background(), &buf)
			if err != nil {
				t.Fatalf("failed to render: %v", err)
			}

			html := buf.String()

			if !strings.Contains(html, tt.wantContains) {
				t.Errorf("expected StatusBadge for %s to contain class %q, got:\n%s",
					tt.status, tt.wantContains, html)
			}
		})
	}
}
