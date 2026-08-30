package search

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// SearXNGService implements Service against a self-hosted SearXNG
// meta-search instance. SearXNG aggregates many upstream engines (Google,
// Bing, DuckDuckGo, Brave, Mojeek, Wikipedia, ...), so a single instance
// removes the single-point-of-failure of scraping one engine directly and
// costs nothing beyond one container.
//
// There is deliberately NO client-side rate limiter here: SearXNG is
// self-hosted and handles concurrent searches internally (its per-engine
// rate limiting lives in its own settings). A client limiter would serialize
// every query to ~1 req/s and turn concurrent load into a queue — exactly
// what the load benchmark caught before it was removed.
type SearXNGService struct {
	client   *http.Client
	endpoint string
}

// NewSearXNGService creates a service pointing at a SearXNG base URL such as
// "http://localhost:8889". The JSON API must be enabled on the instance
// (search.formats includes json).
func NewSearXNGService(endpoint string) *SearXNGService {
	return &SearXNGService{
		client:   &http.Client{Timeout: 20 * time.Second},
		endpoint: strings.TrimRight(endpoint, "/"),
	}
}

// searxngResult mirrors the JSON shape of a single SearXNG result.
type searxngResult struct {
	URL           string   `json:"url"`
	Title         string   `json:"title"`
	Content       string   `json:"content"`
	Engine        string   `json:"engine"`
	Score         *float64 `json:"score"`
	PublishedDate *string  `json:"publishedDate"`
}

// searxngResponse mirrors the top-level JSON shape of a SearXNG search.
type searxngResponse struct {
	Query           string          `json:"query"`
	NumberOfResults int             `json:"number_of_results"`
	Results         []searxngResult `json:"results"`
}

func (s *SearXNGService) Search(ctx context.Context, opts SearchOptions) ([]Result, int, error) {
	if opts.Limit == 0 {
		opts.Limit = 10
	}
	if opts.Limit > 100 {
		opts.Limit = 100
	}

	u, err := url.Parse(s.endpoint + "/search")
	if err != nil {
		return nil, 0, fmt.Errorf("invalid searxng endpoint: %w", err)
	}
	q := u.Query()
	q.Set("q", opts.Query)
	q.Set("format", "json")
	if opts.Offset > 0 {
		// SearXNG paginates by page number; approximate offset as pageno.
		q.Set("pageno", fmt.Sprintf("%d", opts.Offset/opts.Limit+1))
	}
	if opts.MaxAge != nil {
		switch *opts.MaxAge {
		case 1:
			q.Set("time_range", "day")
		case 7:
			q.Set("time_range", "week")
		case 30:
			q.Set("time_range", "month")
		}
	}
	// Map Exa-style category hints onto SearXNG categories where possible.
	if opts.Mode != "" {
		switch opts.Mode {
		case "news":
			q.Set("categories", "news")
		case "code":
			q.Set("categories", "it")
		}
	}
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, "GET", u.String(), nil)
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36")
	req.Header.Set("Accept", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("searxng request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, 0, fmt.Errorf("searxng status %d", resp.StatusCode)
	}

	var out searxngResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, 0, fmt.Errorf("searxng parse: %w", err)
	}

	results := make([]Result, 0, len(out.Results))
	for i, r := range out.Results {
		if len(results) >= opts.Limit {
			break
		}
		if r.URL == "" || r.Title == "" {
			continue
		}
		relevance := 0.5
		if r.Score != nil {
			relevance = *r.Score
		}
		results = append(results, Result{
			Title:       r.Title,
			URL:         r.URL,
			Description: r.Content,
			ID:          fmt.Sprintf("%s_%d", opts.Query, opts.Offset+i),
			Domain:      extractDomain(r.URL),
			Relevance:   relevance,
		})
	}

	total := out.NumberOfResults
	if total == 0 {
		total = opts.Offset + len(results)
	}
	if len(results) >= opts.Limit {
		// SearXNG does not expose a reliable total; signal more exist.
		total = opts.Offset + len(results) + 1
	}
	return results, total, nil
}
