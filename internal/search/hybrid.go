package search

import (
	"context"
	"log/slog"
)

// HybridService tries a chain of search backends in order and returns the
// first non-empty result set. Backends are ordered cheapest-to-most-robust:
// a self-hosted SearXNG when configured, then DuckDuckGo (free), then Brave
// (only when an API key is set). A backend that errors or returns nothing
// falls through to the next, so a single engine outage never kills search.
type HybridService struct {
	services []Service
}

// NewHybridService builds the fallback chain from configuration.
//
//   - braveAPIKey: optional Brave Search API key. When set it is the last
//     resort (has a free tier: ~2000 queries/month at 1 QPS).
//   - searxngEndpoint: optional self-hosted SearXNG base URL. When set it
//     becomes the primary backend (free, aggregates many engines).
//
// DuckDuckGo is always in the chain as the free middle tier.
func NewHybridService(braveAPIKey, searxngEndpoint string) Service {
	var chain []Service
	if searxngEndpoint != "" {
		chain = append(chain, NewSearXNGService(searxngEndpoint))
	}
	chain = append(chain, NewDuckService())
	if braveAPIKey != "" {
		chain = append(chain, NewBraveService(braveAPIKey))
	}

	if len(chain) == 1 {
		return chain[0]
	}
	return &HybridService{services: chain}
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
