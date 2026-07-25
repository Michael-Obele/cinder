package handlers

import (
	"net/http"
	"strconv"
	"unicode"

	"github.com/gin-gonic/gin"
	"github.com/standard-user/cinder/internal/domain"
	"github.com/standard-user/cinder/internal/scraper"
	"github.com/standard-user/cinder/pkg/logger"
)

type ScrapeRequest struct {
	URL            string `json:"url" binding:"required,url"`
	Render         bool   `json:"render"` // Deprecated: usage ignores Mode if true
	Mode           string `json:"mode"`   // "smart", "static", "dynamic"
	Screenshot     bool   `json:"screenshot"`
	Images         bool   `json:"images"`
	ImageFormat    string `json:"image_format"` // "url" or "blob"
	MaxImages      int    `json:"max_images"`
	MaxImageSizeKB int    `json:"max_image_size_kb"`
}

type ScrapeHandler struct {
	service *scraper.Service
}

func NewScrapeHandler(s *scraper.Service) *ScrapeHandler {
	return &ScrapeHandler{service: s}
}

// wordCount returns the number of whitespace-delimited words in the given text.
func wordCount(text string) int {
	if text == "" {
		return 0
	}
	count := 0
	inWord := false
	for _, r := range text {
		if unicode.IsSpace(r) {
			inWord = false
		} else if !inWord {
			count++
			inWord = true
		}
	}
	return count
}

// Scrape godoc
// @Summary      Scrape a webpage
// @Description  Scrapes a given URL and returns its markdown content, metadata, and optionally captures a screenshot or extracts images if enabled.
// @Tags         scrape
// @Accept       json
// @Produce      json
// @Param        url              query     string  false  "The URL to scrape"
// @Param        mode             query     string  false  "Scraping mode: smart, static, dynamic"
// @Param        render           query     bool    false  "Deprecated: use mode=dynamic instead"
// @Param        screenshot       query     bool    false  "Capture full-page screenshot (requires mode=dynamic or smart)"
// @Param        images           query     bool    false  "Extract images from the page"
// @Param        image_format     query     string  false  "Image transport: 'url' (default, metadata only) or 'blob' (base64 data URIs)"
// @Param        max_images       query     int     false  "Maximum images to extract (default: 10)"
// @Param        max_image_size_kb query    int     false  "Max individual image size in KB (default: 5120)"
// @Param        body             body      ScrapeRequest  false  "JSON request body (alternative to query params)"
// @Success      200    {object}  domain.ScrapeResult
// @Failure      400    {object}  map[string]interface{}
// @Failure      500    {object}  map[string]interface{}
// @Router       /scrape [post]
// @Router       /scrape [get]
func (h *ScrapeHandler) Scrape(c *gin.Context) {
	var req ScrapeRequest

	// Try to bind from JSON first (POST)
	if c.Request.Method == http.MethodPost && c.Request.ContentLength > 0 {
		if err := c.ShouldBindJSON(&req); err != nil {
			logger.Log.Warn("Invalid check", "error", err)
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON body"})
			return
		}
	}

	// Parse parameters from query strings (GET or POST)
	if url := c.Query("url"); url != "" {
		req.URL = url
	}
	if mode := c.Query("mode"); mode != "" {
		req.Mode = mode
	}
	if render := c.Query("render"); render == "true" {
		req.Render = true
	}
	if screenshot := c.Query("screenshot"); screenshot == "true" {
		req.Screenshot = true
	}
	if images := c.Query("images"); images == "true" {
		req.Images = true
	}
	if imageFormat := c.Query("image_format"); imageFormat != "" {
		req.ImageFormat = imageFormat
	}
	if maxImagesStr := c.Query("max_images"); maxImagesStr != "" {
		if maxImages, err := strconv.Atoi(maxImagesStr); err == nil {
			req.MaxImages = maxImages
		}
	}
	if maxSizeStr := c.Query("max_image_size_kb"); maxSizeStr != "" {
		if maxSize, err := strconv.Atoi(maxSizeStr); err == nil {
			req.MaxImageSizeKB = maxSize
		}
	}

	if req.URL == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "URL is required"})
		return
	}

	// Backward compatibility mapping
	mode := req.Mode
	if req.Render {
		mode = "dynamic"
	}
	if mode == "" {
		mode = "smart"
	}

	// Parse image format
	imageFormat := domain.ImageFormatURL
	switch req.ImageFormat {
	case "blob":
		imageFormat = domain.ImageFormatBlob
	case "url":
		imageFormat = domain.ImageFormatURL
	}

	result, err := h.service.Scrape(c.Request.Context(), req.URL, mode, domain.ScrapeOptions{
		Screenshot:     req.Screenshot,
		Images:         req.Images,
		ImageFormat:    imageFormat,
		MaxImages:      req.MaxImages,
		MaxImageSizeKB: req.MaxImageSizeKB,
	})
	if err != nil {
		logger.Log.Error("Scrape failed", "url", req.URL, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Scraping failed"})
		return
	}

	// Extract title and word count for the summary
	title := ""
	if result.Markdown != "" {
		title = extractTitle(result.Markdown)
	}

	c.JSON(http.StatusOK, gin.H{
		"url":        result.URL,
		"title":      title,
		"word_count": wordCount(result.Markdown),
		"markdown":   result.Markdown,
		"html":       result.HTML,
		"metadata":   result.Metadata,
		"screenshot": result.Screenshot,
		"images":     result.Images,
	})
}
