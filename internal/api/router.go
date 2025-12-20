package api

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/standard-user/cinder/internal/api/handlers"
	"github.com/standard-user/cinder/internal/api/middleware"
	"github.com/standard-user/cinder/internal/config"
)

func NewRouter(cfg *config.Config, logger *slog.Logger, scrapeHandler *handlers.ScrapeHandler, crawlHandler *handlers.CrawlHandler) *gin.Engine {
	if cfg.Server.Mode == "release" {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()

	r.Use(gin.Recovery())
	r.Use(middleware.Logger(logger))

	v1 := r.Group("/v1")
	{
		v1.POST("/scrape", scrapeHandler.Scrape)
		
		// Only register crawl routes if Redis/crawl handler is available
		if crawlHandler != nil {
			v1.POST("/crawl", crawlHandler.EnqueueCrawl)
			v1.GET("/crawl/:id", crawlHandler.GetCrawlStatus)
		} else {
			// Return 503 Service Unavailable for crawl endpoints when Redis is not available
			v1.POST("/crawl", func(c *gin.Context) {
				c.JSON(http.StatusServiceUnavailable, gin.H{
					"error": "Asynchronous crawling is not available - Redis connection required",
				})
			})
			v1.GET("/crawl/:id", func(c *gin.Context) {
				c.JSON(http.StatusServiceUnavailable, gin.H{
					"error": "Asynchronous crawling is not available - Redis connection required",
				})
			})
		}
	}

	return r
}
