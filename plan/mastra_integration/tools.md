# Tools Specification

This document defines the MCP tools to be implemented in the Mastra application.

## 1. Scrape Tool

**Name**: `cinder_scrape`
**Description**: Scrape content from a single URL. Supports different modes for handling dynamic content.

### Input Schema (Zod)

```typescript
z.object({
  url: z.string().url().describe("The URL to scrape"),
  mode: z
    .enum(["smart", "static", "dynamic"])
    .optional()
    .default("smart")
    .describe(
      "Scraping mode: 'static' for fast HTML fetching, 'dynamic' for rendering JS, 'smart' for auto-detection"
    ),
});
```

### Cinder API Mapping

- **Endpoint**: `POST /v1/scrape`
- **Payload**:
  ```json
  {
    "url": "<url>",
    "mode": "<mode>"
  }
  ```

---

## 2. Search Tool

**Name**: `cinder_search`
**Description**: Search the web using Cinder's search capabilities.

### Input Schema (Zod)

```typescript
z.object({
  query: z.string().describe("The search query"),
});
```

### Cinder API Mapping

- **Endpoint**: `POST /v1/search`
- **Payload**:
  ```json
  {
    "query": "<query>"
  }
  ```

---

## 3. Crawl Tool

**Name**: `cinder_crawl`
**Description**: Start a background crawl job for a URL. Returns a Crawl ID to check status later.

### Input Schema (Zod)

```typescript
z.object({
  url: z.string().url().describe("The URL to start crawling from"),
  render: z
    .boolean()
    .optional()
    .default(false)
    .describe("Whether to render pages with a browser (slower but handles JS)"),
});
```

### Cinder API Mapping

- **Endpoint**: `POST /v1/crawl`
- **Payload**:
  ```json
  {
    "url": "<url>",
    "render": <render>
  }
  ```

---

## 4. Get Crawl Status Tool

**Name**: `cinder_get_crawl_status`
**Description**: Check the status and results of a crawl job.

### Input Schema (Zod)

```typescript
z.object({
  id: z.string().describe("The Crawl ID returned by cinder_crawl"),
});
```

### Cinder API Mapping

- **Endpoint**: `GET /v1/crawl/:id`
