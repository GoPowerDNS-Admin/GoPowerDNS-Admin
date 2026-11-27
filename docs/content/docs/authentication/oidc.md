---
title: OpenID Connect
description: "Set up OpenID Connect (OIDC) single sign-on in GoPowerDNS-Admin — provider URL, client credentials, scopes, and group claims."
weight: 3
prev: /docs/authentication/totp
next: /docs/authentication/ldap
---

OIDC authentication integrates with any standards-compliant identity provider (Keycloak, Authentik, Okta, Auth0, Google, etc.).

```toml
[auth.OIDC]
enabled       = true
provider_url  = "https://accounts.example.com"
client_id     = "gopowerdns-admin"
client_secret = "your-client-secret"
redirect_url  = "https://pdns.example.com/auth/oidc/callback"
scopes        = ["openid", "profile", "email", "groups"]
groups_claim  = "groups"

[[auth.OIDC.group_roles]]
group = "dns-admins"
role  = "admin"
```

| Key             | Description                                                                   |
| --------------- | ---------------------------------------------------------------------------- |
| `provider_url`  | Issuer URL; the provider's OIDC discovery document must be reachable under it |
| `client_id`     | OAuth2 client ID registered with your provider                               |
| `client_secret` | OAuth2 client secret                                                          |
| `redirect_url`  | Must be `https://<your-domain>/auth/oidc/callback`                            |
| `scopes`        | Scopes requested at login                                                     |
| `groups_claim`  | ID-token claim that contains the user's group names (for group → role mapping)|
| `group_roles`   | Optional. Group → role mappings created at startup (see below)                |

## Provider setup

Register a new OAuth2 / OIDC client in your identity provider with:

- **Redirect URI**: `https://<your-domain>/auth/oidc/callback`
- **Grant type**: Authorization Code
- **Response type**: `code`
- **Scopes**: `openid`, `profile`, `email` (and `groups` if you use group mapping)

## User provisioning

On first OIDC login, a local user record is created automatically using the email
address from the ID token as the username. The user is assigned the default
**viewer** role.

## Group → role mapping

If you request a groups scope and set `groups_claim`, the group names from the
ID token are synchronized to the user's group memberships on every login. Roles
(and therefore permissions) are then granted through **group mappings**: each
local group can be mapped to one role under **Admin → Groups**.

A group is matched by the exact value in the claim. Case matters, and so does a
leading `/`. Most providers, such as Authentik, send the plain group name
(`dns-admins`). Keycloak sends the path (`/dns-admins`) when **Full group path**
is on. To see the exact value, log in once and look up the group's
**External ID** under **Admin → Groups**.

### In the config file

List the mappings under `group_roles`. On every startup, each listed group is
created if it is missing and mapped to its role. This is how an OIDC-only setup
gets its first admin: the local `admin` account can't log in when
`[auth.LocalDB]` is disabled.

```toml
[[auth.OIDC.group_roles]]
group = "dns-admins"
role  = "admin"

[[auth.OIDC.group_roles]]
group = "dns-operators"
role  = "user"
```

`role` is the name of a built-in role (`admin`, `user`, `viewer`) or a custom
role. A mapping is only created when the group has none yet. A mapping changed
under **Admin → Groups** is kept, and the startup log shows a warning.

### In the admin UI

1. Go to **Admin → Groups** and create a group.
2. Set **Source** to **OIDC** and **External ID** to the exact claim value.
3. Pick the **role** the group grants.
4. Members of the IdP group receive that role on their next login.

A group also appears in the list on its own after the first login of one of
its members. You can then edit it and pick a role.

See [Roles & Permissions](/docs/administration/rbac) for the available roles and how
group mappings resolve to permissions.
