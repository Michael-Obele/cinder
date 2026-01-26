# Image and Screenshot Feature Implementation Plan

**Document Location:** `docs/features/image-screenshot-feature.md`

---

## Overview

This document outlines a comprehensive plan to add image and screenshot capture capabilities to the Cinder search service. The feature will allow users to retrieve images from search results and capture website screenshots, enhancing the search experience with visual content.

### Current State

The Cinder search service (`internal/search/service.go`) currently returns text-based search results from the Brave Search API with pagination support and rate limiting.

### Goal

Extend the search service to optionally include images and website screenshots alongside text results, maintaining backward compatibility.

---

## 1. Architecture Overview

### New Service Layer Structure

```
cinder/
├── internal/
│   ├── search/
│   │   ├── service.go          (EXISTING - will extend Result struct)
│   │   └── service_test.go
│   ├── image/                  (NEW)
│   │   ├── service.go          (NEW - image fetching interface)
│   │   ├── screenshot.go       (NEW - screenshot capture)
│   │   ├── processor.go        (NEW - image processing)
│   │   └── service_test.go     (NEW)
│   └── cache/                  (NEW)
│       └── image_cache.go      (NEW - optional caching)
└── docs/
    └── features/
        └── image-screenshot-feature.md (THIS FILE)
```

### Why Separate Services?

Following the **Single Responsibility Principle**, the image service handles all visual content operations independently from text search. This prevents the search service from becoming bloated and allows for:

- Independent testing
- Separate error handling
- Optional feature enablement
- Future extensibility (e.g., image recognition, OCR)

---

## 2. Data Structures to Create

### 2.1 Extending the Result Struct

**File:** `internal/search/service.go`

**Current:**

```go
type Result struct {
    Title       string  `json:"title"`
    URL         string  `json:"url"`
    Description string  `json:"description"`
    ID          string  `json:"id"`
    Domain      string  `json:"domain"`
    Relevance   float64 `json:"relevance"`
}
```

**Modified To Include:**

```go
type Result struct {
    Title       string         `json:"title"`
    URL         string         `json:"url"`
    Description string         `json:"description"`
    ID          string         `json:"id"`
    Domain      string         `json:"domain"`
    Relevance   float64        `json:"relevance"`
    // NEW FIELDS
    Images      []ImageData    `json:"images,omitempty"`
    Screenshot  *ScreenshotData `json:"screenshot,omitempty"`
    FaviconURL  string         `json:"favicon_url,omitempty"`
}
```

### 2.2 New Image Data Structures

**File:** `internal/image/service.go`

```go
// ImageData represents a single image from a search result
type ImageData struct {
    URL        string `json:"url"`
    Title      string `json:"title"`
    Width      int    `json:"width"`
    Height     int    `json:"height"`
    Format     string `json:"format"`        // "png", "jpeg", "webp"
    Size       int64  `json:"size"`          // bytes
    SourceType string `json:"source_type"`   // "og:image", "favicon", "search-result"
    Alt        string `json:"alt,omitempty"` // alt text if available
}

// ScreenshotData represents a captured website screenshot
type ScreenshotData struct {
    Data       []byte    `json:"data"`        // base64 encoded in JSON
    Format     string    `json:"format"`      // "png", "jpeg"
    Width      int       `json:"width"`
    Height     int       `json:"height"`
    CapturedAt time.Time `json:"captured_at"`
    Error      string    `json:"error,omitempty"` // if capture failed
}

// ImageMetadata contains technical details about an image
type ImageMetadata struct {
    Dimensions struct {
        Width  int
        Height int
    }
    Format       string
    Size         int64
    ColorSpace   string
    HasAlpha     bool
    IsAnimated   bool
}

// ScreenshotOptions configures screenshot capture behavior
type ScreenshotOptions struct {
    Width          int           // viewport width in pixels
    Height         int           // viewport height in pixels
    FullPage       bool          // capture entire page vs viewport
    WaitSelector   string        // CSS selector to wait for before capturing
    Timeout        time.Duration // max time to wait for page load
    Format         string        // "png" or "jpeg"
    Quality        int           // 1-100 for jpeg quality
    BlockAds       bool          // block ad scripts
    BlockTrackers  bool          // block tracking scripts
}
```

### 2.3 Updated SearchOptions

**File:** `internal/search/service.go`

```go
type SearchOptions struct {
    Query          string
    Offset         int
    Limit          int
    IncludeDomains []string
    ExcludeDomains []string
    RequiredText   []string
    MaxAge         *int
    Mode           string
    // NEW FIELDS
    IncludeImages    bool             // fetch images for results
    IncludeScreenshot bool            // capture website screenshots
    ImageLimit       int              // max images per result (default 3)
    ScreenshotOpts   *ScreenshotOptions // screenshot configuration
}
```

---

## 3. Service Interfaces

### 3.1 Image Service Interface

**File:** `internal/image/service.go`

