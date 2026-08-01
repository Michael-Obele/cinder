package scraper

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/standard-user/cinder/internal/domain"
	"github.com/standard-user/cinder/internal/image"
	"github.com/standard-user/cinder/pkg/logger"
	"golang.org/x/sync/errgroup"
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

// cacheKeyFor builds a deterministic cache key from the URL, mode, and the
// full scrape options so option changes never serve stale cached results.
func cacheKeyFor(url, mode string, opts domain.ScrapeOptions) string {
	payload := struct {
		URL  string
		Mode string
		Opts domain.ScrapeOptions
	}{URL: url, Mode: mode, Opts: opts}
	data, err := json.Marshal(payload)
	if err != nil {
		// Marshal of these types cannot fail; fall back to the legacy key.
		return fmt.Sprintf("scrape:%s:%s", url, mode)
	}
	sum := sha256.Sum256(data)
	return "scrape:" + hex.EncodeToString(sum[:])
}

func (s *Service) Scrape(ctx context.Context, url string, mode string, opts domain.ScrapeOptions) (*domain.ScrapeResult, error) {
	// Default to smart if empty
	if mode == "" {
		mode = "smart"
	}

	// Page actions require a real browser: force dynamic mode.
	if len(opts.Actions) > 0 {
		switch mode {
		case "dynamic":
		case "static":
			return nil, fmt.Errorf("actions require dynamic mode (got static)")
		default:
			mode = "dynamic"
		}
	}

	// 1. Try Cache
	cacheKey := cacheKeyFor(url, mode, opts)
	if s.redis != nil {
		val, err := s.redis.Get(ctx, cacheKey).Result()
		if err == nil {
			// Try decompressing
			// Note: val is string, convert to byte
			b := bytes.NewReader([]byte(val))
			gz, err := gzip.NewReader(b)
			if err == nil {
				defer gz.Close()
				decompressed, err := io.ReadAll(gz)
				if err == nil {
					var result domain.ScrapeResult
					if err := json.Unmarshal(decompressed, &result); err == nil {
						if result.Metadata == nil {
							result.Metadata = make(map[string]string)
						}
						result.Metadata["cached"] = "true"
						return &result, nil
					}
				}
			} else {
				// Fallback: maybe it's legacy uncompressed data?
				// Try unmarshal directly
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
	}

	// 2. Scrape
	var result *domain.ScrapeResult
	var err error

	// Helper to run dynamic
	runDynamic := func() (*domain.ScrapeResult, error) {
		if s.chromedp == nil {
			return nil, fmt.Errorf("dynamic scraper not configured")
		}
		return s.chromedp.Scrape(ctx, url, opts)
	}

	// Helper to run static
	runStatic := func() (*domain.ScrapeResult, error) {
		if s.colly == nil {
			return nil, fmt.Errorf("static scraper not configured")
		}
		return s.colly.Scrape(ctx, url, opts)
	}

	switch mode {
	case "dynamic":
		result, err = runDynamic()
	case "static":
		result, err = runStatic()
	case "smart":
		// Fallthrough to smart logic
		// If screenshot is requested, we MUST use dynamic.
		if opts.Screenshot {
			return runDynamic()
		}

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

	// 3. Extract images if requested
	if opts.Images && result != nil && result.HTML != "" {
		maxImages := opts.MaxImages
		if maxImages <= 0 {
			maxImages = 10
		}
		result.Images = image.ExtractPageImages(result.HTML, url, maxImages)

		// Fetch and encode each image if blob format requested.
		// Fetches run concurrently (bounded) to avoid N serial round trips;
		// per-image failures are logged and skipped, preserving the response.
		if opts.ImageFormat == domain.ImageFormatBlob && len(result.Images) > 0 {
			proc := image.NewProcessor()
			maxBytes := int64(opts.MaxImageSizeKB) * 1024

			g, gctx := errgroup.WithContext(ctx)
			g.SetLimit(5)
			for i := range result.Images {
				img := &result.Images[i]
				if img.URL == "" {
					continue
				}
				g.Go(func() error {
					blob, fetchErr := proc.FetchAndEncodeLimit(img.URL, maxBytes)
					if fetchErr != nil {
						// Context cancellation is real; everything else is a
						// per-image skip.
						if gctx.Err() != nil {
							return fetchErr
						}
						logger.Log.Warn("Failed to fetch image for blob", "url", img.URL, "error", fetchErr)
						return nil
					}
					img.Blob = blob.DataURI
					if strings.HasPrefix(blob.MimeType, "image/") {
						img.Format = strings.TrimPrefix(blob.MimeType, "image/")
					}
					img.SizeBytes = int64(len(blob.RawBytes))
					return nil
				})
			}
			if gerr := g.Wait(); gerr != nil {
				return nil, fmt.Errorf("image fetch aborted: %w", gerr)
			}

			// Optional post-processing: resize/re-encode fetched blobs.
			if opts.ImageProcess != nil {
				for i := range result.Images {
					if result.Images[i].Blob == "" {
						continue
					}
					raw, _, decErr := image.DecodeDataURI(result.Images[i].Blob)
					if decErr != nil {
						logger.Log.Warn("Failed to decode image blob for processing", "url", result.Images[i].URL, "error", decErr)
						continue
					}
					processed, mime, procErr := image.Reencode(raw, *opts.ImageProcess)
					if procErr != nil {
						logger.Log.Warn("Failed to process image", "url", result.Images[i].URL, "error", procErr)
						continue
					}
					result.Images[i].Blob = image.NewProcessor().EncodeToDataURI(processed, mime)
					result.Images[i].Format = strings.TrimPrefix(mime, "image/")
					result.Images[i].SizeBytes = int64(len(processed))
				}
			}
		}
	}

	// 4. Save to Cache
	if s.redis != nil {
		data, err := json.Marshal(result)
		if err == nil {
			// Compress data
			var b bytes.Buffer
			gz := gzip.NewWriter(&b)
			if _, err := gz.Write(data); err == nil {
				gz.Close()
				// Store for 7 days as requested (low storage usage allows this)
				s.redis.Set(ctx, cacheKey, b.Bytes(), 7*24*time.Hour)
			}
		}
	}

	return result, nil
}
