package kimi

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/licoy/quosparr/internal/platform"
)

const (
	// EnvHome overrides the Kimi Code directory, as the CLI does.
	EnvHome = "KIMI_CODE_HOME"
	// EnvBaseURL overrides the usage endpoint and beats the region marker.
	EnvBaseURL = "KIMI_CODE_BASE_URL"
	// EnvOAuthHost overrides the OAuth host of the refresh endpoint.
	EnvOAuthHost = "KIMI_CODE_OAUTH_HOST"

	// dirName is the Kimi Code directory under the user home.
	dirName = ".kimi-code"
	// CredentialsFileName is the OAuth document inside credentials/.
	CredentialsFileName = "kimi-code.json"

	// BaseMainland and BaseGlobal are the two regional usage endpoints.
	BaseMainland = "https://api.kimi.com/coding/v1"
	BaseGlobal   = "https://api.kimi.ai/coding/v1"
	// OAuthHost and OAuthURL are the public token endpoint of the CLI.
	OAuthHost = "https://auth.kimi.com"
	OAuthURL  = OAuthHost + "/api/oauth/token"
	// ClientID is the public client id of the CLI; it has no secret.
	ClientID = "17e5f671-d194-4dfb-9706-5516cb48c098"

	// RegionMainland and RegionGlobal are the values of the region marker.
	RegionMainland = "mainland-cn"
	RegionGlobal   = "global"

	// MinRefreshThreshold is the floor of the refresh threshold, in seconds.
	MinRefreshThreshold = 300
	// RequestTimeout bounds one request, as the CLI does.
	RequestTimeout = 8 * time.Second
	maxBodyBytes   = 256 << 10
	maxCredential  = 64 << 10
)

// Sentinel errors are safe to show: they carry no token, no path and no file
// bytes.
var (
	ErrNoCredential  = errors.New("kimi: no credential")
	ErrCredential    = errors.New("kimi: credential file is unreadable")
	ErrRefreshReject = errors.New("kimi: refresh token rejected")
	ErrRefresh       = errors.New("kimi: token refresh failed")
	ErrUnauthorized  = errors.New("kimi: credential rejected")
	ErrNetwork       = errors.New("kimi: request failed")
	ErrResponse      = errors.New("kimi: response is not valid")
	ErrNoClient      = errors.New("kimi: no HTTP client")
)

// StatusError is a non-2xx answer of the usage endpoint.
type StatusError struct{ Code int }

func (e *StatusError) Error() string {
	return "kimi: usage endpoint returned HTTP " + strconv.Itoa(e.Code)
}

// Doer sends one HTTP request; *http.Client satisfies it.
type Doer interface {
	Do(*http.Request) (*http.Response, error)
}

// Home is the Kimi Code directory for a user home. KIMI_CODE_HOME wins, as in
// the CLI.
func Home(home string) string {
	if dir := strings.TrimSpace(os.Getenv(EnvHome)); dir != "" {
		return dir
	}
	if strings.TrimSpace(home) == "" {
		return ""
	}
	return filepath.Join(home, dirName)
}

// CredentialsPath is <home>/credentials/kimi-code.json.
func CredentialsPath(home string) string {
	dir := Home(home)
	if dir == "" {
		return ""
	}
	return filepath.Join(dir, "credentials", CredentialsFileName)
}

// SessionsDir is <home>/sessions, which holds one directory per workspace.
func SessionsDir(home string) string {
	dir := Home(home)
	if dir == "" {
		return ""
	}
	return filepath.Join(dir, "sessions")
}

// Region is the lower-cased content of the region marker, or "" when it is
// absent or unreadable.
func Region(home string) string {
	dir := Home(home)
	if dir == "" {
		return ""
	}
	data, err := os.ReadFile(filepath.Join(dir, "region"))
	if err != nil {
		return ""
	}
	return strings.ToLower(strings.TrimSpace(string(data)))
}

