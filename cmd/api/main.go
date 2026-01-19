package main

import (
	"fmt"
	"os"

	"github.com/standard-user/cinder/internal/api"
	"github.com/standard-user/cinder/internal/api/handlers"
	"github.com/standard-user/cinder/internal/config"
	"github.com/standard-user/cinder/internal/scraper"
	"github.com/standard-user/cinder/internal/search"
	"github.com/standard-user/cinder/internal/worker"
	"github.com/standard-user/cinder/pkg/logger"

	"github.com/hibiken/asynq"
	"github.com/redis/go-redis/v9"
)

func main() {
	// 1. Load Config
	cfg, err := config.Load()
	if err != nil {
		fmt.Printf("Failed to load config: %v\n", err)
		os.Exit(1)
	}

	// 2. Init Logger
	logger.Init(cfg.App.LogLevel)
	logger.Log.Info("Starting Cinder API", "port", cfg.Server.Port, "mode", cfg.Server.Mode)

	// 3. Init Scraper
	var redisClient *redis.Client
	if cfg.Redis.URL != "" {
		opt, err := redis.ParseURL(cfg.Redis.URL)
		if err != nil {
			logger.Log.Warn("Invalid Redis URL", "error", err)
		} else {
			redisClient = redis.NewClient(opt)
			logger.Log.Info("Redis caching enabled")
		}
	}

	collyScraper := scraper.NewCollyScraper()
	chromedpScraper := scraper.NewChromedpScraper()
	defer chromedpScraper.Close()
	scraperService := scraper.NewService(collyScraper, chromedpScraper, redisClient)

	// Initialize Handlers
	// Initialize Handlers
	scrapeHandler := handlers.NewScrapeHandler(scraperService)

	// Initialize Search Service (Brave)
	braveService := search.NewBraveService(cfg.Brave.APIKey)
	searchHandler := handlers.NewSearchHandler(braveService)

	// Try to initialize crawl handler (requires Redis)
	var crawlHandler *handlers.CrawlHandler
	if cfg.Redis.URL != "" {
		handler, err := handlers.NewCrawlHandler(cfg.Redis.URL)
		if err != nil {
			logger.Log.Warn("Redis not available, asynchronous crawling disabled", "error", err)
		} else {
			crawlHandler = handler
			defer crawlHandler.Close()
			logger.Log.Info("Asynchronous crawling enabled with Redis")

			// Monolith Mode: Start Embedded Worker if enabled (defaulting to TRUE for Hobby Tier)
			if os.Getenv("DISABLE_WORKER") != "true" {
				logger.Log.Info("Starting Embedded Worker (Monolith Mode)")
				workerServer := worker.NewServer(cfg, logger.Log)
				mux := asynq.NewServeMux()
				worker.RegisterHandlers(mux, scraperService, logger.Log)

				go func() {
					if err := workerServer.Run(mux); err != nil {
						logger.Log.Error("Embedded Worker failed", "error", err)
					}
				}()
			}
		}
	} else {
		logger.Log.Warn("Redis URL not configured, asynchronous crawling disabled")
	}

	// 4. Init Router
	router := api.NewRouter(cfg, logger.Log, scrapeHandler, crawlHandler, searchHandler)

	// 5. Run Server
	addr := fmt.Sprintf(":%s", cfg.Server.Port)
	if err := router.Run(addr); err != nil { // Changed r to router
		logger.Log.Error("Server failed to start", "error", err)
		os.Exit(1)
	}
}
