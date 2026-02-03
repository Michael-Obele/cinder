# Go vs Bun/JS Comparison Report

## Summary

This report compares the current Go implementation with the proposed **Bun-first** stack. It highlights predicted performance tradeoffs, concurrency behavior, memory costs, and maintenance implications. **All conclusions must be validated on Bun (not Node.js).**

## Comparative Table

| Metric                 | Go (Current)                   | Bun/JS (Proposed)           | Notes                                                                |
| ---------------------- | ------------------------------ | --------------------------- | -------------------------------------------------------------------- |
| Cold Start             | Native binary (~fast)          | Bun runtime init            | Bun start-up must be tested on Leapcell                              |
| Memory Footprint       | ~50MB binary + shared Chromium | Bun runtime + Playwright    | Playwright contexts likely heavier than chromedp tabs                |
| Concurrency Model      | Goroutines                     | Event loop + worker threads | Worker threads duplicate runtime state                               |
| Browser Context Weight | Chromedp tabs                  | Playwright contexts         | Playwright contexts are “fast and cheap” but still heavier than tabs |
| Queue Throughput       | Asynq workers                  | BullMQ workers              | Concurrency and worker scaling must be tuned                         |
| Framework Choice       | Gin                            | Hono or Elysia              | Compare throughput + DX on Bun only                                  |
| Dev Experience         | Compile-time safety            | Runtime JS + async/await    | Faster iteration, more runtime pitfalls                              |
| Bundle Size            | Single binary                  | node_modules + runtime      | Larger image size, slower deploys                                    |
| Runtime Compatibility  | N/A                            | Bun Node-compat layer       | Not 100% compatible; validate dependencies                           |

## Cold Start

- Go starts as a native binary with minimal runtime overhead.
- Bun must initialize runtime + load dependencies; Playwright browser launch adds latency.
- **Prediction**: With lazy browser init, cold start can remain under 5s.

## Memory Footprint

- Go’s chromedp uses a shared allocator with lightweight tabs.
- Playwright contexts are isolated and designed to be cheap to create but still incur higher per-context overhead.
- **Prediction**: Memory per dynamic scrape is the primary bottleneck for JS.

## Runtime Compatibility (Bun)

- Bun targets high Node.js API compatibility, but the compatibility matrix is the authoritative source for what is implemented.
- Dependencies that assume Node internals may fail; this is a **migration risk** for Playwright integrations, Redis clients, and observability tooling.
- Require a Bun-first dependency audit before implementation.

## Concurrency Model

- Go: Goroutines are lightweight (2KB stacks) and efficient for IO-heavy concurrency.
- JS: Single event loop for async IO + worker threads for CPU-heavy tasks.
- BullMQ supports concurrency in a single worker and scaling across multiple workers.

## Framework Choice: Hono vs Elysia (Bun)

- **Hono**: web-standards API, broad runtime support, lightweight.
- **Elysia**: Bun-first ergonomics, stronger type safety, performance claims based on benchmark suites.
- Decision requires Cinder-specific benchmarks (routing, validation, queue enqueue, scrape path).

## Queue Throughput

- BullMQ workers process jobs asynchronously; concurrency can be configured per worker and scaled horizontally.
- Sandboxed processors or worker threads can reduce stalled jobs but incur runtime duplication overhead.

## Developer Experience

- JS offers native async/await and more familiar ecosystem tooling.
- Go offers strong compile-time checks; JS requires stricter runtime validation (Valibot).

## Maintenance Burden

- JS stack increases dependency surface area and security updates.
- Playwright and browser dependencies may require frequent updates.

## Predicted Bottleneck Order

1. **Memory (Playwright contexts)**
2. **CPU for JS-heavy rendering**
3. **Redis I/O for high queue throughput**

## Open Questions

- Memory per Playwright context under realistic workloads.
- Bun + Playwright compatibility and stability on Leapcell.
- Throughput ceiling at 4GB RAM with 10-50 concurrent dynamic jobs.
