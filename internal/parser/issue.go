package parser

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"slices"
	"strconv"
	"time"
)

// changesFileMode is the file permission for changes.json output.
const changesFileMode fs.FileMode = 0644

// GitHub API constants.
const (
	// ProposalReviewIssueNumber is the issue number for the proposal review minutes.
	ProposalReviewIssueNumber = 33502

	// defaultBaseURL is the default GitHub API base URL.
	defaultBaseURL = "https://api.github.com"

	// perPage is the number of comments to fetch per request.
	perPage = 100

	// httpClientTimeout is the timeout for HTTP requests.
	httpClientTimeout = 30 * time.Second
)

// ErrNilStateManager is returned when StateManager is nil.
var ErrNilStateManager = errors.New("StateManager is required")

// IssueParserConfig holds configuration for IssueParser.
type IssueParserConfig struct {
	StateManager *StateManager
	Logger       *slog.Logger
	BaseURL      string
	Token        string
}

// IssueParser fetches and parses proposal changes from GitHub issue comments.
type IssueParser struct {
	stateManager  *StateManager
	minutesParser *MinutesParser
	logger        *slog.Logger
	httpClient    *http.Client
	baseURL       string
	token         string
	etag          string
}

// GitHubComment represents a GitHub issue comment.
type GitHubComment struct {
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Body      string    `json:"body"`
	HTMLURL   string    `json:"html_url"`
	ID        int64     `json:"id"`
}

// ChangesOutput is the JSON output format for changes.
type ChangesOutput struct {
	Week    string           `json:"week"`
	Changes []ProposalChange `json:"changes"`
}

// NewIssueParser creates a new IssueParser with the given configuration.
// Returns an error if StateManager is nil.
func NewIssueParser(config IssueParserConfig) (*IssueParser, error) {
	if config.StateManager == nil {
		return nil, ErrNilStateManager
	}

	baseURL := config.BaseURL
	if baseURL == "" {
		baseURL = defaultBaseURL
	}

	logger := config.Logger
	if logger == nil {
		logger = slog.Default()
	}

	return &IssueParser{
		stateManager:  config.StateManager,
		minutesParser: NewMinutesParserWithLogger(logger),
		baseURL:       baseURL,
		token:         config.Token,
		logger:        logger,
		httpClient:    &http.Client{Timeout: httpClientTimeout},
	}, nil
}

// FetchChanges fetches proposal changes since the last processed comment.
// It returns all detected proposal status changes from new comments.
func (ip *IssueParser) FetchChanges(ctx context.Context) ([]ProposalChange, error) {
	if ip.stateManager == nil {
		return nil, ErrNilStateManager
	}

	// Load current state
	state, err := ip.stateManager.LoadState()
	if err != nil {
		ip.logger.Error("failed to load state", "error", err)
		return nil, fmt.Errorf("failed to load state: %w", err)
	}

	ip.logger.Info("fetching comments since last processed",
		"since", state.LastProcessedAt,
		"lastCommentId", state.LastCommentID)

	// Fetch comments from GitHub API
	comments, err := ip.fetchComments(ctx, state.LastProcessedAt)
	if err != nil {
		ip.logger.Error("failed to fetch comments", "error", err)
		return nil, fmt.Errorf("failed to fetch comments: %w", err)
	}

	// Filter out already processed comments using both timestamp and ID
	// GitHub's 'since' parameter uses updated_at, so we filter by UpdatedAt
	lastCommentID, _ := strconv.ParseInt(state.LastCommentID, 10, 64)
	var newComments []GitHubComment
	for _, c := range comments {
		// Use UpdatedAt for filtering since GitHub API's 'since' is based on updated_at
		effectiveTime := c.UpdatedAt
		if effectiveTime.IsZero() {
			effectiveTime = c.CreatedAt
		}

		// Skip if comment is older than last processed
		if effectiveTime.Before(state.LastProcessedAt) {
			continue
		}
		// Skip if comment has same timestamp but ID <= lastCommentID (already processed)
		if effectiveTime.Equal(state.LastProcessedAt) && c.ID <= lastCommentID {
			continue
		}
		newComments = append(newComments, c)
	}

	ip.logger.Info("found new comments", "count", len(newComments))

	if len(newComments) == 0 {
		return []ProposalChange{}, nil
	}

	// Parse each comment for proposal changes
	var allChanges []ProposalChange
	var latestCommentID int64
	var latestTime time.Time

	for _, comment := range newComments {
		changes, err := ip.minutesParser.Parse(comment.Body, comment.CreatedAt)
		if err != nil {
			ip.logger.Warn("failed to parse comment",
				"commentId", comment.ID,
				"error", err)
			continue
		}

		// Add comment URL to each change
		for i := range changes {
			changes[i].CommentURL = comment.HTMLURL
		}

		allChanges = append(allChanges, changes...)

		// Track the latest comment for state update (by UpdatedAt, then by ID)
		effectiveTime := comment.UpdatedAt
		if effectiveTime.IsZero() {
			effectiveTime = comment.CreatedAt
		}
		if effectiveTime.After(latestTime) ||
			(effectiveTime.Equal(latestTime) && comment.ID > latestCommentID) {
			latestTime = effectiveTime
			latestCommentID = comment.ID
		}
	}

	// Update state with the latest processed comment
	if latestCommentID != 0 {
		if err := ip.stateManager.UpdateState(latestTime, strconv.FormatInt(latestCommentID, 10)); err != nil {
			ip.logger.Error("failed to update state", "error", err)
			return nil, fmt.Errorf("failed to update state: %w", err)
		}
	}

	ip.logger.Info("extracted proposal changes", "count", len(allChanges))

	return allChanges, nil
}

