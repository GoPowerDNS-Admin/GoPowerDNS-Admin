---
title: Managing Zones
weight: 1
next: /docs/zone-editor/records
---

## Creating a zone

Navigate to **Zones → Add Zone**. Fill in:

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

## Deleting a zone

Click **Delete Zone** in the zone settings card. A confirmation dialog requires you to type the zone name before deletion proceeds. The full zone snapshot is saved to the activity log and can be restored via **Undo**.
