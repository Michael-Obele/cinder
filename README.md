# Cinder 🔥

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Go](https://img.shields.io/badge/Go-1.25%2B-00ADD8?logo=go&logoColor=white)](https://go.dev)
[![Docker](https://img.shields.io/badge/docker-ready-2496ED?logo=docker&logoColor=white)](https://www.docker.com/)
[![Status](https://img.shields.io/badge/Status-Beta-blue)](https://github.com/Michael-Obele/cinder)

**Turn any website into LLM-ready markdown — self-hosted, one binary, runs on a $5/mo hobby box.** Cinder is a drop-in, open-source alternative to Firecrawl, written in Go.

> One process. Low memory. No per-request browser spawning. Deploys to Fly.io, Railway, Leapcell, or any Docker host in minutes.

---

## 🚀 Quick start

**Prerequisites:** Docker (or Go 1.25+ and Chromium). Redis is only needed for `/crawl`, `/batch`, and `/monitor`.

**Option A — Docker (fastest):**

```bash
docker run -p 8080:8080 -e SERVER_MODE=release cinder
```

**Option B — From source:**

```bash
git clone https://github.com/Michael-Obele/cinder.git
cd cinder
go run ./cmd/api
```

**Your first scrape:**

```bash
curl -X POST http://localhost:8080/v1/scrape \
  -H "Content-Type: application/json" \
  -d '{"url": "https://example.com", "mode": "smart"}'
```

Returns clean markdown in ~200ms (static) to ~1–3s (dynamic):

```json
{
  "url": "https://example.com",
  "markdown": "# Example Domain\n\nThis domain is established to be used for examples...",
  "metadata": { "scraped_at": "2026-08-02T10:30:00Z", "engine": "chromedp" }
}
```

> The root `GET /` returns a JSON overview of every `/v1` endpoint.

---

## 🚢 Deploy

**Fly.io (recommended):** cheap autoscaling Machines with `auto_stop`/`auto_start` keep cost near zero when idle. The included `fly.toml` runs a 1GB shared-CPU machine — plenty for moderate dynamic scraping thanks to browser recycling.

```bash
fly launch          # uses the included fly.toml
fly secrets set REDIS_URL=rediss://... BRAVE_SEARCH_API_KEY=...
fly deploy
```

- **Railway** — native Dockerfile; set `SERVER_MODE=release` (512MB hobby tier works).
- **Leapcell** — 4GB RAM, pay-per-compute-minute; ~$5–15/mo for moderate traffic.
- **Any Docker host** — single static binary; Redis is the only external dependency (and only for queue/batch/monitor).
- **Vercel / AWS Lambda** — possible but not recommended (Chromium ~400MB; Lambda cold starts 10–15s).

---

## ✨ Features

| Feature                 | What it gives you                                                                                                        |
| ----------------------- | ------------------------------------------------------------------------------------------------------------------------ |
| ⚡ **Fast & efficient** | Reuses one Chrome process with lightweight tabs — no ~500ms spawn per request. Parallel image fetching + parallel crawl. |
| 🏭 **Monolith mode**    | API + async worker in a single binary. Pay per container, not per service.                                               |
| 🔄 **Async queues**     | Redis-backed (Asynq) for heavy crawl jobs without blocking clients.                                                      |
| 🧠 **LLM-ready**        | Clean markdown via `html-to-markdown/v2` + readability main-content extraction.                                          |
| 🕵️ **Evasion**          | Auto user-agent rotation + undetected headless flags.                                                                    |
| 🗺️ **URL discovery**    | `/v1/map` from `robots.txt`/`sitemap.xml` with link fallback.                                                            |
| 🖼️ **Image engine v2**  | srcset/lazy-load/`<picture>` extraction, quality-ranked, dimension sniffing, optional resize.                            |
| 📡 **Signed webhooks**  | HMAC-SHA256 crawl & monitor notifications.                                                                               |
| 📊 **Change tracking**  | `/v1/monitor` hashes markdown and alerts on change.                                                                      |
| 📦 **Batch scrape**     | Enqueue up to 20 URLs, aggregated status.                                                                                |
| 🎬 **Page actions**     | `wait_ms`, `wait_selector`, `click`, `scroll_down`, `scroll_to_bottom`.                                                  |
| 🧹 **Cleaner output**   | Schema extraction, summaries, PII redaction, ad blocking — all LLM-free.                                                 |
| 🔐 **Auth & limits**    | Optional `X-API-Key` auth + per-client rate limiting.                                                                    |

---

## 📋 Scraping modes

| Mode              | Engine   | Speed  | JS           | Best for         |
| ----------------- | -------- | ------ | ------------ | ---------------- |
| `static`          | Colly    | ⚡⚡⚡ | ❌           | Traditional HTML |
| `dynamic`         | Chromedp | ⚡     | ✅           | React/Vue/SPAs   |
| `smart` (default) | Auto     | ⚡⚡   | ✅ sometimes | Most sites       |

**Smart mode** tries static first (~200ms), then falls back to dynamic if content is thin or fails.

---

## 🔌 API endpoints

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

## ⚙️ Configuration

| Variable               | Default | Purpose                                                     |
| ---------------------- | ------- | ----------------------------------------------------------- |
| `SERVER_PORT`          | `8080`  | HTTP port                                                   |
| `SERVER_MODE`          | `debug` | `debug` / `release` / `test`                                |
| `LOG_LEVEL`            | `info`  | `debug` / `info` / `warn` / `error`                         |
| `REDIS_URL`            | (none)  | Redis URL — **required** for `/crawl`, `/batch`, `/monitor` |
| `BRAVE_SEARCH_API_KEY` | (none)  | Enables `/v1/search` (last-resort backend)                  |
| `SEARXNG_ENDPOINT`     | (none)  | Self-hosted SearXNG base URL (e.g. `http://localhost:8889`) — primary `/v1/search` backend, free, aggregates many engines |
| `DISABLE_WORKER`       | `false` | Set `true` to run API without the embedded worker           |
| `CHROME_RECYCLE_AFTER` | `100`   | Restart Chrome allocator after N scrapes (bounds memory)    |
| `CRAWL_CONCURRENCY`    | `4`     | Parallel crawl workers (1–10)                               |
| `CRAWL_DOMAIN_DELAY`   | `1`     | Min seconds between requests to the same host               |
| `CRAWL_TIMEOUT`        | `30`    | Overall crawl deadline (minutes; crawl returns `"timeout"`) |
| `CRAWL_SCRAPE_TIMEOUT` | `30`    | Per-attempt scrape deadline inside a crawl (s)              |
| `CRAWL_MAX_RETRIES`    | `2`     | Retries per URL (0–5; 4xx never retried)                    |
| `WEBHOOK_TIMEOUT`      | `10`    | Webhook delivery timeout (s)                                |
| `API_KEYS`             | (none)  | Comma-separated keys; enables `X-API-Key` auth              |
| `RATE_LIMIT_RPM`       | `0`     | Per-client requests/min (0 = unlimited)                     |
| `SSRF_ALLOW_PRIVATE`   | `false` | Set `true` to allow fetching private/loopback addresses     |
| `SHUTDOWN_TIMEOUT`     | `20`    | Seconds to drain in-flight requests and tasks on SIGTERM    |

> Redis alternatives `REDIS_HOST` / `REDIS_PORT` / `REDIS_PASSWORD` also supported. Without Redis, queue endpoints return `503`.

> **SSRF:** every outbound fetch (scrape, crawl, sitemap, image, webhook) refuses non-public destinations — loopback, RFC1918, link-local (including the `169.254.169.254` cloud metadata endpoint), CGNAT, and multicast. The check runs at dial time, so redirects and DNS rebinding are covered too. Set `SSRF_ALLOW_PRIVATE=true` only on an instance that is not publicly reachable, e.g. to scrape an internal wiki or a local dev server.

---

## 🏗️ Architecture

Cinder is a **monolith with an embedded worker** — one binary runs the Gin API and the Asynq worker, optimized for serverless and hobby-tier deployments.

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

- **Singleton allocator** — one Chromium, lightweight tabs (`chromedp.NewContext`). ~200–300MB total.
- **Smart mode** — static first, dynamic fallback.
- **Reliability** — graceful degradation, Redis circuit breaker, browser recycling, option-aware result caching.

See [`plan/architecture.md`](plan/architecture.md) for full design rationale.

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

## 🗺️ Roadmap

| Phase | Goal                                                             | Status  |
| ----- | ---------------------------------------------------------------- | ------- |
| 1–5   | Static, dynamic, async queue, auth, performance                  | ✅ Done |
| 6     | Cinder v2 (images v2, map, batch, actions, monitors, extraction) | ✅ Done |

**Next:** stealth tier (`utls` + CDP stealth), PDF/non-HTML parsing, `pprof` benchmarks, heuristic smart-wait.

---

## 🤝 Contributing

Active development — priorities: stealth/anti-bot, PDF & documents, benchmarks, smart-wait.

1. Fork → `git checkout -b feature/...` → add tests → `git commit` → push → open PR.
2. Run `make check` (gofmt, vet, staticcheck, `-race` tests) before submitting.
3. Use `pkg/logger` for logging; handle all errors explicitly.

---

## ⚖️ License

MIT — see [`LICENSE`](LICENSE).

---

[![Star History Chart](https://api.star-history.com/svg?repos=Michael-Obele/cinder&type=Date)](https://star-history.com/#Michael-Obele/cinder&Date)
