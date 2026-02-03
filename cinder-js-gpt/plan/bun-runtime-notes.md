# Bun Runtime Notes (Cinder-JS)

## Purpose
Document Bun-specific strengths, limitations, and operational implications for Cinder-JS. This is Bun-focused (not Node.js).

## Strengths (Bun-First)
- **Fast startup** and low runtime overhead in many workloads.
- **Built-in Web APIs** (notably `fetch`) reduce dependency weight and simplify the static scrape path.
- **Integrated tooling** (runtime, package manager, bundler) reduces build surface area.
- **Server-side workers** are supported and can be used to isolate the BullMQ worker from the API loop.

## Compatibility & Limitations
- Bun tracks **Node.js API compatibility**, but it is **not complete**. The official compatibility matrix lists supported Node modules/globals and is the authoritative reference.
- **Dependency risk**: any library that assumes Node-only internals (or undocumented Node behavior) may break or behave differently.
- **Playwright compatibility is not guaranteed** by Bun itself; treat as a risk until verified with a Bun-native POC on Leapcell.

## Workers (Concurrency)
- Bun implements a **Web Workers-style API** for server-side concurrency.
- The Bun `Worker` API is **experimental**, including termination behavior (per Bun docs).
- **Memory pressure**: Bun offers a `smol` worker mode to reduce memory usage at a performance cost. This may be relevant for Playwright-heavy workloads.
- `worker.ref()` / `worker.unref()` allow the worker to keep the process alive or be detached.

## Benchmarking Guidance (Bun Docs)
Bun recommends **fast load-testing tools** that can keep up with Bun’s HTTP server performance:
- **bombardier**
- **oha**
- **http_load_test** (uSockets example tool)

For profiling:
- CPU profiling via Bun’s `--cpu-prof` / `--cpu-prof-md`
- Heap profiling via `--heap-prof` / `--heap-prof-md`
- JS heap stats via `bun:jsc` heapStats (for measuring JS heap usage)

## Implications for Cinder-JS
- **Cold start**: likely favorable, but container + Playwright initialization remains the dominant cost.
- **Memory**: Bun runtime baseline + Playwright contexts require strict concurrency caps.
- **Queue isolation**: use Bun workers for the BullMQ worker loop to avoid blocking the API.
- **Compatibility gating**: treat Bun + Playwright as a feasibility gate (Phase 0/1), not a safe default.

## Open Questions
- What is the **memory-per-context** of Playwright under Bun compared with Go/Chromedp?
- Are there **Bun-specific Playwright issues** on Leapcell (sandboxing, Chromium dependencies)?
- Does `smol` mode materially reduce worker memory in scraping workloads without harming throughput?

## References
- Bun Node.js compatibility matrix: https://bun.com/docs/runtime/nodejs-compat
- Bun Workers API: https://bun.com/docs/runtime/workers
- Bun benchmarking guidance: https://bun.com/docs/project/benchmarking
