# Response Format Specification

## Overview

This document defines the standard response format for the Cinder-JS scraping API, including the primary JSON structure, TypeScript type definitions, and alternative output formats.

---

## Primary Response Format: Markdown + Rich Metadata

The default response combines LLM-ready markdown content with comprehensive metadata extracted from the scraped page.

### JSON Structure

```json
{
  "markdown": "string (The full markdown content of the page)",
  "metadata": {
    "og:url": "string",
    "twitter:card": "string",
    "docsearch:docusaurus_tag": "string",
    "language": "string",
    "ogDescription": "string",
    "og:locale:alternate": ["string"],
    "og:title": "string",
    "og:description": "string",
    "generator": "string",
    "og:locale": "string",
    "twitter:image": "string",
    "ogLocale": "string",
    "viewport": "string",
    "docusaurus_version": "string",
    "description": "string",
    "og:image": "string",
    "docsearch:version": "string",
    "title": "string",
    "docsearch:language": "string",
    "ogImage": "string",
    "ogLocaleAlternate": ["string"],
    "docusaurus_locale": "string",
    "ogUrl": "string",
    "twitter:creator": "string",
    "docusaurus_tag": "string",
    "ogTitle": "string",
    "og:type": "string",
    "favicon": "string",
    "scrapeId": "string",
    "sourceURL": "string",
    "url": "string",
    "statusCode": "number",
    "contentType": "string",
    "timezone": "string",
    "proxyUsed": "string",
    "cacheState": "string",
    "indexId": "string",
    "creditsUsed": "number",
    "concurrencyLimited": "boolean"
  }
}
```

### TypeScript Definitions

```typescript
interface ScrapingMetadata {
  // SEO & Social Tags
  "og:url": string;
  "ogUrl": string;
  "og:title": string;
  "ogTitle": string;
  "og:description": string;
  "ogDescription": string;
  "og:image": string;
  "ogImage": string;
  "og:type": string;
  "og:locale": string;
  "ogLocale": string;
  "og:locale:alternate": string[];
  "ogLocaleAlternate": string[];
  
  "twitter:card": string;
  "twitter:image": string;
  "twitter:creator": string;
  
  "title": string;
  "description": string;
  "viewport": string;
  "favicon": string;
  "language": string;
  "generator": string;

  // Docusaurus / Docsearch Specifics
  "docsearch:docusaurus_tag": string;
  "docsearch:version": string;
  "docsearch:language": string;
  "docusaurus_version": string;
  "docusaurus_locale": string;
  "docusaurus_tag": string;

  // Scraper Audit Data
  scrapeId: string;
  sourceURL: string;
  url: string;
  statusCode: number;
  contentType: string;
  timezone: string;
  proxyUsed: string;
  cacheState: string;
  indexId: string;
  creditsUsed: number;
  concurrencyLimited: boolean;
}

interface ScrapedDocument {
  markdown: string;
  metadata: ScrapingMetadata;
}
```

### Key Observations

- **Redundancy**: The metadata contains several redundant keys (e.g., `og:url` and `ogUrl`, `og:locale:alternate` and `ogLocaleAlternate`). This is intentional to support different naming conventions and backward compatibility.
- **Types**: All fields are strings except for `statusCode`, `creditsUsed` (numbers), `concurrencyLimited` (boolean), and the locale alternate arrays.
- **Extensibility**: Additional fields can be added to metadata without breaking existing consumers.

---

## Alternative Output Formats

Beyond the primary markdown format, Cinder-JS supports multiple output formats to suit different use cases.

### Available Formats

| Format | Content Type | Use Case | Generation Method |
|--------|-------------|----------|-------------------|
| `markdown` | `application/json` | LLM ingestion, RAG pipelines | Turndown from HTML |
| `html` | `text/html` | Raw page content preservation | Direct fetch/Playwright capture |
| `text` | `text/plain` | Clean text extraction for NLP | Strip tags from HTML |
| `structured` | `application/json` | Schema-based data extraction | AI/LLM extraction or CSS selectors |
| `screenshot` | `image/png` | Visual page capture | Playwright screenshot API |
| `summary` | `application/json` | Concise page overview | AI summarization or content analysis |
| `links` | `application/json` | URL discovery and mapping | Extract all anchor tags |

### Format Selection API

Users can request specific formats via the API:

```typescript
// Request single format
POST /v1/scrape
{
  "url": "https://example.com",
  "format": "markdown"
}

// Request multiple formats
POST /v1/scrape
{
  "url": "https://example.com",
  "formats": ["markdown", "html", "screenshot"]
}
```

---

## Format Details

