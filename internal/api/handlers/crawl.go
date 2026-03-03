package handlers

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"net/url"

	"github.com/gin-gonic/gin"
	"github.com/hibiken/asynq"
	"github.com/standard-user/cinder/internal/worker"
)

type CrawlRequest struct {
	URL        string `json:"url" binding:"required,url"`
	Render     bool   `json:"render"`
	Screenshot bool   `json:"screenshot"`
	Images     bool   `json:"images"`
	MaxDepth   int    `json:"maxDepth"`
	Limit      int    `json:"limit"`
}

type CrawlResponse struct {
	ID         string `json:"id"`
	URL        string `json:"url"`
	Render     bool   `json:"render"`
	Screenshot bool   `json:"screenshot"`
	Images     bool   `json:"images"`
	MaxDepth   int    `json:"maxDepth"`
	Limit      int    `json:"limit"`
}

type CrawlHandler struct {
	client    *asynq.Client
	inspector *asynq.Inspector
}

func NewCrawlHandler(redisAddr string) (*CrawlHandler, error) {
	u, err := url.Parse(redisAddr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse redis url: %w", err)
	}

	password, _ := u.User.Password()
	addr := u.Host

	redisOpt := asynq.RedisClientOpt{
		Addr:     addr,
		Password: password,
	}

	if u.Scheme == "rediss" {
		redisOpt.TLSConfig = &tls.Config{
			InsecureSkipVerify: false, // Set to true for self-signed certs
			MinVersion:         tls.VersionTLS12,
		}
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

// EnqueueCrawl godoc
// @Summary      Enqueue a URL for asynchronous crawling
// @Description  Submits a URL to be crawled asynchronously. The crawler performs BFS link-following up to maxDepth, scraping up to limit pages on the same domain. Requires Redis.
// @Tags         crawl
// @Accept       json
// @Produce      json
// @Param        body   body      CrawlRequest   true  "JSON request body"
// @Success      202    {object}  CrawlResponse
// @Failure      400    {object}  map[string]interface{}
// @Failure      500    {object}  map[string]interface{}
// @Router       /crawl [post]
func (h *CrawlHandler) EnqueueCrawl(c *gin.Context) {
	var req CrawlRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Apply defaults
	if req.MaxDepth <= 0 {
		req.MaxDepth = 2
	}
	if req.MaxDepth > 10 {
		req.MaxDepth = 10
	}
	if req.Limit <= 0 {
		req.Limit = 10
	}
	if req.Limit > 100 {
		req.Limit = 100
	}

	task, err := worker.NewCrawlTask(req.URL, req.Render, req.Screenshot, req.Images, req.MaxDepth, req.Limit)
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
		ID:         info.ID,
		URL:        req.URL,
		Render:     req.Render,
		Screenshot: req.Screenshot,
		Images:     req.Images,
		MaxDepth:   req.MaxDepth,
		Limit:      req.Limit,
	})
}

// GetCrawlStatus godoc
// @Summary      Get the status of an asynchronous crawl
// @Description  Retrieves the current status and result of a previously enqueued crawl task by its ID.
// @Tags         crawl
// @Produce      json
// @Param        id     path      string  true  "The crawl task ID"
// @Success      200    {object}  map[string]interface{} "The task status and result payload"
// @Failure      404    {object}  map[string]interface{} "Task not found"
// @Router       /crawl/{id} [get]
func (h *CrawlHandler) GetCrawlStatus(c *gin.Context) {
	id := c.Param("id")

	// Search across all configured queues
	queues := []string{"default", "critical", "low"}
	var info *asynq.TaskInfo
	var err error

	for _, q := range queues {
		info, err = h.inspector.GetTaskInfo(q, id)
		if err == nil {
			break
		}
	}

	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "task not found"})
		return
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
