package scraper

import (
	"context"
	"sync/atomic"
	"testing"

	"github.com/standard-user/cinder/internal/domain"
)

func TestResolveScreenshotParams_Defaults(t *testing.T) {
	p := resolveScreenshotParams(nil)
	if p.width != 1920 || p.height != 1080 {
		t.Errorf("expected default viewport 1920x1080, got %dx%d", p.width, p.height)
	}
	if p.format != "jpeg" || p.quality != 90 || p.fullPage {
		t.Errorf("unexpected defaults: format=%s quality=%d fullPage=%v", p.format, p.quality, p.fullPage)
	}
}

func TestResolveScreenshotParams_AppliesAndClamps(t *testing.T) {
	opts := &domain.ScreenshotOptions{
		Width: 800, Height: 600, FullPage: true, Format: "png",
		Quality: 150, WaitSelector: "#app",
	}
	p := resolveScreenshotParams(opts)
	if p.width != 800 || p.height != 600 {
		t.Errorf("expected 800x600, got %dx%d", p.width, p.height)
	}
	if !p.fullPage || p.format != "png" || p.waitSelector != "#app" {
		t.Errorf("unexpected resolved params: %+v", p)
	}
	if p.quality != 90 {
		t.Errorf("quality 150 should clamp to default 90, got %d", p.quality)
	}
}

func TestResolveScreenshotParams_IgnoresBadFormat(t *testing.T) {
	p := resolveScreenshotParams(&domain.ScreenshotOptions{Format: "bmp"})
	if p.format != "jpeg" {
		t.Errorf("unknown format should fall back to jpeg, got %q", p.format)
	}
}

func TestShouldRecycle(t *testing.T) {
	tests := []struct {
		count, after int
		want         bool
	}{
		{99, 100, false},
		{100, 100, true},
		{101, 100, true},
		{5, 0, false}, // disabled
	}
	for _, tt := range tests {
		if got := shouldRecycle(tt.count, tt.after); got != tt.want {
			t.Errorf("shouldRecycle(%d, %d) = %v, want %v", tt.count, tt.after, got, tt.want)
		}
	}
}

// TestBeginScrape_RecyclesAllocator verifies the allocator factory is
// re-invoked once the scrape counter crosses the recycle threshold.
func TestBeginScrape_RecyclesAllocator(t *testing.T) {
	var calls int32
	factory := func() (context.Context, context.CancelFunc) {
		atomic.AddInt32(&calls, 1)
		return context.WithCancel(context.Background())
	}

	s := &ChromedpScraper{recycleAfter: 2, newAllocator: factory}
	s.allocCtx, s.cancel = factory()

	if got := s.beginScrape(); got == nil {
		t.Fatal("beginScrape returned nil context")
	}
	if got := s.beginScrape(); got == nil {
		t.Fatal("beginScrape returned nil context")
	}
	// Threshold (2) crossed on the 2nd scrape → allocator recreated once.
	if c := atomic.LoadInt32(&calls); c != 2 {
		t.Errorf("expected 2 allocator creations (1 init + 1 recycle), got %d", c)
	}
	if s.scrapeCount != 0 {
		t.Errorf("scrapeCount should reset after recycle, got %d", s.scrapeCount)
	}
	if got := s.beginScrape(); got == nil {
		t.Fatal("beginScrape returned nil context after recycle")
	}
	if c := atomic.LoadInt32(&calls); c != 2 {
		t.Errorf("no further recreation expected below threshold, got %d calls", c)
	}
}