// BaseURL resolves the usage endpoint: KIMI_CODE_BASE_URL, else the regional
// default of the marker, else the mainland endpoint.
func BaseURL(home string) string {
	if v := trimBase(os.Getenv(EnvBaseURL)); v != "" {
		return v
	}
	if Region(home) == RegionGlobal {
		return BaseGlobal
	}
	return BaseMainland
}

// OAuthURLFor resolves the refresh endpoint, honoring KIMI_CODE_OAUTH_HOST.
func OAuthURLFor() string {
	host := trimBase(os.Getenv(EnvOAuthHost))
	if host == "" {
		return OAuthURL
	}
	return host + "/api/oauth/token"
}

func trimBase(raw string) string {
	return strings.TrimRight(strings.TrimSpace(raw), "/")
}

// Credentials are the tokens of the CLI. ExpiresAt is a Unix timestamp in
// seconds, not milliseconds. ExpiresIn is the lifetime recorded when the token
// was issued and only feeds the refresh threshold.
type Credentials struct {
	AccessToken  string
	RefreshToken string
	ExpiresAt    int64
	ExpiresIn    int64
	Scope        string
	TokenType    string
}

// refreshThreshold is max(300, expiresIn*0.5) seconds, as the CLI computes it.
func refreshThreshold(expiresIn int64) int64 {
	threshold := expiresIn / 2
	if threshold < MinRefreshThreshold {
		return MinRefreshThreshold
	}
	return threshold
}

// NeedsRefresh reports whether the token is inside its refresh threshold. A
// credential without an expiry is never refreshed ahead of time.
func (c Credentials) NeedsRefresh(now time.Time) bool {
	if c.ExpiresAt == 0 {
		return false
	}
	return c.ExpiresAt-now.Unix() < refreshThreshold(c.ExpiresIn)
}

// credentialFile is the document on disk. Unknown fields are kept so the CLI
// can keep reading its own file after a rotated pair is written back.
type credentialFile struct {
	path   string
	fields map[string]json.RawMessage
	Credentials
}

// LoadCredentials reads the stored tokens. Every failure is one of the
// sentinels.
func LoadCredentials(home string) (Credentials, error) {
	file, err := readCredentialFile(CredentialsPath(home))
	if err != nil {
		return Credentials{}, err
	}
	if strings.TrimSpace(file.AccessToken) == "" || strings.TrimSpace(file.RefreshToken) == "" {
		return Credentials{}, ErrCredential
	}
	return file.Credentials, nil
}

func readCredentialFile(path string) (credentialFile, error) {
	if path == "" {
		return credentialFile{}, ErrNoCredential
	}
	f, err := os.Open(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return credentialFile{}, ErrNoCredential
		}
		return credentialFile{}, ErrCredential
	}
	defer f.Close()
	if info, err := f.Stat(); err != nil || info.IsDir() {
		return credentialFile{}, ErrCredential
	}
	data, err := io.ReadAll(io.LimitReader(f, maxCredential+1))
	if err != nil || len(data) > maxCredential {
		return credentialFile{}, ErrCredential
	}
	fields := map[string]json.RawMessage{}
	if err := json.Unmarshal(data, &fields); err != nil {
		return credentialFile{}, ErrCredential
	}
	out := credentialFile{path: path, fields: fields}
	out.AccessToken = rawString(fields["access_token"])
	out.RefreshToken = rawString(fields["refresh_token"])
	out.ExpiresAt = rawInt64(fields["expires_at"])
	out.ExpiresIn = rawInt64(fields["expires_in"])
	out.Scope = rawString(fields["scope"])
	out.TokenType = rawString(fields["token_type"])
	return out, nil
}

func rawString(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var s string
	if err := json.Unmarshal(raw, &s); err != nil {
		return ""
	}
	return strings.TrimSpace(s)
}

func rawInt64(raw json.RawMessage) int64 {
	if len(raw) == 0 {
		return 0
	}
	var n json.Number
	if err := json.Unmarshal(raw, &n); err != nil {
		return 0
	}
	if v, err := n.Int64(); err == nil {
		return v
	}
	v, err := n.Float64()
	if err != nil {
		return 0
	}
	return int64(v)
}

