# Implementation Roadmap (Documentation Only)

## Guiding Principles
- **No application code** in this repository scope.
- Each phase ends with documentation artifacts and benchmark results.
- A phase does **not** proceed without passing its success gate.

## Phase 0 — Feasibility & Baselines
**Scope**: Collect data and verify assumptions before any build.

**Deliverables**
- Benchmark plan
- Architecture decisions & risk assessment
- Leapcell deployment notes

**Success Gate**
- Confirm Bun + Playwright feasibility on Leapcell
- Clear acceptance criteria for cold start and memory

---

## Phase 1 — Static Scraping API (Design Only)
**Scope**: Design `fetch → Cheerio → Turndown` pipeline and request/response contracts.

**Documentation Required**
- API endpoint spec for `/v1/scrape` with static mode
- Data transformation pipeline spec
- Error handling and retry strategy

**Success Gate**
- API spec approved and mapped to Go responses

---

## Phase 2 — Smart Mode + Playwright (Design Only)
**Scope**: Define heuristic detection + fallback chain with timeouts.

**Documentation Required**
- Smart mode heuristics (signals, thresholds, fallback order)
- Playwright lifecycle plan (context pooling/recycling)
- Anti-detection strategy (stealth plugin, UA rotation)

**Success Gate**
- Heuristics document signed off with measurable thresholds

---

## Phase 3 — Async Queue + Worker Threads (Design Only)
**Scope**: Define BullMQ worker patterns, concurrency, and shutdown behavior.

**Documentation Required**
- Queue design (job payloads, retries, backoff)
- Worker lifecycle plan (start, drain, shutdown)
- Redis config and TLS notes

**Success Gate**
- BullMQ design accepted and operationally safe for monolith use

---

## Phase 4 — Performance Optimization (Design Only)
**Scope**: Define performance tuning strategies and monitoring plan.

**Documentation Required**
- Benchmark scenarios and metrics
- Memory and latency budget per request
- Operational runbook (alerts, SLOs)

**Success Gate**
- Performance plan with clear acceptance thresholds

---

## Resource Requirements
- **Architect**: 1 (solution + API contracts)
- **DevOps**: 1 (Leapcell deployment strategy + runtime configs)
- **QA/Perf**: 1 (benchmarking + measurements)

## Artifacts Checklist
- [x] Architecture & ADRs
- [x] Go vs JS comparison
- [x] Smart mode heuristics
- [x] Performance benchmark plan
- [x] Queue/worker architecture
- [x] Leapcell deployment notes
- [x] Valibot config schema plan
- [x] Anti-detection strategy

