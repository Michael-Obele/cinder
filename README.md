# Cinder 🔥

<!-- [![Go Version](https://img.shields.io/github/go-mod/go-version/standard-user/cinder)](https://golang.org) -->

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Status](https://img.shields.io/badge/Status-Beta-blue)](https://github.com/standard-user/cinder)

**Cinder** is a high-performance, self-hosted web scraping API built with Go. It turns any website into LLM-ready markdown, designed as a drop-in alternative to Firecrawl.

> **Why Cinder?** Heavily optimized for low-memory, serverless, and "hobby tier" environments by using intelligent browser process management and a unified "monolith" architecture.

---

## ✨ Features

- **⚡ Fast & Efficient**: Reuses a single Chrome process with lightweight tabs, avoiding the heavy startup cost of spawning browsers per request.
- **🏭 Monolith Mode**: Runs the API and Async Worker in a single binary/container. Perfect for services like Railway or Leapcell where you pay per active container.
- **🔄 Async Queues**: Redis-backed job queue (Asynq) for handling heavy scrape jobs without blocking HTTP clients.
- **🧠 LLM Ready**: Converts complex HTML/SPAs into clean, structured Markdown using `html-to-markdown/v2`.
- **🕵️ Evasion**: Automatic User-Agent rotation and un-detected headless flags.

---

## 🚀 Quickstart

### Prerequisites

- **Go 1.25+** (for local development)
- **Redis** (Required for `/crawl` endpoints, optional for simple `/scrape`)
- **Chromium** (Installed automatically in Docker or Linux systems)

### System Requirements

- **Memory**:
  - Minimum: 512MB (basic scraping only, no JS rendering)
  - Recommended: 2GB (comfortable for dynamic scraping + async queue)
  - Hobby Tier (4GB): Perfect for production use
- **CPU**: 1+ cores (single core works, multiple cores improve concurrency)
- **Disk**: 50MB (binary + dependencies)

### Local Installation & Running

```bash
# Clone
git clone https://github.com/Michael-Obele/cinder.git
cd cinder

# Install dependencies
go mod download

# Create .env (optional, uses defaults)
cat > .env << 'EOF'
PORT=8080
SERVER_MODE=debug
LOG_LEVEL=info
# REDIS_URL=redis://localhost:6379  # Optional, for async crawling
EOF

# Run (Monolith Mode)
go run ./cmd/api
```

Visit `http://localhost:8080` (returns 404, which is expected—API is at `/v1/scrape`, `/v1/crawl`, etc.)

### Quick Test

```bash
# Test synchronous scrape
curl -X POST http://localhost:8080/v1/scrape \
  -H "Content-Type: application/json" \
  -d '{"url": "https://example.com", "mode": "static"}'

# Should return markdown content in ~500ms
```

### Docker

```bash
# Build
docker build -t cinder .

# Run with environment variables
docker run -p 8080:8080 \
  -e PORT=8080 \
  -e SERVER_MODE=release \
  cinder

# With Redis for async crawling
docker run -p 8080:8080 \
  -e REDIS_URL=redis://host.docker.internal:6379 \
  cinder
```

### Deployment Guides

#### Railway

- Dockerfile support: ✅ Native
- Environment: Set `SERVER_MODE=release`
- Memory: Hobby Tier (512MB) recommended

#### Leapcell (Recommended for Hobby Projects)

- **Why**: 4GB RAM + Unlimited concurrent requests (pay per compute minutes)
- **Cost**: ~$5-15/month for moderate traffic
- **Setup**: Push Docker image, set env vars
- **Note**: Monolith Mode perfectly fits the resource constraints

#### Vercel

- Use as a serverless function (requires API refactor for edge runtime)
- Not recommended due to Chromium size (~400MB)

#### AWS Lambda

- Requires AWS Lambda Container Images
- Cold starts ~10-15s (browser startup)
- Reserve concurrency for faster starts

All endpoints are prefixed with `/v1/`.

### 1. Synchronous Scrape

**Best for**: Single pages, fast turnaround needed.

`POST /v1/scrape`

**Request:**

```bash
curl -X POST http://localhost:8080/v1/scrape \
  -H "Content-Type: application/json" \
  -d '{
    "url": "https://example.com",
    "mode": "smart"
  }'
```

**Parameters:**

- `url` (required): Valid HTTP(S) URL to scrape
- `mode` (optional): Scraping strategy
  - `smart` (default): Auto-detect static vs dynamic
  - `static`: Use Colly (fast, lightweight)
  - `dynamic`: Use Chromedp (handles JavaScript)

**Response (200 OK):**

```json
{
  "url": "https://example.com",
  "markdown": "# Example Domain\n\nThis domain is established to be used for examples...",
  "html": "<!DOCTYPE html>\n<html>\n...",
  "metadata": {
    "scraped_at": "2026-01-20T10:30:00Z",
    "engine": "chromedp"
  }
}
```

---

### 2. Async Crawl (Queue)

**Best for**: Large sites, depth crawling, fire-and-forget jobs.

`POST /v1/crawl`

**Request:**

```bash
curl -X POST http://localhost:8080/v1/crawl \
  -H "Content-Type: application/json" \
  -d '{
    "url": "https://example.com/blog",
    "render": false
  }'
```

**Parameters:**

- `url` (required): Root URL to start crawling
- `render` (optional): Force dynamic rendering (default: false)

**Response (202 Accepted):**

```json
{
  "id": "asynq:task:uuid-here",
  "url": "https://example.com/blog",
  "render": false
}
```

---

### 3. Get Crawl Status

**Check job progress and results.**

`GET /v1/crawl/:id`

**Request:**

```bash
curl http://localhost:8080/v1/crawl/asynq:task:uuid-here
```

**Response (200 OK):**

```json
{
  "id": "asynq:task:uuid-here",
  "queue": "default",
  "state": "completed",
  "max_retry": 3,
  "retried": 0,
  "payload": "{\"url\":\"https://example.com/blog\",\"render\":false}",
  "result": "{\"urls_scraped\": 15, ...}"
}
```

**States:** `pending`, `active`, `completed`, `failed`, `retry`

---

### 4. Search (Powered by Brave)

**Search the web and return results.**

`POST /v1/search`

**Requires:** `BRAVE_SEARCH_API_KEY` environment variable

**Request:**

```bash
curl -X POST http://localhost:8080/v1/search \
  -H "Content-Type: application/json" \
  -d '{"query": "golang web scraping"}'
```

---

## 📋 Scraping Modes Explained

| Mode        | Engine      | Speed       | JS Support   | Best For               |
| ----------- | ----------- | ----------- | ------------ | ---------------------- |
| **static**  | Colly       | ⚡⚡⚡ Fast | ❌ No        | Traditional HTML sites |
| **dynamic** | Chromedp    | ⚡ Slow     | ✅ Yes       | React, Vue, SPAs       |
| **smart**   | Auto-detect | ⚡⚡ Medium | ✅ Sometimes | Most sites (default)   |

**Smart Mode Algorithm:**

- Attempts static scrape first (~200ms)
- Falls back to dynamic if content is minimal or fails

---

## 🔧 Environment Variables

| Variable               | Default | Required      | Description                                                   |
| ---------------------- | ------- | ------------- | ------------------------------------------------------------- |
| `PORT`                 | `8080`  | No            | HTTP server port                                              |
| `SERVER_MODE`          | `debug` | No            | Server mode: `debug`, `release`, `test`                       |
| `LOG_LEVEL`            | `info`  | No            | Log level: `debug`, `info`, `warn`, `error`                   |
| `REDIS_URL`            | (none)  | Conditional\* | Redis connection URL (e.g., `redis://localhost:6379`)         |
| `REDIS_HOST`           | (none)  | Conditional\* | Redis host (alternative to `REDIS_URL`)                       |
| `REDIS_PORT`           | `6379`  | Conditional\* | Redis port                                                    |
| `REDIS_PASSWORD`       | (none)  | Conditional\* | Redis password                                                |
| `BRAVE_SEARCH_API_KEY` | (none)  | No            | API key for Brave Search endpoint                             |
| `DISABLE_WORKER`       | `false` | No            | Set to `true` to disable embedded worker (microservices mode) |

**Note:** \*Redis is required for `/v1/crawl` endpoints. Without it, they return **503 Service Unavailable**.

---

## 🏗️ Architecture

### System Design

Cinder employs a **Monolithic Architecture with Embedded Worker** pattern, optimized for serverless and hobby-tier deployments where minimizing resource usage and cold-start times is critical.

#### Core Components

```
┌─────────────────┐    ┌──────────────────┐    ┌─────────────────┐
│   HTTP API      │    │   Queue Worker   │    │   Scraper       │
│   (Gin Router)  │◄──►│   (Asynq)       │◄──►│   Service       │
│                 │    │                  │    │                 │
│ • /v1/scrape    │    │ • Task Processing│    │ • Mode Selection│
│ • /v1/crawl     │    │ • Retry Logic    │    │ • Caching       │
│ • /v1/search    │    │ • Result Storage │    │ • Result Format │
└─────────────────┘    └──────────────────┘    └─────────────────┘
         │                       │                       │
         └───────────────────────┼───────────────────────┘
                                 ▼
                    ┌──────────────────────┐
                    │   Browser Pool       │
                    │   (Chromedp)         │
                    │                      │
                    │ • Shared Allocator   │
                    │ • Tab Management     │
                    │ • Memory Optimization│
                    └──────────────────────┘
```

#### Request Processing Pipeline

**Synchronous Flow (`/v1/scrape`):**

```
Client Request → Gin Router → Scrape Handler → Scraper Service
    ↓               ↓            ↓              ↓
Validate URL → Select Mode → Check Cache → Execute Scrape
    ↓               ↓            ↓              ↓
Return JSON ← Format Result ← Store Cache ← Browser/Colly
```

**Asynchronous Flow (`/v1/crawl`):**

```
Client Request → Gin Router → Crawl Handler → Redis Queue
    ↓               ↓            ↓              ↓
Validate URL → Create Task → Enqueue Job → Return Job ID
    ↓               ↓            ↓              ↓
    └───────────────────────────────────────────┘
                        │
                        ▼
               Embedded Worker Process
                        ↓
               Task Processor → Scraper Service
                        ↓
               Result Storage → Client Polls Status
```

#### Browser Optimization Strategy

**Problem Solved:** Traditional scraping spawns a new Chrome process per request (~500ms startup + 300MB RAM), making it unsuitable for concurrent workloads.

**Cinder's Solution:**

- **Singleton Allocator**: One Chromium process per container instance
- **Tab Pooling**: Each scrape request creates a lightweight tab (`chromedp.NewContext`)
- **Memory Efficiency**: ~200-300MB total for browser + API server
- **Concurrency**: 10 concurrent tabs (configurable via `internal/worker/server.go`)

**Performance Impact:**

- **Latency**: ~200ms static, ~1-3s dynamic (vs 2-5s with process spawning)
- **Throughput**: 3-5 requests/second on 2GB instances
- **Resource Usage**: 70% less memory than traditional approaches

#### Scalability Considerations

**Horizontal Scaling:**

- **Stateless Design**: API instances can be scaled independently
- **Shared Redis**: Queue coordination across multiple workers
- **Load Balancing**: Standard HTTP load balancers work out-of-the-box

**Vertical Scaling:**

- **Memory**: 4GB recommended for production (handles browser + concurrent requests)
- **CPU**: 1-2 cores sufficient (I/O bound, not CPU bound)
- **Storage**: Minimal disk usage (logs + optional cache)

**Reliability Features:**

- **Graceful Degradation**: Falls back to static scraping if dynamic fails
- **Circuit Breaker**: Redis unavailability doesn't crash the API
- **Health Checks**: Browser process monitoring (planned for Phase 5)
- **Result Caching**: Redis-backed response caching reduces duplicate work

#### Design Decisions

**Why Monolith Mode?**

- **Serverless Optimization**: Single process minimizes cold-start overhead
- **Resource Efficiency**: No inter-service communication overhead
- **Hobby-Tier Friendly**: Fits within free tier limits (Leapcell 4GB RAM)
- **Simplicity**: Easier deployment and debugging

**Why Asynq over Custom Queue?**

- **Battle-Tested**: Production-ready Redis-backed queue
- **Observability**: Built-in metrics and monitoring
- **Reliability**: Automatic retries, dead letter queues, task scheduling
- **Ecosystem**: Active maintenance and community support

**Why Smart Mode Default?**

- **User-Friendly**: Works for most sites without configuration
- **Cost-Effective**: Tries fast static scraping first
- **Fallback Safety**: Gracefully degrades to dynamic rendering

See [plan/architecture.md](plan/architecture.md) for deeper technical details and design rationale.

---

## ⚡ Performance & Benchmarks

Typical latencies on a 2GB instance with hot browser:

| Operation                 | Time      | Notes                      |
| ------------------------- | --------- | -------------------------- |
| Static scrape (Colly)     | 200-500ms | Simple HTML parsing        |
| Dynamic scrape (Chromedp) | 1-3s      | With JS rendering          |
| Browser cold start        | ~1-2s     | One-time on app startup    |
| Queue job enqueue         | 5-10ms    | Redis write                |
| Queue job processing      | 1-5s      | Depends on site complexity |

**Throughput:**

- Concurrent requests: 10 (configurable in worker config)
- QPS (queries per second): ~3-5 on medium instances (site-dependent)

---

## 🐛 Troubleshooting

### Browser Crashes / Out of Memory

**Problem**: Container kills after ~1-2 hours

- **Cause**: Chrome memory leak after N pages
- **Solution**:
  - Increase container memory (switch to 2GB+ tier)
  - Reduce concurrent workers (lower `Concurrency` in `internal/worker/server.go`)
  - Enable browser restart after N requests (planned for Phase 5)

### No Redis = `/crawl` Returns 503

**Problem**: `POST /v1/crawl` returns Service Unavailable

- **Cause**: `REDIS_URL` not set or invalid
- **Solution**: Set `REDIS_URL=redis://localhost:6379` or equivalent
- **Workaround**: Use synchronous `/v1/scrape` instead

### Dynamic Scraping Returns Empty Content

**Problem**: Markdown is mostly empty for modern sites

- **Cause**: Site not fully hydrated before HTML capture
- **Solution**:
  - Try `mode=dynamic` explicitly
  - Increase page load timeout (future feature)
  - Check browser console logs: `LOG_LEVEL=debug`

### Slow Performance

**Problem**: Requests taking >5s

- **Cause**:
  1. Colly/Chromedp waiting for slow site
  2. Cold browser start (first request)
  3. Browser memory fragmentation
- **Solution**:
  1. Use `mode=static` for fast sites
  2. Warm up the browser: `curl http://localhost:8080/v1/scrape -d '{"url":"https://example.com","mode":"static"}'`
  3. Increase container memory

---

## �🗺️ Roadmap & Status

| Phase       | Goal                          | Status         |
| :---------- | :---------------------------- | :------------- |
| **Phase 1** | Static Scraping (Colly)       | ✅ Done        |
| **Phase 2** | Dynamic Scraping (Chromedp)   | ✅ Done        |
| **Phase 3** | Async Queue (Asynq + Redis)   | ✅ Done        |
| **Phase 4** | Performance Tuning (Monolith) | ✅ Done        |
| **Phase 5** | Hardening & Testing           | 🚧 In Progress |

**Current Focus**:

- Adding a comprehensive Unit & Integration Test Suite (Currently 0% coverage).
- Implementing "Smart Wait" heuristics for slower SPAs.
- Adding a "Browser Health Check" to kill/restart Chrome after N scrapes.

---

## 🤝 Contributing

Contributions are welcome! This project is in **active development** and priorities are:

1. **Unit & Integration Tests** (Currently 0% coverage)
   - `internal/domain/scraper_test.go`
   - `internal/api/handlers/scrape_test.go`
   - `internal/scraper/chromedp_test.go`

2. **Smart Waiting Strategies** for SPAs
   - Network idle detection
   - Configurable wait conditions
   - Better heuristics for "page ready"

3. **Browser Health Check**
   - Restart browser after N requests to prevent memory leaks
   - Automatic OOM recovery

**How to Contribute:**

1. Fork the Project
2. Create your Feature Branch (`git checkout -b feature/amazing-feature`)
3. Add tests for your changes
4. Commit your Changes (`git commit -m 'Add amazing feature'`)
5. Push to the Branch (`git push origin feature/amazing-feature`)
6. Open a Pull Request

**Code Standards:**

- Use `go fmt` for formatting
- Add structured logging via `pkg/logger`
- Include error handling (avoid silent failures)
- Test your code locally: `go test ./...`

---

## ⚖️ License

Distributed under the MIT License. See `LICENSE` for more information.
