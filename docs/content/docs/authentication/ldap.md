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

On first LDAP login, a local user record is created automatically. Its username is
the name typed at login. The user is assigned the default **viewer** role.

## Group → role mapping

When `group_base_dn` is set, the user's LDAP groups are looked up on each login
with `group_filter` and synchronized to their group memberships. Roles are then
granted through **group mappings**.

A group is matched by its full DN, for example
`cn=dns-admins,ou=groups,dc=example,dc=com`. Its display name is the first part
of the DN (`dns-admins`). `group_member_attr` and `group_name_attr` are accepted
but not used.

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
role. The entries are applied on every start:

- A group without a mapping gets one. This also brings back a mapping that was
  removed under **Admin → Groups**. To revoke a role for good, delete the entry
  from the config.
- A group that maps to a different role keeps it, and the startup log shows a
  warning.

### In the admin UI

1. Go to **Admin → Groups** and create a group.
2. Set **Source** to **LDAP** and **External ID** to the group's full DN.
3. Pick the **role** the group grants.
4. Members of the LDAP group receive that role on their next login.

See [Roles & Permissions](/docs/administration/rbac) for details.
