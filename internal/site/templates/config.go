// Package templates provides templ-based templates for the static site.
package templates

// DefaultFeedURL is the default RSS feed URL used when no custom URL is specified.
const DefaultFeedURL = "/feed.xml"

// DefaultOGPImageURL is the default OGP image URL.
const DefaultOGPImageURL = "/ogp.png"

// DefaultSiteName is the default site name for OGP.
const DefaultSiteName = "Go Proposal Weekly Digest"

// ResolveFeedURL returns the provided feed URL if non-empty, otherwise returns DefaultFeedURL.
// This is the single source of truth for feed URL fallback logic.
func ResolveFeedURL(feedURL string) string {
	if feedURL == "" {
		return DefaultFeedURL
	}
	return feedURL
}

// LayoutConfig holds configuration for the base layout template.
type LayoutConfig struct {
	// Title is the page title shown in the browser tab.
	Title string
	// FeedURL is the URL for the RSS feed autodiscovery link.
	// If empty, DefaultFeedURL is used.
	FeedURL string
	// OGP holds Open Graph Protocol metadata for social media sharing.
	OGP OGPConfig
}

// OGPConfig holds Open Graph Protocol metadata.
type OGPConfig struct {
	// Title is the OG title. If empty, the page title is used.
	Title string
	// Description is the OG description.
	Description string
	// ImageURL is the absolute URL to the OG image.
	ImageURL string
	// URL is the canonical URL of the page.
	URL string
	// Type is the OG type (e.g., "website", "article"). Defaults to "website".
	Type string
}

// GetFeedURL returns the feed URL, using the default if not set.
func (c LayoutConfig) GetFeedURL() string {
	return ResolveFeedURL(c.FeedURL)
}

// PageConfig holds configuration for page rendering with layout.
type PageConfig struct {
	// Title is the page title shown in the browser tab.
	Title string
	// CurrentPath is the current page path for navigation highlighting.
	CurrentPath string
	// FeedURL is the URL for the RSS feed autodiscovery and navigation link.
	// If empty, DefaultFeedURL is used.
	FeedURL string
	// OGP holds Open Graph Protocol metadata for social media sharing.
	OGP OGPConfig
}

// GetFeedURL returns the feed URL, using the default if not set.
func (c PageConfig) GetFeedURL() string {
	return ResolveFeedURL(c.FeedURL)
}

// GetOGPTitle returns the OGP title, falling back to the provided page title if not set.
func (ogp OGPConfig) GetOGPTitle(pageTitle string) string {
	if ogp.Title == "" {
		return pageTitle
	}
	return ogp.Title
}

// GetOGPType returns the OGP type, defaulting to "website" if not set.
func (ogp OGPConfig) GetOGPType() string {
	if ogp.Type == "" {
		return "website"
	}
	return ogp.Type
}

// NewOGPConfig creates a default OGP configuration with the given parameters.
// siteURL is the base URL of the site (e.g., "https://example.com").
func NewOGPConfig(siteURL, path, title, description string) OGPConfig {
	imageURL := siteURL + DefaultOGPImageURL
	pageURL := siteURL + path

	return OGPConfig{
		Title:       title,
		Description: description,
		ImageURL:    imageURL,
		URL:         pageURL,
		Type:        "website",
	}
}
