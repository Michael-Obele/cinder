# Cinder JS - Architecture RFC

> **RFC Status:** Draft  
> **Author:** Technical Architecture Team  
> **Last Updated:** 2026-02-03
> **Revision:** Updated memory baselines (JSC vs V8) and performance data

---

## Table of Contents

1. [Executive Summary](#executive-summary)
2. [Recommendation](#recommendation)
3. [Problem Statement](#problem-statement)
4. [Proposed Architecture](#proposed-architecture)
5. [Architecture Decision Records](#architecture-decision-records)
6. [Feature Parity Matrix](#feature-parity-matrix)
7. [Risk Assessment](#risk-assessment)
8. [Cost Projection](#cost-projection)
9. [Success Criteria](#success-criteria)
10. [Open Questions](#open-questions)

---

## Executive Summary

### Context

Cinder is a high-performance Go-based web scraping API that converts websites into LLM-ready markdown. It features:
- Dual-mode scraping (static Colly + dynamic Chromedp)
- Redis-backed async queues (Asynq)
- Smart auto-detection of dynamic content
- "Monolith" deployment pattern (API + worker in one container)

### Proposed Change

Evaluate porting Cinder to a JavaScript/TypeScript stack:
- **Runtime:** Bun 1.1+ (for speed and unified toolchain)
- **Framework:** Hono (lightweight, Web Standard compliant)
- **Browser Automation:** Playwright (replacing Chromedp)
- **Queue:** BullMQ (replacing Asynq)

### Key Findings

| Metric               | Go (Current) | JS (Projected) | Assessment            |
| -------------------- | ------------ | -------------- | --------------------- |
| Cold Start           | ~1-2s        | ~3-5s          | ⚠️ Regression expected |
| Memory (idle)        | ~50MB        | ~45-65MB       | ✅ Comparable baseline |
| Memory (10 contexts) | ~500MB       | ~700MB         | ✅ Manageable          |
| Dev Velocity         | Moderate     | High           | ✅ Improvement         |
| Maintenance          | Moderate     | Lower          | ✅ Improvement         |
| Ecosystem            | Go-specific  | NPM (vast)     | ✅ Larger ecosystem    |

---

## Recommendation

### 🟡 Conditional Go-Ahead

**Proceed with Phase 1 prototype** to validate memory assumptions before full commitment.

**Rationale:**
1. **Developer Experience Gains:** JS/TS offers faster iteration, easier debugging, and broader talent pool
2. **Memory Risk is Manageable:** Leapcell's 4GB limit provides ~3x headroom over projected needs
3. **Performance Trade-offs Acceptable:** Cold start regression is mitigable with keep-warm strategies
4. **Reversible Decision:** Phase 1 can be abandoned with minimal investment if benchmarks fail

**Gate Criteria for Phase 2:**
- [ ] Memory usage under 2GB at 10 concurrent dynamic scrapes
- [ ] P95 latency within 20% of Go baseline
- [ ] Cold start under 5 seconds on Leapcell

---

## Problem Statement

### Current State

The Go-based Cinder works well but presents maintenance challenges:

1. **Talent Pool:** Fewer Go developers familiar with Chromedp
2. **Iteration Speed:** Compile cycle slower than interpreted languages
3. **Debugging:** CDP debugging in Go requires specialized knowledge
4. **Ecosystem:** Smaller selection of scraping utilities vs. JS/Node

### Desired State

A JavaScript/TypeScript implementation that:
- Maintains 100% API compatibility
- Achieves comparable performance within acceptable margins
- Improves developer velocity and maintainability
- Deploys seamlessly on Leapcell's infrastructure

---

## Proposed Architecture

### System Design

```
┌─────────────────────────────────────────────────────────────────────┐
│                         CONTAINER (Leapcell 4GB)                    │
├─────────────────────────────────────────────────────────────────────┤
│                                                                     │
│  ┌─────────────────────┐    ┌──────────────────────────────────┐   │
│  │   Main Thread       │    │      Worker Thread               │   │
│  │   (Bun Runtime)     │    │      (BullMQ Worker)             │   │
│  │                     │    │                                  │   │
│  │  ┌───────────────┐  │    │  ┌────────────────────────────┐  │   │
│  │  │  Hono Server  │  │    │  │  Queue Processor           │  │   │
│  │  │               │  │    │  │                            │  │   │
│  │  │ • /v1/scrape  │◄─┼────┼──┤  • Job handler             │  │   │
│  │  │ • /v1/crawl   │  │    │  │  • Concurrency: 10         │  │   │
│  │  │ • /v1/search  │  │    │  │  • Retry logic             │  │   │
│  │  └───────┬───────┘  │    │  └────────────┬───────────────┘  │   │
│  │          │          │    │               │                  │   │
│  │  ┌───────▼───────┐  │    │  ┌────────────▼───────────────┐  │   │
│  │  │ Scraper       │  │    │  │  Scraper Service           │  │   │
│  │  │ Service       │  │    │  │  (shared instance)         │  │   │
│  │  │               │  │    │  │                            │  │   │
│  │  │ • Mode select │◄─┼────┼──┤  • Same scraping logic     │  │   │
│  │  │ • Caching     │  │    │  │  • Browser context pool    │  │   │
│  │  └───────┬───────┘  │    │  └────────────────────────────┘  │   │
│  │          │          │    │                                  │   │
│  └──────────┼──────────┘    └──────────────────────────────────┘   │
│             │                                                       │
│  ┌──────────▼───────────────────────────────────────────────────┐  │
│  │                    Browser Pool (Playwright)                  │  │
│  │                                                               │  │
│  │  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐           │  │
│  │  │ Context 1   │  │ Context 2   │  │ Context N   │           │  │
│  │  │ (Tab)       │  │ (Tab)       │  │ (Tab)       │           │  │
│  │  └─────────────┘  └─────────────┘  └─────────────┘           │  │
│  │                                                               │  │
│  │  Single Browser Instance (shared allocator pattern)          │  │
│  └───────────────────────────────────────────────────────────────┘  │
│                                                                     │
└─────────────────────────────────────────────────────────────────────┘
                              │
                              ▼
                    ┌─────────────────┐
                    │   Redis Cloud   │
                    │   (Upstash/     │
                    │   Leapcell)     │
                    └─────────────────┘
```

### Request Flow

#### Synchronous Scrape (`POST /v1/scrape`)

```
Request → Hono Router → Valibot Validation → Scraper Service
    │                                              │
    │                                    ┌─────────┴─────────┐
    │                                    │                   │
    │                              Smart Mode           Explicit Mode
    │                                    │                   │
    │                         ┌──────────┴──────────┐        │
    │                         │                     │        │
    │                    Check Cache          Try Static     │
    │                         │                     │        │
    │                  Cache Hit?            Heuristics      │
    │                    │     │              Detect         │
    │                   Yes   No                  │          │
    │                    │     │         ┌────────┴────────┐ │
    │                    │     │    Need Dynamic?    No     │ │
    │                    │     │         │           │      │ │
    │                    │     │        Use       Return    │ │
    │                    │     │    Playwright    Static    │ │
    │                    │     │         │           │      │ │
    │                    ▼     ▼         ▼           ▼      ▼ │
    └───────────────────────── Return Result ◄───────────────┘
```

#### Asynchronous Crawl (`POST /v1/crawl`)

```
Request → Hono Router → Valibot Validation
    │
    ▼
Create Job ID (UUID)
    │
    ▼
Enqueue to BullMQ
    │
    ▼
Return 202 Accepted + Job ID
    │
    │
    │   ┌────────────────────────────────────────────┐
    │   │              Worker Thread                  │
    │   │                                            │
    └───┼───► BullMQ Worker picks up job             │
        │         │                                  │
        │         ▼                                  │
        │    Process Crawl                           │
        │         │                                  │
        │         ▼                                  │
        │    Store Result in Redis                   │
        │                                            │
        └────────────────────────────────────────────┘

Polling: GET /v1/crawl/:id → Retrieve job status/result from Redis
```

### Monolith Pattern Details

The critical challenge is running **API server + queue worker in one container** on Leapcell:

```typescript
// Conceptual architecture (not implementation code)
// 
// Main Thread: Hono HTTP server
// Worker Thread: BullMQ processor
// 
// Communication: Shared Redis connection pool
// Browser Pool: Shared between threads via lazy initialization
```

**Key Considerations:**

1. **Worker Threads (Not Goroutines):** Node.js worker_threads require explicit thread management
2. **Redis Connection Sharing:** Use connection pooling to avoid exhausting connections
3. **Browser Context Sharing:** Single browser instance, contexts created per-scrape
4. **Graceful Shutdown:** Drain queue → Close contexts → Close browser → Exit

---

## Architecture Decision Records

### ADR-001: Bun as Runtime

**Status:** Accepted

**Context:**
Need a JavaScript runtime with fast startup and good TypeScript support.

**Decision:**
Use **Bun 1.1+** instead of Node.js.

**Rationale:**
1. **Startup Speed:** Bun starts 4x faster than Node.js (~30ms vs ~120ms)
2. **Native TypeScript:** No transpilation step required
3. **Built-in Bundler:** Simplifies build pipeline
4. **Unified Toolchain:** Package manager, test runner, bundler in one

**Consequences:**
- Positive: Faster cold starts, simpler tooling
- Negative: Smaller ecosystem than Node.js, potential edge-case incompatibilities

**Risks:**
- Playwright officially supports Node.js; Bun compatibility tested but not guaranteed
- Some npm packages may have Bun-specific issues

---

### ADR-002: Hono as Web Framework

**Status:** Accepted

**Context:**
Need a lightweight web framework that works with Bun.

**Decision:**
Use **Hono** instead of Express or Fastify.

**Rationale:**
1. **Web Standard Compliant:** Uses native Request/Response objects
2. **Lightweight:** ~14KB minified (vs Express ~200KB)
3. **Multi-Runtime:** Works on Bun, Node, Deno, Cloudflare Workers
4. **Stability:** More stable and predictable than Elysia (which had recent regressions)

> **Note on Elysia:** While Elysia 1.4+ is faster (~300k req/s vs Hono's ~200k req/s), Hono was chosen for stability, portability, and wider team familiarity. The HTTP layer is not the bottleneck in this scraping application.

**Consequences:**
- Positive: Fast, small, portable
- Negative: Smaller community than Express, fewer middleware options

---

### ADR-003: Playwright Over Puppeteer

**Status:** Accepted

**Context:**
Need browser automation library for dynamic scraping.

**Decision:**
Use **Playwright** instead of Puppeteer.

**Rationale:**
1. **Multi-Browser:** Chromium, Firefox, WebKit support
2. **Better API:** Auto-waiting, network interception built-in
3. **Stealth Ecosystem:** playwright-extra + stealth plugin available
4. **Active Development:** Microsoft backing, frequent updates

**Consequences:**
- Positive: Better DX, more features, stealth plugin ecosystem
- Negative: Slightly higher memory per context than Chromedp (~50MB vs ~30MB)

**Comparison with Chromedp:**
| Feature         | Chromedp (Go)          | Playwright (JS)          |
| --------------- | ---------------------- | ------------------------ |
| Memory per tab  | ~30MB                  | ~50MB                    |
| API Complexity  | Moderate               | Lower (auto-wait)        |
| Stealth Plugins | undetected-chromedp    | playwright-extra-stealth |
| Debugging       | CDP knowledge required | Inspector built-in       |

---

### ADR-004: BullMQ for Queue System

**Status:** Accepted

**Context:**
Need Redis-backed job queue for async crawling.

**Decision:**
Use **BullMQ** to replace Asynq.

**Rationale:**
1. **Production-Ready:** Battle-tested at scale (GitLab, Microsoft)
2. **Feature Parity:** Priorities, retries, delayed jobs, rate limiting
3. **Active Maintenance:** Regular updates, TypeScript support
4. **Worker Threads Support:** Can run workers in separate threads

**Consequences:**
- Positive: Full feature parity with Asynq
- Negative: More complex than simple job queues

**Configuration Mapping:**
| Asynq (Go)          | BullMQ (JS)                  |
| ------------------- | ---------------------------- |
| `Concurrency: 10`   | `concurrency: 10`            |
| Queue priorities    | `defaultJobOptions.priority` |
| `TaskCheckInterval` | `drainDelay`                 |
| `MaxRetry`          | `attempts`                   |

---

### ADR-005: Valibot for Schema Validation

**Status:** Accepted

**Context:**
Need runtime schema validation for API requests.

**Decision:**
Use **Valibot** instead of Zod.

**Rationale:**
1. **Bundle Size:** <700 bytes vs Zod's 13KB
2. **Modular Design:** Tree-shakeable, only import what you use
3. **Type Safety:** Full TypeScript inference like Zod
4. **API Similarity:** Easy migration path from Zod

**Consequences:**
- Positive: Smaller bundle, same functionality
- Negative: Smaller community, fewer examples

---

### ADR-006: Cheerio for Static HTML Parsing

**Status:** Accepted

**Context:**
Need fast HTML parsing for static content.

**Decision:**
Use **Cheerio** with Bun's native `fetch`.

**Rationale:**
1. **Performance:** 8-12x faster than browser-based parsing
2. **Memory Efficient:** No browser overhead
3. **Familiar API:** jQuery-like syntax known to most developers
4. **Mature:** Well-tested, stable library

**Consequences:**
- Positive: Fast, lightweight, familiar
- Negative: No JavaScript execution (by design)

---

### ADR-007: Turndown for Markdown Conversion

**Status:** Accepted

**Context:**
Need HTML to Markdown conversion.

**Decision:**
Use **Turndown** to replace html-to-markdown/v2.

**Rationale:**
1. **Standard:** Most widely used HTML-to-Markdown library
2. **Extensible:** Plugin system for custom rules
3. **Maintained:** Active development, good documentation
4. **Customizable:** Fine control over output format

**Consequences:**
- Positive: Standard, extensible, well-documented
- Negative: May require custom rules for edge cases

---

## Feature Parity Matrix

### API Endpoints

| Endpoint            | Go Status | JS Status | Notes                    |
| ------------------- | --------- | --------- | ------------------------ |
| `POST /v1/scrape`   | ✅         | 📋 Planned | Full parity expected     |
| `GET /v1/scrape`    | ✅         | 📋 Planned | Convenience endpoint     |
| `POST /v1/crawl`    | ✅         | 📋 Planned | Async crawling           |
| `GET /v1/crawl/:id` | ✅         | 📋 Planned | Job status polling       |
| `POST /v1/search`   | ✅         | 📋 Planned | Brave Search integration |

### Scraping Modes

| Mode    | Go Implementation     | JS Implementation | Parity |
| ------- | --------------------- | ----------------- | ------ |
| Static  | Colly                 | Cheerio + fetch   | ✅ Full |
| Dynamic | Chromedp              | Playwright        | ✅ Full |
| Smart   | Heuristics → fallback | Same logic        | ✅ Full |

### Features

| Feature            | Go         | JS               | Notes        |
| ------------------ | ---------- | ---------------- | ------------ |
| UA rotation        | gofakeit   | random-useragent | ✅ Equivalent |
| Cache (Redis)      | go-redis   | ioredis          | ✅ Equivalent |
| Compression (gzip) | stdlib     | zlib/native      | ✅ Equivalent |
| TLS Redis          | crypto/tls | native TLS       | ✅ Equivalent |
| Structured logging | slog       | Pino             | ✅ Equivalent |
| Health checks      | Custom     | Custom           | 📋 Planned    |
| Browser restart    | Planned    | Planned          | 📋 Planned    |

### Gaps

| Feature         | Status | Mitigation |
| --------------- | ------ | ---------- |
| None identified | N/A    | N/A        |

---

## Risk Assessment

### Risk 1: Memory Overhead (🔴 High)

**Description:**
V8 (Bun) + Playwright may exceed Leapcell's 4GB limit under load.

**Analysis:**
**Analysis:**
- Go baseline: ~200-300MB for browser + API
- JS projected: ~300MB baseline (JSC), ~700MB-1.0GB at 10 contexts
- Leapcell limit: 4GB

**Probability:** Medium (40%)

**Impact:** High (project viability)

**Mitigation:**
1. Benchmark early in Phase 1
2. Implement context pooling with hard limits
3. Add browser restart after N requests
4. Reduce concurrency if needed (10 → 5)

**Monitoring:**
- Track `process.memoryUsage()` in production
- Set up alerts at 3GB threshold

---

### Risk 2: Cold Start Regression (🟡 Medium)

**Description:**
Bun + Playwright initialization may exceed 5-second target.

**Analysis:**
- Go cold start: ~1-2s (binary + browser)
- JS cold start: ~3-5s (runtime + deps + browser)
- Target: <5s

**Probability:** Medium (50%)

**Impact:** Medium (user experience)

**Mitigation:**
1. **Lazy browser init:** Don't start Playwright until first dynamic request
2. **Keep-warm pings:** Configure Leapcell health checks to prevent cold starts
3. **Pre-bundling:** Use Bun's bundler to reduce dependency loading

**Monitoring:**
- Track time from container start to first successful request

---

### Risk 3: Worker Thread Complexity (🟡 Medium)

**Description:**
Node.js worker_threads pattern is more complex than Go goroutines.

**Analysis:**
- Go: Goroutines are lightweight (2KB stack)
- JS: Worker threads are OS-level threads (Bun workers share JSC runtime, lighter than Node V8 isolates)
- Challenge: Sharing browser instance across threads

**Probability:** Low (30%)

**Impact:** Medium (development time)

**Mitigation:**
1. Document pattern thoroughly before implementation
2. Consider single-threaded fallback (concurrent within event loop)
3. Use existing BullMQ worker patterns

---

### Risk 4: Playwright Detection (🟢 Low)

**Description:**
Websites may detect Playwright automation.

**Analysis:**
- Go (Chromedp): Uses undetected-chromedp flags
- JS: playwright-extra + stealth plugin provides equivalent evasion

**Probability:** Low (20%)

**Impact:** Low (workarounds exist)

**Mitigation:**
1. Use playwright-extra with stealth plugin
2. Implement same evasion flags as Go version
3. Regular testing against bot detection sites

---

## Cost Projection

### Development Costs

| Phase                 | Effort         | Duration    | Notes                   |
| --------------------- | -------------- | ----------- | ----------------------- |
| Phase 1: Static API   | 2-3 days       | Week 1      | Cheerio + Hono setup    |
| Phase 2: Smart Mode   | 3-4 days       | Week 2      | Playwright + heuristics |
| Phase 3: Async Queue  | 2-3 days       | Week 3      | BullMQ + worker threads |
| Phase 4: Optimization | 2-3 days       | Week 4      | Performance tuning      |
| **Total**             | **10-15 days** | **4 weeks** | Conservative estimate   |

### Leapcell Infrastructure

| Configuration          | Go              | JS              | Notes                  |
| ---------------------- | --------------- | --------------- | ---------------------- |
| Memory                 | 2GB comfortable | 4GB recommended | JS needs more headroom |
| Estimated monthly cost | ~$5-10          | ~$10-20         | Higher memory tier     |

---

## Success Criteria

### Must Pass (Phase 1 → 2 Gate)

- [ ] Memory usage under **2GB** at 10 concurrent dynamic scrapes
- [ ] P95 latency within **20%** of Go baseline
- [ ] Cold start under **5 seconds** on Leapcell

### Should Pass (Production Readiness)

- [ ] Queue throughput **1000+ jobs/hour** at 4GB RAM
- [ ] **100%** API compatibility (drop-in replacement)
- [ ] Anti-detection passes bot.sannysoft.com tests

### Nice to Have

- [ ] Developer velocity improvement measurable (time to implement new feature)
- [ ] Bundle size under 10MB (excluding Chromium)

---

## Open Questions

1. **Bun + Playwright compatibility:** Has this combination been tested at scale?
   - *Action:* Research and document in Phase 1

2. **Worker thread browser sharing:** Can Playwright contexts be used across threads?
   - *Action:* Prototype and document pattern

3. **Leapcell cold start behavior:** How does Leapcell handle container recycling?
   - *Action:* Test with deployed prototype

4. **Valibot vs Zod ecosystem:** Are there critical Zod features missing in Valibot?
   - *Action:* Document any gaps found during implementation

---

## Appendices

- [Go vs JS Comparison](./go-vs-js-comparison.md)
- [Implementation Roadmap](./implementation-roadmap.md)
- [Smart Mode Heuristics](./smart-mode-heuristics.md)
- [Anti-Detection Strategy](./anti-detection-strategy.md)
- [Queue Architecture](./queue-architecture.md)
- [Performance Benchmark Plan](./performance-benchmark-plan.md)

---

*Document Version: 1.0.0-draft*  
*Last Updated: 2026-02-02*
