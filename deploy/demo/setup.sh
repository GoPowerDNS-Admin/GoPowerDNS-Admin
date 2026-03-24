#!/bin/bash
# One-time setup: install Podman, download demo files, prompt for config, and start.
# Supports both direct execution and piped via curl:
#   bash <(curl -fsSL https://raw.githubusercontent.com/GoPowerDNS-Admin/GoPowerDNS-Admin/main/deploy/demo/setup.sh)
#
# Installs to $HOME/gopowerdns-admin by default. Override with:
#   GOPOWERDNS_INSTALL_DIR=/custom/path bash <(curl -fsSL ...)

set -e

INSTALL_DIR="${GOPOWERDNS_INSTALL_DIR:-$HOME/gopowerdns-admin}"
RAW_BASE=https://raw.githubusercontent.com/GoPowerDNS-Admin/GoPowerDNS-Admin/main/deploy/demo

echo "==> Installing Podman and podman-compose..."
SUDO=""
[ "$(id -u)" -ne 0 ] && SUDO="sudo"
if command -v dnf &>/dev/null; then
  $SUDO dnf install -y podman podman-compose cronie
elif command -v apt-get &>/dev/null; then
  $SUDO apt-get update && $SUDO apt-get install -y podman podman-compose cron
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
PDNS_KEY=$(openssl rand -hex 32)

cat > "$INSTALL_DIR/.env" <<EOF
DOMAIN=${DOMAIN}
COOKIE_KEY=${COOKIE_KEY}
ARGON2_SALT=${ARGON2_SALT}
PDNS_KEY=${PDNS_KEY}
EOF

echo ""
echo "==> Starting the app..."
cd "$INSTALL_DIR" && podman compose up -d

echo ""
echo "==> Enabling auto-start on reboot..."
if [ "$(id -u)" -eq 0 ]; then
  # Running as root — install system-wide systemd service
  SYSTEMD_DIR=/etc/systemd/system
  podman generate systemd --new --name --restart-policy=always -t 10 app pdns 2>/dev/null \
    | tee "$SYSTEMD_DIR/gopowerdns-demo.service" > /dev/null || \
  cat > "$SYSTEMD_DIR/gopowerdns-demo.service" <<UNIT
[Unit]
Description=GoPowerDNS-Admin Demo
After=network-online.target
Wants=network-online.target

[Service]
Type=oneshot
RemainAfterExit=yes
WorkingDirectory=$INSTALL_DIR
ExecStart=/usr/bin/podman compose up -d
ExecStop=/usr/bin/podman compose down
TimeoutStartSec=120

[Install]
WantedBy=multi-user.target
UNIT
  systemctl daemon-reload
  systemctl enable --now gopowerdns-demo.service
else
  # Rootless — install per-user systemd service
  SYSTEMD_DIR="$HOME/.config/systemd/user"
  mkdir -p "$SYSTEMD_DIR"
  cat > "$SYSTEMD_DIR/gopowerdns-demo.service" <<UNIT
[Unit]
Description=GoPowerDNS-Admin Demo
After=network-online.target
Wants=network-online.target

[Service]
Type=oneshot
RemainAfterExit=yes
WorkingDirectory=$INSTALL_DIR
ExecStart=$(command -v podman) compose up -d
ExecStop=$(command -v podman) compose down
TimeoutStartSec=120

[Install]
WantedBy=default.target
UNIT
  systemctl --user daemon-reload
  systemctl --user enable --now gopowerdns-demo.service
  # Enable linger so the service starts without a user session
  loginctl enable-linger "$(id -un)"
fi

echo ""
echo "==> Enabling daily demo reset at midnight UTC..."
(crontab -l 2>/dev/null; echo "0 0 * * * $INSTALL_DIR/reset.sh >> $INSTALL_DIR/reset.log 2>&1") | crontab -

echo ""
echo "==> Done! App is running on http://localhost:8080"
echo "    Public URL: https://${DOMAIN} (TLS handled by HAProxy)"
