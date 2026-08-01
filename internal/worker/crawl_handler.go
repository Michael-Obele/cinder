package worker

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"path"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/hibiken/asynq"
	"github.com/standard-user/cinder/internal/domain"
	"github.com/standard-user/cinder/internal/scraper"
)

// CrawlResult is the aggregated output of a multi-page crawl.
type CrawlResult struct {
	Status     string                `json:"status"` // "completed", "partial"
	TotalPages int                   `json:"total"`
	MaxDepth   int                   `json:"maxDepth"`
	Limit      int                   `json:"limit"`
	Pages      []domain.ScrapeResult `json:"data"`
	FailedURLs []FailedURL           `json:"failedUrls,omitempty"`
}

// FailedURL records a URL that could not be scraped during the crawl.
type FailedURL struct {
	URL   string `json:"url"`
	Error string `json:"error"`
}

// CrawlTaskHandler processes multi-page crawl tasks using BFS.
type CrawlTaskHandler struct {
	scraper *scraper.Service
	logger  *slog.Logger
}

func NewCrawlTaskHandler(scraper *scraper.Service, logger *slog.Logger) *CrawlTaskHandler {
	return &CrawlTaskHandler{
		scraper: scraper,
		logger:  logger,
	}
}

// ProcessTask is the Asynq entry point — it deserializes the payload,
// delegates to ExecuteCrawl, and writes the result back to Asynq.
func (h *CrawlTaskHandler) ProcessTask(ctx context.Context, t *asynq.Task) error {
	var payload CrawlPayload
	if err := json.Unmarshal(t.Payload(), &payload); err != nil {
		return fmt.Errorf("failed to unmarshal crawl payload: %w, task_id=%s", err, t.ResultWriter().TaskID())
	}

	taskID := t.ResultWriter().TaskID()

	result, err := h.ExecuteCrawl(ctx, payload, taskID)
	if err != nil {
		return err
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return fmt.Errorf("failed to marshal crawl result: %w", err)
	}

	if _, err := t.ResultWriter().Write(resultJSON); err != nil {
		h.logger.Error("Failed to write crawl result", "error", err, "task_id", taskID)
	}

	// Fire the completion webhook (best-effort; failures are logged only).
	if payload.WebhookURL != "" {
		if err := Deliver(ctx, payload.WebhookURL, payload.WebhookSecret, resultJSON); err != nil {
			h.logger.Warn("Webhook delivery failed", "url", payload.WebhookURL, "error", err, "task_id", taskID)
		} else {
			h.logger.Info("Webhook delivered", "url", payload.WebhookURL, "task_id", taskID)
		}
	}

	return nil
}

