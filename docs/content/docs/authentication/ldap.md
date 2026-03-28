---
title: LDAP
weight: 4
prev: /docs/authentication/oidc
---

LDAP authentication performs a bind against your directory server to validate credentials.

```toml
[auth.ldap]
Enabled    = true
Server     = "ldap://ldap.example.com:389"
BindDN     = "cn=reader,dc=example,dc=com"
BindPass   = "secret"
BaseDN     = "ou=users,dc=example,dc=com"
UserFilter = "(uid=%s)"
```

## User provisioning

On first LDAP login, a local user record is created automatically. The user is assigned the default viewer role — an admin must grant additional roles as needed.

## TLS

For LDAPS, change the scheme:

```toml
Server = "ldaps://ldap.example.com:636"
```

StartTLS is not currently supported.
