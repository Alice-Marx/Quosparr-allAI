package gemini

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/licoy/quosparr/internal/netx"
	"github.com/licoy/quosparr/internal/platform"
)

// The refresh grant needs the OAuth client the Gemini CLI was built with. That
// pair is published inside the CLI, but a literal of it in this repository is
// indistinguishable from a leaked secret to a credential scanner, so this
// package carries neither half and reads both from the environment instead.
// Without them a refresh is refused rather than attempted, and the user renews
// the file by running the CLI itself.
const (
	// TokenEndpoint is the Google OAuth token endpoint.
	TokenEndpoint = "https://oauth2.googleapis.com/token"
	// EnvClientID and EnvClientSecret carry the OAuth client of the Gemini CLI
	// sign-in. A refresh needs both; either one empty refuses the refresh.
	EnvClientID     = "GEMINI_OAUTH_CLIENT_ID"
	EnvClientSecret = "GEMINI_OAUTH_CLIENT_SECRET"
)

// Credential field names of the CLI's oauth_creds.json.
const (
	credsAccessToken  = "access_token"
	credsRefreshToken = "refresh_token"
	credsIDToken      = "id_token"
	credsScope        = "scope"
	credsTokenType    = "token_type"
	credsExpiryDate   = "expiry_date"
	// credsFilePerm matches the CLI's own private credential file.
	credsFilePerm       = 0o600
	maxTokenBytes       = 32 * 1024
	maxBodyBytes  int64 = 512 * 1024
)

// Sentinel errors. They never carry file bytes, tokens or paths.
var (
	// ErrMissing means the credential file does not exist.
	ErrMissing = errors.New("gemini: credential file missing")
	// ErrUnreadable means the credential file exists but cannot be read.
	ErrUnreadable = errors.New("gemini: credential file unreadable")
	// ErrBadJSON means the credential file is not a JSON object.
	ErrBadJSON = errors.New("gemini: credential is not valid JSON")
	// ErrTooLarge means a file is larger than this package reads.
	ErrTooLarge = errors.New("gemini: file is too large")
	// ErrNoRefreshToken means the credential cannot be refreshed.
	ErrNoRefreshToken = errors.New("gemini: credential has no refresh token")
	// ErrNoClientSecret means the OAuth client is not configured, so this
	// package will not attempt a refresh. It never carries the values.
	ErrNoClientSecret = errors.New("gemini: OAuth client is not set in the environment")
	// ErrRefreshRejected means the token endpoint refused the refresh token
	// (invalid_grant): the sign-in is gone and the user must sign in again.
	ErrRefreshRejected = errors.New("gemini: refresh token rejected")
	// ErrRefreshFailed means the refresh failed for another reason.
	ErrRefreshFailed = errors.New("gemini: refresh failed")
	// ErrNetwork means the request did not complete.
	ErrNetwork = errors.New("gemini: network request failed")
	// ErrUnauthorized means the endpoint answered 401.
	ErrUnauthorized = errors.New("gemini: access token refused")
	// ErrBadResponse means the endpoint answered with a body this package
	// cannot read.
	ErrBadResponse = errors.New("gemini: unexpected response body")
)

// HTTPError is a non-2xx answer. Reason is the google.rpc ErrorInfo reason
// reduced to letters, digits, underscore and dash; the response body and any
// message in it are dropped so no remote text reaches the panel.
type HTTPError struct {
	Status int
	Reason string
}

func (e *HTTPError) Error() string {
	if e.Reason == "" {
		return "gemini: HTTP " + strconv.Itoa(e.Status)
	}
	return "gemini: HTTP " + strconv.Itoa(e.Status) + " " + e.Reason
}

func isUnauthorized(err error) bool {
	var httpErr *HTTPError
	return errors.As(err, &httpErr) && httpErr.Status == http.StatusUnauthorized
}

// Creds is the CLI credential, reduced to the fields this package uses. The
// whole object is kept so a write-back leaves unknown fields alone.
type Creds struct {
	AccessToken  string
	RefreshToken string
	// ExpiryMillis is expiry_date, which the CLI writes as epoch
	// milliseconds.
	ExpiryMillis int64
	HasExpiry    bool
	raw          map[string]json.RawMessage
	path         string
}

// Path is the file the credential came from.
func (c Creds) Path() string { return c.path }

