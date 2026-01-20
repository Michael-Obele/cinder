# Tools Specification - Improved Version

This document defines the MCP tools to be implemented in the Mastra application.

## Overview

The Cinder MCP provides a comprehensive set of tools for web scraping, searching, and crawling. Each tool is designed with clear descriptions to help AI models decide when and how to use them, similar to Exa's MCP implementation.

### Tool Selection Guidelines for AI Models

- Use **Search** for finding multiple relevant pages across the web
- Use **Scrape** when you need detailed content from a specific URL
- Use **Crawl** for systematic exploration of related pages from a domain
- Use **Search + Scrape** combination for researching topics deeply

---

## 1. Scrape Tool

**Name**: `cinder_scrape`

**Description**: Extract detailed content from a specific URL. Use this when you need to access the complete text, metadata, or structured data from a known webpage. Supports automatic JavaScript rendering for dynamic content.

### When to Use

- You have a specific URL and need its full content
- You need structured data extraction from a page
- The target page uses JavaScript that needs to be rendered
- You want metadata (title, description, headings) along with content

### Input Schema (Zod)

```typescript
z.object({
  url: z
    .string()
    .url()
    .describe(
      "The URL to scrape. Must be a complete, valid web address (http/https).",
    ),
  mode: z
    .enum(["smart", "static", "dynamic"])
    .optional()
    .default("smart")
    .describe(
      "Scraping mode strategy. 'static': fast HTML fetching (best for static sites, 1-2s). 'dynamic': renders JavaScript (slower but required for SPA/interactive sites, 5-10s). 'smart': automatically detects and chooses the best approach based on content type.",
    ),
  includeMetadata: z
    .boolean()
    .optional()
    .default(true)
    .describe(
      "Include page metadata like title, description, images, and structured data (JSON-LD, OpenGraph).",
    ),
  returnFormat: z
    .enum(["markdown", "html", "json"])
    .optional()
    .default("markdown")
    .describe(
      "Return format: 'markdown' for readability, 'html' for structure preservation, 'json' for structured data extraction.",
    ),
});
```

### Output Schema

```typescript
z.object({
  url: z.string().describe("The requested URL"),
  title: z.string().optional().describe("Page title"),
  content: z.string().describe("Main page content in requested format"),
  metadata: z
    .object({
      description: z.string().optional(),
      keywords: z.array(z.string()).optional(),
      images: z.array(z.string()).optional(),
      author: z.string().optional(),
      publishDate: z.string().optional(),
    })
    .optional(),
  statusCode: z.number().describe("HTTP status code"),
  loadTime: z.number().describe("Time taken to load in milliseconds"),
});
```

### Cinder API Mapping

- **Endpoint**: `POST /v1/scrape`
- **Payload**:
  ```json
  {
    "url": "<url>",
    "mode": "<mode>",
    "includeMetadata": true,
    "returnFormat": "markdown"
  }
  ```

---

## 2. Search Tool

**Name**: `cinder_search`

**Description**: Search the web for multiple relevant results. Use this for discovery, research, and finding relevant pages. Supports pagination to explore additional results beyond the initial set.

### When to Use

- You need to find multiple relevant pages for a topic
- You're researching a subject and need various perspectives
- You want to discover new information across different sources
- You need pagination to explore more results

### Key Features

- **Pagination Support**: Get more results by using cursor-based pagination
- **Search Modes**: Control search depth (fast vs. comprehensive)
- **Result Count**: Configurable number of results (1-100)
- **Filtering**: Optional domain filtering and content requirements

### Input Schema (Zod)

```typescript
z.object({
  query: z
    .string()
    .describe(
      "The search query. Be specific and use relevant keywords. E.g., 'how to deploy Svelte app to Vercel' instead of just 'deploy'.",
    ),
  mode: z
    .enum(["fast", "balanced", "deep"])
    .optional()
    .default("balanced")
    .describe(
      "Search depth. 'fast': quick results from primary sources (< 2s). 'balanced': moderate depth with good coverage (2-5s). 'deep': comprehensive search with multiple sources and fallbacks (5-15s).",
    ),
  numResults: z
    .number()
    .int()
    .min(1)
    .max(100)
    .optional()
    .default(10)
    .describe(
      "Number of search results to return. Default 10, useful for getting 20-50 results for comprehensive research.",
    ),
  offset: z
    .number()
    .int()
    .min(0)
    .optional()
    .default(0)
    .describe(
      "Number of results to skip. Default 0. Use 10 for second page if numResults=10. Useful for pagination.",
    ),
  includeDomains: z
    .array(z.string())
    .optional()
    .describe(
      "Limit results to specific domains. E.g., ['github.com', 'stackoverflow.com'] for tech questions.",
    ),
  excludeDomains: z
    .array(z.string())
    .optional()
    .describe(
      "Exclude results from specific domains. E.g., ['reddit.com'] to avoid social media.",
    ),
  requiredText: z
    .array(z.string())
    .optional()
    .describe(
      "Each result must contain ALL of these text phrases. Useful for precise queries.",
    ),
  maxAge: z
    .number()
    .optional()
    .describe(
      "Only return results from last N days. E.g., 7 for past week, 30 for past month. Useful for current events.",
    ),
});
```

