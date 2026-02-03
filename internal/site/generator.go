// Package site provides functionality for generating the static site.
package site

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"

	"github.com/a-h/templ"
	"github.com/mazrean/go-proposal-review-meeting/internal/content"
	"github.com/mazrean/go-proposal-review-meeting/internal/site/templates"
)

// publicDir is the directory containing static files to be copied to dist.
const publicDir = "web/public"

// dirPerm is the permission mode for created directories.
const dirPerm = 0o755

// filePerm is the permission mode for created files.
const filePerm = 0o644

// Generator handles static site generation from content data.
type Generator struct {
	distDir string
	siteURL string
}

// Option is a functional option for configuring Generator.
type Option func(*Generator)

// WithDistDir sets the output directory for generated files.
func WithDistDir(dir string) Option {
	return func(g *Generator) {
		g.distDir = dir
	}
}

// WithGeneratorSiteURL sets the site URL for RSS feed generation.
func WithGeneratorSiteURL(url string) Option {
	return func(g *Generator) {
		g.siteURL = url
	}
}

// NewGenerator creates a new site Generator with the given options.
func NewGenerator(opts ...Option) *Generator {
	g := &Generator{
		distDir: "dist",
		siteURL: "https://example.com",
	}
	for _, opt := range opts {
		opt(g)
	}
	return g
}

// Generate generates the static site from the given weekly contents.
// It creates:
// - index.html (home page with week listing)
// - YYYY/wWW/index.html (weekly index pages)
// - YYYY/wWW/NNNNN.html (individual proposal pages)
// - feed.xml (RSS 2.0 feed)
// - Static files copied from web/public/ to dist/
func (g *Generator) Generate(ctx context.Context, weeks []*content.WeeklyContent) error {
	// Check for context cancellation at the start
	if err := ctx.Err(); err != nil {
		return err
	}

	// Create the dist directory
	if err := os.MkdirAll(g.distDir, dirPerm); err != nil {
		return fmt.Errorf("failed to create dist directory: %w", err)
	}

	// Copy static files from web/public/ to dist/
	if err := g.copyPublicFiles(ctx); err != nil {
		return fmt.Errorf("failed to copy static files: %w", err)
	}

	// Convert weeks to template data
	var weeklyDataList []templates.WeeklyData
	for _, week := range weeks {
		if week != nil {
			weeklyDataList = append(weeklyDataList, templates.ConvertToWeeklyData(week))
		}
	}

	// Sort weeks by date (newest first)
	sort.Slice(weeklyDataList, func(i, j int) bool {
		if weeklyDataList[i].Year != weeklyDataList[j].Year {
			return weeklyDataList[i].Year > weeklyDataList[j].Year
		}
		return weeklyDataList[i].Week > weeklyDataList[j].Week
	})

	// Generate home page
	if err := g.generateHomePage(ctx, weeklyDataList); err != nil {
		return fmt.Errorf("failed to generate home page: %w", err)
	}

	// Generate weekly pages and proposal pages
	for _, week := range weeks {
		if week == nil {
			continue
		}

		// Check for context cancellation
		if err := ctx.Err(); err != nil {
			return err
		}

		weeklyData := templates.ConvertToWeeklyData(week)

		// Generate weekly index page
		if err := g.generateWeeklyIndexPage(ctx, weeklyData); err != nil {
			return fmt.Errorf("failed to generate weekly index page for %d-W%02d: %w",
				week.Year, week.Week, err)
		}

		// Generate individual proposal pages
		for _, proposal := range week.Proposals {
			if err := ctx.Err(); err != nil {
				return err
			}

			detailData := templates.ConvertToProposalDetailData(week, proposal.IssueNumber)
			if detailData == nil {
				return fmt.Errorf("failed to convert proposal data for #%d: proposal not found in week data",
					proposal.IssueNumber)
			}
			if err := g.generateProposalPage(ctx, *detailData); err != nil {
				return fmt.Errorf("failed to generate proposal page for #%d: %w",
					proposal.IssueNumber, err)
			}
		}
	}

	// Generate RSS feed
	if err := g.generateRSSFeed(ctx, weeks); err != nil {
		return fmt.Errorf("failed to generate RSS feed: %w", err)
	}

	return nil
}

// generateHomePage generates the home page (index.html).
func (g *Generator) generateHomePage(ctx context.Context, weeks []templates.WeeklyData) error {
	homeData := templates.ConvertToHomeData(weeks, g.siteURL)
	component := templates.HomePage(homeData)

	filePath := filepath.Join(g.distDir, "index.html")
	return g.renderToFile(ctx, filePath, component)
}

