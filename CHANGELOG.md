# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).

## [Unreleased]

### Bug Fixes

- **workflow:** Update PR title check workflow configuration ([b5f5dc2](b5f5dc2bf40d74111e08e9990197bc0809d2c773))
- **workflow:** Update runner configuration for PR title validation ([9529781](95297816ce7e855d35f26d42681cd3a5a1f8bb27))
- **zone:** Replace deprecated VisitAll with All iterator (#4) ([c937cfc](c937cfc379ea1ac440347a817bc098bb23206357))
- **zone:** Correct record rename deletes old RRset (#14) ([28c243c](28c243c7ee023c923157ea425668a02df199a0f3))
- **ci:** Replace deprecated macos-13 runner with universal Clang cross-compile ([a733ff9](a733ff981c9f6a77be4623164e9c594b23a5bf64))
- **ci:** Fix FreeBSD arm64 arch name and update vmactions to Node.js 24 ([973a78a](973a78a499681815e46d2538210c60c28369416b))
- **ci:** Upgrade actions to Node.js 24 to resolve Node.js 20 deprecation ([2520b85](2520b85fd9283292532e148c20d1e0b21e03be2a))
- **ui:** Clean up header navbar and fix mobile sidebar behavior (#26) ([dd90399](dd90399bb5facb6160512ecff3841170d0b8b8ca))
- **ui:** Fix zone settings card collapse toggle (#27) ([7e9cde7](7e9cde7a1a955b26d077650f0883f3e8ae70123e))
- **zonetag:** Sort zones by name ([a6258c1](a6258c1bc2e5b9f695df07e99f628ad46f3afc83))
- **zonetag:** Default page size to 10 ([0aedc74](0aedc748e12045f4c6f459e8f7a85130f2d96659))
- **ui:** Use full width on TTL presets settings page ([ec99b01](ec99b01890b52aa62db5e592b072783c8f98f363))
- **ui:** Fix favicon path in base layout ([37c801d](37c801dd21dac4dc4fc3d22fe9b6be9defb95ef6))

### Dependencies

- **deps:** Update dependencies in go.mod and go.sum (#3) ([1673016](1673016b567e6eb01fb58c6134c9efc92ceb0cb6))
- **deps:** Vendor CDN dependencies into static assets (#9) ([93e7990](93e799057ccaa40c067ff6b06fa9406004d50211))
- **deps:** Bump dependencies to latest versions (#30) ([3423b03](3423b035aa8152d7b1099d082a5807089f81dac9))
- **deps:** Bump dependencies to latest versions (#31) ([9730399](973039955a9218437a3c2c3f469b99862bf1d57d))

### Documentation

- Update README to reflect current features and local config setup (#21) ([6f293a8](6f293a8a5e8a5aa4eac0a3a014c0e0151a42b7b6))
- Add Buy Me a Coffee badge and FUNDING.yml ([9934746](9934746328f34f785536420e071b5de9d98ad98a))
- Update README for zone editor, TTL presets, roles, and activity log improvements ([8d77384](8d7738409046f7e00c62b2f82bacbb1d5144988c))
- Add Zone Tag Access Control section to README ([93ffde3](93ffde3e2274dc6cfeba520d788d6a3bec3a21fb))
- **readme:** Update status to alpha, add security headers feature ([11acf4d](11acf4d3672210e9dba1e19d652ed1d82afd6b80))

### Features

- Add core web application with PowerDNS admin interface ([d4731e6](d4731e64e5986b0e37679d1e9791d8b0ed27bb67))
- Implement zone record settings management with CRUD operations ([b80b456](b80b456bf079516cec60c11a05fa69fc2c048cc4))
- Add server configuration management with pagination and filtering ([fe737d8](fe737d8379eb588a59862c4454f6f268b9fe45cb))
- Refactor file structure and add zone creation functionality ([32b20ff](32b20ff663b26d9c005fc386c18134173e55ed55))
- Update .gitignore to exclude build and binary directories ([d480218](d480218e3b7bb5c2ff981aab57f4949e83e6da0a))
- **docs:** Add “Background & Inspiration” to README referencing PowerDNS-Admin ([b784186](b78418698fd80ec1e53393b9a32ff9ed942ba144))
- **zone:** Add record type validation and loading functionality ([45ba24c](45ba24c50641546e3f74ad92b53e1a7a5f99ccfe))
- **docs:** Update README to reflect implementation status of features ([b12ca19](b12ca1928379560d9a1a3b83c84423919c7810b5))
- **user:** Add first name and last name fields to user forms and listings ([caf50cb](caf50cba30ca0376f9106a8ddb9cf2c5afc7f920))
- **logout:** Implement logout handler and integrate with main application ([9477d89](9477d8937e604eefa17c00b3822e0a68c00576fb))
- **logout:** Implement logout handler and integrate with main application ([347cc7e](347cc7e544d8d29e7db3f55ee33af244fb96ba25))
- **tests:** Add test command to Makefile and update form data in tests (#1) ([1718250](1718250f46691ac58179ca13eb02a9588ab77f29))
- **activity-log:** Add admin activity log with zone diff tracking (#6) ([1894a02](1894a02ed7f926a9988ad72702792fe8af42028c))
- **activity-log:** Add undo record changes action (#7) ([0d6f92c](0d6f92c188858e5c8006331db621299f6747cb72))
- **activity-log:** Allow user and viewer roles to view activity log (#8) ([faad731](faad731cc7b345333b51e71e8bf7374d3210e58c))
- **activity-log:** Add undo zone delete action (#10) ([c4a0459](c4a0459eaf8a777b3114653c4df6739b1c03f775))
- **deps:** Upgrade Fiber v2 to v3 across the codebase (#11) ([b66b3e6](b66b3e6d9abbc86fed3eaf621c56438e90322f1a))
- **oidc-auth:** Complete OIDC authentication support (#15) ([18ef5d8](18ef5d8d1f47a8940589cc42f4b18346a927cdc9))
- **rbac:** Add role CRUD, user profile, zone tag access control (#16) ([c9cd085](c9cd08571a911d6508af79b93eda164f62309305))
- **order:** Replace hardcoded order clauses with constants for user and role queries ([9cbe4a8](9cbe4a897d79f4c8d284a849b9e066291fa8bb85))
- **ldap:** Add LDAP docker-compose setup and fix several code quality issues (#17) ([2110ccf](2110ccfa1b54b12e9b31213e0d5d5eb61a669ffa))
- **ci:** Add nightly build workflow for Linux, macOS, and FreeBSD ([915ee32](915ee327b8e7265559bf526749c692778e726bd7))
- **db:** Add PostgreSQL support (#18) ([bdeb364](bdeb36496b9aa55ffa1aacb750a0068cee5966b8))
- Add SQLite database support (#20) ([454eade](454eade69b55ddf041f0585cdfcc63f4ef76e674))
- **dnssec:** Filter DNSSEC-managed records from zone editor and dashboard (#22) ([3d1fe20](3d1fe20b01207c477a9255bc705c559162818fa9))
- **ui:** Overhaul zone edit/add templates with AdminLTE4 patterns (#24) ([8cea564](8cea564c9dc4e970acef56e19ab4a13f5241c298))
- **auth:** Add TOTP two-factor authentication for local users (#25) ([bf31adc](bf31adc4cda749da5a3548f7c65efa0bb7173575))
- **ui:** Improve zone editor, activity log, roles, and add TTL presets (#28) ([9bc79a1](9bc79a12e38ecaa1780ab35d030d41d6e9d5c6df))
- **zonetag:** Add Alpine.js client-side pagination and search to zone-tag list ([e0b1123](e0b1123837ab63e15dbefc9df49d561f77765416))
- **zonetag:** Add rows-per-page selector to zone-tag list ([cf874c6](cf874c63b5d4760a279b03eb2df39a63a43c8aa7))
- **health:** Add /health endpoint for liveness and readiness probes (#32) ([281f5b5](281f5b5c94d7ee6f162f8f22a554034122966ca0))
- **config:** Add startup validation for secrets, DB, and auth providers (#33) ([49ef155](49ef155709ad696dbb4abd0ac23c8fd328fc7fc3))
- **tls:** Add native TLS and Let's Encrypt / ACME support (#34) ([d8bd169](d8bd1690bf2e21f184bf3c06dbac06f704fc5d0a))
- **proxy:** Add reverse proxy support for HAProxy, nginx, and Traefik (#35) ([9f5eddb](9f5eddbb9dff71270134537023bd6c673ec47ce7))
- **docker:** Add Dockerfile with non-root user and Makefile targets (#36) ([048a4f2](048a4f255c088d12dd2c12cf106fd35c34eab515))
- **security:** Add security headers via helmet middleware ([ed9b68a](ed9b68a0d7b8f616e2ddaccaa7482b2bcbaf91a8))
- **release:** Add release workflow, CHANGELOG, and wire BrowseStatic config ([455b66b](455b66b3d9759682acc26763079d19bbbe2bc577))

### Refactoring

- **auth/oidc:** Replace if-else chain with switch for DB error handling ([9e0c473](9e0c473a18dd182a65f6ba76681d8a7260421336))
- **zone/add:** Extract resolveZoneName and createZone helpers to reduce cyclomatic complexity ([98dbff1](98dbff1e2bdda3e427270780f71ac05f9d817fc9))
- **zone/add:** Split add.go into types.go and create.go ([2a64a6d](2a64a6d64b06afb3a8ce6d8c6ee5d8357969ebb3))
