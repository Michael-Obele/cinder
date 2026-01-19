# Performance & Reliability Analysis

## 🚨 Critical Bottlenecks

### 1. Browser Process Management (The "Anti-Pattern")
**Severity: Critical**
- **Current Behavior:** `internal/scraper/chromedp.go` calls `chromedp.NewExecAllocator` inside the `Scrape` method.
- **Impact:** Every single URL scrape spawns a **new, fresh Chromium process**.
- **Cost:**
  - **Latency:** ~500ms - 2000ms overhead *just to start the browser* per request.
  - **Memory:** Each process has its own substantial valid memory footprint.
  - **Stability:** frequent process forking/killing destabilizes the container environment.
- **Fix:** Move the `Allocator` to the service level (Singleton). Start one browser (or a fixed pool), and spawn *tabs* (`NewContext`) for each request.

### 2. Leapcell Hobby Quota (The Hidden Killer)
**Severity: Critical**
- **Constraint:** The Hobby plan includes **4GB RAM** (Generous) but only **3 GB-hours** of execution time per month.
- **Current Architecture:** Separate `cmd/api` and `cmd/worker` services.
- **Impact:**
    - Running a separate Worker container 24/7 is impossible (would exceed quota in ~3 hours).
    - Separation doubles the cold-start overhead.
- **Fix:**
    - **Monolith Mode**: Run the `Asynq` server *inside* the API process (concurrency permitting).
    - This ensures we only burn quota when handling API requests or processing immediate jobs.
    - 4GB RAM is sufficient to run both components and the browser in one container.

### 3. Upstash/Redis Latency & Costs
**Severity: Moderate**
- **Current Behavior:** `asynq` is configured with a 5s polling interval (`TaskCheckInterval`).
- **Impact:**
  - **Latency:** Jobs might sit in the queue for up to 5 seconds before being picked up.
  - **Cost:** Standard polling generates constant network traffic and Redis commands.
- **Constraint:** Upstash is serverless and billing is often command-based or request-based, but latency is the bigger issue here since the worker and Redis are likely not in the same VPC (Leapcell <-> Upstash).
- **Fix:**
  - Tune connection timeouts.
  - Optimization of the `asynq` server config.
  - Consider "Group" aggregation if tasks are small (though scraping tasks are long-running, so less critical).

### 4. Worker Concurrency
**Severity: High**
- **Current Behavior:** Concurrency set to `2`.
- **Reasoning:** Likely set low to avoid crashing the container due to Issue #1 (Browser Spawning).
- **Fix:** With **4GB RAM** available on Hobby tier, we can safely increase concurrency to **10+**. The bottleneck is CPU/Network, not Memory.

### 5. Naive Wait Strategy
**Severity: Moderate**
- **Current Behavior:** `chromedp.WaitVisible("body")`.
- **Impact:** SPAs (React/Vue) often render a generic `<body><div id="root"></div></body>` instantly, then load content via JS. The scraper returns empty/loading shells.
- **Fix:** Implement heuristic waiting (e.g., `WaitReady`, wait for Network Idle, or a fixed small delay after load).

---

## 🏗️ Architecture Improvements for "Leapcell + Upstash"

### Resource Constraints
Leapcell wraps the Go binary in a container. We must ensure:
1.  **Memory Safety:** The shared browser doesn't leak memory over time. (Chrome is notorious for this).
2.  **Zombie Cleanup:** Ensure the main browser process is killed if the Go app crashes.

### Reliability
1.  **Context Timeouts:** Hard timeouts on scraping (already implemented, but needs tuning).
2.  **Retry Strategy:** Asynq handles this, but we need to ensure we don't retry "Fatal" errors (like 404s) endlessly.
