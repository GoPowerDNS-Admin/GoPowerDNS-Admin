---
title: Local Accounts
weight: 1
next: /docs/authentication/totp
---

Local authentication stores credentials in the application database. It is enabled by default.

```toml
[auth.local]
Enabled = true
```

## Password policy

Passwords are hashed with bcrypt. There is no enforced minimum length in the current release — set a strong password manually.

## TOTP (two-factor authentication)

TOTP can be enabled per-user or enforced globally. See [TOTP](totp) for details.

## Default admin account

A default admin account (`admin` / `changeme`) is seeded on first run. Change the password immediately after your first login via **Profile → Change Password**.
