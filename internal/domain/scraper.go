package domain

import "context"

type ScrapeResult struct {
	URL      string            `json:"url"`
	Markdown string            `json:"markdown"`
	HTML     string            `json:"html,omitempty"`
	Metadata map[string]string `json:"metadata,omitempty"`

	// Image fields (omitted when not requested)
	Screenshot *ScreenshotData `json:"screenshot,omitempty"`
	Images     []ImageData     `json:"images,omitempty"`
}

type ScrapeOptions struct {
	Mode           string               `json:"mode,omitempty"`
	Screenshot     bool                 `json:"screenshot"`
	Images         bool                 `json:"images"`
	ImageFormat    ImageTransportFormat `json:"image_format,omitempty"`
	ScreenshotOpts *ScreenshotOptions   `json:"screenshot_opts,omitempty"`
	MaxImages      int                  `json:"max_images,omitempty"`
	MaxImageSizeKB int                  `json:"max_image_size_kb,omitempty"`
	ImageProcess   *ImageProcessOptions `json:"image_process,omitempty"`
	Actions        []Action             `json:"actions,omitempty"`
}

// Action is a single page interaction executed before content capture
// (dynamic mode only). Supported types: wait_ms, wait_selector, click,
// scroll_down, scroll_to_bottom.
type Action struct {
	Type     string `json:"type"`
	Selector string `json:"selector,omitempty"`
	Ms       int    `json:"ms,omitempty"`
}

type Scraper interface {
	Scrape(ctx context.Context, url string, opts ScrapeOptions) (*ScrapeResult, error)
}
