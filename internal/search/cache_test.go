package search

import (
	"errors"
	"testing"
)

func TestRetryableDDG(t *testing.T) {
	tests := []struct {
		name  string
		err   error
		retry bool
	}{
		{name: "429 is retryable", err: errors.New("duckduckgo status 429"), retry: true},
		{name: "5xx is retryable", err: errors.New("duckduckgo status 503"), retry: true},
		{name: "network error is retryable", err: errors.New("duckduckgo request: dial tcp: i/o timeout"), retry: true},
		{name: "client timeout is retryable", err: errors.New("Get \"...\": context deadline exceeded"), retry: true},
		{name: "404 is not retryable", err: errors.New("duckduckgo status 404"), retry: false},
		{name: "parse error is not retryable", err: errors.New("parse html: unexpected EOF"), retry: false},
		{name: "nil is not retryable", err: nil, retry: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := retryableDDG(tt.err); got != tt.retry {
				t.Errorf("retryableDDG(%v) = %v, want %v", tt.err, got, tt.retry)
			}
		})
	}
}

func TestSearchCacheKeyDeterministic(t *testing.T) {
	a := SearchOptions{Query: "golang", Limit: 10, IncludeDomains: []string{"go.dev"}}
	b := SearchOptions{Query: "golang", Limit: 10, IncludeDomains: []string{"go.dev"}}
	c := SearchOptions{Query: "golang", Limit: 10, IncludeDomains: []string{"github.com"}}

	if searchCacheKey(a) != searchCacheKey(b) {
		t.Error("identical options must produce identical cache keys")
	}
	if searchCacheKey(a) == searchCacheKey(c) {
		t.Error("different filters must produce different cache keys")
	}
}
