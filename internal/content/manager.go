// Package content provides functionality for managing weekly proposal digest content.
package content

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/mazrean/go-proposal-review-meeting/internal/parser"
)

// File permission constants.
const (
	// dirPerm is the permission mode for created directories.
	dirPerm = 0o755
	// filePerm is the permission mode for created files.
	filePerm = 0o644
)

// regexMatchMinGroups is the minimum number of groups expected from regex matches.
const regexMatchMinGroups = 3

// Link represents a related link for a proposal.
type Link struct {
	Title string `yaml:"title"`
	URL   string `yaml:"url"`
}

// ProposalContent represents the content for a single proposal.
type ProposalContent struct {
	ChangedAt      time.Time     `yaml:"changed_at"`
	Title          string        `yaml:"title"`
	PreviousStatus parser.Status `yaml:"previous_status"`
	CurrentStatus  parser.Status `yaml:"current_status"`
	CommentURL     string        `yaml:"comment_url"`
	Summary        string        `yaml:"-"`
	Links          []Link        `yaml:"related_issues"`
	IssueNumber    int           `yaml:"issue_number"`
}

// WeeklyContent represents the content for a single week.
type WeeklyContent struct {
	CreatedAt time.Time
	Proposals []ProposalContent
	Year      int
	Week      int
}

// Manager handles the creation and management of weekly content.
type Manager struct {
	baseDir      string
	summariesDir string
}

// Option is a functional option for configuring Manager.
type Option func(*Manager)

// WithBaseDir sets the base directory for content storage.
func WithBaseDir(dir string) Option {
	return func(m *Manager) {
		m.baseDir = dir
	}
}

// WithSummariesDir sets the directory for AI-generated summary files.
func WithSummariesDir(dir string) Option {
	return func(m *Manager) {
		m.summariesDir = dir
	}
}

// NewManager creates a new content Manager with the given options.
func NewManager(opts ...Option) *Manager {
	m := &Manager{
		baseDir:      "content",
		summariesDir: "summaries",
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
	if err := os.MkdirAll(dirPath, dirPerm); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dirPath, err)
	}

	// Write each proposal file
	for _, proposal := range content.Proposals {
		filename := proposalFilename(proposal.IssueNumber)
		filePath := filepath.Join(dirPath, filename)

		fileContent := generateMarkdown(proposal)
		if err := os.WriteFile(filePath, []byte(fileContent), filePerm); err != nil {
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

// MergeContent merges new content into existing content for the same week.
// If existing is nil, returns the new content as-is.
// For proposals that exist in both, it updates the status while preserving
// the original previous_status and summary (if new summary is empty).
func (m *Manager) MergeContent(existing, newContent *WeeklyContent) *WeeklyContent {
	if newContent == nil {
		return existing
	}
	if existing == nil {
		return newContent
	}

	// Create a map of existing proposals by issue number
	proposalMap := make(map[int]ProposalContent)
	for _, p := range existing.Proposals {
		proposalMap[p.IssueNumber] = p
	}

	// Merge new proposals
	for _, newProposal := range newContent.Proposals {
		if existingProposal, ok := proposalMap[newProposal.IssueNumber]; ok {
			// Update existing proposal
			merged := mergeProposal(existingProposal, newProposal)
			proposalMap[newProposal.IssueNumber] = merged
		} else {
			// Add new proposal
			proposalMap[newProposal.IssueNumber] = newProposal
		}
	}

	// Convert map back to slice
	proposals := make([]ProposalContent, 0, len(proposalMap))
	for _, p := range proposalMap {
		proposals = append(proposals, p)
	}

	return &WeeklyContent{
		Year:      newContent.Year,
		Week:      newContent.Week,
		Proposals: proposals,
		CreatedAt: existing.CreatedAt, // Preserve original creation time
	}
}

// mergeProposal merges two proposals for the same issue.
// Preserves original previous_status and summary (if new is empty).
// Updates current_status and merges links.
func mergeProposal(existing, newProposal ProposalContent) ProposalContent {
	merged := ProposalContent{
		IssueNumber:    newProposal.IssueNumber,
		Title:          newProposal.Title,
		PreviousStatus: existing.PreviousStatus, // Preserve original previous status
		CurrentStatus:  newProposal.CurrentStatus,
		ChangedAt:      newProposal.ChangedAt,
		CommentURL:     newProposal.CommentURL,
		Summary:        newProposal.Summary,
		Links:          mergeLinks(existing.Links, newProposal.Links),
	}

	// Preserve existing summary if new one is empty
	if merged.Summary == "" && existing.Summary != "" {
		merged.Summary = existing.Summary
	}

	return merged
}

// mergeLinks merges two link slices, deduplicating by URL.
func mergeLinks(existing, newLinks []Link) []Link {
	urlMap := make(map[string]Link)

	// Add existing links first
	for _, link := range existing {
		urlMap[link.URL] = link
	}

	// Add/update with new links
	for _, link := range newLinks {
		urlMap[link.URL] = link
	}

	// Convert back to slice
	result := make([]Link, 0, len(urlMap))
	for _, link := range urlMap {
		result = append(result, link)
	}

	return result
}

// ReadExistingContent reads existing content for the given year and week.
// Returns nil if no content exists for the specified week.
func (m *Manager) ReadExistingContent(year, week int) (*WeeklyContent, error) {
	dirPath := filepath.Join(m.baseDir, weekDirPath(year, week))

	// Check if directory exists
	if _, err := os.Stat(dirPath); os.IsNotExist(err) {
		return nil, nil
	}

	// Read all proposal files in the directory
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory %s: %w", dirPath, err)
	}

	proposals := make([]ProposalContent, 0)
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasPrefix(entry.Name(), "proposal-") || !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}

		filePath := filepath.Join(dirPath, entry.Name())
		proposal, err := parseProposalFile(filePath)
		if err != nil {
			return nil, fmt.Errorf("failed to parse proposal file %s: %w", filePath, err)
		}

		proposals = append(proposals, *proposal)
	}

	if len(proposals) == 0 {
		return nil, nil
	}

	return &WeeklyContent{
		Year:      year,
		Week:      week,
		Proposals: proposals,
		CreatedAt: time.Time{}, // Will be set from file if needed
	}, nil
}

