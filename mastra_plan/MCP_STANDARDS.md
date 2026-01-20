# MCP Standards & Responsiveness Guidelines

This document outlines how the Cinder MCP should follow MCP standards and implement responsive, Exa-like behavior.

## MCP Protocol Compliance

The Cinder MCP server should implement:

### 1. Tool Discovery (tools/list)

Every MCP tool must be discoverable:

```typescript
MCPServer.getToolListInfo() returns:
- Tool name (id)
- Description
- Input schema (Zod → JSON schema)
- Output schema (Zod → JSON schema)
```

**Standard Implementation**:

```typescript
const toolInfo = server.getToolInfo("cinder_search");
// Returns: {
//   id: "cinder_search",
//   name: "cinder_search",
//   description: "Search the web for multiple relevant results...",
//   inputSchema: { type: "object", properties: { ... } },
//   outputSchema: { type: "object", properties: { ... } }
// }
```

### 2. Tool Execution (tools/call)

All tools must be callable with their input schema:

```typescript
server.executeTool("cinder_search", {
  query: "svelte authentication",
  mode: "balanced",
  numResults: 10,
});
```

Returns structured output matching the outputSchema.

### 3. Error Handling

MCP requires proper error responses:

```typescript
// On error, return:
{
  isError: true,
  content: [{
    type: "text",
    text: "Descriptive error message"
  }]
}
```

## Responsive Behavior Patterns

### 1. Pagination as Standard Pattern

Following Exa MCP, pagination should be implemented consistently:

**Request**:

```json
{
  "query": "topic",
  "numResults": 10,
  "cursor": "optional_cursor_from_previous_response"
}
```

**Response**:

```json
{
  "results": [...],
  "hasMore": true,
  "nextCursor": "encoded_cursor_for_next_batch",
  "searchTime": 1234
}
```

**Model Integration**:

```typescript
// Pseudo-code showing how a model would use pagination
const page1 = await search({ query: "topic", numResults: 10 });
if (page1.hasMore) {
  const page2 = await search({
    query: "topic",
    numResults: 10,
    cursor: page1.nextCursor,
  });
}
```

### 2. Progressive Result Refinement

Tools should support refinement through parameters:

**Search**: Combine with filters progressively

```
1. search({ query: "AI frameworks" })
   → 10 generic results

2. search({
   query: "AI frameworks",
   includeDomains: ["github.com"],
   requiredText: ["TypeScript"]
   })
   → Narrowed to 5 relevant results
```

**Scrape**: Choose mode based on content type

```
1. scrape({ url: "...", mode: "smart" })
   → Auto-detection (fast)

2. If JS rendering needed, retry with:
   scrape({ url: "...", mode: "dynamic" })
   → Full JS rendering
```

### 3. Result Quality Indicators

All tools should provide quality metrics:

```typescript
// Search results include relevance scores
{
  results: [
    { url: "...", title: "...", relevance: 0.95 },
    { url: "...", title: "...", relevance: 0.87 },
    { url: "...", title: "...", relevance: 0.72 }
  ]
}

// Scrape includes load time (indicates complexity)
{
  content: "...",
  loadTime: 2500,  // 2.5s for dynamic, fast for static
  statusCode: 200  // HTTP status
}

// Crawl includes progress
{
  status: "running",
  progress: 45,        // 45% complete
  pagesFound: 247,
  pagesCrawled: 112
}
```

### 4. Response Time Optimization

Tool behavior should adapt to response time requirements:

```typescript
// Fast mode (UI)
search({ query: "...", mode: "fast" });
// Returns in < 2 seconds
// Uses cached indexes, primary sources only

// Balanced mode (default)
search({ query: "...", mode: "balanced" });
// Returns in 2-5 seconds
// Good coverage, acceptable latency

// Deep mode (research)
search({ query: "...", mode: "deep" });
// Takes 5-15 seconds
// Multiple sources, fallbacks, thorough

// Dynamic scraping
scrape({ url: "...", mode: "dynamic" });
// Takes 5-10 seconds (waits for JS)
// static mode takes 1-2 seconds (just HTML)
```

### 5. Batching Capabilities

Support combining operations for efficiency:

```typescript
// Single combined call instead of search + scrape
searchAndScrape({
  query: "topic",
  numResults: 3, // Get top 3
  mode: "balanced",
  // Auto-scrapes the 3 results
});
```

## Tool Behavior Standards

### Each Tool Should Have

#### 1. Clear Description Structure

