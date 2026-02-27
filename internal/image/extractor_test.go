package image

import (
	"testing"
)

func TestExtractPageImages_OGImage(t *testing.T) {
	html := `<html><head>
		<meta property="og:image" content="https://example.com/og-hero.jpg">
	</head><body><p>Content</p></body></html>`

	images := ExtractPageImages(html, "https://example.com", 10)

	if len(images) != 1 {
		t.Fatalf("Expected 1 image, got %d", len(images))
	}

	if images[0].URL != "https://example.com/og-hero.jpg" {
		t.Errorf("URL mismatch: got %q", images[0].URL)
	}

	if images[0].SourceType != "og:image" {
		t.Errorf("Source type mismatch: got %q", images[0].SourceType)
	}
}

func TestExtractPageImages_TwitterCard(t *testing.T) {
	html := `<html><head>
		<meta name="twitter:image" content="https://example.com/twitter-card.jpg">
	</head><body></body></html>`

	images := ExtractPageImages(html, "https://example.com", 10)

	if len(images) != 1 {
		t.Fatalf("Expected 1 image, got %d", len(images))
	}

	if images[0].SourceType != "twitter:image" {
		t.Errorf("Source type mismatch: got %q", images[0].SourceType)
	}
}

func TestExtractPageImages_ContentImages(t *testing.T) {
	html := `<html><body>
		<img src="https://example.com/photo1.jpg" alt="Photo 1" title="First Photo">
		<img src="https://example.com/photo2.png" alt="Photo 2">
		<img src="https://example.com/photo3.webp">
	</body></html>`

	images := ExtractPageImages(html, "https://example.com", 10)

	if len(images) != 3 {
		t.Fatalf("Expected 3 images, got %d", len(images))
	}

	if images[0].Alt != "Photo 1" {
		t.Errorf("Alt mismatch: got %q", images[0].Alt)
	}
	if images[0].Title != "First Photo" {
		t.Errorf("Title mismatch: got %q", images[0].Title)
	}
	if images[0].SourceType != "content" {
		t.Errorf("Source type mismatch: got %q", images[0].SourceType)
	}
}

func TestExtractPageImages_RelativeURLs(t *testing.T) {
	html := `<html><body>
		<img src="/images/photo.jpg" alt="Relative">
		<img src="assets/icon.png" alt="Relative path">
	</body></html>`

	images := ExtractPageImages(html, "https://example.com/blog/post", 10)

	if len(images) != 2 {
		t.Fatalf("Expected 2 images, got %d", len(images))
	}

	if images[0].URL != "https://example.com/images/photo.jpg" {
		t.Errorf("Resolved URL mismatch: got %q", images[0].URL)
	}

	if images[1].URL != "https://example.com/blog/assets/icon.png" {
		t.Errorf("Resolved URL mismatch: got %q", images[1].URL)
	}
}

func TestExtractPageImages_DataURIs_Skipped(t *testing.T) {
	html := `<html><body>
		<img src="data:image/png;base64,iVBORw0KGgo=" alt="Data URI">
		<img src="https://example.com/real.jpg" alt="Real image">
	</body></html>`

	images := ExtractPageImages(html, "https://example.com", 10)

	if len(images) != 1 {
		t.Fatalf("Expected 1 image (data URIs skipped), got %d", len(images))
	}

	if images[0].URL != "https://example.com/real.jpg" {
		t.Errorf("URL mismatch: got %q", images[0].URL)
	}
}

func TestExtractPageImages_TrackingPixels_Skipped(t *testing.T) {
	html := `<html><body>
		<img src="https://example.com/tracking-pixel.gif" alt="">
		<img src="https://example.com/analytics/beacon.png" alt="">
		<img src="https://example.com/real-photo.jpg" alt="Real">
	</body></html>`

	images := ExtractPageImages(html, "https://example.com", 10)

	if len(images) != 1 {
		t.Fatalf("Expected 1 image (trackers skipped), got %d", len(images))
	}

	if images[0].URL != "https://example.com/real-photo.jpg" {
		t.Errorf("URL mismatch: got %q", images[0].URL)
	}
}

func TestExtractPageImages_MaxLimit(t *testing.T) {
	html := `<html><body>
		<img src="https://example.com/1.jpg">
		<img src="https://example.com/2.jpg">
		<img src="https://example.com/3.jpg">
		<img src="https://example.com/4.jpg">
		<img src="https://example.com/5.jpg">
	</body></html>`

	images := ExtractPageImages(html, "https://example.com", 3)

	if len(images) != 3 {
		t.Errorf("Expected max 3 images, got %d", len(images))
	}
}

