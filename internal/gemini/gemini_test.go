package gemini

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

var testNow = time.Date(2026, 10, 10, 12, 0, 0, 0, time.UTC)

// isolateEnv clears the variables that decide where the credential lives and
// which project is asked for, so a developer shell cannot change a test.
func isolateEnv(t *testing.T) {
	t.Helper()
	for _, key := range []string{
		HomeEnvVar, "GOOGLE_CLOUD_PROJECT", "GOOGLE_CLOUD_PROJECT_ID", EndpointEnvVar, VersionEnvVar,
		EnvClientID, EnvClientSecret,
	} {
		t.Setenv(key, "")
	}
}

// writeHomeCreds writes a credential file into a temporary home.
func writeHomeCreds(t *testing.T, expiryMillis int64, extra string) string {
	t.Helper()
	home := t.TempDir()
	dir := Dir(home)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	body := fmt.Sprintf(
		`{"access_token":"access-token","refresh_token":"refresh-token","scope":"scope-value","token_type":"Bearer","id_token":"id-token","expiry_date":%d%s}`,
		expiryMillis, extra)
	if err := os.WriteFile(filepath.Join(dir, CredsFileName), []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	return home
}

func okJSON(body string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, body)
	}
}

func statusJSON(status int, body string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		fmt.Fprint(w, body)
	}
}

const subscriptionRequired = `{"error":{"code":403,"message":"You do not have a valid license of this product","status":"PERMISSION_DENIED","details":[{"@type":"type.googleapis.com/google.rpc.ErrorInfo","reason":"SUBSCRIPTION_REQUIRED","domain":"cloudcompanion.googleapis.com"}]}}`

const bucketsBody = `{"buckets":[
{"modelId":"gemini-2.5-pro","remainingAmount":"42","remainingFraction":0.42,"resetTime":"2026-10-11T00:00:00Z"},
{"modelId":"gemini-3.8-flash","remainingFraction":0.9,"resetTime":"2026-10-11T00:00:00Z"},
{"modelId":"","remainingFraction":0.5}]}`

func TestLoadCredsReadsExpiryAsEpochMilliseconds(t *testing.T) {
	isolateEnv(t)
	home := writeHomeCreds(t, testNow.Add(time.Hour).UnixMilli(), "")
	creds, err := LoadCreds(CredsPath(home))
	if err != nil {
		t.Fatal(err)
	}
	if creds.AccessToken == "" || creds.RefreshToken == "" {
		t.Fatal("tokens were not read")
	}
	if !creds.HasExpiry || creds.Expired(testNow, 0) {
		t.Fatalf("a credential expiring in an hour must not be expired: %+v", creds.Expiry())
	}
	if !creds.Expired(testNow, 2*time.Hour) {
		t.Fatal("the skew must count the credential as expiring")
	}
}

func TestLoadCredsStatesTheReasonWithoutTheContent(t *testing.T) {
	isolateEnv(t)
	home := t.TempDir()
	path := CredsPath(home)
	if _, err := LoadCreds(path); !errors.Is(err, ErrMissing) {
		t.Fatalf("missing file: %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("{not json"), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err := LoadCreds(path)
	if !errors.Is(err, ErrBadJSON) {
		t.Fatalf("bad json: %v", err)
	}
	if strings.Contains(err.Error(), "not json") || strings.Contains(err.Error(), path) {
		t.Fatalf("the error carries file content or a path: %q", err.Error())
	}
}

func TestRefreshPostsTheGrantAndKeepsUnknownFields(t *testing.T) {
	isolateEnv(t)
	home := writeHomeCreds(t, testNow.Add(-time.Hour).UnixMilli(), `,"kept":{"nested":true}`)
	t.Setenv(EnvClientID, "test-client-id-value")
	t.Setenv(EnvClientSecret, "test-client-secret-value")
	want := map[string]string{
		"client_id":     "test-client-id-value",
		"client_secret": "test-client-secret-value",
		"grant_type":    "refresh_token",
		"refresh_token": "refresh-token",
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s", r.Method)
		}
		if ct := r.Header.Get("Content-Type"); ct != "application/x-www-form-urlencoded" {
			t.Errorf("content type = %q", ct)
		}
		if err := r.ParseForm(); err != nil {
			t.Error(err)
		}
		for key, value := range want {
			if got := r.PostForm.Get(key); got != value {
				t.Errorf("%s = %q, want %q", key, got, value)
			}
		}
		fmt.Fprint(w, `{"access_token":"new-access","expires_in":3599,"scope":"new-scope","token_type":"Bearer","id_token":"new-id"}`)
	}))
	defer server.Close()

	creds, err := LoadCreds(CredsPath(home))
	if err != nil {
		t.Fatal(err)
	}
	tokens, err := Refresher{HTTP: server.Client(), Endpoint: server.URL, Timeout: 5 * time.Second}.Refresh(context.Background(), creds.RefreshToken)
	if err != nil {
		t.Fatal(err)
	}
	refreshed := creds.WithTokens(tokens, testNow)
	if err := refreshed.Save(); err != nil {
		t.Fatal(err)
	}
	again, err := LoadCreds(CredsPath(home))
	if err != nil {
		t.Fatal(err)
	}
	if again.AccessToken != "new-access" || again.RefreshToken != "refresh-token" {
		t.Fatalf("access = %q refresh = %q", again.AccessToken, again.RefreshToken)
	}
	if want := testNow.Add(3599 * time.Second).UnixMilli(); again.ExpiryMillis != want {
		t.Fatalf("expiry = %d, want %d", again.ExpiryMillis, want)
	}
	if _, kept := again.raw["kept"]; !kept {
		t.Fatal("an unknown field was dropped")
	}
	if scope := rawString(again.raw[credsScope]); scope != "new-scope" {
		t.Fatalf("scope = %q", scope)
	}
}

