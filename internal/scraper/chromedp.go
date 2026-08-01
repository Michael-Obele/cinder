package scraper

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"sync"
	"time"

	md "github.com/JohannesKaufmann/html-to-markdown/v2"
	"github.com/brianvoe/gofakeit/v6"
	"github.com/chromedp/cdproto/emulation"
	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/chromedp"
	"github.com/standard-user/cinder/internal/domain"
	"github.com/standard-user/cinder/pkg/logger"
)

// defaultRecycleAfter is the default number of scrapes before the Chrome
// allocator is restarted to bound long-term memory growth.
const defaultRecycleAfter = 100

// ChromedpScraper reuses a single Chrome allocator across requests, spawning
// lightweight tabs per scrape, and restarts the allocator periodically to
// prevent memory leaks.
type ChromedpScraper struct {
	mu           sync.Mutex
	allocCtx     context.Context
	cancel       context.CancelFunc
	scrapeCount  int
	recycleAfter int
	newAllocator func() (context.Context, context.CancelFunc)
}

// NewChromedpScraper creates a scraper with the default recycle threshold.
func NewChromedpScraper() *ChromedpScraper {
	return NewChromedpScraperWithLimit(defaultRecycleAfter)
}

// NewChromedpScraperWithLimit creates a scraper that restarts its Chrome
// allocator after recycleAfter scrapes. Values <= 0 fall back to the default.
func NewChromedpScraperWithLimit(recycleAfter int) *ChromedpScraper {
	if recycleAfter <= 0 {
		recycleAfter = defaultRecycleAfter
	}
	s := &ChromedpScraper{
		recycleAfter: recycleAfter,
		newAllocator: buildAllocator,
	}
	s.allocCtx, s.cancel = s.newAllocator()
	warmUp(s.allocCtx)
	return s
}

// buildAllocator constructs a fresh Chrome exec allocator with the flags
// required for headless container operation.
func buildAllocator() (context.Context, context.CancelFunc) {
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", true),
		chromedp.Flag("disable-gpu", true),
		chromedp.Flag("no-sandbox", true),
		chromedp.Flag("disable-dev-shm-usage", true), // Critical for Docker
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

	return chromedp.NewExecAllocator(context.Background(), opts...)
}

// warmUp starts the browser synchronously so failures surface at startup.
func warmUp(allocCtx context.Context) {
	warmCtx, warmCancel := chromedp.NewContext(allocCtx)
	defer warmCancel()
	timedCtx, cancelTimeout := context.WithTimeout(warmCtx, 15*time.Second)
	defer cancelTimeout()
	if err := chromedp.Run(timedCtx); err != nil {
		logger.Log.Warn("Chrome browser not available — dynamic rendering disabled", "error", err)
	} else {
		logger.Log.Info("Chrome browser started successfully (dynamic rendering enabled)")
	}
}

// shouldRecycle reports whether the scrape counter crossed the threshold.
func shouldRecycle(count, after int) bool {
	return after > 0 && count >= after
}

// beginScrape atomically bumps the scrape counter, recycling the allocator
// when the threshold is crossed, and returns the current allocator context.
func (s *ChromedpScraper) beginScrape() context.Context {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.scrapeCount++
	if shouldRecycle(s.scrapeCount, s.recycleAfter) {
		s.recycleLocked()
	}
	return s.allocCtx
}

// recycleLocked cancels the old allocator and swaps in a fresh one. Callers
// must hold s.mu. The new browser spawns lazily on the next tab use.
func (s *ChromedpScraper) recycleLocked() {
	if s.cancel != nil {
		s.cancel()
	}
	s.allocCtx, s.cancel = s.newAllocator()
	s.scrapeCount = 0
	logger.Log.Info("Chrome allocator recycled to bound memory growth")
}

func (s *ChromedpScraper) Close() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.cancel != nil {
		s.cancel()
		s.cancel = nil
	}
}