// apply records a refreshed pair and writes the file back atomically with the
// owner-only permission the CLI uses. Nothing is written when the pair did not
// change.
func (f *credentialFile) apply(tok Token, now time.Time) error {
	changed := tok.AccessToken != f.AccessToken || tok.RefreshToken != f.RefreshToken
	f.AccessToken = tok.AccessToken
	f.RefreshToken = tok.RefreshToken
	f.ExpiresIn = tok.ExpiresIn
	f.ExpiresAt = now.Unix() + tok.ExpiresIn
	if tok.Scope != "" {
		f.Scope = tok.Scope
	}
	if tok.TokenType != "" {
		f.TokenType = tok.TokenType
	}
	if !changed {
		return nil
	}
	if f.path == "" {
		return ErrCredential
	}
	if f.fields == nil {
		f.fields = map[string]json.RawMessage{}
	}
	set := func(key string, value any) {
		raw, err := json.Marshal(value)
		if err != nil {
			return
		}
		f.fields[key] = raw
	}
	set("access_token", f.AccessToken)
	set("refresh_token", f.RefreshToken)
	set("expires_at", f.ExpiresAt)
	set("expires_in", f.ExpiresIn)
	if f.TokenType != "" {
		set("token_type", f.TokenType)
	}
	if f.Scope != "" {
		set("scope", f.Scope)
	}
	body, err := json.Marshal(f.fields)
	if err != nil {
		return ErrCredential
	}
	if err := platform.WriteFileAtomic(f.path, body, 0o600); err != nil {
		return ErrCredential
	}
	return nil
}

// Token is the answer of a refresh.
type Token struct {
	AccessToken  string
	RefreshToken string
	ExpiresIn    int64
	Scope        string
	TokenType    string
}

// refreshMu serializes refreshes in this process. The CLI takes a cross-process
// lock only when NODE_ENV=test, so a rotation race with a running CLI is out of
// reach here; the CLI repairs it on its next refresh.
var refreshMu sync.Mutex

// Refresh exchanges the refresh token for a new pair. A 4xx answer means the
// grant is gone (ErrRefreshReject: the user must sign in again); everything
// else is ErrRefresh. Persisting the rotation is the caller's job.
func Refresh(ctx context.Context, client Doer, endpoint, refreshToken string, timeout time.Duration) (Token, error) {
	if client == nil {
		return Token{}, ErrNoClient
	}
	if strings.TrimSpace(refreshToken) == "" {
		return Token{}, ErrRefreshReject
	}
	if timeout <= 0 {
		timeout = RequestTimeout
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	form := url.Values{
		"client_id":     {ClientID},
		"grant_type":    {"refresh_token"},
		"refresh_token": {refreshToken},
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return Token{}, ErrRefresh
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return Token{}, ErrRefresh
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxBodyBytes))
	if err != nil {
		return Token{}, ErrRefresh
	}
	if resp.StatusCode >= 400 && resp.StatusCode < 500 {
		return Token{}, ErrRefreshReject
	}
	if resp.StatusCode != http.StatusOK {
		return Token{}, ErrRefresh
	}
	var raw struct {
		AccessToken  string      `json:"access_token"`
		RefreshToken string      `json:"refresh_token"`
		ExpiresIn    json.Number `json:"expires_in"`
		Scope        string      `json:"scope"`
		TokenType    string      `json:"token_type"`
	}
	if err := json.Unmarshal(body, &raw); err != nil {
		return Token{}, ErrRefresh
	}
	expiresIn, err := raw.ExpiresIn.Int64()
	if err != nil || expiresIn <= 0 {
		return Token{}, ErrRefresh
	}
	access := strings.TrimSpace(raw.AccessToken)
	refresh := strings.TrimSpace(raw.RefreshToken)
	if access == "" || refresh == "" {
		return Token{}, ErrRefresh
	}
	tokenType := strings.TrimSpace(raw.TokenType)
	if tokenType == "" {
		tokenType = "Bearer"
	}
	return Token{AccessToken: access, RefreshToken: refresh, ExpiresIn: expiresIn, Scope: raw.Scope, TokenType: tokenType}, nil
}