// parseProposalFile parses a proposal markdown file and returns its content.
func parseProposalFile(filePath string) (proposal *ProposalContent, err error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer func() {
		if closeErr := file.Close(); closeErr != nil && err == nil {
			err = closeErr
		}
	}()

	scanner := bufio.NewScanner(file)
	var p ProposalContent
	var inFrontmatter bool
	var inBody bool
	var bodyBuilder strings.Builder
	var currentLinkTitle string

	issueRe := regexp.MustCompile(`^issue_number:\s*(\d+)`)
	titleRe := regexp.MustCompile(`^title:\s*"(.+)"`)
	prevStatusRe := regexp.MustCompile(`^previous_status:\s*(\w+)`)
	currStatusRe := regexp.MustCompile(`^current_status:\s*(\w+)`)
	changedAtRe := regexp.MustCompile(`^changed_at:\s*(.+)`)
	commentURLRe := regexp.MustCompile(`^comment_url:\s*(.+)`)
	linkTitleRe := regexp.MustCompile(`^\s*-\s*title:\s*"(.+)"`)
	linkURLRe := regexp.MustCompile(`^\s*url:\s*(.+)`)

	for scanner.Scan() {
		line := scanner.Text()
		// Handle CRLF line endings (e.g., Windows files)
		line = strings.TrimSuffix(line, "\r")

		if line == "---" {
			if !inFrontmatter {
				inFrontmatter = true
				continue
			} else {
				inFrontmatter = false
				inBody = true
				continue
			}
		}

		if inFrontmatter {
			if m := issueRe.FindStringSubmatch(line); m != nil {
				issueNum, parseErr := strconv.Atoi(m[1])
				if parseErr != nil {
					return nil, fmt.Errorf("failed to parse issue_number: %w", parseErr)
				}
				p.IssueNumber = issueNum
			} else if m := titleRe.FindStringSubmatch(line); m != nil {
				p.Title = m[1]
			} else if m := prevStatusRe.FindStringSubmatch(line); m != nil {
				p.PreviousStatus = parser.Status(m[1])
			} else if m := currStatusRe.FindStringSubmatch(line); m != nil {
				p.CurrentStatus = parser.Status(m[1])
			} else if m := changedAtRe.FindStringSubmatch(line); m != nil {
				changedAt, parseErr := time.Parse(time.RFC3339, m[1])
				if parseErr != nil {
					return nil, fmt.Errorf("failed to parse changed_at: %w", parseErr)
				}
				p.ChangedAt = changedAt
			} else if m := commentURLRe.FindStringSubmatch(line); m != nil {
				p.CommentURL = m[1]
			} else if m := linkTitleRe.FindStringSubmatch(line); m != nil {
				currentLinkTitle = m[1]
			} else if m := linkURLRe.FindStringSubmatch(line); m != nil {
				if currentLinkTitle == "" {
					return nil, fmt.Errorf("link URL found without preceding title: %s", m[1])
				}
				p.Links = append(p.Links, Link{
					Title: currentLinkTitle,
					URL:   m[1],
				})
				currentLinkTitle = ""
			}
		} else if inBody {
			// Extract summary from body (between ## 要約 and ## 関連リンク)
			if strings.HasPrefix(line, "## 要約") {
				continue
			}
			if strings.HasPrefix(line, "## 関連リンク") {
				break
			}
			if strings.TrimSpace(line) != "" {
				if bodyBuilder.Len() > 0 {
					bodyBuilder.WriteString("\n")
				}
				bodyBuilder.WriteString(line)
			}
		}
	}

	p.Summary = strings.TrimSpace(bodyBuilder.String())

	if scanErr := scanner.Err(); scanErr != nil {
		return nil, scanErr
	}

	// Validate required fields
	if p.IssueNumber == 0 {
		return nil, fmt.Errorf("missing required field: issue_number")
	}
	if p.Title == "" {
		return nil, fmt.Errorf("missing required field: title")
	}
	if p.PreviousStatus == "" {
		return nil, fmt.Errorf("missing required field: previous_status")
	}
	if p.CurrentStatus == "" {
		return nil, fmt.Errorf("missing required field: current_status")
	}
	if p.ChangedAt.IsZero() {
		return nil, fmt.Errorf("missing required field: changed_at")
	}
	if p.CommentURL == "" {
		return nil, fmt.Errorf("missing required field: comment_url")
	}

	return &p, nil
}

