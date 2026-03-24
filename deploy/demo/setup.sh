#!/bin/bash
# One-time setup: install Podman, create directories, configure cron reset.
# Run as root on the QEMU VM (Fedora/RHEL/Debian/Ubuntu).

set -e

INSTALL_DIR=/opt/gopowerdns-admin

echo "==> Installing Podman and podman-compose..."
if command -v dnf &>/dev/null; then
  dnf install -y podman podman-compose
elif command -v apt-get &>/dev/null; then
  apt-get update && apt-get install -y podman podman-compose
else
  echo "Unsupported package manager — install Podman manually." >&2
  exit 1
fi

echo "==> Creating directories..."
mkdir -p "$INSTALL_DIR/config" "$INSTALL_DIR/data"

echo "==> Copying files..."
cp -r "$(dirname "$0")"/* "$INSTALL_DIR/"
cp "$INSTALL_DIR/.env.example" "$INSTALL_DIR/.env"
chmod +x "$INSTALL_DIR/reset.sh"

echo ""
echo "==> Edit $INSTALL_DIR/.env before starting:"
echo "    - Set DOMAIN to the public hostname (HAProxy handles TLS)"
echo "    - Generate secrets:"
echo "        COOKIE_KEY=\$(openssl rand -base64 32)"
echo "        ARGON2_SALT=\$(openssl rand -base64 24)"
echo ""
echo "==> Then start the app:"
echo "    cd $INSTALL_DIR && podman compose up -d"
echo ""
echo "==> To enable daily demo reset at midnight UTC:"
echo "    (crontab -l 2>/dev/null; echo '0 0 * * * $INSTALL_DIR/reset.sh >> /var/log/gopowerdns-reset.log 2>&1') | crontab -"
