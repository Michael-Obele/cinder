# Actionable Todos

## Phase 1: Setup & Static Scraping

- [ ] **Init Project**: Run `go mod init github.com/yourname/cinder`.
- [ ] **Install Deps**: `go get -u github.com/gin-gonic/gin github.com/gocolly/colly/v2 github.com/spf13/viper github.com/brianvoe/gofakeit/v6`.
- [ ] **Config**: Create `internal/config` package to load `env` variables using Viper.
- [ ] **Logger**: Set up a structured logger (slog or zap) in `pkg/logger`.
- [ ] **Scraper Interface**: Define the `Scraper` interface in `internal/domain`.
- [ ] **Colly Impl**: Implement the static scraper in `internal/scraper/colly.go`.
  - [ ] Configure User-Agent rotation using `gofakeit`.
  - [ ] Add `html-to-markdown` conversion.
- [ ] **API Handler**: Create `internal/api/handlers/scrape.go`.
- [ ] **Router**: Wire up `POST /v1/scrape` in `internal/api/router.go`.
- [ ] **Test**: Verify scraping a simple HTML page (e.g., `http://example.com`).

## Phase 2: Dynamic Scraping (Chromedp)

- [ ] **Install Deps**: `go get -u github.com/chromedp/chromedp`.
- [ ] **Chromedp Impl**: Implement dynamic scraper in `internal/scraper/chromedp.go`.
  - [ ] Setup `chromedp.NewContext`.
  - [ ] Implement `chromedp.Navigate`, `chromedp.WaitVisible`, `chromedp.OuterHTML`.
- [ ] **Smart Switch**: Update `internal/scraper/service.go` to choose between Colly/Chromedp based on `render: true` flag.
- [ ] **Docker**: Create `Dockerfile` with Chromium installation (see `architecture.md`).
- [ ] **Test**: Verify scraping a React site (e.g., a dynamic todo app).

## Phase 3: Async Queue (Asynq)

- [ ] **Install Deps**: `go get -u github.com/hibiken/asynq`.
- [ ] **Redis Setup**: Configure `asynq.RedisClientOpt` in `internal/config` (ensure TLS support).
- [ ] **Task Definition**: Create `internal/worker/tasks.go` (define `TypeCrawl`).
- [ ] **Task Handler**: Create `internal/worker/handlers.go` (logic to call Scraper Service).
- [ ] **Server**: Create `cmd/worker/main.go` to run the Asynq server.
- [ ] **API Update**: Add `POST /v1/crawl` to enqueue tasks.
- [ ] **Status Endpoint**: Add `GET /v1/crawl/:id` to query job status (requires storing results in Redis or Postgres).

## Phase 4: Polish & Auth

- [ ] **Middleware**: Implement `APIKeyAuth` middleware in `internal/api/middleware/auth.go`.
- [ ] **Apply Middleware**: Protect `/v1/*` routes in `router.go`.
- [ ] **Rate Limiting**: Add `gin-contrib/rate` or custom Redis rate limiter.
- [ ] **Cleanup**: Ensure `defer cancel()` is called on all contexts to prevent memory leaks.
- [ ] **Documentation**: Generate Swagger/OpenAPI spec if needed.
