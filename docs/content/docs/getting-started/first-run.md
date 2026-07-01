---
title: First Run
description: "First run of GoPowerDNS-Admin — the default admin login, connecting to PowerDNS, and verifying the health check."
weight: 4
prev: /docs/getting-started/powerdns
---

## Default credentials

On first startup, a default admin account is seeded:

| Field    | Value      |
| -------- | ---------- |
| Username | `admin`    |
| Password | `changeme` |

{{< callout type="warning" >}}
Change the admin password immediately after your first login.
{{< /callout >}}

## Connecting to PowerDNS

Make sure your PowerDNS server has its API enabled and reachable first — see
[PowerDNS Server](/docs/getting-started/powerdns).

If you did not set the `[pdns]` bootstrap section in `main.toml`, connect via the UI:

1. Log in as `admin`
2. Go to **Admin → PowerDNS Server Settings**
3. Enter your PowerDNS API URL, API key, and virtual host
4. Click **Save** — the connection is tested immediately

## Health check

Verify the application is running correctly:

```bash
curl http://localhost:8080/health
```

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

A `powerdns: "not_configured"` response means the PowerDNS connection has not been set up yet — this does not affect overall status.
