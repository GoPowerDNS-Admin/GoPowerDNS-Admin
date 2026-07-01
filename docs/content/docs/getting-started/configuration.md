---
title: Configuration
weight: 2
prev: /docs/getting-started/installation
next: /docs/getting-started/powerdns
---

Configuration is read from a TOML file, with **environment variables** taking
the highest priority. Every setting has an env-var override, so a container
deployment can be configured entirely without editing files — see
[Environment variable overrides](#environment-variable-overrides) below.

The config file lives at `/etc/go-pdns/main.toml` in the Docker image (mount
your own there, read-only). For a binary or source checkout it defaults to
`etc/main.toml`, or you can pass a path explicitly.

## Quick start

**Docker / binary** — set the essentials with environment variables and let
SQLite handle storage. The `GPDNS_PDNS_*` variables bootstrap the PowerDNS
connection on first startup:

```bash
docker run -d \
  --name gopowerdns-admin \
  -p 8080:8080 \
  -v gopowerdns-data:/var/lib/go-pdns \
  -e GPDNS_WEBSERVER_COOKIEENCRYPTIONKEY="replace_with_a_random_string_of_at_least_32_chars" \
  -e GPDNS_WEBSERVER_ARGON2SALT="replace_with_a_random_salt" \
  -e GPDNS_DB_NAME="/var/lib/go-pdns/go-pdns.db" \
  -e GPDNS_PDNS_APISERVERURL="http://powerdns:8081" \
  -e GPDNS_PDNS_APIKEY="changeme" \
  -e GPDNS_PDNS_VHOST="localhost" \
  ghcr.io/gopowerdns-admin/gopowerdns-admin:latest
```

Prefer a file? Mount your own `main.toml` at `/etc/go-pdns/` (or edit
`etc/main.toml` next to the binary). The keys are documented below.

## Config file and overlays (development)

When running from source you can point the app at an **overlay file** to
override only the keys you care about — the overlay is merged on top of
`main.toml`:

```bash
go run . start                       # loads etc/main.toml
go run . start etc/local/dev.toml    # loads etc/main.toml, then merges dev.toml on top
```

When an overlay path like `etc/local/dev.toml` is given, `main.toml` is derived
from its grandparent directory (`etc/main.toml`) and loaded first, then the
overlay is merged. A common workflow:

```bash
mkdir -p etc/local
cp etc/main.toml etc/local/dev.toml
# edit etc/local/dev.toml, then start with it as the overlay:
go run . start etc/local/dev.toml
```

{{< callout type="warning" >}}
Two keys in `[webserver]` **must** be replaced before production use:
`CookieEncryptionKey` (a random string of at least 32 characters) and
`Argon2Salt` (a random salt). Changing `Argon2Salt` after users exist forces
password resets.
{{< /callout >}}

## `[webserver]`

```toml
[webserver]
Domain              = "localhost"
Port                = 8080
URL                 = "http://localhost:8080"
CookieEncryptionKey = "replace_with_a_random_string_of_at_least_32_chars"
Argon2Salt          = "replace_with_a_random_salt"
BrowseStatic        = false
```

| Key                   | Description                                                            |
| --------------------- | --------------------------------------------------------------------- |
| `Domain`              | Hostname the application serves on                                    |
| `Port`                | TCP port to listen on                                                 |
| `URL`                 | Public base URL (used to build absolute links and OIDC redirects)     |
| `CookieEncryptionKey` | **Required.** Secret used to encrypt session cookies; ≥ 32 characters |
| `Argon2Salt`          | **Required.** Salt for Argon2id password hashing                      |

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

For MySQL/MariaDB:

```toml
[DB]
GormEngine = "mysql"
Host       = "127.0.0.1"
Port       = 3306
User       = "go-pdns"
Password   = "secret"
Name       = "go-pdns"
```

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

Periodically queries the GitHub releases API and shows a hint in the footer
(visible to admins) when a newer version is available. Set `enabled = false` to
disable all outbound version checks. `interval` is clamped to a minimum of 1h.

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

Any config key can be overridden with an environment variable prefixed `GPDNS_`.
The path to the key is upper-cased and joined with underscores
(`webserver.port` → `GPDNS_WEBSERVER_PORT`):

```bash
GPDNS_TITLE="Acme DNS"
GPDNS_WEBSERVER_PORT=9090
GPDNS_DB_GORMENGINE=sqlite
GPDNS_DB_NAME=/var/lib/go-pdns/go-pdns.db
GPDNS_PDNS_APIKEY=changeme
```

See `internal/config/structs.go` for the full set of available fields.
