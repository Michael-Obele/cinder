# Cinder 🔥

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Go](https://img.shields.io/badge/Go-1.25%2B-00ADD8?logo=go&logoColor=white)](https://go.dev)
[![Docker](https://img.shields.io/badge/docker-ready-2496ED?logo=docker&logoColor=white)](https://www.docker.com/)
[![Status](https://img.shields.io/badge/Status-Beta-blue)](https://github.com/Michael-Obele/cinder)

**Turn any website into LLM-ready markdown — self-hosted, one binary, runs on a $5/mo hobby box.** Drop-in, open-source Firecrawl alternative in Go. No per-request browser spawn. No per-token bill.

> One process. ~200ms static, 1–3s dynamic. `fly deploy` in minutes. Redis only for async.

---

## Prerequisites

| Requirement | Version | Notes |
|---|---|---|
| **Go** | 1.25+ | Only for `go run` from source — Docker needs nothing |
| **Chromium** | Any recent | Auto-installed in Docker; local `go run` needs `chromium` or `google-chrome` on `PATH` |
| **Docker** (recommended) | 20.10+ | Handles Go + Chromium + Redis + SearXNG in one command |
| **Redis** | 7+ | Only for `/v1/crawl`, `/v1/batch`, `/v1/monitor` — sync scrape/search/map work without it |

---

## Try it in 60 seconds

### Option A — Full stack with Docker Compose (recommended)

Spin up Cinder + Redis + SearXNG (self-hosted search) together:

```bash
git clone https://github.com/Michael-Obele/cinder.git && cd cinder
docker compose up -d          # builds api, starts redis + searxng sidecars
curl http://localhost:8080/health   # → {"status":"ok","service":"cinder"}
```

`docker compose` is the fastest path to *every* feature — crawl, batch, monitor, and search all work out of the box. SearXNG is exposed on `http://localhost:8889`.

### Option B — Single container

```bash
docker build -t cinder . && docker run --rm -p 8080:8080 -e SERVER_MODE=release cinder
# or, once published:
# docker run --rm -p 8080:8080 -e SERVER_MODE=release ghcr.io/michael-obele/cinder
```

> The published image `ghcr.io/michael-obele/cinder` is rebuilt on every push to `main`. If you get `denied` or `not found`, build locally with the line above — same Dockerfile, same binary.

### Option C — From source

```bash
git clone https://github.com/Michael-Obele/cinder.git && cd cinder
go run ./cmd/api              # needs Go 1.25+ and Chromium on PATH
# Redis only for /v1/crawl, /v1/batch, /v1/monitor — scrape/map/search work without it
# With Redis: REDIS_URL=redis://localhost:6379 go run ./cmd/api
# With SearXNG: SEARXNG_ENDPOINT=http://localhost:8889 go run ./cmd/api
```

### Your first scrape

```bash
curl -X POST http://localhost:8080/v1/scrape \
  -H "Content-Type: application/json" \
  -d '{"url": "https://example.com"}'
```

```json
{
  "url": "https://example.com",
  "markdown": "# Example Domain\n\nThis domain is for use in documentation examples...",
  "metadata": { "title": "Example Domain", "description": "Example Domain" }
}
```

No `mode` needed — `smart` is the default. Explicit `mode: "static"` skips JS; `mode: "dynamic"` always renders with Chromedp.

**Verify everything is wired:**

```bash
curl http://localhost:8080/          # lists every /v1 endpoint
curl http://localhost:8080/health    # unauthenticated liveness probe (also at /v1/ping)
curl http://localhost:8889/search?q=cinder&format=json  # SearXNG is up (compose only)
```

---

## Why Cinder (not just another scraper)

**If you're paying Firecrawl/Exa by the token or spawning a Playwright per request, you're overpaying.**

| What hurts with hosted APIs | What Cinder gives you instead | Outcome |
|---|---|---|
| **$0.01–0.10 per scrape** + rate limits | **$0 self-hosted** — one binary, hobby-tier RAM | Ship RAG without a cloud bill |
| **500ms Chrome spawn per request** | **One shared allocator + lightweight tabs** | ~200ms static, parallel image + crawl pools |
| **JS SPAs return empty HTML** | **Smart mode** — static first, fallback to Chromedp on thin shells | Works on React/Vue without you guessing the mode |
| **Noisy HTML (nav/ads/footer)** | **Readability main-content** + ad block before `html-to-markdown` | Clean markdown your LLM actually wants |
| **Crawl needs a separate worker fleet** | **Monolith** — Gin + Asynq in one process | Pay per container, not per service |

**Proof, not promises:** Self-hosted SearXNG search benchmark (`scripts/search-bench.py`, 10 workers/30s) — **Cinder 560 req/s, p50 11ms** vs Firecrawl self-hosted **1.9 req/s, p50 5.4s** — same $0 cost, ~300× throughput. See [`docs/SEARCH_COMPARISON.md`](docs/SEARCH_COMPARISON.md). Full feature parity in [`docs/EXA_PARITY.md`](docs/EXA_PARITY.md).

---

## How it works — 3 steps

1. **Scrape** `POST /v1/scrape` with `mode: smart` (default). Smart runs heuristics on static HTML; if it looks like an SPA shell (`#__next`, `data-reactroot`, tiny HTML + scripts) it retries dynamically. No mode guessing.
2. **Discover** `POST /v1/map` (sitemap → `robots.txt` → link fallback, no Redis) or **Search** `POST /v1/search` (SearXNG aggregated, Brave fallback, Redis-cached).
3. **Scale** `POST /v1/crawl` (async BFS, 202 + job ID) → poll `GET /v1/crawl/:id`. Batch 20 URLs, monitor changes, fire signed webhooks. All without leaving the binary.

```
Client → Gin Router → Scraper Service (Colly / Chromedp + Readability)
                 ↘ Asynq (Redis) → Embedded Worker → same Scraper Service
                                         ↘ Browser Pool (one allocator, recycled every 100 scrapes)
```

---

## Scraping modes

| Mode | Engine | Speed | JS | Use when |
|---|---|---|---|---|
| `static` | Colly | ⚡⚡⚡ 200–500ms | ❌ | Classic HTML, blogs, docs |
| `dynamic` | Chromedp | ⚡ 1–3s | ✅ | React/Vue/SPAs, lazy-loaded |
| `smart` (default) | Auto | ⚡⚡ | ✅ when needed | You don't want to think about it |

---

## API — all under `/v1`

### 1. Scrape (sync)
`POST /v1/scrape` — fastest path for one page. Also accepts `GET /v1/scrape?url=...` for quick probes. **Sync multi-URL:** `POST /v1/scrape` with `{"urls": ["https://a.com","https://b.com"]}` (max 10, exclusive with `url`) → `{results: [{url, markdown, metadata, ...}]}` in one call — no Redis, errgroup limit 5, mirrors `web_fetch_exa` and Firecrawl `POST /v2/batch/scrape` (see https://docs.firecrawl.dev/features/scrape + https://docs.firecrawl.dev/api-reference/endpoint/scrape).

**Key params:** `url` (required unless `urls` provided), `urls` (max 10, sync batch — exclusive with `url`), `mode` (`smart`/`static`/`dynamic`), `images` + `image_format` (`url`|`blob`), `max_images`, `screenshot` + `screenshot_opts` (`width`/`height`/`full_page`/`format`/`quality`/`wait_selector`), `actions` (`wait_ms`, `wait_selector`, `click`, `scroll_down`, `scroll_to_bottom`), `extract_schema` (CSS selector → `{selector, attr, multiple}`), `summary` + `summary_sentences`, `redact_pii`, `block_ads` (default true), `remove_base64_images` (default true), `include_links` (default true — `links: [{url, text, isInternal}]` after readability, like Firecrawl `formats: ["links"]`). Full table: [`docs/guides/API_REFERENCE.md`](docs/guides/API_REFERENCE.md).

```bash
curl -X POST http://localhost:8080/v1/scrape \
  -H "Content-Type: application/json" \
  -d '{
    "url": "https://example.com",
    "images": true,
    "max_images": 5,
    "summary": true,
    "extract_schema": {
      "title": {"selector": "h1"},
      "links": {"selector": "a", "attr": "href", "multiple": true}
    }
  }'

# Sync multi-URL (up to 10, parallel limit 5, same params applied to each URL):
curl -X POST http://localhost:8080/v1/scrape \
  -H "Content-Type: application/json" \
  -d '{"urls": ["https://example.com","https://example.org"], "mode":"smart"}'
# → {"results": [{"url":"https://example.com","markdown":"...","metadata":{...}}, ...]}
```

### 2. Crawl (async)
`POST /v1/crawl` → `202 {id, url, maxDepth, limit}` · `GET /v1/crawl/:id` → `{state: pending|active|completed|failed, pages[], failed_urls[]}`

Params: `maxDepth` (1–10, default 2), `limit` (1–100, default 10), `include_paths`/`exclude_paths` globs (exclusion wins), `webhook_url` + `webhook_secret` (HMAC `X-Cinder-Signature`), `render`/`screenshot`/`images`. Parallel pool `CRAWL_CONCURRENCY=4` (cap 10), 1s politeness, 2 retries for 5xx (4xx never retried). **Requires `REDIS_URL`.**

### 3. Search (SearXNG, cached) — highlights + category + rerank
`POST /v1/search` — needs `SEARXNG_ENDPOINT=http://localhost:8889` (Docker: `searxng:8080`) or `BRAVE_SEARCH_API_KEY` as fallback. Repeat queries hit Redis cache (~2ms). Returns `highlights: ["…query window…"]` per result (120-char) + `relevance` reranked. Filters: `includeDomains`, `excludeDomains`, `requiredText`, `maxAge` (1/7/30d), `category` (`general`/`news`/`code`), `rerank` (`true` → TF-IDF pure Go), `limit`/`offset`. `GET /v1/search?query=...&category=news&rerank=true` works too.

```bash
curl -X POST http://localhost:8080/v1/search \
  -H "Content-Type: application/json" \
  -d '{"query": "cinder web scraper", "limit": 5}'
```

### 4. Map (no Redis)
`POST /v1/map` → `{count, links: [{url, source: sitemap|link}]}` — `search` substring filter, `limit` 100 (max 5000).

### 5. Batch
`POST /v1/batch/scrape` `{urls: []}` (max 20) → `{batch_id, tasks}` · `GET /v1/batch/:id` aggregated. **Requires `REDIS_URL`.**

### 6. Monitor (change tracking)
`POST /v1/monitor` `{url, interval_seconds >=3600, webhook_url, webhook_secret}` → baseline hash stored, webhook on change → `GET`/`DELETE /v1/monitor/:id`. Markdown SHA-256, Redis + Asynq scheduler. **Requires `REDIS_URL`.**

### 7. Auth & limits (optional)
Set `APP_API_KEYS=sk_a,sk_b` → `X-API-Key: sk_a` required on `/v1/*` (else 401). `APP_RATE_LIMIT_RPM=60` → 429 + `retry_after`. Redis = sliding window; no Redis = in-memory fallback.

---

## MCP — use Cinder from your AI assistant

Cinder ships a Model Context Protocol server ([`cinder-tmcp`](https://github.com/Michael-Obele/cinder-tmcp) — built with [TMCP](https://github.com/paoloricciuti/tmcp) / [tmcp.io](https://tmcp.io)) so Claude, Cursor, or any MCP client can scrape, crawl, and search without leaving the chat.

### Run the MCP server with Docker (self-hosted, no paid API)

This is the **Docker** config — it talks to your local Cinder on `http://localhost:8080`. No Firecrawl cloud key, no per-token bill.

```bash
git clone https://github.com/Michael-Obele/cinder-tmcp.git && cd cinder-tmcp
docker compose up -d          # → cinder-mcp on http://localhost:3000
curl http://localhost:3000/health  # → {"service":"cinder-mcp","status":"ok"}
```

Point your MCP client at `http://localhost:3000/mcp` (Streamable HTTP) or `http://localhost:3000/sse` (legacy SSE).

**Zed** — add to `~/.config/zed/settings.json` under `context_servers`:

```json
{
  "context_servers": {
    "cinder": {
      "enabled": true,
      "url": "http://localhost:3000/mcp"
    }
  }
}
```

**Claude Code / Cursor / Windsurf** — stdio transport (no Docker needed):

```json
{
  "mcpServers": {
    "cinder": {
      "command": "bunx",
      "args": ["-y", "cinder-tmcp"],
      "env": { "CINDER_API_URL": "http://localhost:8080" }
    }
  }
}
```

### Available MCP tools

| Tool | What it does |
|---|---|
| `cinder_scrape` | Scrape one URL → clean markdown, optional screenshot/images/summary/schema extraction |
| `cinder_crawl` | Async BFS crawl → task ID |
| `cinder_crawl_status` | Poll crawl job (pending → active → completed/failed) |
| `cinder_search` | Web search via SearXNG/Brave, domain filters, pagination |
| `cinder_monitor` | `create` / `status` / `delete` change-tracking monitors |

Full docs: [`cinder-tmcp/README.md`](https://github.com/Michael-Obele/cinder-tmcp#readme) · Live docs at `http://localhost:3000/health`.

> **Firecrawl users:** if you also use Firecrawl self-hosted, its MCP is a separate package (`firecrawl-mcp`). Point it at your local Firecrawl with `FIRECRAWL_API_URL=http://localhost:3002` — don't use the hosted `https://mcp.firecrawl.dev/...` URL when running Docker. See [firecrawl-mcp-server](https://github.com/firecrawl/firecrawl-mcp-server#configuration) for the `FIRECRAWL_API_URL` env.

---

## Deploy — one binary, your cloud

**Fly.io (recommended):** `auto_stop` = near-zero idle cost. `fly.toml` is 512MB `shared-cpu-1x` in `lhr` — enough for moderate dynamic scraping (Chrome ~150MB, recycled every `CHROME_RECYCLE_AFTER=100`).

```bash
fly launch          # uses included fly.toml
fly secrets set REDIS_URL=rediss://... SEARXNG_ENDPOINT=http://searxng.internal:8080 APP_API_KEYS=sk_... APP_RATE_LIMIT_RPM=60
fly deploy
```

Sidecar SearXNG on Fly? Same app, second Machine via [`scripts/fly-searxng.sh`](scripts/fly-searxng.sh) + [`docs/guides/SEARXNG_FLY.md`](docs/guides/SEARXNG_FLY.md) (Option A).

- **Railway:** `SERVER_MODE=release`, 512MB hobby works.
- **Leapcell:** 4GB pay-per-minute, ~$5–15/mo.
- **Any Docker host:** Redis is the only external dep — and only for queue/batch/monitor. `docker compose up -d` is the whole deploy.
- **Vercel/Lambda:** not recommended (Chromium ~400MB, cold 10–15s).

---

## Features — outcomes, not buzzwords

| Feature | Benefit → Outcome |
|---|---|
| **Reusable Chrome** + parallel fetch | No spawn tax → 200ms repeat scrapes (gzip Redis 7d), not 17s |
| **Monolith API+Worker** | One deploy, one bill → hobby-tier viable |
| **Readability + cleaner** | Strips boilerplate → LLM gets signal, not nav |
| **Image engine v2** (srcset/picture/lazy, ranked, dimension-sniffed) | Hero, not avatar → better vision RAG |
| **Page actions** | Click/scroll before capture → lazy content loads |
| **Deterministic extract** (`extract_schema`) + summary + PII redact | No LLM cost → structured data safely |
| **Links extraction** (`links: [{url, text, isInternal}]` after readability, deduped) | Firecrawl `formats: ["links"]` parity — `include_links` (default true) |
| **Sync multi-URL scrape** (`urls: []` max 10, errgroup limit 5) | One call like `web_fetch_exa` / Firecrawl `batch/scrape` → `{results: [{url, markdown, metadata}]}` — no Redis |
| **Search highlights** (`highlights: ["…query window…"]`) | Firecrawl `highlights:true` parity — 120-char query-biased snippet per result, always returned |
| **Category filters** (`category: general\|news\|code`) | Exa `category` parity → SearXNG `categories` + Brave `search_type`, cache-aware |
| **TF-IDF rerank** (`?rerank=true`) | Lightweight pure-Go re-rank (no ONNX) — `tf*idf*0.8 + original*0.2`, `bge-small` alternative without hobby-tier penalty |
| **Map / Search / Batch / Monitor** | Discover → scrape → watch → webhook → done |
| **MCP server** (`cinder-tmcp` — [TMCP](https://github.com/paoloricciuti/tmcp) / [tmcp.io](https://tmcp.io)) | 8 tools: `cinder_scrape`/`crawl`/`crawl_status`/`search`/`monitor`/`map`/`batch_scrape`/`links` — no REST glue |

---

## Configuration

| Variable | Default | Why it matters |
|---|---|---|
| `SERVER_PORT` | `8080` | HTTP listen port |
| `SERVER_MODE` | `debug` | `release` disables Swagger + swag regeneration |
| `LOG_LEVEL` / `APP_LOGLEVEL` | `info` | `debug` shows browser hydration + per-scrape timing |
| `REDIS_URL` | — | **Required for `/v1/crawl`, `/v1/batch`, `/v1/monitor`** — also enables search cache + rate-limit sliding window |
| `REDIS_HOST` / `REDIS_PORT` / `REDIS_PASSWORD` | — | Alt to `REDIS_URL` — `host:port` form |
| `UPSTASH_REDIS_REST_URL` / `UPSTASH_REDIS_REST_TOKEN` | — | Derives `rediss://` URL for Upstash REST → Redis |
| `SEARXNG_ENDPOINT` | — | `http://localhost:8889` locally, `http://searxng.internal:8080` on Fly |
| `BRAVE_SEARCH_API_KEY` | — | Fallback when SearXNG unset/unreachable |
| `APP_API_KEYS` | — | Comma-separated — enables `X-API-Key` auth on `/v1/*` |
| `APP_RATE_LIMIT_RPM` | `0` | Per-client req/min, 429 + `retry_after` (Redis = sliding window) |
| `CHROME_RECYCLE_AFTER` | `100` | Restarts allocator every N scrapes — lower to 50 on 512MB if OOM |
| `CRAWL_CONCURRENCY` / `CRAWL_DOMAIN_DELAY` | `4` / `1s` | 1–10 workers, politeness delay |
| `CRAWL_TIMEOUT` / `CRAWL_SCRAPE_TIMEOUT` / `CRAWL_MAX_RETRIES` | `30m` / `30s` / `2` | Crawl deadline / per-page timeout / retry count (4xx never retried) |
| `SSRF_ALLOW_PRIVATE` | `false` | `true` only for scraping internal wikis — disables SSRF guard |
| `DISABLE_WORKER` | `false` | `true` = API only, no embedded Asynq worker |
| `SHUTDOWN_TIMEOUT` | `20` | Must stay below Fly `kill_timeout` (25s) |

All vars also load from `.env` (see [`.env.example`](.env.example)). `REDIS_URL` takes precedence over `REDIS_HOST`/`UPSTASH_*`.

**SSRF:** Every outbound fetch blocks loopback/RFC1918/link-local (169.254.169.254)/CGNAT/multicast at dial time — redirects/DNS rebinding covered. `SSRF_ALLOW_PRIVATE=true` opts out.

---

## Performance

| Operation | Latency | Notes |
|---|---|---|
| Static (Colly) | 200–500ms | No JS |
| Dynamic (Chromedp) | 1–3s | Warm browser; cold start ~1–2s once |
| Queue enqueue | 5–10ms | Redis |
| Search (cached) | ~2ms | After first SearXNG hit |

`CHROME_RECYCLE_AFTER` bounds leaks; singleton allocator = ~200–300MB steady.

---

## Troubleshooting — fix it yourself

**OOM / browser killed after 1–2h?** → Lower `CRAWL_CONCURRENCY`, set `CHROME_RECYCLE_AFTER=50`, or bump to 1GB. Chrome leaks; recycling is intentional.

**`POST /v1/crawl` → 503?** → `REDIS_URL` missing or unreachable. Use `/v1/scrape` sync or set `REDIS_URL=redis://localhost:6379` and `docker compose up -d redis`.

**`POST /v1/search` → 503 / empty?** → `SEARXNG_ENDPOINT` unreachable and no `BRAVE_SEARCH_API_KEY` fallback. With compose, `SEARXNG_ENDPOINT=http://searxng:8080` is set for you; standalone, set `SEARXNG_ENDPOINT=http://localhost:8889` after `docker compose up -d searxng`.

**Dynamic returns empty?** → Page not hydrated. Try `mode: dynamic` + `actions: [{type:"wait_selector", selector:"#app"}]` or `scroll_to_bottom`. Run with `LOG_LEVEL=debug`.

**MCP `cinder` shows disconnected in Zed?** → Check `docker ps | grep cinder-mcp` and `curl http://localhost:3000/health`. Ensure `~/.config/zed/settings.json` has `"cinder": {"url": "http://localhost:3000/mcp"}` and restart Zed.

**Slow?** → `mode: static` for static sites, warm with `curl /v1/scrape` once, check `searxng` health at `http://localhost:8889/healthz`.

Full walkthrough: [`docs/guides/ARCHITECTURE.md`](docs/guides/ARCHITECTURE.md) · [`docs/guides/CODE_WALKTHROUGH.md`](docs/guides/CODE_WALKTHROUGH.md)

---

## Roadmap

| Phase | Done |
|---|---|
| Static/dynamic/smart, queue, auth/limits, perf | ✅ |
| v2: images v2, map, batch, actions, monitors, extract/summary/PII | ✅ |
| MCP server (`cinder-tmcp`) — scrape/search/crawl/monitor as AI tools | ✅ |
| Next: stealth tier (`utls` + CDP stealth), PDF/non-HTML, `pprof` benchmarks, Smart Wait heuristics |  |

Parity gaps tracked honestly in [`docs/EXA_PARITY.md`](docs/EXA_PARITY.md) — **closed:** highlights, category, TF-IDF rerank, multi-URL sync, links, MCP map/batch; **remaining:** full vector semantic search (TF-IDF is lightweight lite).

---

## Contributing

1. Fork → `feature/...` → tests → `make check` (`gofmt`, `vet`, `staticcheck`, `-race`) → PR
2. Use `pkg/logger` (slog), explicit errors, interfaces in `internal/domain`

---

## License

MIT — [`LICENSE`](LICENSE). If Cinder saves you a Firecrawl bill, a ⭐ helps more builders find it.
