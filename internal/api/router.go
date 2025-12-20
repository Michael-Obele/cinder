package api

import (
	"github.com/gin-gonic/gin"
	"github.com/standard-user/cinder/internal/api/handlers"
	"github.com/standard-user/cinder/internal/config"
	"github.com/standard-user/cinder/internal/domain"
	"github.com/standard-user/cinder/pkg/logger"
)

func NewRouter(cfg *config.Config, scraper domain.Scraper) *gin.Engine {
	if cfg.Server.Mode == "release" {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()
	r.Use(gin.Recovery())

	// Custom logging middleware using our slog logger
	r.Use(func(c *gin.Context) {
		logger.Log.Info("Request", "method", c.Request.Method, "path", c.Request.URL.Path, "ip", c.ClientIP())
		c.Next()
	})

	scrapeHandler := handlers.NewScrapeHandler(scraper)

	v1 := r.Group("/v1")
	{
		v1.POST("/scrape", scrapeHandler.Scrape)
	}

	return r
}
