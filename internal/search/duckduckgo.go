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

func (s *DuckService) Search(ctx context.Context, opts SearchOptions) ([]Result, int, error) {
	if opts.Limit == 0 {
		opts.Limit = 10
	}
	if opts.Limit > 100 {
		opts.Limit = 100
	}
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
