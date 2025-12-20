# Test Report: Docker Deployment Verification
**Date:** 2025-12-20
**Environment:** Docker (Docker Compose)
**Services:** API, Worker, Redis

## Summary
All core features of the Cinder application were tested in a fully containerized environment. All tests passed successfully.

## Test Results

| Feature | Type | Status | Artifact |
| :--- | :--- | :--- | :--- |
| **Static Scraping** | Synchronous | ✅ PASS | [static.json](./static.json) |
| **Dynamic Scraping** | Synchronous | ✅ PASS | [dynamic.json](./dynamic.json) |
| **Async Crawl** | Asynchronous | ✅ PASS | [crawl_response.json](./crawl_response.json) |
| **Job Status** | Asynchronous | ✅ PASS | [crawl_status.json](./crawl_status.json) |

## Detailed Verification

### 1. Static Scraping (Colly)
- **Endpoint:** `POST /v1/scrape`
- **Payload:** `{"url": "https://example.com", "render": false}`
- **Observation:** API returned parsed Markdown and HTML. Engine was identified as `colly`.

### 2. Dynamic Scraping (Chromedp)
- **Endpoint:** `POST /v1/scrape`
- **Payload:** `{"url": "https://example.com", "render": true}`
- **Observation:** API returned parsed Markdown and HTML. Engine was identified as `chromedp`. This confirms Chromium is correctly installed and accessible within the Docker container.

### 3. Async Crawl & Worker Processing
- **Endpoint:** `POST /v1/crawl`
- **Payload:** `{"url": "https://example.com", "maxPages": 1}`
- **Task ID:** `84ddd92f-8620-45ff-8612-b54a185110a0`
- **Worker Log Verification:**
  ```text
  INFO msg="Processing scrape task" url="https://example.com" task_id="84ddd92f..."
  INFO msg="Scrape successful" url="https://example.com" engine="colly"
  ```
- **Observation:** The job was successfully enqueued to Redis, picked up by the Worker service, and processed.

## Conclusion
The application is fully functional in the Docker environment. The `chrome` dependency is satisfied by the Dockerfile configuration, making it safe to deploy to any container-supporting platform (like Leapcell).
