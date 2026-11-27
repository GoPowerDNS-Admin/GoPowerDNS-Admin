---
title: Managing Zones
description: "Create, edit, and delete DNS zones in GoPowerDNS-Admin, including zone kind, SOA-EDIT-API, and master servers."
weight: 1
next: /docs/zone-editor/records
---

## Creating a zone

Click **Add Zone** in the sidebar (needs the `zone.create` permission). Fill in:

| Field            | Description                                                                               |
| ---------------- | ----------------------------------------------------------------------------------------- |
| **Name**         | Zone name, e.g. `example.com` — a trailing dot is added automatically                     |
| **Kind**         | `Native`, `Master`, or `Slave`                                                            |
| **SOA-EDIT-API** | How PowerDNS increments the SOA serial on changes (`DEFAULT`, `INCREASE`, `EPOCH`, `OFF`) |
| **Masters**      | Comma-separated list of master IP addresses — only shown for Slave zones                  |

### Reverse zones

For reverse zones, you can enter a CIDR prefix instead of a zone name:

| Input            | Created zone                |
| ---------------- | --------------------------- |
| `192.168.1.0/24` | `1.168.192.in-addr.arpa.`   |
| `10.0.0.0/8`     | `10.in-addr.arpa.`          |
| `2001:db8::/32`  | `8.b.d.0.1.0.0.2.ip6.arpa.` |

Duplicate zones are detected before creation and a direct link to the existing zone is shown.

## Zone settings

Each zone has a collapsible **Zone Settings** card at the top of the editor. Changes here (kind, SOA-EDIT-API, masters, Auto-PTR) are saved independently of record changes and redirect back to the same zone with a success notification.

Secondary (Slave) zones show their records as a read-only table, because they come from the master server. They have no DNSSEC card.

## Who can do what

| Action                                          | Permission    |
| ----------------------------------------------- | ------------- |
| Open a zone and see its records and DNSSEC keys | `zone.read`   |
| Change settings, records, and DNSSEC            | `zone.update` |
| Delete the zone                                 | `zone.delete` |

Users with `zone.read` but not `zone.update` see the zone read-only. The settings fields are greyed out, the records card shows a **Read-Only** badge, and the add, edit, save, delete, and DNSSEC actions are hidden. Zone tags can further limit which zones a user sees — see [Zone Tags](/docs/administration/zone-tags).

## Deleting a zone

Click **Delete Zone** in the zone settings card. A confirmation dialog requires you to type the zone name before deletion proceeds. A snapshot of the zone (kind, SOA-EDIT-API, masters, and all records) is saved to the activity log and can be restored via **Undo**. The undo brings all records back enabled, and it does not restore record comments or DNSSEC keys.
