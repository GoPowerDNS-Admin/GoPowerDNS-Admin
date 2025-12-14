// Package zoneedit provides HTTP handlers and helper utilities for editing
// existing DNS zones in the GoPowerDNS-Admin web UI.
//
// Overview
//   - Renders the Edit Zone page with navigation breadcrumbs and current zone data.
//   - Loads zone metadata (kind, SOA-EDIT-API, masters) and DNSSEC state from PowerDNS.
//   - Displays and updates RRsets with support for:
//   - TXT/SPF quoting normalization and validation.
//   - URI record content normalization per RFC 7553 (priority, weight, target).
//   - Record comments and enabled/disabled state handling.
//   - Filtering of allowed record types based on application settings.
//   - Persists changes via the shared PowerDNS engine and API client.
//
// Primary entrypoints
//   - (*Service).Get: renders the edit form for a given zone.
//   - (*Service).Post: updates general zone properties (kind, SOA-EDIT-API, masters).
//   - (*Service).PostRecords: applies record (RRset) changes.
//
// Conventions and helpers
//   - Zone names are treated as fully-qualified (with a trailing dot); see normalizeZoneName.
//   - Display names for records omit the zone suffix; see getDisplayNameForZone.
//   - SOA-EDIT-API values are extracted with safe defaults; see getSOAEditAPIFromZone.
//   - Quoted string validation and normalization for record content is provided by
//     isQuotedStringSequence and ensureQuotedContent.
//
// Templates & routing
//   - The package uses the "zone/edit" template for rendering the edit page.
//   - Routes typically mount the handler at /zone/edit/:name.
package zoneedit