```go
package image

import (
    "context"
)

// Service defines the image operations interface
type Service interface {
    // FetchImage downloads and returns image bytes from a URL
    FetchImage(ctx context.Context, url string) ([]byte, *ImageMetadata, error)

    // CaptureScreenshot captures a website screenshot using headless browser
    CaptureScreenshot(ctx context.Context, url string, opts ScreenshotOptions) ([]byte, *ImageMetadata, error)

    // GetImageMetadata extracts metadata without downloading full image
    GetImageMetadata(ctx context.Context, url string) (*ImageMetadata, error)

    // ExtractPageImages extracts all images from a webpage
    ExtractPageImages(ctx context.Context, url string, limit int) ([]ImageData, error)
}

// ImageProcessor handles image manipulation
type ImageProcessor interface {
    // Resize scales an image to specified dimensions
    Resize(data []byte, width, height int) ([]byte, error)

    // Compress reduces image file size
    Compress(data []byte, quality int) ([]byte, error)

    // Convert changes image format
    Convert(data []byte, fromFormat, toFormat string) ([]byte, error)
}
```

---

## 4. Implementation Strategy

### Phase 1: Foundation (No Breaking Changes)

#### Step 1: Create Image Service Stub

- Create `internal/image/service.go` with interface definitions
- Implement a basic `HTTPImageService` using existing `http.Client`
- Add simple image validation (MIME type checking)
- **Impact:** Zero impact on existing search functionality

#### Step 2: Extend Result Struct

- Add optional image fields to `Result` struct with `omitempty` tags
- Update JSON marshaling tests
- **Impact:** Backward compatible—old clients ignore new fields

#### Step 3: Update SearchOptions

- Add image-related options with defaults
- Default to `IncludeImages: false` and `IncludeScreenshot: false`
- **Impact:** Existing code works unchanged

#### Step 4: Add Image Processor

- Integrate `github.com/disintegration/imaging` for basic operations
- Support resize and format conversion
- **Impact:** Optional feature, doesn't affect core search

### Phase 2: Screenshot Capture

#### Step 5: Implement Headless Browser Integration

- Add `github.com/chromedp/chromedp` dependency
- Create `ChromeScreenshotService` with connection pooling
- Implement timeout and error handling
- **Impact:** Optional feature flag controls activation

#### Step 6: Browser Pool Management

- Create `BrowserPool` to manage Chrome connections
- Implement graceful shutdown
- Add concurrent request limits (default 3-5)
- **Impact:** Isolated to screenshot service

### Phase 3: Integration

#### Step 7: Wire Services Together

- Create `ImageServiceFactory` for dependency injection
- Add configuration options for enabling/disabling features
- Integrate with existing search workflow
- **Impact:** Configurable—can be disabled via settings

#### Step 8: Add Caching Layer (Optional)

- Implement in-memory cache with TTL
- Add cache invalidation strategies
- **Impact:** Performance optimization, doesn't affect functionality

---

## 5. Key Dependencies

### Required Go Packages

```bash
# Image processing (pure Go, no C dependencies)
go get github.com/disintegration/imaging@latest

# Headless browser automation
go get github.com/chromedp/chromedp@latest

# Context management (already imported)
# golang.org/x/time/rate (already imported)
```

### Why These Choices?

| Package                    | Reason                                                 | Alternative                          |
| -------------------------- | ------------------------------------------------------ | ------------------------------------ |
| **disintegration/imaging** | Pure Go, no native deps, simple API                    | bimg (needs libvips), graphicsmagick |
| **chromedp/chromedp**      | Official Chrome DevTools Protocol, actively maintained | playwright-go, puppeteer-go          |

---

## 6. Error Handling & Safety

### Preventing Breakage

```go
// 1. Optional Features - All image operations are opt-in
if opts.IncludeImages {
    // only execute if explicitly requested
}

// 2. Graceful Degradation - Missing images don't fail search
results, err := s.Search(ctx, opts) // succeeds even if image fetch fails
// individual images may be nil, but search results are complete

// 3. Timeout Protection
ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
defer cancel()
// prevents hanging requests

// 4. Rate Limiting - Apply separate limits for images
imageLimiter := rate.NewLimiter(rate.Every(500*time.Millisecond), 1)

// 5. Validation - Check URLs before processing
if !isValidImageURL(url) {
    return nil, fmt.Errorf("invalid image URL")
}
```

### Testing Strategy

**File:** `internal/image/service_test.go`

```go
func TestImageService_FetchImage_InvalidURL(t *testing.T) {
    // Test with malformed URLs
}

func TestImageService_Screenshot_Timeout(t *testing.T) {
    // Test timeout handling
}

func TestSearchService_BackwardCompatibility(t *testing.T) {
    // Ensure old SearchOptions still work
}

func TestResultJSON_OmitEmptyImages(t *testing.T) {
    // Verify images field omitted when empty
}
```

---

## 7. Configuration

### Environment Variables

```bash
# Enable image features
CINDER_ENABLE_IMAGES=true
CINDER_ENABLE_SCREENSHOTS=true

# Chrome settings
CINDER_CHROME_EXECUTABLE=/usr/bin/chromium
CINDER_CHROME_POOL_SIZE=3
CINDER_CHROME_TIMEOUT=30s

# Image settings
CINDER_IMAGE_CACHE_SIZE=100
CINDER_IMAGE_CACHE_TTL=1h
CINDER_MAX_IMAGE_SIZE=5MB
```

