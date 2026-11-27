#!/bin/bash
# Wipe the demo database and restart the app.
# Add to crontab for automatic resets:
#   0 0 * * * /opt/gopowerdns-admin/reset.sh >> /var/log/gopowerdns-reset.log 2>&1

set -e
cd "$(dirname "$0")"

run_step() {
  local step="$1"
  shift
  echo "[$(date)] ${step}..."
  if ! "$@"; then
    echo "[$(date)] ERROR: ${step} failed." >&2
    exit 1
  fi
}

echo "[$(date)] Resetting demo..."
run_step "Stopping and removing demo containers/volumes" podman compose down -v
# Pull the newest stable image (compose pins the floating `latest` tag) so the
# nightly reset also upgrades the demo to the most recent release.
run_step "Pulling latest demo image" podman compose pull
run_step "Starting demo stack" podman compose up -d
IMAGE_DIGEST=$(podman image inspect --format '{{ index .RepoDigests 0 }}' ghcr.io/gopowerdns-admin/gopowerdns-admin:latest 2>/dev/null || echo unknown)
echo "[$(date)] Reset complete (running ${IMAGE_DIGEST})."