// ExecuteCrawl performs a parallel BFS crawl. Pages at the same depth are
// scraped concurrently (up to crawlConcurrency workers) while respecting a
// per-domain politeness delay, retrying transient failures with backoff,
// and never retrying 4xx errors.
func (h *CrawlTaskHandler) ExecuteCrawl(ctx context.Context, payload CrawlPayload, taskID string) (*CrawlResult, error) {
	// Apply sensible defaults and caps
	maxDepth := payload.MaxDepth
	if maxDepth <= 0 {
		maxDepth = 2
	}
	if maxDepth > 10 {
		maxDepth = 10
	}

	limit := payload.Limit
	if limit <= 0 {
		limit = 10
	}
	if limit > 100 {
		limit = 100
	}

	// Resolve scraping mode
	mode := payload.Mode
	if payload.Render {
		mode = "dynamic"
	}
	if mode == "" {
		mode = "smart"
	}

	seedURL, err := url.Parse(payload.URL)
	if err != nil {
		return nil, fmt.Errorf("invalid seed URL %q: %w", payload.URL, err)
	}
	allowedHost := seedURL.Hostname()

	h.logger.Info("Starting crawl",
		"url", payload.URL,
		"maxDepth", maxDepth,
		"limit", limit,
		"mode", mode,
		"concurrency", crawlConcurrency(),
		"task_id", taskID,
	)

	scrapeOpts := domain.ScrapeOptions{
		Screenshot: payload.Screenshot,
		Images:     payload.Images,
	}
	switch payload.ImageFormat {
	case "blob":
		scrapeOpts.ImageFormat = domain.ImageFormatBlob
	case "url":
		scrapeOpts.ImageFormat = domain.ImageFormatURL
	}

	type queueEntry struct {
		url   string
		depth int
	}

	queue := make(chan queueEntry, crawlConcurrency()*2)

	var mu sync.Mutex
	visited := map[string]bool{normalizeURL(payload.URL): true}
	var results []domain.ScrapeResult
	var failed []FailedURL
	lastRequest := make(map[string]time.Time) // per-host politeness
	pending := 1                              // the seed entry

	var closeOnce sync.Once
	closeQueue := func() { closeOnce.Do(func() { close(queue) }) }

	// Workers exit early when the crawl is cancelled.
	go func() {
		<-ctx.Done()
		closeQueue()
	}()

	queue <- queueEntry{url: normalizeURL(payload.URL), depth: 0}

	var wg sync.WaitGroup
	worker := func() {
		defer wg.Done()
		for entry := range queue {
			if ctx.Err() != nil {
				return
			}

			mu.Lock()
			overLimit := len(results) >= limit
			mu.Unlock()
			if overLimit {
				closeQueue()
				return
			}

			// Per-domain politeness: enforce a minimum interval between
			// requests to the same host across all workers.
			host := hostOf(entry.url)
			delay := time.Duration(0)
			mu.Lock()
			if last, ok := lastRequest[host]; ok {
				delay = domainDelay() - time.Since(last)
				if delay < 0 {
					delay = 0
				}
			}
			lastRequest[host] = time.Now().Add(delay)
			mu.Unlock()
			if delay > 0 {
				select {
				case <-time.After(delay):
				case <-ctx.Done():
					return
				}
			}

			h.logger.Info("Crawling page",
				"url", entry.url,
				"depth", entry.depth,
				"scraped", len(results),
				"queued", len(queue),
				"task_id", taskID,
			)

			result, scrapeErr := scrapeWithRetry(ctx, h.scraper, entry.url, mode, scrapeOpts)
			if scrapeErr != nil {
				mu.Lock()
				failed = append(failed, FailedURL{URL: entry.url, Error: scrapeErr.Error()})
				pending--
				empty := pending == 0
				mu.Unlock()
				if empty {
					closeQueue()
				}
				continue
			}

			mu.Lock()
			if len(results) >= limit {
				mu.Unlock()
				closeQueue()
				return
			}
			results = append(results, *result)
			mu.Unlock()

			// If we haven't reached maxDepth, extract links and enqueue.
			if entry.depth < maxDepth {
				links := filterByPatterns(extractLinks(result.HTML, entry.url, allowedHost),
					payload.IncludePaths, payload.ExcludePaths)

				var newEntries []queueEntry
				mu.Lock()
				for _, link := range links {
					normalized := normalizeURL(link)
					if visited[normalized] {
						continue
					}
					if len(results)+len(newEntries) >= limit*3 {
						break // cap queue growth; BFS already has enough work
					}
					visited[normalized] = true
					pending++
					newEntries = append(newEntries, queueEntry{url: normalized, depth: entry.depth + 1})
				}
				mu.Unlock()

				for _, e := range newEntries {
					select {
					case queue <- e:
					case <-ctx.Done():
						return
					}
				}
			}

			mu.Lock()
			pending--
			empty := pending == 0
			mu.Unlock()
			if empty {
				closeQueue()
			}
		}
	}

	for i := 0; i < crawlConcurrency(); i++ {
		wg.Add(1)
		go worker()
	}
	wg.Wait()

	status := "completed"
	if ctx.Err() != nil {
		status = "cancelled"
	} else if len(failed) > 0 && len(results) == 0 {
		status = "failed"
	} else if len(failed) > 0 {
		status = "partial"
	}

	crawlResult := &CrawlResult{
		Status:     status,
		TotalPages: len(results),
		MaxDepth:   maxDepth,
		Limit:      limit,
		Pages:      results,
		FailedURLs: failed,
	}

	h.logger.Info("Crawl completed",
		"url", payload.URL,
		"totalPages", len(results),
		"failedPages", len(failed),
		"status", status,
		"task_id", taskID,
	)

	return crawlResult, nil
}

