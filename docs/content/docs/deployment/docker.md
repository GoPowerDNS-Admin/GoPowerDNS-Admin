---
title: Docker
description: "Run GoPowerDNS-Admin with Docker — published images, volumes, environment variables, and a Compose stack with PowerDNS."
weight: 1
next: /docs/deployment/tls
---

## Published image

Release images are published to the GitHub Container Registry on every tagged release:

```bash
docker pull ghcr.io/gopowerdns-admin/gopowerdns-admin:latest
# or pin to a specific release:
docker pull ghcr.io/gopowerdns-admin/gopowerdns-admin:v0.2.0
```

Supported platforms: `linux/amd64`, `linux/arm64`, `linux/arm/v7`.

## Volumes

| Path                | Purpose                                                            |
| ------------------- | ------------------------------------------------------------------ |
| `/etc/go-pdns/`     | Configuration — mount your `main.toml` here (read-only)            |
| `/var/lib/go-pdns/` | Persistent data — SQLite database files and ACME certificate cache |

## Run

```bash
docker run -d \
  --name gopowerdns-admin \
  -p 8080:8080 \
  -v /path/to/your/etc:/etc/go-pdns:ro \
  -v gopowerdns-data:/var/lib/go-pdns \
  ghcr.io/gopowerdns-admin/gopowerdns-admin:latest
```

## Docker Compose (with local PowerDNS)

```yaml
services:
  gopowerdns-admin:
    image: ghcr.io/gopowerdns-admin/gopowerdns-admin:latest
    ports:
      - "8080:8080"
    volumes:
      - ./etc:/etc/go-pdns:ro
      - gopowerdns-data:/var/lib/go-pdns
    environment:
      GPDNS_PDNS_APISERVERURL: "http://powerdns:8081"
      GPDNS_PDNS_APIKEY: "changeme"
      GPDNS_PDNS_VHOST: "localhost"
    depends_on:
      - powerdns

  powerdns:
    image: powerdns/pdns-auth-49:4.9.6
    ports:
      - "5300:53/udp"
      - "5300:53/tcp"
    command:
      - "--api=yes"
      - "--api-key=changeme"
      - "--webserver=yes"
      - "--webserver-address=0.0.0.0"
      - "--webserver-port=8081"
      - "--webserver-allow-from=0.0.0.0/0"

volumes:
  gopowerdns-data:
```

The `GPDNS_PDNS_*` environment variables bootstrap the PowerDNS connection on
first startup; `APIKEY` must match the `--api-key` you pass to PowerDNS.

## Build locally

```bash
make docker-build
# with custom tag:
make docker-build IMAGE_NAME=ghcr.io/myorg/gopowerdns-admin IMAGE_TAG=v1.0.0
```

## Tips

- **SQLite**: set `Name = "/var/lib/go-pdns/go-pdns.db"` so the database lands in the persistent volume.
- **ACME**: set `ACMECacheDir = "/var/lib/go-pdns/acme-cache"` for the same reason.
