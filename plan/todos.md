# Actionable Todos

## Phase 1: Setup & Static Scraping

- [x] **Init Project**: Run `go mod init github.com/standard-user/cinder`.
- [x] **Install Deps**: `go get -u github.com/gin-gonic/gin github.com/gocolly/colly/v2 github.com/spf13/viper github.com/brianvoe/gofakeit/v6`.
- [x] **Config**: Create `internal/config` package to load `env` variables using Viper.
- [x] **Logger**: Set up a structured logger (slog or zap) in `pkg/logger`.
- [x] **Scraper Interface**: Define the `Scraper` interface in `internal/domain`.
- [x] **Colly Impl**: Implement the static scraper in `internal/scraper/colly.go`.
  - [x] Configure User-Agent rotation using `gofakeit`.
  - [x] Add `html-to-markdown` conversion.
- [x] **API Handler**: Create `internal/api/handlers/scrape.go`.
- [x] **Router**: Wire up `POST /v1/scrape` in `internal/api/router.go`.
- [x] **Test**: Verify scraping a simple HTML page (e.g., `http://example.com`).

## Phase 2: Dynamic Scraping (Chromedp)

- [x] **Install Deps**: `go get -u github.com/chromedp/chromedp`.
- [x] **Chromedp Impl**: Implement dynamic scraper in `internal/scraper/chromedp.go`.
  - [x] Setup `chromedp.NewContext`.
  - [x] Implement `chromedp.Navigate`, `chromedp.WaitVisible`, `chromedp.OuterHTML`.
- [x] **Smart Switch**: Update `internal/scraper/service.go` to choose between Colly/Chromedp based on `render: true` flag.
- [x] **Docker**: Create `Dockerfile` with Chromium installation (see `architecture.md`).
- [x] **Test**: Verify scraping a React site (e.g., a dynamic todo app).

## Phase 3: Async Queue (Asynq)

- [x] **Install Deps**: `go get -u github.com/hibiken/asynq`.
- [x] **Redis Setup**: Configure `asynq.RedisClientOpt` in `internal/config` (ensure TLS support).
- [x] **Task Definition**: Create `internal/worker/tasks.go` (define `TypeCrawl`).
- [x] **Task Handler**: Create `internal/worker/handlers.go` (logic to call Scraper Service).
- [x] **Server**: Create `cmd/worker/main.go` to run the Asynq server.
- [x] **API Update**: Add `POST /v1/crawl` to enqueue tasks.
- [x] **Status Endpoint**: Add `GET /v1/crawl/:id` to query job status (implemented using Asynq Inspector).

## Phase 4: Polish & Auth

- [ ] **Middleware**: Implement `APIKeyAuth` middleware in `internal/api/middleware/auth.go`.
- [ ] **Apply Middleware**: Protect `/v1/*` routes in `router.go`.
- [ ] **Rate Limiting**: Add `gin-contrib/rate` or custom Redis rate limiter.
- [ ] **Cleanup**: Ensure `defer cancel()` is called on all contexts to prevent memory leaks.
- [ ] **Documentation**: Generate Swagger/OpenAPI spec if needed.

## Phase 5: High Performance & Reliability (Leapcell/Upstash)

- [ ] **Refactor Scraper**: Move `chromedp` Allocator to a specific Service/Singleton to reuse the browser process.
- [ ] **Tuning**: Increase Worker concurrency to `5+` and reduce Asynq polling interval (`1s`).
- [ ] **Smart Waiting**: Implement `WaitVisible` or Network Idle detection in `chromedp.go`.
- [ ] **Stability**: Implement periodic browser restarts (every ~100 scrapes) to prevent memory leaks.
- [ ] **Resilience**: Tune Redis timeouts for high-latency environments.
