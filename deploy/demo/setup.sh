#!/bin/bash
# One-time setup: install Podman, download demo files, prompt for config, and start.
# Supports both direct execution and piped via curl:
#   bash <(curl -fsSL https://raw.githubusercontent.com/GoPowerDNS-Admin/GoPowerDNS-Admin/main/deploy/demo/setup.sh)

set -e

INSTALL_DIR=/opt/gopowerdns-admin
RAW_BASE=https://raw.githubusercontent.com/GoPowerDNS-Admin/GoPowerDNS-Admin/main/deploy/demo

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

echo "==> Downloading demo files..."
curl -fsSL "$RAW_BASE/compose.yml"        -o "$INSTALL_DIR/compose.yml"
curl -fsSL "$RAW_BASE/reset.sh"           -o "$INSTALL_DIR/reset.sh"
curl -fsSL "$RAW_BASE/.env.example"       -o "$INSTALL_DIR/.env.example"
curl -fsSL "$RAW_BASE/config/main.toml"   -o "$INSTALL_DIR/config/main.toml"
chmod +x "$INSTALL_DIR/reset.sh"

echo ""
echo "==> Configuration"

read -rp "    Public hostname (e.g. gopowerdns-admin.duckdns.org): " DOMAIN
DOMAIN="${DOMAIN:-gopowerdns-admin.duckdns.org}"

COOKIE_KEY=$(openssl rand -base64 32)
ARGON2_SALT=$(openssl rand -base64 24)

read -rp "    PowerDNS API URL (leave empty to configure via UI): " PDNS_URL
read -rp "    PowerDNS API key (leave empty to configure via UI): " PDNS_KEY
read -rp "    PowerDNS vhost [localhost]: " PDNS_VHOST
PDNS_VHOST="${PDNS_VHOST:-localhost}"

cat > "$INSTALL_DIR/.env" <<EOF
DOMAIN=${DOMAIN}
COOKIE_KEY=${COOKIE_KEY}
ARGON2_SALT=${ARGON2_SALT}
PDNS_URL=${PDNS_URL}
PDNS_KEY=${PDNS_KEY}
PDNS_VHOST=${PDNS_VHOST}
EOF

echo ""
echo "==> Starting the app..."
cd "$INSTALL_DIR" && podman compose up -d

echo ""
echo "==> Enabling daily demo reset at midnight UTC..."
(crontab -l 2>/dev/null; echo "0 0 * * * $INSTALL_DIR/reset.sh >> /var/log/gopowerdns-reset.log 2>&1") | crontab -

echo ""
echo "==> Done! App is running on http://localhost:8080"
echo "    Public URL: https://${DOMAIN} (TLS handled by HAProxy)"
