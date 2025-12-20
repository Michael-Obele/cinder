package scraper

import (
	"context"
	"fmt"

	"github.com/standard-user/cinder/internal/domain"
)

// Service acts as the main entry point and chooses the right scraper
type Service struct {
	colly    domain.Scraper
	chromedp domain.Scraper
}

func NewService(colly domain.Scraper, chromedp domain.Scraper) *Service {
	return &Service{
		colly:    colly,
		chromedp: chromedp,
	}
}

func (s *Service) Scrape(ctx context.Context, url string, render bool) (*domain.ScrapeResult, error) {
	if render {
		if s.chromedp == nil {
			return nil, fmt.Errorf("dynamic scraper not configured")
		}
		return s.chromedp.Scrape(ctx, url)
	}
	
	if s.colly == nil {
		return nil, fmt.Errorf("static scraper not configured")
	}
	return s.colly.Scrape(ctx, url)
}
