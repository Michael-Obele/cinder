package search

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"golang.org/x/time/rate"
)

// BraveSearchResponse represents the JSON response from Brave Search
type BraveSearchResponse struct {
	Web struct {
		Results []struct {
			Title       string `json:"title"`
			URL         string `json:"url"`
			Description string `json:"description"`
		} `json:"results"`
	} `json:"web"`
}

// Result represents a single search result
type Result struct {
	Title       string  `json:"title"`
	URL         string  `json:"url"`
	Description string  `json:"description"`
	ID          string  `json:"id"`
	Domain      string  `json:"domain"`
	Relevance   float64 `json:"relevance"`
}

// SearchOptions contains options for the search
type SearchOptions struct {
	Query          string
	Offset         int
	Limit          int
	IncludeDomains []string
	ExcludeDomains []string
	RequiredText   []string
	MaxAge         *int
	Mode           string
}

// Service defines the search service interface
type Service interface {
	Search(ctx context.Context, opts SearchOptions) ([]Result, int, error)
}

// BraveService implements Service using Brave Search API
type BraveService struct {
	apiKey  string
	client  *http.Client
	limiter *rate.Limiter
}

// NewBraveService creates a new instance of BraveService
func NewBraveService(apiKey string) *BraveService {
	return &BraveService{
		apiKey: apiKey,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
		// Limit to 1 request per 1.1 seconds to be safe and avoid 429s
		limiter: rate.NewLimiter(rate.Every(1100*time.Millisecond), 1),
	}
}

// Search performs a search on Brave Search with pagination support
// Returns: (results, totalCount, error)
// Note: Brave API doesn't provide totalCount, so we estimate based on typical result patterns
func (s *BraveService) Search(ctx context.Context, opts SearchOptions) ([]Result, int, error) {
	if s.apiKey == "" {
		return nil, 0, fmt.Errorf("brave search api key is missing")
	}

	// Set defaults
	if opts.Limit == 0 {
		opts.Limit = 10
	}
	if opts.Limit > 100 {
		opts.Limit = 100
	}

	// Wait for rate limiter
	if err := s.limiter.Wait(ctx); err != nil {
		return nil, 0, fmt.Errorf("rate limit wait: %w", err)
	}

	endpoint := "https://api.search.brave.com/res/v1/web/search"

	// Create request
	req, err := http.NewRequestWithContext(ctx, "GET", endpoint, nil)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to create request: %w", err)
	}

	// Add query parameters with pagination support
	q := req.URL.Query()
	q.Add("q", opts.Query)
	q.Add("count", fmt.Sprintf("%d", opts.Limit))
	if opts.Offset > 0 {
		q.Add("offset", fmt.Sprintf("%d", opts.Offset))
	}

	// Add filtering options if provided
	if opts.Mode != "" {
		// Note: Mode is for MCP layer speed control, not passed to Brave
		// But could implement with freshness parameter
		if opts.Mode == "fast" {
			q.Add("freshness", "day") // Recent results only
		}
	}

	if opts.MaxAge != nil {
		// Convert MaxAge to freshness parameter
		switch *opts.MaxAge {
		case 1:
			q.Add("freshness", "day")
		case 7:
			q.Add("freshness", "week")
		case 30:
			q.Add("freshness", "month")
		}
	}

	req.URL.RawQuery = q.Encode()

	// Set headers
	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-Subscription-Token", s.apiKey)

	// Execute request
	resp, err := s.client.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, 0, fmt.Errorf("brave search api error: status %d", resp.StatusCode)
	}

	// Parse JSON
	var braveResponse BraveSearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&braveResponse); err != nil {
		return nil, 0, fmt.Errorf("failed to decode response: %w", err)
	}

	// Convert to Result
	var results []Result
	for idx, item := range braveResponse.Web.Results {
		results = append(results, Result{
			Title:       item.Title,
			URL:         item.URL,
			Description: item.Description,
			ID:          fmt.Sprintf("%s_%d", opts.Query, opts.Offset+idx),
			Domain:      extractDomain(item.URL),
			Relevance:   1.0 - float64(idx)*0.05, // Simple relevance scoring
		})
	}

	// Determine total count
	// Brave API doesn't always provide a stable totalCount field for web results
	// We use the count of returned results and the requested limit to estimate if more exist
	estimatedTotal := opts.Offset + len(results)
	if len(results) >= opts.Limit && opts.Limit > 0 {
		// If we got as many results as we asked for, assume there might be more
		// We'll signal that there are at least 100 more results to keep pagination going
		estimatedTotal += 100
	}

	return results, estimatedTotal, nil
}

// extractDomain extracts domain from URL
func extractDomain(urlStr string) string {
	// Simple extraction - would be improved in production
	// Extract domain from URL string
	if urlStr == "" {
		return ""
	}
	// Parse and get hostname
	u, err := parseURLDomain(urlStr)
	if err != nil {
		return ""
	}
	return u
}

// parseURLDomain is a simple URL domain parser
func parseURLDomain(rawURL string) (string, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return "", err
	}
	return u.Hostname(), nil
}
