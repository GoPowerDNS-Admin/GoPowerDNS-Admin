#!/bin/sh
# Wipe the demo database and restart the app.
# Add to crontab for automatic resets:
#   0 0 * * * /opt/gopowerdns-admin/reset.sh >> /var/log/gopowerdns-reset.log 2>&1

set -e
cd "$(dirname "$0")"

echo "[$(date)] Resetting demo..."
podman compose down -v
# Pull the newest stable image (compose pins the floating `latest` tag) so the
# nightly reset also upgrades the demo to the most recent release.
podman compose pull
podman compose up -d
echo "[$(date)] Reset complete (running $(podman image inspect --format '{{ index .RepoDigests 0 }}' ghcr.io/gopowerdns-admin/gopowerdns-admin:latest 2>/dev/null || echo unknown))."
