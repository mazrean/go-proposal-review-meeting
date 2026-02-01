// Package main provides the command-line interface for parsing proposal changes.
package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/mazrean/go-proposal-review-meeting/internal/parser"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	// Parse command-line flags
	statePath := flag.String("state", "content/state.json", "Path to the state file")
	changesPath := flag.String("output", "changes.json", "Path to output changes.json")
	token := flag.String("token", "", "GitHub API token (optional, can also be set via GITHUB_TOKEN env var)")
	flag.Parse()

	// Get token from environment if not provided via flag
	githubToken := *token
	if githubToken == "" {
		githubToken = os.Getenv("GITHUB_TOKEN")
	}

	// Setup context with signal handling
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	config := parseConfig{
		statePath:   *statePath,
		changesPath: *changesPath,
		baseURL:     "", // Use default GitHub API URL
		token:       githubToken,
		stdout:      os.Stdout,
	}

	return runParse(ctx, config)
}

// parseConfig holds configuration for the parse operation.
type parseConfig struct {
	statePath   string
	changesPath string
	baseURL     string
	token       string
	stdout      io.Writer
}

// runParse executes the parse operation and writes results.
func runParse(ctx context.Context, config parseConfig) error {
	// Create logger
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	// Create state manager
	stateManager := parser.NewStateManager(config.statePath)

	// Create issue parser
	parserConfig := parser.IssueParserConfig{
		StateManager: stateManager,
		Logger:       logger,
		BaseURL:      config.baseURL,
		Token:        config.token,
	}

	issueParser, err := parser.NewIssueParser(parserConfig)
	if err != nil {
		return fmt.Errorf("failed to create issue parser: %w", err)
	}

	// Fetch changes
	changes, err := issueParser.FetchChanges(ctx)
	if err != nil {
		return fmt.Errorf("failed to fetch changes: %w", err)
	}

	// Write changes to JSON file
	if err := issueParser.WriteChangesJSON(changes, config.changesPath); err != nil {
		return fmt.Errorf("failed to write changes: %w", err)
	}

	// Output has_changes flag for GitHub Actions
	hasChanges := len(changes) > 0
	fmt.Fprintf(config.stdout, "has_changes=%t\n", hasChanges)
	fmt.Fprintf(config.stdout, "changes_count=%d\n", len(changes))

	return nil
}
