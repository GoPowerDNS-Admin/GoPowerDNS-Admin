# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).

## [0.6.0] - 2026-11-27

### Bug Fixes

- **zone:** Let viewers open zones read-only (#133) ([d2f55f3](d2f55f3f4b3bd80cc5ca6c57e7ae1a5ceed3fc8b))
- **auth:** Sync OIDC groups when the token has none (#143) ([5f05b0e](5f05b0e4b7f67e0e43de4a72c9ce1f2afd9304cd))
- **zone:** Check reverse zone access in Auto-PTR (#142) ([65007a8](65007a8e7a09dd4bd98f48e1e70f34fe72c426af))
- **auth:** Let group admins skip zone tag limits (#145) ([0908382](090838209336d7fadac33928820f0c51841cb816))
- Update shebang to use /bin/bash in reset.sh (#144) ([723bda1](723bda1092bb6e4b34c03f85f13b9f394cebed5e))

### Documentation

- Bring README and docs in line with the code (#135) ([26748b8](26748b890cf3d8158c26803743af2ce02dda5076))
- Mention group_roles in the README (#141) ([abf68f1](abf68f115bda991d7d6e98eff6b8cf7691c9d3d9))

### Features

- **auth:** Map groups to roles from config (#134) ([b43136e](b43136e768af57f42fea6462e457f6e811f1121d))
- **web:** Add dark mode and update frontend libraries (#146) ([de02838](de0283891ab72d5fe43085cdff070d1dcf32b4f3))
