# Performance Benchmark Plan

## Objectives

- Measure cold start time, memory per context, and throughput under load.
- Validate if Bun/JS can meet **<5s cold start** and **<2GB memory** at 10 dynamic scrapes.

## Test Environment

- **Platform**: Leapcell 4GB RAM tier
- **Runtime**: Bun 1.1+
- **Browser**: Playwright Chromium
- **Redis**: Managed Redis (TLS)

## Bun Benchmarking Tools (Bun Docs)

- **Load testing**: use tools that can keep up with Bun’s HTTP server performance.
  - Recommended: **bombardier**, **oha**, or **http_load_test** (uSockets example tool)
- **CPU profiling**: Bun `--cpu-prof` / `--cpu-prof-md`
- **Heap profiling**: Bun `--heap-prof` / `--heap-prof-md`
- **JS heap stats**: `bun:jsc` `heapStats()`

## Key Metrics

- **Cold start**: container start → first 200 OK response
- **P95 latency**: per request, static/dynamic/smart
- **Memory per context**: RSS delta per new context
- **Queue throughput**: jobs/hour with BullMQ

## Scenarios

### Scenario A — Static Page

- Target: simple HTML (no JS)
- Concurrency: 10, 50, 100
- Expected: Cheerio path, low latency

### Scenario B — JS-heavy SPA

- Target: React/Vue app
- Concurrency: 10, 25
- Expected: Playwright path

### Scenario C — Mixed Smart Mode

- 70% static, 30% dynamic
- Concurrency: 10, 50

### Scenario D — Framework Overhead (Hono vs Elysia)

- Same endpoints and middleware for both frameworks
- Measure P95 latency and throughput at 10/50 concurrency
- Validate request parsing + validation overhead with Valibot

## Success Criteria

- Cold start < 5 seconds
- P95 latency within 20% of Go version
- Memory < 2GB at 10 concurrent Playwright scrapes
- Throughput ≥ 1000 jobs/hour at 4GB RAM

## Instrumentation Plan

- Collect container RSS and CPU usage
- Track queue wait time, processing time, retries
- Log per request: `engine`, `duration_ms`, `bytes_in`, `bytes_out`

## Measurement Procedure

1. Warm container to establish baseline.
2. Restart container to measure cold start.
3. Run each scenario for 10 minutes with steady load.
4. Capture P95, P99, and peak memory.

## Reporting

- Provide a per-scenario table with key metrics.
- Highlight failures vs success criteria.
