package scraper

import (
	"context"
	"fmt"
	"testing"

	"github.com/standard-user/cinder/internal/domain"
)

// --- Mock Scrapers ---

type mockScraper struct {
	result *domain.ScrapeResult
	err    error
}

func (m *mockScraper) Scrape(ctx context.Context, url string) (*domain.ScrapeResult, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.result, nil
}

func newMockResult(engine string) *domain.ScrapeResult {
	return &domain.ScrapeResult{
		URL:      "https://example.com",
		Markdown: "# Example",
		HTML:     "<html><body><h1>Example</h1>" + makeStaticHTML() + "</body></html>",
		Metadata: map[string]string{
			"engine": engine,
		},
	}
}

// makeStaticHTML produces HTML large enough that heuristics won't flag it as dynamic
func makeStaticHTML() string {
	s := ""
	for i := 0; i < 100; i++ {
		s += "<p>Paragraph of meaningful content for testing purposes.</p>"
	}
	return s
}

// --- Service Tests ---

func TestService_ScrapeStatic(t *testing.T) {
	colly := &mockScraper{result: newMockResult("colly")}
	chromedp := &mockScraper{result: newMockResult("chromedp")}

	svc := NewService(colly, chromedp, nil)

	result, err := svc.Scrape(context.Background(), "https://example.com", "static")
	if err != nil {
		t.Fatalf("Static scrape failed: %v", err)
	}

	if result.Metadata["engine"] != "colly" {
		t.Errorf("Expected colly engine, got %q", result.Metadata["engine"])
	}
}

func TestService_ScrapeDynamic(t *testing.T) {
	colly := &mockScraper{result: newMockResult("colly")}
	chromedp := &mockScraper{result: newMockResult("chromedp")}

	svc := NewService(colly, chromedp, nil)

	result, err := svc.Scrape(context.Background(), "https://example.com", "dynamic")
	if err != nil {
		t.Fatalf("Dynamic scrape failed: %v", err)
	}

	if result.Metadata["engine"] != "chromedp" {
		t.Errorf("Expected chromedp engine, got %q", result.Metadata["engine"])
	}
}

func TestService_ScrapeSmart_UsesStaticFirst(t *testing.T) {
	colly := &mockScraper{result: newMockResult("colly")}
	chromedp := &mockScraper{result: newMockResult("chromedp")}

	svc := NewService(colly, chromedp, nil)

	result, err := svc.Scrape(context.Background(), "https://example.com", "smart")
	if err != nil {
		t.Fatalf("Smart scrape failed: %v", err)
	}

	// Smart mode tries static first; our mock returns rich content so it stays with colly
	if result.Metadata["engine"] != "colly" {
		t.Errorf("Smart mode should use colly for static content, got %q", result.Metadata["engine"])
	}
}

func TestService_ScrapeSmart_FallsToDynamic(t *testing.T) {
	// Colly returns an SPA shell → heuristics should trigger dynamic
	spaShell := &domain.ScrapeResult{
		URL:      "https://spa.example.com",
		Markdown: "",
		HTML:     `<html><body><div id="root"></div><script src="bundle.js"></script></body></html>`,
		Metadata: map[string]string{"engine": "colly"},
	}
	colly := &mockScraper{result: spaShell}
	chromedp := &mockScraper{result: newMockResult("chromedp")}

	svc := NewService(colly, chromedp, nil)

	result, err := svc.Scrape(context.Background(), "https://spa.example.com", "smart")
	if err != nil {
		t.Fatalf("Smart scrape failed: %v", err)
	}

	if result.Metadata["engine"] != "chromedp" {
		t.Errorf("Smart mode should fall back to chromedp for SPA shells, got %q", result.Metadata["engine"])
	}
}

func TestService_ScrapeUnknownMode(t *testing.T) {
	svc := NewService(nil, nil, nil)

	_, err := svc.Scrape(context.Background(), "https://example.com", "invalid")
	if err == nil {
		t.Error("Expected error for unknown mode")
	}
}

func TestService_ScrapeDefaultMode(t *testing.T) {
	colly := &mockScraper{result: newMockResult("colly")}
	chromedp := &mockScraper{result: newMockResult("chromedp")}

	svc := NewService(colly, chromedp, nil)

	// Empty mode should default to "smart"
	result, err := svc.Scrape(context.Background(), "https://example.com", "")
	if err != nil {
		t.Fatalf("Default mode scrape failed: %v", err)
	}

	if result == nil {
		t.Fatal("Result should not be nil")
	}
}

func TestService_ScrapeStaticNotConfigured(t *testing.T) {
	svc := NewService(nil, nil, nil)

	_, err := svc.Scrape(context.Background(), "https://example.com", "static")
	if err == nil {
		t.Error("Expected error when static scraper is not configured")
	}
}

func TestService_ScrapeDynamicNotConfigured(t *testing.T) {
	svc := NewService(nil, nil, nil)

	_, err := svc.Scrape(context.Background(), "https://example.com", "dynamic")
	if err == nil {
		t.Error("Expected error when dynamic scraper is not configured")
	}
}

func TestService_ScrapeStaticError(t *testing.T) {
	colly := &mockScraper{err: fmt.Errorf("connection refused")}

	svc := NewService(colly, nil, nil)

	_, err := svc.Scrape(context.Background(), "https://example.com", "static")
	if err == nil {
		t.Error("Expected error from failed static scrape")
	}
}

func TestService_ScrapeDynamicError(t *testing.T) {
	chromedp := &mockScraper{err: fmt.Errorf("browser timeout")}

	svc := NewService(nil, chromedp, nil)

	_, err := svc.Scrape(context.Background(), "https://example.com", "dynamic")
	if err == nil {
		t.Error("Expected error from failed dynamic scrape")
	}
}

func TestNewService(t *testing.T) {
	svc := NewService(nil, nil, nil)
	if svc == nil {
		t.Error("NewService should not return nil")
	}
}
