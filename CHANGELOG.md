# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).

## [Unreleased]

### Dependencies

- **deps:** Bump Go dependencies to latest ([7d0c1e1](7d0c1e12d277e1d5e506063d736b111234c25bc1))

## [0.1.0-beta.2] - 2026-05-16

### Bug Fixes

- **activity-log:** Default page size to 10 rows ([79e84ff](79e84ffe0e63d3a311bf8ae81eb3f9d1db94650f))

### Dependencies

- **deps:** Bump github.com/gofiber/fiber/v3 from 3.1.0 to 3.2.0 ([98a187a](98a187a51115d21095fec538118faa75b1b5e94b))

### Documentation

- **readme:** Document contributor tooling and activity-log enhancements ([2045e5c](2045e5c5949a77784d496e91ed41bb9a4ecf4e11))

### Features

- **activity-log:** Zone filter, windowed pager, persisted page size ([8532f8e](8532f8e1171dfce9936e67615378d7307f283612))

## [0.1.0-beta.1] - 2026-04-28

### Dependencies

- **deps:** Bump github.com/go-jose/go-jose/v4 from 4.1.3 to 4.1.4 ([3736a81](3736a816a59684f203ab6cd8cc17aa3f94ea9789))
- **deps:** Bump github.com/jackc/pgx/v5 from 5.9.1 to 5.9.2 ([42bdf16](42bdf169da638da64294dcb517823b904e0a1919))
- **deps:** Bump github.com/Azure/go-ntlmssp from 0.1.0 to 0.1.1 ([33d912a](33d912a89e7b4f105c646aa182650af859a0daf3))

### Documentation

- Add docs site link, update Docker tag, document admin TOTP disable ([0337d61](0337d6101b6f984624b1ce8991bfafbedba3e33a))

## [0.1.0-alpha.11] - 2026-03-28

### Bug Fixes

- **docs:** Remove unsupported br shortcode from homepage ([271ec53](271ec53e288094919ef6f0952e8569b5b7e8cdce))
- **docs:** Ignore docs/content in prettier, fix homepage shortcodes ([1569b4d](1569b4d988130c015bb01108524f97b08fb8bff7))
- **docs:** Replace unavailable code-bracket icon with command-line ([0e95425](0e95425ef296ea5c8ac14ff6583d019afbe67dca))
- **docs:** Remove icon attributes from feature cards to avoid missing icon errors ([34d5b00](34d5b0099533942f57150fa76f0d1eb58f9828c1))
- **docs:** Build with custom domain baseURL instead of GitHub Pages subpath ([f1bcf9d](f1bcf9d8851bf4bba6e19167f26f9151efa50176))
- **docs:** Trigger workflow on changes to docs.yaml itself ([94a5b69](94a5b695bbe9c5bcda353896475b69b75faa9dcd))
- **ui:** Use absolute path for sidebar logo to fix 404 on nested routes ([8adbf9a](8adbf9ae802a8e46077a7f243d767aac0607cbe3))
- **admin:** Move disable-TOTP form out of nested form into separate card ([b9f6bcc](b9f6bcc76457f090eb2e24c4d314e7602f197051))
- **admin:** Stay on edit page after updating a user ([58e206f](58e206f3420cd1851ba3f6c7bf493e7c1988afbb))
- **admin:** Label secondary button Cancel on create, Back to list on edit ([577e9fc](577e9fc1578e7b79d305816751f6c5132e3df67e))

### Documentation

- Scaffold Hugo documentation site with Hextra theme ([40bcdc9](40bcdc9c1b881b0c72b174d7f98d823dac20cd1a))
- Document auto-PTR feature and cross-zone hint badges ([3992a0b](3992a0beffa543491f1f4dd6f096a6b82814f9cb))

### Features

- **admin:** Allow admin to disable TOTP for local users not under forced TOTP ([be141a1](be141a1d75aac97f16ecd4061cfb061689b2a552))

## [0.1.0-alpha.10] - 2026-03-28

### Bug Fixes

