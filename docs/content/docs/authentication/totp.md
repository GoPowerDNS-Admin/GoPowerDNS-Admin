---
title: TOTP (2FA)
weight: 2
prev: /docs/authentication/local
next: /docs/authentication/oidc
---

TOTP two-factor authentication is available for local accounts only. OIDC and LDAP users rely on their identity provider for MFA.

## User opt-in

Users can enable TOTP for their own account:

1. Log in and go to **Profile → Two-Factor Authentication**.
2. Scan the QR code with an authenticator app (Google Authenticator, Aegis, etc.).
3. Enter the 6-digit code to confirm and activate.

## Admin enforcement

Admins can require all local users to enroll TOTP before accessing the application. Configure in `main.toml`:

```toml
[auth.local]
Enabled         = true
TOTPEnforced    = true
```

When enforcement is on, users without TOTP configured are redirected to the enrollment page on their next login.

## Recovery

If a user loses access to their authenticator, an admin can disable TOTP for their account via **Admin → Users → Edit**.
