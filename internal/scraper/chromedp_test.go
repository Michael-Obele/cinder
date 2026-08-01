package scraper

import (
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