// WriteContentWithMerge writes content, merging with any existing content for the same week.
// Past week data is not modified.
func (m *Manager) WriteContentWithMerge(content *WeeklyContent) error {
	if content == nil || len(content.Proposals) == 0 {
		return nil
	}

	// Read existing content for the same week
	existing, err := m.ReadExistingContent(content.Year, content.Week)
	if err != nil {
		return fmt.Errorf("failed to read existing content: %w", err)
	}

	// Merge with existing content
	merged := m.MergeContent(existing, content)

	// Write merged content
	return m.WriteContent(merged)
}

// IntegrateSummaries integrates AI-generated summaries into the content.
// It also extracts any GitHub issue links from the summaries and adds them to the Links.
// The "関連リンク" section is stripped from summaries to avoid duplication with the auto-generated section.
func (m *Manager) IntegrateSummaries(content *WeeklyContent, summaries map[int]string) error {
	if content == nil {
		return nil
	}

	for i := range content.Proposals {
		issueNumber := content.Proposals[i].IssueNumber
		summary, ok := summaries[issueNumber]
		if !ok || summary == "" {
			continue
		}

		// Extract links from the summary before stripping the section
		extractedLinks := extractLinksFromMarkdown(summary)
		content.Proposals[i].Links = mergeLinks(content.Proposals[i].Links, extractedLinks)

		// Strip the "関連リンク" section from the summary to avoid duplication
		summary = stripRelatedLinksSection(summary)
		content.Proposals[i].Summary = summary
	}

	return nil
}

// stripRelatedLinksSection removes the "関連リンク" section from markdown text.
// This prevents duplication since generateMarkdown adds its own related links section.
func stripRelatedLinksSection(text string) string {
	// Find the "## 関連リンク" header and remove everything from there to the end
	// or until the next ## header
	lines := strings.Split(text, "\n")
	var result []string
	inRelatedLinks := false

	for _, line := range lines {
		// Check if this is the start of the related links section
		if strings.HasPrefix(line, "## 関連リンク") {
			inRelatedLinks = true
			continue
		}

		// Check if we've hit another section header (exit related links section)
		if inRelatedLinks && strings.HasPrefix(line, "## ") {
			inRelatedLinks = false
		}

		if !inRelatedLinks {
			result = append(result, line)
		}
	}

	return strings.TrimSpace(strings.Join(result, "\n"))
}

// ApplyFallback applies fallback text to proposals that have no summary.
// The fallback contains basic information: proposal number, title, and status change.
func (m *Manager) ApplyFallback(content *WeeklyContent) error {
	if content == nil {
		return nil
	}

	for i := range content.Proposals {
		if content.Proposals[i].Summary != "" {
			continue
		}

		p := content.Proposals[i]
		content.Proposals[i].Summary = generateFallbackSummary(p)
	}

	return nil
}

// ReadSummaries reads all summary files from the summaries directory.
// Returns a map of issue number to summary content.
func (m *Manager) ReadSummaries() (map[int]string, error) {
	summaries := make(map[int]string)

	// Check if directory exists
	if _, err := os.Stat(m.summariesDir); os.IsNotExist(err) {
		return summaries, nil
	}

	entries, err := os.ReadDir(m.summariesDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read summaries directory %s: %w", m.summariesDir, err)
	}

	summaryFileRe := regexp.MustCompile(`^(\d+)\.md$`)

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		matches := summaryFileRe.FindStringSubmatch(entry.Name())
		if matches == nil {
			continue
		}

		issueNumber, err := strconv.Atoi(matches[1])
		if err != nil {
			continue
		}

		filePath := filepath.Join(m.summariesDir, entry.Name())
		data, err := os.ReadFile(filePath)
		if err != nil {
			return nil, fmt.Errorf("failed to read summary file %s: %w", filePath, err)
		}

		summaries[issueNumber] = strings.TrimSpace(string(data))
	}

	return summaries, nil
}

