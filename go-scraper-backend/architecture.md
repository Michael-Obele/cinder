# Architecture & Technical Design

## System Overview

We use a **Hexagonal Architecture** (also known as Ports and Adapters). This keeps our core logic (Scraping, Cleaning) independent of the framework (Gin) or the tools (Colly/Chromedp).

```mermaid
graph TD
    User[User] --> |HTTP| API[Gin API Layer]
    API --> |Calls| Service[Scraper Service]

    subgraph "Core Domain"
        Service --> |Selects| Engine{Engine Selector}
        Engine --> |Static| Colly[Colly Adapter]
        Engine --> |Dynamic| Chrome[Chromedp Adapter]
        Service --> |Cleans| Converter[HTML-to-Markdown]
    end

    subgraph "Async Worker"
        Queue[Redis/Asynq] --> |Triggers| Worker[Crawl Worker]
        Worker --> |Calls| Service
    end
```

## 📂 Project Structure

This follows the [Standard Go Project Layout](https://github.com/golang-standards/project-layout).

```text
go-scraper-backend/
├── cmd/
│   ├── api/                # Main entry point for HTTP API
│   │   └── main.go
│   └── worker/             # Main entry point for Async Worker
│       └── main.go
├── internal/
│   ├── api/                # Gin Handlers & Middleware
│   │   ├── handlers/
│   │   ├── middleware/
│   │   └── router.go
│   ├── config/             # Viper configuration loading
│   ├── domain/             # Interfaces & Data Structs (Pure Go)
│   ├── scraper/            # The scraping logic
│   │   ├── colly.go        # Static implementation
│   │   ├── chromedp.go     # Dynamic implementation
│   │   └── service.go      # Logic to choose engine
│   └── worker/             # Asynq task handlers
├── pkg/                    # Public utilities (logger, etc.)
├── Dockerfile
├── go.mod
└── go.sum
```

## 💻 Code Samples & Patterns

### 1. The Scraper Interface (Polymorphism)

This allows us to switch between Colly and Chromedp easily.

```go
// internal/domain/scraper.go

type ScrapeRequest struct {
    URL         string
    RenderJS    bool   // If true, use Chromedp
    WaitFor     int    // Milliseconds to wait for JS
}

type ScrapeResult struct {
    Content  string // Markdown
    HTML     string // Raw HTML (optional)
    Title    string
    Metadata map[string]string
}

// The Interface
type ScraperEngine interface {
    Visit(ctx context.Context, req ScrapeRequest) (*ScrapeResult, error)
}
```

### 2. User Agent Rotation (gofakeit)

We use `gofakeit` to generate a random User Agent for every request to avoid detection.

```go
// internal/scraper/service.go
import "github.com/brianvoe/gofakeit/v6"

func (s *ScraperService) GetUserAgent() string {
    // You can customize this to only return Chrome/Desktop agents
    return gofakeit.UserAgent()
}

// Usage in Colly
c := colly.NewCollector()
c.OnRequest(func(r *colly.Request) {
    r.Headers.Set("User-Agent", s.GetUserAgent())
})

// Usage in Chromedp
chromedp.Run(ctx,
    chromedp.UserAgent(s.GetUserAgent()),
    chromedp.Navigate(url),
)
```

### 3. Gin Handler with "Smart Switch"

How we handle the request in `internal/api/handlers/scrape.go`.

```go
func (h *ScrapeHandler) HandleScrape(c *gin.Context) {
    var req domain.ScrapeRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(400, gin.H{"error": err.Error()})
        return
    }

    // "Smart Switch" Logic
    // If user asks for render OR we detect a need (future), use Dynamic
    var result *domain.ScrapeResult
    var err error

    if req.RenderJS {
        result, err = h.DynamicScraper.Visit(c.Request.Context(), req)
    } else {
        result, err = h.StaticScraper.Visit(c.Request.Context(), req)
    }

    if err != nil {
        c.JSON(500, gin.H{"error": "Failed to scrape: " + err.Error()})
        return
    }

    c.JSON(200, gin.H{"success": true, "data": result})
}
```

### 3. Redis TLS Configuration (For Leapcell/Upstash)

This is critical for your deployment. Asynq needs a specific setup for TLS.

```go
// internal/config/redis.go

import (
    "crypto/tls"
    "github.com/hibiken/asynq"
)

func NewRedisOpt(url string) (*asynq.RedisClientOpt, error) {
    // Parse the URL (e.g., "rediss://user:pass@host:port")
    opts, err := asynq.ParseRedisURI(url)
    if err != nil {
        return nil, err
    }

    // If using rediss:// (TLS), we might need to ensure TLSConfig is set
    // ParseRedisURI usually handles this, but for some providers you might need:
    if opts.TLSConfig == nil && strings.HasPrefix(url, "rediss://") {
        opts.TLSConfig = &tls.Config{
            InsecureSkipVerify: false, // Set true only for self-signed certs
            MinVersion:         tls.VersionTLS12,
        }
    }

    return &opts, nil
}
```

### 4. Simple API Key Middleware

Secure your API for future public use.

```go
// internal/api/middleware/auth.go

func APIKeyAuth(validKey string) gin.HandlerFunc {
    return func(c *gin.Context) {
        // Check header "Authorization: Bearer <key>" or "X-API-Key"
        key := c.GetHeader("X-API-Key")
        if key == "" {
            // Fallback to Bearer token
            authHeader := c.GetHeader("Authorization")
            if len(authHeader) > 7 && strings.ToUpper(authHeader[0:6]) == "BEARER" {
                key = authHeader[7:]
            }
        }

        if key != validKey {
            c.AbortWithStatusJSON(401, gin.H{"error": "Unauthorized"})
            return
        }

        c.Next()
    }
}
```

### 5. Dockerfile for Chromedp

Running Chrome in Docker requires specific dependencies.

```dockerfile
# Build Stage
FROM golang:1.23-alpine AS builder
WORKDIR /app
COPY . .
RUN go build -o api cmd/api/main.go

# Final Stage
FROM alpine:latest

# CRITICAL: Install Chromium and dependencies
RUN apk add --no-cache \
    chromium \
    ca-certificates \
    tzdata

# Set env to tell Chromedp where chrome is (optional, usually auto-detected)
ENV CHROME_BIN=/usr/bin/chromium-browser
ENV CHROME_PATH=/usr/lib/chromium/

WORKDIR /app
COPY --from=builder /app/api .
COPY .env .

EXPOSE 8080
CMD ["./api"]
```
