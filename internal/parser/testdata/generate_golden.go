//go:build ignore

package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/mazrean/go-proposal-review-meeting/internal/parser"
)

type Comment struct {
	ID        int64  `json:"id"`
	CreatedAt string `json:"created_at"`
	Body      string `json:"body"`
}

type GoldenEntry struct {
	CommentID   int64                  `json:"comment_id"`
	CommentedAt string                 `json:"commented_at"`
	Changes     []parser.ProposalChange `json:"changes"`
}

func main() {
	// Read comments from stdin (piped from gh api)
	var comments []Comment
	decoder := json.NewDecoder(os.Stdin)
	for decoder.More() {
		var c Comment
		if err := decoder.Decode(&c); err != nil {
			fmt.Fprintf(os.Stderr, "Error decoding comment: %v\n", err)
			os.Exit(1)
		}
		comments = append(comments, c)
	}

	fmt.Fprintf(os.Stderr, "Processing %d comments...\n", len(comments))

	p := parser.NewMinutesParser()
	var entries []GoldenEntry

	for _, c := range comments {
		commentedAt, err := time.Parse(time.RFC3339, c.CreatedAt)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error parsing date for comment %d: %v\n", c.ID, err)
			continue
		}

		changes, err := p.Parse(c.Body, commentedAt)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error parsing comment %d: %v\n", c.ID, err)
			continue
		}

		// Sort changes by issue number for deterministic output
		sort.Slice(changes, func(i, j int) bool {
			return changes[i].IssueNumber < changes[j].IssueNumber
		})

		entries = append(entries, GoldenEntry{
			CommentID:   c.ID,
			CommentedAt: c.CreatedAt,
			Changes:     changes,
		})
	}

	// Write golden file
	output, err := json.MarshalIndent(entries, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error marshaling golden data: %v\n", err)
		os.Exit(1)
	}

	goldenPath := filepath.Join("testdata", "golden.json")
	if err := os.WriteFile(goldenPath, output, 0644); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing golden file: %v\n", err)
		os.Exit(1)
	}

	// Also write individual comment files for reference
	commentsDir := filepath.Join("testdata", "comments")
	os.MkdirAll(commentsDir, 0755)

	for _, c := range comments {
		commentPath := filepath.Join(commentsDir, fmt.Sprintf("%d.txt", c.ID))
		if err := os.WriteFile(commentPath, []byte(c.Body), 0644); err != nil {
			fmt.Fprintf(os.Stderr, "Error writing comment file %d: %v\n", c.ID, err)
		}
	}

	fmt.Fprintf(os.Stderr, "Generated golden.json with %d entries\n", len(entries))

	// Print summary
	totalChanges := 0
	for _, e := range entries {
		totalChanges += len(e.Changes)
	}
	fmt.Fprintf(os.Stderr, "Total status changes detected: %d\n", totalChanges)
}
