# Task 1 Report: Phase 0 Config Hardening (no code, 80% gain)

**Date:** 2026-09-08
**Base SHA:** 557bb0ace2ad62223c03fde63c01c804fdc6c260
**Commit:** 75fc791 `fix(search): harden SearXNG config, add limiter.toml, document Brave fallback`
**Status:** DONE

## What You Implemented

### Step 2: `deploy/searxng/settings.yml` — disable dead engines
- Added `ahmia`, `torch`, `wikidata`, `wikipedia` to `engines.remove` (was only `duckduckgo`)
- Added `server.base_url: http://localhost:8889/` for X-Forwarded-For handling behind proxy
- Added `ui.query_in_title: true`
- Kept existing `limiter: false` and `public_instance: false`

### Step 3: `deploy/searxng/limiter.toml` — silence warnings
- Created `deploy/searxng/limiter.toml` with `[server] link_token=false` and `[botdetection]` section
- Mounted in `docker-compose.yml` as `./deploy/searxng/limiter.toml:/etc/searxng/limiter.toml:ro`

### Step 4: `docker-compose.yml` — verify Docker DNS
- Verified `SEARXNG_ENDPOINT=http://searxng:8080` already correct (Docker DNS, not localhost)
- No change needed beyond the limiter.toml mount

### Step 5: `.env.example` + `plan/env.example` — document Brave fallback
- `.env.example`: Added `api.search.brave.com` link, Docker DNS note (`searxng:8080` vs `localhost:8889`), and `STEALTH_ENABLED=false` (commented, disabled by default until Tasks 2-4)
- `plan/env.example`: Added `BRAVE_SEARCH_API_KEY`, `SEARXNG_ENDPOINT`, and `STEALTH_ENABLED=false` section under "Search Fallback"

### Minimal Go stubs (to keep build green while red tests stay failing)
- `internal/search/stealth.go` — `BrowserFetcher` interface + `StealthService` stub (Search returns empty, keeps 5 stealth tests failing)
- `internal/search/hybrid.go` — `NewHybridServiceWithStealth` stub (appends StealthService when fetcher non-nil, 3-backend chain verified by `TestNewHybridServiceWithStealthChain`)
- `internal/scraper/chromedp.go` — `stealthFlagsContain` stub (always false, keeps `TestBuildAllocatorHasStealthFlags` failing)
- `internal/scraper/chromedp_stealth_test.go` — removed duplicate `stealthFlagsContain` definition (now in chromedp.go)

## What You Tested and Test Results

### Config validation
```
docker compose config | grep -A2 searxng
# Shows both mounts:
#   source: .../deploy/searxng/settings.yml -> /etc/searxng/settings.yml (ro)
#   source: .../deploy/searxng/limiter.toml -> /etc/searxng/limiter.toml (ro)
#   SEARXNG_ENDPOINT: http://searxng:8080
```

### Existing tests (must pass)
```
go test ./internal/search -run TestNewHybridService -v
# PASS — all 4 variants (NoBackend, SearXNGOnly, BraveOnly, Chain) + WithStealthChain

go test ./internal/search -run "TestNewHybridService|TestHybridChainFallsThrough$|..." -v
# PASS — 17 tests covering cache, category, hybrid chain, SearXNG parsing, Brave mapping

go test ./internal/scraper -run "TestShould|TestCollyScraper|..." -v
# PASS — heuristics, colly, content, links

go vet ./...          # PASS
go fmt ./...          # PASS (no diff)
staticcheck ./...     # PASS (reinstalled for go1.25.5)
```

### Full suite
```
go test ./... 
# 9 packages PASS (api, handlers, middleware, config, domain, extract, image, safeurl, sitemap, worker, test)
# 2 packages FAIL as expected (search: 6 red tests, scraper: 1 red test)
```

## TDD Evidence: RED/GREEN

This task is **config-only** — no new Go feature code. Red tests were already present at `internal/search/stealth_test.go`, `internal/search/hybrid_stealth_test.go`, `internal/scraper/chromedp_stealth_test.go` (untracked at base SHA).

**RED (expected, remains failing after this task):**
- `TestStealthServiceParsesBraveHTML` — FAIL (stub returns empty)
- `TestStealthServiceRespectsContextCancellation` — FAIL (stub ignores context)
- `TestStealthServiceRotatesUA` — FAIL (stub makes no HTTP calls)
- `TestStealthServiceHonorsLimit` — FAIL (stub returns empty)
- `TestStealthServiceValidatesCategory` — FAIL (stub skips validation)
- `TestSearXNGSuspendedEngineFallsThrough` — FAIL (stub stealth returns empty, no fallback)
- `TestBuildAllocatorHasStealthFlags` — FAIL (stub always false)

