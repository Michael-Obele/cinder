# 📚 Cinder Documentation Index

Welcome to the Cinder documentation! This guide is designed to help **full-stack Svelte/TypeScript developers** understand this Go codebase.

---

## Quick Start

If you're new to Go, read these in order:

| #   | Document                                      | Description                         | Time   |
| --- | --------------------------------------------- | ----------------------------------- | ------ |
| 1   | [Go for Svelte Devs](GO_FOR_SVELTE_DEVS.md)   | Mental models, comparisons to JS/TS | 20 min |
| 2   | [Go Syntax Reference](GO_SYNTAX_REFERENCE.md) | Quick reference for Go syntax       | 10 min |
| 3   | [Code Walkthrough](CODE_WALKTHROUGH.md)       | File-by-file annotated code         | 30 min |

---

## Documentation Overview

### [GO_FOR_SVELTE_DEVS.md](GO_FOR_SVELTE_DEVS.md)

**Audience**: Svelte/TypeScript developers new to Go

**Covers**:

- Go vs TypeScript mental model comparison
- Variable declarations (`:=` vs `var`)
- Functions and methods
- Structs (like TS interfaces/classes)
- Error handling (no try/catch!)
- Pointers explained simply
- Packages and imports
- Context (like SvelteKit's event)
- Goroutines (async/await equivalent)

### [GO_SYNTAX_REFERENCE.md](GO_SYNTAX_REFERENCE.md)

**Audience**: Anyone needing a quick syntax lookup

**Covers**:

- Variable declarations and zero values
- Function signatures and returns
- Struct tags and methods
- Interface definitions
- Control flow (if, switch, for)
- Slices and maps
- Pointers
- Concurrency (goroutines, channels)
- JSON encoding/decoding
- Testing patterns
- Common idioms

### [CODE_WALKTHROUGH.md](CODE_WALKTHROUGH.md)

**Audience**: Developers wanting to understand the actual codebase

**Covers**:

- Entry points (`cmd/api/main.go`, `cmd/worker/main.go`)
- Configuration (`internal/config/`)
- HTTP routing and handlers (`internal/api/`)
- Business logic (`internal/scraper/`, `internal/search/`)
- Background jobs (`internal/worker/`)
- Shared utilities (`pkg/logger/`)

---

## Project Architecture

```
cinder/
├── cmd/                    # Entry points (executables)
│   ├── api/main.go         # HTTP server + embedded worker
│   └── worker/main.go      # Standalone worker
├── internal/               # Private application code
│   ├── api/                # HTTP layer
│   │   ├── router.go       # Route definitions
│   │   ├── handlers/       # Request handlers
│   │   └── middleware/     # HTTP middleware
│   ├── config/             # Configuration loading
│   ├── domain/             # Core types & interfaces
│   ├── scraper/            # Scraping implementations
│   ├── search/             # Search service
│   └── worker/             # Job processing
├── pkg/                    # Public shared libraries
│   └── logger/             # Logging utilities
├── docs/                   # Documentation
│   └── guides/             # Developer guides (you are here!)
└── go.mod                  # Dependencies (like package.json)
```

---

## Key Concepts Map

| SvelteKit Concept   | Go Equivalent     | Cinder Location              |
| ------------------- | ----------------- | ---------------------------- |
| `+page.server.ts`   | Handler function  | `internal/api/handlers/*.go` |
| `hooks.server.ts`   | Middleware        | `internal/api/middleware/`   |
| Route group `(api)` | Router group      | `v1 := r.Group("/v1")`       |
| `$lib`              | `internal/`       | `internal/*`                 |
| `types.ts`          | Domain package    | `internal/domain/`           |
| Environment vars    | Viper config      | `internal/config/`           |
| `package.json`      | `go.mod`          | Project root                 |
| `npm install`       | `go mod download` | —                            |

---

## Request Flow

```
HTTP Request
    │
    ▼
┌─────────────────────┐
│  router.go          │  ← Route matching
└─────────┬───────────┘
          │
          ▼
┌─────────────────────┐
│  middleware/        │  ← Logging, recovery
│  logger.go          │
└─────────┬───────────┘
          │
          ▼
┌─────────────────────┐
│  handlers/          │  ← Parse request, call service
│  scrape.go          │
└─────────┬───────────┘
          │
          ▼
┌─────────────────────┐
│  scraper/           │  ← Business logic
│  service.go         │
└─────────┬───────────┘
          │
          ├─────────────────────────┐
          │                         │
          ▼                         ▼
┌─────────────────────┐   ┌─────────────────────┐
│  scraper/           │   │  scraper/           │
│  colly.go           │   │  chromedp.go        │
│  (static scraping)  │   │  (dynamic scraping) │
└─────────────────────┘   └─────────────────────┘
```

---

## Async Job Flow (Crawl)

```
POST /v1/crawl
    │
    ▼
┌─────────────────────┐
│  handlers/crawl.go  │
│  EnqueueCrawl()     │
└─────────┬───────────┘
          │
          ▼
┌─────────────────────┐
│  Redis Queue        │  ← Job stored
│  (Asynq)            │
└─────────┬───────────┘
          │
          │ (async, later)
          ▼
┌─────────────────────┐
│  worker/handlers.go │
│  ProcessTask()      │
└─────────┬───────────┘
          │
          ▼
┌─────────────────────┐
│  scraper/service.go │
│  Scrape()           │
└─────────────────────┘
```

---

## Quick Commands

```bash
# Development
go run ./cmd/api          # Start server
go run ./cmd/worker       # Start standalone worker

# Building
go build -o bin/api ./cmd/api
go build -o bin/worker ./cmd/worker

# Testing
go test ./...             # All tests
go test -v ./...          # Verbose
go test -cover ./...      # With coverage

# Dependencies
go mod download           # Install deps
go mod tidy               # Clean up go.mod

# Code quality
go fmt ./...              # Format code
go vet ./...              # Static analysis
```

---

## Environment Variables

| Variable               | Default | Description                      |
| ---------------------- | ------- | -------------------------------- |
| `PORT`                 | `8080`  | HTTP server port                 |
| `SERVER_MODE`          | `debug` | `debug`, `release`, `test`       |
| `LOG_LEVEL`            | `info`  | `debug`, `info`, `warn`, `error` |
| `REDIS_URL`            | —       | Redis connection URL             |
| `BRAVE_SEARCH_API_KEY` | —       | Brave Search API key             |
| `DISABLE_WORKER`       | `false` | Disable embedded worker          |

---

## Further Reading

- [Official Go Tour](https://go.dev/tour/) – Interactive Go tutorial
- [Effective Go](https://go.dev/doc/effective_go) – Go best practices
- [Go by Example](https://gobyexample.com/) – Annotated code examples
- [Gin Documentation](https://gin-gonic.com/docs/) – HTTP framework used
- [Asynq Documentation](https://github.com/hibiken/asynq) – Job queue library
