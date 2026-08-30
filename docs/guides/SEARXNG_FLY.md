# Deploy SearXNG on Fly.io for Cinder Search

> [!TIP]
> Cinder's `/v1/search` uses a self-hosted SearXNG when `SEARXNG_ENDPOINT` is
> set, falling back to the `BRAVE_SEARCH_API_KEY` secret otherwise. SearXNG
> aggregates Google, Bing, DuckDuckGo, Brave, Mojeek and Wikipedia — free,
> no per-query cost, and no single fragile engine to scrape.

SearXNG is _not_ bundled into the Cinder image (the Cinder Dockerfile only
builds `cinder-api` and `cinder-worker`), but there are two ways to run it
on Fly with one app to manage:

- **Option A — same app, separate machine** (sidecar): add a second Fly
  machine running SearXNG _inside_ the `cinder9630` app. One app, one
  dashboard entry, private-network access for free.
- **Option B — separate app**: SearXNG as its own Fly app. Cleanest
  isolation, but a second app to manage.

Both reach SearXNG over Fly's private network (`*.internal`), which is free.

## Option A — Same app, separate machine (sidecar)

The repo already ships the pieces: `deploy/searxng-fly/Dockerfile` bakes the
JSON-enabled `deploy/searxng/settings.yml` into the official image.

> [!TIP]
> There is an idempotent helper for all of this — `scripts/fly-searxng.sh`
> (`deploy` / `status` / `destroy`). It builds, pushes, creates or updates the
> machine, sets the secret, and verifies from inside the Cinder machine. The
> manual steps below are what it runs.

### 1. Build and push the sidecar image

> [!NOTE]
> Fly's registry only accepts repositories named after an **app**, so the
> sidecar image is tagged under the app's own repo as `:searxng` (not a new
> repo name). `fly auth docker` also needs a scratch `DOCKER_CONFIG` on some
> machines (its `pass` credential store fails uninitialized) — the helper
> script handles both.

```bash
cd deploy
docker build -f searxng-fly/Dockerfile -t registry.fly.io/cinder9630:searxng .
# fly auth docker (use the helper's scratch DOCKER_CONFIG if pass is broken)
docker push registry.fly.io/cinder9630:searxng
```

### 2. Create the SearXNG machine in the SAME app

```bash
fly machine run registry.fly.io/cinder9630:searxng \
  --app cinder9630 \
  --name searxng \
  --memory 512 \
  --region ams
```

Then set its process group. This flyctl version has no `--process-group` flag
on `machine run`/`update` — the group lives in
`config.metadata["fly_process_group"]`, set via the Machines API (the helper
script does this):

```bash
fly machine update searxng --app cinder9630 \
  --machine-config <(echo '{"config":{"metadata":{"fly_process_group":"searxng"}}}') -y
```

Notes:

- **No public port** — SearXNG is reached only over the private network, so
  it is never exposed publicly (no auth on the instance).
- **Process group `searxng`** keeps it out of the app's default group.
  Critically, the machine must **not** carry `fly_platform_version: v2`
  metadata — that would mark it Fly-Launch-managed and `fly deploy` would
  destroy it (its group is not in `fly.toml`). Without that key it is a
  *detached* machine that `fly deploy` leaves alone.
- `--memory 512` gives SearXNG headroom over its ~200–300 MB baseline.
- `auto_stop`/`auto_start` are **service-level** settings (they only apply to
  machines with public services). The sidecar has no public port, so it just
  stays running — which is what a search backend wants.

### 3. Point Cinder at it

```bash
fly secrets set --app cinder9630 SEARXNG_ENDPOINT=http://searxng.internal:8080
```

(`searxng.internal` is the machine's private-network name; `8080` is the
port SearXNG listens on in the container.)

### 4. Verify

```bash
# From inside the Cinder machine, SearXNG must answer on the private network:
fly ssh console --app cinder9630 -C "wget -qO- http://searxng.internal:8080/search?q=golang&format=json | head -c 120"

# Through Cinder:
curl -s -X POST https://cinder9630.fly.dev/v1/search \
  -H 'Content-Type: application/json' -d '{"query":"golang concurrency","limit":3}'
```

---

## Option B — Separate app

### Step 1 — Create the SearXNG app

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
    - json # <-- required: Cinder queries the JSON API
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
