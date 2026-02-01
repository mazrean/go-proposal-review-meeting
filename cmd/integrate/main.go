// Package main provides content integration from changes.json and summaries.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"sort"

	"github.com/mazrean/go-proposal-review-meeting/internal/content"
	"github.com/mazrean/go-proposal-review-meeting/internal/parser"
)

// ChangesFile represents the structure of changes.json
type ChangesFile struct {
	Week    string               `json:"week"`
	Changes []parser.ProposalChange `json:"changes"`
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	changesPath := flag.String("changes", "changes.json", "Path to changes.json")
	contentDir := flag.String("content", "content", "Path to content directory")
	summariesDir := flag.String("summaries", "summaries", "Path to summaries directory")
	flag.Parse()

	// Read changes.json
	// Note: PreviousStatus is already set by the parse command based on
	// the proposal's status at the time of the immediately preceding comment.
	// Changes are already filtered to include only actual status changes.
	data, err := os.ReadFile(*changesPath)
	if err != nil {
		return fmt.Errorf("failed to read changes.json: %w", err)
	}

	var changesFile ChangesFile
	if err := json.Unmarshal(data, &changesFile); err != nil {
		return fmt.Errorf("failed to parse changes.json: %w", err)
	}

	fmt.Printf("Loaded %d changes from %s\n", len(changesFile.Changes), *changesPath)

	if len(changesFile.Changes) == 0 {
		fmt.Println("No changes to process")
		return nil
	}

	// Create content manager
	mgr := content.NewManager(
		content.WithBaseDir(*contentDir),
		content.WithSummariesDir(*summariesDir),
	)

	// Group changes by week
	weeklyChanges := groupByWeek(changesFile.Changes)
	fmt.Printf("Grouped into %d weeks\n", len(weeklyChanges))

	// Sort week keys to process in chronological order
	weekKeys := make([]string, 0, len(weeklyChanges))
	for key := range weeklyChanges {
		weekKeys = append(weekKeys, key)
	}
	sort.Strings(weekKeys)

	// Read summaries
	summaries, err := mgr.ReadSummaries()
	if err != nil {
		return fmt.Errorf("failed to read summaries: %w", err)
	}
	fmt.Printf("Loaded %d summaries\n", len(summaries))

	// Process each week in chronological order
	for _, weekKey := range weekKeys {
		changes := weeklyChanges[weekKey]
		fmt.Printf("Processing week %s with %d changes\n", weekKey, len(changes))

		// Deduplicate: keep the latest change for each issue within the week
		deduped := deduplicateByIssue(changes)
		if len(deduped) != len(changes) {
			fmt.Printf("  Deduplicated from %d to %d changes\n", len(changes), len(deduped))
		}

		// Prepare content
		weeklyContent := mgr.PrepareContent(deduped)

		// Integrate summaries
		if err := mgr.IntegrateSummaries(weeklyContent, summaries); err != nil {
			return fmt.Errorf("failed to integrate summaries: %w", err)
		}

		// Apply fallback for missing summaries
		if err := mgr.ApplyFallback(weeklyContent); err != nil {
			return fmt.Errorf("failed to apply fallback: %w", err)
		}

		// Write content with merge
		if err := mgr.WriteContentWithMerge(weeklyContent); err != nil {
			return fmt.Errorf("failed to write content: %w", err)
		}

		fmt.Printf("  Written %d proposals for week %d-W%02d\n",
			len(weeklyContent.Proposals), weeklyContent.Year, weeklyContent.Week)
	}

	fmt.Println("Content integration completed successfully!")
	return nil
}

// groupByWeek groups proposal changes by their ISO week
func groupByWeek(changes []parser.ProposalChange) map[string][]parser.ProposalChange {
	result := make(map[string][]parser.ProposalChange)

	for _, change := range changes {
		year, week := change.ChangedAt.ISOWeek()
		key := fmt.Sprintf("%d-W%02d", year, week)
		result[key] = append(result[key], change)
	}

	return result
}

// deduplicateByIssue keeps the latest change for each issue number
func deduplicateByIssue(changes []parser.ProposalChange) []parser.ProposalChange {
	issueMap := make(map[int]parser.ProposalChange)

	for _, change := range changes {
		if existing, ok := issueMap[change.IssueNumber]; ok {
			if change.ChangedAt.After(existing.ChangedAt) {
				issueMap[change.IssueNumber] = change
			}
		} else {
			issueMap[change.IssueNumber] = change
		}
	}

	result := make([]parser.ProposalChange, 0, len(issueMap))
	for _, change := range issueMap {
		result = append(result, change)
	}

	return result
}

