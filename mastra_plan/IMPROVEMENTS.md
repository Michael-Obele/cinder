# Plan Improvements Summary

This document outlines the key improvements made to the Mastra MCP integration plan based on analysis of Exa MCP and Mastra best practices.

## Problem Statement

The original Cinder MCP plan had several limitations:

1. **Limited Tool Guidance**: Minimal descriptions for tools - LLM models didn't have clear guidance on when to use each tool
2. **No Pagination Support**: Search results returned all-or-nothing; no ability to fetch more results for comprehensive research
3. **Basic Search**: Only query parameter, no search modes or filtering options
4. **Missing Hybrid Tools**: No combined tools (e.g., search + scrape in one call)
5. **No Structured Extraction**: No way to extract specific fields or structured data from pages

## Key Improvements

### 1. Enhanced Tool Descriptions

Each tool now includes:

- **Clear purpose statement**: What the tool does in plain language
- **"When to Use" section**: Specific scenarios where this tool is appropriate
- **Timing information**: E.g., "static: 1-2s, dynamic: 5-10s, smart: auto-detects"
- **Parameter guidance**: Detailed descriptions with examples for each parameter
- **Decision logic**: How models should choose between tools

**Example improvements:**

```typescript
// BEFORE
description: "Search the web using Cinder's search capabilities."

// AFTER
description: "Search the web for multiple relevant results. Use this for
discovery, research, and finding relevant pages. Supports pagination to
explore additional results beyond the initial set."
```

### 2. Pagination Support

**Search Tool** now supports cursor-based pagination:

```typescript
// NEW: Pagination parameters
cursor: z.string().optional()
  .describe("Pagination cursor for fetching next batch of results...")

// NEW: Pagination indicators in response
hasMore: z.boolean().describe("Whether more results are available"),
nextCursor: z.string().optional()
  .describe("Cursor for fetching next batch of results using pagination"),
```

**Use case**: A model can now search for "Svelte best practices", get 10 results, then use the cursor to fetch 10 more without re-querying.

### 3. Search Modes (Like Exa)

Added configurable search depth similar to Exa MCP:

```typescript
mode: z.enum(["fast", "balanced", "deep"])
  .optional()
  .default("balanced")
  .describe("Search depth. 'fast': < 2s. 'balanced': 2-5s. 'deep': 5-15s.");
```

This allows models to optimize for:

- **Fast mode**: Interactive/real-time use (UI, quick questions)
- **Balanced mode**: Default for most research
- **Deep mode**: Comprehensive research, multiple sources

### 4. Advanced Filtering

Search tool now supports:

```typescript
includeDomains: z.array(z.string())
  .describe("Limit to specific domains. E.g., ['github.com', 'stackoverflow.com']"),

excludeDomains: z.array(z.string())
  .describe("Exclude domains. E.g., ['reddit.com']"),

requiredText: z.array(z.string())
  .describe("Each result must contain ALL these phrases"),

maxAge: z.number()
  .describe("Only last N days. E.g., 7 for past week, 30 for past month")
```

### 5. New Tools

#### Search and Scrape (Hybrid)

Combines search + scraping in one tool call:

```typescript
searchAndScrapeTool: z.object({
  query: z.string(),
  numResults: z.number().default(3), // How many to scrape
  mode: z.enum(["fast", "balanced"]),
  scrapeMode: z.enum(["smart", "static", "dynamic"]),
});
```

Use case: "Research Svelte + get the content from top 3 results" in one call.

#### Structured Data Extraction

New `cinder_extract` tool for LLM-guided extraction:

```typescript
extractTool: z.object({
  urls: z.array(z.string().url()),
  prompt: z.string(), // "Extract product name, price, rating"
  schema: z.object({}), // Optional structured output schema
  mode: z.enum(["smart", "static", "dynamic"]),
});
```

Use case: Build datasets from web content, extract prices/contacts/specifications.

### 6. Tool Decision Tree

Added clear guidance for model tool selection:

```
START: Do you need content?
├─ YES: Do you have a specific URL?
│   ├─ YES: Use SCRAPE (for single URL content)
│   └─ NO: Do you need multiple pages?
│       ├─ YES: Use SEARCH first, then SCRAPE top results
│       └─ NO: Use SEARCH with pagination
├─ NO: Do you need to explore a whole site?
│   ├─ YES: Use CRAWL
│   └─ NO: Use SEARCH
```