func TestRefreshWithoutClientCredentialsSendsNothing(t *testing.T) {
	isolateEnv(t)
	var calls int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		fmt.Fprint(w, `{"access_token":"must-not-be-used","expires_in":3599}`)
	}))
	defer server.Close()

	_, err := Refresher{HTTP: server.Client(), Endpoint: server.URL}.Refresh(context.Background(), "refresh-token")
	if !errors.Is(err, ErrNoClientSecret) {
		t.Fatalf("err = %v", err)
	}
	if calls != 0 {
		t.Fatalf("the token endpoint was called %d times without a client", calls)
	}
	if strings.Contains(err.Error(), "must-not-be-used") || strings.Contains(err.Error(), "refresh-token") {
		t.Fatalf("the error carries a credential: %q", err.Error())
	}
	// One half is not enough either.
	t.Setenv(EnvClientID, "test-client-id-value")
	if _, err := (Refresher{HTTP: server.Client(), Endpoint: server.URL}).Refresh(context.Background(), "refresh-token"); !errors.Is(err, ErrNoClientSecret) {
		t.Fatalf("only the client id is set: %v", err)
	}
	if calls != 0 {
		t.Fatalf("the token endpoint was called %d times with half a client", calls)
	}
}

func TestFetchWithoutClientCredentialsReportsExpiryWithoutRequesting(t *testing.T) {
	isolateEnv(t)
	home := writeHomeCreds(t, testNow.Add(-time.Hour).UnixMilli(), "")
	var tokenCalls int
	tokens := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tokenCalls++
		fmt.Fprint(w, `{"access_token":"must-not-be-used","expires_in":3599}`)
	}))
	defer tokens.Close()
	qs := newQuotaServer(t, nil, okJSON(bucketsBody))
	opt := qs.options()
	opt.TokenEndpoint = tokens.URL

	reading := Fetch(context.Background(), home, opt)
	if reading.Status != StatusExpired {
		t.Fatalf("status = %v err = %v", reading.Status, reading.Err)
	}
	if !errors.Is(reading.Err, ErrNoClientSecret) {
		t.Fatalf("err = %v", reading.Err)
	}
	if tokenCalls != 0 || qs.quotaCalls != 0 {
		t.Fatalf("token calls = %d quota calls = %d", tokenCalls, qs.quotaCalls)
	}
}

