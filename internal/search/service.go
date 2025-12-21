package search

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
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
	Title       string `json:"title"`
	URL         string `json:"url"`
	Description string `json:"description"`
}

// Service defines the search service interface
type Service interface {
	Search(ctx context.Context, query string) ([]Result, error)
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

// Search performs a search on Brave Search
func (s *BraveService) Search(ctx context.Context, query string) ([]Result, error) {
	if s.apiKey == "" {
		return nil, fmt.Errorf("brave search api key is missing")
	}

	// Wait for rate limiter
	if err := s.limiter.Wait(ctx); err != nil {
		return nil, fmt.Errorf("rate limit wait: %w", err)
	}

	endpoint := "https://api.search.brave.com/res/v1/web/search"

	// Create request
	req, err := http.NewRequestWithContext(ctx, "GET", endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Add query parameters
	q := req.URL.Query()
	q.Add("q", query)
	q.Add("count", "10") // Default to 10 results
	req.URL.RawQuery = q.Encode()

	// Set headers
	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-Subscription-Token", s.apiKey)

	// Execute request
	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("brave search api error: status %d", resp.StatusCode)
	}

	// Parse JSON
	var braveResponse BraveSearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&braveResponse); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// Convert to Result
	var results []Result
	for _, item := range braveResponse.Web.Results {
		results = append(results, Result{
			Title:       item.Title,
			URL:         item.URL,
			Description: item.Description,
		})
	}

	return results, nil
}
