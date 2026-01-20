# Implementation Plan - Improved

## Overview

Build a Mastra MCP server that exposes Cinder's web scraping, searching, and crawling capabilities as standardized tools with responsive, pagination-enabled behavior similar to Exa MCP.

## Key Improvements from Original Plan

1. **Enhanced Tool Descriptions**: Each tool now has clear "When to Use" sections and parameter descriptions to help AI models select the right tool
2. **Pagination Support**: Search and crawl status tools now support offset-based pagination for exploring large result sets
3. **Search Modes**: Added fast/balanced/deep modes for controlling search depth (like Exa)
4. **Tool Chaining**: Support for combined operations (e.g., search_and_scrape)
5. **Structured Data Extraction**: New extract tool for LLM-guided data extraction
6. **Tool Decision Tree**: Guidance for models on which tool to use for different scenarios

---

## 1. Development Setup

See [Setup Guide](./setup.md).

---

## 2. Tool Implementation in Mastra

Each tool is created using Mastra's `createTool()` function with comprehensive descriptions.

### 2.1 Scrape Tool

```typescript
import { createTool } from "@mastra/core/tools";
import { z } from "zod";

export const scrapeTool = createTool({
  id: "cinder_scrape",
  description:
    "Extract detailed content from a specific URL. Use this when you need to access the complete text, metadata, or structured data from a known webpage. Supports automatic JavaScript rendering for dynamic content.",

  inputSchema: z.object({
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
        "Scraping mode strategy. 'static': fast HTML (1-2s). 'dynamic': renders JS (5-10s). 'smart': auto-detects.",
      ),
    includeMetadata: z
      .boolean()
      .optional()
      .default(true)
      .describe(
        "Include page metadata like title, description, images, structured data.",
      ),
    returnFormat: z
      .enum(["markdown", "html", "json"])
      .optional()
      .default("markdown")
      .describe(
        "Return format: 'markdown' for readability, 'html' for structure, 'json' for data.",
      ),
  }),

  outputSchema: z.object({
    url: z.string(),
    title: z.string().optional(),
    content: z.string(),
    metadata: z
      .object({
        description: z.string().optional(),
        keywords: z.array(z.string()).optional(),
        images: z.array(z.string()).optional(),
        author: z.string().optional(),
        publishDate: z.string().optional(),
      })
      .optional(),
    statusCode: z.number(),
    loadTime: z.number(),
  }),

  execute: async ({ context }) => {
    const response = await fetch("https://cinder-api.example.com/v1/scrape", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({
        url: context.url,
        mode: context.mode,
        includeMetadata: context.includeMetadata,
        returnFormat: context.returnFormat,
      }),
    });

    if (!response.ok) throw new Error(`Scrape failed: ${response.statusText}`);
    return response.json();
  },
});
```

### 2.2 Search Tool with Pagination

```typescript
export const searchTool = createTool({
  id: "cinder_search",
  description:
    "Search the web for multiple relevant results. Use this for discovery, research, and finding pages. Supports pagination to explore additional results beyond the initial set.",

  inputSchema: z.object({
    query: z.string().describe("The search query. Be specific with keywords."),
    mode: z
      .enum(["fast", "balanced", "deep"])
      .optional()
      .default("balanced")
      .describe("Search depth. 'fast': < 2s. 'balanced': 2-5s. 'deep': 5-15s."),
    numResults: z
      .number()
      .int()
      .min(1)
      .max(100)
      .optional()
      .default(10)
      .describe(
        "Number of results to return. Default 10, useful for 20-50 for research.",
      ),
    offset: z
      .number()
      .int()
      .min(0)
      .optional()
      .default(0)
      .describe(
        "Number of results to skip. Default 0. Use nextOffset from previous response for pagination.",
      ),
    includeDomains: z
      .array(z.string())
      .optional()
      .describe(
        "Limit to specific domains. E.g., ['github.com', 'stackoverflow.com'].",
      ),
    excludeDomains: z
      .array(z.string())
      .optional()
      .describe("Exclude domains. E.g., ['reddit.com']."),
    requiredText: z
      .array(z.string())
      .optional()
      .describe("Results must contain ALL these phrases."),
    maxAge: z
      .number()
      .optional()
      .describe("Only last N days. E.g., 7 for past week, 30 for past month."),
  }),

  outputSchema: z.object({
    query: z.string(),
    results: z.array(
      z.object({
        id: z.string(),
        title: z.string(),
        url: z.string(),
        domain: z.string(),
        summary: z.string(),
        relevance: z.number(),
        publishedDate: z.string().optional(),
      }),
    ),
    hasMore: z
      .boolean()
      .describe("Whether more results available for pagination"),
    nextOffset: z
      .number()
      .optional()
      .describe(
        "Suggested offset for next batch: offset + numResults. Use with same query and numResults.",
      ),
    searchTime: z.number(),
  }),

  execute: async ({ context }) => {
    const response = await fetch("https://cinder-api.example.com/v1/search", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({
        query: context.query,
        mode: context.mode,
        numResults: context.numResults,
        offset: context.offset,
        includeDomains: context.includeDomains,
        excludeDomains: context.excludeDomains,
        requiredText: context.requiredText,
        maxAge: context.maxAge,
      }),
    });

    if (!response.ok) throw new Error(`Search failed: ${response.statusText}`);
    return response.json();
  },
});
```