### Output Schema

```typescript
z.object({
  query: z.string().describe("The search query used"),
  results: z
    .array(
      z.object({
        id: z.string().describe("Unique result identifier"),
        title: z.string().describe("Page title"),
        url: z.string().describe("Result URL"),
        domain: z.string().describe("Domain name"),
        summary: z.string().describe("Brief summary of the page content"),
        relevance: z.number().describe("Relevance score 0-1"),
        publishedDate: z
          .string()
          .optional()
          .describe("When the page was published"),
      }),
    )
    .describe("Array of search results"),
  hasMore: z.boolean().describe("Whether more results are available beyond current offset"),
  nextOffset: z
    .number()
    .optional()
    .describe("Suggested offset for fetching next batch: offset + numResults. Use with same numResults for consistency."),
  searchTime: z.number().describe("Time taken to search in milliseconds"),
});
```

### Cinder API Mapping

- **Endpoint**: `POST /v1/search`
- **Payload**:
  ```json
  {
    "query": "<query>",
    "mode": "balanced",
    "numResults": 10,
    "offset": 0,
    "includeDomains": [],
    "excludeDomains": [],
    "requiredText": [],
    "maxAge": null
  }
  ```

---

## 3. Scrape with Search Integration

**Name**: `cinder_search_and_scrape`

**Description**: Combine search and scrape in a single tool. This searches for relevant pages and scrapes the top results for comprehensive information gathering. Useful for research tasks.

### When to Use

- You need comprehensive information on a topic (search + content)
- You want quick research without multiple tool calls
- You need to verify information across multiple sources
- Time is critical and you want batched operations

### Input Schema (Zod)

```typescript
z.object({
  query: z.string().describe("The search query"),
  numResults: z
    .number()
    .optional()
    .default(3)
    .describe("How many top results to scrape"),
  mode: z
    .enum(["fast", "balanced"])
    .optional()
    .default("balanced")
    .describe("Search depth mode"),
  scrapeMode: z
    .enum(["smart", "static", "dynamic"])
    .optional()
    .default("smart")
    .describe("How to scrape the pages"),
});
```

### Cinder API Mapping

- Executes `POST /v1/search` followed by `POST /v1/scrape` for top results

---

## 4. Crawl Tool

**Name**: `cinder_crawl`

**Description**: Start a background crawl job to systematically explore all pages from a domain. Use this for comprehensive site analysis, competitive research, or discovering all pages within a website.

### When to Use

- You need to explore all pages on a website
- You're performing competitive analysis
- You need to understand site structure and content
- You want comprehensive data from a domain (use with caution for large sites)

### Input Schema (Zod)

```typescript
z.object({
  url: z
    .string()
    .url()
    .describe(
      "The starting URL for crawling. Crawl will follow links within the same domain.",
    ),
  render: z
    .boolean()
    .optional()
    .default(false)
    .describe(
      "Whether to render pages with a browser. Set to true for JavaScript-heavy sites, but crawling will be slower. Default false for speed.",
    ),
  maxDepth: z
    .number()
    .int()
    .optional()
    .default(2)
    .describe(
      "Maximum depth to crawl (1=just start URL, 2=one level of links, etc). Limit this for large sites.",
    ),
  maxPages: z
    .number()
    .int()
    .optional()
    .default(100)
    .describe(
      "Maximum number of pages to crawl. Prevents runaway crawls on large sites.",
    ),
  includePatterns: z
    .array(z.string())
    .optional()
    .describe(
      "Only crawl URLs matching these patterns (regex). E.g., ['/blog.*', '/docs.*'] to limit crawl scope.",
    ),
  excludePatterns: z
    .array(z.string())
    .optional()
    .describe(
      "Skip URLs matching these patterns (regex). E.g., ['/admin.*', '.*logout.*'] to avoid sensitive areas.",
    ),
});
```

### Output Schema

