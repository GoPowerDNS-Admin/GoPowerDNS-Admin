---
title: Roles & Permissions
description: "Configure role-based access control in GoPowerDNS-Admin — define roles, assign fine-grained permissions, and restrict users to zone tags."
weight: 1
next: /docs/administration/zone-tags
---

GoPowerDNS-Admin uses role-based access control (RBAC). Roles are assigned to users and groups; each role contains a set of permissions.

## Built-in roles

Three roles are seeded on first run:

| Role     | Description                              | Permissions                                                                                        |
| -------- | ---------------------------------------- | -------------------------------------------------------------------------------------------------- |
| `admin`  | Full access to all features and settings | Every permission                                                                                   |
| `user`   | Can manage zones and records             | `dashboard.view`, `zone.create`, `zone.read`, `zone.update`, `zone.delete`, `zone.list`, `admin.activity.log` |
| `viewer` | Read-only access to zones and records    | `dashboard.view`, `zone.read`, `zone.list`, `admin.server.config`, `admin.activity.log`            |

## Role editor

Navigate to **Admin → Roles**. Permissions are grouped by resource with icon badges:

- Each resource group shows a count badge and an **All** toggle (checked / indeterminate / unchecked)
- **Select All** / **Deselect All** buttons apply across all groups at once
- Tri-state toggles indicate partial grants within a group

## Permissions

Permissions follow the pattern `resource.action`. The full set:

| Group        | Permissions                                                                                                    |
| ------------ | ------------------------------------------------------------------------------------------------------------- |
| Dashboard    | `dashboard.view`                                                                                              |
| Zones        | `zone.create`, `zone.read`, `zone.update`, `zone.delete`, `zone.list`                                         |
| Admin        | `admin.settings`, `admin.server.config`, `admin.pdns.server`, `admin.zone.records`, `admin.users`, `admin.roles`, `admin.groups`, `admin.group.mappings`, `admin.tags`, `admin.zone.tags`, `admin.ttl.presets`, `admin.branding` |
| Activity log | `admin.activity.log`, `admin.activity.log.undo`                                                               |

{{< callout >}}
The actions are `read` and `update` — not `view`/`edit`. So a read-only grant is
`zone.read`, and editing records is `zone.update`.
{{< /callout >}}

## Groups

Users can be assigned to groups; permissions are resolved as the union of all role
assignments across the user's individual roles and group roles. Groups also
provide the bridge for external identity providers: see
[Group Mappings](/docs/administration/group-mappings) for mapping OIDC/LDAP groups to roles.

## Zone tag access control

Restrict which zones a user or group can see using zone tags. See [Zone Tags](/docs/administration/zone-tags).
