# GoPowerDNS-Admin

GoPowerDNS-Admin is a web-based administration UI for PowerDNS. It helps you manage zones and DNS records, configure your PowerDNS server connection, and handle authentication with multiple providers — all from a modern web interface written in Go.

## Status: Heavy Development

This project is under active, heavy development. Interfaces and configuration may change without notice. Expect incomplete features, rough edges, and potential breaking changes between commits. Use in production at your own risk.

## Key Features

- Zone and record management (create, edit, and manage DNS records)
- PowerDNS server settings stored in the application database
- Multiple authentication methods:
  - Local database
  - OpenID Connect (OIDC)
  - LDAP
- Go backend with server-rendered templates and a small amount of client-side JS

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
   Adjust webserver, authentication, and record settings as needed.

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
- `internal/config` — configuration structs
- `etc/` — example configuration
- `docker/` — Docker/Compose helpers (including a PDNS setup)

## Configuration Highlights

- PDNS server settings are stored in the database (key `pdns_server`). See `internal/db/controller/pdnsserver/settings.go`.
- Authentication supports Local DB, OIDC, and LDAP; see `internal/config/structs.go` for the available fields.

## Contributing

Contributions are welcome! Given the rapid pace of changes, please open an issue or draft PR early to align on direction.

## License

This project is licensed under the terms specified in the `LICENSE` file.
