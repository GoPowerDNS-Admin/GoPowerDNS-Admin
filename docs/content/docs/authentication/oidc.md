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
```

| Key             | Description                                                                   |
| --------------- | ---------------------------------------------------------------------------- |
| `provider_url`  | Issuer URL; the provider's OIDC discovery document must be reachable under it |
| `client_id`     | OAuth2 client ID registered with your provider                               |
| `client_secret` | OAuth2 client secret                                                          |
| `redirect_url`  | Must be `https://<your-domain>/auth/oidc/callback`                            |
| `scopes`        | Scopes requested at login                                                     |
| `groups_claim`  | ID-token claim that contains the user's group names (for group → role mapping)|

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

To grant roles based on your IdP's groups:

1. Create a local **group** whose name matches the group name your IdP sends.
2. Map that group to the desired **role** when creating or editing it.
3. Members of the matching IdP group receive that role automatically on their next login.

See [Roles & Permissions](/docs/administration/rbac) for the available roles and how
group mappings resolve to permissions.