### 2.3 Crawl Tool with Status Tracking

```typescript
export const crawlTool = createTool({
  id: "cinder_crawl",
  description:
    "Start a background crawl job to systematically explore pages from a domain. Use for site analysis, competitive research, or discovering all pages within a website.",

  inputSchema: z.object({
    url: z
      .string()
      .url()
      .describe(
        "Starting URL for crawling. Crawl follows links within the same domain.",
      ),
    render: z
      .boolean()
      .optional()
      .default(false)
      .describe(
        "Render with browser. Set true for JS-heavy sites (slower). Default false for speed.",
      ),
    maxDepth: z
      .number()
      .int()
      .optional()
      .default(2)
      .describe("Max crawl depth. 1=start URL, 2=one level of links, etc."),
    maxPages: z
      .number()
      .int()
      .optional()
      .default(100)
      .describe("Max pages to crawl. Prevents runaway crawls on large sites."),
    includePatterns: z
      .array(z.string())
      .optional()
      .describe(
        "Only crawl URLs matching patterns. E.g., ['/blog.*', '/docs.*'].",
      ),
    excludePatterns: z
      .array(z.string())
      .optional()
      .describe(
        "Skip URLs matching patterns. E.g., ['/admin.*', '.*logout.*'].",
      ),
  }),

  outputSchema: z.object({
    crawlId: z.string(),
    url: z.string(),
    status: z.enum(["queued", "running", "completed", "failed"]),
    pagesFound: z.number(),
    pagesCrawled: z.number(),
    createdAt: z.string(),
  }),

  execute: async ({ context }) => {
    const response = await fetch("https://cinder-api.example.com/v1/crawl", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({
        url: context.url,
        render: context.render,
        maxDepth: context.maxDepth,
        maxPages: context.maxPages,
        includePatterns: context.includePatterns,
        excludePatterns: context.excludePatterns,
      }),
    });

    if (!response.ok) throw new Error(`Crawl failed: ${response.statusText}`);
    return response.json();
  },
});
```

### 2.4 Crawl Status Tool with Pagination

```typescript
export const crawlStatusTool = createTool({
  id: "cinder_get_crawl_status",
  description:
    "Check the progress and retrieve results from a background crawl job. Monitor long-running crawls or fetch results when ready.",

  inputSchema: z.object({
    id: z.string().describe("The Crawl ID returned by cinder_crawl"),
    cursor: z
      .string()
      .optional()
      .describe("Pagination cursor for next batch of crawled pages"),
    limit: z
      .number()
      .optional()
      .default(50)
      .describe("Number of results per page"),
  }),

  outputSchema: z.object({
    crawlId: z.string(),
    status: z.enum(["queued", "running", "completed", "failed"]),
    pagesFound: z.number(),
    pagesCrawled: z.number(),
    progress: z.number().describe("Progress percentage 0-100"),
    pages: z
      .array(
        z.object({
          url: z.string(),
          title: z.string().optional(),
          statusCode: z.number(),
          contentLength: z.number().optional(),
        }),
      )
      .optional(),
    nextCursor: z.string().optional().describe("For fetching next batch"),
    completedAt: z.string().optional(),
    error: z.string().optional(),
  }),

  execute: async ({ context }) => {
    const params = new URLSearchParams();
    if (context.cursor) params.append("cursor", context.cursor);
    if (context.limit) params.append("limit", context.limit.toString());

    const response = await fetch(
      `https://cinder-api.example.com/v1/crawl/${context.id}?${params}`,
      { method: "GET" },
    );

    if (!response.ok)
      throw new Error(`Status check failed: ${response.statusText}`);
    return response.json();
  },
});
```

---

## 3. MCPServer Setup

Create the MCP server to expose these tools:

```typescript
import { MCPServer } from "@mastra/mcp";
import { scrapeTool, searchTool, crawlTool, crawlStatusTool } from "./tools";

