package search

import (
	"context"
	"net/http"
	"strings"
	"time"
)

// BrowserFetcher fetches HTML for a URL in a stealth browser tab.
// TODO(Task 3): This interface lives in search but will be implemented by
// scraper.ChromedpScraper — cross-package import would create a cycle.
// Task 3 should move the interface to internal/domain or inject via func
// to avoid circular dependency; for now the stub keeps the package compiling.
// In production scraper.ChromedpScraper will satisfy this; tests use a stub.
type BrowserFetcher interface {
	FetchHTML(ctx context.Context, url string) (string, error)
}

// StealthService is a Task 1 stub — will be replaced in Task 3 with full
// Brave HTML parsing and UA rotation. It exists only so the package compiles
// and `go test -run TestNewHybridService` / `go vet` / `staticcheck` pass
// while red tests (stealth_test.go) remain failing as TDD scaffolding.
type StealthService struct {
	fetcher  BrowserFetcher
	endpoint string
	// TODO(Task 3): client is unused in Task 1 stub (Search returns empty);
	// real HTTP fetch in Task 3 will use it for Brave HTML retrieval.
	client *http.Client
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

// Search is a Task 1 stub — will be replaced in Task 3. Always returns empty
// results so red tests stay failing while the package compiles.
func (s *StealthService) Search(_ context.Context, _ SearchOptions) ([]Result, int, error) {
	return nil, 0, nil
}