// screenshotParams is the resolved, validated set of screenshot settings.
type screenshotParams struct {
	width        int
	height       int
	format       string // "jpeg" or "png"
	quality      int
	fullPage     bool
	waitSelector string
}

// resolveScreenshotParams applies defaults and clamps values from the
// user-supplied ScreenshotOptions. A nil opts yields jpeg @1920x1080 q90.
func resolveScreenshotParams(opts *domain.ScreenshotOptions) screenshotParams {
	p := screenshotParams{width: 1920, height: 1080, format: "jpeg", quality: 90}
	if opts == nil {
		return p
	}
	if opts.Width > 0 {
		p.width = opts.Width
	}
	if opts.Height > 0 {
		p.height = opts.Height
	}
	switch opts.Format {
	case "png":
		p.format = "png"
	case "jpeg", "jpg", "":
		p.format = "jpeg"
	default:
		p.format = "jpeg"
	}
	if opts.Quality > 0 && opts.Quality <= 100 {
		p.quality = opts.Quality
	}
	p.fullPage = opts.FullPage
	p.waitSelector = opts.WaitSelector
	return p
}

func (s *ChromedpScraper) Scrape(ctx context.Context, url string, opts domain.ScrapeOptions) (*domain.ScrapeResult, error) {
	// Create a new tab (Context) from the existing allocator
	// This is much faster than starting a new browser process
	taskCtx, cancelTask := chromedp.NewContext(s.beginScrape())
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

	// Navigate and capture HTML first.
	// Rotate the User-Agent per tab for anti-detection (mirrors colly).
	// Use Evaluate instead of OuterHTML to avoid stale node references
	// on SPAs that replace the DOM after the initial page load.
	err := chromedp.Run(taskCtx,
		emulation.SetUserAgentOverride(gofakeit.UserAgent()),
		chromedp.Navigate(url),
		chromedp.WaitVisible("body", chromedp.ByQuery),
		chromedp.Evaluate(`document.documentElement.outerHTML`, &htmlContent),
	)

	if err != nil {
		return nil, fmt.Errorf("chromedp failed: %w", err)
	}

	// If screenshot is requested, do it in a separate Run call so the
	// viewport resize doesn't invalidate the HTML node references from
	// the navigation action above.
	if opts.Screenshot {
		p := resolveScreenshotParams(opts.ScreenshotOpts)

		actions := []chromedp.Action{}
		if p.waitSelector != "" {
			actions = append(actions, chromedp.WaitVisible(p.waitSelector, chromedp.ByQuery))
		}
		actions = append(actions,
			chromedp.EmulateViewport(int64(p.width), int64(p.height)),
			chromedp.ActionFunc(func(ctx context.Context) error {
				var err error
				screenshotBuf, err = page.CaptureScreenshot().
					WithFormat(page.CaptureScreenshotFormat(p.format)).
					WithQuality(int64(p.quality)).
					WithCaptureBeyondViewport(p.fullPage).
					Do(ctx)
				return err
			}),
		)
		err = chromedp.Run(taskCtx, actions...)
		if err != nil {
			// Log screenshot failure but don't fail the whole scrape
			logger.Log.Warn("Screenshot failed, returning HTML-only result", "url", url, "error", err)
		}
	}

	if htmlContent == "" {
		return nil, fmt.Errorf("empty response from browser")
	}

	// Extract the main content before markdown conversion so nav/ads/footers
	// don't pollute the LLM-ready output. Falls back to full HTML on failure.
	rc, _ := ExtractMainContent(htmlContent, url)

	markdown, err := md.ConvertString(rc.ContentHTML)
	if err != nil {
		return nil, fmt.Errorf("markdown conversion failed: %w", err)
	}

	metadata := map[string]string{
		"scraped_at": time.Now().Format(time.RFC3339),
		"engine":     "chromedp",
	}
	applyReadabilityMetadata(metadata, rc)

	result := &domain.ScrapeResult{
		URL:      url,
		Markdown: markdown,
		HTML:     htmlContent,
		Metadata: metadata,
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
