package search

import (
	"context"
	"errors"
	"log/slog"
)

// HybridService tries a chain of search backends in order and returns the
// first non-empty result set. Backends are ordered cheap-to-robust: a
// self-hosted SearXNG when configured, then Brave (only when an API key is
// set). A backend that errors or returns nothing falls through to the next,
// so a single engine outage never kills search.
type HybridService struct {
	services []Service
}

// NewHybridService builds the search backend chain from configuration.
//
//   - braveAPIKey: optional Brave Search API key. When set it is the
//     fallback when SearXNG is unavailable or unconfigured (free tier:
//     ~2000 queries/month at 1 QPS).
//   - searxngEndpoint: self-hosted SearXNG base URL. When set it is the
//     primary backend — free, aggregates many engines (Google, Bing,
//     DuckDuckGo, Brave, Mojeek, Wikipedia, ...), and has no per-query cost.
//
// When neither is configured, a Service is returned that fails with a clear
// message so a misconfigured deployment fails loudly instead of silently
// returning empty results.
func NewHybridService(braveAPIKey, searxngEndpoint string) Service {
	var chain []Service
	if searxngEndpoint != "" {
		chain = append(chain, NewSearXNGService(searxngEndpoint))
	}
	if braveAPIKey != "" {
		chain = append(chain, NewBraveService(braveAPIKey))
	}

	switch len(chain) {
	case 0:
		return noBackendService{}
	case 1:
		return chain[0]
	default:
		return &HybridService{services: chain}
	}
}

func (h *HybridService) Search(ctx context.Context, opts SearchOptions) ([]Result, int, error) {
	var lastErr error
	for _, s := range h.services {
		results, total, err := s.Search(ctx, opts)
		if err == nil && len(results) > 0 {
			return results, total, nil
		}
		if err != nil {
			lastErr = err
			slog.Info("search: backend failed, trying next", "error", err)
		}
	}
	if lastErr != nil {
		return nil, 0, lastErr
	}
	return nil, 0, nil
}

// noBackendService fails every search with a configuration error. It is
// returned when neither SearXNG nor Brave is configured, so operators see
// exactly what to set instead of an empty result set.
type noBackendService struct{}

func (noBackendService) Search(context.Context, SearchOptions) ([]Result, int, error) {
	return nil, 0, errors.New("search is not configured: set SEARXNG_ENDPOINT (recommended) or BRAVE_SEARCH_API_KEY")
}
