---
title: TTL Presets
weight: 4
prev: /docs/administration/activity-log
---

TTL presets appear in the TTL drop-down when editing records, giving users a curated list of common values instead of a free-form number field.

## Managing presets

Navigate to **Admin → TTL Presets**. Each preset has:

- **Label** — displayed in the drop-down (e.g. `5 minutes`)
- **Value** — TTL in seconds (e.g. `300`)

Presets are sorted by value in ascending order.

## Default presets

The following presets are seeded on first run:

| Label      | Seconds |
| ---------- | ------- |
| 1 minute   | 60      |
| 5 minutes  | 300     |
| 15 minutes | 900     |
| 1 hour     | 3600    |
| 6 hours    | 21600   |
| 12 hours   | 43200   |
| 1 day      | 86400   |
| 1 week     | 604800  |

You can delete, rename, or add presets at any time.