func TestNoSecretReachesAnErrorOrTheCredentialFile(t *testing.T) {
	isolateEnv(t)
	const (
		clientID     = "test-client-id-value"
		clientSecret = "test-client-secret-value"
	)
	t.Setenv(EnvClientID, clientID)
	t.Setenv(EnvClientSecret, clientSecret)
	home := writeHomeCreds(t, testNow.Add(-time.Hour).UnixMilli(), "")
	var sawForm bool
	tokens := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Error(err)
		}
		if r.PostForm.Get("client_secret") == clientSecret && r.PostForm.Get("client_id") == clientID {
			sawForm = true
		}
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer tokens.Close()
	qs := newQuotaServer(t, nil, okJSON(bucketsBody))
	opt := qs.options()
	opt.TokenEndpoint = tokens.URL

	reading := Fetch(context.Background(), home, opt)
	if !sawForm {
		t.Fatal("the refresh did not carry the configured client")
	}
	if reading.Status != StatusError || reading.Err == nil {
		t.Fatalf("a 500 from the token endpoint is an error: %v %v", reading.Status, reading.Err)
	}
	if message := reading.Err.Error(); strings.Contains(message, clientSecret) || strings.Contains(message, clientID) {
		t.Fatalf("the error carries a client credential: %q", message)
	}
	body, err := os.ReadFile(CredsPath(home))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(body), clientSecret) || strings.Contains(string(body), clientID) {
		t.Fatal("the credential file kept the client credential")
	}
}

func TestRefreshRejectsAnInvalidGrant(t *testing.T) {
	isolateEnv(t)
	t.Setenv(EnvClientID, "test-client-id-value")
	t.Setenv(EnvClientSecret, "test-client-secret-value")
	server := httptest.NewServer(okJSON(`{"error":"invalid_grant","error_description":"Token has been expired or revoked."}`))
	defer server.Close()
	_, err := Refresher{HTTP: server.Client(), Endpoint: server.URL}.Refresh(context.Background(), "refresh-token")
	if !errors.Is(err, ErrRefreshRejected) {
		t.Fatalf("err = %v", err)
	}
	if _, err := (Refresher{HTTP: server.Client(), Endpoint: server.URL}).Refresh(context.Background(), ""); !errors.Is(err, ErrNoRefreshToken) {
		t.Fatalf("empty refresh token: %v", err)
	}
}

func TestClientPostsToTheColonMethodWithBearerToken(t *testing.T) {
	isolateEnv(t)
	var seenPath, seenAuth, seenBody string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seenPath, seenAuth = r.URL.Path, r.Header.Get("Authorization")
		var body map[string]string
		_ = json.NewDecoder(r.Body).Decode(&body)
		seenBody = body["project"]
		fmt.Fprint(w, bucketsBody)
	}))
	defer server.Close()

	client := Client{HTTP: server.Client(), Endpoint: server.URL, Token: "access-token"}
	quota, err := client.UserQuota(context.Background(), "example-project")
	if err != nil {
		t.Fatal(err)
	}
	if seenPath != "/v1internal:retrieveUserQuota" {
		t.Fatalf("path = %q", seenPath)
	}
	if seenAuth != "Bearer access-token" {
		t.Fatalf("authorization = %q", seenAuth)
	}
	if seenBody != "example-project" {
		t.Fatalf("project = %q", seenBody)
	}
	if len(quota.Buckets) != 2 {
		t.Fatalf("a bucket without a model id must be skipped: %+v", quota.Buckets)
	}
	// Worst bucket first.
	pro := quota.Buckets[0]
	if pro.Model != "gemini-2.5-pro" || !pro.HasLimit || pro.Limit != 100 || pro.Remaining != 42 {
		t.Fatalf("pro bucket = %+v", pro)
	}
	if used, ok := pro.UsedPercent(); !ok || used < 57.9 || used > 58.1 {
		t.Fatalf("used = %v %v", used, ok)
	}
	flash := quota.Buckets[1]
	if flash.Model != "gemini-3.8-flash" || !flash.HasLimit || flash.Limit != 100 || flash.Remaining != 90 {
		t.Fatalf("flash bucket = %+v", flash)
	}
	if !pro.HasReset || !pro.ResetAt.Equal(time.Date(2026, 10, 11, 0, 0, 0, 0, time.UTC)) {
		t.Fatalf("reset = %v", pro.ResetAt)
	}
}

