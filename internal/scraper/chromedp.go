package scraper

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
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
		chromedp.Flag("single-process", true),
		chromedp.UserAgent("Mozilla/5.0 (compatible; CinderBot/1.0; +http://github.com/standard-user/cinder)"),
	)

	// Respect CHROME_BIN env var if set (Dockerfile sets it)
	if chromeBin := os.Getenv("CHROME_BIN"); chromeBin != "" {
		if _, err := os.Stat(chromeBin); err == nil {
			opts = append(opts, chromedp.ExecPath(chromeBin))
			logger.Log.Info("Chrome binary found", "path", chromeBin)
		} else {
			logger.Log.Warn("CHROME_BIN set but binary not found", "path", chromeBin, "error", err)
		}
	}

	allocCtx, allocCancel := chromedp.NewExecAllocator(context.Background(), opts...)

	// Warm up Chrome synchronously so we fail fast at startup if unavailable.
	// Use a short timeout — this just validates the browser starts.
	warmCtx, warmCancel := chromedp.NewContext(allocCtx)
	defer warmCancel()
	timedCtx, cancelTimeout := context.WithTimeout(warmCtx, 15*time.Second)
	defer cancelTimeout()
	if err := chromedp.Run(timedCtx); err != nil {
		logger.Log.Warn("Chrome browser not available — dynamic rendering disabled", "error", err)
	} else {
		logger.Log.Info("Chrome browser started successfully (dynamic rendering enabled)")
	}

	return &ChromedpScraper{
		allocCtx: allocCtx,
		cancel:   allocCancel,
	}
}

func (s *ChromedpScraper) Close() {
	if s.cancel != nil {
		s.cancel()
	}
}

func (s *ChromedpScraper) Scrape(ctx context.Context, url string, opts domain.ScrapeOptions) (*domain.ScrapeResult, error) {
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
	var screenshotBuf []byte

	logger.Log.Info("Chromedp Scraping", "url", url, "screenshot", opts.Screenshot)

	actions := []chromedp.Action{
		chromedp.Navigate(url),
		// Wait for body to be visible - this ensures some content is loaded.
		chromedp.WaitVisible("body", chromedp.ByQuery),
		chromedp.OuterHTML("html", &htmlContent),
	}

	if opts.Screenshot {
		actions = append(actions, chromedp.FullScreenshot(&screenshotBuf, 90))
	}

	err := chromedp.Run(taskCtx, actions...)

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

	result := &domain.ScrapeResult{
		URL:      url,
		Markdown: markdown,
		HTML:     htmlContent,
		Metadata: map[string]string{
			"scraped_at": time.Now().Format(time.RFC3339),
			"engine":     "chromedp",
		},
	}

	if opts.Screenshot && len(screenshotBuf) > 0 {
		result.Screenshot = &domain.ScreenshotData{
			Blob:       base64.StdEncoding.EncodeToString(screenshotBuf),
			Format:     "jpeg",
			SizeBytes:  int64(len(screenshotBuf)),
			CapturedAt: time.Now(),
		}
	}

	return result, nil
}
