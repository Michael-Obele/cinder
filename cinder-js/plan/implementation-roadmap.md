# Implementation Roadmap

> **Status:** Planning Phase  
> **Estimated Duration:** 4 weeks (10-15 working days)  
> **Last Updated:** 2026-02-03

---

## Table of Contents

1. [Phase Overview](#phase-overview)
2. [Phase 1: Static Scraping API](#phase-1-static-scraping-api)
3. [Phase 2: Smart Mode + Playwright](#phase-2-smart-mode--playwright)
4. [Phase 3: Async Queue System](#phase-3-async-queue-system)
5. [Phase 4: Optimization & Hardening](#phase-4-optimization--hardening)
6. [Resource Requirements](#resource-requirements)
7. [Risk Mitigation Timeline](#risk-mitigation-timeline)

---

## Phase Overview

```
Week 1          Week 2          Week 3          Week 4
┌───────────┐   ┌───────────┐   ┌───────────┐   ┌───────────┐
│ PHASE 1   │   │ PHASE 2   │   │ PHASE 3   │   │ PHASE 4   │
│           │   │           │   │           │   │           │
│ Static    │──►│ Dynamic   │──►│ Async     │──►│ Optimize  │
│ Scraping  │   │ Scraping  │   │ Queue     │   │ Harden    │
│           │   │           │   │           │   │           │
│ - Hono    │   │-Playwright│   │ - BullMQ  │   │ - Memory  │
│ - Cheerio │   │- Heuristic│   │ - Workers │   │ - Stealth │
│ - Valibot │   │- Fallback │   │ - Redis   │   │ - Monitor │
│           │   │           │   │           │   │           │
│ GO/NO-GO  │   │           │   │           │   │ LAUNCH    │
│ GATE ──►  │   │           │   │           │   │ READY     │
└───────────┘   └───────────┘   └───────────┘   └───────────┘
    2-3 days        3-4 days        2-3 days        2-3 days
```

### Phase Summary

| Phase | Scope                            | Duration | Success Gate                   |
| ----- | -------------------------------- | -------- | ------------------------------ |
| **1** | Static scraping, project setup   | 2-3 days | Cheerio + Hono functional      |
| **2** | Dynamic scraping, smart mode     | 3-4 days | Memory <2GB at 10 contexts     |
| **3** | BullMQ integration               | 2-3 days | Queue throughput 1000+ jobs/hr |
| **4** | Performance, stealth, monitoring | 2-3 days | All success criteria met       |

---

## Phase 1: Static Scraping API

**Duration:** 2-3 days  
**Risk Level:** 🟢 Low  
**Goal:** Establish core project structure and validate Bun + Hono stack

### Deliverables

- [ ] Project scaffolding (Bun + TypeScript)
- [ ] Hono HTTP server with `/v1/scrape` endpoint
- [ ] Valibot request validation
- [ ] Cheerio + fetch static scraper
- [ ] Turndown markdown conversion
- [ ] Pino structured logging
- [ ] Basic Dockerfile (without Playwright)
- [ ] Unit tests for core components

### Milestone Checklist

#### Day 1: Project Setup
- [ ] Initialize Bun project with TypeScript
- [ ] Configure tsconfig.json for Bun
- [ ] Set up project structure:
  ```
  cinder-js/
  ├── src/
  │   ├── index.ts          # Entry point
  │   ├── config/           # Valibot config schema
  │   ├── routes/           # Hono route handlers
  │   ├── services/         # Business logic
  │   └── lib/              # Utilities
  ├── tests/
  ├── Dockerfile
  ├── package.json
  ├── tsconfig.json
  └── bunfig.toml
  ```
- [ ] Install core dependencies:
  - `hono` - Web framework
  - `cheerio` - HTML parsing
  - `turndown` - Markdown conversion
  - `valibot` - Schema validation
  - `pino` - Logging

#### Day 2: Core Implementation
- [ ] Implement scraper service interface
- [ ] Implement Cheerio-based static scraper
- [ ] Implement Turndown markdown converter
- [ ] Create `/v1/scrape` endpoint
- [ ] Add request validation with Valibot
- [ ] Add structured logging with Pino
- [ ] Write unit tests for scraper service

#### Day 3: Integration & Testing
- [ ] End-to-end testing with real URLs
- [ ] Error handling and edge cases
- [ ] Basic Dockerfile (Bun only)
- [ ] Local Docker build verification
- [ ] Documentation: API contract

### Success Criteria

| Criterion             | Target           | Measurement            |
| --------------------- | ---------------- | ---------------------- |
| Static scrape latency | <500ms P95       | `time curl /v1/scrape` |
| Response format       | Match Go API     | Schema comparison      |
| Error handling        | Graceful 4xx/5xx | Invalid URL tests      |
| Test coverage         | >80% services    | Bun test coverage      |

### API Contract (Phase 1)

**Request:**
```http
POST /v1/scrape HTTP/1.1
Content-Type: application/json

{
  "url": "https://example.com",
  "mode": "static"
}
```

**Response:**
```json
{
  "url": "https://example.com",
  "markdown": "# Example Domain\n\nThis domain is...",
  "html": "<!DOCTYPE html>...",
  "metadata": {
    "scraped_at": "2026-02-02T10:30:00Z",
    "engine": "cheerio"
  }
}
```

### Go/No-Go Gate

At the end of Phase 1, evaluate:

| Question                          | Threshold             | Result          |
| --------------------------------- | --------------------- | --------------- |
| Is Bun + Hono stable?             | No critical issues    | ☐ Pass / ☐ Fail |
| Is development velocity improved? | Subjective assessment | ☐ Pass / ☐ Fail |
| Are there blocking issues?        | No blockers           | ☐ Pass / ☐ Fail |

**If Phase 1 fails:** Stop and document learnings. Continue with Go.

---

## Phase 2: Smart Mode + Playwright

**Duration:** 3-4 days  
**Risk Level:** 🟡 Medium  
**Goal:** Add dynamic scraping and validate memory constraints

### Deliverables

- [ ] Playwright integration
- [ ] Smart mode with heuristics
- [ ] Fallback chain (fetch → Cheerio → Playwright)
- [ ] Browser singleton pattern
- [ ] Context pooling
- [ ] Memory benchmarks
- [ ] Updated Dockerfile with Playwright
- [ ] Integration tests

### Milestone Checklist

#### Day 1: Playwright Integration
- [ ] Install Playwright and dependencies
- [ ] Implement browser singleton pattern:
  ```
  Browser Manager
  ├── Lazy initialization (first dynamic request)
  ├── Single browser instance
  ├── Context pool (max 15)
  └── Context lifecycle management
  ```
- [ ] Implement Playwright-based dynamic scraper
- [ ] Integrate Turndown for dynamic content

#### Day 2: Smart Mode Logic
- [ ] Port heuristics from Go:
  ```
  Smart Mode Decision Tree
  │
  ├── Check: noscript tags with JS warnings?
  │   └── Yes → Dynamic
  │
  ├── Check: SPA root markers (id="root", __NEXT_DATA__)?
  │   └── Yes + Small body (<5KB) → Dynamic
  │
  ├── Check: Tiny body (<2KB) + script tags?
  │   └── Yes → Dynamic
  │
  └── Default → Static
  ```
- [ ] Implement fallback chain
- [ ] Add mode selection to service layer

#### Day 3: Memory Optimization
- [ ] Implement context pooling with limits
- [ ] Add browser health monitoring
- [ ] Memory benchmark at 5, 10, 15 contexts
- [ ] Document findings

#### Day 4: Integration & Testing
- [ ] Update Dockerfile with Playwright:
  ```dockerfile
  FROM mcr.microsoft.com/playwright:v1.50.0-jammy
  # Update to Bun 1.2+ for memory optimizations
  RUN curl -fsSL https://bun.sh/install | bash
  ```
- [ ] End-to-end testing with dynamic sites (React, Next.js)
- [ ] Deploy to Leapcell staging
- [ ] Measure cold start time
- [ ] Document memory usage

### Success Criteria

| Criterion              | Target    | Measurement                      |
| ---------------------- | --------- | -------------------------------- |
| Memory at 10 contexts  | <2GB      | `process.memoryUsage()`          |
| Dynamic scrape latency | <3.5s P95 | Automated test suite             |
| Cold start             | <5s       | Container start to first request |
| Smart mode accuracy    | >90%      | Manual testing 50 sites          |

### Memory Benchmark Procedure

```javascript
// Benchmark script outline
async function memoryBenchmark() {
  const results = [];
  
  for (const concurrency of [1, 5, 10, 15, 20]) {
    const baseline = process.memoryUsage().heapUsed;
    
    // Create N concurrent contexts
    const contexts = await Promise.all(
      Array(concurrency).fill(0).map(() => browser.newContext())
    );
    
    const afterContexts = process.memoryUsage().heapUsed;
    
    // Scrape with each context
    await Promise.all(
      contexts.map(ctx => scrapePage(ctx, 'https://example.com'))
    );
    
    const afterScrape = process.memoryUsage().heapUsed;
    
    // Cleanup
    await Promise.all(contexts.map(ctx => ctx.close()));
    
    results.push({
      concurrency,
      contextMemory: afterContexts - baseline,
      scrapeMemory: afterScrape - baseline,
    });
  }
  
  return results;
}
```

### Go/No-Go Gate (Critical)

**This is the critical decision point:**

| Question               | Threshold     | Result          |
| ---------------------- | ------------- | --------------- |
| Memory at 10 contexts  | <2GB          | ☐ Pass / ☐ Fail |
| Cold start on Leapcell | <5s           | ☐ Pass / ☐ Fail |
| Smart mode works       | >90% accuracy | ☐ Pass / ☐ Fail |

**If Phase 2 fails:** Abort cinder-js project. Memory constraints not viable.

---

## Phase 3: Async Queue System

**Duration:** 2-3 days  
**Risk Level:** 🟢 Low  
**Goal:** Implement Redis-backed job queue for async crawling

### Deliverables

- [ ] BullMQ integration
- [ ] `/v1/crawl` endpoint (enqueue)
- [ ] `/v1/crawl/:id` endpoint (status)
- [ ] Worker process (same container)
- [ ] Graceful shutdown handling
- [ ] Queue monitoring
- [ ] Integration tests

### Milestone Checklist

#### Day 1: BullMQ Setup
- [ ] Install BullMQ and ioredis
- [ ] Configure Redis connection with TLS support
- [ ] Create queue definitions:
  ```
  Queues:
  ├── scrape:critical (priority 6)
  ├── scrape:default  (priority 3)
  └── scrape:low      (priority 1)
  ```
- [ ] Implement job producer

#### Day 2: Worker Implementation
- [ ] Implement job processor
- [ ] Integrate with scraper service
- [ ] Handle job results storage
- [ ] Implement retry logic:
  ```
  Retry Strategy:
  ├── Max attempts: 3
  ├── Backoff: exponential
  └── Failed → Dead letter queue
  ```
- [ ] Add `/v1/crawl` and `/v1/crawl/:id` endpoints

#### Day 3: Monolith Pattern
- [ ] Implement worker startup in same process:
  ```javascript
  // Conceptual: Running API + Worker together
  async function main() {
    // Start API server
    const app = createHonoApp();
    
    // Start worker in same process
    const worker = createBullMQWorker();
    
    // Graceful shutdown
    process.on('SIGTERM', async () => {
      await worker.close();
      await app.close();
      process.exit(0);
    });
    
    Bun.serve({ fetch: app.fetch, port: 8080 });
  }
  ```
- [ ] Add graceful shutdown handling
- [ ] Test with Redis on Leapcell/Upstash
- [ ] Load testing

### Success Criteria

| Criterion             | Target             | Measurement      |
| --------------------- | ------------------ | ---------------- |
| Queue enqueue latency | <15ms P95          | Automated test   |
| Jobs/hour at 4GB      | >1000              | Load test        |
| Graceful shutdown     | All jobs drained   | Manual test      |
| Redis TLS             | Works with Upstash | Integration test |

### Queue Configuration Mapping

**From Go (Asynq):**
```go
asynq.Config{
    Concurrency: 10,
    Queues: map[string]int{
        "critical": 6,
        "default":  3,
        "low":      1,
    },
}
```

**To JS (BullMQ):**
```javascript
const worker = new Worker('scrape', processor, {
    concurrency: 10,
    connection: redisConnection,
});

// Priority via job options
await queue.add('scrape', data, { priority: 1 }); // critical
await queue.add('scrape', data, { priority: 5 }); // default
await queue.add('scrape', data, { priority: 10 }); // low
```

---

## Phase 4: Optimization & Hardening

**Duration:** 2-3 days  
**Risk Level:** 🟢 Low  
**Goal:** Performance tuning, anti-detection, and production readiness

### Deliverables

- [ ] Stealth configuration (playwright-extra)
- [ ] User-Agent rotation
- [ ] Caching layer (Redis)
- [ ] Memory monitoring
- [ ] Health check endpoints
- [ ] Browser restart mechanism
- [ ] Production Dockerfile
- [ ] leapcell.yaml configuration
- [ ] Operations runbook

### Milestone Checklist

#### Day 1: Anti-Detection
- [ ] Integration playwright-extra with stealth plugin
- [ ] Configure stealth options:
  ```
  Stealth Configuration:
  ├── navigator.webdriver = undefined
  ├── WebGL vendor/renderer spoofing
  ├── User-Agent consistency
  ├── Chrome runtime emulation
  └── Permission API patches
  ```
- [ ] Implement User-Agent rotation
- [ ] Test against bot.sannysoft.com
- [ ] Test against fingerprint.js

#### Day 2: Caching & Performance
- [ ] Implement Redis caching layer:
  ```
  Cache Strategy:
  ├── Key: scrape:{url}:{mode}
  ├── TTL: 7 days
  ├── Compression: gzip
  └── Invalidation: Manual/API
  ```
- [ ] Add memory monitoring with alerts
- [ ] Implement browser restart after N requests
- [ ] Performance profiling and optimization

#### Day 3: Production Readiness
- [ ] Finalize Dockerfile (multi-stage, optimized)
- [ ] Create leapcell.yaml
- [ ] Add health check endpoints:
  - `GET /health` - Basic liveness
  - `GET /health/ready` - Full readiness (Redis, browser)
- [ ] Write operations runbook
- [ ] Security review (env vars, secrets)
- [ ] Final integration testing on Leapcell

### Success Criteria

| Criterion                     | Target                      | Measurement   |
| ----------------------------- | --------------------------- | ------------- |
| Stealth test                  | Pass bot.sannysoft.com      | Manual test   |
| Cache hit rate                | >50% (repeat requests)      | Redis metrics |
| Browser stability             | No crashes in 1hr load test | Monitoring    |
| All success criteria from RFC | Met                         | Checklist     |

### Final Success Criteria Checklist (From RFC)

**Must Pass:**
- [ ] Memory usage under 2GB at 10 concurrent dynamic scrapes
- [ ] P95 latency within 20% of Go baseline
- [ ] Cold start under 5 seconds on Leapcell

**Should Pass:**
- [ ] Queue throughput 1000+ jobs/hour at 4GB RAM
- [ ] 100% API compatibility (drop-in replacement)
- [ ] Anti-detection passes bot.sannysoft.com tests

**Nice to Have:**
- [ ] Developer velocity improvement measurable
- [ ] Bundle size under 10MB (excluding Chromium)

---

## Resource Requirements

### Personnel

| Role                 | Phase 1 | Phase 2 | Phase 3 | Phase 4 |
| -------------------- | ------- | ------- | ------- | ------- |
| Full-stack developer | 100%    | 100%    | 100%    | 100%    |
| DevOps (optional)    | 0%      | 25%     | 25%     | 50%     |

### Infrastructure

| Resource        | Phase 1   | Phase 2+  |
| --------------- | --------- | --------- |
| Leapcell (dev)  | 2GB       | 4GB       |
| Redis (Upstash) | Free tier | Free tier |
| GitHub Actions  | Standard  | Standard  |

### External Dependencies

| Dependency    | License     | Cost                         |
| ------------- | ----------- | ---------------------------- |
| Bun           | MIT         | Free                         |
| Hono          | MIT         | Free                         |
| Playwright    | Apache 2.0  | Free                         |
| BullMQ        | MIT         | Free                         |
| Leapcell      | Proprietary | Pay-as-you-go (~$10-20/mo)   |
| Upstash Redis | Proprietary | Free tier (10k requests/day) |

---

## Risk Mitigation Timeline

```
Week 1      Week 2      Week 3      Week 4
│           │           │           │
├─ Phase 1 ─┼─ Phase 2 ─┼─ Phase 3 ─┼─ Phase 4 ─┤
│           │           │           │           │
│  LOW      │  MEDIUM   │  LOW      │  LOW      │
│  RISK     │  RISK     │  RISK     │  RISK     │
│           │           │           │           │
│           │    ▲      │           │           │
│           │    │      │           │           │
│           │ CRITICAL  │           │           │
│           │ GATE HERE │           │           │
│           │           │           │           │
│  If fails:│ If fails: │           │           │
│  Easy     │ Abort     │           │           │
│  rollback │ project   │           │           │
└───────────┴───────────┴───────────┴───────────┘
```

### Rollback Points

| Phase   | Rollback Action           | Cost Lost  |
| ------- | ------------------------- | ---------- |
| Phase 1 | Abandon, continue with Go | 2-3 days   |
| Phase 2 | Abandon, continue with Go | 5-7 days   |
| Phase 3 | Abandon, continue with Go | 8-10 days  |
| Phase 4 | Fix issues, don't launch  | 10-12 days |

**Recommendation:** Make go/no-go decision at end of Phase 2 to minimize potential waste.

---

*Document Version: 1.0.1*  
*Last Updated: 2026-02-03*
