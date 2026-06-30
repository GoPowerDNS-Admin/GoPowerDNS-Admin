---
title: Local Accounts
weight: 1
next: /docs/authentication/totp
---

Local authentication stores credentials in the application database. It is enabled by default.

```toml
[auth.LocalDB]
enabled = true
```

## Password policy

Passwords are hashed with [Argon2id](https://en.wikipedia.org/wiki/Argon2). There is no enforced minimum length in the current release — set a strong password manually.

{{< callout >}}
The `Argon2Salt` key under `[webserver]` is used for session/cookie cryptography, **not** for password hashing — Argon2id generates a per-password salt automatically.
{{< /callout >}}

## TOTP (two-factor authentication)

TOTP can be enabled by a user for their own account, or required per-user by an admin. See [TOTP](/docs/authentication/totp) for details.

## Default admin account

A default admin account (`admin` / `changeme`) is seeded on first run. Change the password immediately after your first login via **Profile → Change Password**.
