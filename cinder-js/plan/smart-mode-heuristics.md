# Smart Mode Heuristics

> **Purpose:** Document the detection algorithms for determining when to use dynamic scraping  
> **Based on:** Go implementation in `internal/scraper/heuristics.go`  
> **Last Updated:** 2026-02-02

---

## Table of Contents

1. [Overview](#overview)
2. [Fallback Chain Architecture](#fallback-chain-architecture)
3. [Detection Algorithms](#detection-algorithms)
4. [Threshold Configurations](#threshold-configurations)
5. [Decision Flowchart](#decision-flowchart)
6. [Known Patterns](#known-patterns)
7. [Tuning Guidelines](#tuning-guidelines)

---

## Overview

Smart Mode automatically determines the optimal scraping strategy for each URL, balancing speed and completeness.

### Goals

1. **Speed First:** Use static scraping (Cheerio) when possible (~200ms vs ~2s)
2. **Graceful Fallback:** Detect when dynamic scraping (Playwright) is needed
3. **Minimal False Positives:** Avoid unnecessary browser launches
4. **Minimal False Negatives:** Capture content that requires JavaScript

### Mode Priority

```
┌─────────────────────────────────────────────────────────────┐
│                     SMART MODE CHAIN                        │
│                                                             │
│   TIER 1           TIER 2              TIER 3               │
│   (Fastest)        (Fast)              (Complete)           │
│                                                             │
│   ┌─────────┐      ┌────────────┐      ┌───────────────┐   │
│   │  fetch  │ ───► │  Cheerio   │ ───► │  Playwright   │   │
│   │  only   │      │  (parse)   │      │  (full)       │   │
│   └─────────┘      └────────────┘      └───────────────┘   │
│                                                             │
│   ~50ms            ~200ms              ~2000ms              │
│   Raw HTML         Parse HTML          Full render          │
│   No parse         No JS               JS execution         │
└─────────────────────────────────────────────────────────────┘
```

---

## Fallback Chain Architecture

### Tier 1: Raw Fetch (Optional Fast Path)

**When Used:** Initial content assessment
**Time:** ~50ms
**Output:** Raw HTML string

```javascript
// Conceptual implementation
async function tier1Fetch(url) {
  const response = await fetch(url, {
    headers: { 'User-Agent': getRandomUA() }
  });
  return response.text();
}
```

**Purpose:** Quick retrieval for heuristics check, avoiding full parse if dynamic is obviously needed.

---

### Tier 2: Cheerio Parse (Default)

**When Used:** Static HTML content
**Time:** ~200ms
**Output:** Parsed DOM + Markdown

```javascript
// Conceptual implementation
async function tier2Cheerio(html) {
  const $ = cheerio.load(html);
  
  // Remove unwanted elements
  $('script, style, noscript').remove();
  
  // Extract content
  const content = $('main, article, #content, body').html();
  
  // Convert to markdown
  return turndown.turndown(content);
}
```

**Advantages:**
- 8-12x faster than browser
- 40% less memory
- No browser process required

**Limitations:**
- No JavaScript execution
- Cannot handle SPAs
- Cannot handle dynamic content

---

### Tier 3: Playwright Render (Fallback)

**When Used:** Dynamic content detected
**Time:** ~2000ms
**Output:** Fully rendered DOM + Markdown

```javascript
// Conceptual implementation
async function tier3Playwright(url) {
  const context = await browser.newContext();
  const page = await context.newPage();
  
  try {
    await page.goto(url, { 
      waitUntil: 'networkidle',
      timeout: 30000 
    });
    
    // Wait for content to stabilize
    await page.waitForLoadState('domcontentloaded');
    
    // Extract content
    const html = await page.content();
    return turndown.turndown(html);
  } finally {
    await context.close();
  }
}
```

**Advantages:**
- Full JavaScript execution
- Handles SPAs, React, Vue, Angular
- Handles lazy loading, infinite scroll

**Limitations:**
- 10x slower than static
- Higher memory usage
- Browser process required

---

## Detection Algorithms

### Algorithm 1: Noscript Warning Detection

**Trigger:** `<noscript>` tag contains JavaScript requirement message

**Implementation:**
```javascript
function detectNoscriptWarning(html) {
  const lowerHtml = html.toLowerCase();
  
  if (!lowerHtml.includes('<noscript>')) {
    return false;
  }
  
  const warningPhrases = [
    'enable javascript',
    'need javascript',
    'requires javascript',
    'javascript is required',
    'javascript must be enabled',
    'please enable javascript',
    'you need to enable javascript'
  ];
  
  return warningPhrases.some(phrase => lowerHtml.includes(phrase));
}
```

**Examples:**
```html
<!-- TRIGGERS DYNAMIC -->
<noscript>
  <div class="no-js-warning">
    You need to enable JavaScript to run this app.
  </div>
</noscript>

<!-- DOES NOT TRIGGER (common analytics fallback) -->
<noscript>
  <img src="https://analytics.example.com/pixel.gif" />
</noscript>
```

**Confidence:** High (95%)

---

### Algorithm 2: SPA Root Detection

**Trigger:** Known SPA framework markers + small body size

**Implementation:**
```javascript
function detectSPARoot(html) {
  const spaMarkers = [
    // React
    'id="root"',
    'id="app"', 
    'data-reactroot',
    
    // Next.js
    'id="__next"',
    '__NEXT_DATA__',
    
    // Vue
    'data-v-',
    'id="__nuxt"',
    
    // Angular
    'ng-version',
    '<app-root>',
    'ng-app',
    
    // Generic
    'window.__INITIAL_STATE__',
    'window.__PRELOADED_STATE__',
    'window.__REDUX_STATE__'
  ];
  
  const hasMarker = spaMarkers.some(marker => html.includes(marker));
  
  if (hasMarker) {
    // Check if content is minimal (likely a shell)
    const SHELL_THRESHOLD = 5000; // bytes
    if (html.length < SHELL_THRESHOLD) {
      return true;
    }
    
    // Even with larger content, check for empty main containers
    const $ = cheerio.load(html);
    const mainContent = $('main, #root, #app, #__next').text().trim();
    if (mainContent.length < 100) {
      return true;
    }
  }
  
  return false;
}
```

**Framework-Specific Patterns:**

| Framework | Markers                        | Shell Size          |
| --------- | ------------------------------ | ------------------- |
| React     | `id="root"`, `data-reactroot`  | < 3KB               |
| Next.js   | `id="__next"`, `__NEXT_DATA__` | < 5KB (SSR partial) |
| Vue       | `data-v-*`, `id="app"`         | < 3KB               |
| Nuxt      | `id="__nuxt"`                  | < 5KB               |
| Angular   | `ng-version`, `<app-root>`     | < 4KB               |

**Confidence:** Medium (75%)

**False Positive Mitigation:** Require BOTH marker AND small size. SSR frameworks (Next.js, Nuxt) often have markers but also pre-rendered content.

---

### Algorithm 3: Minimal Content Heuristic

**Trigger:** Very small body with script tags

**Implementation:**
```javascript
function detectMinimalContent(html) {
  const MINIMAL_THRESHOLD = 2000; // bytes
  const lowerHtml = html.toLowerCase();
  
  if (html.length < MINIMAL_THRESHOLD && lowerHtml.includes('<script')) {
    // Additional check: is there meaningful text?
    const $ = cheerio.load(html);
    const textContent = $('body').text().trim();
    
    // If body text is also minimal, it's likely a shell
    if (textContent.length < 200) {
      return true;
    }
  }
  
  return false;
}
```

**Examples:**
```html
<!-- TRIGGERS DYNAMIC (minimal shell) -->
<!DOCTYPE html>
<html>
<head><title>App</title></head>
<body>
  <div id="root"></div>
  <script src="/static/js/bundle.js"></script>
</body>
</html>

<!-- DOES NOT TRIGGER (has content) -->
<!DOCTYPE html>
<html>
<head><title>Blog</title></head>
<body>
  <h1>Welcome to My Blog</h1>
  <p>This is my first post with lots of content...</p>
  <script src="/analytics.js"></script>
</body>
</html>
```

**Confidence:** Medium (70%)

---

### Algorithm 4: Network Request Pattern (Future Enhancement)

**Trigger:** Heavy XHR/fetch usage detected

**Concept:**
```javascript
// Future enhancement: analyze network patterns
function detectAjaxHeavy(html) {
  const ajaxPatterns = [
    'XMLHttpRequest',
    'fetch(',
    'axios.',
    '$.ajax',
    '$.get',
    '$.post'
  ];
  
  let hitCount = 0;
  for (const pattern of ajaxPatterns) {
    if (html.includes(pattern)) {
      hitCount++;
    }
  }
  
  // If multiple AJAX patterns found, likely needs JS
  return hitCount >= 2;
}
```

**Status:** Not implemented in current Go version. Consider for Phase 2.

---

## Threshold Configurations

### Configurable Thresholds

```javascript
// Suggested default configuration
const SMART_MODE_CONFIG = {
  // Size thresholds (bytes)
  spaShellThreshold: 5000,      // Max size to consider SPA shell
  minimalBodyThreshold: 2000,   // Max size for "minimal content" check
  emptyContentThreshold: 200,   // Min text length to consider "has content"
  
  // Timing thresholds (ms)
  staticTimeout: 5000,          // Max time for static scrape
  dynamicTimeout: 30000,        // Max time for dynamic scrape
  networkIdleWait: 2000,        // Wait for network idle after load
  
  // Behavior flags
  alwaysTryStaticFirst: true,   // Always attempt static before dynamic
  cacheHeuristics: true,        // Cache heuristic results per domain
  heuristicsCacheTtl: 3600000,  // 1 hour cache for domain heuristics
};
```

### Domain-Specific Overrides (Future)

```javascript
// Example: Known SPAs that always need dynamic
const KNOWN_DYNAMIC_DOMAINS = new Set([
  'twitter.com',
  'instagram.com',
  'linkedin.com',
  'tiktok.com',
  'discord.com',
  'reddit.com',   // Some pages
  'netflix.com',
]);

// Known static sites that never need dynamic
const KNOWN_STATIC_DOMAINS = new Set([
  'example.com',
  'wikipedia.org',
  'github.com',   // Most pages
  'news.ycombinator.com',
]);
```

---

## Decision Flowchart

```
                    ┌─────────────────┐
                    │  Receive URL    │
                    └────────┬────────┘
                             │
            ┌────────────────▼────────────────┐
            │  Is domain in KNOWN_DYNAMIC?    │
            └────────────────┬────────────────┘
                    Yes │         │ No
                        ▼         │
         ┌──────────────────┐     │
         │ Use Playwright   │     │
         │ (skip static)    │     │
         └──────────────────┘     │
                                  ▼
            ┌────────────────────────────────┐
            │  Fetch HTML (Tier 1)           │
            └────────────────┬───────────────┘
                             │
            ┌────────────────▼────────────────┐
            │  Run Heuristics Battery         │
            │                                 │
            │  1. Noscript warning?           │
            │  2. SPA root + small body?      │
            │  3. Minimal content + scripts?  │
            └────────────────┬────────────────┘
                             │
         ┌───────────────────┴───────────────────┐
         │ Any heuristic                         │ All heuristics
         │ returned TRUE                         │ returned FALSE
         ▼                                       ▼
┌────────────────────┐              ┌────────────────────┐
│ Use Playwright     │              │ Use Cheerio        │
│ (Tier 3)           │              │ (Tier 2)           │
│                    │              │                    │
│ • Full JS render   │              │ • Parse HTML       │
│ • Wait for idle    │              │ • Extract content  │
│ • ~2000ms          │              │ • ~200ms           │
└────────┬───────────┘              └────────┬───────────┘
         │                                   │
         │                                   │
         └──────────────┬────────────────────┘
                        │
                        ▼
            ┌───────────────────────────┐
            │  Validate Output          │
            │                           │
            │  Content length OK?       │
            │  Markdown non-empty?      │
            └───────────┬───────────────┘
                        │
         ┌──────────────┴──────────────┐
         │ Valid                       │ Invalid/Empty
         ▼                             ▼
┌────────────────────┐      ┌────────────────────────────┐
│ Return Result      │      │ If static failed:          │
│                    │      │   Try dynamic as fallback  │
│ • markdown         │      │                            │
│ • html             │      │ If dynamic failed:         │
│ • metadata         │      │   Return error with        │
└────────────────────┘      │   partial content (if any) │
                            └────────────────────────────┘
```

---

## Known Patterns

### Sites That Require Dynamic Scraping

| Pattern                | Examples                     | Detection                |
| ---------------------- | ---------------------------- | ------------------------ |
| **Social Media**       | Twitter, Instagram, LinkedIn | Domain blocklist         |
| **React SPAs**         | Many startups, dashboards    | `id="root"` + small body |
| **Next.js CSR**        | Client-only routes           | `id="__next"` + empty    |
| **Angular Apps**       | Enterprise tools             | `ng-version`             |
| **Infinite Scroll**    | Pinterest, Reddit            | XHR detection (future)   |
| **Paywall/Auth Walls** | Medium, NY Times             | noscript warnings        |

### Sites That Work With Static

| Pattern             | Examples              | Notes                                |
| ------------------- | --------------------- | ------------------------------------ |
| **Static Sites**    | GitHub pages, docs    | Always static                        |
| **SSR Frameworks**  | Next.js SSR, Nuxt SSR | Has `__NEXT_DATA__` but also content |
| **Traditional CMS** | WordPress, Drupal     | HTML mostly complete                 |
| **News Sites**      | Most newspapers       | HTML with analytics scripts          |
| **Documentation**   | Docs sites, wikis     | Pure HTML                            |

### Edge Cases

| Case                     | Behavior           | Notes                            |
| ------------------------ | ------------------ | -------------------------------- |
| **SSR + Hydration**      | Static often works | Check content length             |
| **Lazy Images**          | Static works       | Images load but no JS needed     |
| **Lazy Text**            | Dynamic needed     | Content via IntersectionObserver |
| **Auth Required**        | Both fail          | Need session handling (future)   |
| **Cloudflare Challenge** | Both may fail      | Need challenge solving (future)  |

---

## Tuning Guidelines

### Reducing False Positives (Unnecessary Dynamic)

**Problem:** Some sites trigger dynamic mode but don't need it.

**Solutions:**
1. Increase `spaShellThreshold` from 5KB to 8KB
2. Add content length validation after static parse
3. Cache results: if static worked once, prefer static

### Reducing False Negatives (Missing Dynamic)

**Problem:** Some SPAs slip through and return empty content.

**Solutions:**
1. Add output validation: if too little content, retry with dynamic
2. Decrease `spaShellThreshold` to 3KB
3. Add more SPA markers to detection list

### Performance Tuning

**Goal:** Minimize time spent in heuristics.

**Strategies:**
1. Check domain blocklist first (O(1))
2. Run cheap heuristics first (string contains)
3. Parse HTML only if needed (Cheerio is heavy)
4. Cache heuristic results per domain

### Monitoring Recommendations

Track these metrics to tune heuristics:

```javascript
const metrics = {
  // Decision tracking
  staticAttempts: 0,
  dynamicAttempts: 0,
  staticSuccesses: 0,
  dynamicSuccesses: 0,
  staticFallbackToDynamic: 0,
  
  // Timing
  avgStaticTime: 0,
  avgDynamicTime: 0,
  avgHeuristicsTime: 0,
  
  // Quality
  emptyResults: 0,
  shortResults: 0,  // < 100 chars
};
```

**Dashboard Alerts:**
- `staticFallbackToDynamic > 30%` → Heuristics may be too aggressive
- `emptyResults > 10%` → Missing edge cases
- `avgHeuristicsTime > 50ms` → Optimize detection code

---

## Appendix: Port from Go

### Original Go Implementation

```go
// From internal/scraper/heuristics.go
func ShouldUseDynamic(htmlBody string) bool {
    lowerBody := strings.ToLower(htmlBody)

    // 1. Check for <noscript> instructions
    if strings.Contains(lowerBody, "<noscript>") {
        if strings.Contains(lowerBody, "enable javascript") ||
            strings.Contains(lowerBody, "need javascript") ||
            strings.Contains(lowerBody, "requires javascript") {
            return true
        }
    }

    // 2. Check for SPA Root Elements
    spaRoots := []string{
        `id="root"`,
        `id="app"`,
        `id="__next"`,
        `data-reactroot`,
        `__NEXT_DATA__`,
        `ng-version`,
        `<app-root>`,
    }

    for _, marker := range spaRoots {
        if strings.Contains(htmlBody, marker) {
            if len(htmlBody) < 5000 {
                return true
            }
        }
    }

    // 3. Simple Content Size Heuristic
    if len(htmlBody) < 2000 && strings.Contains(lowerBody, "<script") {
        return true
    }

    return false
}
```

### Equivalent TypeScript Implementation

```typescript
export function shouldUseDynamic(htmlBody: string): boolean {
  const lowerBody = htmlBody.toLowerCase();

  // 1. Check for <noscript> instructions
  if (lowerBody.includes('<noscript>')) {
    if (
      lowerBody.includes('enable javascript') ||
      lowerBody.includes('need javascript') ||
      lowerBody.includes('requires javascript')
    ) {
      return true;
    }
  }

  // 2. Check for SPA Root Elements
  const spaRoots = [
    'id="root"',
    'id="app"',
    'id="__next"',
    'data-reactroot',
    '__NEXT_DATA__',
    'ng-version',
    '<app-root>',
  ];

  for (const marker of spaRoots) {
    if (htmlBody.includes(marker)) {
      if (htmlBody.length < 5000) {
        return true;
      }
    }
  }

  // 3. Simple Content Size Heuristic
  if (htmlBody.length < 2000 && lowerBody.includes('<script')) {
    return true;
  }

  return false;
}
```

---

*Document Version: 1.0.0-draft*  
*Last Updated: 2026-02-02*
