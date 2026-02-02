# Cinder JS - Documentation Index

> **Status:** RFC / Planning Phase  
> **Created:** 2026-02-02  
> **Version:** 0.1.0-draft

---

## Executive Summary

This documentation package evaluates the feasibility of porting **Cinder** (a Go-based web scraping API) to a modern JavaScript/TypeScript stack using **Bun + Hono**. The goal is to determine if JS offers better developer velocity and easier maintenance while retaining comparable performance for the Leapcell deployment target.

---

## Quick Navigation

### Core Architecture Documents

| Document                                                  | Description                                                                         | Priority   |
| --------------------------------------------------------- | ----------------------------------------------------------------------------------- | ---------- |
| [**Architecture RFC**](./cinder-js-architecture.md)       | Primary decision document with executive summary, ADRs, and go/no-go recommendation | 🔴 Critical |
| [**Go vs JS Comparison**](./go-vs-js-comparison.md)       | Detailed performance analysis and feature parity matrix                             | 🔴 Critical |
| [**Implementation Roadmap**](./implementation-roadmap.md) | Phased approach with milestones and success criteria                                | 🟡 High     |

### Technical Specifications

| Document                                                                | Description                                             | Priority |
| ----------------------------------------------------------------------- | ------------------------------------------------------- | -------- |
| [**Smart Mode Heuristics**](./smart-mode-heuristics.md)                 | Dynamic content detection algorithms and fallback chain | 🟡 High   |
| [**Anti-Detection Strategy**](./anti-detection-strategy.md)             | Stealth scraping and bot evasion patterns               | 🟡 High   |
| [**Queue Architecture**](./queue-architecture.md)                       | BullMQ configuration and worker thread patterns         | 🟡 High   |
| [**Response Format Specification**](./response-format-specification.md) | Output formats, metadata schema, TypeScript definitions | 🟡 High   |
| [**API Specification**](./api-specification.md)                         | Endpoint documentation and request/response schemas     | 🟢 Medium |

### Performance & Operations

| Document                                                          | Description                                              | Priority |
| ----------------------------------------------------------------- | -------------------------------------------------------- | -------- |
| [**Performance Benchmark Plan**](./performance-benchmark-plan.md) | Methodology for measuring cold start, throughput, memory | 🟢 Medium |
| [**Operations Runbook**](./operations-runbook.md)                 | Monitoring, troubleshooting, and incident response       | 🟢 Medium |

### Infrastructure Configuration

| Document                              | Description                                        | Priority   |
| ------------------------------------- | -------------------------------------------------- | ---------- |
| [**Dockerfile**](../Dockerfile)       | Multi-stage build for Bun + Playwright on Leapcell | 🔴 Critical |
| [**leapcell.yaml**](../leapcell.yaml) | Leapcell deployment configuration                  | 🔴 Critical |

---

## Decision Status

| Question                          | Status                         | Document                                        |
| --------------------------------- | ------------------------------ | ----------------------------------------------- |
| Should we proceed with cinder-js? | ✅ Documented (Pending Phase 1) | [Architecture RFC](./cinder-js-architecture.md) |
| Can we achieve memory targets?    | ✅ Documented (Pending Phase 2) | [Go vs JS Comparison](./go-vs-js-comparison.md) |
| Is API parity achievable?         | ✅ Documented                   | [API Specification](./api-specification.md)     |

---

## Technology Stack Summary

| Component           | Go (Current)        | JS (Proposed)              | Rationale                                  |
| ------------------- | ------------------- | -------------------------- | ------------------------------------------ |
| Runtime             | Go 1.25+            | **Bun 1.1+**               | Fast startup, native TS, unified toolchain |
| Web Framework       | Gin                 | **Hono**                   | Lightweight, Web Standard compliant        |
| Static Scraping     | Colly               | **Cheerio + native fetch** | Equivalent functionality, familiar API     |
| Dynamic Scraping    | Chromedp            | **Playwright**             | Better DX, more features, stealth plugins  |
| Queue System        | Asynq               | **BullMQ**                 | Redis-backed, mature ecosystem             |
| Markdown Conversion | html-to-markdown/v2 | **Turndown**               | Standard, extensible                       |
| Config Validation   | Viper               | **Valibot**                | Modular, tiny bundle (<700 bytes)          |
| Logging             | slog                | **Pino**                   | Structured JSON, high performance          |

---

## Key Risks Summary

| Risk                               | Severity | Mitigation                          | Status     |
| ---------------------------------- | -------- | ----------------------------------- | ---------- |
| Memory overhead of V8 + Playwright | 🔴 High   | Benchmark at 10 concurrent contexts | Pending    |
| Cold start regression              | 🟡 Medium | Lazy browser init, keep-warm pings  | Documented |
| BullMQ worker thread complexity    | 🟡 Medium | Reference pattern documented        | Documented |
| Feature parity gaps                | 🟢 Low    | Full API mapping completed          | Documented |

---

## How to Read This Documentation

1. **Start with** [Architecture RFC](./cinder-js-architecture.md) for the executive summary
2. **Review** [Go vs JS Comparison](./go-vs-js-comparison.md) for detailed analysis
3. **Check** [Implementation Roadmap](./implementation-roadmap.md) for phased approach
4. **Reference** technical specs as needed for implementation details

---

## Contributing to This RFC

This is a living document. To propose changes:

1. Create a new branch from `main`
2. Edit the relevant markdown files
3. Submit a PR with clear rationale
4. Tag `@Michael-Obele` for review

---

*Last Updated: 2026-02-02*
