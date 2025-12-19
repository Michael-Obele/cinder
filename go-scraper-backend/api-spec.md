# API Specification

## 1. Scrape Single Page

Synchronous endpoint. Visits a URL and returns the content immediately.

**Endpoint:** `POST /v1/scrape`

**Headers:**

- `X-API-Key`: `your_secret_key`
- `Content-Type`: `application/json`

**Request Body:**

```json
{
  "url": "https://example.com/blog/article-1",
  "formats": ["markdown", "html"],
  "render": false,
  "waitFor": 0,
  "headers": {
    "Cookie": "session=123"
  },
  "excludeTags": ["#ad-banner", ".footer", "nav"],
  "includeTags": ["article", "main"]
}
```

| Field         | Type   | Description                                                                  |
| :------------ | :----- | :--------------------------------------------------------------------------- |
| `url`         | string | **Required**. The URL to scrape.                                             |
| `formats`     | array  | Output formats. Default `["markdown"]`. Options: `markdown`, `html`, `text`. |
| `render`      | bool   | If `true`, uses Headless Chrome (slower, handles JS). Default `false`.       |
| `waitFor`     | int    | Milliseconds to wait after page load (only used if `render: true`).          |
| `excludeTags` | array  | CSS selectors to remove from the output (cleaning).                          |

**Response (200 OK):**

```json
{
  "success": true,
  "data": {
    "markdown": "# Article Title\n\nHere is the content...",
    "html": "<html>...</html>",
    "metadata": {
      "title": "Article Title",
      "description": "SEO description",
      "language": "en",
      "statusCode": 200
    }
  }
}
```

---

## 2. Crawl Domain (Async)

Asynchronous endpoint. Starts a job to crawl a website (BFS/DFS).

**Endpoint:** `POST /v1/crawl`

**Request Body:**

```json
{
  "url": "https://docs.example.com",
  "limit": 100,
  "maxDepth": 2,
  "webhook": "https://my-server.com/webhook"
}
```

**Response (200 OK):**

```json
{
  "success": true,
  "jobId": "c8f9d2a1-4b3c-..."
}
```

---

## 3. Get Crawl Status

Check the status of a crawl job.

**Endpoint:** `GET /v1/crawl/:jobId`

**Response (200 OK):**

```json
{
  "success": true,
  "status": "completed",
  "progress": 100,
  "total": 45,
  "data": [
    { "url": "https://docs.example.com/page1", "markdown": "..." },
    { "url": "https://docs.example.com/page2", "markdown": "..." }
  ]
}
```

_Status values: `pending`, `active`, `completed`, `failed`_
