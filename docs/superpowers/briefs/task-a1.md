# Task A1: Add go-readability dependency + main-content extraction

**Files:**
- Modify: `go.mod` (via `go get`)
- Create: `internal/scraper/content.go`
- Test: `internal/scraper/content_test.go`

**Interfaces:**
- Produces: `func ExtractMainContent(rawHTML string, pageURL string) (*ReadabilityResult, error)` where `ReadabilityResult{ Title, Excerpt, Byline, SiteName, ContentHTML string }`. Consumed by colly.go, chromedp.go in Task A2.

## Steps

1. Write the failing test:

```go
package scraper

import "testing"

func TestExtractMainContent_StripsBoilerplate(t *testing.T) {
	html := `<html><head><title>T</title></head><body><nav>nav junk</nav><article><h1>Hello</h1><p>Real content here with enough text to pass readability thresholds for extraction.</p></article><footer>footer junk</footer></body></html>`
	res, err := ExtractMainContent(html, "https://example.com")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res == nil || res.ContentHTML == "" {
		t.Fatal("expected content")
	}
}

func TestExtractMainContent_FallsBackOnUnparseable(t *testing.T) {
	res, err := ExtractMainContent("<p>x</p>", "https://example.com")
	// must not error hard — fallback returns original
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res == nil {
		t.Fatal("expected fallback result, got nil")
	}
}
```

2. Run `go test ./internal/scraper/ -run TestExtractMainContent -v` → must FAIL (undefined function).

3. Implement `internal/scraper/content.go`:

```go
package scraper

import (
	"net/url"
	"strings"

	"github.com/go-shiori/go-readability"
)

// ReadabilityResult holds main-content extraction output.
type ReadabilityResult struct {
	Title       string
	Excerpt     string
	Byline      string
	SiteName    string
	ContentHTML string
}

// ExtractMainContent runs readability on rawHTML. On failure it returns a
// fallback result containing the original HTML and a nil error — callers
// must never fail a scrape because extraction failed.
func ExtractMainContent(rawHTML string, pageURL string) (*ReadabilityResult, error) {
	parsed, err := readability.FromReader(strings.NewReader(rawHTML), mustParseURL(pageURL))
	if err != nil {
		return &ReadabilityResult{ContentHTML: rawHTML}, nil
	}
	return &ReadabilityResult{
		Title:       parsed.Title,
		Excerpt:     parsed.Excerpt,
		Byline:      parsed.Byline,
		SiteName:    parsed.SiteName,
		ContentHTML: parsed.Content,
	}, nil
}

// mustParseURL parses pageURL, returning a zero URL on error so readability
// still functions for relative-link resolution.
func mustParseURL(pageURL string) *url.URL {
	u, err := url.Parse(pageURL)
	if err != nil {
		return &url.URL{}
	}
	return u
}
```

Then run `go get github.com/go-shiori/go-readability@v0.44.0` and `go mod tidy`.

4. Test passes.
5. `git add -A && git commit -m "feat(scraper): add readability main-content extraction"`

## Notes

- Verify the actual readability v0.44.0 API: `readability.FromReader(io.Reader, *url.URL) (readability.Article, error)` — if the signature differs in this version, adapt the implementation accordingly (but keep our exported signature exactly as specified).
- All exported symbols must have doc comments.
- Run `make check` (or `gofmt -l .`, `go vet ./...`, `go test ./internal/scraper/ -race`) before committing.
