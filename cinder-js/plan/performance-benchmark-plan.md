# Performance Benchmark Plan

> **Purpose:** Define methodology for measuring and comparing Go vs JS performance  
> **When:** Execute during Phase 2 of implementation  
> **Last Updated:** 2026-02-02

---

## Table of Contents

1. [Objectives](#objectives)
2. [Test Environment](#test-environment)
3. [Benchmark Categories](#benchmark-categories)
4. [Test Scenarios](#test-scenarios)
5. [Measurement Methodology](#measurement-methodology)
6. [Success Criteria](#success-criteria)
7. [Reporting Template](#reporting-template)

---

## Objectives

### Primary Goals

1. **Validate memory assumptions** - Confirm JS version stays under 2GB at 10 concurrent contexts
2. **Measure cold start** - Ensure <5 seconds on Leapcell
3. **Compare latencies** - Confirm P95 within 20% of Go baseline

### Secondary Goals

1. Identify performance bottlenecks
2. Tune concurrency settings
3. Establish baseline metrics for monitoring

---

## Test Environment

### Hardware Requirements

| Environment   | Specification            | Purpose                      |
| ------------- | ------------------------ | ---------------------------- |
| Local Dev     | MacBook Pro M1, 16GB RAM | Initial testing              |
| Leapcell Dev  | 2GB RAM instance         | Quick smoke tests            |
| Leapcell Prod | 4GB RAM instance         | **Primary benchmark target** |

### Software Versions

```yaml
# Pin versions for reproducibility
go: 1.25.x
bun: 1.1.x
playwright: 1.50.x
chromium: matching playwright
redis: 7.x (Upstash)
```

### Baseline: Go Version

Before benchmarking JS, establish Go baselines:

```bash
# Run Go version with same configuration
docker run -m 4g --cpus 2 cinder-go:baseline

# Record metrics for each scenario
```

---

## Benchmark Categories

### Category 1: Cold Start

**Definition:** Time from container start to first successful request

**Measurement Points:**
```
T0: Container start signal
T1: Runtime loaded (Bun binaries in memory)
T2: Dependencies loaded (node_modules parsed)
T3: Server listening (HTTP port open)
T4: First request handled (full round-trip)
T5: Browser initialized (first dynamic scrape complete)
```

**Target:** T4 < 5 seconds, T5 < 8 seconds

### Category 2: Memory Usage

**Definition:** Heap + external memory under various loads

**Measurement Points:**
- Idle (no active scrapes)
- Light load (5 concurrent static)
- Medium load (5 concurrent dynamic)
- Heavy load (10 concurrent dynamic)
- Burst load (20 concurrent dynamic)

**Target:** Heavy load < 2GB

### Category 3: Throughput

**Definition:** Requests per second at various loads

**Scenarios:**
- Static scraping (Cheerio only)
- Dynamic scraping (Playwright)
- Queue processing (jobs/hour)

### Category 4: Latency

**Definition:** Request processing time (P50, P95, P99)

**Measurement:**
- End-to-end (client → response)
- Server processing only
- By scrape mode (static vs dynamic)

---

## Test Scenarios

### Scenario 1: Static Page Scrape

**URL:** `https://example.com` (lightweight, consistent)

**Test Matrix:**

| Metric             | Concurrency 1 | Concurrency 10 | Concurrency 50 |
| ------------------ | ------------- | -------------- | -------------- |
| Latency P50        | -             | -              | -              |
| Latency P95        | -             | -              | -              |
| Memory             | -             | -              | -              |
| Throughput (req/s) | -             | -              | -              |

**Go Baseline (Fill Before Testing):**

| Metric      | Concurrency 1 | Concurrency 10 | Concurrency 50 |
| ----------- | ------------- | -------------- | -------------- |
| Latency P50 | ~200ms        | ~250ms         | ~500ms         |
| Latency P95 | ~400ms        | ~600ms         | ~1200ms        |
| Memory      | ~50MB         | ~100MB         | ~200MB         |
| Throughput  | ~5 req/s      | ~40 req/s      | ~40 req/s      |

### Scenario 2: JavaScript-Heavy SPA

**URL:** React/Next.js site requiring Playwright

**Test Matrix:**

| Metric          | Concurrency 1 | Concurrency 5 | Concurrency 10 |
| --------------- | ------------- | ------------- | -------------- |
| Latency P50     | -             | -             | -              |
| Latency P95     | -             | -             | -              |
| Memory          | -             | -             | -              |
| Contexts active | -             | -             | -              |

**Go Baseline:**

| Metric      | Concurrency 1 | Concurrency 5 | Concurrency 10 |
| ----------- | ------------- | ------------- | -------------- |
| Latency P50 | ~1.5s         | ~2s           | ~3s            |
| Latency P95 | ~2.5s         | ~3.5s         | ~5s            |
| Memory      | ~300MB        | ~450MB        | ~550MB         |

### Scenario 3: Mixed Load (Realistic)

**Traffic Pattern:**
- 70% static scrapes
- 30% dynamic scrapes
- Random intervals

**Duration:** 5 minutes sustained load

**Metrics:**
- Total requests handled
- Error rate
- Memory high-water mark
- P95 latency

### Scenario 4: Queue Throughput

**Setup:**
- Pre-queue 100 jobs
- Start worker
- Measure time to completion

**Target:** 1000+ jobs/hour at 4GB RAM

### Scenario 5: Cold Start Timing

**Procedure:**
1. Stop container completely
2. Start container with fresh image
3. Time until:
   - HTTP 200 on `/health` (server ready)
   - HTTP 200 on `/v1/scrape?url=example.com&mode=static` (static ready)
   - HTTP 200 on `/v1/scrape?url=example.com&mode=dynamic` (browser ready)

**Target:** Full readiness < 5 seconds

---

## Measurement Methodology

### Memory Measurement

```javascript
// Memory snapshot function
function measureMemory() {
  const usage = process.memoryUsage();
  
  return {
    heapUsed: Math.round(usage.heapUsed / 1024 / 1024) + 'MB',
    heapTotal: Math.round(usage.heapTotal / 1024 / 1024) + 'MB',
    external: Math.round(usage.external / 1024 / 1024) + 'MB',
    rss: Math.round(usage.rss / 1024 / 1024) + 'MB',
    timestamp: Date.now(),
  };
}

// Continuous monitoring
setInterval(() => {
  console.log('Memory:', measureMemory());
}, 5000);
```

### Latency Measurement

```bash
# Using wrk for load testing
wrk -t12 -c100 -d30s \
  -s post.lua \
  http://localhost:8080/v1/scrape

# post.lua content:
# wrk.method = "POST"
# wrk.headers["Content-Type"] = "application/json"
# wrk.body = '{"url": "https://example.com", "mode": "static"}'
```

### Cold Start Measurement

```bash
#!/bin/bash
# cold-start-test.sh

# Kill any existing container
docker rm -f cinder-js-test 2>/dev/null

# Record start time
START=$(date +%s%3N)

# Start container
docker run -d --name cinder-js-test \
  -p 8080:8080 \
  -m 4g \
  cinder-js:test

# Wait for health check
while ! curl -s http://localhost:8080/health > /dev/null 2>&1; do
  sleep 0.1
done

# Record ready time
READY=$(date +%s%3N)

# Wait for static scrape
while ! curl -s -X POST http://localhost:8080/v1/scrape \
  -H "Content-Type: application/json" \
  -d '{"url":"https://example.com","mode":"static"}' > /dev/null 2>&1; do
  sleep 0.1
done

STATIC_READY=$(date +%s%3N)

# Wait for dynamic scrape
while ! curl -s -X POST http://localhost:8080/v1/scrape \
  -H "Content-Type: application/json" \
  -d '{"url":"https://example.com","mode":"dynamic"}' > /dev/null 2>&1; do
  sleep 0.1
done

DYNAMIC_READY=$(date +%s%3N)

# Calculate times
echo "Time to ready: $((READY - START))ms"
echo "Time to static: $((STATIC_READY - START))ms"
echo "Time to dynamic: $((DYNAMIC_READY - START))ms"
```

### Queue Throughput Measurement

```javascript
// Queue 100 jobs and measure processing time
async function measureQueueThroughput() {
  const startTime = Date.now();
  const jobCount = 100;
  const jobs = [];
  
  // Enqueue jobs
  for (let i = 0; i < jobCount; i++) {
    jobs.push(queue.add('scrape', {
      url: 'https://example.com',
      render: false,
    }));
  }
  
  await Promise.all(jobs);
  console.log(`Enqueued ${jobCount} jobs in ${Date.now() - startTime}ms`);
  
  // Wait for completion
  const completionStart = Date.now();
  
  while (true) {
    const [waiting, active] = await Promise.all([
      queue.getWaitingCount(),
      queue.getActiveCount(),
    ]);
    
    if (waiting === 0 && active === 0) break;
    await new Promise(r => setTimeout(r, 1000));
  }
  
  const totalTime = Date.now() - completionStart;
  const jobsPerHour = (jobCount / totalTime) * 3600000;
  
  console.log(`Processed ${jobCount} jobs in ${totalTime}ms`);
  console.log(`Throughput: ${Math.round(jobsPerHour)} jobs/hour`);
  
  return { jobCount, totalTime, jobsPerHour };
}
```

---

## Success Criteria

### Must Pass (Phase 2 Gate)

| Criterion                     | Target               | Measurement                 |
| ----------------------------- | -------------------- | --------------------------- |
| Memory at 10 dynamic contexts | < 2GB                | `process.memoryUsage().rss` |
| P95 latency (static)          | < 600ms (Go: ~500ms) | wrk benchmark               |
| P95 latency (dynamic)         | < 3.6s (Go: ~3s)     | wrk benchmark               |
| Cold start (full)             | < 5 seconds          | cold-start-test.sh          |

### Should Pass (Production Ready)

| Criterion              | Target           | Measurement              |
| ---------------------- | ---------------- | ------------------------ |
| Queue throughput       | > 1000 jobs/hour | measureQueueThroughput() |
| Error rate under load  | < 1%             | wrk errors/total         |
| Memory stability (1hr) | No leaks         | Continuous monitoring    |

### Nice to Have

| Criterion                 | Target      | Measurement         |
| ------------------------- | ----------- | ------------------- |
| Throughput (static)       | > 30 req/s  | wrk                 |
| Cold start (server ready) | < 2 seconds | curl /health timing |

---

## Reporting Template

### Benchmark Report: [Date]

#### Environment

```
Platform: Leapcell / Local Docker
Memory Limit: 4GB
CPU Limit: 2 cores
Go Version: x.x.x
Bun Version: x.x.x
Playwright Version: x.x.x
```

#### Go Baseline

| Metric                | Value     |
| --------------------- | --------- |
| Cold start (full)     | X ms      |
| Memory (idle)         | X MB      |
| Memory (10 contexts)  | X MB      |
| P95 latency (static)  | X ms      |
| P95 latency (dynamic) | X ms      |
| Queue throughput      | X jobs/hr |

#### JS Results

| Metric                | Value     | vs Go |
| --------------------- | --------- | ----- |
| Cold start (full)     | X ms      | +X%   |
| Memory (idle)         | X MB      | +X%   |
| Memory (10 contexts)  | X MB      | +X%   |
| P95 latency (static)  | X ms      | +X%   |
| P95 latency (dynamic) | X ms      | +X%   |
| Queue throughput      | X jobs/hr | +X%   |

#### Pass/Fail

| Criterion       | Target   | Actual | Status |
| --------------- | -------- | ------ | ------ |
| Memory < 2GB    | < 2048MB | X MB   | ✅/❌    |
| P95 < 20% of Go | < X ms   | X ms   | ✅/❌    |
| Cold start < 5s | < 5000ms | X ms   | ✅/❌    |
| Queue > 1000/hr | > 1000   | X      | ✅/❌    |

#### Observations

- [Notable findings]
- [Unexpected behaviors]
- [Optimization recommendations]

#### Decision

☐ **PASS** - Proceed to Phase 3  
☐ **FAIL** - Abort cinder-js, continue with Go  
☐ **CONDITIONAL** - Proceed with specific constraints

---

## Appendix: Benchmark Scripts

### wrk POST Script

```lua
-- post_scrape.lua
wrk.method = "POST"
wrk.headers["Content-Type"] = "application/json"
wrk.body = '{"url":"https://example.com","mode":"static"}'

response = function(status, headers, body)
   if status ~= 200 then
      print("Error: " .. status)
   end
end
```

### Usage

```bash
# Static scraping benchmark
wrk -t4 -c10 -d30s -s post_scrape.lua http://localhost:8080/v1/scrape

# Dynamic scraping benchmark (lower concurrency)
wrk -t2 -c5 -d30s -s post_dynamic.lua http://localhost:8080/v1/scrape
```

---

*Document Version: 1.0.0-draft*  
*Last Updated: 2026-02-02*
