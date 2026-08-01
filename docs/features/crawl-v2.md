# Crawl v2 — Parallel, Polite, Retryable

> Feature spec — implemented 2026-08-01 (`internal/worker/crawl_handler.go`)

## What changed

### 1. Parallel worker pool
The BFS crawl now runs on a shared worker pool (`CRAWL_CONCURRENCY`, default 4,
max 10) instead of one page at a time. A page at depth N that links to 10
pages at depth N+1 has all 10 scraped concurrently.

### 2. Per-domain politeness
A minimum interval between requests to the **same host** is enforced across
all workers (`CRAWL_DOMAIN_DELAY`, default 1s). Different domains are not
throttled against each other.

### 3. Retry policy
- Transient failures (network, 5xx) are retried twice with backoff (1s, 2s).
- **4xx errors are never retried** — recorded as failed immediately
  (`scraper.StatusError` carries the HTTP status through the stack).

### 4. Include/exclude patterns
`include_paths` / `exclude_paths` are glob patterns matched against the URL
path (`path.Match` semantics). Exclusion wins; empty include = allow all.

### 5. Signed completion webhooks
`webhook_url` + `webhook_secret` POST the full `CrawlResult` on completion
with header `X-Cinder-Signature: sha256=<hmac-hex>`. Three delivery attempts
with backoff; 4xx responses are not retried. Delivery failure never fails the
crawl — it's logged and visible via the task result.

## API

```json
{
  "url": "https://docs.example.com",
  "maxDepth": 3,
  "limit": 50,
  "include_paths": ["/docs/*"],
  "exclude_paths": ["/docs/internal/*", "/login"],
  "webhook_url": "https://myapp.example.com/hooks/cinder",
  "webhook_secret": "s3cret"
}
```

## Env vars

| Var | Default | Purpose |
| --- | ------- | ------- |
| `CRAWL_CONCURRENCY` | 4 | Parallel workers (1-10) |
| `CRAWL_DOMAIN_DELAY` | 1 | Seconds between same-host requests |
| `WEBHOOK_TIMEOUT` | 10 | Webhook HTTP timeout (s) |

## Concurrency correctness
- `visited` dedup, result/failure collection, and politeness bookkeeping are
  mutex-guarded; queue closure is `sync.Once`-guarded (both exhaustion and
  context cancellation close it exactly once).
- Result order is completion order (documented non-determinism).
