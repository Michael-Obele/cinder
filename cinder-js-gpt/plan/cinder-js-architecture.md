# Cinder-JS Architecture & ADRs

## Executive Summary

Cinder-JS is a **documentation-only** proposal for porting the Go-based Cinder scraping API to **Bun + Hono/Elysia** while preserving the monolith deployment pattern (API + worker in one container). The proposal is **conditionally viable** if the following can be proven in benchmarks:

- **Cold start** under 5 seconds on Leapcell
- **Memory under 2GB** with 10 concurrent dynamic scrapes
- **Queue throughput** at 1000+ jobs/hour with 4GB RAM

**Recommendation**: **Go** for a prototype/benchmark phase, **conditional Go/No-Go** for production pending memory-per-context and cold-start validation.

## System Goals

- Match current Cinder API behavior and response schema.
- Preserve smart mode fallback chain and monolith deployment model.
- Keep operational simplicity for Leapcell deployments.
- Maintain anti-detection capabilities comparable to undetected-chromedp.

## Bun Runtime Findings (2026)

- **Strengths**: fast startup, built-in `fetch`, integrated package manager/bundler, strong TypeScript ergonomics, good fit for single-container deployments where cold-start is a key cost driver.
- **Limitations / Risks**: Node.js compatibility is good but **not complete**; some packages assume Node-specific internals. Playwright does not advertise first-class Bun support, so compatibility must be proven via POC and deployment tests.
- **Operational Implication**: treat Playwright + Bun as a **compatibility risk** until validated on Leapcell; assume extra engineering time for runtime-specific fixes.
- **Workers**: Bun’s `Worker` API is available but **experimental** (notably termination). Use cautiously for the BullMQ worker thread and document lifecycle controls.

## Proposed System Topology

```
┌─────────────────────────────────────────────┐
│                 Container                   │
│                                             │
│  ┌──────────────────┐   ┌──────────────────┐ │
│  │ Hono/Elysia API   │◄─►│ BullMQ Queue     │ │
│  │ /v1/scrape        │   │ (Redis-backed)  │ │
│  │ /v1/crawl         │   └─────────┬────────┘ │
│  └────────┬─────────┘             │            │
│        │                     ▼              │
│        │           ┌──────────────────────┐ │
│        │           │ Worker Thread        │ │
│        │           │ (BullMQ Worker)      │ │
│        │           └─────────┬────────────┘ │
│        ▼                     ▼              │
│  ┌───────────────────────────────────────┐ │
│  │ Scraper Service                        │ │
│  │ - Fetch (fast path)                    │ │
│  │ - Cheerio (static)                     │ │
│  │ - Playwright (dynamic)                 │ │
│  └───────────────────────────────────────┘ │
└─────────────────────────────────────────────┘
```

## Framework Evaluation: Hono vs Elysia (Bun-First)

- **Hono**: minimal, Web-standards compliant, multi-runtime (Bun/Deno/Workers/Node), excellent for portability and edge alignment.
- **Elysia**: Bun-first ergonomics, stronger type safety and DX focus, often benchmarked as faster on Bun, but less cross-runtime portability.
- **Decision Criteria**:
  - **Throughput & latency** on Bun under real API workloads (not just hello-world).
  - **Middleware and plugin ecosystem** maturity (logging, validation, rate limiting, OpenAPI).
  - **Type-safety guarantees** for handler inputs/outputs.
  - **Operational constraints** (compatibility with Bun runtime quirks, structured logging, request context model).
- **Action**: run benchmark comparisons on Bun (realistic endpoints + queue enqueue) before finalizing framework selection.

## ADRs (Architecture Decision Records)

### ADR-001: Runtime = Bun v1.1+

- **Context**: Faster startup vs Node; built-in fetch and modern JS runtime.
- **Decision**: Bun is the only supported runtime.
- **Consequences**: Faster cold starts; **compatibility risk** for libraries that rely on Node internals. Playwright integration must be validated in an early POC.

### ADR-002: HTTP Framework = Hono or Elysia (TBD)