const server = new MCPServer({
  name: "Cinder Web Tools",
  version: "1.0.0",
  description: "Web scraping, searching, and crawling capabilities",

  tools: {
    cinder_scrape: scrapeTool,
    cinder_search: searchTool,
    cinder_crawl: crawlTool,
    cinder_get_crawl_status: crawlStatusTool,
  },
});

// Start SSE transport for MCP clients
export default server;
```

---

## 4. HTTP Server Integration

Integrate with an HTTP server to expose MCP via SSE:

```typescript
import http from "http";
import { server } from "./mcp-server";

const PORT = process.env.PORT || 3000;

const httpServer = http.createServer(async (req, res) => {
  if (req.url === "/mcp" && req.method === "POST") {
    // Handle MCP SSE endpoint
    await server.startSSE({
      url: new URL(req.url || "", `http://localhost:${PORT}`),
      ssePath: "/mcp",
      messagePath: "/mcp-message",
      req,
      res,
    });
  } else {
    res.writeHead(404);
    res.end("Not found");
  }
});

httpServer.listen(PORT, () => {
  console.log(`Cinder MCP Server running on http://localhost:${PORT}`);
});
```

---

## 5. Pagination Example

Handle pagination in your agent:

```typescript
import { Agent } from "@mastra/core";
import { searchTool } from "./tools";

const agent = new Agent({
  name: "Research Agent",
  instructions:
    "Use search tool to find information. If hasMore is true, fetch next batch using cursor.",
  tools: { search: searchTool },
  model: openai("gpt-4"),
});

// Example: pagination flow
async function researchTopic(topic: string) {
  const results = [];
  let cursor: string | undefined;
  let hasMore = true;

  while (hasMore) {
    const response = await agent.generate(
      cursor
        ? `Continue researching '${topic}' using pagination cursor.`
        : `Research '${topic}' and find 10 relevant pages.`,
      {
        context: { cursor, topic },
      },
    );

    results.push(...response.results);
    cursor = response.nextCursor;
    hasMore = response.hasMore;
  }

  return results;
}
```

---

## 6. Tool Decision Logic

Agents should use this logic to select the right tool:

```
START
├─ Do you have a specific URL?
│  ├─ YES → Use SCRAPE
│  └─ NO → Continue
├─ Do you need multiple pages?
│  ├─ YES → Use SEARCH with pagination
│  └─ NO → Continue
└─ Do you need to explore a whole site?
   ├─ YES → Use CRAWL + GET_CRAWL_STATUS
   └─ NO → Use SEARCH
```

---

## 7. Deployment

### Option 1: Leapcell (Current)

- Deploy Mastra MCP as a service on Leapcell
- Call Cinder Go Backend via API

### Option 2: Docker

```dockerfile
FROM node:18-alpine
WORKDIR /app
COPY . .
RUN npm ci
CMD ["npm", "start"]
```

---

## 8. Performance Considerations

1. **Search Modes**: Use `fast` mode for UI interactions, `deep` for research
2. **Pagination**: Always check `hasMore` before calling with next `cursor`
3. **Crawling**: Set reasonable `maxPages` to avoid excessive API usage
4. **Caching**: Implement result caching at tool level for repeated queries
5. **Timeouts**: Set appropriate timeouts per search mode (fast < balanced < deep)

---

## 9. Error Handling

Each tool should handle:

- Network errors (timeout, connection refused)
- HTTP errors (404, 429 rate limit, 500 server error)
- Invalid parameters
- Empty results

Example:

```typescript
execute: async ({ context }) => {
  try {
    const response = await fetch(url, { timeout: 30000 });
    if (response.status === 429) {
      throw new Error("Rate limited. Please wait before retrying.");
    }
    if (!response.ok) {
      throw new Error(`API error: ${response.status}`);
    }
    return response.json();
  } catch (error) {
    if (error.name === "AbortError") {
      throw new Error("Request timeout");
    }
    throw error;
  }
};
```
