package scraper

import (
	"context"
	"fmt"
	"time"

	"github.com/chromedp/chromedp"
	md "github.com/JohannesKaufmann/html-to-markdown/v2"
	"github.com/standard-user/cinder/internal/domain"
	"github.com/standard-user/cinder/pkg/logger"
)

type ChromedpScraper struct {
}

func NewChromedpScraper() *ChromedpScraper {
	return &ChromedpScraper{}
}

func (s *ChromedpScraper) Scrape(ctx context.Context, url string) (*domain.ScrapeResult, error) {
	// Create context
	// We use the parent context to respect timeouts/cancellation
	// But we also need an allocator.
	// For now, we assume a local chrome instance or a default allocator.
	// In production (Docker), we might need to point to a remote allocator or use default with specific flags.
	
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", true),
		chromedp.Flag("disable-gpu", true),
		chromedp.Flag("no-sandbox", true),
		chromedp.Flag("disable-dev-shm-usage", true), // Critical for Docker
		chromedp.UserAgent("Mozilla/5.0 (compatible; CinderBot/1.0; +http://github.com/standard-user/cinder)"),
	)

	allocCtx, cancelAlloc := chromedp.NewExecAllocator(ctx, opts...)
	defer cancelAlloc()

	taskCtx, cancelTask := chromedp.NewContext(allocCtx)
	defer cancelTask()

	// Set a hard timeout for the browser actions
	taskCtx, cancelTimeout := context.WithTimeout(taskCtx, 60*time.Second)
	defer cancelTimeout()

	var htmlContent string
	
	logger.Log.Info("Chromedp Scraping", "url", url)

	err := chromedp.Run(taskCtx,
		chromedp.Navigate(url),
		// Wait for body to be visible - this ensures some content is loaded.
		// For complex SPAs, we might want to wait for network idle, but that can be flaky.
		chromedp.WaitVisible("body", chromedp.ByQuery),
		chromedp.OuterHTML("html", &htmlContent),
	)

	if err != nil {
		return nil, fmt.Errorf("chromedp failed: %w", err)
	}

	if htmlContent == "" {
		return nil, fmt.Errorf("empty response from browser")
	}

	markdown, err := md.ConvertString(htmlContent)
	if err != nil {
		return nil, fmt.Errorf("markdown conversion failed: %w", err)
	}

	return &domain.ScrapeResult{
		URL:      url,
		Markdown: markdown,
		HTML:     htmlContent,
		Metadata: map[string]string{
			"scraped_at": time.Now().Format(time.RFC3339),
			"engine":     "chromedp",
		},
	}, nil
}
