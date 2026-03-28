---
title: Reverse Proxy
weight: 3
prev: /docs/deployment/tls
---

When running behind HAProxy, nginx, Traefik, or any other reverse proxy, enable the `[webserver.reverseproxy]` block so that the real client IP is used in activity log entries and IP-based logic.

```toml
[webserver.reverseproxy]
enabled     = true
trustedips  = ["127.0.0.1", "10.0.0.0/8", "172.16.0.0/12", "192.168.0.0/16"]
proxyheader = "X-Forwarded-For"   # or "X-Real-IP" for nginx
```

| Field         | Default           | Description                                                                                     |
| ------------- | ----------------- | ----------------------------------------------------------------------------------------------- |
| `enabled`     | `false`           | Activates trusted-proxy IP checking. When `false`, the proxy header is trusted unconditionally. |
| `trustedips`  | _(none)_          | Required when `enabled = true`. IPv4/IPv6 addresses or CIDR ranges of your upstream proxies.    |
| `proxyheader` | `X-Forwarded-For` | HTTP header used to read the real client IP.                                                    |

{{< callout type="warning" >}}
`trustedips` must not be empty when `enabled = true`. The application refuses to start if this constraint is violated.
{{< /callout >}}

## HAProxy example

```haproxy
frontend https
    bind *:443 ssl crt /etc/ssl/certs/fullchain.pem
    default_backend gopowerdns

backend gopowerdns
    option forwardfor
    server app 127.0.0.1:8080
```

## nginx example

```nginx
location / {
    proxy_pass         http://127.0.0.1:8080;
    proxy_set_header   X-Real-IP $remote_addr;
    proxy_set_header   Host $host;
}
```

Set `proxyheader = "X-Real-IP"` in `main.toml` when using this nginx config.