// generateWeeklyIndexPage generates a weekly index page.
func (g *Generator) generateWeeklyIndexPage(ctx context.Context, data templates.WeeklyData) error {
	// Set the site URL for OGP tags
	data.SiteURL = g.siteURL
	component := templates.WeeklyIndexPage(data)

	// Create directory path: dist/YYYY/wWW/
	dirPath := filepath.Join(g.distDir, fmt.Sprintf("%d", data.Year), fmt.Sprintf("w%02d", data.Week))
	if err := os.MkdirAll(dirPath, dirPerm); err != nil {
		return fmt.Errorf("failed to create weekly directory: %w", err)
	}

	filePath := filepath.Join(dirPath, "index.html")
	return g.renderToFile(ctx, filePath, component)
}

// generateProposalPage generates an individual proposal page.
func (g *Generator) generateProposalPage(ctx context.Context, data templates.ProposalDetailData) error {
	// Set the site URL for OGP tags
	data.SiteURL = g.siteURL
	component := templates.ProposalDetailPage(data)

	// Create directory path: dist/YYYY/wWW/
	dirPath := filepath.Join(g.distDir, fmt.Sprintf("%d", data.Year), fmt.Sprintf("w%02d", data.Week))
	if err := os.MkdirAll(dirPath, dirPerm); err != nil {
		return fmt.Errorf("failed to create proposal directory: %w", err)
	}

	filePath := filepath.Join(dirPath, fmt.Sprintf("%d.html", data.IssueNumber))
	return g.renderToFile(ctx, filePath, component)
}

// renderToFile renders a templ component to a file.
// If rendering fails, the partially written file is removed to avoid serving corrupted HTML.
func (g *Generator) renderToFile(ctx context.Context, filePath string, component templ.Component) (err error) {
	file, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer func() {
		if closeErr := file.Close(); closeErr != nil && err == nil {
			err = fmt.Errorf("failed to close file: %w", closeErr)
		}
		// Remove partial file on error
		if err != nil {
			_ = os.Remove(filePath)
		}
	}()

	if err := component.Render(ctx, io.Writer(file)); err != nil {
		return fmt.Errorf("failed to render component: %w", err)
	}

	return nil
}

// generateRSSFeed generates the RSS feed (feed.xml).
// If writing fails, any partially written file is removed.
func (g *Generator) generateRSSFeed(ctx context.Context, weeks []*content.WeeklyContent) error {
	fg := NewFeedGenerator(WithSiteURL(g.siteURL))

	feedData, err := fg.GenerateFeed(ctx, weeks)
	if err != nil {
		return fmt.Errorf("failed to generate feed: %w", err)
	}

	feedPath := filepath.Join(g.distDir, "feed.xml")
	if err := os.WriteFile(feedPath, feedData, filePerm); err != nil {
		// Remove partial file on error
		_ = os.Remove(feedPath)
		return fmt.Errorf("failed to write feed.xml: %w", err)
	}

	return nil
}

// copyPublicFiles copies static files from web/public/ to dist/.
// If the public directory doesn't exist, it returns without error.
func (g *Generator) copyPublicFiles(ctx context.Context) error {
	// Check if public directory exists
	if _, err := os.Stat(publicDir); os.IsNotExist(err) {
		return nil // No public directory, skip
	} else if err != nil {
		return fmt.Errorf("failed to stat public directory: %w", err)
	}

	// Walk through the public directory
	return filepath.Walk(publicDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Check for context cancellation
		if err := ctx.Err(); err != nil {
			return err
		}

		// Skip the root directory itself
		if path == publicDir {
			return nil
		}

		// Get relative path from public directory
		relPath, err := filepath.Rel(publicDir, path)
		if err != nil {
			return fmt.Errorf("failed to get relative path: %w", err)
		}

		destPath := filepath.Join(g.distDir, relPath)

		if info.IsDir() {
			// Create directory in dist
			if err := os.MkdirAll(destPath, dirPerm); err != nil {
				return fmt.Errorf("failed to create directory %s: %w", destPath, err)
			}
			return nil
		}

		// Copy file
		return copyFile(path, destPath)
	})
}

// copyFile copies a file from src to dst.
func copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("failed to open source file: %w", err)
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("failed to create destination file: %w", err)
	}
	defer func() {
		if closeErr := destFile.Close(); closeErr != nil && err == nil {
			err = fmt.Errorf("failed to close destination file: %w", closeErr)
		}
		// Remove partial file on error
		if err != nil {
			_ = os.Remove(dst)
		}
	}()

	if _, err := io.Copy(destFile, sourceFile); err != nil {
		return fmt.Errorf("failed to copy file: %w", err)
	}

	// Copy file permissions
	sourceInfo, err := os.Stat(src)
	if err != nil {
		return fmt.Errorf("failed to stat source file: %w", err)
	}
	if err := os.Chmod(dst, sourceInfo.Mode()); err != nil {
		return fmt.Errorf("failed to set file permissions: %w", err)
	}

	return nil
}
