#!/bin/bash
# Remove all demo containers, volumes, images, and install directory.

set -e
cd "$(dirname "$0")"

INSTALL_DIR="${GOPOWERDNS_INSTALL_DIR:-$HOME/gopowerdns-admin}"

echo "==> Stopping and removing containers and volumes..."
cd "$INSTALL_DIR" && podman compose down -v --rmi all 2>/dev/null || true

echo "==> Removing install directory..."
rm -rf "$INSTALL_DIR"

echo "==> Removing systemd service..."
if [ "$(id -u)" -eq 0 ]; then
  systemctl disable --now gopowerdns-demo.service 2>/dev/null || true
  rm -f /etc/systemd/system/gopowerdns-demo.service
  systemctl daemon-reload
else
  systemctl --user disable --now gopowerdns-demo.service 2>/dev/null || true
  rm -f "$HOME/.config/systemd/user/gopowerdns-demo.service"
  systemctl --user daemon-reload
  loginctl disable-linger "$(id -un)" 2>/dev/null || true
fi

echo "==> Removing cron entry..."
crontab -l 2>/dev/null | grep -v "gopowerdns-admin/reset.sh" | crontab - 2>/dev/null || true

echo "==> Done."