// fetchComments retrieves comments from the GitHub API with pagination.
func (ip *IssueParser) fetchComments(ctx context.Context, since time.Time) ([]GitHubComment, error) {
	var allComments []GitHubComment
	page := 1

	for {
		comments, hasMore, err := ip.fetchCommentsPage(ctx, since, page)
		if err != nil {
			return nil, err
		}

		allComments = append(allComments, comments...)

		if !hasMore {
			break
		}
		page++
	}

	return allComments, nil
}

// fetchCommentsPage retrieves a single page of comments.
func (ip *IssueParser) fetchCommentsPage(ctx context.Context, since time.Time, page int) ([]GitHubComment, bool, error) {
	url := fmt.Sprintf("%s/repos/golang/go/issues/%d/comments?per_page=%d&page=%d&since=%s",
		ip.baseURL, ProposalReviewIssueNumber, perPage, page, since.Format(time.RFC3339))

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, false, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")

	if ip.token != "" {
		req.Header.Set("Authorization", "Bearer "+ip.token)
	}

	// Add ETag header for caching
	if ip.etag != "" && page == 1 {
		req.Header.Set("If-None-Match", ip.etag)
	}

	resp, err := ip.httpClient.Do(req)
	if err != nil {
		return nil, false, fmt.Errorf("failed to execute request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	// Handle 304 Not Modified (cached response)
	if resp.StatusCode == http.StatusNotModified {
		return []GitHubComment{}, false, nil
	}

	// Handle error responses
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, false, fmt.Errorf("GitHub API error: status=%d body=%s", resp.StatusCode, string(body))
	}

	// Store ETag for future requests
	if etag := resp.Header.Get("ETag"); etag != "" && page == 1 {
		ip.etag = etag
	}

	// Parse response
	var comments []GitHubComment
	if err := json.NewDecoder(resp.Body).Decode(&comments); err != nil {
		return nil, false, fmt.Errorf("failed to decode response: %w", err)
	}

	// Check if there are more pages
	hasMore := len(comments) == perPage

	return comments, hasMore, nil
}

// WriteChangesJSON writes the changes to a JSON file.
// Changes are sorted by ChangedAt for deterministic output.
func (ip *IssueParser) WriteChangesJSON(changes []ProposalChange, path string) error {
	// Sort changes by ChangedAt for deterministic output
	sortedChanges := make([]ProposalChange, len(changes))
	copy(sortedChanges, changes)
	slices.SortFunc(sortedChanges, func(a, b ProposalChange) int {
		return a.ChangedAt.Compare(b.ChangedAt)
	})

	// Determine the week string from the latest change
	var year, weekNum int
	if len(sortedChanges) > 0 {
		year, weekNum = sortedChanges[len(sortedChanges)-1].ChangedAt.ISOWeek()
	} else {
		year, weekNum = time.Now().ISOWeek()
	}
	week := fmt.Sprintf("%d-W%02d", year, weekNum)

	output := ChangesOutput{
		Week:    week,
		Changes: sortedChanges, // Use sorted changes for deterministic output
	}

	data, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal changes: %w", err)
	}

	if err := os.WriteFile(path, data, changesFileMode); err != nil {
		return fmt.Errorf("failed to write changes file: %w", err)
	}

	ip.logger.Info("wrote changes to file",
		"path", path,
		"changeCount", len(changes))

	return nil
}
