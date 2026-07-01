---
title: TOTP (2FA)
description: "Enable TOTP two-factor authentication (2FA) for local GoPowerDNS-Admin accounts using any authenticator app."
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

Enforcement is **per-user**, not a global config option. An admin can require a
specific user to use TOTP by enabling the **TOTP Required** flag when creating or
editing that user under **Admin → Users → Edit**.

When the flag is set, the user is redirected to the enrollment page on their next
login and cannot proceed until they have configured an authenticator. A user
cannot disable TOTP on their own account while the flag is enforced.

## Recovery

If a user loses access to their authenticator, an admin can disable TOTP for their account via **Admin → Users → Edit**.
