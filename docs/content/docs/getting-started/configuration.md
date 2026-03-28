---
title: Configuration
weight: 2
prev: /docs/getting-started/installation
next: /docs/getting-started/first-run
---

Configuration is read from a TOML file. The application looks for config in the following order:

1. `etc/local/main.toml` (gitignored — for local overrides)
2. `etc/main.toml` (committed defaults)

## Quick start

```bash
mkdir -p etc/local
cp etc/main.toml etc/local/main.toml
# edit etc/local/main.toml to match your setup
```

## Key sections

### `[webserver]`

```toml
[webserver]
Host = "0.0.0.0"
Port = 8080
```

### `[database]`

```toml
[database]
GORMEngine = "sqlite"          # sqlite | mysql | postgres
Name       = "./go-pdns.db"   # SQLite: file path; MySQL/PG: database name
```

For MySQL/MariaDB:

```toml
[database]
GORMEngine = "mysql"
Host       = "127.0.0.1"
Port       = 3306
User       = "gopdns"
Password   = "secret"
Name       = "gopdns"
```

### `[pdns]`

Bootstrap the PowerDNS connection on first startup. All three fields must be set:

```toml
[pdns]
APIServerURL = "http://127.0.0.1:8081"
APIKey       = "your-api-key"
VHost        = "localhost"
```

### `[auth]`

Enable authentication methods independently:

```toml
[auth]
  [auth.local]
  Enabled = true

  [auth.oidc]
  Enabled      = false
  ProviderURL  = "https://accounts.example.com"
  ClientID     = "gopowerdns-admin"
  ClientSecret = "secret"
  RedirectURL  = "https://pdns.example.com/auth/oidc/callback"

  [auth.ldap]
  Enabled  = false
  Server   = "ldap://ldap.example.com:389"
  BindDN   = "cn=reader,dc=example,dc=com"
  BindPass = "secret"
  BaseDN   = "ou=users,dc=example,dc=com"
```

## Environment variable overrides

Every config key can be overridden with an environment variable prefixed `GPDNS_`:

```bash
GPDNS_WEBSERVER_PORT=9090
GPDNS_DB_GORMENGINE=sqlite
GPDNS_DB_NAME=/var/lib/go-pdns/go-pdns.db
GPDNS_PDNS_APIKEY=changeme
```

See `internal/config/structs.go` for all available fields.