// scrapeWithRetry scrapes a URL with up to 2 retries and exponential
// backoff (1s, 2s) for transient failures. 4xx errors are never retried.
func scrapeWithRetry(ctx context.Context, svc *scraper.Service, url, mode string, opts domain.ScrapeOptions) (*domain.ScrapeResult, error) {
	var lastErr error
	for attempt := 0; attempt <= 2; attempt++ {
		if attempt > 0 {
			select {
			case <-time.After(time.Duration(attempt) * time.Second):
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		}
		result, err := svc.Scrape(ctx, url, mode, opts)
		if err == nil {
			return result, nil
		}
		lastErr = err
		var statusErr *scraper.StatusError
		if errors.As(err, &statusErr) && statusErr.StatusCode >= 400 && statusErr.StatusCode < 500 {
			return nil, err
		}
	}
	return nil, lastErr
}

// crawlConcurrency returns the worker pool size (env CRAWL_CONCURRENCY,
// default 4, capped at 10).
func crawlConcurrency() int {
	return clampEnvInt("CRAWL_CONCURRENCY", 4, 1, 10)
}

// domainDelay returns the minimum interval between requests to the same
// host (env CRAWL_DOMAIN_DELAY seconds, default 1s).
func domainDelay() time.Duration {
	return time.Duration(clampEnvInt("CRAWL_DOMAIN_DELAY", 1, 0, 60)) * time.Second
}

// clampEnvInt reads an integer env var, returning fallback when unset or
// invalid, clamped to [min, max].
func clampEnvInt(key string, fallback, min, max int) int {
	raw := os.Getenv(key)
	if raw == "" {
		return fallback
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n < min {
		return fallback
	}
	if n > max {
		return max
	}
	return n
}

// hostOf returns the hostname of a URL (empty on parse failure).
func hostOf(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return ""
	}
	return u.Hostname()
}

// filterByPatterns applies include/exclude glob patterns (matched against
// the URL path). Exclusions win; an empty include list allows everything.
func filterByPatterns(links []string, include, exclude []string) []string {
	if len(include) == 0 && len(exclude) == 0 {
		return links
	}
	out := make([]string, 0, len(links))
	for _, link := range links {
		u, err := url.Parse(link)
		if err != nil {
			continue
		}
		p := u.Path
		if matchesAny(exclude, p) {
			continue
		}
		if len(include) > 0 && !matchesAny(include, p) {
			continue
		}
		out = append(out, link)
	}
	return out
}

// matchesAny reports whether path matches any glob pattern.
func matchesAny(patterns []string, p string) bool {
	for _, pat := range patterns {
		if ok, _ := path.Match(pat, p); ok {
			return true
		}
	}
	return false
}

// extractLinks parses HTML and returns same-domain, non-resource links.
func extractLinks(htmlBody string, pageURL string, allowedHost string) []string {
	if htmlBody == "" {
		return nil
	}

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(htmlBody))
	if err != nil {
		return nil
	}

	base, err := url.Parse(pageURL)
	if err != nil {
		return nil
	}

	var links []string
	seen := make(map[string]bool)

	doc.Find("a[href]").Each(func(_ int, s *goquery.Selection) {
		href, exists := s.Attr("href")
		if !exists || href == "" {
			return
		}

		// Skip non-HTTP schemes (mailto:, tel:, javascript:, #anchors)
		if strings.HasPrefix(href, "mailto:") ||
			strings.HasPrefix(href, "tel:") ||
			strings.HasPrefix(href, "javascript:") ||
			strings.HasPrefix(href, "#") {
			return
		}

		parsed, err := url.Parse(href)
		if err != nil {
			return
		}

		// Resolve relative to absolute
		resolved := base.ResolveReference(parsed)

		// Domain lock: only follow same-host links
		if resolved.Hostname() != allowedHost {
			return
		}

		// Skip non-HTML resource links
		if isResourceFile(resolved.Path) {
			return
		}

		// Strip fragment
		resolved.Fragment = ""

		canonical := resolved.String()
		if !seen[canonical] {
			seen[canonical] = true
			links = append(links, canonical)
		}
	})

	return links
}

// normalizeURL strips fragments and trailing slashes for deduplication.
func normalizeURL(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return rawURL
	}
	u.Fragment = ""
	result := u.String()
	// Remove trailing slash for consistency (except root "/")
	if len(result) > 1 && strings.HasSuffix(result, "/") {
		result = strings.TrimRight(result, "/")
	}
	return result
}

// isResourceFile returns true if the URL path points to a non-HTML resource.
func isResourceFile(urlPath string) bool {
	ext := strings.ToLower(path.Ext(urlPath))
	resourceExts := map[string]bool{
		".pdf": true, ".jpg": true, ".jpeg": true, ".png": true,
		".gif": true, ".svg": true, ".webp": true, ".ico": true,
		".mp4": true, ".mp3": true, ".avi": true, ".mov": true,
		".zip": true, ".tar": true, ".gz": true, ".rar": true,
		".css": true, ".js": true, ".woff": true, ".woff2": true,
		".ttf": true, ".eot": true, ".xml": true, ".rss": true,
		".json": true, ".txt": true,
	}
	return resourceExts[ext]
}
