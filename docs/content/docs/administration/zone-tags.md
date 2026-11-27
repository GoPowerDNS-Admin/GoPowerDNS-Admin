---
title: Zone Tags
description: "Organize zones with tags in GoPowerDNS-Admin and use them to scope role-based access for users and groups."
weight: 2
prev: /docs/administration/rbac
next: /docs/administration/activity-log
---

Zone tags limit which zones a user can see. The limit is set on the user, not on the zone: as soon as a user has a tag, they only see zones that carry one of their tags.

## Creating tags

Create tags first under **Admin → Tags** (needs the `admin.tags` permission).

## Assigning tags to zones

Navigate to **Admin → Zone Tags**. The list is searchable and paginated. Assign one or more tags to a zone — users and groups that have at least one matching tag will be able to see it.

## Assigning tags to users / groups

Tags are assigned to users via **Admin → Users → Edit** and to groups via **Admin → Groups → Edit**. Leaving all tags unchecked gives unrestricted access.

## Access resolution

1. A user whose own role is `admin` sees every zone. A role granted through a group does not count for this.
2. A user with no tags, directly or through a group, sees every zone, tagged or not.
3. Any other user sees only zones that carry at least one of their tags. Zones without tags are hidden from them.

Tags do not filter the activity log. It shows entries for all zones.
