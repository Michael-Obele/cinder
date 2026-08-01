# Map, Batch Scrape & Page Actions

> Feature spec — implemented 2026-08-01 (`internal/sitemap`, `internal/api/handlers`, `internal/scraper`)

## 1. `/v1/map` — URL Discovery

Discovers a site's URLs **without scraping their content**, the Firecrawl
Map equivalent. Discovery order:

1. `robots.txt` → `Sitemap:` lines
2. Default `/sitemap.xml` (when robots.txt declares none)
3. Recursive sitemap-index traversal (depth cap 3, total cap 5000 URLs)
4. **Fallback**: one-level same-domain link discovery from the seed page

```json
// POST /v1/map
{ "url": "https://docs.example.com", "search": "/docs", "limit": 200 }

// 200 OK
{
  "url": "https://docs.example.com",
  "count": 2,
  "links": [
    { "url": "https://docs.example.com/docs/intro", "source": "sitemap" },
    { "url": "https://docs.example.com/docs/api", "source": "link" }
  ]
}
```

Params: `url` (required), `search` (substring filter), `limit` (default 100,
max 5000). 30s request timeout. No Redis required.

## 2. `/v1/batch/scrape` — Batch Scraping

Enqueue up to 20 URLs as individual Asynq scrape tasks under one batch ID
(Redis required). Batch record lives in Redis for 7 days.

```json
// POST /v1/batch/scrape
{ "urls": ["https://a.example.com", "https://b.example.com"] }

// 202 Accepted
{
  "batch_id": "3f2a...",
  "tasks": [{ "id": "t1", "url": "https://a.example.com" }, ...]
}

// GET /v1/batch/:id → 200
{
  "batch_id": "3f2a...",
  "total": 2, "completed": 1, "failed": 0,
  "tasks": [...]
}
```

## 3. Page Actions (dynamic mode)

Run browser interactions **before** HTML capture so interaction-driven
content is included. Static mode rejects actions with a 400; smart mode
auto-upgrades to dynamic.

| Type | Fields | Behavior |
| ---- | ------ | -------- |
| `wait_ms` | `ms` | Sleep (default 500ms) |
| `wait_selector` | `selector` | Wait until visible (5s timeout) |
| `click` | `selector` | Click first matching node |
| `scroll_down` | — | Scroll one viewport height |
| `scroll_to_bottom` | — | Settle loop: scroll → 500ms → compare `scrollHeight`, max 10 iterations |

```json
{
  "url": "https://infinite.example.com",
  "mode": "dynamic",
  "actions": [
    { "type": "wait_selector", "selector": "#app .content" },
    { "type": "scroll_to_bottom" }
  ]
}
```

Max 10 actions per request; unknown types return 400.
