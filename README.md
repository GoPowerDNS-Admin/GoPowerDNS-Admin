# GoPowerDNS-Admin

[![Buy Me a Coffee](https://img.shields.io/badge/Buy%20Me%20a%20Coffee-ffdd00?style=flat&logo=buy-me-a-coffee&logoColor=black)](https://buymeacoffee.com/chrisamti)

GoPowerDNS-Admin is a web-based administration UI for PowerDNS. It helps you manage zones and DNS records, configure your PowerDNS server connection, and handle authentication with multiple providers — all from a modern web interface written in Go.

## Status: Alpha

Core features are functional. Interfaces and configuration may still change between releases. Bug reports and feedback are welcome — please open an issue.

## Key Features

- Zone and record management (create, edit, and manage DNS records)
- Forward and reverse zone creation with automatic CIDR-to-zone-name conversion for IPv4 and IPv6
- Duplicate zone detection with a direct link to the existing zone
- PowerDNS server settings stored in the application database
- Multiple authentication methods:
  - Local database (with optional TOTP two-factor authentication)
  - OpenID Connect (OIDC)
  - LDAP
- TOTP two-factor authentication for local accounts (per-user opt-in or admin-enforced)
- RBAC (Role-Based Access Control) with grouped permission cards and tri-state toggles
- Zone tag-based access control
- Activity/audit log with diff tracking, detail view, and undo support (record changes and zone deletes)
- Admin-configurable TTL presets shown as a dropdown in the record modal
- DNSSEC-managed records automatically hidden from the zone editor and dashboard
- Multiple database backends: MySQL/MariaDB, PostgreSQL, SQLite
- Responsive web UI built on AdminLTE 4 / Bootstrap 5 with Alpine.js for reactive components
- Go backend with server-rendered templates
- Nightly builds for Linux (amd64, arm64, armv7), macOS (amd64, arm64), and FreeBSD (amd64, aarch64)
- Health check endpoint (`GET /health`) for load balancer and container probes
- Native TLS (manual cert/key) and automatic TLS via Let's Encrypt / ACME
- Reverse proxy support (HAProxy, nginx, Traefik) with configurable trusted IP allowlist and proxy header
- Security headers on all responses: `X-Frame-Options`, `X-Content-Type-Options`, `Content-Security-Policy`, `Referrer-Policy`, and more

## Getting Started (Development)

Prerequisites:

- Go (see `go.mod` for the required version)
- Docker and Docker Compose (optional, for local PowerDNS)

1. Start a local PowerDNS for development (optional):

   If you want a local PDNS instance, you can use the provided compose setup:

   ```bash
   docker compose up -d
   ```

2. Configure the app:

   Review the sample configuration in `etc/main.toml` (or the copy under `internal/db/controller/setting/etc/main.toml`).
   For local testing, copy and adjust the sample into `etc/local/main.toml` — this directory is gitignored and any TOML files placed there override the settings in `etc/main.toml`:

   ```bash
   mkdir -p etc/local
   cp etc/main.toml etc/local/main.toml
   # edit etc/local/main.toml to match your local setup
   ```

   Adjust webserver, database, authentication, and record settings as needed.

3. Run the application:

   Default (embedded templates and static assets):

   ```bash
   go run . start
   ```

   Development mode (use templates from local filesystem, auto‑reload enabled):

   ```bash
   go run . start --dev
   ```

## Project Structure (high level)

- `internal/web` — HTTP handlers, templates, static assets
- `internal/db` — models and controllers (GORM)
- `internal/powerdns` — PowerDNS API integration
- `internal/activitylog` — activity log recording and diff helpers
- `internal/config` — configuration structs
- `etc/` — example configuration
- `docker/` — Docker/Compose helpers (including PDNS and LDAP setups)

## Database Support

GoPowerDNS-Admin supports three database backends:

| Backend       | Status    |
| ------------- | --------- |
| MySQL/MariaDB | Supported |
| PostgreSQL    | Supported |
| SQLite        | Supported |

Configure the backend in `etc/main.toml` under the `[database]` section.

## Authentication

Three authentication methods are supported and can be enabled independently:

- **Local DB** — username/password stored in the application database; supports optional TOTP (per-user or admin-enforced)
- **OIDC** — OpenID Connect (e.g. Keycloak, Authentik, Okta)
- **LDAP** — bind-based LDAP authentication

TOTP is only available for local accounts. OIDC and LDAP users rely on their identity provider for MFA.

See `internal/config/structs.go` for available configuration fields.

## Zone Editor

The zone editor provides a full-featured DNS record management interface:

- Reactive UI powered by Alpine.js — no page reloads for record add/edit/delete
- Unified record modal for all record types with type-specific fields (MX priority, TXT chunking, DS algorithm badge)
- TXT records: monospace textarea with automatic RFC-compliant 255-byte chunking and quote handling
- Live search and per-type filter pills to quickly find records
- Apex (`@`) records sorted first within each name group
- Inline zone metadata (kind, serial, DNSSEC status) in the record list header
- Collapsible zone settings card (SOA-EDIT-API, kind, masters)
- DNSSEC-managed records (RRSIG, NSEC, NSEC3, DNSKEY, CDS, CDNSKEY) are automatically hidden to prevent accidental edits
- Admin-configurable TTL presets available as a dropdown in the record modal (see TTL Presets below)

## TTL Presets

Admins can configure a list of TTL presets at `/admin/settings/ttl-presets`. Presets appear as a dropdown in the record modal so operators can pick a standard TTL without typing. A **Custom…** option is always available for manual override. Sensible defaults (1 min – 1 week) are seeded on first run.

## Zone Tag Access Control

Zone tags restrict which users and groups can see a zone. Zones without any tags are accessible to all authenticated users.

- Assign tags to zones at `/admin/zone-tag`; the list is searchable and paginated (configurable rows per page)
- Zones are sorted alphabetically
- Assign tags to users and groups to grant access to matching zones

## Roles & Permissions

The role editor groups permissions by resource with icon badges and tri-state toggles:

- Each resource group shows a count badge and an **All** toggle (checked / indeterminate / unchecked)
- **Select All** / **Deselect All** buttons apply across all groups at once

## Activity Log & Undo

All significant user actions (login, logout, zone create/update/delete, record changes) are recorded in the activity log, accessible at `/admin/activity`. Each entry stores a before/after diff. Admins can:

- Browse the paginated list with filters; long diffs are truncated with a fade and a link to the detail page
- Open a full detail page (`/admin/activity/:id`) showing the complete diff and an **Undo** button
- **Undo record changes** — restores a zone's RRsets to their previous state
- **Undo zone deletes** — recreates the zone including all RRsets from a full snapshot taken at delete time

## Health Check

`GET /health` is available without authentication and returns a JSON status object:

```json
{
  "status": "ok",
  "checks": {
    "alive": "ok",
    "database": "ok",
    "powerdns": "ok"
  }
}
```

| Field      | Values                  | Notes                                                                  |
| ---------- | ----------------------- | ---------------------------------------------------------------------- |
| `status`   | `ok` / `degraded`       | Overall health                                                         |
| `alive`    | `ok` / `shutting_down`  | Set to `shutting_down` during graceful shutdown                        |
| `database` | `ok` / `error`          | DB ping with 2 s timeout                                               |
| `powerdns` | `ok` / `not_configured` | PDNS client presence; `not_configured` does not degrade overall status |

Returns **200** when healthy, **503** when `alive=shutting_down` or the database is unreachable. Suitable for Kubernetes liveness/readiness probes and load balancer health checks.

## TLS / HTTPS

Three modes are supported; configure them in `etc/main.toml` (or an overlay file).

### Plain HTTP (default)

No extra config needed. Suitable when TLS is terminated by a reverse proxy.

### Manual certificate

Provide both fields together — they must be set as a pair:

```toml
[webserver]
TLSCertFile = "/etc/ssl/certs/server.crt"
TLSKeyFile  = "/etc/ssl/private/server.key"
```

The certificate is loaded once at startup. To pick up a renewed certificate, restart the server.

### Let's Encrypt / ACME (automatic)

Obtains and renews a certificate automatically. The server must be reachable on port 80 for the HTTP-01 challenge:

```toml
[webserver]
ACMEEnabled  = true
ACMEDomain   = "pdns.example.com"
ACMEEmail    = "admin@example.com"
ACMECacheDir = "/var/lib/go-pdns/acme-cache"
```

Renewed certificates are picked up on the next TLS handshake — no restart required. ACME and manual `TLSCertFile`/`TLSKeyFile` are mutually exclusive.

## Reverse Proxy

When running behind HAProxy, nginx, Traefik, or any other reverse proxy, enable the `[webserver.reverseproxy]` block so that `c.IP()` returns the real client IP instead of the proxy's address. This affects activity log entries and any IP-based logic.

```toml
[webserver.reverseproxy]
enabled     = true
trustedips  = ["127.0.0.1", "10.0.0.0/8", "172.16.0.0/12", "192.168.0.0/16"]
proxyheader = "X-Forwarded-For"   # or "X-Real-IP" for nginx
```

| Field         | Default           | Description                                                                                  |
| ------------- | ----------------- | -------------------------------------------------------------------------------------------- |
| `enabled`     | `false`           | Activates trusted-proxy IP checking. When `false`, `proxyheader` is trusted unconditionally. |
| `trustedips`  | _(none)_          | Required when `enabled = true`. IPv4/IPv6 addresses or CIDR ranges of your upstream proxies. |
| `proxyheader` | `X-Forwarded-For` | HTTP header used to read the real client IP.                                                 |

> **Note:** `trustedips` must not be empty when `enabled = true`; the application will refuse to start otherwise.

## Docker

A multi-stage `Dockerfile` is included. The final image is based on Alpine, contains only the compiled binary, and runs as a non-root user (`gopdns`).

| Path                | Purpose                                                      |
| ------------------- | ------------------------------------------------------------ |
| `/etc/go-pdns/`     | Configuration — mount your `main.toml` here (read-only)      |
| `/var/lib/go-pdns/` | Persistent data — SQLite DB files and ACME certificate cache |

### Build

```bash
make docker-build
# or with a custom tag:
make docker-build IMAGE_NAME=ghcr.io/myorg/gopowerdns-admin IMAGE_TAG=v1.0.0
```

### Run

```bash
make docker-run
# or manually:
docker run -d \
  --name gopowerdns-admin \
  -p 8080:8080 \
  -v /path/to/your/etc:/etc/go-pdns:ro \
  -v gopowerdns-data:/var/lib/go-pdns \
  gopowerdns-admin
```

### Push

```bash
make docker-push IMAGE_NAME=ghcr.io/myorg/gopowerdns-admin IMAGE_TAG=v1.0.0
```

### Tips

- **SQLite:** set `Name = "/var/lib/go-pdns/go-pdns.db"` in `main.toml` so the database lands in the persistent volume.
- **ACME:** set `ACMECacheDir = "/var/lib/go-pdns/acme-cache"` for the same reason.
- **Port:** defaults to `8080`; override with `GPDNS_WEBSERVER_PORT` (see below).

### Environment variable overrides

All config keys can be overridden via environment variables prefixed with `GPDNS_`. The key maps directly to the TOML path with `_` as the separator:

```bash
docker run -d \
  -e GPDNS_WEBSERVER_PORT=9090 \
  -e GPDNS_DB_GORMENGINE=sqlite \
  -e GPDNS_DB_NAME=/var/lib/go-pdns/go-pdns.db \
  ...
```

## Background & Inspiration

The idea for this Go-based version came from the PowerDNS-Admin project (https://github.com/PowerDNS-Admin/PowerDNS-Admin), which is not further developed. This repository provides a Go implementation that follows similar goals while evolving independently.

## Configuration Highlights

- PDNS server settings are stored in the database (key `pdns_server`). See `internal/db/controller/pdnsserver/settings.go`.
- Authentication supports Local DB, OIDC, and LDAP; see `internal/config/structs.go` for the available fields.

## Contributing

Contributions are welcome! Given the rapid pace of changes, please open an issue or draft PR early to align on direction.

## License

This project is licensed under the terms specified in the `LICENSE` file.
