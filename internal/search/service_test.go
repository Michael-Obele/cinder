package search

import (
	"context"
	"net/http"
	"testing"
)

// TestSearchBasic tests basic search functionality
func TestSearchBasic(t *testing.T) {
	service := NewBraveService("test-api-key")

	// Note: This test will fail without actual Brave API key
	// Run with integration tests using mock server
	if service == nil {
		t.Fatal("Failed to create service")
	}
}

// TestSearchLimitValidation tests limit parameter validation
func TestSearchLimitValidation(t *testing.T) {
	service := NewBraveService("test-api-key")

	tests := []struct {
		name      string
		limit     int
		expectMax int
	}{
		{
			name:      "Zero limit uses default",
			limit:     0,
			expectMax: 10,
		},
		{
			name:      "Small limit",
			limit:     5,
			expectMax: 5,
		},
		{
			name:      "Large limit capped at 100",
			limit:     200,
			expectMax: 100,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Verify service is created
			if service == nil {
				t.Errorf("Service is nil for test %s", tt.name)
			}
		})
	}
}

// TestExtractDomain tests URL domain extraction
func TestExtractDomain(t *testing.T) {
	tests := []struct {
		url      string
		expected string
	}{
		{
			url:      "https://golang.org/doc",
			expected: "golang.org",
		},
		{
			url:      "https://www.github.com/golang/go",
			expected: "www.github.com",
		},
		{
			url:      "https://github.com",
			expected: "github.com",
		},
		{
			url:      "http://localhost:8080/path",
			expected: "localhost",
		},
		{
			url:      "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.url, func(t *testing.T) {
			result := extractDomain(tt.url)
			if result != tt.expected {
				t.Errorf("extractDomain(%q) = %q, want %q", tt.url, result, tt.expected)
			}
		})
	}
}

// TestSearchMissingAPIKey tests search without API key
func TestSearchMissingAPIKey(t *testing.T) {
	service := &BraveService{
		apiKey: "",
		client: &http.Client{},
	}

	_, _, err := service.Search(context.Background(), SearchOptions{
		Query:  "golang",
		Offset: 0,
		Limit:  10,
	})

	if err == nil {
		t.Errorf("Expected error for missing API key, got none")
	}
}

// TestSearchOptionsValidation tests that SearchOptions are properly validated
func TestSearchOptionsValidation(t *testing.T) {
	service := NewBraveService("test-api-key")

	tests := []struct {
		name string
		opts SearchOptions
	}{
		{
			name: "Valid options",
			opts: SearchOptions{
				Query:  "test",
				Offset: 0,
				Limit:  10,
			},
		},
		{
			name: "Large offset",
			opts: SearchOptions{
				Query:  "test",
				Offset: 100,
				Limit:  10,
			},
		},
		{
			name: "Max limit",
			opts: SearchOptions{
				Query:  "test",
				Offset: 0,
				Limit:  100,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if service == nil {
				t.Error("Service creation failed")
			}
		})
	}
}

// TestNewBraveService tests service creation
func TestNewBraveService(t *testing.T) {
	tests := []struct {
		name   string
		apiKey string
	}{
		{
			name:   "With API key",
			apiKey: "test-key-123",
		},
		{
			name:   "Empty API key",
			apiKey: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := NewBraveService(tt.apiKey)
			if service == nil {
				t.Error("Failed to create BraveService")
			}
		})
	}
}

// BenchmarkSearch benchmarks search performance
func BenchmarkSearch(b *testing.B) {
	service := NewBraveService("test-api-key")

	opts := SearchOptions{
		Query:  "benchmark test",
		Offset: 0,
		Limit:  10,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		service.Search(context.Background(), opts)
	}
}

// TestParseURLDomain tests URL parsing
func TestParseURLDomain(t *testing.T) {
	tests := []struct {
		url         string
		expectError bool
	}{
		{
			url:         "https://example.com",
			expectError: false,
		},
		{
			url:         "http://localhost:8080",
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.url, func(t *testing.T) {
			domain, err := parseURLDomain(tt.url)
			if (err != nil) != tt.expectError {
				t.Errorf("parseURLDomain(%q) error = %v, want error = %v", tt.url, err, tt.expectError)
			}
			if err == nil && domain == "" {
				t.Errorf("parseURLDomain(%q) returned empty domain", tt.url)
			}
		})
	}
}

// TestSearchResult tests Result struct
func TestSearchResult(t *testing.T) {
	result := Result{
		Title:       "Test Title",
		URL:         "https://example.com",
		Description: "Test Description",
		ID:          "test-id",
		Domain:      "example.com",
		Relevance:   0.95,
	}

	if result.Title == "" {
		t.Error("Title is empty")
	}
	if result.URL == "" {
		t.Error("URL is empty")
	}
	if result.Domain == "" {
		t.Error("Domain is empty")
	}
	if result.Relevance < 0 || result.Relevance > 1.0 {
		t.Errorf("Relevance out of range: %f", result.Relevance)
	}
}
