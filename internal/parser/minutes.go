// Package parser provides functionality for parsing Go proposal review meeting minutes.
package parser

import (
	"log/slog"
	"strconv"
	"strings"
	"time"
)

// Constants for parsing.
const (
	// commentPreviewLength is the maximum length of comment preview in logs.
	commentPreviewLength = 100
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

// sectionHeaderPatterns maps section header keywords to their status.
// Section headers are lines like "**Active**" or "**Likely Accept**" that
// indicate the status of all proposals listed under them.
var sectionHeaderPatterns = []struct {
	keyword string
	status  Status
}{
	{"**accepted**", StatusAccepted},
	{"**declined**", StatusDeclined},
	{"**likely accept**", StatusLikelyAccept},
	{"**likely decline**", StatusLikelyDecline},
	{"**active**", StatusActive},
	{"**hold**", StatusHold},
	{"**discussions**", StatusDiscussions},
	{"**discussion**", StatusDiscussions},
}

// statusPatterns defines keywords for detecting status in indented action lines.
// These are used as fallback when no section header is present.
// Patterns are ordered by specificity - more specific patterns come first.
var statusPatterns = []struct {
	keywords       []string
	status         Status
	requiresEndPos bool // true if the last keyword must be at the end of the line
}{
	// Accepted patterns
	{[]string{"**no final comments; accepted"}, StatusAccepted, false},
	{[]string{"; **accepted**"}, StatusAccepted, false},
	{[]string{"**accepted** 🎉"}, StatusAccepted, false},
	{[]string{"**accepted**"}, StatusAccepted, false},
	{[]string{"accepted 🎉"}, StatusAccepted, false},

	// Declined patterns
	{[]string{"**no final comments; declined"}, StatusDeclined, false},
	{[]string{"; **declined**"}, StatusDeclined, false},
	{[]string{"**declined**"}, StatusDeclined, false},
	{[]string{"retracted", "**declined**"}, StatusDeclined, false},
	{[]string{"**closed**"}, StatusDeclined, false},

	// Likely accept/decline patterns
	{[]string{"**likely accept"}, StatusLikelyAccept, false},
	{[]string{"**likely decline"}, StatusLikelyDecline, false},

	// Hold patterns
	{[]string{"put on hold"}, StatusHold, false},
	{[]string{"on hold"}, StatusHold, true}, // Must be at end of line

	// Active patterns
	{[]string{"**active**"}, StatusActive, false},

	// Discussion patterns
	{[]string{"discussion ongoing"}, StatusDiscussions, false},
}

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

	// Find date header
	var meetingDate time.Time
	for _, line := range lines {
		dateStr := extractDateFromLine(line)
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
		if status, ok := detectSectionHeader(line); ok {
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
			currentSectionStatus = status
			continue
		}

		// Check if this is a new proposal line
		if issueNumber, title, ok := parseProposalLine(line); ok {
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

		// Fallback: Check for status keywords in indented lines when no section header exists
		// Only check indented lines (action lines under proposals)
		// Section headers like "**Accepted**" start at column 0, while action lines
		// are indented with "  - " prefix
		if currentProposal != nil && currentSectionStatus == "" && (strings.HasPrefix(line, "  ") || strings.HasPrefix(line, "\t")) {
			if status, ok := detectStatusInLine(line); ok {
				currentProposal.status = status
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

// extractDateFromLine extracts a date string (YYYY-MM-DD format) from a line.
// Supports two formats:
// - **YYYY-MM-DD** or **YYYY-MM-DD /
// - YYYY-MM-DD / **@name (without leading **)
func extractDateFromLine(line string) string {
	line = strings.TrimSpace(line)

	// Try format: **YYYY-MM-DD
	if strings.HasPrefix(line, "**") {
		line = strings.TrimPrefix(line, "**")
		if len(line) >= 10 && isDateFormat(line[:10]) {
			return line[:10]
		}
	}

	// Try format: YYYY-MM-DD /
	if len(line) >= 10 && isDateFormat(line[:10]) {
		if len(line) == 10 || (len(line) > 10 && strings.HasPrefix(line[10:], " /")) {
			return line[:10]
		}
	}

	return ""
}

// isDateFormat checks if a string matches YYYY-MM-DD format.
func isDateFormat(s string) bool {
	if len(s) != 10 {
		return false
	}
	if s[4] != '-' || s[7] != '-' {
		return false
	}
	for i, c := range s {
		if i == 4 || i == 7 {
			continue
		}
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}

// detectSectionHeader checks if a line is a section header and returns its status.
// Section headers are lines like "**Active**" or "**Likely Accept**".
func detectSectionHeader(line string) (Status, bool) {
	line = strings.TrimSpace(line)
	lineLower := strings.ToLower(line)

	// Section headers should start with ** and not be indented
	if !strings.HasPrefix(lineLower, "**") {
		return "", false
	}

	for _, sh := range sectionHeaderPatterns {
		if strings.HasPrefix(lineLower, sh.keyword) {
			return sh.status, true
		}
	}

	return "", false
}

// parseProposalLine parses a proposal line and extracts the issue number and title.
// Supports multiple formats:
// - [#NNNNN](URL) **title**
// - #NNNNN **title**
// - **title** [#NNNNN](URL)
func parseProposalLine(line string) (issueNumber int, title string, ok bool) {
	line = strings.TrimSpace(line)

	// Must start with "- "
	if !strings.HasPrefix(line, "- ") {
		return 0, "", false
	}
	line = strings.TrimPrefix(line, "- ")

	// Try format: [#NNNNN](URL) **title** or #NNNNN **title**
	if strings.HasPrefix(line, "[#") || strings.HasPrefix(line, "#") {
		var numStr string
		var rest string

		if strings.HasPrefix(line, "[#") {
			// Format: [#NNNNN](URL) **title**
			line = strings.TrimPrefix(line, "[#")
			closeBracketIdx := strings.Index(line, "]")
			if closeBracketIdx == -1 {
				return 0, "", false
			}
			numStr = line[:closeBracketIdx]

			// Skip the (URL) part
			parenIdx := strings.Index(line[closeBracketIdx:], ")")
			if parenIdx == -1 {
				return 0, "", false
			}
			rest = strings.TrimSpace(line[closeBracketIdx+parenIdx+1:])
		} else {
			// Format: #NNNNN **title**
			line = strings.TrimPrefix(line, "#")
			spaceIdx := strings.Index(line, " ")
			if spaceIdx == -1 {
				return 0, "", false
			}
			numStr = line[:spaceIdx]
			rest = strings.TrimSpace(line[spaceIdx+1:])
		}

		// Extract title from **title**
		if !strings.HasPrefix(rest, "**") {
			return 0, "", false
		}
		rest = strings.TrimPrefix(rest, "**")
		endIdx := strings.Index(rest, "**")
		if endIdx == -1 {
			return 0, "", false
		}
		title = rest[:endIdx]

		issueNumber, err := strconv.Atoi(numStr)
		if err != nil {
			return 0, "", false
		}

		return issueNumber, title, true
	}

	// Try format: **title** [#NNNNN](URL)
	if after, ok0 := strings.CutPrefix(line, "**"); ok0 {
		line = after
		before, after0, ok1 := strings.Cut(line, "**")
		if !ok1 {
			return 0, "", false
		}
		title = before
		rest := strings.TrimSpace(after0)

		// Extract issue number from [#NNNNN](URL)
		if !strings.HasPrefix(rest, "[#") {
			return 0, "", false
		}
		rest = strings.TrimPrefix(rest, "[#")
		closeBracketIdx := strings.Index(rest, "]")
		if closeBracketIdx == -1 {
			return 0, "", false
		}
		numStr := rest[:closeBracketIdx]

		issueNumber, err := strconv.Atoi(numStr)
		if err != nil {
			return 0, "", false
		}

		return issueNumber, title, true
	}

	return 0, "", false
}

// detectStatusInLine detects status keywords in an indented line.
// This is used as fallback when no section header is present.
// Returns the detected status and true if found, otherwise returns empty status and false.
func detectStatusInLine(line string) (Status, bool) {
	lineLower := strings.ToLower(strings.TrimSpace(line))

	// Check patterns in order (most specific first)
	for _, sp := range statusPatterns {
		allMatch := true
		for _, keyword := range sp.keywords {
			if !strings.Contains(lineLower, strings.ToLower(keyword)) {
				allMatch = false
				break
			}
		}
		if !allMatch {
			continue
		}

		// If the pattern requires end position, check that the last keyword is at the end
		if sp.requiresEndPos && len(sp.keywords) > 0 {
			lastKeyword := strings.ToLower(sp.keywords[len(sp.keywords)-1])
			if !strings.HasSuffix(lineLower, lastKeyword) {
				continue
			}
		}

		return sp.status, true
	}

	return "", false
}
