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
func (g *Generator) Generate(ctx context.Context, weeks []*content.WeeklyContent) error {
	// Check for context cancellation at the start
	if err := ctx.Err(); err != nil {
		return err
	}

	// Create the dist directory
	if err := os.MkdirAll(g.distDir, dirPerm); err != nil {
		return fmt.Errorf("failed to create dist directory: %w", err)
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
	homeData := templates.ConvertToHomeData(weeks)
	component := templates.HomePage(homeData)

	filePath := filepath.Join(g.distDir, "index.html")
	return g.renderToFile(ctx, filePath, component)
}

// generateWeeklyIndexPage generates a weekly index page.
func (g *Generator) generateWeeklyIndexPage(ctx context.Context, data templates.WeeklyData) error {
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
