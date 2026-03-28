---
title: GoPowerDNS-Admin
layout: hextra-home
---

{{< hextra/hero-badge >}}
<div class="hx-w-2 hx-h-2 hx-rounded-full hx-bg-primary-400"></div>
<span>Alpha — interfaces may change</span>
{{< icon name="arrow-right" attributes="height=14" >}}
{{< /hextra/hero-badge >}}

<div class="hx-mt-6 hx-mb-6">
{{< hextra/hero-headline >}}
Modern DNS management for PowerDNS
{{< /hextra/hero-headline >}}
</div>

<div class="hx-mb-12">
{{< hextra/hero-subtitle >}}
GoPowerDNS-Admin is a web-based administration UI for PowerDNS.
Manage zones, records, and users from a clean Go-powered interface.
{{< /hextra/hero-subtitle >}}
</div>

<div class="hx-mb-6">
{{< hextra/hero-button text="Get Started" link="docs/getting-started" >}}
{{< hextra/hero-button text="Live Demo" link="https://demo.gopowerdnsadmin.org/login" style="secondary" >}}
</div>

<div class="hx-mt-6"></div>

{{< hextra/feature-grid >}}
{{< hextra/feature-card title="Zone & Record Management" subtitle="Create, edit, and delete DNS records with a reactive UI. No full page reloads." >}}
{{< hextra/feature-card title="Auto-PTR" subtitle="Automatically manage PTR records in reverse zones when A/AAAA records change." >}}
{{< hextra/feature-card title="Multi-Auth" subtitle="Local accounts with TOTP, OpenID Connect, and LDAP — enable any combination." >}}
{{< hextra/feature-card title="RBAC" subtitle="Fine-grained role-based access control with zone tag restrictions per user or group." >}}
{{< hextra/feature-card title="Activity Log & Undo" subtitle="Every change is recorded with a before/after diff. Undo record changes and zone deletes." >}}
{{< hextra/feature-card title="Pure Go" subtitle="No CGO, no C toolchain required. SQLite, MySQL/MariaDB, and PostgreSQL supported." >}}
{{< /hextra/feature-grid >}}
