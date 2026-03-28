---
title: Managing Records
weight: 2
next: /docs/zone-editor/auto-ptr
prev: /docs/zone-editor/zones
---

## Record list

The record table shows all non-DNSSEC records in the zone. DNSSEC-managed types (RRSIG, NSEC, NSEC3, DNSKEY, CDS, CDNSKEY) are automatically hidden to prevent accidental edits.

- **Search** — filters by name, content, or comment
- **Type filter** — pills above the table narrow to a single record type
- **Pagination** — configurable page size (saved per user)
- **Sorting** — click any column header; apex (`@`) records always sort first

## Adding a record

Click **Add Record**. The modal adapts to the selected type:

| Type         | Special behaviour                                                                                   |
| ------------ | --------------------------------------------------------------------------------------------------- |
| **MX**       | Separate priority and hostname fields                                                               |
| **TXT**      | Monospace textarea; content is automatically chunked into RFC-compliant 255-byte strings and quoted |
| **SOA**      | Dedicated SOA modal with individual fields (MNAME, RNAME, serial, refresh, retry, expire, minimum)  |
| **A / AAAA** | Content is validated as a valid IPv4 / IPv6 address before staging                                  |
| **DS**       | Algorithm badge shown next to the algorithm number                                                  |

## Editing a record

Click the **edit** icon on any row. The modal pre-fills with the current values. The SOA record can be viewed and edited via its dedicated modal — it cannot be deleted.

## Deleting a record

Click the **delete** icon and confirm. The change is staged but not yet sent to PowerDNS.

## Saving changes

All staged additions, edits, and deletions are sent to PowerDNS in a single batch when you click **Save Changes**. A summary of the pending count is shown in the toolbar.

{{< callout >}}
Navigating away without saving discards all staged changes. Use **Discard Changes** to explicitly clear the pending queue.
{{< /callout >}}

## Cross-zone hint badges

A/AAAA records with an existing PTR entry in a reverse zone show a **PTR →** badge in the Data column. PTR records show a **← A** or **← AAAA** badge linking back to the forward zone. Clicking a badge navigates to the target zone and highlights the matching record row.
