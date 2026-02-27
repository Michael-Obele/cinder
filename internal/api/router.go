package api

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/standard-user/cinder/internal/api/handlers"
	"github.com/standard-user/cinder/internal/api/middleware"
	"github.com/standard-user/cinder/internal/config"

	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"github.com/swaggo/swag"
)

func NewRouter(cfg *config.Config, logger *slog.Logger, scrapeHandler *handlers.ScrapeHandler, crawlHandler *handlers.CrawlHandler, searchHandler *handlers.SearchHandler) *gin.Engine {
	if cfg.Server.Mode == "release" {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()

	r.Use(gin.Recovery())
	r.Use(middleware.Logger(logger))

	// Swagger Docs mapping
	if cfg.Server.Mode == "debug" {
		// Register a dummy doc so ginSwagger doesn't 404
		swag.Register(swag.Name, &swag.Spec{
			InfoInstanceName: "swagger",
			SwaggerTemplate:  `{"swagger":"2.0", "info":{"title":"Swagger UI", "version":"1.0"}}`,
		})
		r.GET("/swagger-docs/swagger.json", func(c *gin.Context) {
			c.File("./internal/api/docs/swagger.json")
		})
		r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler, ginSwagger.URL("/swagger-docs/swagger.json")))
	} else {
		// Optionally, you might want to disable swagger entirely in release
		r.GET("/swagger/*any", func(c *gin.Context) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Swagger available only in debug mode"})
		})
	}

	v1 := r.Group("/v1")
	{
		v1.POST("/scrape", scrapeHandler.Scrape)
		v1.GET("/scrape", scrapeHandler.Scrape)
		v1.POST("/search", searchHandler.Search)
		v1.GET("/search", searchHandler.Search)

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
