// Package glm reads the GLM coding plan quota from the vendor monitor
// endpoint and prices the local ZCode CLI usage database.
//
// The credential comes from the ZCode CLI store (CredentialPath), or from
// ZCODE_BIGMODEL_USAGE_API_KEY / BIGMODEL_USAGE_API_KEY, which win over the
// store exactly as they do in the CLI. The store is read-only: ZCode keeps
// no refresh token and its token endpoint has no refresh grant, so an
// expired credential is reported as ErrUnauthorized and the user signs in
// again. A credential is never logged, never part of an error and never
// reaches a value that formats itself: Candidate prints itself redacted.
//
// The endpoint is GET <host>/api/monitor/usage/quota/limit with the raw
// credential in the authorization header. It must not carry a "Bearer "
// prefix: the monitor API wants the credential verbatim, and a prefixed one
// is rejected with the same message as an expired token, which makes the
// mistake hard to diagnose.
//
// Two response fields are derived from the ZCode CLI source and not from a
// verified success response, because no credential of the machine that
// produced that source was valid:
//
//   - percentage is the used share from 0 to 100, not a 0..1 fraction;
//   - nextResetTime is milliseconds since the epoch.
//
// Both are handled conservatively. A share outside 0..100 is clamped, a
// missing one is derived from remaining and total, and an epoch is scaled
// by magnitude (see UnixTime) so a seconds value is corrected instead of
// landing near 1970. Unknown fields are ignored, a missing field never
// panics, and usageDetails is not read because its unit is not verified.
//
// The cost side prices model_usage of the ZCode database with the models.dev
// rates. Cache reads are a separate column and are priced as cache reads;
// reasoning tokens are already inside the output count. A model without a
// rate, such as the *-highspeed variants whose upstream cost is zero, is
// reported as unpriced rather than free.
//
// Nothing here calls a write endpoint. POST /api/v1/coding-plan/reset, which
// resets the quota period, is out of scope by design.
package glm
