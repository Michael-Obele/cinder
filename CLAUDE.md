# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project

Cinder is a self-hosted web scraping API in Go that turns websites into LLM-ready markdown (an open-source Firecrawl alternative). It ships as a **monolith with an embedded worker**: `cmd/api` runs both the Gin HTTP server and the Asynq background worker in one process, sized for 512MB–1GB hobby-tier hosts.

## Commands

```bash
make check        # fmt + vet + staticcheck + test — run this before finishing work
make test         # go test -v ./...
make fmt          # go fmt ./...
make vet          # go vet ./...
make staticcheck  # requires: go install honnef.co/go/tools/cmd/staticcheck@latest
```

```bash
go run ./cmd/api                                          # API + embedded worker
DISABLE_WORKER=true go run ./cmd/api                      # API only
go run ./cmd/worker                                       # standalone worker (exits 0 if no REDIS_URL)

go test ./internal/scraper/... -v                         # one package
go test ./internal/scraper/... -run TestShouldUseDynamic  # one test
go test ./internal/... ./pkg/... -race -count=1           # race, no cache
go test ./test/... -v                                     # integration (package `integration`, mocks only)
go test ./internal/... ./pkg/... -coverprofile=coverage.out && go tool cover -func=coverage.out
```

Docker: `docker build -t cinder . && docker run -p 8080:8080 -e SERVER_MODE=release cinder`. `docker-compose.yml` brings up Redis alongside. Deploy target is Fly.io via `fly.toml`.

Tests need no Redis and no Brave key — everything external is mocked. Chromedp tests need a local Chromium.

## Architecture

Dependency direction is one-way: `cmd/` → `internal/api/` → `internal/scraper/` → `internal/domain/`. Interfaces live in `internal/domain` (`domain.Scraper` is the seam), so `internal/*` should only reach for `domain` or `pkg/`. `internal/search` is fully standalone — it imports nothing from the project and defines its own `Service` interface and types.

**Startup wiring** (`cmd/api/main.go`) is the map of the system: config → logger → optional Redis client → Colly scraper + Chromedp scraper → `scraper.Service` → handlers → optional crawl handler + embedded worker + monitor scheduler → router.

**Engine selection** (`internal/scraper/service.go:55`) is the core decision point:
- `static` → Colly. `dynamic` → Chromedp. `smart` (default) → Colly first, then `ShouldUseDynamic(html)` (`heuristics.go`, SPA-shell markers + content-size checks) decides whether to redo it in Chromedp.
- Page `actions` force dynamic, and `static` + actions is a hard error. `screenshot` in smart mode goes straight to dynamic.
- Post-scrape enrichment runs in a fixed order: schema extraction → summary → PII redaction → image extraction/blob fetch (bounded `errgroup`, limit 5).

**Caching** is gzip-compressed JSON in Redis, 7-day TTL. The key is a SHA-256 of URL + mode + the *whole* `ScrapeOptions` struct (`cacheKeyFor`), so adding a field to `ScrapeOptions` automatically prevents stale hits — don't hand-roll a narrower key. Compression is deliberate (hobby-tier storage), and the read path still tolerates legacy uncompressed values. `Service.cache` is a two-method `cacheStore` interface rather than `*redis.Client` so tests can drive the cache branches; `NewService` still takes the concrete client and assigns it only when non-nil, because a nil `*redis.Client` in an interface is not a nil interface.

**Browser lifetime** (`internal/scraper/chromedp.go`): one shared exec allocator for the process, a lightweight tab per scrape, and a full allocator restart every `CHROME_RECYCLE_AFTER` scrapes to bound Chrome's memory growth. Never spawn a browser per request. The allocator warms up synchronously at startup so a missing Chromium surfaces as a startup warning and dynamic mode degrades instead of failing later.

**Async work** (`internal/worker/`): task types are declared in `tasks.go` (`scrape:url`, `crawl:site`, `monitor:check`) and registered on the mux in `server.go`. Asynq runs concurrency 5 across `critical`/`default`/`low` queues with a 15s `TaskCheckInterval` — both numbers are tuned for a 256MB VM and Upstash's free command budget, so treat them as intentional.

`crawl_handler.go:ExecuteCrawl` is a parallel BFS with per-domain politeness delay, retry-with-backoff that never retries 4xx, and a bounded queue that drops excess links rather than blocking a producer. It enforces its own deadline and returns status `"timeout"`; the Asynq task timeout is deliberately `crawlTimeout() + 5m` as an outer safety net. The seed URL is always scraped — `include_paths`/`exclude_paths` apply only to discovered links.

Crawl tuning is read from the environment at call time via `clampEnvInt` helpers at the bottom of `crawl_handler.go` (`CRAWL_CONCURRENCY`, `CRAWL_DOMAIN_DELAY`, `CRAWL_TIMEOUT`, `CRAWL_SCRAPE_TIMEOUT`, `CRAWL_MAX_RETRIES`) rather than through `internal/config` — follow whichever pattern the surrounding file already uses.

