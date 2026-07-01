---
title: Managing Records
description: "Create, edit, and delete DNS records in GoPowerDNS-Admin with a reactive UI and no full page reloads."
weight: 2
next: /docs/zone-editor/auto-ptr
prev: /docs/zone-editor/zones
---

## Record list

The record table shows all non-DNSSEC records in the zone. DNSSEC-managed types (RRSIG, NSEC, NSEC3, DNSKEY, CDS, CDNSKEY) are automatically hidden to prevent accidental edits.

- **Search** — filters by name, content, or comment
- **Type filter** — pills above the table narrow to a single record type
- **Pagination** — configurable page size (saved per user); your current page is preserved when you save or discard changes
- **Sorting** — click any column header; apex (`@`) records always sort first
- **Sticky toolbar** — the records header (search, type filter, and the Add / Save / Discard actions) stays pinned to the top as you scroll, so the actions are always reachable in long zones
- **Responsive table** — on narrow screens the table scrolls horizontally instead of overflowing; long values such as large TXT records are truncated in the Data column, with the full value shown on hover
- **Status & comments** — each row shows an **Active** / **Disabled** badge and any record comment (truncated, full text on hover)

## Adding a record

Click **Add Record**. The modal adapts to the selected type:

| Type         | Special behaviour                                                                                   |
| ------------ | --------------------------------------------------------------------------------------------------- |
| **MX**       | Separate priority and hostname fields                                                               |
| **TXT**      | Monospace textarea; content is automatically chunked into RFC-compliant 255-byte strings and quoted |
| **SOA**      | Dedicated SOA modal with individual fields (MNAME, RNAME, serial, refresh, retry, expire, minimum)  |
| **A / AAAA** | Content is validated as a valid IPv4 / IPv6 address before staging                                  |

Every record type also supports an optional **comment** (up to 255 characters)
and a **Disabled** toggle that marks the record inactive in PowerDNS without deleting it.

## Editing a record

Click the **edit** icon on any row. The modal pre-fills with the current values, including the comment and disabled state. The SOA record can be viewed and edited via its dedicated modal — it cannot be deleted.

## Deleting a record

Click the **delete** icon and confirm. The change is staged but not yet sent to PowerDNS.

## Saving changes

All staged additions, edits, and deletions are sent to PowerDNS in a single batch when you click **Save Changes**. A summary of the pending count is shown in the toolbar. After saving, the record list reloads but keeps you on the page you were viewing — editing a record on page 2 leaves you on page 2.

{{< callout >}}
Navigating away without saving discards all staged changes. Use **Discard Changes** to explicitly clear the pending queue.
{{< /callout >}}

## Cross-zone hint badges

A/AAAA records with an existing PTR entry in a reverse zone show a **PTR** badge in the Data column. PTR records show a **fwd** badge linking back to the forward zone. Clicking a badge navigates to the target zone and highlights the matching record row.
