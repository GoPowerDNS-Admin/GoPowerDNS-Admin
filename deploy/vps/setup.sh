#!/bin/bash
# One-time server setup: install Docker, create directories, configure cron reset.
# Run as root on a fresh Ubuntu/Debian VM.

set -e

INSTALL_DIR=/opt/gopowerdns-admin

echo "==> Installing Docker..."
curl -fsSL https://get.docker.com | sh

echo "==> Creating directories..."
mkdir -p "$INSTALL_DIR/config" "$INSTALL_DIR/data"

echo "==> Copying files..."
cp -r "$(dirname "$0")"/* "$INSTALL_DIR/"
cp "$INSTALL_DIR/.env.example" "$INSTALL_DIR/.env"
chmod +x "$INSTALL_DIR/reset.sh"

echo ""
echo "==> Edit $INSTALL_DIR/.env before starting:"
echo "    - Set DOMAIN, GPDNS_WEBSERVER_ACMEDOMAIN, GPDNS_WEBSERVER_ACMEEMAIL"
echo "    - Generate secrets:"
echo "        COOKIE_KEY=\$(openssl rand -base64 32)"
echo "        ARGON2_SALT=\$(openssl rand -base64 24)"
echo ""
echo "==> Then start the app:"
echo "    cd $INSTALL_DIR && docker compose up -d"
echo ""
echo "==> To enable daily demo reset at midnight UTC:"
echo "    (crontab -l 2>/dev/null; echo '0 0 * * * $INSTALL_DIR/reset.sh >> /var/log/gopowerdns-reset.log 2>&1') | crontab -"
