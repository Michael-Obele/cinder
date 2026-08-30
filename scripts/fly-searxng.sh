#!/usr/bin/env bash
# fly-searxng.sh — deploy SearXNG as a sidecar machine inside the SAME Fly app
# as Cinder (Option A). Idempotent: safe to re-run — it recreates the machine
# if it was wiped, and just updates it if it already exists.
#
# Usage:
#   ./scripts/fly-searxng.sh          # full deploy (build → push → machine → secret)
#   ./scripts/fly-searxng.sh status   # show the searxng machine + secret state
#   ./scripts/fly-searxng.sh destroy  # remove the searxng machine (keeps secret)
#
# Env overrides:
#   FLY_APP      (default: cinder9630)
#   FLY_REGION   (default: ams — must match the app's primary_region)
#   FLY_IMAGE    (default: registry.fly.io/cinder9630:searxng — Fly's registry
#                 only accepts repos named after an app, so the sidecar image
#                 lives as a tag on the app's own repo)
#   SEARXNG_PORT (default: 8080 — container port SearXNG listens on)
set -euo pipefail

APP="${FLY_APP:-cinder9630}"
REGION="${FLY_REGION:-ams}"
IMAGE="${FLY_IMAGE:-registry.fly.io/${APP}:searxng}"
PORT="${SEARXNG_PORT:-8080}"
ENDPOINT="http://searxng.internal:${PORT}"
REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

require_fly() { command -v fly >/dev/null || { echo "error: fly CLI not found" >&2; exit 1; }; }
require_docker() { command -v docker >/dev/null || { echo "error: docker not found" >&2; exit 1; }; }

# fly auth docker writes credentials via docker's credsStore, which on some
# machines is `pass` (not initialized) and fails. Using a scratch DOCKER_CONFIG
# without a credsStore avoids that. Also: the image must be tagged under the
# app's own repo (registry.fly.io/$APP) — Fly's registry rejects other names.
fly_docker_auth() {
  local cfg="${DOCKER_FLY_CONFIG:-/tmp/docker-fly-config}"
  rm -rf "$cfg" && cp -r ~/.docker "$cfg"
  python3 - "$cfg" <<'PY'
import json, sys
p = sys.argv[1] + '/config.json'
d = json.load(open(p))
d.pop('credsStore', None)
d['auths'] = {}
json.dump(d, open(p, 'w'))
PY
  DOCKER_CONFIG="$cfg" fly auth docker
}

machine_exists() {
  fly machine list --app "$APP" --json 2>/dev/null \
    | python3 -c 'import json,sys; print(any(m.get("name")=="searxng" for m in json.load(sys.stdin)))'
}

build_and_push() {
  echo "==> Building sidecar image (JSON API enabled)..."
  ( cd "$REPO_ROOT/deploy" && docker build -f searxng-fly/Dockerfile -t "$IMAGE" . )
  echo "==> Pushing to Fly registry..."
  fly_docker_auth
  DOCKER_CONFIG="${DOCKER_FLY_CONFIG:-/tmp/docker-fly-config}" docker push "$IMAGE"
}

create_machine() {
  if [[ "$(machine_exists)" == "True" ]]; then
    echo "==> searxng machine already exists — updating image..."
    fly machine update searxng --app "$APP" --image "$IMAGE" -y
  else
    echo "==> Creating searxng machine in app '$APP' (no public port)..."
    fly machine run "$IMAGE" \
      --app "$APP" \
      --name searxng \
      --memory 512 \
      --region "$REGION"
  fi
  # The process group lives in config.metadata["fly_process_group"] (not a
  # top-level field). Critically we do NOT set fly_platform_version — that
  # would mark the machine as Fly-Launch-managed and `fly deploy` would try
  # to destroy it (group 'searxng' is not in fly.toml). A machine without
  # fly_platform_version is "detached" and left alone by `fly deploy`.
  local mid
  mid=$(fly machine list --app "$APP" --json 2>/dev/null \
    | python3 -c 'import json,sys
print([m["id"] for m in json.load(sys.stdin) if m.get("name")=="searxng"][0])')
  local cfg="${DOCKER_FLY_CONFIG:-/tmp/docker-fly-config}"
  cat > /tmp/searxng-machine.json <<EOF
{
  "config": {
    "init": {},
    "guest": {"cpu_kind": "shared", "cpus": 1, "memory_mb": 512},
    "image": "$IMAGE",
    "restart": {"policy": "on-failure", "max_retries": 10},
    "dns": {},
    "metadata": {"fly_process_group": "searxng"}
  }
}
EOF
  local tok
  tok=$(fly auth token 2>/dev/null)
  curl -s -X POST -H "Authorization: Bearer $tok" \
    -H "Content-Type: application/json" \
    -d @/tmp/searxng-machine.json \
    "https://api.machines.dev/v1/apps/$APP/machines/$mid" >/dev/null
  echo "==> Process group 'searxng' set (detached from fly deploy)."
}

set_secret() {
  echo "==> Pointing Cinder at the sidecar ($ENDPOINT)..."
  fly secrets set --app "$APP" "SEARXNG_ENDPOINT=$ENDPOINT"
}

verify() {
  echo "==> Verifying from inside the Cinder machine..."
  fly ssh console --app "$APP" -C \
    "wget -qO- '${ENDPOINT}/search?q=golang&format=json' | head -c 120"
  echo
  echo "==> Secret:"
  fly secrets list --app "$APP" | grep -E 'NAME|SEARXNG' || true
}

status() {
  echo "==> searxng machine:"
  fly machine list --app "$APP" --json \
    | python3 -c 'import json,sys
for m in json.load(sys.stdin):
    if m.get("name")=="searxng":
        img = m.get("image_ref", {})
        ref = img.get("registry","") + "/" + img.get("repository","") + ":" + (img.get("tag") or "?")
        cfg = m.get("config", {})
        print("  name:", m["name"], "| state:", m["state"], "| region:", m.get("region"))
        print("  image:", ref)
        print("  process_group:", cfg.get("metadata", {}).get("fly_process_group") or "(none)")
        print("  fly_platform_version:", cfg.get("metadata", {}).get("fly_platform_version") or "(none)")'
  echo "==> SEARXNG_ENDPOINT secret:"
  fly secrets list --app "$APP" | grep -E 'NAME|SEARXNG' || echo "  (not set — Cinder falls back to Brave)"
}

destroy() {
  if [[ "$(machine_exists)" == "True" ]]; then
    echo "==> Destroying searxng machine (secret kept)..."
    fly machine destroy searxng --app "$APP" --force
  else
    echo "==> No searxng machine to destroy."
  fi
}

require_fly
case "${1:-deploy}" in
  deploy)  require_docker; build_and_push; create_machine; set_secret; verify ;;
  status)  status ;;
  destroy) destroy ;;
  *) echo "usage: $0 [deploy|status|destroy]" >&2; exit 1 ;;
esac