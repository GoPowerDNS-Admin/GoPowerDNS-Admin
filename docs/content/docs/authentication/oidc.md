---
title: OpenID Connect
weight: 3
prev: /docs/authentication/totp
next: /docs/authentication/ldap
---

OIDC authentication integrates with any standards-compliant identity provider (Keycloak, Authentik, Okta, Auth0, etc.).

```toml
[auth.oidc]
Enabled      = true
ProviderURL  = "https://accounts.example.com"
ClientID     = "gopowerdns-admin"
ClientSecret = "your-client-secret"
RedirectURL  = "https://pdns.example.com/auth/oidc/callback"
```

## Provider setup

Register a new OAuth2 / OIDC client in your identity provider with:

- **Redirect URI**: `https://<your-domain>/auth/oidc/callback`
- **Grant type**: Authorization Code
- **Response type**: `code`
- **Scopes**: `openid`, `profile`, `email`

## User provisioning

On first OIDC login, a local user record is created automatically using the email address from the ID token as the username. The user is assigned the default viewer role — an admin must grant additional roles as needed.
