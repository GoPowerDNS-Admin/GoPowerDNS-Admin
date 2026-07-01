---
title: Auto-PTR
description: "Automatically create and update PTR records in reverse zones when A/AAAA records change, using GoPowerDNS-Admin Auto-PTR."
weight: 3
prev: /docs/zone-editor/records
---

Auto-PTR automatically manages PTR records in reverse zones when A or AAAA records change. Enable it per-zone in the **Zone Settings** card.

## How it works

When **Auto-PTR** is enabled and you save A or AAAA record changes, the application:

1. Finds the most-specific reverse zone in PowerDNS that covers each IP address.
2. **Creates or replaces** a PTR record pointing to the record's FQDN for every new IP.
3. **Deletes** the PTR record for every IP that was removed — but only if it still points to the same FQDN. If the PTR was manually changed to point elsewhere, it is left untouched.

All PTR operations are recorded in the [activity log](/docs/administration/activity-log).

## Enabling Auto-PTR

1. Open a forward zone in the editor.
2. Expand the **Zone Settings** card.
3. Check **Auto-create PTR records for A / AAAA changes**.
4. Click **Update Zone**.

{{< callout type="warning" >}}
Auto-PTR is silently ignored for reverse zones and Slave zones — the checkbox is hidden for those zone types.
{{< /callout >}}

## Missing reverse zone warning

If Auto-PTR is enabled but no reverse zone exists in PowerDNS for a given IP, a warning toast is shown after saving:

```
Auto-PTR: no reverse zone found for 192.0.2.1
```

If several addresses lack a reverse zone, they are listed together, comma-separated:

```
Auto-PTR: no reverse zone found for 192.0.2.1, 192.0.2.2
```

No PTR record is created in this case. Create the appropriate reverse zone first, then re-save the A/AAAA record.

## Example

Zone `example.com` has Auto-PTR enabled. Reverse zone `2.0.192.in-addr.arpa.` exists.

| Action                    | Result                                                                           |
| ------------------------- | -------------------------------------------------------------------------------- |
| Add `www 300 A 192.0.2.1` | Creates `1.2.0.192.in-addr.arpa. 300 PTR www.example.com.`                       |
| Change IP to `192.0.2.2`  | Deletes old PTR, creates `2.2.0.192.in-addr.arpa. 300 PTR www.example.com.`      |
| Delete `www` A record     | Deletes `2.2.0.192.in-addr.arpa.` PTR (if it still points to `www.example.com.`) |
