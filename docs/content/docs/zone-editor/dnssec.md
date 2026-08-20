---
title: DNSSEC
description: "Enable DNSSEC signing, manage cryptokeys, view DS records with digest-type guidance, and verify the full trust chain from root to zone in GoPowerDNS-Admin."
weight: 4
prev: /docs/zone-editor/auto-ptr
---

DNSSEC signing is managed from the **DNSSEC** card on the zone editor page. The card is available for `Native` and `Master` zones.

## Enabling DNSSEC

1. Open the zone in the editor.
2. Expand the **DNSSEC** card.
3. Click **Enable DNSSEC**.

PowerDNS generates a Combined Signing Key (CSK) using the server's default algorithm (ECDSAP256SHA256 unless overridden in `pdns.conf`). The key is immediately active and published.

## Cryptokey table

Once DNSSEC is enabled the card shows a table of all cryptokeys for the zone:

| Column        | Description                                                  |
| ------------- | ------------------------------------------------------------ |
| **ID**        | Internal PowerDNS key identifier                             |
| **Type**      | `csk` (Combined Signing Key), `ksk` (Key Signing Key), or `zsk` (Zone Signing Key) |
| **Algorithm** | Signing algorithm reported by PowerDNS (e.g. ECDSAP256SHA256) |
| **Bits**      | Key length in bits                                           |
| **Active**    | Whether the key is used for signing                          |
| **Published** | Whether the DNSKEY record is included in the zone            |
| **DS Records**| Opens the DS records modal for this key                      |

The **⋮** action menu on each row lets you activate/deactivate, publish/unpublish, or delete a key.

## DS records modal

Click **DS** on a key row to open the DS records modal. PowerDNS generates one DS record per supported digest type from the same DNSKEY. Each DS record is shown with:

- A **badge** indicating the digest type and algorithm:
  - `Type 1 — SHA-1` (orange) — deprecated, avoid submitting to registrar
  - `Type 2 — SHA-256` (green, recommended)
  - `Type 4 — SHA-384` (blue) — stronger but rarely required by registries
- A **description** explaining whether to use it
- A **copy button** to copy the full DS record string to the clipboard

### Best practice

Submit only the **SHA-256 (type 2)** DS record to your registrar. SHA-1 is considered cryptographically weak and most registries accept or prefer SHA-256 exclusively. Submitting both is harmless, but unnecessary. Never submit SHA-384 unless your registry explicitly requires it.

### Publishing DS records at the registrar

After copying the DS record, log in to your domain registrar and enter the values in the DNSSEC / DS records section:

| Field           | Where to find it              |
| --------------- | ----------------------------- |
| **Key tag**     | First number in the DS string |
| **Algorithm**   | Second number (13 = ECDSAP256SHA256) |
| **Digest type** | Third number (2 = SHA-256)    |
| **Digest**      | Hex string at the end         |

The registrar submits the DS record to the parent TLD registry. Propagation typically takes a few minutes to a few hours.

## Trust chain verification

Once DNSSEC is enabled, a **Verify Trust Chain** button appears below the cryptokey table. Clicking it performs a full chain walk from the DNS root down to the zone, querying authoritative nameservers directly (bypassing resolvers and caches).

For each delegation level the result shows:

| Status | Meaning |
| ------ | ------- |
| ✅ `ok` | DS record found in the parent zone |
| ❌ `missing` | No DS record found — the chain is broken at this level |
| ⚠️ `error` | A DNS query failed (nameserver unreachable, timeout, etc.) |

A green banner confirms a valid chain. A red banner identifies the exact level where the chain is broken, along with the reason.

### Common failure: DS not published at registrar

The most frequent failure is a `missing` DS at the TLD level (e.g. `.cloud`, `.com`). This means DNSSEC is correctly configured in PowerDNS but the DS record has not yet been submitted to the registrar, or has not yet propagated to the TLD nameservers.

**Resolution:** copy the SHA-256 DS record from the DS records modal and submit it to your registrar. After propagation, click **Verify Trust Chain** again to confirm.

## Disabling DNSSEC

Click **Disable DNSSEC** in the DNSSEC card. This deletes all cryptokeys for the zone. Validators will stop verifying the zone once their caches expire.

{{< callout type="warning" >}}
Remove the DS records from your registrar **before** disabling DNSSEC. If the DS records remain in the parent zone after the keys are deleted, validators will return SERVFAIL for the zone until the DS TTL expires.
{{< /callout >}}
