---
title: PowerDNS Server
weight: 3
prev: /docs/getting-started/configuration
next: /docs/getting-started/first-run
---

GoPowerDNS-Admin does not run PowerDNS — it drives an existing PowerDNS
**authoritative** server through its built-in REST API. Before it can connect,
that API has to be turned on and reachable. This page covers what to enable on
the PowerDNS side; the matching client settings live in the
[`[pdns]` configuration](/docs/getting-started/configuration#pdns).

{{< callout type="info" >}}
The API is only available in the PowerDNS **Authoritative Server**
(`pdns_server`), not the Recursor. GoPowerDNS-Admin talks to the authoritative
server.
{{< /callout >}}

## Required settings

The REST API is served by PowerDNS's built-in web server, so both have to be
enabled. In `pdns.conf` (or a drop-in under `pdns.d/`):

```ini
# Enable the REST API and set an API key
api=yes
api-key=changeme

# The API rides on the built-in web server — it must be on too
webserver=yes
webserver-address=0.0.0.0      # bind so GoPowerDNS-Admin can reach it
webserver-port=8081
webserver-allow-from=0.0.0.0/0 # tighten to the admin's IP/CIDR — see below
```

| Setting                | Why it matters                                                                 |
| ---------------------- | ------------------------------------------------------------------------------ |
| `api=yes`              | Turns on the `/api/v1` REST endpoints GoPowerDNS-Admin calls                    |
| `api-key`             | Shared secret; must match `apikey` in the `[pdns]` config                       |
| `webserver=yes`        | The API is exposed through the web server — without it the API is unreachable   |
| `webserver-address`    | Must be an address GoPowerDNS-Admin can connect to (not `127.0.0.1` if remote)  |
| `webserver-port`       | The port in your `apiserverurl` (PowerDNS defaults to `8081`)                   |
| `webserver-allow-from` | IP/CIDR allowlist for the web server — **must include GoPowerDNS-Admin's IP**   |

{{< callout type="warning" >}}
`webserver-allow-from` is an allowlist. If GoPowerDNS-Admin's source address is
not in it, every API request is rejected with **403 Forbidden** — the most
common reason a connection fails. In Docker/Kubernetes the source is the
container's network address, not the host's.
{{< /callout >}}

### Hashing the API key (optional)

The `api-key` may be stored as a hash instead of plaintext. Generate one with:

```bash
pdnsutil hash-password
```

Paste the resulting `$scrypt$…` string as the `api-key` value. GoPowerDNS-Admin
still sends the **plaintext** key — the hash only changes how PowerDNS stores it.

## The vhost / server ID

PowerDNS addresses each server by an ID in its API path
(`/api/v1/servers/<id>`). For a standard single-server install this ID is
`localhost`, which is why the `vhost` setting defaults to `localhost`. Set it to
match your server ID only if you have customised it.

## How GoPowerDNS-Admin connects

With the above in place, GoPowerDNS-Admin issues authenticated requests such as:

```bash
curl -H "X-API-Key: changeme" \
  http://<powerdns-host>:8081/api/v1/servers/localhost
```

Use that same command from wherever GoPowerDNS-Admin runs to confirm
reachability before configuring the app. You should get a JSON document
describing the server. Common failures:

| Result                       | Likely cause                                                        |
| ---------------------------- | ------------------------------------------------------------------- |
| Connection refused / timeout | `webserver=no`, wrong port, or a firewall between the two           |
| `403 Forbidden`              | Source IP not in `webserver-allow-from`                             |
| `401 Unauthorized`           | `X-API-Key` missing or does not match `api-key`                     |
| `404` on `/servers/<vhost>`  | `vhost` does not match the PowerDNS server ID (usually `localhost`) |

## Example (Docker Compose)

The flag form used by the official `powerdns/pdns-auth` image:

```yaml
services:
  powerdns:
    image: powerdns/pdns-auth-49:4.9.6
    command:
      - "--api=yes"
      - "--api-key=changeme"
      - "--webserver=yes"
      - "--webserver-address=0.0.0.0"
      - "--webserver-port=8081"
      - "--webserver-allow-from=0.0.0.0/0"
```

See [Deployment → Docker](/docs/deployment/docker) for the full stack that pairs
this with GoPowerDNS-Admin, and [First Run](/docs/getting-started/first-run) for
entering these details in the UI.
