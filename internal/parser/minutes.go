// Package parser provides functionality for parsing Go proposal review meeting minutes.
package parser

import (
	"log/slog"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Constants for parsing.
const (
	// commentPreviewLength is the maximum length of comment preview in logs.
	commentPreviewLength = 100

	// proposalRe1MatchGroups is the expected number of match groups for proposalRe1.
	// Groups: full match, issue number (link), issue number (plain), title
	proposalRe1MatchGroups = 4

	// proposalRe2MatchGroups is the expected number of match groups for proposalRe2.
	// Groups: full match, title, issue number
	proposalRe2MatchGroups = 3
)

// Status represents the status of a Go proposal.
type Status string

const (
	StatusDiscussions   Status = "discussions"
	StatusLikelyAccept  Status = "likely_accept"
	StatusLikelyDecline Status = "likely_decline"
	StatusAccepted      Status = "accepted"
	StatusDeclined      Status = "declined"
	StatusHold          Status = "hold"
	StatusActive        Status = "active"
)

// ProposalChange represents a detected status change for a proposal.
type ProposalChange struct {
	ChangedAt      time.Time `json:"changed_at"`
	Title          string    `json:"title"`
	PreviousStatus Status    `json:"previous_status"`
	CurrentStatus  Status    `json:"current_status"`
	CommentURL     string    `json:"comment_url"`
	RelatedIssues  []int     `json:"related_issues"`
	IssueNumber    int       `json:"issue_number"`
}

// Pre-compiled regular expressions for parsing.
var (
	// dateHeaderRe matches date header formats:
	// - **YYYY-MM-DD** or **YYYY-MM-DD /
	// - YYYY-MM-DD / **@name (without leading **)
	dateHeaderRe1 = regexp.MustCompile(`^\*\*(\d{4}-\d{2}-\d{2})`)
	dateHeaderRe2 = regexp.MustCompile(`^(\d{4}-\d{2}-\d{2})\s*/`)

	// proposalRe1 matches proposal lines with issue number first:
	// - #NNNNN **title**
	// - [#NNNNN](URL) **title**
	proposalRe1 = regexp.MustCompile(`^- (?:\[#(\d+)\]\([^)]+\)|#(\d+)) \*\*(.+?)\*\*`)

	// proposalRe2 matches proposal lines with title first:
	// - **title** [#NNNNN](URL)
	proposalRe2 = regexp.MustCompile(`^- \*\*(.+?)\*\* \[#(\d+)\]\([^)]+\)`)

	// sectionHeaderPatterns maps section header patterns to their default status.
	// Section headers are lines like "**Active**" or "**Likely Accept**" that
	// indicate the status of all proposals listed under them.
	sectionHeaderPatterns = []struct {
		re     *regexp.Regexp
		status Status
	}{
		{regexp.MustCompile(`(?i)^\*\*Accepted\*\*`), StatusAccepted},
		{regexp.MustCompile(`(?i)^\*\*Declined\*\*`), StatusDeclined},
		{regexp.MustCompile(`(?i)^\*\*Likely Accept\*\*`), StatusLikelyAccept},
		{regexp.MustCompile(`(?i)^\*\*Likely Decline\*\*`), StatusLikelyDecline},
		{regexp.MustCompile(`(?i)^\*\*Active\*\*`), StatusActive},
		{regexp.MustCompile(`(?i)^\*\*Hold\*\*`), StatusHold},
		{regexp.MustCompile(`(?i)^\*\*Discussions?\*\*`), StatusDiscussions},
	}

	// Status detection patterns ordered by specificity.
	// More specific patterns should come before general ones.
	statusPatterns = []struct {
		re     *regexp.Regexp
		status Status
	}{
		// Accepted patterns
		{regexp.MustCompile(`(?i)\*\*no final comments; accepted`), StatusAccepted},
		{regexp.MustCompile(`(?i);\s*\*\*accepted\*\*`), StatusAccepted},
		{regexp.MustCompile(`(?i)\*\*accepted\*\*\s*🎉`), StatusAccepted},
		{regexp.MustCompile(`(?i)\*\*accepted\*\*`), StatusAccepted},
		{regexp.MustCompile(`(?i)accepted\s*🎉`), StatusAccepted},

		// Declined patterns
		{regexp.MustCompile(`(?i)\*\*no final comments; declined`), StatusDeclined},
		{regexp.MustCompile(`(?i);\s*\*\*declined\*\*`), StatusDeclined},
		{regexp.MustCompile(`(?i)\*\*declined\*\*`), StatusDeclined},
		{regexp.MustCompile(`(?i)retracted.*\*\*declined\*\*`), StatusDeclined},
		{regexp.MustCompile(`(?i)\*\*closed\*\*`), StatusDeclined},

		// Likely accept/decline patterns (may or may not have closing **)
		{regexp.MustCompile(`(?i)\*\*likely accept`), StatusLikelyAccept},
		{regexp.MustCompile(`(?i)\*\*likely decline`), StatusLikelyDecline},

		// Hold patterns
		{regexp.MustCompile(`(?i)put on hold`), StatusHold},
		{regexp.MustCompile(`(?i)on hold$`), StatusHold},

		// Active patterns
		{regexp.MustCompile(`(?i)\*\*active\*\*`), StatusActive},

		// Discussion patterns
		{regexp.MustCompile(`(?i)discussion ongoing`), StatusDiscussions},
	}
)

// MinutesParser parses proposal review meeting minutes.
type MinutesParser struct {
	logger *slog.Logger
}

// NewMinutesParser creates a new MinutesParser.
func NewMinutesParser() *MinutesParser {
	return &MinutesParser{
		logger: slog.Default(),
	}
}

// NewMinutesParserWithLogger creates a new MinutesParser with a custom logger.
func NewMinutesParserWithLogger(logger *slog.Logger) *MinutesParser {
	return &MinutesParser{
		logger: logger,
	}
}

// Parse extracts proposal changes from a meeting minutes comment.
// Returns an empty slice (not nil) when no changes are found.
func (p *MinutesParser) Parse(comment string, commentedAt time.Time) ([]ProposalChange, error) {
	if comment == "" {
		return []ProposalChange{}, nil
	}

	lines := strings.Split(comment, "\n")

	// Find date header (try both formats)
	var meetingDate time.Time
	for _, line := range lines {
		var dateStr string

		// Try format: **YYYY-MM-DD...
		if matches := dateHeaderRe1.FindStringSubmatch(line); len(matches) > 1 {
			dateStr = matches[1]
		} else if matches := dateHeaderRe2.FindStringSubmatch(line); len(matches) > 1 {
			// Try format: YYYY-MM-DD / **@name
			dateStr = matches[1]
		}

		if dateStr != "" {
			var err error
			meetingDate, err = time.Parse("2006-01-02", dateStr)
			if err != nil {
				p.logger.Warn("invalid date format in minutes header",
					"line", line,
					"error", err)
				continue
			}
			break
		}
	}

	// If no valid date header, this is not a valid minutes format
	if meetingDate.IsZero() {
		p.logger.Warn("no valid date header found in comment",
			"comment_preview", truncate(comment, commentPreviewLength))
		return []ProposalChange{}, nil
	}

	changes := []ProposalChange{}
	var currentProposal *proposalContext
	var currentSectionStatus Status // Track the current section's default status

	for _, line := range lines {
		// Check for section headers (e.g., "**Active**", "**Likely Accept**")
		// These determine the default status for proposals listed under them
		for _, sh := range sectionHeaderPatterns {
			if sh.re.MatchString(line) {
				// Save previous proposal before changing section
				if currentProposal != nil && currentProposal.status != "" {
					changes = append(changes, ProposalChange{
						IssueNumber:   currentProposal.issueNumber,
						Title:         currentProposal.title,
						CurrentStatus: currentProposal.status,
						ChangedAt:     meetingDate,
					})
					currentProposal = nil
				}
				currentSectionStatus = sh.status
				break
			}
		}

		// Check if this is a new proposal line
		// Try both formats: issue number first, or title first
		var issueNumber int
		var title string
		var matched bool

		if matches := proposalRe1.FindStringSubmatch(line); len(matches) >= proposalRe1MatchGroups {
			// Format: - [#NNNNN](URL) **title** or - #NNNNN **title**
			// Issue number is in group 1 (link format) or group 2 (plain format)
			issueNumStr := matches[1]
			if issueNumStr == "" {
				issueNumStr = matches[2]
			}
			issueNumber, _ = strconv.Atoi(issueNumStr)
			title = matches[3]
			matched = true
		} else if matches := proposalRe2.FindStringSubmatch(line); len(matches) >= proposalRe2MatchGroups {
			// Format: - **title** [#NNNNN](URL)
			title = matches[1]
			issueNumber, _ = strconv.Atoi(matches[2])
			matched = true
		}

		if matched {
			// Save previous proposal if it had a status change
			if currentProposal != nil && currentProposal.status != "" {
				changes = append(changes, ProposalChange{
					IssueNumber:   currentProposal.issueNumber,
					Title:         currentProposal.title,
					CurrentStatus: currentProposal.status,
					ChangedAt:     meetingDate,
				})
			}

			currentProposal = &proposalContext{
				issueNumber: issueNumber,
				title:       title,
				status:      currentSectionStatus, // Use section's default status
			}
			continue
		}

		// Check for status changes in the current proposal's actions
		// Only check indented lines (action lines under proposals), not section headers
		// Section headers like "**Accepted**" start at column 0, while action lines
		// are indented with "  - " prefix
		if currentProposal != nil && (strings.HasPrefix(line, "  ") || strings.HasPrefix(line, "\t")) {
			for _, sp := range statusPatterns {
				if sp.re.MatchString(line) {
					currentProposal.status = sp.status
					break
				}
			}
		}
	}

	// Don't forget the last proposal
	if currentProposal != nil && currentProposal.status != "" {
		changes = append(changes, ProposalChange{
			IssueNumber:   currentProposal.issueNumber,
			Title:         currentProposal.title,
			CurrentStatus: currentProposal.status,
			ChangedAt:     meetingDate,
		})
	}

	return changes, nil
}

// truncate returns the first n characters of s, or s if shorter.
func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}

type proposalContext struct {
	title       string
	status      Status
	issueNumber int
}
