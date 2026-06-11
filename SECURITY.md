# Security Policy

## Supported Versions

GoPowerDNS-Admin is currently in pre-1.0 development. Only the latest release receives security fixes.

| Version | Supported |
| ------- | --------- |
| latest  | yes       |
| older   | no        |

## Reporting a Vulnerability

Please **do not** open a public GitHub issue for security vulnerabilities.

Report vulnerabilities privately using [GitHub's private vulnerability reporting](https://github.com/GoPowerDNS-Admin/GoPowerDNS-Admin/security/advisories/new).

Include as much detail as possible:

- Description of the vulnerability and its potential impact
- Steps to reproduce or proof-of-concept
- Affected version(s)
- Any suggested mitigations (optional)

You can expect an acknowledgement within **72 hours** and a status update within **7 days**. If a fix is warranted, a patched release will be issued and a security advisory published.

## Scope

In scope:

- Authentication and session handling (local, LDAP, OIDC, TOTP)
- Authorization bypass or privilege escalation (RBAC, permission checks)
- Injection vulnerabilities (SQL, template, command)
- Cross-site scripting (XSS) or cross-site request forgery (CSRF)
- Sensitive data exposure (credentials, tokens, API keys)
- DNS-related logic that could allow unauthorized zone or record manipulation

Out of scope:

- Vulnerabilities in PowerDNS itself (report those upstream to the [PowerDNS project](https://github.com/PowerDNS/pdns/security))
- Issues that require physical access to the server
- Denial-of-service attacks requiring no authentication
- The live demo instance (it resets daily and is intentionally open)

## Security Considerations for Deployment

- Run behind a reverse proxy (HAProxy, nginx, Traefik) and restrict access to trusted networks where possible.
- Use TLS — either native (cert/key) or automatic via Let's Encrypt/ACME.
- Set `trusted_proxies` in your config if running behind a proxy to avoid IP spoofing.
- Change the default admin password immediately after first login.
- Enable TOTP two-factor authentication for admin accounts.
- Limit database user privileges to only the GoPowerDNS-Admin schema.
