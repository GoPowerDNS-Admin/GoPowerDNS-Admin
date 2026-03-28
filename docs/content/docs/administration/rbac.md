---
title: Roles & Permissions
weight: 1
next: /docs/administration/zone-tags
---

GoPowerDNS-Admin uses role-based access control (RBAC). Roles are assigned to users and groups; each role contains a set of permissions.

## Built-in roles

| Role     | Description                              |
| -------- | ---------------------------------------- |
| `admin`  | Full access to all features and settings |
| `user`   | Can manage zones and records             |
| `viewer` | Read-only access to zones and records    |

## Role editor

Navigate to **Admin → Roles**. Permissions are grouped by resource with icon badges:

- Each resource group shows a count badge and an **All** toggle (checked / indeterminate / unchecked)
- **Select All** / **Deselect All** buttons apply across all groups at once
- Tri-state toggles indicate partial grants within a group

## Permissions

Permissions follow the pattern `resource.action`, for example:

- `zone.view`, `zone.create`, `zone.edit`, `zone.delete`
- `admin.users`, `admin.roles`, `admin.activity.log`, `admin.activity.log.undo`

## Groups

Users can be assigned to groups; permissions are resolved as the union of all role assignments across the user's individual roles and group roles.

## Zone tag access control

Restrict which zones a user or group can see using zone tags. See [Zone Tags](zone-tags).
