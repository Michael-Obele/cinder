package worker

import (
	"crypto/tls"
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"time"

	"github.com/hibiken/asynq"
	"github.com/standard-user/cinder/internal/config"
	"github.com/standard-user/cinder/internal/scraper"
)

type AsynqLogger struct {
	logger *slog.Logger
}

func (l *AsynqLogger) Debug(args ...interface{}) {
	l.logger.Debug(fmt.Sprint(args...))
}

func (l *AsynqLogger) Info(args ...interface{}) {
	l.logger.Info(fmt.Sprint(args...))
}

func (l *AsynqLogger) Warn(args ...interface{}) {
	l.logger.Warn(fmt.Sprint(args...))
}

func (l *AsynqLogger) Error(args ...interface{}) {
	l.logger.Error(fmt.Sprint(args...))
}

func (l *AsynqLogger) Fatal(args ...interface{}) {
	l.logger.Error(fmt.Sprint(args...))
	os.Exit(1)
}

func NewServer(cfg *config.Config, logger *slog.Logger) *asynq.Server {
	redisURL := cfg.Redis.URL
	u, err := url.Parse(redisURL)
	if err != nil {
		panic(fmt.Sprintf("failed to parse redis url: %v", err))
	}

	password, _ := u.User.Password()
	addr := u.Host

	redisOpt := asynq.RedisClientOpt{
		Addr:     addr,
		Password: password,
	}

	if u.Scheme == "rediss" {
		redisOpt.TLSConfig = &tls.Config{
			InsecureSkipVerify: false,
			MinVersion:         tls.VersionTLS12,
		}
	}

	srv := asynq.NewServer(
		redisOpt,
		asynq.Config{
			Concurrency: 5, // Reduced for hobby tier (256MB VM)
			Queues: map[string]int{
				"critical": 6,
				"default":  3,
				"low":      1,
			},
			// 15s poll interval keeps Redis commands well within Upstash
			// free tier (500K/mo): ~172K checks/mo vs 2.6M at 1s.
			TaskCheckInterval: 15 * time.Second,
			Logger:            &AsynqLogger{logger: logger},
		},
	)

	return srv
}

func RegisterHandlers(mux *asynq.ServeMux, scraper *scraper.Service, logger *slog.Logger) {
	scrapeHandler := NewScrapeTaskHandler(scraper, logger)
	mux.HandleFunc(TypeScrape, scrapeHandler.ProcessTask)

	crawlHandler := NewCrawlTaskHandler(scraper, logger)
	mux.HandleFunc(TypeCrawl, crawlHandler.ProcessTask)
}
