---
title: LDAP
description: "Configure LDAP authentication in GoPowerDNS-Admin — bind settings, user filters, StartTLS, and group mapping."
weight: 4
prev: /docs/authentication/oidc
---

LDAP authentication binds against your directory server to validate credentials.

```toml
[auth.LDAP]
enabled           = true
host              = "ldap.example.com"
port              = 389
use_ssl           = false
use_tls           = true                          # StartTLS
skip_verify       = false
bind_dn           = "cn=reader,dc=example,dc=com"
bind_password     = "secret"
base_dn           = "ou=users,dc=example,dc=com"
user_filter       = "(uid={username})"
group_base_dn     = "ou=groups,dc=example,dc=com"
group_filter      = "(member={userdn})"
group_member_attr = "member"
username_attr     = "uid"
email_attr        = "mail"
first_name_attr   = "givenName"
last_name_attr    = "sn"
group_name_attr   = "cn"
timeout           = 10

[[auth.LDAP.group_roles]]
group = "cn=dns-admins,ou=groups,dc=example,dc=com"
role  = "admin"
```

| Key             | Description                                                              |
| --------------- | ----------------------------------------------------------------------- |
| `host` / `port` | Directory server address and port (`389` for plain/StartTLS, `636` LDAPS)|
| `use_ssl`       | Connect over LDAPS (implicit TLS on `636`)                              |
| `use_tls`       | Upgrade a plain connection with **StartTLS**                            |
| `skip_verify`   | Skip TLS certificate verification (testing only)                       |
| `bind_dn` / `bind_password` | Service account used to search for users                   |
| `base_dn`       | Search base for user lookups                                           |
| `user_filter`   | User search filter; `{username}` is replaced with the login name       |
| `group_roles`   | Optional. Group → role mappings created at startup (see below)         |

## TLS

LDAP supports both transport modes:

```toml
# LDAPS (implicit TLS)
host    = "ldap.example.com"
port    = 636
use_ssl = true

# StartTLS (upgrade a plain connection)
host    = "ldap.example.com"
port    = 389
use_tls = true
```

Set `skip_verify = true` only for testing against a self-signed certificate.

## User provisioning

On first LDAP login, a local user record is created automatically (username taken
from `username_attr`). The user is assigned the default **viewer** role.

## Group → role mapping

When the group attributes are configured (`group_base_dn`, `group_filter`,
`group_member_attr`, `group_name_attr`), the user's LDAP groups are extracted on
each login and synchronized to their group memberships. Roles are then granted
through **group mappings**.

A group is matched by its full DN, for example
`cn=dns-admins,ou=groups,dc=example,dc=com`. The `group_name_attr` value is
only used as the display name.

### In the config file

List the mappings under `group_roles`. On every startup, each listed group is
created if it is missing and mapped to its role. This is how an LDAP-only setup
gets its first admin: the local `admin` account can't log in when
`[auth.LocalDB]` is disabled.

```toml
[[auth.LDAP.group_roles]]
group = "cn=dns-admins,ou=groups,dc=example,dc=com"
role  = "admin"
```

`role` is the name of a built-in role (`admin`, `user`, `viewer`) or a custom
role. A mapping is only created when the group has none yet. A mapping changed
under **Admin → Groups** is kept, and the startup log shows a warning.

### In the admin UI

1. Go to **Admin → Groups** and create a group.
2. Set **Source** to **LDAP** and **External ID** to the group's full DN.
3. Pick the **role** the group grants.
4. Members of the LDAP group receive that role on their next login.

See [Roles & Permissions](/docs/administration/rbac) for details.