func TestParseQuotaClampsAndKeepsBucketsWithoutAFraction(t *testing.T) {
	isolateEnv(t)
	quota, err := ParseQuota([]byte(`{"buckets":[
		{"modelId":"m-fraction-over","remainingAmount":"10","remainingFraction":1.4},
		{"modelId":"m-no-fraction","remainingAmount":"7"},
		{"modelId":"m-zero","remainingAmount":"0","remainingFraction":0}]}`))
	if err != nil {
		t.Fatal(err)
	}
	byModel := map[string]Bucket{}
	for _, b := range quota.Buckets {
		byModel[b.Model] = b
	}
	over := byModel["m-fraction-over"]
	if over.Fraction != 1 || !over.HasFraction {
		t.Fatalf("a fraction above 1 must clamp: %+v", over)
	}
	if used, ok := over.UsedPercent(); !ok || used != 0 {
		t.Fatalf("used = %v %v", used, ok)
	}
	without := byModel["m-no-fraction"]
	if without.HasFraction || without.Remaining != 7 {
		t.Fatalf("a bucket without a fraction keeps its count only: %+v", without)
	}
	if _, ok := without.UsedPercent(); ok {
		t.Fatal("a bucket without a fraction has no percentage")
	}
}

func TestProjectFromEnvAndSettings(t *testing.T) {
	isolateEnv(t)
	if got := ProjectFromEnv(); got != "" {
		t.Fatalf("project = %q", got)
	}
	t.Setenv("GOOGLE_CLOUD_PROJECT", "12345")
	if got := ProjectFromEnv(); got != "" {
		t.Fatalf("a numeric project id must be rejected: %q", got)
	}
	t.Setenv("GOOGLE_CLOUD_PROJECT", "env-project")
	if got := ProjectFromEnv(); got != "env-project" {
		t.Fatalf("project = %q", got)
	}
	home := t.TempDir()
	if got := ProjectFromSettings(SettingsPath(home)); got != "" {
		t.Fatalf("missing settings file = %q", got)
	}
	dir := Dir(home)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, SettingsFileName), []byte(`{"cloudaicompanionProject":"settings-project"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if got := ProjectFromSettings(SettingsPath(home)); got != "settings-project" {
		t.Fatalf("project = %q", got)
	}
}

// quotaServer answers both colon methods and counts the quota calls.
type quotaServer struct {
	server     *httptest.Server
	quotaCalls int
	authSeen   string
}

func newQuotaServer(t *testing.T, tier, quota http.HandlerFunc) *quotaServer {
	t.Helper()
	qs := &quotaServer{}
	qs.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1internal:loadCodeAssist":
			if tier == nil {
				fmt.Fprint(w, `{"currentTier":{"id":"free-tier"}}`)
				return
			}
			tier(w, r)
		case "/v1internal:retrieveUserQuota":
			qs.quotaCalls++
			qs.authSeen = r.Header.Get("Authorization")
			quota(w, r)
		default:
			t.Errorf("unexpected path %q", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(qs.server.Close)
	return qs
}

func (qs *quotaServer) options() Options {
	return Options{
		HTTP:     qs.server.Client(),
		Endpoint: qs.server.URL,
		Now:      func() time.Time { return testNow },
		Timeout:  5 * time.Second,
	}
}

func TestFetchReadsTheQuotaThroughTheStoredCredential(t *testing.T) {
	isolateEnv(t)
	home := writeHomeCreds(t, testNow.Add(time.Hour).UnixMilli(), "")
	qs := newQuotaServer(t,
		okJSON(`{"currentTier":{"id":"standard-tier","name":"Gemini Code Assist"},"cloudaicompanionProject":"from-assist"}`),
		okJSON(bucketsBody))

	reading := Fetch(context.Background(), home, qs.options())
	if reading.Status != StatusReady || !reading.Ready() {
		t.Fatalf("status = %v err = %v", reading.Status, reading.Err)
	}
	if reading.Project != "from-assist" || reading.Tier.ID != "standard-tier" {
		t.Fatalf("project = %q tier = %+v", reading.Project, reading.Tier)
	}
	if len(reading.Buckets) != 2 {
		t.Fatalf("buckets = %+v", reading.Buckets)
	}
	if qs.authSeen != "Bearer access-token" {
		t.Fatalf("authorization = %q", qs.authSeen)
	}
}

func TestFetchReportsNoSubscriptionAsItsOwnState(t *testing.T) {
	isolateEnv(t)
	home := writeHomeCreds(t, testNow.Add(time.Hour).UnixMilli(), "")
	qs := newQuotaServer(t,
		okJSON(`{"currentTier":{"id":"free-tier","name":"Gemini Code Assist for individuals"},"ineligibleTiers":[{"reasonCode":"UNSUPPORTED_CLIENT","tierId":"free-tier"}],"cloudaicompanionProject":"proj"}`),
		statusJSON(http.StatusForbidden, subscriptionRequired))

	reading := Fetch(context.Background(), home, qs.options())
	if reading.Status != StatusNoSubscription {
		t.Fatalf("status = %v err = %v", reading.Status, reading.Err)
	}
	if reading.Reason != "SUBSCRIPTION_REQUIRED" {
		t.Fatalf("reason = %q", reading.Reason)
	}
	if reading.Err != nil {
		t.Fatalf("no subscription is not an error: %v", reading.Err)
	}
	if len(reading.Buckets) != 0 {
		t.Fatalf("buckets = %+v", reading.Buckets)
	}
}

func TestFetchWithoutAProjectDoesNotAskForQuota(t *testing.T) {
	isolateEnv(t)
	home := writeHomeCreds(t, testNow.Add(time.Hour).UnixMilli(), "")
	qs := newQuotaServer(t, okJSON(`{"currentTier":{"id":"free-tier"},"ineligibleTiers":[{"reasonCode":"UNSUPPORTED_CLIENT"}]}`), okJSON(bucketsBody))

	reading := Fetch(context.Background(), home, qs.options())
	if reading.Status != StatusNoSubscription {
		t.Fatalf("status = %v", reading.Status)
	}
	if reading.Reason != "UNSUPPORTED_CLIENT" {
		t.Fatalf("reason = %q", reading.Reason)
	}
	if qs.quotaCalls != 0 {
		t.Fatalf("quota was asked %d times without a project", qs.quotaCalls)
	}
}

func TestFetchStateWithoutACredential(t *testing.T) {
	isolateEnv(t)
	reading := Fetch(context.Background(), t.TempDir(), Options{Now: func() time.Time { return testNow }})
	if reading.Status != StatusNoCreds || reading.Err != nil {
		t.Fatalf("status = %v err = %v", reading.Status, reading.Err)
	}
	if empty := Fetch(context.Background(), "", Options{}); empty.Status != StatusNoCreds {
		t.Fatalf("empty home: %v", empty.Status)
	}
}

func TestFetchStateWhenTheSignInIsGone(t *testing.T) {
	isolateEnv(t)
	home := writeHomeCreds(t, testNow.Add(-time.Hour).UnixMilli(), "")
	t.Setenv(EnvClientID, "test-client-id-value")
	t.Setenv(EnvClientSecret, "test-client-secret-value")
	tokens := httptest.NewServer(okJSON(`{"error":"invalid_grant"}`))
	defer tokens.Close()
	qs := newQuotaServer(t, nil, okJSON(bucketsBody))
	opt := qs.options()
	opt.TokenEndpoint = tokens.URL

	reading := Fetch(context.Background(), home, opt)
	if reading.Status != StatusExpired {
		t.Fatalf("status = %v err = %v", reading.Status, reading.Err)
	}
	if !errors.Is(reading.Err, ErrRefreshRejected) {
		t.Fatalf("err = %v", reading.Err)
	}
	if qs.quotaCalls != 0 {
		t.Fatal("an unrefreshable credential must not reach the quota endpoint")
	}
}

func TestFetchRefreshesAnExpiredCredentialAndWritesItBack(t *testing.T) {
	isolateEnv(t)
	home := writeHomeCreds(t, testNow.Add(-time.Hour).UnixMilli(), "")
	t.Setenv(EnvClientID, "test-client-id-value")
	t.Setenv(EnvClientSecret, "test-client-secret-value")
	tokens := httptest.NewServer(okJSON(`{"access_token":"refreshed-access","expires_in":3599,"token_type":"Bearer"}`))
	defer tokens.Close()
	qs := newQuotaServer(t, okJSON(`{"cloudaicompanionProject":"proj"}`), okJSON(bucketsBody))
	opt := qs.options()
	opt.TokenEndpoint = tokens.URL

	reading := Fetch(context.Background(), home, opt)
	if reading.Status != StatusReady {
		t.Fatalf("status = %v err = %v", reading.Status, reading.Err)
	}
	if qs.authSeen != "Bearer refreshed-access" {
		t.Fatalf("the refreshed token was not used: %q", qs.authSeen)
	}
	creds, err := LoadCreds(CredsPath(home))
	if err != nil {
		t.Fatal(err)
	}
	if creds.AccessToken != "refreshed-access" || creds.RefreshToken != "refresh-token" {
		t.Fatalf("access = %q refresh = %q", creds.AccessToken, creds.RefreshToken)
	}
	if want := testNow.Add(3599 * time.Second).UnixMilli(); creds.ExpiryMillis != want {
		t.Fatalf("expiry = %d, want %d", creds.ExpiryMillis, want)
	}
}

func TestFetchRetriesOnceOnA401(t *testing.T) {
	isolateEnv(t)
	home := writeHomeCreds(t, testNow.Add(time.Hour).UnixMilli(), "")
	t.Setenv(EnvClientID, "test-client-id-value")
	t.Setenv(EnvClientSecret, "test-client-secret-value")
	var calls int
	qs := newQuotaServer(t, func(w http.ResponseWriter, r *http.Request) {
		calls++
		if calls == 1 {
			statusJSON(http.StatusUnauthorized, `{"error":{"code":401,"status":"UNAUTHENTICATED"}}`)(w, r)
			return
		}
		fmt.Fprint(w, `{"cloudaicompanionProject":"proj"}`)
	}, okJSON(bucketsBody))
	tokens := httptest.NewServer(okJSON(`{"access_token":"retried-access","expires_in":3599}`))
	defer tokens.Close()
	opt := qs.options()
	opt.TokenEndpoint = tokens.URL

	reading := Fetch(context.Background(), home, opt)
	if reading.Status != StatusReady {
		t.Fatalf("status = %v err = %v", reading.Status, reading.Err)
	}
	if calls != 2 {
		t.Fatalf("assist calls = %d", calls)
	}
	if qs.authSeen != "Bearer retried-access" {
		t.Fatalf("authorization = %q", qs.authSeen)
	}
}

func TestFetchReportsAnEndpointFailure(t *testing.T) {
	isolateEnv(t)
	home := writeHomeCreds(t, testNow.Add(time.Hour).UnixMilli(), "")
	qs := newQuotaServer(t, okJSON(`{"cloudaicompanionProject":"proj"}`), statusJSON(http.StatusInternalServerError, "{\"error\":{\"status\":\"INTERNAL\"}}"))
	reading := Fetch(context.Background(), home, qs.options())
	if reading.Status != StatusError {
		t.Fatalf("status = %v", reading.Status)
	}
	var httpErr *HTTPError
	if !errors.As(reading.Err, &httpErr) || httpErr.Status != http.StatusInternalServerError {
		t.Fatalf("err = %v", reading.Err)
	}
}

func TestFetchReportsAnUnreachableEndpoint(t *testing.T) {
	isolateEnv(t)
	home := writeHomeCreds(t, testNow.Add(time.Hour).UnixMilli(), "")
	qs := newQuotaServer(t, okJSON(`{"cloudaicompanionProject":"proj"}`), okJSON(bucketsBody))
	opt := qs.options()
	opt.Endpoint = "http://127.0.0.1:1"
	reading := Fetch(context.Background(), home, opt)
	if reading.Status != StatusError || !errors.Is(reading.Err, ErrNetwork) {
		t.Fatalf("status = %v err = %v", reading.Status, reading.Err)
	}
}

func TestReasonOfKeepsOnlyASafeCode(t *testing.T) {
	isolateEnv(t)
	if got := reasonOf([]byte(`{"error":{"details":[{"reason":"SUBSCRIPTION_REQUIRED"}]}}`)); got != "SUBSCRIPTION_REQUIRED" {
		t.Fatalf("reason = %q", got)
	}
	if got := reasonOf([]byte(`{"error":{"details":[{"reason":"BAD reason \"quoted\" text"}],"status":"INVALID_ARGUMENT"}}`)); got != "BAD" {
		t.Fatalf("reason = %q", got)
	}
	if got := reasonOf([]byte("not json")); got != "" {
		t.Fatalf("reason = %q", got)
	}
}

func TestFormatPercent(t *testing.T) {
	isolateEnv(t)
	for value, want := range map[float64]string{0: "0", 58: "58", 57.95: "58", 6.74: "6.7", 100: "100", 33.33: "33.3"} {
		if got := FormatPercent(value); got != want {
			t.Errorf("FormatPercent(%v) = %q, want %q", value, got, want)
		}
	}
}
