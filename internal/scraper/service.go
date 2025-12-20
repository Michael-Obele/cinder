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

func (s *Service) Scrape(ctx context.Context, url string, render bool) (*domain.ScrapeResult, error) {
	// 1. Try Cache
	cacheKey := fmt.Sprintf("scrape:%s:%t", url, render)
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

	if render {
		if s.chromedp == nil {
			return nil, fmt.Errorf("dynamic scraper not configured")
		}
		result, err = s.chromedp.Scrape(ctx, url)
	} else {
		if s.colly == nil {
			return nil, fmt.Errorf("static scraper not configured")
		}
		result, err = s.colly.Scrape(ctx, url)
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
