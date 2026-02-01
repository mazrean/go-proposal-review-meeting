// Package site provides functionality for generating the static site.
package site

import (
	"context"
	"fmt"
	"html"
	"sort"
	"strings"
	"time"

	"github.com/gopherlibs/feedhub/feedhub"
	"github.com/mazrean/go-proposal-review-meeting/internal/content"
)

// MaxFeedItems is the maximum number of weekly items to include in the RSS feed.
const MaxFeedItems = 20

// FeedGenerator handles RSS feed generation.
type FeedGenerator struct {
	siteURL     string
	siteTitle   string
	siteDesc    string
	authorName  string
	authorEmail string
}

// FeedOption is a functional option for configuring FeedGenerator.
type FeedOption func(*FeedGenerator)

// WithSiteURL sets the site URL for the feed.
func WithSiteURL(url string) FeedOption {
	return func(fg *FeedGenerator) {
		fg.siteURL = url
	}
}

// WithSiteTitle sets the site title for the feed.
func WithSiteTitle(title string) FeedOption {
	return func(fg *FeedGenerator) {
		fg.siteTitle = title
	}
}

// WithSiteDescription sets the site description for the feed.
func WithSiteDescription(desc string) FeedOption {
	return func(fg *FeedGenerator) {
		fg.siteDesc = desc
	}
}

// WithAuthor sets the author information for the feed.
func WithAuthor(name, email string) FeedOption {
	return func(fg *FeedGenerator) {
		fg.authorName = name
		fg.authorEmail = email
	}
}

// NewFeedGenerator creates a new FeedGenerator with the given options.
func NewFeedGenerator(opts ...FeedOption) *FeedGenerator {
	fg := &FeedGenerator{
		siteURL:     "https://example.com",
		siteTitle:   "Go Proposal Weekly Digest",
		siteDesc:    "Go言語のproposal review meeting minutesの週次要約",
		authorName:  "Go Proposal Digest",
		authorEmail: "",
	}
	for _, opt := range opts {
		opt(fg)
	}
	return fg
}

// GenerateFeed generates an RSS 2.0 feed from the given weekly contents.
// It limits the output to the most recent MaxFeedItems weeks.
func (fg *FeedGenerator) GenerateFeed(ctx context.Context, weeks []*content.WeeklyContent) ([]byte, error) {
	// Check for context cancellation
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	now := time.Now()

	feed := &feedhub.Feed{
		Title:       fg.siteTitle,
		Link:        &feedhub.Link{Href: fg.siteURL},
		Description: fg.siteDesc,
		Created:     now,
		Updated:     now,
	}

	// Only set Author if email is provided (RSS 2.0 requires valid email format)
	if fg.authorEmail != "" {
		feed.Author = &feedhub.Author{Name: fg.authorName, Email: fg.authorEmail}
	}

	if weeks == nil || len(weeks) == 0 {
		return fg.renderFeed(feed)
	}

	// Sort weeks by date (newest first)
	sortedWeeks := make([]*content.WeeklyContent, len(weeks))
	copy(sortedWeeks, weeks)
	sort.Slice(sortedWeeks, func(i, j int) bool {
		if sortedWeeks[i].Year != sortedWeeks[j].Year {
			return sortedWeeks[i].Year > sortedWeeks[j].Year
		}
		return sortedWeeks[i].Week > sortedWeeks[j].Week
	})

	// Limit to MaxFeedItems
	limit := len(sortedWeeks)
	if limit > MaxFeedItems {
		limit = MaxFeedItems
	}

	items := make([]*feedhub.Item, 0, limit)
	for i := range limit {
		week := sortedWeeks[i]
		if week == nil {
			continue
		}

		// Check for context cancellation
		if err := ctx.Err(); err != nil {
			return nil, err
		}

		item := fg.weekToFeedItem(week)
		items = append(items, item)
	}

	feed.Items = items

	return fg.renderFeed(feed)
}

// weekToFeedItem converts a WeeklyContent to a feed item.
func (fg *FeedGenerator) weekToFeedItem(week *content.WeeklyContent) *feedhub.Item {
	title := fmt.Sprintf("%d年 第%d週 - Go Proposal 更新", week.Year, week.Week)
	link := fmt.Sprintf("%s/%d/w%02d/", fg.siteURL, week.Year, week.Week)
	guid := fmt.Sprintf("%s/%d/w%02d", fg.siteURL, week.Year, week.Week)

	description := fg.buildDescription(week)

	// Use the latest proposal's changed time, or created time
	pubDate := week.CreatedAt
	for _, p := range week.Proposals {
		if p.ChangedAt.After(pubDate) {
			pubDate = p.ChangedAt
		}
	}

	item := &feedhub.Item{
		Title:       title,
		Link:        &feedhub.Link{Href: link},
		Description: description,
		Created:     pubDate,
		Updated:     pubDate,
		Id:          guid,
	}

	// Only set Author if email is provided (RSS 2.0 requires valid email format)
	if fg.authorEmail != "" {
		item.Author = &feedhub.Author{Name: fg.authorName, Email: fg.authorEmail}
	}

	return item
}

// buildDescription builds the description HTML for a weekly digest.
func (fg *FeedGenerator) buildDescription(week *content.WeeklyContent) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("<p>%d年 第%d週のGo Proposal更新情報</p>", week.Year, week.Week))

	if len(week.Proposals) == 0 {
		sb.WriteString("<p>今週の更新はありません。</p>")
		return sb.String()
	}

	sb.WriteString("<ul>")
	for _, p := range week.Proposals {
		sb.WriteString("<li>")
		sb.WriteString(fmt.Sprintf("<strong>#%d</strong>: %s", p.IssueNumber, escapeHTML(p.Title)))
		sb.WriteString(fmt.Sprintf(" (<code>%s</code> → <code>%s</code>)", p.PreviousStatus, p.CurrentStatus))
		if p.Summary != "" {
			sb.WriteString("<br/>")
			// Truncate summary if too long (rune-aware to handle multibyte characters)
			summary := truncateRunes(p.Summary, 200)
			sb.WriteString(escapeHTML(summary))
		}
		sb.WriteString("</li>")
	}
	sb.WriteString("</ul>")

	return sb.String()
}

// escapeHTML escapes special HTML characters using the standard library.
func escapeHTML(s string) string {
	return html.EscapeString(s)
}

// truncateRunes truncates a string to the specified number of runes.
// This is safe for multibyte characters (e.g., Japanese text).
// If truncation occurs, "..." is appended.
func truncateRunes(s string, maxRunes int) string {
	runes := []rune(s)
	if len(runes) <= maxRunes {
		return s
	}
	// Leave room for "..." (3 characters)
	if maxRunes <= 3 {
		return "..."
	}
	return string(runes[:maxRunes-3]) + "..."
}

// renderFeed renders the feed to RSS 2.0 XML bytes.
func (fg *FeedGenerator) renderFeed(feed *feedhub.Feed) ([]byte, error) {
	rss, err := feed.ToRss()
	if err != nil {
		return nil, fmt.Errorf("failed to generate RSS: %w", err)
	}
	return []byte(rss), nil
}
