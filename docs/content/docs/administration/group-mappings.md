---
title: Group Mappings
description: "Map OIDC and LDAP groups to GoPowerDNS-Admin roles to assign permissions automatically when users log in."
weight: 5
prev: /docs/administration/ttl-presets
next: /docs/administration/branding
---

Group mappings connect **groups** to **roles**. They are how permissions are
granted to users who sign in through an external identity provider (OIDC or
LDAP), and they also let you manage permissions for sets of local users in one
place.

## How it works

- Each **group** can be mapped to exactly one **role**.
- A user's effective permissions are the union of their directly-assigned roles
  and the roles of every group they belong to.
- When a user logs in via OIDC or LDAP, their external groups are synchronized
  to groups with the same **source** and **external ID**. For OIDC the external
  ID is the value in the `groups_claim`. For LDAP it is the group's full DN.
  Missing groups are created, and the user inherits each group's mapped role.

## Mapping a group to a role

1. Navigate to **Admin → Groups**.
2. Create or edit a group. Set its **Source** to OIDC or LDAP, and its
   **External ID** to the exact value your identity provider sends.
3. Select the **role** the group should grant.
4. Save. Members of that group receive the role on their next login.

## Mapping groups in the config file

Mappings can also be listed under `group_roles` in `[auth.OIDC]` or
`[auth.LDAP]`. They are created at startup, so an OIDC-only or LDAP-only setup
has an admin from the first login:

```toml
[[auth.OIDC.group_roles]]
group = "dns-admins"
role  = "admin"
```

A mapping is only created when the group has none. A role changed in the admin
UI is kept. A mapping removed in the admin UI comes back at the next start, so
to revoke a role for good, delete the entry from the config. See
[OpenID Connect](/docs/authentication/oidc#in-the-config-file) and
[LDAP](/docs/authentication/ldap#in-the-config-file) for details.

## Example

| External group (IdP) | Local group | Mapped role |
| -------------------- | ----------- | ----------- |
| `dns-admins`         | `dns-admins`| `admin`     |
| `dns-operators`      | `dns-operators` | `user`  |
| `everyone`           | `everyone`  | `viewer`    |

A user in the IdP's `dns-operators` group is synced into the local
`dns-operators` group on login and is granted the `user` role.

{{< callout >}}
External users are still created with the default `viewer` role on first login.
Group mappings then layer additional roles on top based on group membership.
{{< /callout >}}

See [Roles & Permissions](/docs/administration/rbac) for the role/permission model, and
[OpenID Connect](/docs/authentication/oidc) / [LDAP](/docs/authentication/ldap) for how
external groups are sourced.
