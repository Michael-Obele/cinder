package scraper

import (
	"context"
	"fmt"
	"time"

	md "github.com/JohannesKaufmann/html-to-markdown/v2"
	"github.com/chromedp/chromedp"
	"github.com/standard-user/cinder/internal/domain"
	"github.com/standard-user/cinder/pkg/logger"
)

type ChromedpScraper struct {
	allocCtx context.Context
	cancel   context.CancelFunc
}

func NewChromedpScraper() *ChromedpScraper {
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", true),
		chromedp.Flag("disable-gpu", true),
		chromedp.Flag("no-sandbox", true),
		chromedp.Flag("disable-dev-shm-usage", true), // Critical for Docker
		chromedp.UserAgent("Mozilla/5.0 (compatible; CinderBot/1.0; +http://github.com/standard-user/cinder)"),
	)

	allocCtx, cancel := chromedp.NewExecAllocator(context.Background(), opts...)

	// We start a mock run to ensure the browser process starts immediately,
	// rather than waiting for the first request.
	go func() {
		ctx, c := chromedp.NewContext(allocCtx)
		defer c()
		if err := chromedp.Run(ctx); err != nil {
			logger.Log.Error("Failed to start initial browser process", "error", err)
		}
	}()

	return &ChromedpScraper{
		allocCtx: allocCtx,
		cancel:   cancel,
	}
}

func (s *ChromedpScraper) Close() {
	if s.cancel != nil {
		s.cancel()
	}
}

func (s *ChromedpScraper) Scrape(ctx context.Context, url string) (*domain.ScrapeResult, error) {
	// Create a new tab (Context) from the existing allocator
	// This is much faster than starting a new browser process
	taskCtx, cancelTask := chromedp.NewContext(s.allocCtx)
	defer cancelTask()

	// Set a hard timeout for the browser actions
	// Use the parent context's deadline if available, otherwise default to 60s
	// But we must respect the parent context cancellation
	timeout := 60 * time.Second
	if dl, ok := ctx.Deadline(); ok {
		timeout = time.Until(dl)
	}

	taskCtx, cancelTimeout := context.WithTimeout(taskCtx, timeout)
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