```typescript
z.object({
  crawlId: z.string().describe("Unique crawl job identifier"),
  url: z.string().describe("Starting URL"),
  status: z
    .enum(["queued", "running", "completed", "failed"])
    .describe("Current crawl status"),
  pagesFound: z.number().describe("Number of pages discovered"),
  pagesCrawled: z.number().describe("Number of pages successfully crawled"),
  createdAt: z.string().describe("When the crawl was started"),
}).describe(
  "Crawl job information - use crawlId with cinder_get_crawl_status to check progress",
);
```

### Cinder API Mapping

- **Endpoint**: `POST /v1/crawl`
- **Payload**:
  ```json
  {
    "url": "<url>",
    "render": false,
    "maxDepth": 2,
    "maxPages": 100,
    "includePatterns": [],
    "excludePatterns": []
  }
  ```

---

## 5. Get Crawl Status Tool

**Name**: `cinder_get_crawl_status`

**Description**: Check the progress and retrieve results from a background crawl job. Use this to monitor long-running crawls or fetch results when ready.

### When to Use

- You started a crawl and want to check if it's finished
- You need to retrieve crawl results
- You want to see statistics about the crawl
- You need to get paginated crawl results

### Input Schema (Zod)

```typescript
z.object({
  id: z.string().describe("The Crawl ID returned by cinder_crawl"),
  offset: z
    .number()
    .int()
    .min(0)
    .optional()
    .default(0)
    .describe("Number of pages to skip. Default 0. Use 50 for second page if limit=50."),
  limit: z
    .number()
    .int()
    .min(1)
    .max(100)
    .optional()
    .default(50)
    .describe("Number of crawled pages per batch. Default 50, max 100."),
});
```

### Output Schema

```typescript
z.object({
  crawlId: z.string(),
  status: z.enum(["queued", "running", "completed", "failed"]),
  pagesFound: z.number(),
  pagesCrawled: z.number(),
  progress: z.number().describe("Progress percentage (0-100)"),
  pages: z
    .array(
      z.object({
        url: z.string(),
        title: z.string().optional(),
        statusCode: z.number(),
        contentLength: z.number().optional(),
      }),
    )
    .optional()
    .describe("Paginated list of crawled pages"),
  nextOffset: z
    .number()
    .optional()
    .describe("Suggested offset for next batch: offset + returned pages count"),
  completedAt: z.string().optional(),
  error: z.string().optional().describe("Error message if crawl failed"),
});
```

### Cinder API Mapping

- **Endpoint**: `GET /v1/crawl/:id` with optional query params for pagination

---

## 6. Extract Tool (Structured Data)

**Name**: `cinder_extract`

**Description**: Extract specific structured data or information from URLs using LLM-guided extraction. Use this when you need specific fields or structured information from a page.

### When to Use

- You need specific data points from a page (price, address, contact info)
- You want to extract structured information into a schema
- You need to convert unstructured web content into structured format
- You're building datasets from web content

### Input Schema (Zod)

```typescript
z.object({
  urls: z.array(z.string().url()).describe("URLs to extract data from"),
  prompt: z
    .string()
    .describe(
      "Natural language description of what to extract. E.g., 'Extract product name, price, and customer rating'",
    ),
  schema: z
    .object({})
    .optional()
    .describe("Optional JSON schema defining the expected output structure"),
  mode: z
    .enum(["smart", "static", "dynamic"])
    .optional()
    .default("smart")
    .describe("Content extraction mode"),
});
```

### Cinder API Mapping

- **Endpoint**: `POST /v1/extract` (new endpoint)

---

## Tool Decision Tree for AI Models

Use this guide to help models select the right tool:

```
START: Do you need content?
├─ YES: Do you have a specific URL?
│   ├─ YES: Use SCRAPE (for single URL content)
│   └─ NO: Do you need to explore multiple pages?
│       ├─ YES: Use SEARCH first, then SCRAPE top results
│       └─ NO: Use SEARCH with pagination
├─ NO: Do you need to explore a whole site?
│   ├─ YES: Use CRAWL
│   └─ NO: Use SEARCH
```

---

## Pagination Strategy

Most tools support pagination via offset-based parameters:

1. Initial request uses `offset=0` (default)
2. Response includes `hasMore` flag and `nextOffset` suggestion
3. For next page, use `offset=nextOffset` with same parameters
4. Continue until `hasMore` is false
5. Each tool manages pagination independently

Example flow:

```
1. Search with query="react hooks", numResults=10, offset=0
2. Get 10 results + hasMore=true, nextOffset=10
3. Search again with offset=10 (same query and numResults)
4. Get next 10 results + hasMore=true, nextOffset=20
5. Continue until hasMore=false
```

### Offset Formula

- **Next offset**: `nextOffset = currentOffset + numResults`
- **Previous offset**: `previousOffset = max(0, currentOffset - numResults)`
- **Max offset**: Limited by available results
