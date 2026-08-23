package search

import (
	"context"
	"log/slog"
)

// HybridService tries DuckDuckGo first (free), falls back to Brave if configured and DDG fails/empty.
type HybridService struct {
	primary  Service
	fallback Service // nil if no Brave key
}

func NewHybridService(braveAPIKey string) Service {
	duck := NewDuckService()
	if braveAPIKey == "" {
		slog.Info("search: using DuckDuckGo only (no BRAVE_SEARCH_API_KEY)")
		return duck
	}
	slog.Info("search: using DuckDuckGo with Brave fallback")
	return &HybridService{primary: duck, fallback: NewBraveService(braveAPIKey)}
}

func (h *HybridService) Search(ctx context.Context, opts SearchOptions) ([]Result, int, error) {
	results, total, err := h.primary.Search(ctx, opts)
	if err == nil && len(results) > 0 {
		return results, total, nil
	}
	if h.fallback == nil {
		if err != nil {
			return nil, 0, err
		}
		return results, total, nil
	}
	slog.Info("search: primary failed/empty, falling back to Brave", "error", err)
	return h.fallback.Search(ctx, opts)
}