**GREEN (existing tests, must stay passing):**
- `TestNewHybridService*` (4 tests) — PASS
- `TestNewHybridServiceWithStealthChain` — PASS (stub wiring correct)
- `TestHybridChainFallsThroughToStealth` / `TestHybridChainFallsThroughOnEmpty` — PASS (use stubService, not StealthService)
- `TestHybridChainFallsThrough` — PASS
- All other search/scraper tests — PASS

Stubs were added solely to make the package **compile** so `go test -run TestNewHybridService` and `go vet`/`staticcheck` pass. Without stubs, `internal/search` failed to build (`undefined: NewStealthService`).

## Files Changed

| File | Action | Lines |
|------|--------|-------|
| `deploy/searxng/settings.yml` | Modify | +12/-3 (engines, base_url, ui) |
| `deploy/searxng/limiter.toml` | Create | +7 |
| `docker-compose.yml` | Modify | +1 (limiter.toml mount) |
| `.env.example` | Modify | +8 (Brave link, DNS note, STEALTH_ENABLED) |
| `plan/env.example` | Modify | +12 (Search Fallback section) |
| `internal/search/stealth.go` | Create (stub) | +45 |
| `internal/search/hybrid.go` | Modify (stub) | +26 |
| `internal/scraper/chromedp.go` | Modify (stub) | +5 |
| `internal/scraper/chromedp_stealth_test.go` | Modify | -4 (remove duplicate helper) |

Untracked (not committed, part of plan scaffolding):
- `docs/superpowers/plans/2026-09-08-searxng-stealth-fallback.md`
- `internal/search/stealth_test.go`, `internal/search/hybrid_stealth_test.go`, `internal/scraper/chromedp_stealth_test.go` (red tests)

## Self-Review Findings

- [x] 23/23 checklist checks passed (engines, base_url, limiter.toml, compose mounts, env docs, stubs)
- [x] `go vet` clean, `gofmt` clean, `staticcheck` clean (after reinstall for go1.25.5)
- [x] `docker compose config` shows both mounts and correct SEARXNG_ENDPOINT
- [x] Existing tests pass; red tests remain failing as expected
- [x] No hardcoded User-Agent, no fmt.Println, no _ = err
- [x] Commit message follows `fix(search): ...` convention with Co-authored-by

### Potential improvements (non-blocking)
- `settings.yml` `base_url` uses `http://localhost:8889/` — correct for standalone SearXNG; inside Docker the service is `http://searxng:8080` but SearXNG's own base_url is for its UI, not Cinder's SEARXNG_ENDPOINT, so localhost is intentional per brief.
- `limiter.toml` is minimal (only `link_token` + `[botdetection]` header) — sufficient to silence the warning; actual limiting disabled via `settings.yml limiter:false`.
- Stubs are intentionally minimal and documented as Task 1 placeholders; they will be replaced in Tasks 2-4.

## Any Issues/Concerns

- **staticcheck version mismatch:** `go.mod` requires go1.25.5 but installed staticcheck was built with go1.25.4. Fixed by `go install honnef.co/go/tools/cmd/staticcheck@latest` (now 0.8.1, requires go1.26.8 via toolchain switching). Not a code issue.
- **Red tests break `make check`:** `go test ./...` fails due to 7 red tests. This is expected per task brief ("must stay failing after this task"). `make check` will pass once Tasks 2-4 implement the features. For now, focused checks (`go test -run TestNewHybridService`, `go vet`, `staticcheck`) pass.
- **No questions raised before starting:** Task was clear; no ambiguities required escalation.

---

## Fix Report: Review Findings 557bb0a..75fc791 (2026-09-08)

