package templates_test

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/mazrean/go-proposal-review-meeting/internal/site/templates"
)

func TestHomeContent(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		homeData     templates.HomeData
		wantContains []string
	}{
		{
			name: "renders page title",
			homeData: templates.HomeData{
				Weeks: nil,
			},
			wantContains: []string{
				"週次アーカイブ",
			},
		},
		{
			name: "renders empty state when no weeks",
			homeData: templates.HomeData{
				Weeks: nil,
			},
			wantContains: []string{
				"更新がありません",
			},
		},
		{
			name: "renders week list",
			homeData: templates.HomeData{
				Weeks: []templates.WeekSummary{
					{
						Year:          2026,
						Week:          5,
						ProposalCount: 3,
						URL:           "/2026/w05/",
					},
				},
			},
			wantContains: []string{
				"2026年 第5週",
				"3件",
				"/2026/w05/",
			},
		},
		{
			name: "renders multiple weeks sorted by date (newest first)",
			homeData: templates.HomeData{
				Weeks: []templates.WeekSummary{
					{
						Year:          2026,
						Week:          6,
						ProposalCount: 2,
						URL:           "/2026/w06/",
					},
					{
						Year:          2026,
						Week:          5,
						ProposalCount: 3,
						URL:           "/2026/w05/",
					},
				},
			},
			wantContains: []string{
				"2026年 第6週",
				"2026年 第5週",
				"2件",
				"3件",
			},
		},
		{
			name: "renders week with singular proposal count",
			homeData: templates.HomeData{
				Weeks: []templates.WeekSummary{
					{
						Year:          2026,
						Week:          5,
						ProposalCount: 1,
						URL:           "/2026/w05/",
					},
				},
			},
			wantContains: []string{
				"1件",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var buf bytes.Buffer
			err := templates.HomeContent(tt.homeData).Render(context.Background(), &buf)
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

func TestHomePage(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		homeData     templates.HomeData
		wantContains []string
	}{
		{
			name: "renders full page with layout",
			homeData: templates.HomeData{
				Weeks: []templates.WeekSummary{
					{
						Year:          2026,
						Week:          5,
						ProposalCount: 3,
						URL:           "/2026/w05/",
					},
				},
			},
			wantContains: []string{
				"<!doctype html>",
				"<title>",
				"Go Proposal Weekly Digest",
				"<header",
				"<nav",
				"<main",
				"<footer",
				"2026年 第5週",
				"</html>",
			},
		},
		{
			name: "includes RSS autodiscovery link",
			homeData: templates.HomeData{
				Weeks: nil,
			},
			wantContains: []string{
				`rel="alternate"`,
				`type="application/rss+xml"`,
				`href="/feed.xml"`,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var buf bytes.Buffer
			err := templates.HomePage(tt.homeData).Render(context.Background(), &buf)
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

func TestWeekCard(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		weekSummary    templates.WeekSummary
		wantContains   []string
		wantNotContain []string
	}{
		{
			name: "renders week card with year and week",
			weekSummary: templates.WeekSummary{
				Year:          2026,
				Week:          5,
				ProposalCount: 3,
				URL:           "/2026/w05/",
			},
			wantContains: []string{
				"2026年 第5週",
				"/2026/w05/",
			},
		},
		{
			name: "renders proposal count",
			weekSummary: templates.WeekSummary{
				Year:          2026,
				Week:          5,
				ProposalCount: 10,
				URL:           "/2026/w05/",
			},
			wantContains: []string{
				"10件",
			},
		},
		{
			name: "renders link to week page",
			weekSummary: templates.WeekSummary{
				Year:          2026,
				Week:          5,
				ProposalCount: 3,
				URL:           "/2026/w05/",
			},
			wantContains: []string{
				"href=\"/2026/w05/\"",
				"詳細を見る",
			},
		},
		{
			name: "handles zero proposal count",
			weekSummary: templates.WeekSummary{
				Year:          2026,
				Week:          5,
				ProposalCount: 0,
				URL:           "/2026/w05/",
			},
			wantContains: []string{
				"0件",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var buf bytes.Buffer
			err := templates.WeekCard(tt.weekSummary).Render(context.Background(), &buf)
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

func TestConvertToHomeData(t *testing.T) {
	t.Parallel()

	tests := []struct {
		checkWeeks func(t *testing.T, weeks []templates.WeekSummary)
		name       string
		weeks      []templates.WeeklyData
		wantWeeks  int
	}{
		{
			name:      "nil input returns empty data",
			weeks:     nil,
			wantWeeks: 0,
		},
		{
			name:      "empty slice returns empty data",
			weeks:     []templates.WeeklyData{},
			wantWeeks: 0,
		},
		{
			name: "converts single week",
			weeks: []templates.WeeklyData{
				{
					Year: 2026,
					Week: 5,
					Proposals: []templates.ProposalData{
						{IssueNumber: 12345},
						{IssueNumber: 67890},
					},
				},
			},
			wantWeeks: 1,
			checkWeeks: func(t *testing.T, weeks []templates.WeekSummary) {
				t.Helper()
				if weeks[0].Year != 2026 {
					t.Errorf("expected Year 2026, got %d", weeks[0].Year)
				}
				if weeks[0].Week != 5 {
					t.Errorf("expected Week 5, got %d", weeks[0].Week)
				}
				if weeks[0].ProposalCount != 2 {
					t.Errorf("expected ProposalCount 2, got %d", weeks[0].ProposalCount)
				}
				if weeks[0].URL != "/2026/w05/" {
					t.Errorf("expected URL /2026/w05/, got %s", weeks[0].URL)
				}
			},
		},
		{
			name: "converts multiple weeks",
			weeks: []templates.WeeklyData{
				{
					Year:      2026,
					Week:      6,
					Proposals: []templates.ProposalData{{IssueNumber: 1}},
				},
				{
					Year:      2026,
					Week:      5,
					Proposals: []templates.ProposalData{{IssueNumber: 2}, {IssueNumber: 3}},
				},
			},
			wantWeeks: 2,
			checkWeeks: func(t *testing.T, weeks []templates.WeekSummary) {
				t.Helper()
				// First week should be week 6 (newest)
				if weeks[0].Week != 6 {
					t.Errorf("expected first week to be 6, got %d", weeks[0].Week)
				}
				if weeks[0].ProposalCount != 1 {
					t.Errorf("expected first week ProposalCount 1, got %d", weeks[0].ProposalCount)
				}
				// Second week should be week 5
				if weeks[1].Week != 5 {
					t.Errorf("expected second week to be 5, got %d", weeks[1].Week)
				}
				if weeks[1].ProposalCount != 2 {
					t.Errorf("expected second week ProposalCount 2, got %d", weeks[1].ProposalCount)
				}
			},
		},
		{
			name: "generates correct URLs for different weeks",
			weeks: []templates.WeeklyData{
				{Year: 2026, Week: 1, Proposals: []templates.ProposalData{{IssueNumber: 1}}},
				{Year: 2025, Week: 52, Proposals: []templates.ProposalData{{IssueNumber: 2}}},
			},
			wantWeeks: 2,
			checkWeeks: func(t *testing.T, weeks []templates.WeekSummary) {
				t.Helper()
				// Week 1 of 2026 should be first (newest)
				if weeks[0].URL != "/2026/w01/" {
					t.Errorf("expected URL /2026/w01/, got %s", weeks[0].URL)
				}
				if weeks[1].URL != "/2025/w52/" {
					t.Errorf("expected URL /2025/w52/, got %s", weeks[1].URL)
				}
			},
		},
		{
			name: "week with no proposals has count zero",
			weeks: []templates.WeeklyData{
				{Year: 2026, Week: 5, Proposals: nil},
			},
			wantWeeks: 1,
			checkWeeks: func(t *testing.T, weeks []templates.WeekSummary) {
				t.Helper()
				if weeks[0].ProposalCount != 0 {
					t.Errorf("expected ProposalCount 0, got %d", weeks[0].ProposalCount)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			data := templates.ConvertToHomeData(tt.weeks)

			if len(data.Weeks) != tt.wantWeeks {
				t.Fatalf("expected %d weeks, got %d", tt.wantWeeks, len(data.Weeks))
			}

			if tt.checkWeeks != nil {
				tt.checkWeeks(t, data.Weeks)
			}
		})
	}
}

func TestLatestWeekHighlight(t *testing.T) {
	t.Parallel()

	homeData := templates.HomeData{
		Weeks: []templates.WeekSummary{
			{Year: 2026, Week: 6, ProposalCount: 2, URL: "/2026/w06/"},
			{Year: 2026, Week: 5, ProposalCount: 3, URL: "/2026/w05/"},
		},
	}

	var buf bytes.Buffer
	err := templates.HomeContent(homeData).Render(context.Background(), &buf)
	if err != nil {
		t.Fatalf("failed to render: %v", err)
	}

	html := buf.String()

	// The latest week should be highlighted or shown prominently
	// We expect some indicator for the most recent update
	if !strings.Contains(html, "最新") {
		t.Errorf("expected HTML to contain 最新 (latest) indicator, got:\n%s", html)
	}
}