func TestExtractPageImages_Deduplication(t *testing.T) {
	html := `<html><head>
		<meta property="og:image" content="https://example.com/hero.jpg">
	</head><body>
		<img src="https://example.com/hero.jpg" alt="Same as OG">
		<img src="https://example.com/other.jpg" alt="Different">
	</body></html>`

	images := ExtractPageImages(html, "https://example.com", 10)

	if len(images) != 2 {
		t.Errorf("Expected 2 unique images, got %d", len(images))
	}
}

func TestExtractPageImages_EmptyHTML(t *testing.T) {
	images := ExtractPageImages("", "https://example.com", 10)

	if len(images) != 0 {
		t.Errorf("Expected 0 images for empty HTML, got %d", len(images))
	}
}

func TestExtractPageImages_NoImages(t *testing.T) {
	html := `<html><body><p>No images here</p></body></html>`

	images := ExtractPageImages(html, "https://example.com", 10)

	if len(images) != 0 {
		t.Errorf("Expected 0 images, got %d", len(images))
	}
}

func TestExtractPageImages_EmptySrc(t *testing.T) {
	html := `<html><body>
		<img src="" alt="Empty src">
		<img alt="No src at all">
	</body></html>`

	images := ExtractPageImages(html, "https://example.com", 10)

	if len(images) != 0 {
		t.Errorf("Expected 0 images for empty/missing src, got %d", len(images))
	}
}

func TestExtractPageImages_PriorityOrder(t *testing.T) {
	html := `<html><head>
		<meta property="og:image" content="https://example.com/og.jpg">
		<meta name="twitter:image" content="https://example.com/twitter.jpg">
	</head><body>
		<img src="https://example.com/content.jpg" alt="Content">
	</body></html>`

	images := ExtractPageImages(html, "https://example.com", 10)

	if len(images) != 3 {
		t.Fatalf("Expected 3 images, got %d", len(images))
	}

	// OG should be first
	if images[0].SourceType != "og:image" {
		t.Errorf("First image should be og:image, got %q", images[0].SourceType)
	}

	// Twitter second
	if images[1].SourceType != "twitter:image" {
		t.Errorf("Second image should be twitter:image, got %q", images[1].SourceType)
	}

	// Content last
	if images[2].SourceType != "content" {
		t.Errorf("Third image should be content, got %q", images[2].SourceType)
	}
}

func TestResolveURL(t *testing.T) {
	tests := []struct {
		name     string
		rawURL   string
		pageURL  string
		expected string
	}{
		{
			name:     "Absolute URL unchanged",
			rawURL:   "https://cdn.example.com/img.jpg",
			pageURL:  "https://example.com",
			expected: "https://cdn.example.com/img.jpg",
		},
		{
			name:     "Root-relative URL",
			rawURL:   "/images/photo.jpg",
			pageURL:  "https://example.com/blog/post",
			expected: "https://example.com/images/photo.jpg",
		},
		{
			name:     "Relative path",
			rawURL:   "photo.jpg",
			pageURL:  "https://example.com/blog/",
			expected: "https://example.com/blog/photo.jpg",
		},
		{
			name:     "Data URI returns empty",
			rawURL:   "data:image/png;base64,abc",
			pageURL:  "https://example.com",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := resolveURL(tt.rawURL, tt.pageURL)
			if result != tt.expected {
				t.Errorf("resolveURL(%q, %q) = %q, want %q", tt.rawURL, tt.pageURL, result, tt.expected)
			}
		})
	}
}

func TestIsTrackingPixel(t *testing.T) {
	tests := []struct {
		url      string
		expected bool
	}{
		{"https://example.com/tracking-pixel.gif", true},
		{"https://example.com/analytics/beacon.png", true},
		{"https://example.com/images/1x1.gif", true},
		{"https://example.com/spacer.gif", true},
		{"https://example.com/photo.jpg", false},
		{"https://cdn.example.com/hero-banner.webp", false},
	}

	for _, tt := range tests {
		t.Run(tt.url, func(t *testing.T) {
			result := isTrackingPixel(tt.url)
			if result != tt.expected {
				t.Errorf("isTrackingPixel(%q) = %v, want %v", tt.url, result, tt.expected)
			}
		})
	}
}
