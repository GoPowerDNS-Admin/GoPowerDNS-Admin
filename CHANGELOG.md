# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).

## [Unreleased]

## [0.1.0-alpha.1] - 2026-03-22

### Added

- Zone and record management — create, edit, and delete DNS records across forward and reverse zones
- Automatic CIDR-to-zone-name conversion for IPv4 and IPv6 reverse zones
- Duplicate zone detection with a direct link to the existing zone
- Multiple database backends: MySQL/MariaDB, PostgreSQL, SQLite
- Multiple authentication methods: local database, OpenID Connect (OIDC), LDAP
- TOTP two-factor authentication for local accounts (per-user opt-in or admin-enforced)
- RBAC with grouped permission cards and tri-state toggles
- Zone tag-based access control — restrict zone visibility to specific users and groups
- Activity/audit log with before/after diffs, detail view, and undo support for record changes and zone deletes
- Admin-configurable TTL presets shown as a dropdown in the record modal
- DNSSEC-managed records automatically hidden from the zone editor
- Health check endpoint (`GET /health`) for load balancer and container probes
- Native TLS support (manual certificate/key pair)
- Automatic TLS via Let's Encrypt / ACME (HTTP-01 challenge)
- Reverse proxy support (HAProxy, nginx, Traefik) with configurable trusted IP allowlist and proxy header
- Security headers on all responses: `X-Frame-Options`, `X-Content-Type-Options`, `Content-Security-Policy`, `Referrer-Policy`, and more
- Docker image with multi-stage build and non-root user (`gopdns`)
- Nightly builds for Linux (amd64, arm64, armv7), macOS (amd64, arm64), and FreeBSD (amd64, aarch64)
- Graceful shutdown with configurable drain period for load balancer deregistration
- Environment variable overrides for all config keys (`GPDNS_` prefix)
