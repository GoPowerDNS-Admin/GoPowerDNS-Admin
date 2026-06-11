---
title: Installation
weight: 1
next: /docs/getting-started/configuration
---

## Prerequisites

- **Go** — version specified in `go.mod`; no C toolchain required
- **PowerDNS** — an accessible PowerDNS authoritative server with the API enabled
- **Database** — MySQL/MariaDB, PostgreSQL, or SQLite

## Option A — Run from source

```bash
git clone https://github.com/GoPowerDNS-Admin/GoPowerDNS-Admin.git
cd GoPowerDNS-Admin
go run . start
```

## Option B — Build binary

```bash
make build
./gopowerdns-admin start
```

The version string, branch, and commit hash are baked in at build time:

```bash
./gopowerdns-admin --version
# gopowerdns-admin version v0.2.0-5-gabc1234 (main)
```

## Option C — Docker

```bash
docker pull ghcr.io/gopowerdns-admin/gopowerdns-admin:latest
docker run -d \
  --name gopowerdns-admin \
  -p 8080:8080 \
  -v /path/to/your/etc:/etc/go-pdns:ro \
  -v gopowerdns-data:/var/lib/go-pdns \
  ghcr.io/gopowerdns-admin/gopowerdns-admin:latest
```

Supported platforms: `linux/amd64`, `linux/arm64`, `linux/arm/v7`.

## Local PowerDNS with Docker Compose

A pre-configured Compose file is included for local development:

```bash
docker compose up -d
```

This starts PowerDNS with an in-memory backend and the API enabled on `http://localhost:8081`.

## Development mode

Templates and static assets are reloaded from the filesystem on each request — no rebuild needed:

```bash
go run . start --dev
```
