// Package main provides the command-line interface for the static site generator.
package main

import (
	"context"
	"flag"
	"fmt"
	"net/url"
	"os"

	"github.com/mazrean/go-proposal-review-meeting/internal/content"
	"github.com/mazrean/go-proposal-review-meeting/internal/site"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	// Parse command-line flags
	contentDir := flag.String("content", "content", "Directory containing content files")
	distDir := flag.String("dist", "dist", "Output directory for generated files")
	siteURL := flag.String("site-url", "https://example.com", "Site URL for RSS feed generation")
	flag.Parse()

	// Validate flags
	if *contentDir == "" {
		return fmt.Errorf("content directory cannot be empty")
	}
	if *distDir == "" {
		return fmt.Errorf("dist directory cannot be empty")
	}
	if *siteURL == "" {
		return fmt.Errorf("site URL cannot be empty")
	}
	parsedURL, err := url.Parse(*siteURL)
	if err != nil {
		return fmt.Errorf("invalid site URL: %w", err)
	}
	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return fmt.Errorf("site URL must use http or https scheme: %s", *siteURL)
	}
	if parsedURL.Host == "" {
		return fmt.Errorf("site URL must include a host: %s", *siteURL)
	}

	fmt.Println("Go Proposal Weekly Digest Generator")
	fmt.Printf("Content directory: %s\n", *contentDir)
	fmt.Printf("Output directory: %s\n", *distDir)
	fmt.Printf("Site URL: %s\n", *siteURL)

	// Verify content directory exists and is a directory
	contentInfo, err := os.Stat(*contentDir)
	if os.IsNotExist(err) {
		return fmt.Errorf("content directory does not exist: %s", *contentDir)
	}
	if err != nil {
		return fmt.Errorf("failed to access content directory: %w", err)
	}
	if !contentInfo.IsDir() {
		return fmt.Errorf("content path is not a directory: %s", *contentDir)
	}

	// Create content manager to read content
	contentManager := content.NewManager(content.WithBaseDir(*contentDir))

	// List all weekly contents
	weeks, err := contentManager.ListAllWeeks()
	if err != nil {
		return fmt.Errorf("failed to list weekly contents: %w", err)
	}

	fmt.Printf("Found %d weeks of content\n", len(weeks))

	// Create site generator
	generator := site.NewGenerator(
		site.WithDistDir(*distDir),
		site.WithGeneratorSiteURL(*siteURL),
	)

	// Generate the site
	ctx := context.Background()
	if err := generator.Generate(ctx, weeks); err != nil {
		return fmt.Errorf("failed to generate site: %w", err)
	}

	fmt.Println("Site generation completed successfully!")
	fmt.Println("  - HTML pages generated")
	fmt.Println("  - RSS feed generated (feed.xml)")
	return nil
}