### 1. Markdown (Default)
- **Best for**: LLM consumption, documentation, content extraction
- **Processing**: HTML → Turndown → Clean markdown
- **Includes**: Headers, lists, links, tables, code blocks
- **Strips**: Scripts, styles, ads, navigation

### 2. HTML (Raw)
- **Best for**: Archiving, debugging, custom parsing
- **Processing**: Direct page capture
- **Variants**:
  - `raw_html`: Complete page source with scripts
  - `html`: Sanitized HTML (optional)

### 3. Plain Text
- **Best for**: NLP pipelines, keyword extraction, sentiment analysis
- **Processing**: HTML → Strip tags → Normalize whitespace
- **Characteristics**: No formatting, just content

### 4. Structured Data (JSON)
- **Best for**: Data extraction, API integration, database storage
- **Processing**: 
  - **Schema-based**: Define JSON schema, extract matching data
  - **AI extraction**: Natural language prompts for flexible extraction
- **Example**:
```json
{
  "product": {
    "name": "Widget Pro",
    "price": "$99.99",
    "rating": 4.5,
    "reviews": 128
  }
}
```

### 5. Screenshot
- **Best for**: Visual verification, design analysis, archiving
- **Processing**: Playwright viewport or full-page capture
- **Options**:
  - Full page vs viewport
  - Specific element selector
  - Mobile/desktop viewport sizes

### 6. Summary
- **Best for**: Quick overview, content monitoring, news aggregation
- **Processing**: AI-generated concise summary
- **Characteristics**: Reduces content by 80-90%, preserves key points

### 7. Links
- **Best for**: Site mapping, crawling, URL discovery
- **Processing**: Extract all `<a href>` tags
- **Output**:
```json
{
  "links": [
    {"url": "https://example.com/page1", "text": "Page 1"},
    {"url": "https://example.com/page2", "text": "Page 2"}
  ]
}
```

---

## Implementation Strategy

### Stack Alignment

| Format | Primary Tool | Fallback |
|--------|-------------|----------|
| Markdown | @turndown/turndown | Cheerio text extraction |
| HTML | Fetch / Playwright | N/A |
| Text | Cheerio (strip tags) | Turndown + text extraction |
| Structured | AI/LLM service | CSS selectors |
| Screenshot | Playwright | None (requires browser) |
| Summary | AI/LLM service | Truncated first paragraph |
| Links | Cheerio | Playwright DOM query |

### Cost-Benefit Analysis

| Format | CPU Cost | Memory Cost | Credit Multiplier | Best Use Case |
|--------|---------|-------------|-------------------|---------------|
| Markdown | Low | Low | 1x | General purpose |
| HTML | Low | Medium | 1x | Archiving |
| Text | Low | Low | 1x | NLP pipelines |
| Structured | High | Medium | 2-3x | Data extraction |
| Screenshot | Medium | Medium | 2x | Visual analysis |
| Summary | High | Low | 2x | Quick scanning |
| Links | Low | Low | 1x | Crawling |

*Credit multiplier represents relative API credit consumption compared to base markdown format.*

### Multi-Format Requests

When multiple formats are requested:
1. Perform single scrape operation
2. Generate all requested formats from captured data
3. Return combined response
4. Only count as 1 credit base + format premiums

---

## Response Schema Evolution

### Versioning Strategy
- **Current**: v1 (documented above)
- **Backward Compatibility**: New fields are additive only
- **Deprecation**: Deprecated fields marked but retained for 6 months
- **Migration Guide**: Provided for major version changes

### Future Enhancements
- **Embedding vectors**: Pre-computed embeddings for RAG
- **Content classification**: Auto-detected content type
- **Reading time**: Estimated reading duration
- **Language confidence**: Detection confidence score
- **Entity extraction**: Named entities (people, orgs, locations)

---

## Error Response Format

Standardized error responses for all formats:

```json
{
  "error": {
    "code": "SCRAPE_FAILED",
    "message": "Failed to scrape URL",
    "details": {
      "url": "https://example.com",
      "reason": "HTTP 403 Forbidden",
      "retryable": false
    }
  }
}
```

### Error Codes
- `SCRAPE_FAILED`: General scrape failure
- `TIMEOUT`: Request timeout
- `INVALID_URL`: Malformed or invalid URL
- `BLOCKED`: Site blocks scraping
- `PARSE_ERROR`: Content parsing failure
- `RATE_LIMITED`: API rate limit exceeded

---

## Related Documents
- [Architecture & ADRs](cinder-js-architecture.md) - Technology stack decisions
- [Smart Mode Heuristics](smart-mode-heuristics.md) - Format selection logic
- [Performance Benchmark Plan](performance-benchmark-plan.md) - Format performance testing
- [Valibot Config Schema](valibot-config-schema.md) - Configuration validation