- **Context**: Hono is lightweight and standards-compliant. Elysia is Bun-first with stronger type safety and potential performance advantages.
- **Decision**: **TBD pending Bun benchmarks**. Default to Hono unless Elysia shows a clear throughput/latency advantage with comparable middleware maturity.
- **Consequences**: Framework choice affects request handling overhead, middleware availability, and TypeScript type ergonomics.

### ADR-003: Static Scraping = Fetch + Cheerio

- **Context**: Fast, low-memory static rendering required.
- **Decision**: Use Bun’s native fetch; parse HTML with Cheerio.
- **Consequences**: Comparable to Colly for static HTML; no JS execution.

### ADR-004: Dynamic Scraping = Playwright

- **Context**: Need full JS rendering for SPAs.
- **Decision**: Use Playwright with Chromium contexts.
- **Consequences**: Higher memory per context than chromedp tabs; must control concurrency.

### ADR-005: Queue System = BullMQ + Redis

- **Context**: Asynq equivalent for Node ecosystem with retries and job metadata.
- **Decision**: BullMQ Worker for `/crawl`.
- **Consequences**: Manage worker concurrency; consider sandboxed processors or worker threads for CPU-heavy tasks.

### ADR-006: Markdown Conversion = @turndown/turndown

- **Context**: HTML → Markdown with predictable output.
- **Decision**: Use Turndown.
- **Consequences**: Must validate output parity against Go’s html-to-markdown/v2.

### ADR-007: Config Validation = Valibot

- **Context**: Lightweight schema-based validation with minimal bundle size.
- **Decision**: Valibot for env/config validation.
- **Consequences**: Clear validation rules; must design a schema for all env vars.

### ADR-008: Logging = Pino

- **Context**: Structured logging similar to slog.
- **Decision**: Use Pino for JSON logs.
- **Consequences**: Low overhead; need consistent log fields across API and worker.

### ADR-009: Monolith Deployment Pattern

- **Context**: Leapcell pay-per-container favors single container strategy.
- **Decision**: Run API + worker in one container using worker threads.
- **Consequences**: Must implement explicit thread lifecycle management and shutdown hooks.

## Feature Parity Matrix (Go vs JS)

| Capability      | Go (Current) | JS (Planned) | Gap/Notes                             |
| --------------- | ------------ | ------------ | ------------------------------------- |
| `/v1/scrape`    | ✅           | ✅           | Parity expected                       |
| `/v1/crawl`     | ✅           | ✅           | BullMQ semantics differ from Asynq    |
| `/v1/search`    | ✅ (Brave)   | ⚠️ TBD       | Not included in MVP                   |
| Smart mode      | ✅           | ✅           | Requires new heuristics doc           |
| Browser pooling | ✅           | ⚠️ Partial   | Playwright contexts vs chromedp tabs  |
| Anti-detection  | ✅           | ✅           | Via playwright-extra + stealth        |
| Redis TLS       | ✅           | ✅           | Must ensure BullMQ ioredis TLS config |

## Top Risks & Mitigations

1. **Playwright memory cost**: Limit concurrency, recycle contexts, benchmark memory per context.
2. **Bun + Playwright compatibility**: Validate integration in Phase 1/2 before full migration.
3. **Bun runtime compatibility gaps**: Audit dependencies for Node-only assumptions; prefer Bun-native or web-standards APIs.
4. **Worker thread lifecycle**: Implement controlled shutdown (drain queue, close contexts).

## Cost Projection (Leapcell)

- **Base target**: 4GB RAM container + managed Redis.
- **Assumptions**: Single container, pay-per-compute-minute, low idle cost.
- **Cost drivers**: dynamic scrape concurrency, Playwright startup times, Redis I/O.
- **Action**: confirm pricing against Leapcell dashboard during Phase 0.

## Go/No-Go Gate

Proceed to build only if benchmarks show:

- < 5s cold start
- < 2GB memory at 10 concurrent dynamic scrapes
- 1000+ jobs/hour via BullMQ on 4GB RAM
