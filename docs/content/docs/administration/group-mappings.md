---
title: Group Mappings
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
- When a user logs in via OIDC or LDAP, their external group names are
  synchronized to local groups **by name**. If a matching local group exists, the
  user inherits that group's mapped role.

## Mapping a group to a role

1. Navigate to **Admin → Groups**.
2. Create or edit a group, giving it a **name** that matches the group name your
   identity provider sends (for OIDC, the value in the `groups_claim`; for LDAP,
   the `group_name_attr`).
3. Select the **role** the group should grant.
4. Save. Members of that group receive the role on their next login.

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
