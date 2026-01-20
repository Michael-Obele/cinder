# Cinder 🔥

<!-- [![Go Version](https://img.shields.io/github/go-mod/go-version/standard-user/cinder)](https://golang.org) -->

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Status](https://img.shields.io/badge/Status-Beta-blue)](https://github.com/standard-user/cinder)

**Cinder** is a high-performance, self-hosted web scraping API built with Go. It turns any website into LLM-ready markdown, designed as a drop-in alternative to Firecrawl.

> **Why Cinder?** Heavily optimized for low-memory, serverless, and "hobby tier" environments by using intelligent browser process management and a unified "monolith" architecture.

---

## ✨ Features

- **⚡ Fast & Efficient**: Reuses a single Chrome process with lightweight tabs, avoiding the heavy startup cost of spawning browsers per request.
- **🏭 Monolith Mode**: Runs the API and Async Worker in a single binary/container. Perfect for services like Railway or Leapcell where you pay per active container.
- **🔄 Async Queues**: Redis-backed job queue (Asynq) for handling heavy scrape jobs without blocking HTTP clients.
- **🧠 LLM Ready**: Converts complex HTML/SPAs into clean, structured Markdown using `html-to-markdown/v2`.
- **🕵️ Evasion**: Automatic User-Agent rotation and un-detected headless flags.

---

## 🚀 Quickstart

### Prerequisites

- **Go 1.25+**
- **Redis** (Required for async crawling, optional for simple scraping)
- **Chromium** (Installed automatically by `chromedp` or via Docker)

### Installation

```bash
git clone https://github.com/standard-user/cinder.git
cd cinder
go mod download
```

### Configuration (.env)

```dotenv
PORT=8080
API_KEY=secret_key
# Redis is required for /crawl endpoints and async jobs
REDIS_URL=redis://localhost:6379
# Set to 'true' to run api and worker separately (Microservices mode)
# Default is 'false' (Monolith mode)
DISABLE_WORKER=false
```

### Running

**Option 1: Monolith Mode (Recommended)**
Runs the API server and the Queue Worker in the same process.

```bash
go run ./cmd/api
```

**Option 2: Docker**

```bash
docker build -t cinder .
docker run -p 8080:8080 cinder
```

---

## 🔌 API Endpoints

### 1. Synchronous Scrape

Best for single pages. Blocks until done.
`POST /scrape`

```json
{
  "url": "https://example.com"
}
```

### 2. Async Crawl

Best for entire sites. Returns a Job ID.
`POST /crawl`

```json
{
  "url": "https://example.com/blog",
  "depth": 2
}
```

---

## 🏗️ Architecture

Cinder uses a **Tiered Scraping** approach:

1.  **Request**: API accepts JSON.
2.  **Dispatch**:
    - **Simple**: Handled immediately.
    - **Complex**: Pushed to Redis Queue.
3.  **Optimization**:
    - **Browser**: A specific `ChromedpScraper` singleton maintains a `ExecAllocator`.
    - **Tabs**: Requests spawn `NewContext` (tabs) on the existing browser, saving ~500ms per request.

See [plan/architecture.md](plan/architecture.md) for details.

---

## 🗺️ Roadmap & Status

| Phase       | Goal                          | Status         |
| :---------- | :---------------------------- | :------------- |
| **Phase 1** | Static Scraping (Colly)       | ✅ Done        |
| **Phase 2** | Dynamic Scraping (Chromedp)   | ✅ Done        |
| **Phase 3** | Async Queue (Asynq + Redis)   | ✅ Done        |
| **Phase 4** | Performance Tuning (Monolith) | ✅ Done        |
| **Phase 5** | Hardening & Testing           | 🚧 In Progress |

**Current Focus**:

- Adding a comprehensive Unit & Integration Test Suite (Currently 0% coverage).
- Implementing "Smart Wait" heuristics for slower SPAs.
- Adding a "Browser Health Check" to kill/restart Chrome after N scrapes.

---

## 🤝 Contributing

Contributions are welcome!
**Please Note**: We are currently focused on adding **Tests**. If you submit a PR, please try to include a `_test.go` file for your logic.

1. Fork the Project
2. Create your Feature Branch
3. Commit your Changes
4. Push to the Branch
5. Open a Pull Request

---

## ⚖️ License

Distributed under the MIT License. See `LICENSE` for more information.
