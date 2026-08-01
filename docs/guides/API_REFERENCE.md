# Cinder API Documentation

Cinder provides a high-performance, self-hosted web scraping API. All API endpoints are prefixed with `/v1`. 

## Base URL
```
http://localhost:8080/v1
```
> **Note:** When using Cinder in a production environment, the `http://localhost:8080` portion will be replaced by your actual domain or production URL. All API endpoints and payload structures remain identical.

---

## 1. Scrape
Scrapes a given URL and returns its markdown content, metadata, and optionally captures a screenshot or extracts images if enabled.

### Endpoints
- `POST /v1/scrape`
- `GET /v1/scrape`

### Request Parameters

You can send parameters as a JSON body (for `POST`) or as query string parameters (for both `GET` and `POST`).

| Parameter        | Type    | Required | Default | Description                                                                                                       |
| ---------------- | ------- | -------- | ------- | ----------------------------------------------------------------------------------------------------------------- |
| `url`            | string  | **Yes**  | -       | The full URL of the webpage to scrape.                                                                            |
| `mode`           | string  | No       | `smart` | Scraping mode: `smart`, `static`, or `dynamic`.                                                                   |
| `screenshot`     | boolean | No       | `false` | Capture screenshot (requires mode `dynamic` or `smart`). Returns base64 JPEG blob.                               |
| `screenshot_opts`| object  | No       | -       | Screenshot configuration: `width`, `height`, `full_page`, `format` (`jpeg`/`png`), `quality` (1-100), `wait_selector`. |
| `images`         | boolean | No       | `false` | Extract images as base64 blobs from the document.                                                                 |
| `image_format`   | string  | No       | `url`   | Image transport format: `"url"` (metadata only) or `"blob"` (base64-encoded).                                     |
| `max_images`     | int     | No       | `10`    | Maximum number of images to extract (quality-ranked: og > hero > content > avatar).                              |
| `max_image_size_kb` | int  | No       | `5120`  | Maximum image file size in KB (default 5MB).                                                                      |
| `image_process`  | object  | No       | -       | Resize/re-encode blobs: `format` (`jpeg`/`png`), `max_width`, `quality`.                                          |
| `actions`        | array   | No       | -       | Page interactions before capture (dynamic only): `wait_ms`, `wait_selector`, `click`, `scroll_down`, `scroll_to_bottom`. |
| `extract_schema` | object  | No       | -       | Deterministic CSS-selector extraction: `{field: {selector, attr?, multiple?}}`.                                  |
| `summary`        | boolean | No       | `false` | Return an extractive `summary` field (no LLM).                                                                    |
| `summary_sentences` | int  | No       | `5`     | Sentence count for the extractive summary.                                                                        |
| `redact_pii`     | boolean | No       | `false` | Mask emails, phone numbers, and card-shaped digit runs in markdown/summary.                                       |
| `block_ads`      | boolean | No       | `true`  | Strip common ad/tracker containers before markdown conversion.                                                    |
| `remove_base64_images` | boolean | No | `true`  | Drop inline `data:` images before markdown conversion.                                                            |
| `render`         | boolean | No       | `false` | *Deprecated*. Behaves the same as `mode=dynamic`.                                                                 |

### Example Request (`POST`)
```bash
curl -X POST http://localhost:8080/v1/scrape \
  -H "Content-Type: application/json" \
  -d '{
    "url": "https://example.com",
    "mode": "smart",
    "images": true,
    "max_images": 5,
    "summary": true,
    "extract_schema": {
      "title": {"selector": "h1"},
      "links": {"selector": "a", "attr": "href", "multiple": true}
    },
    "actions": [{"type": "scroll_to_bottom"}]
  }'
```

### Example Request (`GET`)
```bash
curl "http://localhost:8080/v1/scrape?url=https://example.com&mode=smart"
```

