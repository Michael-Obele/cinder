# E2E Test Report

Date: Sat, 20 Dec 2025 17:29:09 WAT

## ✅ Static Scrape Test Passed

Saved to: `static.json`
Response size: 7640 bytes

## ❌ Dynamic Scrape Test Failed

Error: bad status: 500 Internal Server Error, body: {"error":"Scraping failed"}

## ✅ Async Crawl Test Passed

Saved to: `crawl.json`
Final Status: completed


## Analysis & Suggestions

### 1. Dynamic Scrape Failure
**Issue:** The dynamic scraping test failed with a 500 error.
**Root Cause:** The environment is missing a headless browser (Chrome/Chromium). `chromedp` requires a local installation of Chrome to render pages.
**Suggestion:** 
- Install `chromium` or `google-chrome-stable` in the runtime environment.
- Or use a Docker image that includes a browser (e.g., `chromedp/headless-shell`).

### 2. Async Crawl Success
**Status:** Fixed and Passing.
**Fix Implemented:** Added `asynq.Retention(24 * time.Hour)` to task creation.
**Details:** Previously, completed tasks were expiring immediately (or default retention was insufficient/unconfigured), causing `GetTaskInfo` to return 404. Now, completed tasks are stored in Redis for 24 hours, allowing status checks to succeed.
