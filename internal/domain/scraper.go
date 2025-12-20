package domain

import "context"

type ScrapeResult struct {
	URL      string            `json:"url"`
	Markdown string            `json:"markdown"`
	HTML     string            `json:"html,omitempty"`
	Metadata map[string]string `json:"metadata,omitempty"`
}

type Scraper interface {
	Scrape(ctx context.Context, url string) (*ScrapeResult, error)
}