// extractLinksFromMarkdown extracts markdown links from text.
// It looks for patterns like [text](url) and returns them as Links.
// Supports GitHub issue URLs with optional #issuecomment anchors.
func extractLinksFromMarkdown(text string) []Link {
	// Match GitHub issue URLs with optional #issuecomment-NNNN anchors
	linkRe := regexp.MustCompile(`\[([^\]]+)\]\((https://github\.com/golang/go/issues/\d+(?:#issuecomment-\d+)?)\)`)
	matches := linkRe.FindAllStringSubmatch(text, -1)

	links := make([]Link, 0, len(matches))
	for _, match := range matches {
		if len(match) >= regexMatchMinGroups {
			links = append(links, Link{
				Title: match[1],
				URL:   match[2],
			})
		}
	}

	return links
}

// generateFallbackSummary generates a fallback summary when AI summary is not available.
func generateFallbackSummary(p ProposalContent) string {
	return fmt.Sprintf(
		"Proposal #%d「%s」のステータスが %s から %s に変更されました。",
		p.IssueNumber,
		p.Title,
		p.PreviousStatus,
		p.CurrentStatus,
	)
}

// SummaryMinLength is the minimum recommended length for AI-generated summaries.
const SummaryMinLength = 200

// SummaryMaxLength is the maximum recommended length for AI-generated summaries.
const SummaryMaxLength = 500

// ValidateSummaryLength checks if the summary length is within the recommended range (200-500 characters).
// Returns true if valid, false otherwise, along with a reason string.
// This function is provided for external validation (e.g., in workflows or CLI tools).
// It is not called automatically during content integration to allow flexibility.
func ValidateSummaryLength(summary string) (bool, string) {
	length := utf8.RuneCountInString(summary)

	if length < SummaryMinLength {
		return false, fmt.Sprintf("summary too short: %d characters (minimum: %d)", length, SummaryMinLength)
	}

	if length > SummaryMaxLength {
		return false, fmt.Sprintf("summary too long: %d characters (maximum: %d)", length, SummaryMaxLength)
	}

	return true, ""
}

// ListAllWeeks scans the content directory and returns all available weekly contents.
// It reads the directory structure (content/YYYY/WXX/) and parses all proposal files.
// Returns a slice of WeeklyContent sorted by date (newest first).
func (m *Manager) ListAllWeeks() ([]*WeeklyContent, error) {
	// Check if base directory exists
	if _, err := os.Stat(m.baseDir); os.IsNotExist(err) {
		return nil, nil
	}

	// Read year directories
	yearEntries, err := os.ReadDir(m.baseDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read base directory %s: %w", m.baseDir, err)
	}

	yearRe := regexp.MustCompile(`^(\d{4})$`)
	weekRe := regexp.MustCompile(`^W(\d{2})$`)

	var weeks []*WeeklyContent

	for _, yearEntry := range yearEntries {
		if !yearEntry.IsDir() {
			continue
		}

		// Parse year from directory name
		yearMatches := yearRe.FindStringSubmatch(yearEntry.Name())
		if yearMatches == nil {
			continue
		}

		year, err := strconv.Atoi(yearMatches[1])
		if err != nil {
			continue
		}

		// Read week directories for this year
		yearPath := filepath.Join(m.baseDir, yearEntry.Name())
		weekEntries, err := os.ReadDir(yearPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read year directory %s: %w", yearPath, err)
		}

		for _, weekEntry := range weekEntries {
			if !weekEntry.IsDir() {
				continue
			}

			// Parse week from directory name
			weekMatches := weekRe.FindStringSubmatch(weekEntry.Name())
			if weekMatches == nil {
				continue
			}

			week, err := strconv.Atoi(weekMatches[1])
			if err != nil {
				continue
			}

			// Read the weekly content
			content, err := m.ReadExistingContent(year, week)
			if err != nil {
				return nil, fmt.Errorf("failed to read content for %d-W%02d: %w", year, week, err)
			}
			if content == nil {
				continue
			}

			weeks = append(weeks, content)
		}
	}

	// Sort by date (newest first)
	sort.Slice(weeks, func(i, j int) bool {
		if weeks[i].Year != weeks[j].Year {
			return weeks[i].Year > weeks[j].Year
		}
		return weeks[i].Week > weeks[j].Week
	})

	return weeks, nil
}