### Example Response
```json
{
  "url": "https://example.com",
  "markdown": "# Example Domain\n\nThis domain is for use in illustrative examples in documents...",
  "html": "<!doctype html>\n<html>\n...",
  "metadata": {
    "title": "Example Domain",
    "description": "Example Domain Description"
  }
}
```
*(Note: If `screenshot` or `images` are requested, the response payload will also contain `screenshot` and `images` objects with base64 data strings).*

---

## 2. Search
Searches the web using the configured search provider (Brave Search) and returns a list of matching results. Requires `BRAVE_SEARCH_API_KEY` configuration.

### Endpoints
- `POST /v1/search`
- `GET /v1/search`

### Request Parameters

| Parameter        | Type          | Required | Default | Description                                                              |
| ---------------- | ------------- | -------- | ------- | ------------------------------------------------------------------------ |
| `query` or `q`   | string        | **Yes**  | -       | The search query.                                                        |
| `offset`         | int           | No       | `0`     | Pagination offset.                                                       |
| `limit`          | int           | No       | `10`    | Pagination limit (Maximum: 100).                                         |
| `mode`           | string        | No       | -       | Search speed: `"fast"` restricts to recent results (last day).            |
| `includeDomains` | array[string] | No       | -       | Restrict results to these domains (e.g. `["wikipedia.org"]`).            |
| `excludeDomains` | array[string] | No       | -       | Exclude results from these domains.                                      |
| `requiredText`   | array[string] | No       | -       | Filter results containing this text.                                     |
| `maxAge`         | int           | No       | -       | Max age in days: `1` (day), `7` (week), `30` (month).                    |

### Example Request (`POST`)
```bash
curl -X POST http://localhost:8080/v1/search \
  -H "Content-Type: application/json" \
  -d '{
    "query": "cinder web scraper",
    "limit": 5,
    "offset": 0
  }'
```

### Example Response
```json
{
  "query": "cinder web scraper",
  "results": [
    {
      "title": "Cinder on GitHub",
      "url": "https://github.com/standard-user/cinder",
      "description": "A high-performance web crawling API..."
    }
  ],
  "hasMore": true,
  "nextOffset": 5,
  "count": 1
}
```

---

## 3. Asynchronous Crawl (Enqueue)
Submits a seed URL to be crawled asynchronously using the background worker queue. The crawler performs **BFS (breadth-first) link-following**, scraping pages on the same domain up to the configured depth and page limit.

**Important:** Asynchronous crawling requires an active Redis connection (`REDIS_URL` in config).

### Endpoints
- `POST /v1/crawl`

### Request Parameters
Accepts a JSON body with scraping parameters and crawl-specific options.

| Parameter       | Type    | Required | Default | Description                                                       |
| --------------- | ------- | -------- | ------- | ----------------------------------------------------------------- |
| `url`           | string  | **Yes**  | -       | The seed URL to start crawling from.                              |
| `mode`          | string  | No       | `smart` | Scraping mode: `smart`, `static`, or `dynamic`.                   |
| `maxDepth`      | int     | No       | `2`     | Maximum link-following depth from the seed URL. Capped at `10`.   |
| `limit`         | int     | No       | `10`    | Maximum total number of pages to scrape. Capped at `100`.         |
| `render`        | boolean | No       | `false` | Render JavaScript for each page (uses headless browser).          |
| `screenshot`    | boolean | No       | `false` | Capture screenshots for each scraped page.                        |
| `images`        | boolean | No       | `false` | Extract images from each scraped page.                            |
| `include_paths` | array   | No       | -       | Only follow links whose path matches these globs (e.g. `["/blog/*"]`). |
| `exclude_paths` | array   | No       | -       | Never follow links whose path matches these globs (exclusion wins). |
| `webhook_url`   | string  | No       | -       | POST the crawl result here on completion.                         |
| `webhook_secret`| string  | No       | -       | HMAC-SHA256 key for the `X-Cinder-Signature` header.              |

