# API Specification

> **Purpose:** OpenAPI-style specification for cinder-js endpoints  
> **Compatibility:** 100% compatible with Go Cinder API  
> **Last Updated:** 2026-02-02

---

## Table of Contents

1. [Overview](#overview)
2. [Authentication](#authentication)
3. [Endpoints](#endpoints)
4. [Data Models](#data-models)
5. [Error Responses](#error-responses)
6. [Rate Limiting](#rate-limiting)

---

## Overview

### Base URL

```
Production: https://your-domain.leapcell.app
Local:      http://localhost:8080
```

### API Version

All endpoints are prefixed with `/v1`.

### Content Type

```
Request:  application/json
Response: application/json
```

---

## Authentication

### API Key (Optional)

If `API_KEY` environment variable is set, all requests must include:

```http
Authorization: Bearer <api-key>
```

Or as query parameter:

```
?api_key=<api-key>
```

### Unauthenticated Mode

If no `API_KEY` is configured, the API is open (use with caution).

---

## Endpoints

### Health Check

#### `GET /health`

Simple liveness check.

**Response:**
```json
{
  "status": "healthy"
}
```

**Status Codes:**
- `200` - Service is healthy
- `503` - Service is unhealthy

---

#### `GET /health/ready`

Readiness check (includes Redis, browser status).

**Response:**
```json
{
  "status": "ready",
  "redis": "connected",
  "browser": "initialized",
  "queue": {
    "waiting": 5,
    "active": 2
  }
}
```

**Status Codes:**
- `200` - Ready to accept traffic
- `503` - Not ready (starting up or degraded)

---

### Scrape

#### `POST /v1/scrape`

Scrape a single URL and return content.

**Request Body:**
```json
{
  "url": "https://example.com",
  "mode": "smart"
}
```

| Field  | Type   | Required | Description                                                    |
| ------ | ------ | -------- | -------------------------------------------------------------- |
| `url`  | string | ✅        | URL to scrape (http/https)                                     |
| `mode` | string | ❌        | Scraping mode: `static`, `dynamic`, `smart` (default: `smart`) |

**Response:**
```json
{
  "url": "https://example.com",
  "markdown": "# Example Domain\n\nThis domain is for use in illustrative examples...",
  "html": "<!DOCTYPE html>...",
  "metadata": {
    "title": "Example Domain",
    "description": "Example domain for documentation",
    "scraped_at": "2026-02-02T10:30:00Z",
    "engine": "cheerio",
    "mode": "static"
  }
}
```

**Status Codes:**
- `200` - Success
- `400` - Invalid request (bad URL, invalid mode)
- `422` - Scrape failed (site unreachable, timeout)
- `500` - Internal server error
- `503` - Service unavailable (browser not initialized)

---

#### `GET /v1/scrape`

Alternative GET endpoint (URL as query param).

**Query Parameters:**
- `url` (required) - URL to scrape
- `mode` (optional) - Scraping mode

**Example:**
```
GET /v1/scrape?url=https://example.com&mode=static
```

**Response:** Same as POST

---

### Search

#### `POST /v1/search`

Search the web using configured search provider.

**Request Body:**
```json
{
  "query": "web scraping best practices",
  "limit": 10
}
```

| Field   | Type   | Required | Description                        |
| ------- | ------ | -------- | ---------------------------------- |
| `query` | string | ✅        | Search query                       |
| `limit` | number | ❌        | Max results (default: 10, max: 50) |

**Response:**
```json
{
  "query": "web scraping best practices",
  "results": [
    {
      "title": "Web Scraping: Best Practices Guide",
      "url": "https://example.com/guide",
      "description": "Learn the best practices for...",
      "position": 1
    }
  ],
  "total": 10
}
```

**Status Codes:**
- `200` - Success
- `400` - Invalid request
- `503` - Search provider unavailable

---

#### `GET /v1/search`

Alternative GET endpoint.

**Query Parameters:**
- `q` or `query` (required) - Search query
- `limit` (optional) - Max results

---

### Crawl (Async)

#### `POST /v1/crawl`

Start an asynchronous crawl job. Requires Redis.

**Request Body:**
```json
{
  "url": "https://example.com",
  "render": false,
  "max_depth": 2,
  "max_pages": 10
}
```

| Field       | Type    | Required | Description                       |
| ----------- | ------- | -------- | --------------------------------- |
| `url`       | string  | ✅        | Starting URL                      |
| `render`    | boolean | ❌        | Force Playwright (default: false) |
| `max_depth` | number  | ❌        | Max link depth (default: 2)       |
| `max_pages` | number  | ❌        | Max pages to crawl (default: 10)  |

**Response:**
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "status": "queued",
  "message": "Crawl job queued successfully"
}
```

**Status Codes:**
- `202` - Accepted (job queued)
- `400` - Invalid request
- `503` - Redis unavailable (crawl disabled)

---

#### `GET /v1/crawl/:id`

Get crawl job status and results.

**Path Parameters:**
- `id` - Crawl job ID (UUID)

**Response (In Progress):**
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "status": "processing",
  "progress": 45,
  "pages_scraped": 4,
  "pages_total": 10,
  "started_at": "2026-02-02T10:30:00Z"
}
```

**Response (Completed):**
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000", 
  "status": "completed",
  "progress": 100,
  "pages_scraped": 10,
  "pages_total": 10,
  "started_at": "2026-02-02T10:30:00Z",
  "completed_at": "2026-02-02T10:31:45Z",
  "results": [
    {
      "url": "https://example.com",
      "markdown": "# Example...",
      "metadata": {...}
    },
    {
      "url": "https://example.com/about",
      "markdown": "# About...",
      "metadata": {...}
    }
  ]
}
```

**Response (Failed):**
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "status": "failed",
  "error": "Timeout exceeded while crawling",
  "pages_scraped": 3,
  "started_at": "2026-02-02T10:30:00Z",
  "failed_at": "2026-02-02T10:35:00Z"
}
```

**Status Codes:**
- `200` - Success
- `404` - Job not found
- `503` - Redis unavailable

---

## Data Models

### ScrapeResult

```typescript
interface ScrapeResult {
  url: string;
  markdown: string;
  html: string;
  metadata: ScrapeMetadata;
}

interface ScrapeMetadata {
  title?: string;
  description?: string;
  scraped_at: string;      // ISO 8601
  engine: 'cheerio' | 'playwright';
  mode: 'static' | 'dynamic' | 'smart';
  cache_hit?: boolean;
}
```

### CrawlJob

```typescript
interface CrawlJob {
  id: string;             // UUID
  status: CrawlStatus;
  progress: number;       // 0-100
  pages_scraped: number;
  pages_total: number;
  started_at: string;
  completed_at?: string;
  failed_at?: string;
  results?: ScrapeResult[];
  error?: string;
}

type CrawlStatus = 'queued' | 'processing' | 'completed' | 'failed';
```

### SearchResult

```typescript
interface SearchResult {
  title: string;
  url: string;
  description: string;
  position: number;
}

interface SearchResponse {
  query: string;
  results: SearchResult[];
  total: number;
}
```

---

## Error Responses

### Standard Error Format

```json
{
  "error": "Human-readable error message",
  "code": "ERROR_CODE",
  "details": {
    "field": "Additional context"
  }
}
```

### Error Codes

| Code                  | HTTP Status | Description                             |
| --------------------- | ----------- | --------------------------------------- |
| `BAD_REQUEST`         | 400         | Invalid request body or parameters      |
| `INVALID_URL`         | 400         | URL is malformed or unsupported         |
| `INVALID_MODE`        | 400         | Mode must be static, dynamic, or smart  |
| `UNAUTHORIZED`        | 401         | Missing or invalid API key              |
| `NOT_FOUND`           | 404         | Resource (job) not found                |
| `SCRAPE_FAILED`       | 422         | Failed to scrape URL                    |
| `TIMEOUT`             | 422         | Request timed out                       |
| `RATE_LIMITED`        | 429         | Too many requests                       |
| `INTERNAL_ERROR`      | 500         | Unexpected server error                 |
| `SERVICE_UNAVAILABLE` | 503         | Dependency (Redis, browser) unavailable |

### Example Error Response

```json
{
  "error": "Invalid URL provided",
  "code": "INVALID_URL",
  "details": {
    "url": "not-a-valid-url",
    "reason": "URL must start with http:// or https://"
  }
}
```

---

## Rate Limiting

### Default Limits

| Endpoint      | Limit | Window     |
| ------------- | ----- | ---------- |
| `/v1/scrape`  | 60    | per minute |
| `/v1/search`  | 30    | per minute |
| `/v1/crawl`   | 10    | per minute |
| All endpoints | 1000  | per hour   |

### Rate Limit Headers

```http
X-RateLimit-Limit: 60
X-RateLimit-Remaining: 45
X-RateLimit-Reset: 1706872800
```

### Rate Limit Response

```http
HTTP/1.1 429 Too Many Requests
Retry-After: 30

{
  "error": "Rate limit exceeded",
  "code": "RATE_LIMITED",
  "details": {
    "limit": 60,
    "window": "1 minute",
    "retry_after": 30
  }
}
```

---

## OpenAPI Specification (Summary)

```yaml
openapi: 3.0.3
info:
  title: Cinder JS API
  version: 1.0.0
  description: Web scraping and crawling API

servers:
  - url: https://your-domain.leapcell.app/v1

paths:
  /health:
    get:
      summary: Health check
      
  /v1/scrape:
    post:
      summary: Scrape URL
      requestBody:
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/ScrapeRequest'
              
  /v1/crawl:
    post:
      summary: Start crawl job
      
  /v1/crawl/{id}:
    get:
      summary: Get crawl status

components:
  schemas:
    ScrapeRequest:
      type: object
      required: [url]
      properties:
        url:
          type: string
          format: uri
        mode:
          type: string
          enum: [static, dynamic, smart]
          default: smart
```

---

*Document Version: 1.0.0-draft*  
*Last Updated: 2026-02-02*
