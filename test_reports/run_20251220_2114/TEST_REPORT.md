# Production Test Report - Run 20251220_2114

**Date:** 2025-12-20
**Environment:** Production (Leapcell)
**URL:** https://cinder.leapcell.app

## Summary

All tests passed successfully. The application is correctly handling static scraping, dynamic scraping (JS rendering), asynchronous crawling, and Redis caching.

## Test Results

### 1. Static Scrape (Cache Miss)

- **URL:** `https://example.com/?test=1766261746`
- **Status:** ✅ Success
- **Cache Status:** MISS (Expected)
- **Engine:** Colly
- **Output:** [static_miss.json](./static_miss.json)

### 2. Static Scrape (Cache Hit)

- **URL:** `https://example.com/?test=1766261746`
- **Status:** ✅ Success
- **Cache Status:** HIT (Confirmed by `metadata.cached: "true"`)
- **Engine:** Colly
- **Output:** [static_hit.json](./static_hit.json)

### 3. Dynamic Scrape (JS Rendering)

- **URL:** `https://quotes.toscrape.com/js/`
- **Status:** ✅ Success
- **Engine:** Chromedp (Headless Chrome on Alpine)
- **Output:** [dynamic.json](./dynamic.json)
- **Notes:** Successfully rendered JS-generated content (quotes).

### 4. Asynchronous Crawl

- **URL:** `https://books.toscrape.com/`
- **Status:** ✅ Success
- **Job ID:** `ceb7c615-36f1-4516-8780-db2246377a3e`
- **Result:** Scraped successfully.
- **Output:** [crawl_result.json](./crawl_result.json)

## Conclusion

The deployment is fully functional. The fix for Alpine Linux compatibility (`install_browser.sh`) is working as expected, allowing `chromedp` to run in the production environment. Redis is correctly configured and caching responses.
