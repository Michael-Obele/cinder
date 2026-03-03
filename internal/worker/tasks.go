package worker

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/hibiken/asynq"
)

const (
	TypeScrape = "scrape:url"
	TypeCrawl  = "crawl:site"
)

type ScrapePayload struct {
	URL        string `json:"url"`
	Render     bool   `json:"render"` // Deprecated: usage ignores Mode if true
	Mode       string `json:"mode"`   // "smart", "static", "dynamic"
	Screenshot bool   `json:"screenshot"`
	Images     bool   `json:"images"`
}

// CrawlPayload extends ScrapePayload with multi-page crawling options.
type CrawlPayload struct {
	URL        string `json:"url"`
	Render     bool   `json:"render"`
	Mode       string `json:"mode"`
	Screenshot bool   `json:"screenshot"`
	Images     bool   `json:"images"`
	MaxDepth   int    `json:"maxDepth"`
	Limit      int    `json:"limit"`
}

// NewScrapeTask creates a new task for scraping a single URL.
func NewScrapeTask(url string, render bool, screenshot bool, images bool) (*asynq.Task, error) {
	payload := ScrapePayload{
		URL:        url,
		Render:     render,
		Screenshot: screenshot,
		Images:     images,
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal scrape payload: %w", err)
	}
	// Keep result for 7 days
	return asynq.NewTask(TypeScrape, data, asynq.Retention(7*24*time.Hour)), nil
}

// NewCrawlTask creates a new task for crawling a site starting from a seed URL.
func NewCrawlTask(crawlURL string, render bool, screenshot bool, images bool, maxDepth int, limit int) (*asynq.Task, error) {
	payload := CrawlPayload{
		URL:        crawlURL,
		Render:     render,
		Screenshot: screenshot,
		Images:     images,
		MaxDepth:   maxDepth,
		Limit:      limit,
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal crawl payload: %w", err)
	}
	// Crawl tasks can be longer; retain result for 7 days
	return asynq.NewTask(TypeCrawl, data,
		asynq.Retention(7*24*time.Hour),
		asynq.MaxRetry(2),
	), nil
}
