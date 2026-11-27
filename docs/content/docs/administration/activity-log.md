---
title: Activity Log
description: "Track every change in GoPowerDNS-Admin with the activity log — view before/after diffs of zone and record changes and undo them."
weight: 3
prev: /docs/administration/zone-tags
next: /docs/administration/ttl-presets
---

All significant user actions are recorded in the activity log, accessible at **Admin → Activity Log**.

## Recorded events

| Event                   | Details stored                           |
| ----------------------- | ---------------------------------------- |
| Login success / failure | IP address, username                     |
| Logout                  | —                                        |
| Zone created            | Zone name, kind                          |
| Zone settings updated   | Before/after diff of changed fields      |
| Zone deleted            | Full zone snapshot (all RRsets) for undo |
| Records changed         | Per-RRset before/after diff              |
| Record change undone    | The record change that was reverted      |
| Zone deletion undone    | The recreated zone                       |
| DNSSEC enabled          | The created key                          |
| DNSSEC disabled         | Number of keys removed                   |
| DNSSEC key changed      | Key ID and new active/published state    |
| DNSSEC key deleted      | Key ID                                   |

Undo operations are themselves recorded as `record_undone` and
`zone_deleted_undone` events, so the log always shows who reverted what.

Reading the log needs the `admin.activity.log` permission, which all three
built-in roles have. The log covers all zones; zone tags do not filter it.

## Browsing the log

The paginated list supports filtering by:

- **Action type** — e.g. show only `record_changed` events
- **Zone** — filter by zone name, with suggestions from zones seen in the log
- **User** — filter by username
- **Date range** — from and to dates

Long diffs are cut off with a fade-out. Click the eye button on a row to open the entry and see the full diff.

## Detail view

Each log entry has a dedicated detail page (`/admin/activity/:id`) showing the full diff and, where applicable, an **Undo** button.

## Undo

Two undo operations are supported:

### Undo record changes

Restores a zone's RRsets to their state before the change. The undo itself is recorded as a new activity log entry.

### Undo zone deletion

Recreates the deleted zone from the snapshot captured at deletion time: its kind, SOA-EDIT-API, masters, and all RRsets. Records come back enabled, even if some were disabled before. Record comments and DNSSEC keys are not part of the snapshot and are not restored.

{{< callout type="warning" >}}
Undo requires the `admin.activity.log.undo` permission. It is granted to the `admin` role by default.
{{< /callout >}}