This decision tree helps models automatically select the most efficient tool.

### 7. Crawl Improvements

Enhanced crawl tool with:

```typescript
// NEW: Depth and page limits
maxDepth: z.number(),
maxPages: z.number(),

// NEW: Pattern-based filtering
includePatterns: z.array(z.string()),  // /blog.*, /docs.*
excludePatterns: z.array(z.string()),  // /admin.*, .*logout.*

// NEW: Crawl status pagination
cursorSupport: true,  // For fetching results from large crawls
```

### 8. Response Format Flexibility

Scrape tool now supports multiple output formats:

```typescript
returnFormat: z.enum(["markdown", "html", "json"])
  .optional()
  .default("markdown")
  .describe(
    "Return format: 'markdown' for readability, 'html' for structure, 'json' for data",
  );
```

Benefits:

- **Markdown**: Human-readable, good for summaries
- **HTML**: Structure preservation for layout-dependent content
- **JSON**: Structured data extraction

### 9. Comprehensive Error Handling

Implementation guide now includes proper error handling for:

- Network timeouts (30s default)
- Rate limiting (429)
- HTTP errors (404, 500)
- Invalid parameters
- Empty results

Example:

```typescript
if (response.status === 429) {
  throw new Error("Rate limited. Please wait before retrying.");
}
```

### 10. Performance Considerations

Documentation now includes:

- **Search mode timing**: fast < 2s, balanced 2-5s, deep 5-15s
- **Pagination strategy**: How to efficiently batch large result sets
- **Timeout configuration**: Different timeouts per search mode
- **Caching guidelines**: When to cache results locally

## Implementation Benefits

### For AI Models

1. **Better Tool Selection**: Clear descriptions help models pick the right tool
2. **Efficient Research**: Pagination enables deep research without API waste
3. **Flexible Workflows**: Hybrid tools combine operations efficiently
4. **Structured Output**: Extract specific data needed, not just raw content

### For Users

1. **Faster Research**: Search modes optimize for speed vs. quality
2. **Comprehensive Results**: Pagination lets you explore beyond top 10
3. **Precise Filtering**: Domain and text filtering narrow down results
4. **Flexible Formats**: Choose output format for your use case

### For Developers

1. **Clear Architecture**: Decision tree and descriptions guide tool design
2. **Error Resilience**: Comprehensive error handling patterns
3. **Pagination Pattern**: Standard cursor-based pagination across tools
4. **Documentation**: Each tool has clear "when to use" guidance

## Migration Path from Original Plan

1. **Phase 1**: Add pagination support to search tool
2. **Phase 2**: Add search modes and filtering
3. **Phase 3**: Enhance tool descriptions with decision guidance
4. **Phase 4**: Add hybrid tools (search_and_scrape)
5. **Phase 5**: Add extraction tool
6. **Phase 6**: Full testing and documentation

## Comparison with Exa MCP

| Feature           | Original    | Improved                | Exa               |
| ----------------- | ----------- | ----------------------- | ----------------- |
| Tool descriptions | Basic       | Detailed with use cases | Detailed          |
| Search modes      | 1 (default) | 3 (fast/balanced/deep)  | Multiple          |
| Pagination        | None        | Cursor-based            | Cursor-based      |
| Filtering         | None        | Domain, text, age       | Domain, text, age |
| Hybrid tools      | None        | search_and_scrape       | search_and_scrape |
| Extraction        | None        | LLM-guided              | LLM-guided        |
| Error handling    | Basic       | Comprehensive           | Comprehensive     |

## Files Updated

1. **tools.md**: Complete rewrite with enhanced descriptions and pagination support
2. **implementation.md**: Detailed implementation guide with code examples
3. **checklist.md**: Comprehensive implementation checklist with phases
4. **README.md**: Overview of improvements (this document)

## Next Steps

1. Review the improved tool specifications in `tools.md`
2. Follow the implementation guide in `implementation.md`
3. Use the checklist in `checklist.md` to track progress
4. Test with the Cinder API
5. Iterate based on real-world usage patterns