### Crawl Behavior
- **Parallel**: Pages are scraped concurrently (env `CRAWL_CONCURRENCY`, default 4, max 10).
- **Polite**: A minimum interval between requests to the same host (env `CRAWL_DOMAIN_DELAY`, default 1s).
- **Retries**: Transient failures are retried twice with backoff; 4xx errors are never retried.
- **Domain-locked**: The crawler only follows links on the same hostname as the seed URL.
- **Deduplication**: Each URL is visited only once per crawl job.
- **Resource filtering**: Non-HTML resources (`.pdf`, `.jpg`, `.css`, `.js`, etc.) are automatically skipped.

### Example Request
```bash
curl -X POST http://localhost:8080/v1/crawl \
  -H "Content-Type: application/json" \
  -d '{
    "url": "https://docs.example.com",
    "maxDepth": 3,
    "limit": 20,
    "exclude_paths": ["/admin/*", "/login"],
    "webhook_url": "https://myapp.example.com/hooks/cinder",
    "webhook_secret": "s3cret"
  }'
```

### Example Response
Returns an HTTP `202 Accepted` indicating that the crawl task was successfully added to the queue.
```json
{
  "id": "e8a932c0-82af-4a11-bd4a-6f17e29b1111",
  "url": "https://docs.example.com",
  "render": false,
  "screenshot": false,
  "images": false,
  "maxDepth": 3,
  "limit": 20
}
```

---

## 4. Crawl Status
Retrieves the current status and result of a previously enqueued crawl task using its `id`.

### Endpoints
- `GET /v1/crawl/:id`

### Example Request
```bash
curl http://localhost:8080/v1/crawl/e8a932c0-82af-4a11-bd4a-6f17e29b1111
```

### Example Response (In Progress)
```json
{
  "id": "e8a932c0-82af-4a11-bd4a-6f17e29b1111",
  "queue": "default",
  "state": "active"
}
```

> **Note:** The API now returns a cleaned-up response. Fields like `payload`, `max_retry`, `retried`, and raw `result` are no longer exposed — instead, the parsed crawl data is presented in a structured format when the task completes.

### Example Response (Completed)
When the crawl finishes, `state` becomes `"completed"` and a structured `crawl` object appears:
```json
{
  "id": "e8a932c0-82af-4a11-bd4a-6f17e29b1111",
  "queue": "default",
  "state": "completed",
  "crawl": {
    "status": "completed",
    "total_pages": 5,
    "max_depth": 3,
    "limit": 20,
    "pages": [
      {
        "url": "https://docs.example.com",
        "title": "Example Docs",
        "preview": "This is the first 300 characters of the page markdown..."
      }
    ]
  },
  "failed_urls": [
    { "url": "https://docs.example.com/404", "error": "scraping failed: ..." }
  ]
}
```

The `crawl` object contains:

| Field         | Type             | Description                                                     |
| ------------- | ---------------- | --------------------------------------------------------------- |
| `status`      | string           | `"completed"`, `"partial"` (some pages failed), `"failed"`, `"cancelled"` |
| `total_pages` | int              | Total pages successfully scraped.                               |
| `max_depth`   | int              | The maxDepth that was used.                                     |
| `limit`       | int              | The limit that was used.                                        |
| `pages`       | array            | Scraped pages with `url`, `title`, `preview` (first 300 chars). |
| `failed_urls` | array (optional) | URLs that failed to scrape, with error messages.                |

> **Frontend poll pattern**: In your SvelteKit app, poll `GET /v1/crawl/:id` every 5s until `state` is `"completed"` or `"failed"`. Display the `pages` array as a list of scraped URLs with their titles and previews.

---

## 5. Map (URL Discovery)
Discovers the URLs of a site without scraping their content — ideal for planning a crawl or building a sitemap. Reads `robots.txt` → `sitemap.xml` (recursive sitemap-index support, up to 5,000 URLs), falling back to one-level link discovery when no sitemap exists.

### Endpoints
- `POST /v1/map`

### Request Parameters

