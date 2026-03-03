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

| Parameter    | Type    | Required | Default | Description                                                                                                       |
| ------------ | ------- | -------- | ------- | ----------------------------------------------------------------------------------------------------------------- |
| `url`        | string  | **Yes**  | -       | The full URL of the webpage to scrape.                                                                            |
| `mode`       | string  | No       | `smart` | Scraping mode: `smart`, `static`, or `dynamic`.                                                                   |
| `screenshot` | boolean | No       | `false` | Capture full-page screenshot (requires mode `dynamic` or `smart`). Return payload includes base64 representation. |
| `images`     | boolean | No       | `false` | Extract images as base64 blobs from the document.                                                                 |
| `render`     | boolean | No       | `false` | *Deprecated*. Behaves the same as `mode=dynamic`.                                                                 |

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

| Parameter        | Type          | Required | Default | Description                                                   |
| ---------------- | ------------- | -------- | ------- | ------------------------------------------------------------- |
| `query` or `q`   | string        | **Yes**  | -       | The search query.                                             |
| `offset`         | int           | No       | `0`     | Pagination offset.                                            |
| `limit`          | int           | No       | `10`    | Pagination limit (Maximum: 100).                              |
| `includeDomains` | array[string] | No       | -       | Restrict results to these domains (e.g. `["wikipedia.org"]`). |
| `excludeDomains` | array[string] | No       | -       | Exclude results from these domains.                           |
| `requiredText`   | array[string] | No       | -       | Filter results containing this text.                          |
| `maxAge`         | int           | No       | -       | Max age of the result in days.                                |

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
Submits a URL to be crawled asynchronously using the background worker queue.
**Important:** Asynchronous crawling requires an active Redis connection (`REDIS_URL` in config).

### Endpoints
- `POST /v1/crawl`

### Request Parameters
Accepts standard scraping parameters inside a JSON body.

| Parameter    | Type    | Required | Default | Description                          |
| ------------ | ------- | -------- | ------- | ------------------------------------ |
| `url`        | string  | **Yes**  | -       | The URL to crawl asynchronously.     |
| `render`     | boolean | No       | `false` | Render JavaScript.                   |
| `screenshot` | boolean | No       | `false` | Capture screenshots during crawling. |
| `images`     | boolean | No       | `false` | Extract images.                      |

### Example Request
```bash
curl -X POST http://localhost:8080/v1/crawl \
  -H "Content-Type: application/json" \
  -d '{
    "url": "https://example.com",
    "render": true
  }'
```

### Example Response
Returns an HTTP `202 Accepted` indicating that the task was successfully added to the queue.
```json
{
  "id": "e8a932c0-82af-4a11-bd4a-6f17e29b1111",
  "url": "https://example.com",
  "render": true,
  "screenshot": false,
  "images": false
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

### Example Response
```json
{
  "id": "e8a932c0-82af-4a11-bd4a-6f17e29b1111",
  "queue": "default",
  "state": "active",
  "max_retry": 5,
  "retried": 0,
  "payload": "{\"url\":\"https://example.com\",\"render\":true,\"screenshot\":false,\"images\":false}",
  "result": ""
}
```
*Note: Wait for `state` to change to `completed` to consume the content returned in `result` (which will contain the fully serialized JSON of the scrape response).*

---

## Swagger Docs
If you start Cinder in `debug` mode, interactive API documentation is automatically generated by Swagger and available at:
```
http://localhost:8080/swagger/index.html
```
