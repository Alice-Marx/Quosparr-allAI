// Package gemini reads the Gemini Code Assist quota and the token spend of
// the local Gemini CLI.
//
// Two unrelated sources make up one reading:
//
//   - The quota comes from the vendor endpoints
//     (cloudcode-pa.googleapis.com, v1internal:loadCodeAssist and
//     v1internal:retrieveUserQuota) with the OAuth credential the Gemini CLI
//     already stored at <home>/.gemini/oauth_creds.json. When that credential
//     is expired this package refreshes it and writes it back atomically,
//     keeping every field it does not own, because the CLI rereads the file.
//     The refresh grant needs the OAuth client the CLI was built with; that
//     pair is published inside the CLI, but this package carries no half of it
//     and reads both from EnvClientID and EnvClientSecret. Without them no
//     refresh is attempted and the reading says so, because a credential the
//     panel cannot renew is the CLI's to renew: running the CLI once rewrites
//     the file.
//   - The cost comes from the CLI's local chat logs under
//     <home>/.gemini/tmp/<project>/chats/session-*.jsonl. Those logs record
//     the token counts of each answer; the price table lives in this package.
//
// No token, API key or complete credential document is logged, returned in an
// error, or written anywhere but the credential file the CLI owns.
//
// A missing subscription is a normal state, not an error: a free tier the
// service no longer accepts answers retrieveUserQuota with 403
// SUBSCRIPTION_REQUIRED, and the panel draws that as its own state.
//
// Unverified semantics, handled conservatively here:
//
//   - The 200 body of retrieveUserQuota has not been observed on this machine
//     (the account is blocked by 403). The bucket fields follow the CLI's own
//     consumer code: a bucket without remainingFraction keeps its request
//     count but shows no percentage, and a bucket without remainingAmount
//     takes the CLI's fallback limit of 100.
//   - The free-tier window length, the requests-per-minute limit and the daily
//     numbers are not in the CLI source. Only what the service returns (a
//     fraction and a reset time) is shown.
//   - The per-million-token prices are published rates from outside the CLI
//     source. A model missing from the table is reported as unpriced, and a
//     model matched only by family is flagged as an estimate.
//   - The project id can come from GOOGLE_CLOUD_PROJECT(_ID), from a project
//     field in the CLI settings file, or from loadCodeAssist. Without one the
//     reading is the no-subscription state, because retrieveUserQuota answers
//     403 rather than 400 without a project.
//   - ~/.gemini/projects.json maps a workspace path to the directory name of
//     its chat logs. It is not a Cloud project and is never used as one.
package gemini

