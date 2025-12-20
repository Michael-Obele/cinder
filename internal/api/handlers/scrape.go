package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/standard-user/cinder/internal/domain"
	"github.com/standard-user/cinder/pkg/logger"
)

type ScrapeRequest struct {
	URL string `json:"url" binding:"required,url"`
}

type ScrapeHandler struct {
	scraper domain.Scraper
}

func NewScrapeHandler(s domain.Scraper) *ScrapeHandler {
	return &ScrapeHandler{scraper: s}
}

func (h *ScrapeHandler) Scrape(c *gin.Context) {
	var req ScrapeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Log.Warn("Invalid request", "error", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid URL"})
		return
	}

	result, err := h.scraper.Scrape(c.Request.Context(), req.URL)
	if err != nil {
		logger.Log.Error("Scrape failed", "url", req.URL, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Scraping failed"})
		return
	}

	c.JSON(http.StatusOK, result)
}