| Parameter | Type   | Required | Default  | Description                                      |
| --------- | ------ | -------- | -------- | ------------------------------------------------ |
| `url`     | string | **Yes**  | -        | The site to map.                                 |
| `search`  | string | No       | -        | Only return URLs containing this substring.      |
| `limit`   | int    | No       | `100`    | Maximum URLs to return (max 5000).               |

### Example Request
```bash
curl -X POST http://localhost:8080/v1/map \
  -H "Content-Type: application/json" \
  -d '{"url": "https://docs.example.com", "search": "/docs", "limit": 200}'
```

### Example Response
```json
{
  "url": "https://docs.example.com",
  "count": 2,
  "links": [
    { "url": "https://docs.example.com/docs/intro", "source": "sitemap" },
    { "url": "https://docs.example.com/docs/api", "source": "link" }
  ]
}
```

---

## 6. Batch Scrape
Scrapes multiple URLs asynchronously in one call. Requires Redis.

### Endpoints
- `POST /v1/batch/scrape` — enqueue up to 20 URLs
- `GET /v1/batch/:id` — aggregate status

### Example Request
```bash
curl -X POST http://localhost:8080/v1/batch/scrape \
  -H "Content-Type: application/json" \
  -d '{"urls": ["https://a.example.com", "https://b.example.com"]}'
```

### Example Response (202)
```json
{
  "batch_id": "3f2a...",
  "tasks": [
    { "id": "task-id-1", "url": "https://a.example.com" },
    { "id": "task-id-2", "url": "https://b.example.com" }
  ]
}
```

### Status Response
```json
{
  "batch_id": "3f2a...",
  "total": 2,
  "completed": 1,
  "failed": 0,
  "tasks": [ ... ]
}
```

---

## 7. Monitor (Change Tracking)
Scrapes a URL on a schedule, hashes the markdown (SHA-256), and fires a signed webhook when content changes. The first check records the baseline without notifying. Requires Redis.

### Endpoints
- `POST /v1/monitor` — create (interval minimum 3600s / 1h)
- `GET /v1/monitor/:id` — status (config + last hash + next check)
- `DELETE /v1/monitor/:id` — stop monitoring

### Example Request
```bash
curl -X POST http://localhost:8080/v1/monitor \
  -H "Content-Type: application/json" \
  -d '{
    "url": "https://pricing.example.com",
    "interval_seconds": 3600,
    "webhook_url": "https://myapp.example.com/hooks/price-changed",
    "webhook_secret": "s3cret"
  }'
```

### Change Webhook Payload
```json
{
  "monitor_id": "abc123",
  "url": "https://pricing.example.com",
  "changed": true,
  "hash_old": "aaa...",
  "hash_new": "bbb...",
  "changed_at": "2026-08-01T12:00:00Z"
}
```
The webhook carries `X-Cinder-Signature: sha256=<hmac-hex>` (HMAC-SHA256 of the body keyed by `webhook_secret`).

---

## 8. Authentication & Rate Limiting
Set `API_KEYS` (comma-separated) in the environment to require `X-API-Key` on all `/v1/*` routes (401 otherwise). With no keys configured the API stays open.

| Env Var         | Default | Description                                   |
| --------------- | ------- | --------------------------------------------- |
| `API_KEYS`      | empty   | Comma-separated keys, e.g. `sk-a,sk-b`.       |
| `RATE_LIMIT_RPM`| `0`     | Per-client requests/minute (0 = unlimited).   |

```bash
curl -X POST http://localhost:8080/v1/scrape \
  -H "Content-Type: application/json" \
  -H "X-API-Key: sk-a" \
  -d '{"url": "https://example.com"}'
```

When rate limiting is enabled, exceeding the limit returns `429` with `retry_after`. Redis-backed limiting is used when `REDIS_URL` is set; otherwise an in-memory limiter applies per instance.

---

## Swagger Docs
If you start Cinder in `debug` mode, interactive API documentation is automatically generated by Swagger and available at:
```
http://localhost:8080/swagger/index.html
```
