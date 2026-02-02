# Valibot Config Schema Plan

## Goals
- Centralize configuration validation for API + worker.
- Mirror existing Go env vars (Port, Redis, logging, etc.).
- Provide explicit error messages at startup.

## Valibot Highlights
- Schema-based runtime validation.
- No dependencies, small bundle size.
- `parse` throws on invalid data; `safeParse` returns structured errors.

## Proposed Config Domains
### Server
- `PORT`
- `SERVER_MODE` (debug/release/test)

### Redis / BullMQ
- `REDIS_URL` (supports TLS)
- `REDIS_HOST`, `REDIS_PORT`, `REDIS_PASSWORD` (optional)

### Worker
- `WORKER_CONCURRENCY`
- `WORKER_USE_THREADS`

### Scraper
- `PLAYWRIGHT_HEADLESS`
- `PLAYWRIGHT_TIMEOUT_MS`
- `SMART_MODE_THRESHOLD`

### Logging
- `LOG_LEVEL`

## Validation Behavior
- Fail fast on missing required values.
- Provide explicit messages for invalid ports or URLs.
- Allow defaults for optional tuning variables.

## Output
- Produce a single config object consumed by API + worker.

## Sources
- Valibot docs: Introduction and Schemas guides (modular, type-safe schemas; parse/safeParse APIs).