```
Tool: cinder_search

What: Search the web for multiple relevant results
When: For discovery, research, finding pages across the web
How: Supports pagination, filtering, multiple search depths

Parameters:
- query (required): The search term
- mode (optional): fast/balanced/deep, default balanced
- cursor (optional): For pagination to next batch

Output:
- results: Array of matching pages
- hasMore: Whether more results exist
- nextCursor: For pagination
```

#### 2. Sensible Defaults

```typescript
// Users get reasonable results without tuning
search({ query: "topic" });
// Equivalent to:
// search({
//   query: "topic",
//   mode: "balanced",      // Good speed/quality tradeoff
//   numResults: 10,        // Standard page size
//   cursor: null           // First page
// })
```

#### 3. Progressive Failure Modes

```typescript
// Scrape tries smart first, falls back gracefully
scrape({ url: "...", mode: "smart" });
// 1. Try static HTML
// 2. If no content, try dynamic rendering
// 3. If timeout, return partial content
// 4. If error, return error with HTTP status
```

#### 4. Standard Timeout Handling

```typescript
// Each mode has appropriate timeout
execute timeout: {
  fast:     2000,   // 2 seconds
  balanced: 5000,   // 5 seconds
  deep:    15000    // 15 seconds
}

// Respects AbortSignal for cancellation
execute: async ({ context }, { abortSignal }) => {
  const response = await fetch(url, { signal: abortSignal });
  // Auto-handles cancellation
}
```

## Exa MCP Comparison & Alignment

| Aspect                | Cinder MCP            | Exa MCP               | Status  |
| --------------------- | --------------------- | --------------------- | ------- |
| Tool descriptions     | Clear, with use cases | Clear, with use cases | ALIGNED |
| Search modes          | fast/balanced/deep    | fast/balanced/deep    | ALIGNED |
| Pagination            | Cursor-based          | Cursor-based          | ALIGNED |
| Error handling        | Structured errors     | Structured errors     | ALIGNED |
| Response times        | Published in docs     | Published in docs     | ALIGNED |
| Filtering             | Domain/text/age       | Domain/text/age       | ALIGNED |
| Result quality scores | relevance: 0-1        | score: 0-1            | ALIGNED |
| Batching              | search_and_scrape     | search_and_scrape     | ALIGNED |

## Implementation Checklist

For each tool, verify:

- [ ] Tool has clear description with "When to Use"
- [ ] All parameters have description with examples
- [ ] Output schema is well-documented
- [ ] Error cases return proper MCP error format
- [ ] Pagination/cursor pattern is consistent
- [ ] Response times are as documented
- [ ] Sensible defaults work for common use cases
- [ ] Quality indicators provided in response
- [ ] Timeout handling is implemented
- [ ] Batching/combined operations supported where relevant

## Best Practices for Models

Teach models to:

1. **Use tool descriptions**: Read descriptions to select tools
2. **Understand pagination**: Use cursor + hasMore for comprehensive research
3. **Choose search modes**: Use fast for UI, balanced for general, deep for research
4. **Filter progressively**: Start broad, narrow down with filters
5. **Handle errors gracefully**: Retry with adjusted parameters
6. **Respect timeouts**: Don't expect 15-second responses for UI

Example model guidance:

```
When you need information:
1. Check if you have a specific URL
   → YES: Use scrape for that URL
   → NO: Continue
2. Need multiple sources?
   → YES: Use search
   → NO: You're done
3. Search returned results?
   → YES: Check hasMore
   → Use nextCursor for more if needed
4. Got good results?
   → YES: Scrape top results for details
   → NO: Refine search with filters or try deep mode
```

## Monitoring & Responsiveness

Track and maintain:

- [ ] **Response times**: Verify fast < 2s, balanced 2-5s, deep 5-15s
- [ ] **Success rates**: Aim for 99%+ on scrape, 95%+ on search
- [ ] **Error rates**: Monitor and alert on errors > 5%
- [ ] **Pagination efficiency**: Most models should use 2-3 pages max
- [ ] **API quotas**: Ensure sufficient quota for deep searches
- [ ] **Cache hit rates**: Ideally 70%+ for repeated queries

## Future Enhancements

As the MCP matures, consider:

1. **Streaming responses**: Stream results as they're found
2. **Subscriptions**: Subscribe to updates on crawled pages
3. **Advanced filters**: Date ranges, content type, language
4. **Caching**: Return cached results faster
5. **Batch operations**: Process multiple queries in parallel
6. **Rate limiting info**: Return remaining quota in headers
7. **Result grouping**: Group by domain, date, or relevance tier
