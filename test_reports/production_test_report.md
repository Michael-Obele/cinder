# Production Test Report: cinder.leapcell.app
**Date:** 2025-12-20
**Target:** `https://cinder.leapcell.app`

## Executive Summary
The production environment was tested using `curl` against the defined API endpoints.
- **Static Scraping**: ✅ Operational
- **Crawling (Static)**: ✅ Operational
- **Dynamic Scraping (JS Rendering)**: ❌ Failed (500 Internal Server Error)
- **Crawling (Dynamic)**: ❌ Failed (Jobs stuck in retry loop)

## Detailed Test Results

### 1. Static Scrape Endpoints
**Endpoint:** `POST /v1/scrape`
**Method:** Static (Colly engine)
**Result:** **Success**
- The server correctly fetched and parsed `http://example.com` into Markdown and HTML.
- Response time was fast.
- Metadata indicated `engine: colly`.

### 2. Dynamic Scrape Endpoints
**Endpoint:** `POST /v1/scrape`
**Method:** Dynamic (Chromedp/Headless Chrome)
**Payload:** `{"render": true, ...}`
**Result:** **Failed**
- **HTTP Status:** 500 Internal Server Error
- **Error Body:** `{"error":"Scraping failed"}`
- **Diagnosis:** The generic error suggests a failure in the `chromedp` execution. This is commonly caused by:
    - Missing Chromium binary in the runtime environment.
    - Incorrect executable path environment variables (`CHROME_BIN`).
    - Insufficient memory/resources for the headless browser to launch.

### 3. Asynchronous Crawl (Static)
**Endpoint:** `POST /v1/crawl`
**Method:** Static
**Result:** **Success**
- Job submitted successfully (202 Accepted).
- Job ID received.
- Status check (`GET /v1/crawl/:id`) showed `status: completed` and valid results.

### 4. Asynchronous Crawl (Dynamic)
**Endpoint:** `POST /v1/crawl`
**Method:** Dynamic (`render: true`)
**Result:** **Partial Failure**
- Job submitted successfully.
- Status check showed `status: retry` and `retried: 1`.
- This confirms that the worker process is also failing to execute the dynamic render task, likely for the same reason as the synchronous endpoint.

## Recommendations
1. **Verify Chromium Installation:** Ensure the production environment has a compatible Chromium binary installed. If using Docker, verify `apk add chromium` succeeded and paths conform to `CHROME_BIN` settings.
2. **Check Resource Limits:** Headless Chrome requires significant memory. Increase instance memory if possible.
3. **Environment Variables:** Verify `CHROME_BIN` and `CHROME_PATH` are correctly set in the Leapcell dashboard.
