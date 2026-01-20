# Implementation Checklist - Improved

## Phase 1: Project Setup

- [ ] Create `mastra` directory at project root
- [ ] Initialize `package.json` with Mastra dependencies
- [ ] Configure `tsconfig.json` for TypeScript support
- [ ] Install dependencies:
  - [ ] `@mastra/core` - Core Mastra framework
  - [ ] `@mastra/mcp` - MCP server support
  - [ ] `zod` - Schema validation
  - [ ] `typescript` - TypeScript compiler
  - [ ] `tsx` or `ts-node` - TypeScript execution

---

## Phase 2: Tool Implementation

### Basic Tools

- [ ] **Scrape Tool** (`cinder_scrape`)
  - [ ] Input schema with mode, metadata, format options
  - [ ] Output schema with content, metadata, load time
  - [ ] API integration to `/v1/scrape`
  - [ ] Error handling for network/HTTP errors
  - [ ] Test against Cinder API

- [ ] **Search Tool** (`cinder_search`)
  - [ ] Input schema with pagination support
  - [ ] Pagination cursor handling
  - [ ] Search modes (fast/balanced/deep)
  - [ ] Domain filtering options
  - [ ] Output schema with `hasMore` and `nextCursor`
  - [ ] Test pagination workflow
  - [ ] Test against Cinder API

- [ ] **Crawl Tool** (`cinder_crawl`)
  - [ ] Input schema with depth, page limits, patterns
  - [ ] Output includes crawlId for status tracking
  - [ ] Pattern-based inclusion/exclusion
  - [ ] Test against Cinder API

- [ ] **Crawl Status Tool** (`cinder_get_crawl_status`)
  - [ ] Input schema with cursor for pagination
  - [ ] Output schema with progress percentage
  - [ ] Pagination support for large result sets
  - [ ] Test status polling workflow
  - [ ] Test pagination of crawled pages

### Advanced Tools (Optional)

- [ ] **Search and Scrape** (`cinder_search_and_scrape`)
  - [ ] Combine search + top result scraping
  - [ ] Batch processing of results
  - [ ] Error handling per URL

- [ ] **Extract Tool** (`cinder_extract`)
  - [ ] LLM-guided data extraction
  - [ ] Schema validation
  - [ ] Structured output formatting

---

## Phase 3: Tool Descriptions & Documentation

For each tool, implement:

- [ ] Clear, concise tool description (what it does)
- [ ] "When to Use" section in description
- [ ] Parameter descriptions with examples
  - [ ] E.g., "E.g., 'how to deploy Svelte to Vercel' instead of 'deploy'"
  - [ ] E.g., "E.g., ['github.com', 'stackoverflow.com']"
- [ ] Performance/timing information
  - [ ] E.g., "'static': 1-2s, 'dynamic': 5-10s"
- [ ] Describe output structure and meaning

---

## Phase 4: MCP Server Configuration

- [ ] **Server Setup** (`mcp-server.ts`)
  - [ ] Create MCPServer instance
  - [ ] Register all tools
  - [ ] Set metadata (name, version, description)
  - [ ] Add tool decision tree documentation

- [ ] **Transport Configuration**
  - [ ] Choose transport: Stdio or SSE
  - [ ] For Stdio: configure command startup
  - [ ] For SSE/HTTP: setup HTTP server integration
  - [ ] Configure request/response handling

- [ ] **Entry Point** (`index.ts`)
  - [ ] Export configured server
  - [ ] Handle server lifecycle (startup/shutdown)
  - [ ] Proper error handling and logging

---

## Phase 5: HTTP Server Integration (if using SSE)

- [ ] Create HTTP server
- [ ] Route `/mcp` endpoint for SSE
- [ ] Route `/mcp-message` for client messages
- [ ] Implement error handling
- [ ] Add CORS headers if needed
- [ ] Configure port and environment

---

## Phase 6: Pagination Implementation

- [ ] **Search Pagination**
  - [ ] Implement cursor passing from `nextCursor`
  - [ ] Handle `hasMore` flag correctly
  - [ ] Test with large result sets (50+ results)
  - [ ] Verify result consistency across pages

- [ ] **Crawl Results Pagination**
  - [ ] Implement cursor-based page fetching
  - [ ] Handle large crawls (100+ pages)
  - [ ] Verify result ordering maintained

---

## Phase 7: Error Handling & Resilience

For each tool, implement handling for:

- [ ] Network timeouts (30s default)
- [ ] HTTP status codes (404, 429, 500, etc.)
- [ ] Rate limiting (429) with error message
- [ ] Invalid input parameters
- [ ] Empty results (graceful handling)
- [ ] Partial failures in batch operations

---

## Phase 8: Testing

### Unit Tests

- [ ] Test each tool independently
- [ ] Test input validation
- [ ] Mock Cinder API responses

### Integration Tests

- [ ] Test against running Cinder API
  - [ ] `cinder_scrape` with various modes
  - [ ] `cinder_search` with pagination
  - [ ] `cinder_crawl` with status checking
  - [ ] `cinder_get_crawl_status` pagination

### E2E Tests

- [ ] Test full research workflow:
  1. Search for results
  2. Use cursor to get next page
  3. Scrape selected results
  4. Verify content quality

- [ ] Test crawl workflow:
  1. Start crawl
  2. Poll status
  3. Fetch paginated results
  4. Verify page discovery

### Performance Tests

- [ ] Test search modes (fast vs. deep timing)
- [ ] Test pagination with large result sets
- [ ] Measure API response times

---

## Phase 9: Documentation

- [ ] **README** updates
  - [ ] Installation instructions
  - [ ] Configuration examples
  - [ ] Usage examples
  - [ ] Tool selection guide

- [ ] **Tool Documentation**
  - [ ] Comprehensive tool descriptions
  - [ ] Input/output schema documentation
  - [ ] Examples for each tool
  - [ ] Common use cases

- [ ] **API Documentation**
  - [ ] Endpoint mapping to Cinder API
  - [ ] Rate limiting info
  - [ ] Pagination format explanation
  - [ ] Error codes and handling

---

## Phase 10: Deployment

- [ ] **Environment Configuration**
  - [ ] Set Cinder API URL via env var
  - [ ] Configure timeouts
  - [ ] Set up logging

- [ ] **Docker Setup**
  - [ ] Create Dockerfile
  - [ ] Build and test image
  - [ ] Document deployment

- [ ] **Leapcell Deployment** (if applicable)
  - [ ] Configure service
  - [ ] Set environment variables
  - [ ] Deploy and verify

- [ ] **Monitoring**
  - [ ] Logging setup
  - [ ] Error tracking
  - [ ] Performance monitoring

---

## Phase 11: Integration with Clients

- [ ] **MCP Client Integration**
  - [ ] Test with Cursor IDE
  - [ ] Test with Claude Desktop
  - [ ] Verify tool discovery

- [ ] **Mastra Agent Integration**
  - [ ] Create example agent using tools
  - [ ] Test tool selection logic
  - [ ] Demonstrate pagination workflow

---

## Success Criteria

- [ ] All tools discoverable and callable via MCP
- [ ] Tool descriptions guide model selection effectively
- [ ] Pagination working smoothly across 100+ results
- [ ] Error messages are helpful and actionable
- [ ] API calls are performant (< 30s for deep search)
- [ ] Integration tests pass against Cinder API
- [ ] Documentation is clear and complete
