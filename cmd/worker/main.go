package main

import (
	"fmt"
	"os"

	"github.com/hibiken/asynq"
	"github.com/redis/go-redis/v9"
	"github.com/standard-user/cinder/internal/config"
	"github.com/standard-user/cinder/internal/scraper"
	"github.com/standard-user/cinder/internal/worker"
	"github.com/standard-user/cinder/pkg/logger"
)

func main() {
	// 1. Load Configuration
	cfg, err := config.Load()
	if err != nil {
		fmt.Printf("Failed to load config: %v\n", err)
		os.Exit(1)
	}

	// 2. Initialize Logger
	logger.Init(cfg.App.LogLevel)
	logger.Log.Info("Starting Cinder Worker")

	// Check if Redis is configured
	if cfg.Redis.URL == "" {
		logger.Log.Warn("Redis URL not configured, worker cannot start. Use synchronous scraping only.")
		os.Exit(0)
	}

	// 3. Initialize Scrapers
	// Create standard go-redis client for caching (asynq uses its own connection)
	redisOpt, err := redis.ParseURL(cfg.Redis.URL)
	if err != nil {
		logger.Log.Error("Failed to parse Redis URI", "error", err)
		os.Exit(1)
	}
	redisClient := redis.NewClient(redisOpt)

	collyScraper := scraper.NewCollyScraper()
	chromedpScraper := scraper.NewChromedpScraper()
	scraperService := scraper.NewService(collyScraper, chromedpScraper, redisClient)

	// 4. Initialize Asynq Server
	srv := worker.NewServer(cfg, logger.Log)

	// 5. Register Handlers
	mux := asynq.NewServeMux()
	worker.RegisterHandlers(mux, scraperService, logger.Log)

	// 6. Run
	logger.Log.Info("Worker is running...")
	if err := srv.Run(mux); err != nil {
		logger.Log.Error("Could not run worker server", "error", err)
		os.Exit(1)
	}
}
