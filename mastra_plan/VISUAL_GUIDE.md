# Mastra MCP Improved Plan - Visual Guide

## Plan Structure

```
mastra_plan/
├── README.md                  # Overview of the integration
├── SUMMARY.md                 # Executive summary [NEW]
├── INDEX.md                   # This file - navigation guide [NEW]
│
├── Specifications
│   ├── tools.md              # Tool definitions [REWRITTEN]
│   └── architecture.md       # System design
│
├── Implementation
│   ├── implementation.md     # Code & patterns [REWRITTEN]
│   ├── setup.md             # Development setup
│   └── checklist.md         # Implementation tasks [REWRITTEN]
│
└── Analysis & Standards
    ├── IMPROVEMENTS.md      # What changed & why [NEW]
    └── MCP_STANDARDS.md     # Best practices [NEW]
```

## Tool Ecosystem

```
┌─────────────────────────────────────────────────────────────┐
│                    AI Agent / Model                         │
└──────────────────────┬──────────────────────────────────────┘
                       │
                       │ Uses tools via MCP
                       ↓
┌─────────────────────────────────────────────────────────────┐
│              Mastra MCP Server (Node.js)                    │
│                                                              │
│  Tools:                                                      │
│  ├─ cinder_scrape           → Extract content from URL      │
│  ├─ cinder_search           → Search web + pagination       │
│  ├─ cinder_search_and_scrape → Combined search + scrape    │
│  ├─ cinder_crawl            → Start domain crawl           │
│  ├─ cinder_get_crawl_status → Check crawl + paginate      │
│  └─ cinder_extract          → Extract structured data      │
└──────────────────────┬──────────────────────────────────────┘
                       │
                       │ REST API calls
                       ↓
┌─────────────────────────────────────────────────────────────┐
│              Cinder Go Backend                               │
│  ├─ POST /v1/scrape                                         │
│  ├─ POST /v1/search                                         │
│  ├─ POST /v1/crawl                                          │
│  └─ GET  /v1/crawl/:id                                      │
└──────────────────────┬──────────────────────────────────────┘
                       │
                       ↓
┌─────────────────────────────────────────────────────────────┐
│                  Internet / Web Pages                       │
└─────────────────────────────────────────────────────────────┘
```

## Tool Decision Tree

```
                              START
                                │
                    Need content about topic?
                          YES  /  \  NO
                            /        \
                           ↓          → Looking for all pages on site?
                      Specific URL?    YES / NO
                        YES / NO        /       \
                        /    \        /         \
                       ↓      ↓      ↓           ↓
                   SCRAPE  Search  CRAWL       END
                           │
                           ↓
                      Got results?
                        YES / NO
                        /     \
                       ↓       \
                    Check         \
                  hasMore?         \
                   YES / NO         \
                   /    \            \
                  ↓      \            \
              SCRAPE   Use            Try refined
             next with search with
             cursor    better query
```

## Tool Specifications at a Glance

```
┌─────────────────────┬──────────────┬──────────────────────────┐
│ Tool                │ Mode/Speed   │ Best For                 │
├─────────────────────┼──────────────┼──────────────────────────┤
│ scrape              │ 1-10s        │ Single URL content       │
│                     │ (auto/static/│ extraction               │
│                     │  dynamic)    │                          │
├─────────────────────┼──────────────┼──────────────────────────┤
│ search              │ fast: <2s    │ Finding multiple pages   │
│ (pagination)        │ balanced: 2-5│ with pagination          │
│                     │ deep: 5-15s  │                          │
├─────────────────────┼──────────────┼──────────────────────────┤
│ search_and_scrape   │ balanced: 3-8│ Quick research with      │
│ (new, combined)     │ s (includes  │ content from top results │
│                     │ scraping)    │                          │
├─────────────────────┼──────────────┼──────────────────────────┤
│ crawl               │ var: 10-120s │ Explore entire domain    │
│ (async)             │ (background) │ with progress tracking   │
├─────────────────────┼──────────────┼──────────────────────────┤
│ get_crawl_status    │ <1s          │ Check crawl progress     │
│ (with pagination)   │ (sync)       │ and fetch results        │
├─────────────────────┼──────────────┼──────────────────────────┤
│ extract             │ 2-10s        │ Pull structured data     │
│ (new, LLM-guided)   │ (auto/static/│ from pages               │
│                     │  dynamic)    │                          │
└─────────────────────┴──────────────┴──────────────────────────┘
```

## Pagination Pattern

```
Request 1: search({ query: "topic", numResults: 10 })
     │
     ├─ Returns: results[0-9], hasMore: true, nextCursor: "abc123"
     │
Request 2: search({ query: "topic", numResults: 10, cursor: "abc123" })
     │
     ├─ Returns: results[10-19], hasMore: true, nextCursor: "def456"
     │
Request 3: search({ query: "topic", numResults: 10, cursor: "def456" })
     │
     ├─ Returns: results[20-29], hasMore: false
     │
     └─ STOP (no more results available)
```

## Implementation Timeline

