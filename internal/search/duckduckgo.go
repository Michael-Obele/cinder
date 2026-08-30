package search

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"golang.org/x/time/rate"
)

const duckEndpoint = "https://html.duckduckgo.com/html/"

// DuckService scrapes DuckDuckGo HTML search — no API key needed.
type DuckService struct {
	client   *http.Client
	limiter  *rate.Limiter
	endpoint string
}

func NewDuckService() *DuckService {
	return &DuckService{
		client:   &http.Client{Timeout: 15 * time.Second},
		limiter:  rate.NewLimiter(rate.Every(1500*time.Millisecond), 1),
		endpoint: duckEndpoint,
	}
}

// searchRetries is how many times a transient DuckDuckGo failure is retried
// before giving up. DDG rate-limits and intermittently 429s under load; a
// short backoff usually clears it.
const searchRetries = 2

// retryableDDG reports whether a DuckDuckGo failure is worth retrying.
// Malformed HTML and 4xx (other than 429) are definitive; 429, 5xx, and
// network errors are transient.
func retryableDDG(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	switch {
	case strings.Contains(msg, "duckduckgo status 429"),
		strings.Contains(msg, "duckduckgo status 5"),
		strings.Contains(msg, "duckduckgo request:"),
		strings.Contains(msg, "context deadline exceeded"),
		strings.Contains(msg, "Client.Timeout"):
		return true
	}
	return false
}

func (s *DuckService) Search(ctx context.Context, opts SearchOptions) ([]Result, int, error) {
	if opts.Limit == 0 {
		opts.Limit = 10
	}
	if opts.Limit > 100 {
		opts.Limit = 100
	}

	var lastErr error
	for attempt := 0; attempt <= searchRetries; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return nil, 0, ctx.Err()
			case <-time.After(time.Duration(attempt) * 500 * time.Millisecond):
			}
		}
		results, total, err := s.searchOnce(ctx, opts)
		if err == nil {
			return results, total, nil
		}
		lastErr = err
		if !retryableDDG(err) {
			return nil, 0, err
		}
	}
	return nil, 0, lastErr
}

// searchOnce performs a single DuckDuckGo HTML query, honoring the rate
// limiter so upstream never sees a burst.
func (s *DuckService) searchOnce(ctx context.Context, opts SearchOptions) ([]Result, int, error) {
	if err := s.limiter.Wait(ctx); err != nil {
		return nil, 0, fmt.Errorf("rate limit wait: %w", err)
	}
	endpoint := s.endpoint
	if endpoint == "" {
		endpoint = duckEndpoint
	}
	u, _ := url.Parse(endpoint)
	q := u.Query()
	q.Set("q", opts.Query)
	// DuckDuckGo html pagination: s param is offset-ish, plus dc for continuation
	if opts.Offset > 0 {
		q.Set("s", fmt.Sprintf("%d", opts.Offset))
		// dc is next-page token derived from offset; best-effort
		q.Set("dc", fmt.Sprintf("%d", opts.Offset+1))
	}
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, "GET", u.String(), nil)
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36")
	req.Header.Set("Accept", "text/html")

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("duckduckgo request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, 0, fmt.Errorf("duckduckgo status %d", resp.StatusCode)
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, 0, fmt.Errorf("parse html: %w", err)
	}

	var results []Result
	doc.Find(".result").Each(func(i int, sel *goquery.Selection) {
		if len(results) >= opts.Limit {
			return
		}
		titleSel := sel.Find(".result__title a, .result__a").First()
		title := strings.TrimSpace(titleSel.Text())
		href, _ := titleSel.Attr("href")
		// DuckDuckGo wraps via /l/?uddg=<encoded>
		if strings.HasPrefix(href, "/l/") || strings.Contains(href, "uddg=") {
			if parsed, err := url.Parse(href); err == nil {
				if v := parsed.Query().Get("uddg"); v != "" {
					if decoded, err := url.QueryUnescape(v); err == nil {
						href = decoded
					} else {
						href = v
					}
				}
			}
		} else if strings.HasPrefix(href, "//") {
			href = "https:" + href
		}
		snip := strings.TrimSpace(sel.Find(".result__snippet").Text())
		// fallback snippet selectors
		if snip == "" {
			snip = strings.TrimSpace(sel.Find(".result__extras__url").Text())
		}
		if title == "" || href == "" || strings.HasPrefix(href, "/") {
			return
		}
		results = append(results, Result{
			Title:       title,
			URL:         href,
			Description: snip,
			ID:          fmt.Sprintf("%s_%d", opts.Query, opts.Offset+len(results)),
			Domain:      extractDomain(href),
			Relevance:   1.0 - float64(len(results))*0.05,
		})
	})

	estimatedTotal := opts.Offset + len(results)
	if len(results) >= opts.Limit {
		estimatedTotal += 100
	}
	return results, estimatedTotal, nil
}
