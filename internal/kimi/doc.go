// Package kimi reads the Kimi Code subscription quota and the token usage the
// Kimi Code CLI leaves in its own session logs.
//
// The CLI, not this package, owns the credential. The only inputs are the
// Kimi Code home directory (KIMI_CODE_HOME when it is set, otherwise
// <home>/.kimi-code): the OAuth file under credentials/, the region marker and
// sessions/wd_*/session_*/agents/main/wire.jsonl. The quota comes from
// GET {base}/usages with the stored access token. A token inside its refresh
// threshold is refreshed first, and the rotated pair is written back
// atomically, because the CLI keeps its own login in that file. No token,
// refresh token or credential document is logged, returned or printed here.
//
// Two inputs cannot be verified from this machine and are handled defensively:
//
//   - used_ratio has no documented unit. The CLI forwards it without any
//     conversion, so it may be a 0-1 fraction or a 0-100 percentage. A value
//     above 1 is read as a percentage, anything else as a fraction, and both
//     are clamped to 0-100.
//   - The CLI ships no price list and its model names are subscription tiers,
//     so no per-token price is known for them. A model whose price is unknown
//     is reported as unpriced rather than priced by guesswork.
//
// The HTTP client is supplied by the caller and must come from
// netx.ClientFor(ctx, netx.WithoutCookies), so proxy, TLS and cookie policy
// stay the application's business.
package kimi

import (
	"context"
	"errors"
	"time"
)

// State is the outcome of one reading.
type State int

const (
	// StateReady is a successful reading.
	StateReady State = iota
	// StateNoCredential means the CLI never signed in on this machine.
	StateNoCredential
	// StateExpired means the stored credential is gone or was rejected, so the
	// user has to sign in with the CLI again.
	StateExpired
	// StateFailed means the request, the refresh or the response failed.
	StateFailed
)

// Report is one reading. It never carries a token.
type Report struct {
	State State
	// Err is one of the sentinel errors of this package.
	Err    error
	Base   string
	Region string
	Usages Usages
	Wallet *Wallet
	// FetchedAt is the time of the successful reading.
	FetchedAt time.Time
	// ExpiresAt is the access token expiry, in the credential's own second
	// resolution; it is zero when the file carries none.
	ExpiresAt time.Time
	HasExpiry bool
}

// Config are the inputs of one reading. Everything is optional: the home
// directory falls back to the environment, the endpoints to the region marker
// and the clock to time.Now.
type Config struct {
	// Home is the user home directory, not the Kimi Code directory.
	Home string
	// BaseURL overrides the usage endpoint; KIMI_CODE_BASE_URL wins over it.
	BaseURL string
	// OAuthURL overrides the refresh endpoint.
	OAuthURL string
	// Client sends the requests; nil is reported as a failed reading.
	Client Doer
	// Now replaces the clock in tests.
	Now func() time.Time
}

// Fetch reads the quota once. It never returns an error: every failure is a
// Report with a State and an Err, so a caller can render all four states.
func Fetch(ctx context.Context, cfg Config) Report {
	now := time.Now()
	if cfg.Now != nil {
		now = cfg.Now().UTC()
	}
	rep := Report{Base: resolveBase(cfg), Region: Region(cfg.Home)}
	creds, err := LoadCredentials(cfg.Home)
	if err != nil {
		rep.Err = err
		if errors.Is(err, ErrNoCredential) {
			rep.State = StateNoCredential
		} else {
			rep.State = StateFailed
		}
		return rep
	}
	if creds.ExpiresAt != 0 {
		rep.HasExpiry = true
		rep.ExpiresAt = time.Unix(creds.ExpiresAt, 0)
	}
	creds, err = ensureFresh(ctx, cfg, creds, now)
	if err != nil {
		rep.Err = err
		if errors.Is(err, ErrRefreshReject) {
			rep.State = StateExpired
		} else {
			rep.State = StateFailed
		}
		return rep
	}
	if creds.ExpiresAt != 0 {
		rep.HasExpiry = true
		rep.ExpiresAt = time.Unix(creds.ExpiresAt, 0)
	}
	snap, err := FetchUsages(ctx, cfg.Client, rep.Base, creds.AccessToken, now)
	if err != nil {
		rep.Err = err
		if errors.Is(err, ErrUnauthorized) {
			rep.State = StateExpired
		} else {
			rep.State = StateFailed
		}
		return rep
	}
	rep.State = StateReady
	rep.Usages = snap.Usages
	rep.Wallet = snap.Wallet
	rep.FetchedAt = snap.FetchedAt
	return rep
}

// resolveBase is Config.BaseURL, else the environment and the region marker.
func resolveBase(cfg Config) string {
	if v := trimBase(cfg.BaseURL); v != "" {
		return v
	}
	return BaseURL(cfg.Home)
}

// ensureFresh returns a token the usage request can use. A stored token inside
// its refresh threshold is exchanged for a new pair, which is written back to
// the credential file only when it really changed.
func ensureFresh(ctx context.Context, cfg Config, creds Credentials, now time.Time) (Credentials, error) {
	if !creds.NeedsRefresh(now) {
		return creds, nil
	}
	refreshMu.Lock()
	defer refreshMu.Unlock()
	// Another goroutine may have refreshed while this one waited: re-read the
	// file and keep the newer pair.
	current, err := LoadCredentials(cfg.Home)
	if err == nil && !current.NeedsRefresh(now) {
		return current, nil
	}
	file, err := readCredentialFile(CredentialsPath(cfg.Home))
	if err != nil {
		return Credentials{}, err
	}
	endpoint := trimBase(cfg.OAuthURL)
	if endpoint == "" {
		endpoint = OAuthURLFor()
	}
	tok, err := Refresh(ctx, cfg.Client, endpoint, file.RefreshToken, RequestTimeout)
	if err != nil {
		return Credentials{}, err
	}
	if err := file.apply(tok, now); err != nil {
		return Credentials{}, ErrCredential
	}
	return file.Credentials, nil
}
