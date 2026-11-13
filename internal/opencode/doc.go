// Package opencode reads the OpenCode CLI's quota and recorded cost.
//
// The inputs are two vendor endpoints and one local database, nothing else:
//
//	GET https://opencode.ai/zen/go/v1/usage
//	GET https://opencode.ai/console/api/policies/current-budgets
//	<data>/opencode.db, opened read only (tables credential and session_v2)
//
// The database is never written: the CLI may have it open, so the panel reads
// through the same read-only path the CLI itself uses. The access token stays
// unexported and is never logged, printed, or returned through an exported
// field. This package also never refreshes the token: a second refresher would
// race the CLI's own rotation and invalidate the login, so an expired
// credential is reported as expired and the panel asks for a new login.
//
// HTTP 403 means the organization has that feature switched off, which is
// reported as "no data" and not as a failure; 401 means the stored login is no
// longer accepted. Percentages are clamped to 0-100 and a field that is absent
// is reported as absent rather than as zero.
package opencode
