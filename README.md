# Cinder 🔥

**Cinder** is a high-performance, self-hosted web scraping API built with Go. It is designed to be a drop-in alternative to Firecrawl, capable of turning any website into LLM-ready markdown.

**Note:** This repository is currently private.

Currently, the project has completed **Phase 1: Setup & Static Scraping**, **Phase 2: Dynamic Scraping**, and **Phase 3: Async Queue**, with **Phase 4: Polish & Auth** in progress.

---

## 🎯 Goal

Build a robust scraping service that can:

1. **Scrape**: Extract clean Markdown from any URL.
2. **Render**: Handle complex JavaScript/SPA sites (React, Vue, etc.) using a headless browser.
3. **Queue**: Manage heavy crawl jobs asynchronously using Redis.
4. **Scale**: Deploy easily via Docker with low memory footprint.
5. **Evade**: Rotate User Agents automatically to avoid bot detection.

---

## 🚀 Quickstart

### Prerequisites

- **Go 1.25+** installed.
- (Optional) Redis (for async features).
- (Optional) Docker (for containerized deployment).

### Environment Setup

Copy the example environment file and configure it:

```bash
cp go-scraper-backend/env.example .env
```

Edit `.env` with your settings. Key variables:

- `PORT`: Server port (default: 8080)
- `GIN_MODE`: Gin mode (`debug` or `release`)
- `LOG_LEVEL`: Logging level (`info`, `debug`, etc.)
- `API_KEY`: Simple API key for authentication (set to protect endpoints)
- `REDIS_URL`: Redis connection URL (required for async crawling)
- `MAX_CONCURRENCY`: Max concurrent scrapes (default: 5)

### Installation

```bash
git clone https://github.com/Michael-Obele/cinder.git
cd cinder
go mod download
```

### Running the API

```bash
# Run the API server directly
go run ./cmd/api

# Or build and run the binary
go build -o bin/cinder ./cmd/api
./bin/cinder
```

### Testing

```bash
# Run all tests
go test ./...

# Run with coverage
go test -cover ./...
```

---

## 📁 Project Structure

- `cmd/api/` — API server entrypoint (`main.go`) 🔧
- `cmd/worker/` — Async worker server entrypoint (`main.go`) ⏰
- `go-scraper-backend/` — Project documentation and planning 📚
- `internal/` — Internal packages
  - `api/` — Router and HTTP handlers (`router.go`, `handlers/scrape.go`)
  - `config/` — Configuration loader using Viper (`config.go`)
  - `domain/` — Domain models and interfaces (`scraper.go`)
  - `scraper/` — Scraper implementations (`colly.go`, `chromedp.go`)
  - `worker/` — Async task definitions and handlers (`tasks.go`, `handlers.go`)
- `pkg/logger/` — Structured logging helper (`logger.go`)
- `go.mod` — Go module definition
- `Dockerfile` — Docker image with Chromium for dynamic scraping

---

## 🛠️ Tech Stack

- **Language**: Go (1.25+)
- **Web Framework**: [Gin](https://github.com/gin-gonic/gin)
- **Static Scraper**: [Colly](https://github.com/gocolly/colly)
- **Dynamic Scraper**: [Chromedp](https://github.com/chromedp/chromedp)
- **Async Queue**: [Asynq](https://github.com/hibiken/asynq) with Redis
- **HTML -> Markdown**: [html-to-markdown/v2](https://github.com/JohannesKaufmann/html-to-markdown)
- **Config**: [Viper](https://github.com/spf13/viper)
- **User Agents**: [gofakeit](https://github.com/brianvoe/gofakeit)

---

## 🔗 Documentation

Detailed documentation can be found in the `go-scraper-backend/` directory:

- **[Overview & Index](go-scraper-backend/index.md)**: High-level goals and tech stack.
- **[API Specification](go-scraper-backend/api-spec.md)**: Request/Response formats for endpoints.
- **[Architecture Notes](go-scraper-backend/architecture.md)**: Deep dive into the system design.
- **[Actionable Todos](go-scraper-backend/todos.md)**: Current progress and upcoming tasks (Phases 1-3 completed, Phase 4 in progress).

---

## ✨ Roadmap

- **Phase 1: Setup & Static Scraping** ✅
  - Basic static scraping with Colly.
- **Phase 2: Dynamic Scraping** ✅
  - Chromedp integration for JS-rendered sites.
  - Dockerfile with Chromium support.
- **Phase 3: Async Jobs & Queues** ✅
  - Redis-backed job queue using Asynq.
  - Support for large-scale domain crawling.
- **Phase 4: Production Hardening** 🚧
  - API Key Authentication.
  - Rate limiting and enhanced observability.

---

## 🤝 Internal contributions

This repository is currently private. Internal contributions should follow the team's workflow — if you'd like to contribute, please contact the repository owner to get access and guidance.

Suggested guidelines for internal contributors:

- **Branching:** use `feature/<short-desc>` or `fix/<short-desc>` for branches.
- **Testing:** run `go test ./...` before opening a PR.
- **PRs:** open pull requests against the `main` branch with a short description and any relevant test or reproduction steps.
- **Code Style:** keep changes focused and avoid unrelated refactors in the same PR.

If you do not have access, open an issue or contact the maintainer to request contributor access.

---

## ⚖️ License

This project is currently unlicensed. See the repository for updates.
