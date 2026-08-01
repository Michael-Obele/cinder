package scraper

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/standard-user/cinder/internal/domain"
)

// --- Mock Scrapers ---

type mockScraper struct {
	result *domain.ScrapeResult
	err    error
}

func (m *mockScraper) Scrape(ctx context.Context, url string, opts domain.ScrapeOptions) (*domain.ScrapeResult, error) {
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

	result, err := svc.Scrape(context.Background(), "https://example.com", "static", domain.ScrapeOptions{})
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

	result, err := svc.Scrape(context.Background(), "https://example.com", "dynamic", domain.ScrapeOptions{})
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

	result, err := svc.Scrape(context.Background(), "https://example.com", "smart", domain.ScrapeOptions{})
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

	result, err := svc.Scrape(context.Background(), "https://spa.example.com", "smart", domain.ScrapeOptions{})
	if err != nil {
		t.Fatalf("Smart scrape failed: %v", err)
	}

	if result.Metadata["engine"] != "chromedp" {
		t.Errorf("Smart mode should fall back to chromedp for SPA shells, got %q", result.Metadata["engine"])
	}
}

func TestService_ScrapeUnknownMode(t *testing.T) {
	svc := NewService(nil, nil, nil)

	_, err := svc.Scrape(context.Background(), "https://example.com", "invalid", domain.ScrapeOptions{})
	if err == nil {
		t.Error("Expected error for unknown mode")
	}
}

func TestService_ScrapeDefaultMode(t *testing.T) {
	colly := &mockScraper{result: newMockResult("colly")}
	chromedp := &mockScraper{result: newMockResult("chromedp")}

	svc := NewService(colly, chromedp, nil)

	// Empty mode should default to "smart"
	result, err := svc.Scrape(context.Background(), "https://example.com", "", domain.ScrapeOptions{})
	if err != nil {
		t.Fatalf("Default mode scrape failed: %v", err)
	}

	if result == nil {
		t.Fatal("Result should not be nil")
	}
}

func TestService_ScrapeStaticNotConfigured(t *testing.T) {
	svc := NewService(nil, nil, nil)

	_, err := svc.Scrape(context.Background(), "https://example.com", "static", domain.ScrapeOptions{})
	if err == nil {
		t.Error("Expected error when static scraper is not configured")
	}
}

func TestService_ScrapeDynamicNotConfigured(t *testing.T) {
	svc := NewService(nil, nil, nil)

	_, err := svc.Scrape(context.Background(), "https://example.com", "dynamic", domain.ScrapeOptions{})
	if err == nil {
		t.Error("Expected error when dynamic scraper is not configured")
	}
}

func TestService_ScrapeStaticError(t *testing.T) {
	colly := &mockScraper{err: fmt.Errorf("connection refused")}

	svc := NewService(colly, nil, nil)

	_, err := svc.Scrape(context.Background(), "https://example.com", "static", domain.ScrapeOptions{})
	if err == nil {
		t.Error("Expected error from failed static scrape")
	}
}

func TestService_ScrapeDynamicError(t *testing.T) {
	chromedp := &mockScraper{err: fmt.Errorf("browser timeout")}

	svc := NewService(nil, chromedp, nil)

	_, err := svc.Scrape(context.Background(), "https://example.com", "dynamic", domain.ScrapeOptions{})
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

func TestCacheKeyFor_DiffersOnOptions(t *testing.T) {
	baseKey := cacheKeyFor("https://example.com", "smart", domain.ScrapeOptions{Screenshot: false, Images: true, ImageFormat: domain.ImageFormatURL})

	tests := []struct {
		name string
		mode string
		opts domain.ScrapeOptions
	}{
		{"different max images", "smart", domain.ScrapeOptions{Screenshot: false, Images: true, ImageFormat: domain.ImageFormatURL, MaxImages: 5}},
		{"different image format", "smart", domain.ScrapeOptions{Screenshot: false, Images: true, ImageFormat: domain.ImageFormatBlob}},
		{"different screenshot", "smart", domain.ScrapeOptions{Screenshot: true, Images: true, ImageFormat: domain.ImageFormatURL}},
		{"different mode", "dynamic", domain.ScrapeOptions{Screenshot: false, Images: true, ImageFormat: domain.ImageFormatURL}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if key := cacheKeyFor("https://example.com", tt.mode, tt.opts); key == baseKey {
				t.Errorf("cache key should differ for options change: %q == %q", key, baseKey)
			}
		})
	}
}

