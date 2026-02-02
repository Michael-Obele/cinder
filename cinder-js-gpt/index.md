# Cinder-JS GPT Planning Hub

This folder contains **documentation-only** planning artifacts for the `cinder-js` port. No application code is included here.

## Primary Deliverables

- [Architecture & ADRs](plan/cinder-js-architecture.md)
- [Go vs JS Comparison](plan/go-vs-js-comparison.md)
- [Implementation Roadmap](plan/implementation-roadmap.md)
- [Smart Mode Heuristics](plan/smart-mode-heuristics.md)
- [Performance Benchmark Plan](plan/performance-benchmark-plan.md)
- [Anti-Detection & Evasion Strategy](plan/anti-detection-strategy.md)
- [Queue + Worker Architecture](plan/queue-worker-architecture.md)
- [Leapcell Deployment Notes](plan/leapcell-deployment-notes.md)
- [Valibot Config Schema Plan](plan/valibot-config-schema.md)
- [Response Format Specification](plan/response-format-spec.md)
- [Research Sources & Links](plan/research-sources.md)

## Infrastructure Config (Planning-Only)

- [Dockerfile](Dockerfile)
- [leapcell.yaml](leapcell.yaml)

## Scope & Constraints

- Runtime: **Bun v1.1+**
- Framework: **Hono**
- Static scraping: **Fetch + Cheerio**
- Dynamic scraping: **Playwright**
- Queue: **BullMQ + Redis**
- Markdown: **@turndown/turndown**
- Validation: **Valibot**
- Logging: **Pino**

> All documents here are planning artifacts. No `.ts`/`.js` code has been written, per instructions.
