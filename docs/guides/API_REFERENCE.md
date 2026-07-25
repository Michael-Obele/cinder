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
| `screenshot`     | boolean | No       | `false` | Capture full-page screenshot (requires mode `dynamic` or `smart`). Returns base64 JPEG blob.                     |
| `images`         | boolean | No       | `false` | Extract images as base64 blobs from the document.                                                                 |
| `image_format`   | string  | No       | `url`   | Image transport format: `"url"` (metadata only) or `"blob"` (base64-encoded).                                     |
| `max_images`     | int     | No       | `10`    | Maximum number of images to extract.                                                                              |
| `max_image_size_kb` | int  | No       | `5120`  | Maximum image file size in KB (default 5MB).                                                                      |
| `render`         | boolean | No       | `false` | *Deprecated*. Behaves the same as `mode=dynamic`.                                                                 |

### Example Request (`POST`)
```bash
curl -X POST http://localhost:8080/v1/scrape \
  -H "Content-Type: application/json" \
  -d '{
    "url": "https://example.com",
    "mode": "smart",
    "screenshot": false,
    "images": false
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

| Parameter    | Type    | Required | Default | Description                                                       |
| ------------ | ------- | -------- | ------- | ----------------------------------------------------------------- |
| `url`        | string  | **Yes**  | -       | The seed URL to start crawling from.                              |
| `mode`       | string  | No       | `smart` | Scraping mode: `smart`, `static`, or `dynamic`.                   |
| `maxDepth`   | int     | No       | `2`     | Maximum link-following depth from the seed URL. Capped at `10`.   |
| `limit`      | int     | No       | `10`    | Maximum total number of pages to scrape. Capped at `100`.         |
| `render`     | boolean | No       | `false` | Render JavaScript for each page (uses headless browser).          |
| `screenshot` | boolean | No       | `false` | Capture screenshots for each scraped page.                        |
| `images`     | boolean | No       | `false` | Extract images from each scraped page.                            |

### Crawl Behavior
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
    "render": false
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

## Swagger Docs
If you start Cinder in `debug` mode, interactive API documentation is automatically generated by Swagger and available at:
```
http://localhost:8080/swagger/index.html
```