func TestCacheKeyFor_StableForSameOptions(t *testing.T) {
	opts := domain.ScrapeOptions{Images: true, MaxImages: 10, ImageFormat: domain.ImageFormatBlob}
	k1 := cacheKeyFor("https://example.com", "static", opts)
	k2 := cacheKeyFor("https://example.com", "static", opts)
	if k1 != k2 {
		t.Errorf("cache key should be deterministic: %q != %q", k1, k2)
	}
	if len(k1) != len("scrape:")+64 {
		t.Errorf("expected sha256-length key, got %q", k1)
	}
}

// TestService_BlobFetchParallel verifies blob image fetching runs
// concurrently and preserves result order.
func TestService_BlobFetchParallel(t *testing.T) {
	// 4 images, each with a 200ms server-side delay.
	var serverDelay = 200 * time.Millisecond
	urls := []string{"/a.png", "/b.png", "/c.png", "/d.png"}
	var mu sync.Mutex
	var inflight int
	var maxInflight int

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		inflight++
		if inflight > maxInflight {
			maxInflight = inflight
		}
		mu.Unlock()

		time.Sleep(serverDelay)

		mu.Lock()
		inflight--
		mu.Unlock()

		w.Header().Set("Content-Type", "image/png")
		// Minimal 1x1 PNG.
		_, _ = w.Write([]byte{0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a, 0x00, 0x00, 0x00, 0x0d, 0x49, 0x48, 0x44, 0x52, 0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01, 0x08, 0x06, 0x00, 0x00, 0x00, 0x1f, 0x15, 0xc4, 0x89, 0x00, 0x00, 0x00, 0x0a, 0x49, 0x44, 0x41, 0x54, 0x78, 0x9c, 0x63, 0x00, 0x01, 0x00, 0x00, 0x05, 0x00, 0x01, 0x0d, 0x0a, 0x2d, 0xb4, 0x00, 0x00, 0x00, 0x00, 0x49, 0x45, 0x4e, 0x44, 0xae, 0x42, 0x60, 0x82})
	}))
	defer srv.Close()

	// Static mock returns HTML referencing all 4 images.
	html := "<html><body>"
	for _, u := range urls {
		html += `<img src="` + srv.URL + u + `">`
	}
	html += "</body></html>"

	colly := &mockScraper{result: &domain.ScrapeResult{
		URL:      srv.URL + "/page",
		Markdown: "# Page",
		HTML:     html,
		Metadata: map[string]string{"engine": "colly"},
	}}
	svc := NewService(colly, nil, nil)

	start := time.Now()
	result, err := svc.Scrape(context.Background(), srv.URL+"/page", "static", domain.ScrapeOptions{
		Images:      true,
		ImageFormat: domain.ImageFormatBlob,
		MaxImages:   10,
	})
	elapsed := time.Since(start)
	if err != nil {
		t.Fatalf("scrape failed: %v", err)
	}

	if len(result.Images) != 4 {
		t.Fatalf("expected 4 images, got %d", len(result.Images))
	}
	// Order preserved.
	for i, u := range urls {
		if result.Images[i].URL != srv.URL+u {
			t.Errorf("image %d out of order: got %q want %q", i, result.Images[i].URL, srv.URL+u)
		}
		if result.Images[i].Blob == "" {
			t.Errorf("image %d missing blob", i)
		}
	}
	// 4 × 200ms serial would be ~800ms; concurrent (limit 5) is ~200ms.
	if elapsed > 600*time.Millisecond {
		t.Errorf("blob fetch not parallel: took %v", elapsed)
	}
}