// Expiry is expiry_date as a time, or the zero time.
func (c Creds) Expiry() time.Time {
	if !c.HasExpiry || c.ExpiryMillis <= 0 {
		return time.Time{}
	}
	return time.UnixMilli(c.ExpiryMillis)
}

// Expired reports whether the access token is already past its expiry, with
// skew added so a token that dies mid-request is refreshed first. A credential
// without expiry_date counts as expired when it has no access token.
func (c Creds) Expired(now time.Time, skew time.Duration) bool {
	if !c.HasExpiry || c.ExpiryMillis <= 0 {
		return c.AccessToken == ""
	}
	return !now.Add(skew).Before(c.Expiry())
}

// LoadCreds reads path. A missing file is ErrMissing.
func LoadCreds(path string) (Creds, error) {
	f, err := os.Open(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return Creds{}, ErrMissing
		}
		return Creds{}, ErrUnreadable
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil || info.IsDir() {
		return Creds{}, ErrUnreadable
	}
	if info.Size() > maxCredsBytes {
		return Creds{}, ErrTooLarge
	}
	data, err := io.ReadAll(io.LimitReader(f, maxCredsBytes+1))
	if err != nil {
		return Creds{}, ErrUnreadable
	}
	creds, err := parseCreds(data)
	if err != nil {
		return Creds{}, err
	}
	creds.path = path
	return creds, nil
}

func parseCreds(data []byte) (Creds, error) {
	if int64(len(data)) > maxCredsBytes {
		return Creds{}, ErrTooLarge
	}
	var raw map[string]json.RawMessage
	dec := json.NewDecoder(bytes.NewReader(data))
	if err := dec.Decode(&raw); err != nil || raw == nil {
		return Creds{}, ErrBadJSON
	}
	creds := Creds{raw: raw}
	creds.AccessToken = rawString(raw[credsAccessToken])
	creds.RefreshToken = rawString(raw[credsRefreshToken])
	if n, ok := rawNumber(raw[credsExpiryDate]); ok {
		creds.ExpiryMillis, creds.HasExpiry = n, true
	}
	return creds, nil
}

func rawString(v json.RawMessage) string {
	if len(v) == 0 {
		return ""
	}
	var s string
	if json.Unmarshal(v, &s) != nil {
		return ""
	}
	return strings.TrimSpace(s)
}

// rawNumber reads expiry_date. The CLI writes a JSON number; a numeric string
// is accepted so a hand-edited file still refreshes instead of being counted
// as expired.
func rawNumber(v json.RawMessage) (int64, bool) {
	if len(v) == 0 {
		return 0, false
	}
	var n json.Number
	dec := json.NewDecoder(bytes.NewReader(v))
	dec.UseNumber()
	if dec.Decode(&n) != nil {
		return 0, false
	}
	if i, err := n.Int64(); err == nil {
		return i, true
	}
	if f, err := n.Float64(); err == nil {
		return int64(f), true
	}
	return 0, false
}

// WithTokens is the credential as the token endpoint left it. The refresh
// token is kept because the endpoint does not return one.
func (c Creds) WithTokens(t Tokens, now time.Time) Creds {
	raw := make(map[string]json.RawMessage, len(c.raw)+4)
	for k, v := range c.raw {
		raw[k] = v
	}
	out := c
	out.raw = raw
	out.AccessToken = t.AccessToken
	setRawString(raw, credsAccessToken, t.AccessToken)
	if t.IDToken != "" {
		setRawString(raw, credsIDToken, t.IDToken)
	}
	if t.Scope != "" {
		setRawString(raw, credsScope, t.Scope)
	}
	tokenType := t.TokenType
	if tokenType == "" {
		tokenType = "Bearer"
	}
	setRawString(raw, credsTokenType, tokenType)
	if t.ExpiresIn > 0 {
		out.ExpiryMillis = now.Add(time.Duration(t.ExpiresIn) * time.Second).UnixMilli()
		out.HasExpiry = true
		raw[credsExpiryDate] = json.RawMessage(strconv.FormatInt(out.ExpiryMillis, 10))
	}
	return out
}

func setRawString(raw map[string]json.RawMessage, key, value string) {
	encoded, err := json.Marshal(value)
	if err != nil {
		return
	}
	raw[key] = encoded
}

