package scraper

import (
	"context"
	"fmt"

	"encoding/json"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/standard-user/cinder/internal/domain"
)

// Service acts as the main entry point and chooses the right scraper
type Service struct {
	colly    domain.Scraper
	chromedp domain.Scraper
	redis    *redis.Client
}

func NewService(colly domain.Scraper, chromedp domain.Scraper, redis *redis.Client) *Service {
	return &Service{
		colly:    colly,
		chromedp: chromedp,
		redis:    redis,
	}
}

func (s *Service) Scrape(ctx context.Context, url string, mode string) (*domain.ScrapeResult, error) {
	// Default to smart if empty
	if mode == "" {
		mode = "smart"
	}

	// 1. Try Cache
	cacheKey := fmt.Sprintf("scrape:%s:%s", url, mode)
	if s.redis != nil {
		val, err := s.redis.Get(ctx, cacheKey).Result()
		if err == nil {
			var result domain.ScrapeResult
			if err := json.Unmarshal([]byte(val), &result); err == nil {
				if result.Metadata == nil {
					result.Metadata = make(map[string]string)
				}
				result.Metadata["cached"] = "true"
				return &result, nil
			}
		}
	}

	// 2. Scrape
	var result *domain.ScrapeResult
	var err error

	// Helper to run dynamic
	runDynamic := func() (*domain.ScrapeResult, error) {
		if s.chromedp == nil {
			return nil, fmt.Errorf("dynamic scraper not configured")
		}
		return s.chromedp.Scrape(ctx, url)
	}

	// Helper to run static
	runStatic := func() (*domain.ScrapeResult, error) {
		if s.colly == nil {
			return nil, fmt.Errorf("static scraper not configured")
		}
		return s.colly.Scrape(ctx, url)
	}

	switch mode {
	case "dynamic":
		result, err = runDynamic()
	case "static":
		result, err = runStatic()
	case "smart":
		// Fallthrough to smart logic
		// 1. Try static first (fast & cheap)
		result, err = runStatic()

		// If static failed or produced suspicious content, try dynamic
		needsDynamic := false
		if err != nil {
			// If it's a 403/Forbidden, dynamic might help (headless often gets past basic blocks)
			// For now, let's treat errors as candidates for retry if we want robustness.
			// However, simple connectivity errors won't be fixed by headless.
			// Let's focus on content heuristics for now.
			// If err IS NOT nil, we return it, UNLESS we want to be very aggressive.
			// Let's keep it simple: if static fails, we fail, unless it's a specific "empty response" error we added.
		} else if result != nil {
			// Check heuristics
			if ShouldUseDynamic(result.HTML) {
				needsDynamic = true
			}
		}

		if needsDynamic {
			fmt.Printf("Smart Scraper: Heuristics detected dynamic content for %s. Switching to Chromedp.\n", url)
			dynamicResult, dynErr := runDynamic()
			if dynErr == nil {
				result = dynamicResult
			} else {
				// If dynamic fails but static succeeded, return static?
				// Or fail because the page is likely broken?
				// Let's return dynamic error if static was deemed insufficient.
				err = dynErr
			}
		}
	default:
		return nil, fmt.Errorf("unknown mode: %s", mode)
	}

	if err != nil {
		return nil, err
	}

	// 3. Save to Cache
	if s.redis != nil {
		data, err := json.Marshal(result)
		if err == nil {
			s.redis.Set(ctx, cacheKey, data, 24*time.Hour)
		}
	}

	return result, nil
}