import (
	"context"
	"errors"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const (
	// DirName is the Gemini CLI directory under the user's home.
	DirName = ".gemini"
	// CredsFileName is the OAuth credential the CLI writes and refreshes.
	CredsFileName = "oauth_creds.json"
	// SettingsFileName is the CLI settings file.
	SettingsFileName = "settings.json"
	// TempDirName holds one directory per workspace with the chat logs.
	TempDirName = "tmp"
	// ChatDirName is the chat-log directory inside a workspace directory.
	ChatDirName = "chats"
	// SessionFilePrefix is the name prefix of one chat log.
	SessionFilePrefix = "session-"
	// HomeEnvVar replaces the home directory the CLI itself resolves. The CLI
	// applies it to the home, so DirName is still appended.
	HomeEnvVar = "GEMINI_CLI_HOME"
	// maxCredsBytes bounds a small JSON file this package reads.
	maxCredsBytes int64 = 256 * 1024
)

// Dir returns the Gemini CLI directory for home, following the CLI's own home
// override.
func Dir(home string) string {
	if configured := strings.TrimSpace(os.Getenv(HomeEnvVar)); configured != "" {
		home = configured
	}
	return filepath.Join(home, DirName)
}

// CredsPath is <dir>/oauth_creds.json under home.
func CredsPath(home string) string { return filepath.Join(Dir(home), CredsFileName) }

// SettingsPath is <dir>/settings.json under home.
func SettingsPath(home string) string { return filepath.Join(Dir(home), SettingsFileName) }

// TempDir is <dir>/tmp under home, the parent of every chat-log directory.
func TempDir(home string) string { return filepath.Join(Dir(home), TempDirName) }

// ChatsDir is the chat-log directory of one workspace directory name.
func ChatsDir(home, workspace string) string {
	return filepath.Join(TempDir(home), workspace, ChatDirName)
}

// FormatPercent renders a 0–100 value with at most one decimal place.
func FormatPercent(v float64) string {
	rounded := math.Round(v*10) / 10
	if rounded == math.Trunc(rounded) {
		return strconv.FormatFloat(rounded, 'f', 0, 64)
	}
	return strconv.FormatFloat(rounded, 'f', 1, 64)
}

// Status is the outcome of one reading. Only StatusError is a failure; the
// other states are normal answers the panel draws without an error style.
type Status int

// Reading states.
const (
	// StatusUnknown means no reading has completed yet.
	StatusUnknown Status = iota
	// StatusReady means the quota buckets were read.
	StatusReady
	// StatusNoSubscription means the account or this client may not use the
	// quota endpoint. The service answered, so this is not a reading error.
	StatusNoSubscription
	// StatusNoCreds means the Gemini CLI has no OAuth credential to read.
	StatusNoCreds
	// StatusExpired means the stored access token is gone and this package
	// could not renew it: either the refresh token was refused, or no OAuth
	// client is configured so no refresh was attempted. Both are answered by
	// the CLI renewing the file, so neither is a reading error.
	StatusExpired
	// StatusError means the reading failed: network, HTTP, or an unreadable
	// answer.
	StatusError
)

// Reading is one snapshot of the Gemini tab. It never holds a token.
type Reading struct {
	Status    Status
	FetchedAt time.Time
	// Tier is what loadCodeAssist reported, when it answered.
	Tier Tier
	// Project is the Cloud project the quota was asked for.
	Project string
	// Buckets are the per-model quota buckets, worst first.
	Buckets []Bucket
	// Cost is the token spend of the local chat logs. It is read whether or
	// not the endpoints answered.
	Cost Ledger
	// Reason is the service reason code (for example SUBSCRIPTION_REQUIRED)
	// already reduced to letters, digits, underscore and dash. It is empty
	// when the service gave none.
	Reason string
	// Err is the sentinel behind StatusExpired and StatusError.
	Err error
}

// Ready reports whether the reading carries quota buckets.
func (r Reading) Ready() bool { return r.Status == StatusReady }

// Options tunes one reading. Every field has a default.
type Options struct {
	// HTTP replaces the shared netx client. Tests use a local transport.
	HTTP *http.Client
	// Now is the clock; nil uses time.Now.
	Now func() time.Time
	// Endpoint and Version override the Code Assist base URL and API version.
	Endpoint string
	Version  string
	// Timeout bounds one HTTP round trip; zero uses 15s.
	Timeout time.Duration
	// RefreshSkew refreshes a credential that expires within it; zero uses
	// five minutes.
	RefreshSkew time.Duration
	// TokenEndpoint overrides the OAuth token endpoint. Only tests set it.
	TokenEndpoint string
}

func (o Options) now() time.Time {
	if o.Now != nil {
		return o.Now()
	}
	return time.Now()
}

func (o Options) timeout() time.Duration { return orDefault(o.Timeout, 15*time.Second) }

func (o Options) skew() time.Duration { return orDefault(o.RefreshSkew, 5*time.Minute) }

func orDefault(d, fallback time.Duration) time.Duration {
	if d > 0 {
		return d
	}
	return fallback
}

// Fetch reads the credential, refreshes it when needed, asks the Code Assist
// endpoints for the quota, and scans the local chat logs for token spend. It
// returns no error: every outcome is a Status, so the panel can draw a state
// instead of a failure.
func Fetch(ctx context.Context, home string, opt Options) Reading {
	base := Reading{FetchedAt: opt.now(), Cost: ScanCost(home)}
	if strings.TrimSpace(home) == "" {
		base.Status = StatusNoCreds
		return base
	}
	f := fetcher{home: home, opt: opt}
	return f.run(ctx, base)
}

type fetcher struct {
	home string
	opt  Options
}

func (f *fetcher) run(ctx context.Context, base Reading) Reading {
	creds, err := LoadCreds(CredsPath(f.home))
	switch {
	case errors.Is(err, ErrMissing):
		base.Status = StatusNoCreds
		return base
	case err != nil:
		base.Status, base.Err = StatusError, err
		return base
	}
	if err := f.ensureToken(ctx, &creds); err != nil {
		base.Err, base.Status = err, refreshStatus(err)
		return base
	}
	reading, unauthorized := f.attempt(ctx, &creds)
	if unauthorized {
		// The access token was refused even though it did not look expired.
		// One forced refresh is the whole retry budget: a second 401 means
		// the sign-in is gone, not that the clock drifted.
		if err := f.refresh(ctx, &creds); err != nil {
			base.Err, base.Status = err, refreshStatus(err)
			return base
		}
		reading, _ = f.attempt(ctx, &creds)
	}
	reading.Cost = base.Cost
	reading.FetchedAt = base.FetchedAt
	return reading
}

// refreshStatus maps a failed refresh onto a state. A sign-in this package
// cannot renew is the expired state, which the panel explains with an action;
// only a transport or server failure is an error.
func refreshStatus(err error) Status {
	switch {
	case errors.Is(err, ErrRefreshRejected),
		errors.Is(err, ErrNoRefreshToken),
		errors.Is(err, ErrNoClientSecret):
		return StatusExpired
	}
	return StatusError
}

// attempt asks both endpoints once. The bool reports a 401, which the caller
// answers with one forced refresh.
func (f *fetcher) attempt(ctx context.Context, creds *Creds) (Reading, bool) {
	out := Reading{FetchedAt: f.opt.now()}
	client := f.client(creds)
	project := f.project()
	tier, err := client.LoadCodeAssist(ctx, project)
	switch {
	case err != nil && isUnauthorized(err):
		return out, true
	case err != nil && project == "":
		// Without an answer from loadCodeAssist and without a configured
		// project there is nothing to ask, and calling that "no subscription"
		// would hide a network or server failure.
		out.Status, out.Err = StatusError, err
		return out, false
	case err == nil:
		out.Tier = tier
		if project == "" {
			project = tier.Project
		}
	}
	out.Project = project
	out.Reason = tier.IneligibleReason
	if project == "" {
		out.Status = StatusNoSubscription
		if out.Reason == "" {
			out.Reason = ReasonNoProject
		}
		return out, false
	}
	quota, err := client.UserQuota(ctx, project)
	if err != nil {
		if isUnauthorized(err) {
			return out, true
		}
		out.Status, out.Reason, out.Err = classifyError(err)
		return out, false
	}
	out.Buckets = quota.Buckets
	out.Status = StatusReady
	return out, false
}

func (f *fetcher) client(creds *Creds) Client {
	return Client{
		HTTP:     f.opt.HTTP,
		Endpoint: f.opt.Endpoint,
		Version:  f.opt.Version,
		Token:    creds.AccessToken,
		Timeout:  f.opt.timeout(),
	}
}

// ensureToken refreshes the credential only when it is missing or about to
// expire, so a healthy CLI session costs no extra request.
func (f *fetcher) ensureToken(ctx context.Context, creds *Creds) error {
	now := f.opt.now()
	if creds.AccessToken != "" && !creds.Expired(now, f.opt.skew()) {
		return nil
	}
	if creds.RefreshToken == "" {
		return ErrNoRefreshToken
	}
	return f.refresh(ctx, creds)
}

func (f *fetcher) refresh(ctx context.Context, creds *Creds) error {
	if creds.RefreshToken == "" {
		return ErrNoRefreshToken
	}
	refresher := Refresher{HTTP: f.opt.HTTP, Endpoint: f.opt.TokenEndpoint, Timeout: f.opt.timeout()}
	tokens, err := refresher.Refresh(ctx, creds.RefreshToken)
	if err != nil {
		return err
	}
	*creds = creds.WithTokens(tokens, f.opt.now())
	// A failed write-back is ignored on purpose: the refreshed token is good
	// for this reading, and the CLI refreshes the file again on its next run.
	_ = creds.Save()
	return nil
}

// project returns the configured Cloud project, or the empty string. The
// endpoints answer 403 rather than 400 without one, so the caller turns the
// empty project into the no-subscription state.
func (f *fetcher) project() string {
	if p := ProjectFromEnv(); p != "" {
		return p
	}
	return ProjectFromSettings(SettingsPath(f.home))
}

// classifyError maps a failed quota call onto a state. A 403 means the account
// or client may not use the endpoint, which is a normal state; everything else
// is a failure.
func classifyError(err error) (Status, string, error) {
	var httpErr *HTTPError
	if errors.As(err, &httpErr) {
		switch {
		case httpErr.Status == http.StatusUnauthorized:
			return StatusExpired, httpErr.Reason, ErrUnauthorized
		case httpErr.Status == http.StatusForbidden:
			return StatusNoSubscription, httpErr.Reason, nil
		}
	}
	return StatusError, "", err
}
