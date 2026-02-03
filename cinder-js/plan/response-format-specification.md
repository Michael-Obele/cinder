# Response Format Specification

> **Purpose:** Define the output formats and metadata structure for cinder-js scrape responses  
> **Inspired by:** Firecrawl, Jina Reader, Crawlee  
> **Last Updated:** 2026-02-03

---

## Table of Contents

1. [Overview](#overview)
2. [Output Formats](#output-formats)
3. [Response Structure](#response-structure)
4. [Metadata Schema](#metadata-schema)
5. [TypeScript Definitions](#typescript-definitions)
6. [Configuration Options](#configuration-options)
7. [Examples](#examples)

---

## Overview

### Design Principles

1. **Flexibility:** Support multiple output formats for different use cases
2. **LLM-Ready:** Markdown as primary format, optimized for AI consumption
3. **Rich Metadata:** Comprehensive SEO and page metadata extraction
4. **Lean by Default:** Only return requested formats to minimize payload size
5. **Backward Compatible:** Extend Go Cinder response, don't break existing clients

### Supported Output Formats

| Format       | Description                      | Use Case                        |
| ------------ | -------------------------------- | ------------------------------- |
| `markdown`   | Clean markdown content           | LLMs, AI training, RAG          |
| `html`       | Cleaned HTML (no scripts/styles) | Content analysis, archiving     |
| `rawHtml`    | Original page HTML               | Debugging, full fidelity        |
| `text`       | Plain text (no formatting)       | Search indexing, NLP            |
| `links`      | Extracted URLs array             | Crawling, link analysis         |
| `screenshot` | Base64 PNG image                 | Visual verification, thumbnails |

---

## Output Formats

### Markdown (Default)

Primary output format, optimized for LLM consumption.

**Processing:**
1. Remove `<script>`, `<style>`, `<noscript>` tags
2. Extract main content area (`<main>`, `<article>`, or `<body>`)
3. Convert to GitHub-flavored Markdown via Turndown
4. Preserve structure: headings, lists, tables, code blocks, links, images

**Example:**
```markdown
# Example Domain

This domain is for use in illustrative examples in documents.

## More Information

You may use this domain in literature without prior coordination or asking for permission.

[More information...](https://www.iana.org/domains/example)
```

---

### HTML (Cleaned)

Sanitized HTML with boilerplate removed.

**Processing:**
1. Remove `<script>`, `<style>`, `<noscript>`, `<iframe>` tags
2. Remove navigation, footer, sidebar (via heuristics)
3. Preserve semantic structure
4. Remove tracking attributes (`onclick`, `data-analytics-*`)

---

### Raw HTML

Original page HTML exactly as returned by the server/browser.

**Use Cases:**
- Debugging scraping issues
- Full fidelity archiving
- Custom parsing pipelines

---

### Text

Plain text extraction with no formatting.

**Processing:**
1. Remove all HTML tags
2. Collapse whitespace
3. Preserve paragraph breaks
4. Strip navigation/header/footer text

---

### Links

Array of extracted URLs with context.

**Structure:**
```typescript
interface ExtractedLink {
  href: string;        // Resolved absolute URL
  text: string;        // Link text content
  rel?: string;        // Relationship (nofollow, etc.)
  isInternal: boolean; // Same domain as source
}
```

---

### Screenshot

Base64-encoded PNG of the rendered page.

**Requirements:** Only available with `mode: "dynamic"` (Playwright)

**Options:**
```typescript
{
  screenshot: true,              // Enable screenshot
  screenshotOptions: {
    fullPage: false,             // Viewport only vs full page
    quality: 80,                 // JPEG quality (if jpeg format)
    format: "png" | "jpeg",      // Image format
  }
}
```

---

## Response Structure

### Full Response Schema

```typescript
interface ScrapeResponse {
  // === Content Formats (based on request) ===
  markdown: string;              // Always included (default format)
  html?: string;                 // If requested
  rawHtml?: string;              // If requested
  text?: string;                 // If requested
  links?: ExtractedLink[];       // If requested
  screenshot?: string;           // Base64, if requested

  // === Page Metadata ===
  metadata: PageMetadata;

  // === Scraper Operational Info ===
  scrapeInfo: ScrapeInfo;
}
```

### Minimal Response (Default)

When only `markdown` is requested (default):

```json
{
  "markdown": "# Example Domain\n\nThis domain is...",
  "metadata": {
    "title": "Example Domain",
    "description": "Example domain for documentation"
  },
  "scrapeInfo": {
    "url": "https://example.com",
    "statusCode": 200,
    "engine": "cheerio",
    "mode": "static",
    "scrapedAt": "2026-02-02T10:30:00Z",
    "latencyMs": 245
  }
}
```

### Full Response (All Formats)

When all formats requested:

```json
{
  "markdown": "# Example Domain\n\nThis domain is...",
  "html": "<article><h1>Example Domain</h1>...</article>",
  "rawHtml": "<!DOCTYPE html><html>...</html>",
  "text": "Example Domain\n\nThis domain is for use in...",
  "links": [
    {
      "href": "https://www.iana.org/domains/example",
      "text": "More information...",
      "isInternal": false
    }
  ],
  "screenshot": "data:image/png;base64,iVBORw0KGgo...",
  "metadata": {
    "title": "Example Domain",
    "description": "Example domain for documentation",
    "ogTitle": "Example Domain",
    "ogDescription": "Example domain for documentation",
    "ogImage": "https://example.com/og-image.png",
    "ogUrl": "https://example.com",
    "ogType": "website",
    "ogLocale": "en_US",
    "twitterCard": "summary_large_image",
    "twitterTitle": "Example Domain",
    "twitterDescription": "Example domain for documentation",
    "twitterImage": "https://example.com/twitter-image.png",
    "favicon": "https://example.com/favicon.ico",
    "language": "en",
    "author": null,
    "publishedTime": null,
    "modifiedTime": null
  },
  "scrapeInfo": {
    "url": "https://example.com",
    "sourceUrl": "https://example.com",
    "statusCode": 200,
    "contentType": "text/html; charset=utf-8",
    "engine": "playwright",
    "mode": "dynamic",
    "cacheHit": false,
    "scrapedAt": "2026-02-02T10:30:00Z",
    "latencyMs": 2145
  }
}
```

---

## Metadata Schema

### Page Metadata

Extracted from HTML `<head>` and page content.

```typescript
interface PageMetadata {
  // === Core SEO ===
  title: string | null;
  description: string | null;
  
  // === Open Graph ===
  ogTitle: string | null;
  ogDescription: string | null;
  ogImage: string | null;
  ogUrl: string | null;
  ogType: string | null;           // website, article, product, etc.
  ogLocale: string | null;         // en_US, etc.
  ogLocaleAlternate: string[];     // Alternative locales
  ogSiteName: string | null;
  
  // === Twitter Cards ===
  twitterCard: string | null;      // summary, summary_large_image, etc.
  twitterTitle: string | null;
  twitterDescription: string | null;
  twitterImage: string | null;
  twitterSite: string | null;      // @username
  twitterCreator: string | null;   // @username
  
  // === Technical ===
  favicon: string | null;          // Resolved favicon URL
  language: string | null;         // html lang attribute
  charset: string | null;          // Character encoding
  viewport: string | null;         // Viewport meta
  robots: string | null;           // Robots meta
  canonical: string | null;        // Canonical URL
  
  // === Article/Blog ===
  author: string | null;
  publishedTime: string | null;    // ISO 8601
  modifiedTime: string | null;     // ISO 8601
  section: string | null;          // Article section
  tags: string[];                  // Article tags
  
  // === Framework Detection (Optional) ===
  generator: string | null;        // Docusaurus, Next.js, etc.
  framework: FrameworkInfo | null;
}

interface FrameworkInfo {
  name: string;                    // docusaurus, nextjs, gatsby, etc.
  version: string | null;
  // Framework-specific fields
  [key: string]: unknown;
}
```

### Scrape Info (Operational Data)

Information about the scraping operation itself.

```typescript
interface ScrapeInfo {
  // === Request ===
  url: string;                     // Final URL (after redirects)
  sourceUrl: string;               // Original requested URL
  
  // === Response ===
  statusCode: number;              // HTTP status code
  contentType: string | null;      // Content-Type header
  contentLength: number | null;    // Content-Length in bytes
  
  // === Scraper ===
  engine: "cheerio" | "playwright";
  mode: "static" | "dynamic" | "smart";
  
  // === Performance ===
  cacheHit: boolean;               // Was result from cache?
  scrapedAt: string;               // ISO 8601 timestamp
  latencyMs: number;               // Total scrape time in ms
  
  // === Redirects ===
  redirects: number;               // Number of redirects followed
  redirectChain: string[];         // URLs in redirect chain
}
```

---

## TypeScript Definitions

### Complete Type Definitions

```typescript
// ============================================
// REQUEST TYPES
// ============================================

type OutputFormat = "markdown" | "html" | "rawHtml" | "text" | "links" | "screenshot";
type ScrapeMode = "static" | "dynamic" | "smart";

interface ScrapeRequest {
  url: string;
  mode?: ScrapeMode;                           // Default: "smart"
  formats?: OutputFormat[];                    // Default: ["markdown"]
  
  // Optional feature flags
  includeRawHtml?: boolean;                    // Shorthand for formats
  includeScreenshot?: boolean;                 // Shorthand for formats
  
  // Screenshot options (if enabled)
  screenshotOptions?: {
    fullPage?: boolean;                        // Default: false
    quality?: number;                          // 1-100, default: 80
    format?: "png" | "jpeg";                   // Default: "png"
  };
  
  // Metadata options
  extractMetadata?: boolean;                   // Default: true
  detectFramework?: boolean;                   // Default: false
}

// ============================================
// RESPONSE TYPES
// ============================================

interface ScrapeResponse {
  // Content (based on requested formats)
  markdown: string;
  html?: string;
  rawHtml?: string;
  text?: string;
  links?: ExtractedLink[];
  screenshot?: string;
  
  // Metadata
  metadata: PageMetadata;
  scrapeInfo: ScrapeInfo;
}

interface ExtractedLink {
  href: string;
  text: string;
  rel?: string;
  title?: string;
  isInternal: boolean;
}

interface PageMetadata {
  // Core SEO
  title: string | null;
  description: string | null;
  
  // Open Graph
  ogTitle: string | null;
  ogDescription: string | null;
  ogImage: string | null;
  ogUrl: string | null;
  ogType: string | null;
  ogLocale: string | null;
  ogLocaleAlternate: string[];
  ogSiteName: string | null;
  
  // Twitter Cards
  twitterCard: string | null;
  twitterTitle: string | null;
  twitterDescription: string | null;
  twitterImage: string | null;
  twitterSite: string | null;
  twitterCreator: string | null;
  
  // Technical
  favicon: string | null;
  language: string | null;
  charset: string | null;
  viewport: string | null;
  robots: string | null;
  canonical: string | null;
  
  // Article
  author: string | null;
  publishedTime: string | null;
  modifiedTime: string | null;
  section: string | null;
  tags: string[];
  
  // Framework (optional)
  generator: string | null;
  framework: FrameworkInfo | null;
}

interface FrameworkInfo {
  name: string;
  version: string | null;
  [key: string]: unknown;
}

interface ScrapeInfo {
  url: string;
  sourceUrl: string;
  statusCode: number;
  contentType: string | null;
  contentLength: number | null;
  engine: "cheerio" | "playwright";
  mode: ScrapeMode;
  cacheHit: boolean;
  scrapedAt: string;
  latencyMs: number;
  redirects: number;
  redirectChain: string[];
}
```

### Valibot Schema

```typescript
import * as v from 'valibot';

const OutputFormatSchema = v.picklist([
  'markdown', 'html', 'rawHtml', 'text', 'links', 'screenshot'
]);

const ScrapeModeSchema = v.picklist(['static', 'dynamic', 'smart']);

const ScrapeRequestSchema = v.object({
  url: v.pipe(v.string(), v.url()),
  mode: v.optional(ScrapeModeSchema, 'smart'),
  formats: v.optional(v.array(OutputFormatSchema), ['markdown']),
  includeRawHtml: v.optional(v.boolean(), false),
  includeScreenshot: v.optional(v.boolean(), false),
  screenshotOptions: v.optional(v.object({
    fullPage: v.optional(v.boolean(), false),
    quality: v.optional(v.pipe(v.number(), v.minValue(1), v.maxValue(100)), 80),
    format: v.optional(v.picklist(['png', 'jpeg']), 'png'),
  })),
  extractMetadata: v.optional(v.boolean(), true),
  detectFramework: v.optional(v.boolean(), false),
});

type ScrapeRequest = v.InferOutput<typeof ScrapeRequestSchema>;
```

---

## Configuration Options

### Request Parameters

| Parameter           | Type     | Default        | Description               |
| ------------------- | -------- | -------------- | ------------------------- |
| `url`               | string   | required       | URL to scrape             |
| `mode`              | string   | `"smart"`      | Scraping mode             |
| `formats`           | string[] | `["markdown"]` | Output formats to include |
| `includeRawHtml`    | boolean  | `false`        | Include original HTML     |
| `includeScreenshot` | boolean  | `false`        | Include page screenshot   |
| `extractMetadata`   | boolean  | `true`         | Extract SEO metadata      |
| `detectFramework`   | boolean  | `false`        | Detect CMS/framework      |

### Format Selection Examples

**Minimal (default):**
```json
{
  "url": "https://example.com"
}
// Returns: markdown only
```

**Multiple formats:**
```json
{
  "url": "https://example.com",
  "formats": ["markdown", "html", "links"]
}
// Returns: markdown, html, and links
```

**With screenshot:**
```json
{
  "url": "https://example.com",
  "mode": "dynamic",
  "formats": ["markdown", "screenshot"],
  "screenshotOptions": {
    "fullPage": true,
    "format": "jpeg",
    "quality": 70
  }
}
```

---

## Examples

### Example 1: Blog Post Scrape

**Request:**
```json
{
  "url": "https://blog.example.com/post/hello-world",
  "formats": ["markdown", "text"],
  "detectFramework": true
}
```

**Response:**
```json
{
  "markdown": "# Hello World\n\nWelcome to my first blog post...",
  "text": "Hello World\n\nWelcome to my first blog post...",
  "metadata": {
    "title": "Hello World - My Blog",
    "description": "My first blog post about web development",
    "ogTitle": "Hello World",
    "ogType": "article",
    "author": "John Doe",
    "publishedTime": "2026-01-15T10:00:00Z",
    "tags": ["web development", "tutorial"],
    "generator": "Docusaurus 3.0.0",
    "framework": {
      "name": "docusaurus",
      "version": "3.0.0",
      "docusaurusTag": "default",
      "docusaurusLocale": "en"
    }
  },
  "scrapeInfo": {
    "url": "https://blog.example.com/post/hello-world",
    "sourceUrl": "https://blog.example.com/post/hello-world",
    "statusCode": 200,
    "engine": "cheerio",
    "mode": "static",
    "cacheHit": false,
    "scrapedAt": "2026-02-02T10:30:00Z",
    "latencyMs": 312
  }
}
```

### Example 2: SPA with Screenshot

**Request:**
```json
{
  "url": "https://app.example.com/dashboard",
  "mode": "dynamic",
  "formats": ["markdown", "screenshot", "links"],
  "screenshotOptions": {
    "fullPage": false
  }
}
```

**Response:**
```json
{
  "markdown": "# Dashboard\n\n## Recent Activity\n...",
  "screenshot": "data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAA...",
  "links": [
    {"href": "https://app.example.com/settings", "text": "Settings", "isInternal": true},
    {"href": "https://docs.example.com", "text": "Documentation", "isInternal": false}
  ],
  "metadata": {
    "title": "Dashboard - Example App",
    "ogTitle": "Dashboard",
    "generator": "Next.js",
    "framework": {
      "name": "nextjs",
      "version": null
    }
  },
  "scrapeInfo": {
    "url": "https://app.example.com/dashboard",
    "sourceUrl": "https://app.example.com/dashboard",
    "statusCode": 200,
    "engine": "playwright",
    "mode": "dynamic",
    "cacheHit": false,
    "scrapedAt": "2026-02-02T10:30:00Z",
    "latencyMs": 2456
  }
}
```

### Example 3: Crawl Result (Multiple Pages)

For the `/v1/crawl` endpoint, each page in results follows this format:

```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "status": "completed",
  "results": [
    {
      "markdown": "# Home\n...",
      "metadata": {...},
      "scrapeInfo": {...}
    },
    {
      "markdown": "# About\n...",
      "metadata": {...},
      "scrapeInfo": {...}
    }
  ]
}
```

---

## Migration from Simple Response

### Current Go Cinder Response

```json
{
  "url": "https://example.com",
  "markdown": "...",
  "html": "...",
  "metadata": {
    "scraped_at": "...",
    "engine": "..."
  }
}
```

### New cinder-js Response (Backward Compatible)

The new format extends the old one:
- `markdown` and `html` remain at top level
- `metadata` now contains page metadata (not scraper info)
- New `scrapeInfo` field contains operational data
- Old clients can ignore new fields

**Compatibility layer (if needed):**
```typescript
function toLegacyFormat(response: ScrapeResponse): LegacyResponse {
  return {
    url: response.scrapeInfo.url,
    markdown: response.markdown,
    html: response.html ?? '',
    metadata: {
      scraped_at: response.scrapeInfo.scrapedAt,
      engine: response.scrapeInfo.engine,
    },
  };
}
```

---

## Implementation Priority

| Phase       | Formats                    | Metadata                       |
| ----------- | -------------------------- | ------------------------------ |
| **Phase 1** | `markdown`, `html`         | title, description, basic SEO  |
| **Phase 2** | `links`, `rawHtml`, `text` | Full Open Graph, Twitter Cards |
| **Phase 3** | `screenshot`               | Framework detection            |
| **Future**  | `json` (LLM extraction)    | Custom schema extraction       |

---

*Document Version: 1.0.1*  
*Last Updated: 2026-02-03*
