package scraper

import (
	"net/url"
	"strings"

	readability "github.com/go-shiori/go-readability"
)

// ReadabilityResult holds main-content extraction output.
type ReadabilityResult struct {
	Title       string
	Excerpt     string
	Byline      string
	SiteName    string
	ContentHTML string
}

// ExtractMainContent runs readability on rawHTML. On failure it returns a
// fallback result containing the original HTML and a nil error — callers
// must never fail a scrape because extraction failed.
func ExtractMainContent(rawHTML string, pageURL string) (*ReadabilityResult, error) {
	parsed, err := readability.FromReader(strings.NewReader(rawHTML), mustParseURL(pageURL))
	if err != nil {
		return &ReadabilityResult{ContentHTML: rawHTML}, nil
	}
	return &ReadabilityResult{
		Title:       parsed.Title,
		Excerpt:     parsed.Excerpt,
		Byline:      parsed.Byline,
		SiteName:    parsed.SiteName,
		ContentHTML: parsed.Content,
	}, nil
}

// mustParseURL parses pageURL, returning a zero URL on error so readability
// still functions for relative-link resolution.
func mustParseURL(pageURL string) *url.URL {
	u, err := url.Parse(pageURL)
	if err != nil {
		return &url.URL{}
	}
	return u
}
