# Cinder: The Svelte/JS Developer's Guide

Welcome to Cinder! If you're coming from the JavaScript/TypeScript ecosystem (SvelteKit, Next.js, Node.js), this guide is designed to translate Go concepts into terms you already understand.

## 🗺️ Project Map (File Tree)

Here is the layout of the project, annotated with "JS equivalents".

```text
cinder/
├── cmd/                        # 🚀 "scripts/" or "entry points"
│   ├── api/main.go             # The Express/Hono Server entry point
│   └── worker/main.go          # The Background Worker entry point
├── internal/                   # 🔒 "src/" (Private code)
│   ├── api/                    # 🌐 API Routes & Middleware
│   │   ├── router.go           # Like SvelteKit hooks + routes setup
│   │   └── handlers/           # The actual route handlers (controllers)
│   ├── config/                 # ⚙️ dotenv loader
│   ├── domain/                 # 📝 TypeScript Interfaces / Types
│   ├── scraper/                # 🧠 Business Logic (The Service Layer)
│   └── worker/                 # 👷 Background Job Logic (BullMQ consumer)
├── pkg/                        # 📦 Shared libraries (can be imported by others)
├── docs/                       # 📚 Documentation
│   └── guides/
└── go.mod                      # 📦 package.json
```

---

## 🔄 The Rosetta Stone: Go vs JS

| Concept               | Go                                     | JavaScript / SvelteKit             |
| :-------------------- | :------------------------------------- | :--------------------------------- |
| **Server**            | `Gin`                                  | `Hono` / `Express`                 |
| **Database/ORM**      | `Redis` (Raw)                          | `Redis` (ioredis)                  |
| **HTML Parser**       | `Colly`                                | `Cheerio`                          |
| **Headless Browser**  | `Chromedp`                             | `Puppeteer` / `Playwright`         |
| **Job Queue**         | `Asynq`                                | `BullMQ`                           |
| **Concurrency**       | Goroutines (`go func()`)               | `Promise` / `async/await`          |
| **Interfaces**        | Implicit (`type X interface`)          | TypeScript Interfaces              |
| **Project Structure** | Standard Go Layout (`cmd`, `internal`) | Framework Dependent (`src/routes`) |

---

## 🌊 The Flow: Life of a Request

Let's follow a request to `POST /v1/scrape`.

### 1. The Entry Point (`cmd/api/main.go`)

This is like your `server.ts`. It loads the config (`.env`), connects to Redis, initializes the services, and starts the HTTP server.

- **Key Detail**: It runs in "Monolith Mode" by default now. It spins up the **Worker** in a background Goroutine so you don't need a separate container during development!.

### 2. The Router (`internal/api/router.go`)

This sets up the routes.

```go
v1.POST("/scrape", scrapeHandler.Scrape)
```

Equivalent to SvelteKit's `export const POST = ...`.

### 3. The Handler (`internal/api/handlers/scrape.go`)

This verifies the input (JSON body).

- **Validates**: Checks if `url` is present.
- **Maps**: Converts "render: true" to "mode: dynamic" for backward compatibility.
- **Calls Service**: Passes control to the detailed logic layer.

### 4. The Service (`internal/scraper/service.go`)

This is the "Brain". It decides _how_ to scrape.

1.  **Check Cache**: Looks in Redis for `scrape:<url>:<mode>`. If found, returns immediately (fast!).
2.  **Mode Switch**:
    - **Static**: Uses `Colly` (HTTP Request + HTML parsing). Fast, low resource usage.
    - **Dynamic**: Uses `Chromedp` (Headless Chrome). Spins up a browser tab, executes JS, waits for network idle. Slower, but accurate for SPAs.
    - **Smart**: (Default) Can be configured to try Static first, then Dynamic if needed.

### 5. The Scraper (`internal/scraper/`)

- `colly.go`: The "Cheerio" wrapper.
- `chromedp.go`: The "Puppeteer" wrapper. Instead of launching a full browser per request, it connects to a shared browser instance (Context) to save memory (The "Context Pattern").

---

## 🏗️ Architecture Improvements

We recently refactored the project to be friendlier for "Hobby" deployments (like Railway, Render, or Leapcell).

### The Monolith Pattern

Previously, you needed to run two commands:

1. `go run cmd/api/main.go`
2. `go run cmd/worker/main.go`

Now, **`cmd/api/main.go` does both!**
It checks if `DISABLE_WORKER` is false (default), and initializes the `asynq` server inside the same process. This means "Scale to Zero" works perfectly—one container handles the API _and_ processes background crawl jobs.

### Concurrency

We tuned the concurrency limit to **10** (up from 2). Because we use shared browser contexts instead of spawning new chrome processes, memory usage is stable (~500MB), allowing higher throughput on small VPS instances.

---

## 🚀 Running Locally

You only need one command now:

```bash
go run cmd/api/main.go
```

This starts:

- HTTP API on `:8080`
- Background Worker (listening to Redis)

**Test it:**

```bash
curl -X POST http://localhost:8080/v1/scrape \
  -d '{"url": "https://example.com", "mode": "static"}'
```

> **🔥 Pro-Tip for Svelte Devs:** For a comprehensive guide on how to integrate this backend with your Vite/Svelte frontend, as well as how to **Test** and **Debug** using VS Code, see the [Svelte Dev Workflow Guide](SVELTE_DEV_WORKFLOW.md).

---

_This guide was generated by analyzing the codebase with `mcp_sequentialthi_sequentialthinking` and verified against the running application._