**Graceful degradation is a design rule.** Redis is optional: without it, `/v1/crawl` returns 503 and `/v1/batch` + `/v1/monitor` are never registered (`internal/api/router.go:79`). Readability failure returns the raw HTML with a nil error. Per-image fetch failures are logged and skipped. A scrape should not fail because an enrichment step did.

**SSRF defense** (`internal/safeurl`) is two-layered because either layer alone leaks. `safeurl.Client`/`Transport`/`Dialer` install a `net.Dialer.Control` hook that runs *after* DNS resolution, so it catches redirects and DNS rebinding; `safeurl.Check` is a pre-flight URL check used only where we don't own the connection (chromedp drives a real browser). Every outbound fetch path must go through one of them. Private, loopback, link-local, multicast and CGNAT addresses are refused by default; `SSRF_ALLOW_PRIVATE=true` opts out for operators scraping an internal wiki. Tests that use `httptest` bind to 127.0.0.1 and therefore need that variable set — packages do it in `TestMain` (see `internal/scraper/main_test.go`), and the tests that exercise the guard itself override it with `t.Setenv`.

**Shutdown** (`cmd/api/main.go`) drains the HTTP server and the embedded worker concurrently, not in sequence: `signal.NotifyContext` fires → the worker's `Shutdown()` starts in a goroutine → `srv.Shutdown` runs in the foreground → a `select` waits on whichever finishes first, bounded by `SHUTDOWN_TIMEOUT` (default 20s) and a 5s `workerDrainGrace`. Asynq's own shutdown cannot beat its `TaskCheckInterval` on an idle queue (`processor.go` sleeps uninterruptibly for roughly half of it), so 15s of check interval is a floor on worker drain — that is what the grace window absorbs.

Supporting packages: `internal/extract` (deterministic CSS-selector schema extraction, extractive summary, PII redaction — all LLM-free), `internal/image` (srcset/lazy-load/`<picture>` extraction, dimension sniffing, resize/re-encode), `internal/sitemap` (robots.txt/sitemap.xml discovery behind `/v1/map`), `internal/search` (Brave), `internal/safeurl` (SSRF guard), `pkg/logger` (slog wrapper).

## Conventions

- Log through `pkg/logger` (`logger.Log`). No `fmt.Println` or raw `log`. Packages whose tests exercise real callbacks need a `TestMain` calling `logger.Init("error")` or `logger.Log` is nil and they panic.
- Handle every error explicitly; wrap with `fmt.Errorf("...: %w", err)`. Don't use `_ = err`, don't `panic` for control flow, and don't start goroutines without a lifecycle.
- Propagate `context.Context` into anything blocking or networked.
- Table-driven tests with `t.Run` subtests; mock via the `domain.Scraper` / `search.Service` interfaces (see `MockSearchService`, `mockScraper`). Handler tests use `httptest` with `gin.SetMode(gin.TestMode)`.
- User agents come from `gofakeit.UserAgent()` — don't hardcode one.
- New config goes in `internal/config/config.go` (Viper + godotenv, `AutomaticEnv` with `.`→`_`), plus `.env.example` and the README table.

## Gotchas

- `.gitignore` ends with `*.json`, so **no JSON file in the repo is tracked** — including `internal/api/docs/swagger.json`. If a task needs a committed JSON file, it needs an explicit negation or a `git add -f`.
- In `debug` mode `cmd/api/main.go` shells out to `go run .../swag@latest init` on every startup to regenerate `internal/api/docs`. It needs network on first run and only logs on failure. Set `SERVER_MODE=release` to skip it. Swagger UI is served only in debug mode.
- Stale multi-hundred-MB binaries (`api`, `app`, `cinder-api`, `worker`) sit at the repo root. They are untracked now (`git rm --cached`, and `.gitignore` names them) but still exist on disk and in every past commit — they are not build outputs of the current tree, so don't rebuild into those paths or trust them. `scripts/purge-binaries-from-history.sh` removes them from history; it is deliberately not run for you, since it rewrites every SHA and needs a force-push.
- `/`, `/health`, and `/v1/ping` skip auth and rate limiting on purpose (load balancer and uptime probes). Everything else under `/v1` goes through `APIKeyAuth` + `RateLimit`, both no-ops until `API_KEYS` / `RATE_LIMIT_RPM` are set.
- Redis config resolves in precedence order: `REDIS_URL`, then `REDIS_HOST`/`PORT`/`PASSWORD`, then `UPSTASH_REDIS_REST_URL` + `UPSTASH_REDIS_REST_TOKEN` (derived into a `rediss://` URL). `rediss://` gets TLS with a TLS 1.2 minimum in `worker.RedisClientOpt`.
- `ScrapePayload.Render` is a deprecated bool that predates `Mode`; when true it overrides `Mode`. Prefer `Mode`.

## Docs

`README.md` is the API reference for endpoints, parameters, and env vars. `docs/guides/` holds deeper guides (`ARCHITECTURE.md`, `API_REFERENCE.md`, `TESTING.md`, `CODE_WALKTHROUGH.md`, plus Go-for-Svelte-devs onboarding). `docs/features/` documents each v2 feature. `plan/architecture.md` carries the original design rationale. `mastra_plan/` and `test_reports/` are historical.

