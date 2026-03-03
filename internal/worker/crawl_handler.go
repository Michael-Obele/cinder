package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/url"
	"path"
	"strings"

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

	return nil
}

// ExecuteCrawl performs the actual BFS crawl. Extracted from ProcessTask
// so it can be tested independently of Asynq infrastructure.
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
		"task_id", taskID,
	)

	// BFS state
	visited := make(map[string]bool)
	var results []domain.ScrapeResult
	var failed []FailedURL

	// Queue entries: (url, depth)
	type queueEntry struct {
		url   string
		depth int
	}
	queue := []queueEntry{{url: normalizeURL(payload.URL), depth: 0}}
	visited[normalizeURL(payload.URL)] = true

	for len(queue) > 0 && len(results) < limit {
		// Check context cancellation
		if ctx.Err() != nil {
			h.logger.Warn("Crawl cancelled", "task_id", taskID, "reason", ctx.Err())
			break
		}

		// Dequeue
		entry := queue[0]
		queue = queue[1:]

		h.logger.Info("Crawling page",
			"url", entry.url,
			"depth", entry.depth,
			"scraped", len(results),
			"queued", len(queue),
			"task_id", taskID,
		)

		// Scrape the page
		result, scrapeErr := h.scraper.Scrape(ctx, entry.url, mode, domain.ScrapeOptions{
			Screenshot: payload.Screenshot,
			Images:     payload.Images,
		})
		if scrapeErr != nil {
			h.logger.Warn("Failed to scrape page during crawl",
				"url", entry.url, "error", scrapeErr,
				"task_id", taskID,
			)
			failed = append(failed, FailedURL{URL: entry.url, Error: scrapeErr.Error()})
			continue
		}

		results = append(results, *result)

		// If we haven't reached maxDepth, extract links and enqueue
		if entry.depth < maxDepth && len(results) < limit {
			links := extractLinks(result.HTML, entry.url, allowedHost)
			for _, link := range links {
				normalized := normalizeURL(link)
				if !visited[normalized] && len(queue)+len(results) < limit {
					visited[normalized] = true
					queue = append(queue, queueEntry{url: normalized, depth: entry.depth + 1})
				}
			}
		}
	}

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
