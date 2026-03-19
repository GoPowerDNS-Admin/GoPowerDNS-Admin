# GoPowerDNS-Admin

[![Buy Me a Coffee](https://img.shields.io/badge/Buy%20Me%20a%20Coffee-ffdd00?style=flat&logo=buy-me-a-coffee&logoColor=black)](https://buymeacoffee.com/chrisamti)

GoPowerDNS-Admin is a web-based administration UI for PowerDNS. It helps you manage zones and DNS records, configure your PowerDNS server connection, and handle authentication with multiple providers — all from a modern web interface written in Go.

## Status: Heavy Development

This project is under active, heavy development. Interfaces and configuration may change without notice. Expect incomplete features, rough edges, and potential breaking changes between commits. Use in production at your own risk.

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
- RBAC (Role-Based Access Control) with role and permission CRUD
- Zone tag-based access control
- Activity/audit log with diff tracking and undo support (record changes and zone deletes)
- DNSSEC-managed records automatically hidden from the zone editor and dashboard
- Multiple database backends: MySQL/MariaDB, PostgreSQL, SQLite
- Responsive web UI built on AdminLTE 4 / Bootstrap 5
- Go backend with server-rendered templates and a small amount of client-side JS
- Nightly builds for Linux (amd64, arm64, armv7), macOS (amd64, arm64), and FreeBSD (amd64, aarch64)

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

- Live search and per-type filter pills to quickly find records
- Inline zone metadata (kind, serial, DNSSEC status) in the record list header
- Collapsible zone settings card (SOA-EDIT-API, kind, masters)
- DNSSEC-managed records (RRSIG, NSEC, NSEC3, DNSKEY, CDS, CDNSKEY) are automatically hidden to prevent accidental edits

## Activity Log & Undo

All significant user actions (login, logout, zone create/update/delete, record changes) are recorded in the activity log, accessible at `/admin/activity`. Each entry stores a before/after diff. Admins can undo:

- **Record changes** — restores a zone's RRsets to their previous state
- **Zone deletes** — recreates the zone including all RRsets from a full snapshot taken at delete time

## Background & Inspiration

The idea for this Go-based version came from the PowerDNS-Admin project (https://github.com/PowerDNS-Admin/PowerDNS-Admin), which is not further developed. This repository provides a Go implementation that follows similar goals while evolving independently.

## Configuration Highlights

- PDNS server settings are stored in the database (key `pdns_server`). See `internal/db/controller/pdnsserver/settings.go`.
- Authentication supports Local DB, OIDC, and LDAP; see `internal/config/structs.go` for the available fields.

## Contributing

Contributions are welcome! Given the rapid pace of changes, please open an issue or draft PR early to align on direction.

## License

This project is licensed under the terms specified in the `LICENSE` file.
