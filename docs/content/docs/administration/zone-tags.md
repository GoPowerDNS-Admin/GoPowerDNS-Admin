---
title: Zone Tags
description: "Organize zones with tags in GoPowerDNS-Admin and use them to scope role-based access for users and groups."
weight: 2
prev: /docs/administration/rbac
next: /docs/administration/activity-log
---

Zone tags restrict which users and groups can see a zone. Zones without any tags are accessible to all authenticated users.

## Assigning tags to zones

Navigate to **Admin → Zone Tags**. The list is searchable and paginated. Assign one or more tags to a zone — users and groups that have at least one matching tag will be able to see it.

## Assigning tags to users / groups

Tags are assigned to users via **Admin → Users → Edit** and to groups via **Admin → Groups → Edit**.

## Access resolution

A user can access a zone if:

1. The zone has no tags (unrestricted), **or**
2. The user has at least one tag that matches any of the zone's tags (directly or via a group).

Admin users bypass tag restrictions and can always see all zones.
