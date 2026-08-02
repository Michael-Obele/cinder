# Cinder 🔥

<!-- [![Go Version](https://img.shields.io/github/go-mod/go-version/standard-user/cinder)](https://golang.org) -->

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Status](https://img.shields.io/badge/Status-Beta-blue)](https://github.com/standard-user/cinder)

**Cinder** is a high-performance, self-hosted web scraping API built with Go. It turns any website into LLM-ready markdown, designed as a drop-in alternative to Firecrawl.

> **Why Cinder?** Heavily optimized for low-memory, serverless, and "hobby tier" environments by using intelligent browser process management and a unified "monolith" architecture. Deploys cleanly on Fly.io, Railway, Leapcell, or any Docker host.

---

## ✨ Features

- **⚡ Fast & Efficient**: Reuses a single Chrome process with lightweight tabs, avoiding the heavy startup cost of spawning browsers per request. Parallel image blob fetching and parallel crawl workers.
- **🏭 Monolith Mode**: Runs the API and Async Worker in a single binary/container. Perfect for services like Railway or Leapcell where you pay per active container.
- **🔄 Async Queues**: Redis-backed job queue (Asynq) for handling heavy scrape jobs without blocking HTTP clients.
- **🧠 LLM Ready**: Converts complex HTML/SPAs into clean, structured Markdown using `html-to-markdown/v2` + readability main-content extraction.
- **🕵️ Evasion**: Automatic User-Agent rotation and un-detected headless flags.
- **🗺️ URL Discovery**: `/v1/map` finds site URLs via `robots.txt`/`sitemap.xml` with link-discovery fallback.
- **🖼️ Image Engine v2**: srcset/lazy-load/`<picture>` extraction, quality-ranked selection (og > hero > content > avatar), dimension sniffing, and optional resize/re-encode.
- **📡 Signed Webhooks**: Crawl completion and monitor change notifications with HMAC-SHA256 signatures.
- **📊 Change Tracking**: `/v1/monitor` schedules content-hash checks and alerts on change.
- **📦 Batch Scrape**: `/v1/batch/scrape` enqueues up to 20 URLs with aggregated status.
- **🎬 Page Actions**: `wait_ms`, `wait_selector`, `click`, `scroll_down`, `scroll_to_bottom` before capture.
- **🧹 Cleaner Output**: Deterministic schema extraction, extractive summaries, PII redaction, ad blocking, and base64-image removal — all LLM-free.
- **🔐 Auth & Rate Limiting**: Optional `X-API-Key` auth and per-client rate limiting.

---

## 🚀 Quickstart

### Prerequisites

- **Go 1.25+** (for local development)
- **Redis** (Required for `/crawl`, `/batch`, and `/monitor` endpoints, optional for simple `/scrape`)
- **Chromium** (Installed automatically in Docker or Linux systems)

### System Requirements

- **Memory**:
  - Minimum: 512MB (basic scraping only, no JS rendering)
  - Recommended: 1-2GB (comfortable for dynamic scraping + async queue)
  - Hobby Tier (4GB): Perfect for production use
- **CPU**: 1+ cores (single core works, multiple cores improve concurrency)
- **Disk**: 50MB (binary + dependencies)

> **Fly.io default**: the included `fly.toml` runs on a 1GB shared-CPU machine with 1GB swap — plenty for moderate dynamic workloads thanks to browser recycling (`CHROME_RECYCLE_AFTER`). Cinder runs anywhere Docker does: Fly.io, Railway, Leapcell, VPS, or bare metal.

### Local Installation & Running

```bash
# Clone
git clone https://github.com/Michael-Obele/cinder.git
cd cinder

# Install dependencies
go mod download

# Create .env (optional, uses defaults)
cat > .env << 'EOF'
SERVER_PORT=8080
SERVER_MODE=debug
LOG_LEVEL=info
# REDIS_URL=redis://localhost:6379  # Optional, for /crawl, /batch, /monitor
# API_KEYS=sk_a,sk_b               # Optional; when set, /v1/* requires X-API-Key
EOF

# Run (Monolith Mode)
go run ./cmd/api
```

Visit `http://localhost:8080` — the root returns a JSON service overview listing every endpoint under `/v1`.

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
  -e SERVER_PORT=8080 \
  -e SERVER_MODE=release \
  cinder

# With Redis for async crawling
docker run -p 8080:8080 \
  -e REDIS_URL=redis://host.docker.internal:6379 \
  cinder
```

### Deployment Guides

#### Fly.io (Recommended)

- **Why**: cheap autoscaling Machines, global regions, and `auto_stop`/`auto_start` (already in the included `fly.toml`) keep cost near zero when idle
- **Setup**: `fly launch`, then `fly deploy` — the included `fly.toml` runs a 1GB shared-CPU machine with 1GB swap
- **Config**: set `SERVER_MODE=release`; add `REDIS_URL` (e.g. Upstash) and `BRAVE_SEARCH_API_KEY` as secrets
- **Memory**: 1GB is fine for moderate dynamic scraping thanks to browser recycling

```bash
fly secrets set REDIS_URL=rediss://... BRAVE_SEARCH_API_KEY=...
fly deploy
```

#### Railway

- Dockerfile support: ✅ Native
- Environment: Set `SERVER_MODE=release`
- Memory: Hobby Tier (512MB) recommended

#### Leapcell (Great for Hobby Projects)

- **Why**: 4GB RAM + pay-per-compute-minute billing
- **Cost**: ~$5-15/month for moderate traffic
- **Setup**: Push Docker image, set env vars
- **Note**: Monolith Mode perfectly fits the resource constraints

#### Any Docker Host (VPS, ECS, etc.)

Cinder is a single static binary in a Docker image — it runs anywhere containers run. No platform-specific APIs are used; Redis is the only external dependency, and only for queue/batch/monitor features.

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
- and more: `images`, `image_format`, `max_images`, `screenshot_opts`, `actions`, `extract_schema`, `summary`, `redact_pii`, `block_ads` — full parameter table in `docs/guides/API_REFERENCE.md`

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
- `render` (optional): Force dynamic rendering (default: `false`)
- `maxDepth` (optional): Link depth to follow (default `2`, max `10`)
- `limit` (optional): Max pages to scrape (default `10`, max `100`)
- `include_paths` / `exclude_paths` (optional): Glob patterns controlling which paths are followed (exclusion wins)
- `webhook_url` / `webhook_secret` (optional): POST the result on completion, signed with `X-Cinder-Signature` (HMAC-SHA256)

> Crawls run **in parallel** (env `CRAWL_CONCURRENCY`) with per-domain politeness, retries with backoff, and never retry 4xx errors.

**Response (202 Accepted):**

```json
{
  "id": "asynq:task:uuid-here",
  "url": "https://example.com/blog",
  "render": false,
  "screenshot": false,
  "images": false,
  "maxDepth": 2,
  "limit": 10
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
  "crawl": {
    "status": "completed",
    "total_pages": 15,
    "max_depth": 2,
    "limit": 10,
    "pages": [
      {
        "url": "https://example.com/blog/post-1",
        "title": "Post 1",
        "preview": "First 300 characters of the page markdown..."
      }
    ]
  },
  "failed_urls": []
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

### 5. Map a Website (URL Discovery)

Discover a site's URLs from `robots.txt`/`sitemap.xml` (falling back to one-level link discovery) without scraping content. No Redis required.

`POST /v1/map`

```bash
curl -X POST http://localhost:8080/v1/map \
  -H "Content-Type: application/json" \
  -d '{"url": "https://example.com", "search": "/docs", "limit": 200}'
```

**Parameters:** `url` (required), `search` (optional substring filter), `limit` (default `100`, max `5000`).

**Response:**

```json
{
  "url": "https://example.com",
  "count": 2,
  "links": [
    { "url": "https://example.com/docs/intro", "source": "sitemap" },
    { "url": "https://example.com/docs/api", "source": "link" }
  ]
}
```

---

### 6. Batch Scrape

Enqueue up to 20 URLs at once as individual queue jobs under one batch ID. Requires Redis.

`POST /v1/batch/scrape` → returns `batch_id` + per-task IDs · `GET /v1/batch/:id` → aggregated status

```bash
curl -X POST http://localhost:8080/v1/batch/scrape \
  -H "Content-Type: application/json" \
  -d '{"urls": ["https://a.example.com", "https://b.example.com"]}'
```

---

### 7. Change-Tracking Monitor

Scrape a URL on a schedule, hash the markdown, and fire a signed webhook when content changes. The first check records the baseline without notifying. Requires Redis.

`POST /v1/monitor` → create · `GET /v1/monitor/:id` → status · `DELETE /v1/monitor/:id` → stop

```bash
curl -X POST http://localhost:8080/v1/monitor \
  -H "Content-Type: application/json" \
  -d '{
    "url": "https://pricing.example.com",
    "interval_seconds": 3600,
    "webhook_url": "https://myapp.example.com/hooks/cinder",
    "webhook_secret": "s3cret"
  }'
```

Minimum `interval_seconds`: 3600 (1 hour).

---

### 8. Authentication & Rate Limiting

Optional and env-configured — with no keys set, the API stays open. When `API_KEYS` is set, every `/v1/*` request must send the `X-API-Key` header.

```bash
# env: API_KEYS=sk_a,sk_b  RATE_LIMIT_RPM=60
curl -X POST http://localhost:8080/v1/scrape \
  -H "Content-Type: application/json" \
  -H "X-API-Key: sk_a" \
  -d '{"url": "https://example.com"}'
```

Exceeding the rate limit returns `429` with a `retry_after` hint. Redis-backed limiting when `REDIS_URL` is set; in-memory fallback otherwise.

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

| Variable               | Default | Required      | Description                                                          |
| ---------------------- | ------- | ------------- | -------------------------------------------------------------------- |
| `SERVER_PORT`          | `8080`  | No            | HTTP server port                                                     |
| `SERVER_MODE`          | `debug` | No            | Server mode: `debug`, `release`, `test`                              |
| `LOG_LEVEL`            | `info`  | No            | Log level: `debug`, `info`, `warn`, `error`                          |
| `REDIS_URL`            | (none)  | Conditional\* | Redis connection URL (e.g., `redis://localhost:6379`)                |
| `REDIS_HOST`           | (none)  | Conditional\* | Redis host (alternative to `REDIS_URL`)                              |
| `REDIS_PORT`           | `6379`  | Conditional\* | Redis port                                                           |
| `REDIS_PASSWORD`       | (none)  | Conditional\* | Redis password                                                       |
| `BRAVE_SEARCH_API_KEY` | (none)  | No            | API key for Brave Search endpoint                                    |
| `DISABLE_WORKER`       | `false` | No            | Set to `true` to disable embedded worker (microservices mode)        |
| `CHROME_RECYCLE_AFTER` | `100`   | No            | Restart the Chrome allocator after N scrapes to bound memory growth  |
| `CRAWL_CONCURRENCY`    | `4`     | No            | Parallel crawl workers (1-10)                                        |
| `CRAWL_DOMAIN_DELAY`   | `1`     | No            | Minimum seconds between requests to the same host during a crawl     |
| `WEBHOOK_TIMEOUT`      | `10`    | No            | Webhook delivery timeout in seconds                                  |
| `API_KEYS`             | (none)  | No            | Comma-separated keys; enables `X-API-Key` auth on `/v1/*`            |
| `RATE_LIMIT_RPM`       | `0`     | No            | Per-client requests/minute (0 = unlimited)                           |

**Note:** \*Redis is required for `/v1/crawl`, `/v1/batch/*`, and `/v1/monitor*` endpoints. Without it, they return **503 Service Unavailable**.

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
- **Concurrency**: 5 queue workers (configurable in `internal/worker/server.go`) + a parallel crawl pool (`CRAWL_CONCURRENCY`, default 4)

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
- **Browser Recycling**: Chrome allocator restarts after `CHROME_RECYCLE_AFTER` scrapes (default 100) to bound memory growth
- **Result Caching**: Redis-backed, option-aware response caching reduces duplicate work

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
  - Lower `CRAWL_CONCURRENCY` (fewer parallel tabs)
  - Set `CHROME_RECYCLE_AFTER=50` to restart the browser more frequently (default: 100 scrapes)

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
  - Add page actions to wait for content: `"actions": [{"type": "wait_selector", "selector": "#app"}]`
  - Scroll lazy-loaded content: `"actions": [{"type": "scroll_to_bottom"}]`
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

| Phase       | Goal                                                                  | Status      |
| :---------- | :-------------------------------------------------------------------- | :---------- |
| **Phase 1** | Static Scraping (Colly)                                               | ✅ Done     |
| **Phase 2** | Dynamic Scraping (Chromedp)                                           | ✅ Done     |
| **Phase 3** | Async Queue (Asynq + Redis)                                           | ✅ Done     |
| **Phase 4** | Polish & Auth (API keys, rate limiting)                               | ✅ Done     |
| **Phase 5** | High Performance & Reliability (browser recycling, parallel crawl)    | ✅ Done     |
| **Phase 6** | Cinder v2 (images v2, map, batch, actions, monitors, extraction)      | ✅ Done     |

**Current Focus**:

- **Stealth tier**: TLS fingerprint spoofing (`refraction-networking/utls`) + CDP stealth-script injection for Cloudflare/DataDome-protected sites.
- **PDF & documents**: parse PDFs and other non-HTML assets in `/v1/scrape` (Firecrawl parity).
- **Benchmarks**: `pprof` profiles + benchmark suite for scraper and image-processing hot paths.
- **Heuristic "smart wait"**: network-idle detection and configurable readiness conditions beyond the current selector/scroll actions.

---

## 🤝 Contributing

Contributions are welcome! This project is in **active development** and priorities are:

1. **Stealth & anti-bot** — TLS fingerprint spoofing (`utls`) and CDP stealth-script injection for Cloudflare/DataDome-protected sites.
2. **PDF & documents** — parse PDFs (and other non-HTML assets) in `/v1/scrape`.
3. **Benchmarks** — `pprof` + benchmark suite for the scraper and image-processing hot paths.
4. **Heuristic "smart wait"** — network-idle detection and configurable page-readiness conditions.

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
- Test your code locally: `make check` (gofmt, vet, staticcheck, tests with `-race`)

---

## ⚖️ License

Distributed under the MIT License. See `LICENSE` for more information.
