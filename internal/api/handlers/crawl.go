package handlers

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/hibiken/asynq"
	"github.com/standard-user/cinder/internal/worker"
)

type CrawlRequest struct {
	URL    string `json:"url" binding:"required,url"`
	Render bool   `json:"render"`
}

type CrawlResponse struct {
	ID string `json:"id"`
	URL string `json:"url"`
	Render bool `json:"render"`
}

type CrawlHandler struct {
	client *asynq.Client
	inspector *asynq.Inspector
}

func NewCrawlHandler(redisAddr string) (*CrawlHandler, error) {
	redisOpt, err := asynq.ParseRedisURI(redisAddr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse redis uri: %w", err)
	}
	client := asynq.NewClient(redisOpt)
	inspector := asynq.NewInspector(redisOpt)
	return &CrawlHandler{
		client:    client,
		inspector: inspector,
	}, nil
}

func (h *CrawlHandler) Close() {
	h.client.Close()
	h.inspector.Close()
}

func (h *CrawlHandler) EnqueueCrawl(c *gin.Context) {
	var req CrawlRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	task, err := worker.NewScrapeTask(req.URL, req.Render)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create task"})
		return
	}

	info, err := h.client.Enqueue(task)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to enqueue task"})
		return
	}

	c.JSON(http.StatusAccepted, CrawlResponse{
		ID:     info.ID,
		URL:    req.URL,
		Render: req.Render,
	})
}

func (h *CrawlHandler) GetCrawlStatus(c *gin.Context) {
	id := c.Param("id")
	info, err := h.inspector.GetTaskInfo("default", id) // Assuming default queue for simple lookup, but we might need to search all queues
	// Actually GetTaskInfo takes (queue, id). If we don't know the queue, passing empty string usually fails or isn't supported directly easily without listing.
	// Asynq Inspector GetTaskInfo requires queue name.
	// Let's assume 'default' queue for now or check 'critical'/'low' if allowed.
	// A better way is using `inspector.GetTaskInfo` (it DOES require queue name).
	// For simplicity, we'll try "default". If not found, we might return error.
	
	// Wait, generic GetTaskInfo might not exist across queues.
	// Let's check simply providing queue "default".
	if err != nil {
		// Try critical
		info, err = h.inspector.GetTaskInfo("critical", id)
		if err != nil {
			// Try low
			info, err = h.inspector.GetTaskInfo("low", id)
			if err != nil {
				c.JSON(http.StatusNotFound, gin.H{"error": "task not found"})
				return
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"id":        info.ID,
		"queue":     info.Queue,
		"state":     info.State.String(),
		"max_retry": info.MaxRetry,
		"retried":   info.Retried,
		"payload":   string(info.Payload),
		"result":    string(info.Result), 
	})
}
