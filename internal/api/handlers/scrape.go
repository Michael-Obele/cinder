package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/standard-user/cinder/internal/scraper"
	"github.com/standard-user/cinder/pkg/logger"
)

type ScrapeRequest struct {
	URL    string `json:"url" binding:"required,url"`
	Render bool   `json:"render"` // Deprecated: usage ignores Mode if true
	Mode   string `json:"mode"`   // "smart", "static", "dynamic"
}

type ScrapeHandler struct {
	service *scraper.Service
}

func NewScrapeHandler(s *scraper.Service) *ScrapeHandler {
	return &ScrapeHandler{service: s}
}

// Scrape godoc
// @Summary      Scrape a webpage
// @Description  Scrapes a given URL and returns its markdown content, metadata, and optionally captures a screenshot or extracts images if enabled.
// @Tags         scrape
// @Accept       json
// @Produce      json
// @Param        url    query     string  false  "The URL to scrape"
// @Param        mode   query     string  false  "Scraping mode: smart, static, dynamic"
// @Param        render query     bool    false  "Deprecated: use mode=dynamic instead"
// @Param        body   body      ScrapeRequest  false  "JSON request body (alternative to query params)"
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

	result, err := h.service.Scrape(c.Request.Context(), req.URL, mode)
	if err != nil {
		logger.Log.Error("Scrape failed", "url", req.URL, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Scraping failed"})
		return
	}

	c.JSON(http.StatusOK, result)
}
