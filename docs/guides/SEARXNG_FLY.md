# Deploy SearXNG on Fly.io for Cinder Search

> [!TIP]
> Cinder's `/v1/search` uses a self-hosted SearXNG when `SEARXNG_ENDPOINT` is
> set, falling back to the `BRAVE_SEARCH_API_KEY` secret otherwise. SearXNG
> aggregates Google, Bing, DuckDuckGo, Brave, Mojeek and Wikipedia — free,
> no per-query cost, and no single fragile engine to scrape.

This guide deploys SearXNG as its **own Fly app** (it is *not* bundled into
the Cinder image — the Cinder Dockerfile only builds `cinder-api` and
`cinder-worker`). Once deployed, Cinder reaches it over Fly's private
network (`http://searxng.internal`), which is free between your apps.

## Prerequisites

- `flyctl` installed and logged in (`fly auth login`)
- The Cinder repo cloned locally (for the `fly secrets` step)

## Step 1 — Create the SearXNG app

SearXNG's official image is `searxng/searxng`. We bake a `settings.yml` into
it so the JSON API (which Cinder queries) is enabled — the stock image has
JSON disabled.

```bash
mkdir searxng-fly && cd searxng-fly
```

**`Dockerfile`**

```dockerfile
FROM searxng/searxng:latest
COPY settings.yml /etc/searxng/settings.yml
```

**`settings.yml`** — replace the secret key with a random 64-char string
(`openssl rand -hex 32`):

```yaml
use_default_settings: true

server:
  # Generate with: openssl rand -hex 32
  secret_key: "REPLACE_WITH_64_HEX_CHARS"
  limiter: false
  public_instance: false

search:
  formats:
    - html
    - json   # <-- required: Cinder queries the JSON API
```

## Step 2 — Scaffold and deploy

```bash
# Scaffold the Fly app (generates fly.toml, no deploy yet)
fly launch --name searxng --no-deploy

# SearXNG listens on 8080 inside the container — make sure fly.toml has:
#   [http_service]
#     internal_port = 8080
# (fly launch usually detects it; verify before deploying)

fly deploy
```

Give the machine a moment to boot, then check the JSON API responds:

```bash
curl -s "https://searxng.fly.dev/search?q=golang&format=json" | head -c 200
```

## Step 3 — Point Cinder at it

Set the endpoint as a **secret** (stays out of the repo — cleaner than the
commented `SEARXNG_ENDPOINT` in `fly.toml`):

```bash
cd ~/Documents/GitHub/cinder

# Private network address — free, not exposed publicly
fly secrets set SEARXNG_ENDPOINT=http://searxng.internal

# ...or, if you want it reachable publicly:
# fly secrets set SEARXNG_ENDPOINT=https://searxng.fly.dev
```

> [!NOTE]
> `fly secrets set` triggers a redeploy of the Cinder app. If you prefer
> config-in-repo, uncomment `SEARXNG_ENDPOINT` in `fly.toml` instead — but
> secrets keep the URL out of version control.

## Step 4 — Verify end to end

```bash
# SearXNG direct (JSON API)
curl -s "https://searxng.fly.dev/search?q=fly.io+deployment&format=json" | \
  python3 -c "import json,sys; print(len(json.load(sys.stdin)['results']), 'results')"

# Through Cinder
curl -s -X POST https://cinder9630.fly.dev/v1/search \
  -H 'Content-Type: application/json' \
  -d '{"query":"fly.io deployment","limit":3}'
```

If search still falls back to Brave, confirm the secret landed:
`fly secrets list | grep SEARXNG`.

## Local equivalent (docker-compose)

The Cinder repo's `docker-compose.yml` already runs SearXNG as a sibling
service (port `8889` host → `8080` container) with the JSON API enabled via
`deploy/searxng/settings.yml`. Local `.env` points `SEARXNG_ENDPOINT` at
`http://localhost:8889`:

```bash
docker compose up -d searxng     # start just the search backend
go run ./cmd/api                 # local Cinder now searches via SearXNG
```

## Notes & troubleshooting

- **Empty results**: SearXNG occasionally returns zero results when all its
  upstream engines hiccup at once (visible in its logs as engine timeouts /
  DDG CAPTCHAs). Cinder does **not** cache empty result sets, so the next
  request re-queries and recovers. If you want a hard fallback for these
  windows, also set `BRAVE_SEARCH_API_KEY` — Cinder tries SearXNG first,
  then Brave.
- **Memory**: SearXNG is light (~150 MB). Fly's default shared-1x-256MB VM
  is enough; bump to 512MB if you enable many engines.
- **Rate limiting**: Cinder deliberately does **not** rate-limit SearXNG
  client-side — SearXNG handles concurrency internally (its own limiter is
  disabled in the settings above, which is fine for a private instance).
- **Private networking**: `http://searxng.internal` only resolves between
  apps in the same Fly organization. It is the recommended address — no
  public exposure, no egress cost.