package main

import (
	"fmt"
	"os"

	"github.com/standard-user/cinder/internal/api"
	"github.com/standard-user/cinder/internal/api/handlers"
	"github.com/standard-user/cinder/internal/config"
	"github.com/standard-user/cinder/internal/scraper"
	"github.com/standard-user/cinder/pkg/logger"
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
	collyScraper := scraper.NewCollyScraper()
	chromedpScraper := scraper.NewChromedpScraper()
	scraperService := scraper.NewService(collyScraper, chromedpScraper)

	// Initialize Handlers
	scrapeHandler := handlers.NewScrapeHandler(scraperService)
	crawlHandler := handlers.NewCrawlHandler(cfg.Redis.URL)
	defer crawlHandler.Close()

	// 4. Init Router
	// Router
	router := api.NewRouter(cfg, logger.Log, scrapeHandler, crawlHandler)

	// 5. Run Server
	addr := fmt.Sprintf(":%s", cfg.Server.Port)
	if err := router.Run(addr); err != nil { // Changed r to router
		logger.Log.Error("Server failed to start", "error", err)
		os.Exit(1)
	}
}
