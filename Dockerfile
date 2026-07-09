# ─── Builder ─────────────────────────────────────────────────────────────────
FROM golang:1.26.4-alpine AS builder

ARG VERSION=dev

WORKDIR /build

# Download dependencies first so this layer is cached when only source changes.
COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux \
    go build \
    -ldflags="-s -w -X github.com/GoPowerDNS-Admin/GoPowerDNS-Admin/internal/version.version=${VERSION}" \
    -o go-pdns .

# ─── Runtime ─────────────────────────────────────────────────────────────────
FROM alpine:3

LABEL org.opencontainers.image.title="GoPowerDNS-Admin" \
      org.opencontainers.image.description="A modern, web-based administration UI for PowerDNS, written in Go. Manage forward and reverse zones and records, configure your PowerDNS server, and authenticate via local accounts, OIDC, or LDAP." \
      org.opencontainers.image.source="https://github.com/GoPowerDNS-Admin/GoPowerDNS-Admin" \
      org.opencontainers.image.url="https://github.com/GoPowerDNS-Admin/GoPowerDNS-Admin" \
      org.opencontainers.image.documentation="https://docs.gopowerdnsadmin.org" \
      org.opencontainers.image.licenses="MIT"

# ca-certificates: needed for ACME/Let's Encrypt and OIDC provider connections.
# tzdata: allows the container timezone to be set via TZ env var.
RUN apk add --no-cache ca-certificates tzdata \
    && addgroup -S gopdns && adduser -S -G gopdns gopdns

WORKDIR /app

COPY --from=builder /build/go-pdns /app/go-pdns

# /etc/go-pdns  — mount your main.toml here (required)
# /var/lib/go-pdns — persistent data: SQLite DB files, ACME certificate cache
RUN mkdir -p /etc/go-pdns /var/lib/go-pdns \
    && chown gopdns:gopdns /etc/go-pdns /var/lib/go-pdns

VOLUME ["/etc/go-pdns", "/var/lib/go-pdns"]

USER gopdns

EXPOSE 8080

ENTRYPOINT ["/app/go-pdns"]
CMD ["start", "-c", "/etc/go-pdns/"]
