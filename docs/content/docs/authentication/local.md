---
title: Local Accounts
description: "Manage local database accounts in GoPowerDNS-Admin, including Argon2 password hashing and optional TOTP two-factor auth."
weight: 1
next: /docs/authentication/totp
---

Local authentication stores credentials in the application database. It is enabled by default.

```toml
[auth.LocalDB]
enabled = true
```

## Password policy

Passwords are hashed with [Argon2id](https://en.wikipedia.org/wiki/Argon2). When users change their own password under **Profile**, it must have at least 8 characters. Passwords set by an admin under **Admin → Users** have no minimum length, so set a strong one.

{{< callout >}}
Argon2id generates a random salt for each password. The `Argon2Salt` key under `[webserver]` is not used for password hashing. It is only checked at startup.
{{< /callout >}}

## TOTP (two-factor authentication)

TOTP can be enabled by a user for their own account, or required per-user by an admin. See [TOTP](/docs/authentication/totp) for details.

## Default admin account

A default admin account (`admin` / `changeme`) is seeded on first run. Change the password immediately after your first login via **Profile → Change Password**.
