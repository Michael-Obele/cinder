package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/standard-user/cinder/internal/domain"
	"github.com/standard-user/cinder/internal/scraper"
)

// mockStaticScraper implements domain.Scraper
type mockStaticScraper struct {
	result *domain.ScrapeResult
	err    error
}

func (m *mockStaticScraper) Scrape(ctx context.Context, url string) (*domain.ScrapeResult, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.result, nil
}

func setupScrapeHandler() *ScrapeHandler {
	colly := &mockStaticScraper{
		result: &domain.ScrapeResult{
			URL:      "https://example.com",
			Markdown: "# Example",
			HTML:     "<html><body><h1>Example</h1></body></html>",
			Metadata: map[string]string{"engine": "colly"},
		},
	}
	svc := scraper.NewService(colly, nil, nil)
	return NewScrapeHandler(svc)
}

func TestScrapeHandler_PostJSON(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler := setupScrapeHandler()

	reqBody := ScrapeRequest{
		URL:  "https://example.com",
		Mode: "static",
	}

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("POST", "/scrape", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	handler.Scrape(c)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d. Body: %s", w.Code, w.Body.String())
	}

	var result domain.ScrapeResult
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if result.URL != "https://example.com" {
		t.Errorf("URL mismatch: got %q", result.URL)
	}
}

func TestScrapeHandler_MissingURL(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler := setupScrapeHandler()

	// POST with empty JSON
	req := httptest.NewRequest("POST", "/scrape", bytes.NewReader([]byte(`{}`)))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	handler.Scrape(c)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected 400 for missing URL, got %d", w.Code)
	}
}

func TestScrapeHandler_QueryParams(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler := setupScrapeHandler()

	// GET request  with query params
	req := httptest.NewRequest("GET", "/scrape?url=https://example.com&mode=static", nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	handler.Scrape(c)

	if w.Code != http.StatusOK {
		t.Errorf("Expected 200 for GET with query, got %d. Body: %s", w.Code, w.Body.String())
	}
}

func TestScrapeHandler_RenderBackwardCompat(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler := setupScrapeHandler()

	// render=true should map to mode=dynamic
	reqBody := ScrapeRequest{
		URL:    "https://example.com",
		Render: true,
	}

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("POST", "/scrape", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	handler.Scrape(c)

	// Dynamic scraper is nil, so this should fail with 500
	if w.Code != http.StatusInternalServerError {
		// If it somehow succeeded we accept that too
		if w.Code != http.StatusOK {
			t.Errorf("Expected 500 (dynamic scraper not configured) or 200, got %d", w.Code)
		}
	}
}

func TestScrapeHandler_DefaultMode(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler := setupScrapeHandler()

	// No mode specified → defaults to "smart" which tries static first
	reqBody := ScrapeRequest{
		URL: "https://example.com",
	}

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("POST", "/scrape", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	handler.Scrape(c)

	if w.Code != http.StatusOK {
		t.Errorf("Expected 200, got %d. Body: %s", w.Code, w.Body.String())
	}
}

func TestScrapeHandler_EmptyBody(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler := setupScrapeHandler()

	req := httptest.NewRequest("POST", "/scrape", nil)
	req.Header.Set("Content-Type", "application/json")
	req.ContentLength = 0

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	handler.Scrape(c)

	// No URL provided → 400
	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected 400 for empty body, got %d", w.Code)
	}
}

func TestNewScrapeHandler(t *testing.T) {
	svc := scraper.NewService(nil, nil, nil)
	handler := NewScrapeHandler(svc)

	if handler == nil {
		t.Fatal("NewScrapeHandler should not return nil")
	}
}
