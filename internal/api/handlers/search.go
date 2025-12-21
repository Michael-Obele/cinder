package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/standard-user/cinder/internal/search"
)

type SearchRequest struct {
	Query string `json:"query" binding:"required"`
}

type SearchHandler struct {
	service search.Service
}

func NewSearchHandler(s search.Service) *SearchHandler {
	return &SearchHandler{
		service: s,
	}
}

func (h *SearchHandler) Search(c *gin.Context) {
	var req SearchRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	results, err := h.service.Search(c.Request.Context(), req.Query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"query":   req.Query,
		"results": results,
		"count":   len(results),
	})
}
