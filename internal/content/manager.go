// Package content provides functionality for managing weekly proposal digest content.
package content

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/mazrean/go-proposal-review-meeting/internal/parser"
)

// Link represents a related link for a proposal.
type Link struct {
	Title string `yaml:"title"`
	URL   string `yaml:"url"`
}

// ProposalContent represents the content for a single proposal.
type ProposalContent struct {
	IssueNumber    int           `yaml:"issue_number"`
	Title          string        `yaml:"title"`
	PreviousStatus parser.Status `yaml:"previous_status"`
	CurrentStatus  parser.Status `yaml:"current_status"`
	ChangedAt      time.Time     `yaml:"changed_at"`
	CommentURL     string        `yaml:"comment_url"`
	Summary        string        `yaml:"-"` // Not in frontmatter, in body
	Links          []Link        `yaml:"related_issues"`
}

// WeeklyContent represents the content for a single week.
type WeeklyContent struct {
	Year      int
	Week      int
	Proposals []ProposalContent
	CreatedAt time.Time
}

// Manager handles the creation and management of weekly content.
type Manager struct {
	baseDir string
}

// Option is a functional option for configuring Manager.
type Option func(*Manager)

// WithBaseDir sets the base directory for content storage.
func WithBaseDir(dir string) Option {
	return func(m *Manager) {
		m.baseDir = dir
	}
}

// NewManager creates a new content Manager with the given options.
func NewManager(opts ...Option) *Manager {
	m := &Manager{
		baseDir: "content",
	}
	for _, opt := range opts {
		opt(m)
	}
	return m
}

// PrepareContent creates a WeeklyContent structure from the given proposal changes.
func (m *Manager) PrepareContent(changes []parser.ProposalChange) *WeeklyContent {
	if len(changes) == 0 {
		return &WeeklyContent{
			Year:      0,
			Week:      0,
			Proposals: nil,
			CreatedAt: time.Time{},
		}
	}

	// Use the first change's date to determine the year and week
	year, week := changes[0].ChangedAt.ISOWeek()

	proposals := make([]ProposalContent, len(changes))
	for i, change := range changes {
		links := make([]Link, 0, 1+len(change.RelatedIssues))

		// Add main proposal link
		links = append(links, Link{
			Title: "proposal issue",
			URL:   fmt.Sprintf("https://github.com/golang/go/issues/%d", change.IssueNumber),
		})

		// Add related issue links
		for _, relatedIssue := range change.RelatedIssues {
			links = append(links, Link{
				Title: "related discussion",
				URL:   fmt.Sprintf("https://github.com/golang/go/issues/%d", relatedIssue),
			})
		}

		proposals[i] = ProposalContent{
			IssueNumber:    change.IssueNumber,
			Title:          change.Title,
			PreviousStatus: change.PreviousStatus,
			CurrentStatus:  change.CurrentStatus,
			ChangedAt:      change.ChangedAt,
			CommentURL:     change.CommentURL,
			Summary:        "",
			Links:          links,
		}
	}

	return &WeeklyContent{
		Year:      year,
		Week:      week,
		Proposals: proposals,
		CreatedAt: time.Now(),
	}
}

// WriteContent writes the weekly content to the filesystem.
func (m *Manager) WriteContent(content *WeeklyContent) error {
	if content == nil || len(content.Proposals) == 0 {
		return nil
	}

	// Create the directory
	dirPath := filepath.Join(m.baseDir, weekDirPath(content.Year, content.Week))
	if err := os.MkdirAll(dirPath, 0o755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dirPath, err)
	}

	// Write each proposal file
	for _, proposal := range content.Proposals {
		filename := proposalFilename(proposal.IssueNumber)
		filePath := filepath.Join(dirPath, filename)

		fileContent := generateMarkdown(proposal)
		if err := os.WriteFile(filePath, []byte(fileContent), 0o644); err != nil {
			return fmt.Errorf("failed to write file %s: %w", filePath, err)
		}
	}

	return nil
}

// weekDirPath returns the directory path for the given year and week.
func weekDirPath(year, week int) string {
	return fmt.Sprintf("%d/W%02d", year, week)
}

// proposalFilename returns the filename for the given issue number.
func proposalFilename(issueNumber int) string {
	return fmt.Sprintf("proposal-%d.md", issueNumber)
}

// generateMarkdown generates the markdown content for a proposal.
func generateMarkdown(p ProposalContent) string {
	var b strings.Builder

	// Frontmatter
	b.WriteString("---\n")
	fmt.Fprintf(&b, "issue_number: %d\n", p.IssueNumber)
	fmt.Fprintf(&b, "title: %q\n", p.Title)
	fmt.Fprintf(&b, "previous_status: %s\n", p.PreviousStatus)
	fmt.Fprintf(&b, "current_status: %s\n", p.CurrentStatus)
	fmt.Fprintf(&b, "changed_at: %s\n", p.ChangedAt.UTC().Format(time.RFC3339))
	fmt.Fprintf(&b, "comment_url: %s\n", p.CommentURL)

	b.WriteString("related_issues:\n")
	for _, link := range p.Links {
		fmt.Fprintf(&b, "  - title: %q\n", link.Title)
		fmt.Fprintf(&b, "    url: %s\n", link.URL)
	}

	b.WriteString("---\n")

	// Body section
	b.WriteString("\n## 要約\n\n")
	if p.Summary != "" {
		b.WriteString(p.Summary)
		b.WriteString("\n")
	}

	b.WriteString("\n## 関連リンク\n\n")
	for _, link := range p.Links {
		fmt.Fprintf(&b, "- [%s](%s)\n", link.Title, link.URL)
	}

	return b.String()
}
