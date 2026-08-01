# Monitor, Extraction & Cleaner Output

> Feature spec — implemented 2026-08-01 (`internal/worker/monitor.go`, `internal/extract`, `internal/scraper/content.go`)

## 1. Change-Tracking Monitors

All intelligence here is **deterministic — no LLM anywhere**.

Flow: a background scheduler (30s tick) scans Redis for monitors whose
`next_check` has passed → enqueues `monitor:check` tasks → the handler
scrapes (smart mode), hashes the markdown (SHA-256), and compares to the
stored hash.

- **First check**: stores the baseline, no notification.
- **Change detected**: stores the new hash + fires the signed webhook.
- **No change**: silent; `next_check` advances by `interval_seconds`.

```json
// POST /v1/monitor
{
  "url": "https://pricing.example.com",
  "interval_seconds": 3600,
  "webhook_url": "https://myapp.example.com/hooks/price-changed",
  "webhook_secret": "s3cret"
}
```

- `interval_seconds` minimum: **3600** (1h).
- `GET /v1/monitor/:id` → config, `last_hash`, `next_check`.
- `DELETE /v1/monitor/:id` → stops monitoring.
- Requires Redis; survives restarts (state lives in Redis).

## 2. Deterministic JSON Extraction

Selector-based extraction evaluated with goquery — the "deterministicJson"
pattern without the LLM:

```json
{
  "url": "https://product.example.com",
  "extract_schema": {
    "title":     { "selector": "h1" },
    "price":     { "selector": ".price" },
    "sku":       { "selector": "meta[itemprop=sku]", "attr": "content" },
    "gallery":   { "selector": ".gallery img", "attr": "src", "multiple": true }
  }
}
```

`attr`: `text` (default), `html`, or any attribute name. Missing selectors
omit the field; `multiple: true` collects arrays. Result lands in the
response `extracted` object.

## 3. Extractive Summary

`summary: true` adds a `summary` field — no LLM. A readability excerpt wins
when present; otherwise the top-N sentences (default 5,
`summary_sentences`) by term frequency with a position bonus, returned in
document order.

## 4. PII Redaction

`redact_pii: true` masks in markdown + summary:

| Pattern | Token |
| ------- | ----- |
| Emails | `[EMAIL]` |
| Phone-shaped numbers (requires separators) | `[PHONE]` |
| Card-shaped digit runs (13-19 digits) | `[CARD]` |

Regexes live in `internal/extract/redact.go` and are table-tested. Order
matters: card runs are masked before phone matching so digit runs with
spaces are never mis-tokenized.

## 5. Cleaner Output Defaults

Applied before markdown conversion (both engines):

- `block_ads: true` (default) — strips common ad/tracker containers
  (`.ad`, `[class*='advert']`, doubleclick iframes, …).
- `remove_base64_images: true` (default) — drops inline `data:` images.

Pass `false` explicitly to disable either.

## 6. Auth & Rate Limiting

- `API_KEYS=sk_a,sk_b` → `X-API-Key` required on `/v1/*` (401 otherwise).
  Unset = open API (documented).
- `RATE_LIMIT_RPM=60` → per-client sliding minute window. Redis-backed when
  `REDIS_URL` set, in-memory fallback otherwise; 429 + `retry_after` on
  excess.
