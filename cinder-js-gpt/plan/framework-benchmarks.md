# Bun Framework Benchmarks (Hono vs Elysia)

## Purpose
Summarize available benchmark sources and document how we will compare **Hono vs Elysia** on **Bun** for Cinder-JS. This doc avoids hard claims beyond published sources and highlights what remains to be validated in our own benchmarks.

## Published Sources (Directional Only)
- **Hono benchmarks** show routing/middleware performance across runtimes, including Bun, with published benchmark artifacts and images. These are router-level tests and do not reflect full application workloads. Source: https://hono.dev/docs/concepts/benchmarks
- **Elysia vs Hono** guidance from Elysia docs emphasizes Bun-first optimization, sound type safety, and published performance claims based on TechEmpower JSON tests. Source: https://elysiajs.com/migrate/from-hono
- **Web Frameworks Benchmark** comparison pages list Hono/Elysia results in a shared benchmark suite. Use only as directional signal. Source: https://web-frameworks-benchmark.netlify.app/compare?f=elysia,hono,hono-bun,h3

## What the Sources Indicate (Non-Numeric)
- **Hono**: multi-runtime, Web Standards aligned, typically strong router throughput on Bun and edge platforms; performance evidence is mostly router-level.
- **Elysia**: Bun-first framework with strong type safety and lifecycle hooks; claims higher performance and better OpenAPI ergonomics on Bun.
- **Actionable takeaway**: both are viable; we need **Cinder-specific benchmarks** to decide (not generic hello-world tests).

## Cinder-Specific Benchmark Plan (Required)
Run these on **Bun** with identical handlers and middleware:

1. **Routing-only test**
   - 10–50 routes, zero middleware
   - Confirms baseline overhead

2. **Validation-heavy test**
   - Request body + query + header validation (Valibot)
   - Measures validation integration costs in each framework

3. **Queue enqueue test**
   - `/v1/crawl` enqueue path: latency & throughput
   - Measures framework impact on high-frequency enqueue

4. **Scrape API path (non-Playwright)**
   - Fast `fetch → Cheerio → Turndown` pipeline
   - Measures framework overhead under real handler load

5. **Mixed workload test**
   - 70% fast endpoints, 30% scrape endpoints
   - Simulates real traffic

## Decision Criteria
- **Throughput** under comparable latency budgets
- **Latency** P95 with moderate concurrency (10–50)
- **Memory overhead**: total RSS including framework + middleware
- **DX**: type inference, ergonomics for request/response typing
- **Operational fit**: lifecycle hooks, structured logging integration, OpenAPI workflow

## Decision Rule
- Default to **Hono** unless **Elysia shows a clear win** in measured throughput/latency **and** middleware ergonomics are equal or better for our needs.

## References
- Hono benchmarks: https://hono.dev/docs/concepts/benchmarks
- Elysia migration guide (Hono comparison + performance note): https://elysiajs.com/migrate/from-hono
- Web Frameworks Benchmark comparison: https://web-frameworks-benchmark.netlify.app/compare?f=elysia,hono,hono-bun,h3
