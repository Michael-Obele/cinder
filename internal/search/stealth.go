package search

import (
	"context"
	"net/http"
	"strings"
	"time"
)

// BrowserFetcher fetches HTML for a URL in a stealth browser tab.
// ChromedpScraper implements this; tests use a stub.
// This is a minimal stub for Task 1 config hardening — full implementation
// lands in Task 3. It compiles so existing tests can run, but keeps red
// tests failing as expected.
type BrowserFetcher interface {
	FetchHTML(ctx context.Context, url string) (string, error)
}

// StealthService is a stub implementing Service via a browser fetcher.
// Full Brave HTML parsing and UA rotation will be implemented in Task 3.
type StealthService struct {
	fetcher  BrowserFetcher
	endpoint string
	client   *http.Client
}

// NewStealthService creates a StealthService. Endpoint defaults to
// https://search.brave.com when empty.
func NewStealthService(fetcher BrowserFetcher, endpoint string) *StealthService {
	if endpoint == "" {
		endpoint = "https://search.brave.com"
	}
	return &StealthService{
		fetcher:  fetcher,
		endpoint: strings.TrimRight(endpoint, "/"),
		client:   &http.Client{Timeout: 20 * time.Second},
	}
}

// Search is a stub that always returns empty results. This keeps red tests
// failing (they expect parsed results) while allowing the package to compile
// so `go test -run TestNewHybridService` passes.
func (s *StealthService) Search(_ context.Context, _ SearchOptions) ([]Result, int, error) {
	return nil, 0, nil
}
