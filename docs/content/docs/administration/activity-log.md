---
title: Activity Log
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

## Browsing the log

The paginated list supports filtering by:

- **Action type** — e.g. show only `record_changed` events
- **Resource name** — filter by zone name
- **User** — filter by username

Long diffs are truncated with a fade-out and a **Show more** link to the detail page.

## Detail view

Each log entry has a dedicated detail page (`/admin/activity/:id`) showing the full diff and, where applicable, an **Undo** button.

## Undo

Two undo operations are supported:

### Undo record changes

Restores a zone's RRsets to their state before the change. The undo itself is recorded as a new activity log entry.

### Undo zone deletion

Recreates the deleted zone including all RRsets from the snapshot captured at deletion time.

{{< callout type="warning" >}}
Undo requires the `admin.activity.log.undo` permission. It is granted to the `admin` role by default.
{{< /callout >}}
