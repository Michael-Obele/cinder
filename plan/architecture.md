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

**Optimization Note:** For `Chromedp`, we use a **Shared Allocator** pattern. We spin up one Chrome instance at startup and create lightweight "tabs" (Contexts) for each request. This saves ~1s per request.

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

## 🛠️ Local Development Setup

### Prerequisites

- **Go 1.23+**: Download from [golang.org](https://golang.org/dl/)
- **Redis**: For async queue processing
  - Local: Install Redis server (`brew install redis` on macOS, `apt install redis-server` on Ubuntu)
  - Or use a cloud Redis like Upstash/Leapcell Redis

### Environment Setup

1. **Clone the repository**:

   ```bash
   git clone <your-repo-url>
   cd go-scraper-backend
   ```

2. **Install dependencies**:

   ```bash
   go mod tidy
   ```

3. **Create environment file** (`.env`):

   ```bash
   # Copy from env.example
   cp env.example .env

   # Edit .env with your values:
   REDIS_URL=redis://localhost:6379  # or rediss:// for TLS
   API_KEY=your-secret-key           # for API authentication
   ```

4. **Start Redis** (if running locally):
   ```bash
   redis-server
   ```

### Building the Project

```bash
# Build the API server
go build -o bin/api cmd/api/main.go

# Build the async worker
go build -o bin/worker cmd/worker/main.go

# Or build both at once
go build ./cmd/...
```

### Running Tests

```bash
# Run all tests
go test ./...

# Run tests with coverage
go test -cover ./...

# Run tests for specific package
go test ./internal/scraper/...

# Run tests with verbose output
go test -v ./...
```

### Running Locally

#### Development Mode (API + Worker)

1. **Start the API server**:

   ```bash
   go run cmd/api/main.go
   ```

   Server starts on `http://localhost:8080`

2. **Start the async worker** (in another terminal):
   ```bash
   go run cmd/worker/main.go
   ```

#### Testing the API

```bash
# Test scraping endpoint
curl -X POST http://localhost:8080/v1/scrape \
  -H "Content-Type: application/json" \
  -H "X-API-Key: your-secret-key" \
  -d '{"url": "https://example.com", "renderJS": false}'

# Should return scraped content in JSON format
```

### Development Workflow

1. **Make changes** to code
2. **Run tests**: `go test ./...`
3. **Build**: `go build ./cmd/...`
4. **Run locally**: `go run cmd/api/main.go`
5. **Test endpoints** with curl or Postman
6. **Commit and push** changes

## 🚀 Deployment on Leapcell

Leapcell is a serverless platform that supports Go applications with pay-as-you-go pricing. Perfect for this scraper backend with its async processing capabilities.

### Prerequisites

- GitHub account
- Leapcell account ([leapcell.io](https://leapcell.io))

### Deployment Steps

1. **Push your code to GitHub**:

   ```bash
   git add .
   git commit -m "Ready for deployment"
   git push origin main
   ```

2. **Connect GitHub to Leapcell**:

   - Go to [Leapcell Dashboard](https://leapcell.io/dashboard)
   - Follow instructions to connect your GitHub account

3. **Create a new service**:

   - Click "New Service"
   - Select your repository from the list
   - Configure the service:

   | Field             | Value                                                         |
   | ----------------- | ------------------------------------------------------------- |
   | **Runtime**       | Go (Any version)                                              |
   | **Build Command** | `go mod tidy && go build -tags netgo -ldflags '-s -w' -o app` |
   | **Start Command** | `./app`                                                       |
   | **Port**          | `8080`                                                        |

4. **Environment Variables**:
   Add these in Leapcell dashboard:
   - `REDIS_URL`: Your Redis connection string (use Leapcell Redis or Upstash)
   - `API_KEY`: Your API key for authentication

### Redis Configuration for Leapcell

Leapcell provides Redis service. For TLS connections (required for Leapcell/Upstash Redis):

```bash
# In your .env or Leapcell env vars
REDIS_URL=rediss://username:password@host:port
```

The code already handles TLS configuration (see Redis TLS Configuration section above).

### Accessing Your Deployed App

Once deployed, Leapcell provides a URL like `your-app-name.leapcell.dev`

Test your deployment:

```bash
curl -X POST https://your-app-name.leapcell.dev/v1/scrape \
  -H "Content-Type: application/json" \
  -H "X-API-Key: your-api-key" \
  -d '{"url": "https://example.com"}'
```

### Continuous Deployments

Every push to the connected branch automatically triggers a build and deploy. Failed builds are safely rolled back, keeping your service running.

### Scaling and Monitoring

- **Auto-scaling**: Leapcell automatically scales based on traffic
- **Monitoring**: Check the Leapcell dashboard for logs, metrics, and performance
- **Pay-as-you-go**: Only pay for actual usage - no charges when idle

### Troubleshooting

- **Build failures**: Check Leapcell logs for Go compilation errors
- **Runtime errors**: Verify environment variables and Redis connectivity
- **TLS issues**: Ensure Redis URL uses `rediss://` for secure connections

For more help, join the [Leapcell Discord community](https://discord.gg/qF7efny8x2).
