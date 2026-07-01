---
title: Installation
description: "Install GoPowerDNS-Admin using the Docker image, a prebuilt binary, or from source — prerequisites and supported platforms."
weight: 1
next: /docs/getting-started/configuration
---

## Prerequisites

- **PowerDNS** — an accessible PowerDNS authoritative server with the API enabled
- **Database** — SQLite (built in, nothing to install), MySQL/MariaDB, or PostgreSQL

That's it for running GoPowerDNS-Admin itself — the Docker image and release
binaries are self-contained. A Go toolchain is only needed if you build from
source (see [From source](#from-source-development)).

## Option A — Docker (recommended)

Release images are published to the GitHub Container Registry:

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

Mount your `main.toml` at `/etc/go-pdns/` and keep the SQLite database on the
`/var/lib/go-pdns` volume. For a full Compose stack that also runs PowerDNS, and
details on the volumes, see [Deployment → Docker](/docs/deployment/docker).

## Option B — Prebuilt binary

Every [release](https://github.com/GoPowerDNS-Admin/GoPowerDNS-Admin/releases)
ships static binaries for Linux, macOS, and FreeBSD (amd64/arm64/armv7). Download
the one for your platform, make it executable, and run it:

```bash
# example: Linux amd64 — adjust the asset name for your platform
curl -LO https://github.com/GoPowerDNS-Admin/GoPowerDNS-Admin/releases/latest/download/gopowerdns-admin-linux-amd64
chmod +x gopowerdns-admin-linux-amd64
./gopowerdns-admin-linux-amd64 start
```

Check the version at any time:

```bash
./gopowerdns-admin-linux-amd64 --version
```

## From source (development)

Building from source needs the **Go** version pinned in `go.mod` (no C toolchain
required). This path is aimed at contributors and local development.

```bash
git clone https://github.com/GoPowerDNS-Admin/GoPowerDNS-Admin.git
cd GoPowerDNS-Admin

go run . start        # run directly
# or build a binary:
make build
./gopowerdns-admin start
```

The version string, branch, and commit hash are baked in at build time:

```bash
./gopowerdns-admin --version
# gopowerdns-admin version v0.3.3-5-gabc1234 (main)
```

### Development mode

Templates and static assets are reloaded from the filesystem on each request —
no rebuild needed:

```bash
go run . start --dev
```

### Local PowerDNS for development

A pre-configured Compose file is included to spin up a PowerDNS instance to
develop against:

```bash
docker compose up -d
```

This starts PowerDNS with the API enabled on `http://localhost:8081`.