```
Week 1: Core Setup & Basic Tools
├─ Day 1-2: Setup (Phase 1)
├─ Day 3-4: Implement scrape + search (Phase 2)
└─ Day 5:   Basic testing (Phase 8a)

Week 2: Advanced Features
├─ Day 1:   Add pagination (Phase 6)
├─ Day 2-3: Implement crawl tools (Phase 2)
├─ Day 4:   Search modes + filtering (Phase 2 cont)
└─ Day 5:   Integration testing (Phase 8b)

Week 3: Polish & New Tools
├─ Day 1:   Error handling (Phase 7)
├─ Day 2-3: Implement hybrid tools (Phase 2)
├─ Day 4:   Documentation (Phase 9)
└─ Day 5:   E2E testing (Phase 8c)

Week 4: Deployment
├─ Day 1-2: Deployment setup (Phase 10)
├─ Day 3:   Client integration (Phase 11)
├─ Day 4:   Performance testing
└─ Day 5:   Production launch
```

## Key Metrics to Track

```
Performance
├─ Search:     fast < 2s, balanced 2-5s, deep 5-15s
├─ Scrape:     static 1-2s, dynamic 5-10s
├─ Crawl:      ~5 pages/sec background
└─ Extract:    2-10s depending on complexity

Reliability
├─ Success rate: >= 99% for scrape
├─ Search success: >= 95%
├─ Error rate: <= 5%
└─ Timeout rate: <= 1%

Usage Patterns
├─ Pagination: Most queries use 1-3 pages
├─ Tool distribution: ~60% search, ~30% scrape, ~10% crawl
├─ Avg response time: 2-5 seconds
└─ Error handling: 99%+ graceful error responses
```

## Search Modes Explained

```
FAST MODE (< 2 seconds)
├─ Primary search indexes only
├─ Limited geographic coverage
├─ Cached results preferred
├─ Use case: UI, quick answers
└─ Example: User typing in search box

BALANCED MODE (2-5 seconds) [DEFAULT]
├─ Standard search indexes
├─ Global coverage
├─ Mix of cached + fresh results
├─ Use case: General research, most common
└─ Example: Research blog post topic

DEEP MODE (5-15 seconds)
├─ Multiple search providers
├─ Comprehensive geographic coverage
├─ Fallback sources included
├─ Use case: Thorough research, academic
└─ Example: Competitive analysis, deep investigation
```

## Tool Workflow Examples

### Example 1: Quick Research

```
User: "What are Svelte best practices?"

Model:
1. search({
   query: "Svelte best practices",
   mode: "fast",
   numResults: 5
}) → Get 5 quick results

2. Check results, if good → Done!
   If more needed → Continue to pagination
```

### Example 2: Comprehensive Research

```
User: "Compare Svelte and React architectures"

Model:
1. search({
   query: "Svelte architecture patterns",
   mode: "balanced",
   numResults: 10
}) → Get 10 moderate results

2. search({
   query: "Svelte architecture patterns",
   mode: "balanced",
   numResults: 10,
   cursor: result.nextCursor
}) → Get next 10 if needed

3. scrape(top_5_urls) → Get detailed content
```

### Example 3: Competitive Analysis

```
User: "What pages does competitor.com have?"

Model:
1. crawl({
   url: "https://competitor.com",
   maxDepth: 3,
   maxPages: 100
}) → Start crawl, get crawlId

2. Poll status with get_crawl_status(crawlId)
   until status === "completed"

3. get_crawl_status(crawlId, cursor: null)
   → Get first 50 pages

4. get_crawl_status(crawlId, cursor: result.nextCursor)
   → Get next 50 pages if available
```

## File Size & Scope

```
Planning Documents:
├─ README.md (850 B)          Overview
├─ SUMMARY.md (6.3 KB)        Quick summary
├─ INDEX.md (4 KB)            Navigation [THIS FILE]
├─ IMPROVEMENTS.md (8.3 KB)   What changed
└─ MCP_STANDARDS.md (8.7 KB)  Standards

Specifications:
├─ tools.md (13.5 KB)         6 tools fully spec'd
└─ architecture.md (1.4 KB)   System design

Implementation:
├─ implementation.md (13.4 KB) Code examples + patterns
├─ checklist.md (6.6 KB)      11 phases, 50+ tasks
└─ setup.md (960 B)           Dev setup

TOTAL: ~59 KB of comprehensive documentation
```

## Quick Links by Use Case

**"I need to implement this now"**

1. Read: [setup.md](setup.md)
2. Follow: [implementation.md](implementation.md)
3. Track: [checklist.md](checklist.md)

**"What's different from before?"**

1. Read: [IMPROVEMENTS.md](IMPROVEMENTS.md)
2. Compare: Comparison tables inside

**"How do I use pagination?"**

1. See: Pagination Pattern (above)
2. Code: [implementation.md](implementation.md) Section 5

**"Which tool should I use?"**

1. See: Tool Decision Tree (above)
2. Details: [tools.md](tools.md) Section headers

**"What are the standards?"**

1. Read: [MCP_STANDARDS.md](MCP_STANDARDS.md)
2. Checklist: Standards section in same file

---

**Pro Tip**: Bookmark this visual guide for quick reference during implementation!