- **uniuri:** Widen TestBias tolerance from 5% to 10% to prevent flaky failures ([55529ef](55529ef2d927fe7ff7dd38cab363444188f64b52))

### Features

- **zone:** Auto-create PTR records for A/AAAA changes ([99fc1e0](99fc1e0524ced5655e539c1c83ee240cde23b100))

## [0.1.0-alpha.9] - 2026-03-27

### Bug Fixes

- **script:** Mask API key in curl example output ([8daf52e](8daf52e22901c80d9b5df4196bf4e99c28b499af))

### Documentation

- Add SECURITY.md with vulnerability reporting policy ([945c934](945c934eeec5f2302351af1c3689398a0e0203fc))
- Update demo site URL to demo.gopowerdnsadmin.org ([aa105cb](aa105cb4c2ae949157fe33457727833521ba1dee))
- Note demo mode restrictions on password and profile changes ([2ae9def](2ae9defa95531fc62a6184b9646b117aff135cac))

### Features

- **zone:** Warn when SOA-EDIT-API is set to OFF ([a56ebe7](a56ebe7fd3e665e6a62b4181f5f3e145d339f242))
- **login:** Display version on login page ([94bea8b](94bea8baebd1ad2141aa675d83a733c8f1714d1d))
- **login:** Add Go gopher to login page footer ([753bc37](753bc37f5b1b4acbdf71eeb1af43984612f11994))
- **demo:** Block admin user modifications in demo mode ([3b6907a](3b6907a92ddd03578b602aa03e0c832eee93ccd4))
- **login:** Replace hand-drawn gopher with official go.dev favicon gopher (CC BY 4.0) ([e0d11ef](e0d11ef4118c31ff9aa0e1786705d646d9936ecf))
- **ui:** Update page titles and login logo to GoPowerDNS-Admin ([1a16ec8](1a16ec8cd8aee0d46bd237621851a1197f7aff21))
- **ui:** Replace top-left PowerDNS logo circle with Go gopher face ([8420d8b](8420d8b5fbb1401a2824bde525a8e3aa2790d6c6))
- **ui:** Use SVG favicon with gopher logo, keep PNG as fallback ([51b1ae0](51b1ae0535d9156cac8fee38ca4c4f02a951c4cc))
- **ui:** Recolor top-left favicon circle to Go gopher blue ([eb03bfb](eb03bfb6db5c8d398f256977cf097f531e3f3f45))
- **ui:** Recolor top-left favicon circle to Go gopher blue ([4101c47](4101c47a12fbb5717c1787fb6ecd5df0a2786024))
- **ui:** Recolor top-left favicon circle to Go gopher blue (#76e1fe) ([3b36fed](3b36fed5cab4f19951a360367bb657432c88cf53))

## [0.1.0-alpha.8] - 2026-03-26

### Features

- **demo:** Disable password, TOTP, and admin role permission changes in demo mode ([7c2f4de](7c2f4deebe1cf4ffe031c23f4d61d99675afa0a0))

## [0.1.0-alpha.7] - 2026-03-26

### Bug Fixes

- **demo:** Use DEFAULT SOA-EDIT-API for demo zones ([1fd4f08](1fd4f08665ee54ac38f397e1bfb204740cfbd9d7))
- **zone:** Fix SOA modal x-model.number coercion + browser tests + Go 1.26 ([fe3265e](fe3265ecbeae6ec0f2d1ce6dad0aa90a90d06a35))

## [0.1.0-alpha.6] - 2026-03-25

### Bug Fixes

- **security:** Disable COEP header — not needed, frame-ancestors none covers embedding attacks ([591e95b](591e95b89a864baf31ebcfb157ffb1b594995051))
- **config:** Return error instead of panic when config file is missing ([109ced7](109ced77b809b6ad1c176682efd3cb9750a46ca0))
- **config:** Trust Podman container subnet in reverse proxy config ([2f06254](2f062544783c06f0ed945f2b6bbfe38272f407d8))
- **zone:** SOA save fails when serial is 0 and incremented to 1 ([683f537](683f53729d082a6d05205133bb5f17b4e06e6081))
- **ci:** Supply base branch for changelog PR when checked out at tag ([880c70a](880c70a46ec39d85e61088a660e5917a05290196))

## [0.1.0-alpha.5] - 2026-03-24

### Bug Fixes

- **toml:** Improve comments for clarity in main.toml ([9aea832](9aea832e7baf7cd56c9f64b7c6d5e9bd1653007e))
- **release:** Grant write permissions for pull requests in release workflow ([ef6a01a](ef6a01a98b4aefe91a4ba60c14a266d1c2749aaa))
- **compose:** Set pull policy to always for app service ([ad7a37f](ad7a37f7ae9ffd45a3d926f81e7f8e560ebf6b37))

### Features

- **demo:** Seed demo user and pre-populated test zones ([bd188da](bd188da4bab4f2574418a0200c8b4ca4a862d998))

## [0.1.0-alpha.4] - 2026-03-24

### Bug Fixes

- **ci:** Lowercase repository owner in Docker image references ([f253ca5](f253ca5446a0541df3c1b4a54bfaa2bb5b3ee24c))
- **logging:** Exclude health check requests from access logs ([ce86b70](ce86b70940bb263a8de0a9c8d58c9fd87d36a2f1))

### Documentation

- **readme:** Add live demo link with credentials and reset info ([0977cf4](0977cf45e471ad6e2a959873eae71e35b060f3e0))

### Features

- **web:** Add zerolog access log middleware logging method, path, status, latency, and IP ([b42b82e](b42b82eb86049085d8e1f6eba6a5a9432e7afdcb))
- **web:** Replace plain error strings with AdminLTE error page (#46) ([f175a83](f175a8321df30f473e2fc49fa57a3ff784537d6f))
- **web:** Replace plain error strings with AdminLTE error page ([71bd60b](71bd60b2e84256e8ab136b1ae73c8eb82f18b4b2))

## [0.1.0-alpha.3] - 2026-03-24

### Bug Fixes

- **deploy:** Use HOME-relative install dir for non-root compatibility ([c455e32](c455e32d2c71f7a411d7b177c4fa94936521eeeb))
- **deploy:** Use sudo for package install when non-root; qualify pdns image name ([65b438f](65b438f868b3036b72ec85f520cbe04d677b909a))
- **deploy:** Remove port 53 from pdns service; not needed for demo and fails rootless ([51d937a](51d937a14727f53680c778fb685d7c05ac0d538e))
- **deploy:** Bind pdns DNS port to 5353 on host for rootless compatibility ([5e84f15](5e84f15ab5e6c3ff38d7080a9a6d3fcb3372ee85))
- **deploy:** Install cron/cronie as part of setup dependencies ([61c19d4](61c19d4c3c249e0faf095981fc5b97d232ad08a1))
- **deploy:** Correct reverseproxy config to use [webserver.reverseproxy] section ([ce56fec](ce56fec8308be95568c28016d1aae4695f0bf860))
- **deploy:** Fix rootless podman volume permissions with podman unshare chown ([3c93211](3c932115c605e64ba9cbb9f0f81662900091627a))
- **deploy:** Configure pdns to listen on port 5353 to avoid privileged port in rootless container ([47c80a6](47c80a615d27492b3f5a8143c221763278e25f78))
- **deploy:** Use named volume for app data to avoid rootless podman UID remapping issues ([49aaa21](49aaa218534de015983834c71b477afa698b05e8))
- **deploy:** Pre-configure pdns URL and vhost in demo main.toml for auto-seed on first startup ([def71d0](def71d0e18553032c02408d89cc078cef7c1b3b9))
- **deploy:** Remove dirname cd from cleanup.sh to work when piped via curl ([99abdd3](99abdd3bf7838c8fb4d1b4fe266786625461863a))
- **deploy:** Replace unsupported --rmi all with explicit podman rm and rmi in cleanup ([ee1e8f2](ee1e8f29d98ab78533b6eaf90ca83c79b0eedd88))

### Documentation

- **readme:** Document pure-Go SQLite driver and SOA protection ([8661203](866120366d8e64349515e4037586b75386f68627))

### Features

- Replace CGO SQLite with pure-Go driver (#42) ([0bdee0a](0bdee0a6aa90a6271df009d6d1214d718287bdea))
- **deploy:** Add Fly.io and VPS demo deployment with auto-reset (#44) ([7b01ff9](7b01ff948e463c233f2022302284f276c411cd99))
- **deploy:** Make demo setup.sh curl-pipeable with interactive prompts ([928d5b5](928d5b5cbe68f7014cb5d646d5eb806e68a09691))
- **deploy:** Add PowerDNS service to demo compose; auto-generate PDNS key ([672c756](672c7564a10f703ff1fcceba7277d6d82c0030ba))
- **deploy:** Enable auto-start on reboot via systemd service ([f8d0d10](f8d0d10f158e1fd47f143b2da4af64f4c42e188f))
- **deploy:** Add cleanup.sh to remove all demo containers, volumes, and config ([7f1e24e](7f1e24e3f6bfb0e675789254fe211877196e1845))

## [0.1.0-alpha.1] - 2026-03-23

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
- **ui:** Show only version as link in footer, remove redundant text ([d4c1469](d4c146919d4e36c3edb90d4eaa050b1f935c3022))
- **docker:** Replace vendor copy with go mod download for CI compatibility ([fd74c8f](fd74c8f0c4c306a082472f65c03ff51779665d54))
- **csp:** Eliminate inline scripts and event handlers to comply with CSP (#37) ([7653a52](7653a5233f5fd9a5508e99a021cba73ebf73f0ea))
- **zone:** Allow editing existing records of disallowed types and protect SOA (#38) ([922d5fb](922d5fb4ac0868dedf376a122c6931b8c51c3cd7))
- **release:** Pull --rebase before pushing CHANGELOG to avoid rejection ([3f94bbe](3f94bbe5cd3093840437e79fe84534c1a7a8644a))
- **release:** Open PR for CHANGELOG instead of pushing directly to main ([b86b8af](b86b8af2f2e383b090f1637e53efa15d0d22d3ab))
- **release:** Force-push changelog branch to handle re-runs ([b5b38b6](b5b38b6d034531161cd13fcdb8e7f631ca921d02))
- **release:** Delete changelog branch before pushing to avoid workflow permission error ([5491977](549197703b03485c62ec5ae6ee313b02b13ee025))

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
- **readme:** Add version flag, make build, and Docker registry info ([f67be7a](f67be7a30b276945c153f5746b3a6e1d94eb1e3e))

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
- **release:** Add multi-platform Docker image build and push to ghcr.io ([9c90f7f](9c90f7fe43596ab41f26bbae1204ebb2bf2a834d))
- **cli:** Add --version / -v flag to show application version ([ee70619](ee70619545744c68622e2f7e3c695ec68caf7814))
- **ui:** Show app version in footer ([83fd8a7](83fd8a7dc8a90c2809d43422a303206e42e53e33))
- **version:** Fall back to VCS commit hash when version not set via ldflags ([6049634](60496342f65ce53699946cbd92a940bf969f6671))
- **version:** Include branch name in version string ([c3520b5](c3520b583d9618d1dbbdff972a5cef0e45e48a6e))

### Refactoring

- **auth/oidc:** Replace if-else chain with switch for DB error handling ([9e0c473](9e0c473a18dd182a65f6ba76681d8a7260421336))
- **zone/add:** Extract resolveZoneName and createZone helpers to reduce cyclomatic complexity ([98dbff1](98dbff1e2bdda3e427270780f71ac05f9d817fc9))
- **zone/add:** Split add.go into types.go and create.go ([2a64a6d](2a64a6d64b06afb3a8ce6d8c6ee5d8357969ebb3))
