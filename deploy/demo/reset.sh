#!/bin/sh
# Wipe the demo database and restart the app.
# Add to crontab for automatic resets:
#   0 0 * * * /opt/gopowerdns-admin/reset.sh >> /var/log/gopowerdns-reset.log 2>&1

set -e
cd "$(dirname "$0")"

echo "[$(date)] Resetting demo..."
podman compose down
rm -f data/go-pdns.db data/go-pdns.db-sessions.db
[ "$(id -u)" -ne 0 ] && podman unshare chown -R 0:0 data
podman compose up -d
echo "[$(date)] Reset complete."
