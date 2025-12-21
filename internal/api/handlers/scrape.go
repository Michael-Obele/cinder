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

func (h *ScrapeHandler) Scrape(c *gin.Context) {
	var req ScrapeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Log.Warn("Invalid request", "error", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid URL"})
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