// Save writes the credential back atomically, keeping the fields this package
// does not own. An empty path writes nothing.
func (c Creds) Save() error {
	if c.path == "" {
		return ErrUnreadable
	}
	body, err := json.Marshal(c.raw)
	if err != nil {
		return ErrBadJSON
	}
	body = append(body, '\n')
	if err := platform.WriteFileAtomic(c.path, body, credsFilePerm); err != nil {
		return ErrUnreadable
	}
	return nil
}

// Tokens is one token-endpoint answer.
type Tokens struct {
	AccessToken string
	IDToken     string
	Scope       string
	TokenType   string
	ExpiresIn   int
}

// Refresher exchanges a refresh token for an access token.
type Refresher struct {
	HTTP     *http.Client
	Endpoint string
	Timeout  time.Duration
	// ClientID and ClientSecret default to EnvClientID and EnvClientSecret.
	ClientID     string
	ClientSecret string
}

// credentials resolves the OAuth client, preferring the explicit fields.
func (r Refresher) credentials() (id, secret string) {
	id = strings.TrimSpace(r.ClientID)
	if id == "" {
		id = strings.TrimSpace(os.Getenv(EnvClientID))
	}
	secret = strings.TrimSpace(r.ClientSecret)
	if secret == "" {
		secret = strings.TrimSpace(os.Getenv(EnvClientSecret))
	}
	return id, secret
}

type refreshAnswer struct {
	AccessToken  string      `json:"access_token"`
	IDToken      string      `json:"id_token"`
	Scope        string      `json:"scope"`
	TokenType    string      `json:"token_type"`
	ExpiresIn    json.Number `json:"expires_in"`
	Error        string      `json:"error"`
	RefreshToken string      `json:"refresh_token"`
}

// Refresh posts the refresh grant. The response carries no refresh token, so
// the caller keeps the old one. invalid_grant is ErrRefreshRejected. Without a
// configured OAuth client it returns ErrNoClientSecret and sends nothing.
func (r Refresher) Refresh(ctx context.Context, refreshToken string) (Tokens, error) {
	if strings.TrimSpace(refreshToken) == "" {
		return Tokens{}, ErrNoRefreshToken
	}
	clientID, clientSecret := r.credentials()
	if clientID == "" || clientSecret == "" {
		return Tokens{}, ErrNoClientSecret
	}
	endpoint := r.Endpoint
	if endpoint == "" {
		endpoint = TokenEndpoint
	}
	form := url.Values{
		"client_id":     {clientID},
		"client_secret": {clientSecret},
		"refresh_token": {refreshToken},
		"grant_type":    {"refresh_token"},
	}
	timeout := r.Timeout
	if timeout <= 0 {
		timeout = 15 * time.Second
	}
	client, err := r.client(ctx)
	if err != nil {
		return Tokens{}, err
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return Tokens{}, ErrRefreshFailed
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return Tokens{}, ErrNetwork
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(io.LimitReader(resp.Body, maxBodyBytes+1))
	if err != nil {
		return Tokens{}, ErrNetwork
	}
	var answer refreshAnswer
	parseErr := json.Unmarshal(data, &answer)
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		if answer.Error == "invalid_grant" {
			return Tokens{}, ErrRefreshRejected
		}
		return Tokens{}, &HTTPError{Status: resp.StatusCode}
	}
	if parseErr != nil {
		return Tokens{}, ErrBadResponse
	}
	if answer.Error == "invalid_grant" {
		return Tokens{}, ErrRefreshRejected
	}
	if answer.AccessToken == "" {
		return Tokens{}, ErrBadResponse
	}
	if len(answer.AccessToken) > maxTokenBytes {
		return Tokens{}, ErrBadResponse
	}
	expiresIn, _ := answer.ExpiresIn.Int64()
	return Tokens{
		AccessToken: answer.AccessToken,
		IDToken:     answer.IDToken,
		Scope:       answer.Scope,
		TokenType:   answer.TokenType,
		ExpiresIn:   int(expiresIn),
	}, nil
}

func (r Refresher) client(ctx context.Context) (*http.Client, error) {
	if r.HTTP != nil {
		return r.HTTP, nil
	}
	client, err := netx.ClientFor(ctx, netx.WithoutCookies)
	if err != nil {
		return nil, ErrNetwork
	}
	return client, nil
}