### Configuration Struct

**File:** `internal/config/config.go`

```go
type ImageConfig struct {
    EnableImages       bool
    EnableScreenshots  bool
    MaxImageSize       int64
    CacheSize          int
    CacheTTL           time.Duration
    ScreenshotTimeout  time.Duration
    ChromePoolSize     int
    ChromeExecutable   string
}
```

---

## 8. Migration Path

### For JavaScript Developers

Since you're coming from a JS background, think of this like this:

```javascript
// BEFORE (like current Cinder)
const results = await search("golang tutorials");
// Returns: [ { title, url, description, ... } ]

// AFTER (with image features)
const results = await search("golang tutorials", {
  includeImages: true,
  includeScreenshot: true,
});
// Returns: [ {
//   title, url, description,
//   images: [ { url, width, height, ... } ],
//   screenshot: { data, width, height, ... }
// } ]

// Backward compatible - old code works:
const results = await search("golang tutorials");
// Still works, images not fetched
```

### Go Equivalent

```go
// BEFORE
results, _, err := s.Search(ctx, SearchOptions{Query: "golang"})

// AFTER - with images (opt-in)
results, _, err := s.Search(ctx, SearchOptions{
    Query:           "golang",
    IncludeImages:   true,
    IncludeScreenshot: true,
})

// BACKWARD COMPATIBLE - old code unaffected
results, _, err := s.Search(ctx, SearchOptions{Query: "golang"})
// IncludeImages defaults to false
```

---

## 9. Implementation Checklist

### Phase 1: Foundation

- [ ] Create `internal/image/service.go` with interfaces
- [ ] Create `internal/image/processor.go` with imaging integration
- [ ] Extend `Result` struct in `internal/search/service.go`
- [ ] Update `SearchOptions` struct
- [ ] Write tests for backward compatibility
- [ ] Update README with new optional fields

### Phase 2: Screenshots

- [ ] Create `internal/image/screenshot.go`
- [ ] Implement `ChromeScreenshotService`
- [ ] Create browser pool manager
- [ ] Add connection pool tests
- [ ] Document Chrome installation requirements

### Phase 3: Integration

- [ ] Create `ImageServiceFactory` for dependency injection
- [ ] Wire image service into search workflow
- [ ] Add configuration loading
- [ ] Create integration tests
- [ ] Add examples in `examples/` directory

### Phase 4: Optimization (Optional)

- [ ] Implement image caching layer
- [ ] Add cache invalidation strategy
- [ ] Performance benchmarks
- [ ] Load testing with concurrent requests

---

## 10. Files Modified/Created Summary

### Modified Files

| File                         | Changes                                                 |
| ---------------------------- | ------------------------------------------------------- |
| `internal/search/service.go` | Add image fields to Result struct, extend SearchOptions |
| `go.mod`                     | Add chromedp, imaging dependencies                      |

### New Files

| File                                        | Purpose                                          |
| ------------------------------------------- | ------------------------------------------------ |
| `internal/image/service.go`                 | Image fetching interface and HTTP implementation |
| `internal/image/screenshot.go`              | Screenshot capture using chromedp                |
| `internal/image/processor.go`               | Image processing/manipulation                    |
| `internal/image/service_test.go`            | Unit tests for image service                     |
| `internal/config/image_config.go`           | Configuration for image features                 |
| `docs/features/image-screenshot-feature.md` | This documentation                               |

---

## 11. Security Considerations

### URL Validation

```go
// Always validate URLs before fetching
func isValidImageURL(rawURL string) bool {
    u, err := url.Parse(rawURL)
    if err != nil {
        return false
    }
    // Only http/https
    return u.Scheme == "http" || u.Scheme == "https"
}
```

### Size Limits

```go
// Prevent downloading huge files
const MaxImageSize = 5 * 1024 * 1024 // 5MB
if resp.ContentLength > MaxImageSize {
    return fmt.Errorf("image too large")
}
```

### Timeout Protection

```go
// Always use context with timeout
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()
```

---

## 12. References

### Go Documentation

- [Context Package](https://pkg.go.dev/context) - Request-scoped values and cancellation
- [Go Image Package](https://go.dev/blog/image) - Image manipulation basics
- [HTTP Client](https://pkg.go.dev/net/http) - Making HTTP requests

### External Libraries

- [disintegration/imaging](https://pkg.go.dev/github.com/disintegration/imaging) - Image processing
- [chromedp/chromedp](https://pkg.go.dev/github.com/chromedp/chromedp) - Browser automation
- [Rate Limiting in Go](https://pkg.go.dev/golang.org/x/time/rate) - Request throttling

---

## 13. Next Steps

1. **Review this plan** with the team
2. **Create feature branch**: `feature/image-screenshot-service`
3. **Start Phase 1** with foundation work
4. **Write tests first** (TDD approach)
5. **Gradually integrate** each component
6. **Performance test** before Phase 3 completion

---

**Document Version:** 1.0  
**Last Updated:** January 26, 2026  
**Status:** Planning Phase (Ready for Implementation)
