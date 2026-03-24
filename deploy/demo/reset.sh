#!/bin/sh
# Wipe the demo database and restart the app.
# Add to crontab for automatic resets:
#   0 0 * * * /opt/gopowerdns-admin/reset.sh >> /var/log/gopowerdns-reset.log 2>&1

set -e
cd "$(dirname "$0")"

echo "[$(date)] Resetting demo..."
podman compose stop app
rm -f data/go-pdns.db data/go-pdns.db-sessions.db
podman compose start app
echo "[$(date)] Reset complete."