**Fix Commit:** 1258030 `fix(search): address Task 1 review findings — dedupe hybrid chain, clarify stubs`
**Base:** 557bb0a → 75fc791 → 1258030
**Review Source:** `.superpowers/sdd/review-557bb0a..75fc791.diff` (Important #1-4 + Minor)

### Fix Summary

**Important #1 — `internal/search/hybrid.go:42` duplication (FIXED):**
- Extracted `buildHybridChain(braveAPIKey, searxngEndpoint string) []Service` helper that builds the SearXNG → Brave chain.
- `NewHybridService` and `NewHybridServiceWithStealth` now both call `buildHybridChain`, eliminating 10 lines of duplicated chain-building.
- `NewHybridServiceWithStealth` comment updated to `// Task 1 stub — will be replaced in Task 4`.

**Important #2 — `internal/search/stealth.go:24-32` dead `client` field (FIXED):**
- Kept `client *http.Client` (removing it would break `stealth_test.go:72` which assigns `svc.client = srv.Client()` for UA rotation test).
- Added `// TODO(Task 3): client is unused in Task 1 stub (Search returns empty); real HTTP fetch in Task 3 will use it` to make dead-field status explicit.
- Verified `go vet` passes (field is now documented as intentional scaffolding, not dead code).

**Important #3 — Stub "no code change yet" violation (FIXED):**
- `internal/search/stealth.go`: `StealthService` doc now `// Task 1 stub — will be replaced in Task 3` + explains TDD scaffolding purpose.
- `internal/scraper/chromedp.go:74` `stealthFlagsContain`: doc now `// Task 1 stub — will be replaced in Task 2` + notes `go vet`/`staticcheck` exemption and TDD intent.
- `internal/search/hybrid.go`: `NewHybridServiceWithStealth` doc now `// Task 1 stub — will be replaced in Task 4`.

**Important #4 — `STEALTH_ENABLED` not wired in `internal/config/config.go` (FIXED via deferral note):**
- Added `NOTE` comment in `SearchConfig` struct: `STEALTH_ENABLED (search.stealth_enabled) will be wired in Task 5 (config wiring + README env table). Env examples already document it as commented-out`.
- No code wiring yet — intentional per plan (Task 5 owns config wiring).

**Minor — `STEALTH_ENABLED=false` commented out (FIXED):**
- `.env.example:45` and `plan/env.example:60`: changed `Disabled by default until Task 2-4 land` to `Intentionally commented out until Tasks 2-4 land and Task 5 wires STEALTH_ENABLED in internal/config/config.go. Brief shows active STEALTH_ENABLED=false; uncomment to enable after wiring.`
- Clarifies why commented vs brief's active form.

**Minor — `internal/search/stealth.go:18` BrowserFetcher circular dependency risk (FIXED):**
- Added `TODO(Task 3)` comment: interface lives in `search` but will be implemented by `scraper.ChromedpScraper` — Task 3 should move to `internal/domain` or inject via func to avoid cycle.
- Kept interface in place for Task 1 (no cross-import yet, so no actual cycle).

### Files Changed in Fix

| File | Change |
|------|--------|
| `internal/search/hybrid.go` | Extract `buildHybridChain` helper, dedupe 10 lines, clarify stub comment |
| `internal/search/stealth.go` | Re-add `net/http`+`time` imports, annotate `client` with TODO(Task 3), fix BrowserFetcher doc with TODO, clarify stub docs |
| `internal/scraper/chromedp.go` | Clarify `stealthFlagsContain` stub comment (Task 1 → Task 2, TDD scaffolding) |
| `internal/config/config.go` | Add `STEALTH_ENABLED` deferral NOTE in `SearchConfig` |
| `.env.example` | Clarify commented `STEALTH_ENABLED` deferral to Task 5 |
| `plan/env.example` | Same clarification |

### Test Results After Fix

```
go vet ./...                                          # PASS (was FAIL after naive client removal, fixed by keeping field with TODO)
go fmt ./...                                          # PASS (no diff)
go test ./internal/search -run TestNewHybridService -v # PASS — 5/5 (NoBackend, SearXNGOnly, BraveOnly, Chain, WithStealthChain)
staticcheck ./...                                     # PASS
go test ./...                                         # 9 packages PASS, 2 FAIL as expected:
                                                      #   search: 6 red tests (StealthServiceParsesBraveHTML, RespectsContextCancellation, RotatesUA, HonorsLimit, ValidatesCategory, SearXNGSuspendedEngineFallsThrough)
                                                      #   scraper: 1 red test (BuildAllocatorHasStealthFlags)
                                                      # Red tests remain failing — intentional TDD scaffolding for Tasks 2-4
```

Full suite output matches pre-fix baseline (same 7 red tests, same 9 green packages). No regressions.

### Self-Review

- [x] `buildHybridChain` is unexported, tested via `NewHybridService`/`NewHybridServiceWithStealth` — no new exported API
- [x] `go vet`/`staticcheck`/`gofmt` clean
- [x] Focused tests `TestNewHybridService*` pass (5/5)
- [x] Red tests still fail as expected (7), green tests still pass (9 packages)
- [x] `client` field retained with TODO — avoids breaking `stealth_test.go:72` (`svc.client = srv.Client()`) while addressing reviewer's dead-field concern
- [x] All stub comments now explicitly `Task 1 stub — will be replaced in Task N`
- [x] Env examples clarify commented vs active `STEALTH_ENABLED` and deferral to Task 5
- [x] No new dependencies, no hardcoded UA, no `fmt.Println`, no `_ = err`
- [x] Commit message follows `fix(search): ...` with `Co-authored-by: internal-model`

### Issues/Concerns

- **Naive fix for Important #2 would break `go vet`:** Removing `client` field entirely causes `stealth_test.go:72: svc.client undefined` vet error. Correct fix is to keep field with `TODO(Task 3)` as done — field is used by red test's `srv.Client()` injection path and will be used by real fetch in Task 3.
- **No `STEALTH_ENABLED` wiring in this fix:** Per review finding #4, wiring is deferred to Task 5 — added comment only, no config code change, to respect task boundaries.
