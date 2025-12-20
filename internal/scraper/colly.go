package scraper

import (
	"context"
	"fmt"
	"time"

	md "github.com/JohannesKaufmann/html-to-markdown/v2"
	"github.com/brianvoe/gofakeit/v6"
	"github.com/gocolly/colly/v2"
	"github.com/standard-user/cinder/internal/domain"
	"github.com/standard-user/cinder/pkg/logger"
)

type CollyScraper struct {
}

func NewCollyScraper() *CollyScraper {
	return &CollyScraper{}
}

func (s *CollyScraper) Scrape(ctx context.Context, url string) (*domain.ScrapeResult, error) {
	c := colly.NewCollector(
		colly.Async(true),
	)

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

	// Convert to Markdown
	markdown, err := md.ConvertString(htmlContent)
	if err != nil {
		return nil, fmt.Errorf("markdown conversion failed: %w", err)
	}

	return &domain.ScrapeResult{
		URL:      url,
		Markdown: markdown,
		HTML:     htmlContent, // Optional: might want to toggle this
		Metadata: map[string]string{
			"scraped_at": time.Now().Format(time.RFC3339),
			"engine":     "colly",
		},
	}, nil
}
