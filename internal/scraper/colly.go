package scraper

import (
	"context"
	"fmt"
	"time"

	md "github.com/JohannesKaufmann/html-to-markdown/v2"
	"github.com/brianvoe/gofakeit/v6"
	"github.com/gocolly/colly/v2"
	"github.com/standard-user/cinder/internal/domain"
	"github.com/standard-user/cinder/internal/safeurl"
	"github.com/standard-user/cinder/pkg/logger"
)

type CollyScraper struct {
}

func NewCollyScraper() *CollyScraper {
	return &CollyScraper{}
}

func (s *CollyScraper) Scrape(ctx context.Context, url string, opts domain.ScrapeOptions) (*domain.ScrapeResult, error) {
	c := colly.NewCollector(
		colly.Async(true),
	)

	// Refuse non-public destinations. The guarded transport checks at dial
	// time, so it covers colly's redirect following as well as the initial
	// request — a pre-flight URL check alone would not.
	c.WithTransport(safeurl.Transport())

	// Rotate User-Agents
	c.OnRequest(func(r *colly.Request) {
		r.Headers.Set("User-Agent", gofakeit.UserAgent())
		logger.Log.Info("Scraping", "url", r.URL, "user_agent", r.Headers.Get("User-Agent"))
	})

	c.SetRequestTimeout(30 * time.Second)

	var htmlContent string
	var scrapeErr error

	c.OnHTML("html", func(e *colly.HTMLElement) {
		htmlContent, _ = e.DOM.Html()
	})

	c.OnError(func(r *colly.Response, err error) {
		if r != nil && r.StatusCode >= 400 {
			// Carry the HTTP status so retry policies can skip 4xx.
			scrapeErr = &StatusError{StatusCode: r.StatusCode, Err: err}
			return
		}
		scrapeErr = fmt.Errorf("scraping failed: %w", err)
	})

	err := c.Visit(url)
	if err != nil {
		return nil, err
	}

	c.Wait()

	if scrapeErr != nil {
		return nil, scrapeErr
	}

	if htmlContent == "" {
		return nil, fmt.Errorf("empty response")
	}

	// Apply cleaner-output defaults (block ads, drop base64 images) before
	// readability strips the class/id/aria-label attributes the selectors need.
	clean := cleanContent(htmlContent,
		opts.BlockAds == nil || *opts.BlockAds,
		opts.RemoveBase64Images == nil || *opts.RemoveBase64Images,
	)

	// Extract the main content after cleaning so nav/ads/footers don't pollute
	// the LLM-ready output. Falls back to full HTML on failure.
	rc, _ := ExtractMainContent(clean, url)

	// Convert to Markdown
	markdown, err := md.ConvertString(rc.ContentHTML)
	if err != nil {
		return nil, fmt.Errorf("markdown conversion failed: %w", err)
	}

	metadata := map[string]string{
		"scraped_at": time.Now().Format(time.RFC3339),
		"engine":     "colly",
	}
	applyReadabilityMetadata(metadata, rc)

	return &domain.ScrapeResult{
		URL:      url,
		Markdown: markdown,
		HTML:     htmlContent, // Optional: might want to toggle this
		Metadata: metadata,
	}, nil
}
