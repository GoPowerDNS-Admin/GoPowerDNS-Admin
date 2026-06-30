---
title: Branding
weight: 6
prev: /docs/administration/group-mappings
---

Branding lets you replace the product name, logo, and favicon shown in the
sidebar, login, and TOTP pages — useful when running GoPowerDNS-Admin as part of
a larger platform.

## Editing branding

Navigate to **Admin → Settings → Branding**. You can set:

| Field         | Notes                                                                  |
| ------------- | ---------------------------------------------------------------------- |
| **Name**      | Product name shown in the UI; falls back to the configured `Title`     |
| **Logo**      | Sidebar/login logo (SVG or PNG); no aspect-ratio constraint            |
| **Favicon (SVG)** | Primary favicon; must be **square**                                |
| **Favicon (PNG)** | Fallback favicon for browsers without SVG favicon support; must be **square** |

Uploaded images are limited to **1 MiB** each. Runtime settings configured here
take precedence over the `[branding]` values in `main.toml`.

## Setting branding in config

You can also set branding in `main.toml` (see [Configuration](/docs/getting-started/configuration#branding-optional)).
Any field left unset falls back to a default — the name to `Title`, the images to
the bundled PowerDNS logo.

```toml
[branding]
Name          = "Acme DNS"
LogoURL       = "/static/img/your-logo.svg"
FaviconURL    = "/static/img/your-favicon.svg"
FaviconPNGURL = "/static/img/your-favicon.png"
```

## Same-origin image constraint

{{< callout type="warning" >}}
The Content-Security-Policy restricts images to same-origin (`'self'`) and
`data:` URIs. Custom images **must** be served from the same origin — drop a file
into the mounted `static/img` directory and reference it as
`/static/img/your-logo.svg`, or supply a `data:` URI. External image URLs are
blocked by the CSP. Uploads made through the admin UI are served same-origin
automatically.
{{< /callout >}}
