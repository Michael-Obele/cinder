package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/standard-user/cinder/internal/search"
)

type SearchRequest struct {
	Query          string   `json:"query" binding:"required"`
	Offset         int      `json:"offset"`
	Limit          int      `json:"limit"`
	IncludeDomains []string `json:"includeDomains,omitempty"`
	ExcludeDomains []string `json:"excludeDomains,omitempty"`
	RequiredText   []string `json:"requiredText,omitempty"`
	MaxAge         *int     `json:"maxAge,omitempty"`
	Mode           string   `json:"mode"`
}

type SearchResponse struct {
	Query      string          `json:"query"`
	Results    []search.Result `json:"results"`
	HasMore    bool            `json:"hasMore"`
	NextOffset int             `json:"nextOffset"`
	Count      int             `json:"count"`
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

	// Parse offset and limit from query params if provided (alternative to JSON body)
	if offsetStr := c.Query("offset"); offsetStr != "" {
		if offset, err := strconv.Atoi(offsetStr); err == nil {
			req.Offset = offset
		}
	}
	if limitStr := c.Query("limit"); limitStr != "" {
		if limit, err := strconv.Atoi(limitStr); err == nil {
			req.Limit = limit
		}
	}

	// Set defaults
	if req.Limit == 0 {
		req.Limit = 10
	}
	if req.Limit > 100 {
		req.Limit = 100
	}
	if req.Offset < 0 {
		req.Offset = 0
	}

	results, totalCount, err := h.service.Search(c.Request.Context(), search.SearchOptions{
		Query:          req.Query,
		Offset:         req.Offset,
		Limit:          req.Limit,
		IncludeDomains: req.IncludeDomains,
		ExcludeDomains: req.ExcludeDomains,
		RequiredText:   req.RequiredText,
		MaxAge:         req.MaxAge,
		Mode:           req.Mode,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	hasMore := req.Offset+req.Limit < totalCount
	nextOffset := req.Offset + req.Limit

	c.JSON(http.StatusOK, SearchResponse{
		Query:      req.Query,
		Results:    results,
		HasMore:    hasMore,
		NextOffset: nextOffset,
		Count:      len(results),
	})
}
