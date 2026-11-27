---
title: Configuration
description: "Configure GoPowerDNS-Admin with TOML and environment variables — web server, database, authentication, PowerDNS, and branding settings."
weight: 2
prev: /docs/getting-started/installation
next: /docs/getting-started/powerdns
---

Configuration is read from a TOML file. **Environment variables** can override
keys that appear in that file — see
[Environment variable overrides](#environment-variable-overrides) below.

The app reads `main.toml` from the folder given with `-c` (default: `etc/`). In
the Docker image that folder is `/etc/go-pdns/`. The image ships no config, so
you must mount your own `main.toml` there.

## Quick start

**Docker** — copy [`etc/main.toml`](https://github.com/GoPowerDNS-Admin/GoPowerDNS-Admin/blob/main/etc/main.toml)
from the repository into a local folder, adjust it, and mount that folder. To
bootstrap the PowerDNS connection on first startup, fill in the
[`[pdns]`](#pdns) section:

```bash
docker run -d \
  --name gopowerdns-admin \
  -p 8080:8080 \
  -v ./etc:/etc/go-pdns:ro \
  -v gopowerdns-data:/var/lib/go-pdns \
  ghcr.io/gopowerdns-admin/gopowerdns-admin:latest
```

**Binary** — put `main.toml` in an `etc/` folder next to the binary, or pass
another folder with `-c`. The keys are documented below.

## Config file and overlays (development)

When running from source you can point the app at an **overlay file** to
override only the keys you care about. Pass it with `-c`:

```bash
go run . start                          # loads etc/main.toml
go run . start -c etc/local/dev.toml    # loads etc/main.toml, then merges dev.toml on top
```

When `-c` points at a `.toml` file, `main.toml` is taken from that file's
grandparent folder (`etc/main.toml`) and loaded first. The overlay is then
merged on top. Files in `etc/local/` are not read unless you pass them with
`-c`. A common workflow:

```bash
mkdir -p etc/local
cp etc/main.toml etc/local/dev.toml
# edit etc/local/dev.toml, then start with it as the overlay:
go run . start -c etc/local/dev.toml
```

To check the result, `go run . -c etc/local/dev.toml --dump-config` prints the
merged configuration as JSON.

{{< callout type="warning" >}}
`CookieEncryptionKey` and `Argon2Salt` are checked at startup, but nothing uses
them yet. Startup fails if either one is set to `change_this_to_a_random_string`
or is shorter than 32 (key) or 16 (salt) characters. Session cookies hold a
random session ID, and each password hash gets its own random salt.
{{< /callout >}}

## `[webserver]`

```toml
[webserver]
Port                = 8080
URL                 = "http://localhost:8080"
CookieEncryptionKey = "replace_with_a_random_string_of_at_least_32_chars"
Argon2Salt          = "replace_with_a_random_salt"
ShutDownTime        = 5
BrowseStatic        = false

[webserver.session]
ExpiryTime = "24h"
```

| Key                   | Description                                                                         |
| --------------------- | ----------------------------------------------------------------------------------- |
| `Port`                | **Required.** TCP port to listen on (all interfaces)                                |
| `URL`                 | **Required.** Public base URL. Used as the redirect target after an OIDC logout     |
| `CookieEncryptionKey` | Checked at startup, not used yet (see the callout above)                            |
| `Argon2Salt`          | Checked at startup, not used yet (see the callout above)                            |
| `ShutDownTime`        | Seconds the health check reports `shutting_down` before the server stops (default 5) |
| `session.ExpiryTime`  | Session lifetime, e.g. `24h`                                                        |

`Domain`, `CacheEnabled`, `CleanPath`, and `DisableRecover` are accepted but not
used.

TLS, ACME, reverse-proxy, and session sub-keys also live under `[webserver]` —
see [TLS / HTTPS](/docs/deployment/tls) and [Reverse Proxy](/docs/deployment/reverse-proxy).

## `[DB]`

```toml
[DB]
GormEngine = "sqlite"                      # sqlite | mysql | postgres
Name       = "/var/lib/go-pdns/go-pdns.db" # SQLite: file path; MySQL/PG: database name
```

For SQLite, only `Name` is used (the file path). Sessions are stored in a
separate file named `<Name>-sessions.db`. `Host`, `Port`, `User`, and `Password`
are ignored.

For MySQL/MariaDB — `Extras` is added to the connection string as its query
part. Set `parseTime=True`, because GORM needs it to read timestamps:

```toml
[DB]
GormEngine = "mysql"
Host       = "127.0.0.1"
Port       = 3306
User       = "go-pdns"
Password   = "secret"
Name       = "go-pdns"
Extras     = "charset=utf8mb4&parseTime=True&loc=Local"
```

With MySQL and PostgreSQL, sessions are stored in a `sessions` table in the same
database.

For PostgreSQL — `Extras` sets the `sslmode` (`disable`, `require`,
`verify-full`, …; defaults to `disable` when empty):

```toml
[DB]
GormEngine = "postgres"
Host       = "127.0.0.1"
Port       = 5432
User       = "go-pdns"
Password   = "secret"
Name       = "go-pdns"
Extras     = "disable"   # sslmode value
```

## `[auth]`

Authentication methods are enabled independently. See the
[Authentication](/docs/authentication) section for full details on each provider.

```toml
[auth.LocalDB]
enabled = true

[auth.OIDC]
enabled       = false
provider_url  = "https://accounts.example.com"
client_id     = "gopowerdns-admin"
client_secret = "your-client-secret"
redirect_url  = "https://pdns.example.com/auth/oidc/callback"
scopes        = ["openid", "profile", "email", "groups"]
groups_claim  = "groups"

[auth.LDAP]
enabled       = false
host          = "ldap.example.com"
port          = 389
use_ssl       = false
use_tls       = true                 # StartTLS
skip_verify   = false
bind_dn       = "cn=reader,dc=example,dc=com"
bind_password = "secret"
base_dn       = "ou=users,dc=example,dc=com"
user_filter   = "(uid={username})"
```

{{< callout >}}
Note the section names are `[auth.LocalDB]`, `[auth.OIDC]`, and `[auth.LDAP]`,
and the keys inside them are `snake_case`.
{{< /callout >}}

If you disable `[auth.LocalDB]`, list at least one admin group under
`group_roles` in `[auth.OIDC]` or `[auth.LDAP]`. Otherwise nobody can reach the
admin pages. See [Group Mappings](/docs/administration/group-mappings#mapping-groups-in-the-config-file).

## `[pdns]`

Optionally bootstrap the PowerDNS connection on first startup. When all three
fields are set and no PowerDNS server is already configured in the database, the
connection is seeded automatically — handy for demo or automated deployments
where the admin UI can't be used for initial setup. Otherwise, configure it from
the UI after first login (see [First Run](/docs/getting-started/first-run)).

The PowerDNS server itself must have its API enabled and reachable for any of
this to work — see [PowerDNS Server](/docs/getting-started/powerdns) for the
required `pdns.conf` settings.

```toml
[pdns]
apiserverurl = "http://127.0.0.1:8081"
apikey       = "your-api-key"
vhost        = "localhost"
```

## `[update]`

Periodically queries the GitHub releases API and shows a hint in the footer when
a newer version is available. The hint is visible to users with the
`admin.server.config` permission. The check is off unless `enabled = true`; the
sample config turns it on. An `interval` below 1h falls back to 24h.

```toml
[update]
enabled    = true
interval   = "24h"
repository = "GoPowerDNS-Admin/GoPowerDNS-Admin"
```

## `[branding]` (optional)

Override the product name and logo shown in the sidebar, login, and TOTP pages.
Any field left unset falls back to a default. These can also be edited at runtime
via **Settings → Branding** (which takes precedence). See [Branding](/docs/administration/branding)
for details, including the same-origin image constraint.

```toml
[branding]
Name          = "Acme DNS"
LogoURL       = "/static/img/your-logo.svg"
FaviconURL    = "/static/img/your-favicon.svg"
FaviconPNGURL = "/static/img/your-favicon.png"
```

## Environment variable overrides

A config key can be overridden with an environment variable prefixed `GPDNS_`.
The path to the key is upper-cased and joined with underscores
(`webserver.port` → `GPDNS_WEBSERVER_PORT`):

```bash
GPDNS_TITLE="Acme DNS"
GPDNS_WEBSERVER_PORT=9090
GPDNS_DB_GORMENGINE=sqlite
GPDNS_DB_NAME=/var/lib/go-pdns/go-pdns.db
```

{{< callout type="warning" >}}
An env var only takes effect when its key also appears in the TOML file. For
example, `GPDNS_PDNS_APIKEY` does nothing unless `main.toml` has a `[pdns]`
section with an `apikey` key. An empty value is enough:

```toml
[pdns]
apiserverurl = ""
apikey       = ""
vhost        = ""
```
{{< /callout >}}

See `internal/config/structs.go` for the full set of available fields.
