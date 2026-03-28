---
title: TLS / HTTPS
weight: 2
prev: /docs/deployment/docker
next: /docs/deployment/reverse-proxy
---

Three TLS modes are supported. Configure them in the `[webserver]` section of `main.toml`.

## Plain HTTP (default)

No TLS configuration needed. Suitable when TLS is terminated by a reverse proxy.

## Manual certificate

```toml
[webserver]
TLSCertFile = "/etc/ssl/certs/server.crt"
TLSKeyFile  = "/etc/ssl/private/server.key"
```

Both fields must be set together. The certificate is loaded once at startup — restart to pick up a renewed certificate.

## Let's Encrypt / ACME

Automatically obtains and renews a certificate. The server must be reachable on port 80 for the HTTP-01 challenge.

```toml
[webserver]
ACMEEnabled  = true
ACMEDomain   = "pdns.example.com"
ACMEEmail    = "admin@example.com"
ACMECacheDir = "/var/lib/go-pdns/acme-cache"
```

Renewed certificates are picked up on the next TLS handshake — no restart required.

{{< callout type="warning" >}}
ACME and `TLSCertFile`/`TLSKeyFile` are mutually exclusive. Set only one.
{{< /callout >}}
