package kimi

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/licoy/quosparr/internal/usage/decimal"
)

// nl is a newline, spelled without an escape sequence so this file stays plain.
var nl = string([]byte{10})

// stubDoer answers every request through fn, so a test never opens a socket.
type stubDoer struct {
	fn func(*http.Request) (*http.Response, error)
}

func (s stubDoer) Do(req *http.Request) (*http.Response, error) { return s.fn(req) }

func bodyResponse(code int, body string) *http.Response {
	return &http.Response{
		StatusCode: code,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}

const sampleUsage = `{
  "usages": {
    "limit_5h": {"used_ratio": 0.42, "reset_time": "2026-11-13T09:00:00Z"},
    "limit_7d": {"used_ratio": 13},
    "limit_month_total": {"used_ratio": "5.5"},
    "limit_month_code": {}
  },
  "boosterWallet": {
    "balance": {"type": "BOOSTER", "amount": 20000000, "amountLeft": 7500000},
    "monthlyChargeLimitEnabled": true,
    "monthlyChargeLimit": {"priceInCents": 5000, "currency": "cnY"},
    "monthlyUsed": {"priceInCents": 1200}
  }
}`

func TestParseUsagesNormalizesPercentAndCents(t *testing.T) {
	now := time.Date(2026, 11, 13, 8, 0, 0, 0, time.UTC)
	snap, err := ParseUsages([]byte(sampleUsage), "https://api.kimi.com/coding/v1/", now)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if snap.Usages.Limit5h == nil || snap.Usages.Limit5h.UsedPercent != 42 {
		t.Fatalf("limit_5h = %+v", snap.Usages.Limit5h)
	}
	if !snap.Usages.Limit5h.HasReset || !snap.Usages.Limit5h.ResetAt.Equal(time.Date(2026, 11, 13, 9, 0, 0, 0, time.UTC)) {
		t.Fatalf("reset = %+v", snap.Usages.Limit5h)
	}
	if snap.Usages.Limit7d == nil || snap.Usages.Limit7d.UsedPercent != 13 {
		t.Fatalf("limit_7d = %+v", snap.Usages.Limit7d)
	}
	if snap.Usages.Limit7d.HasReset {
		t.Fatal("a window without reset_time reported a reset")
	}
	if snap.Usages.MonthTotal == nil || snap.Usages.MonthTotal.UsedPercent != 5.5 {
		t.Fatalf("limit_month_total = %+v", snap.Usages.MonthTotal)
	}
	if snap.Usages.MonthCode != nil {
		t.Fatal("an entry without used_ratio must not become a window")
	}
	wallet := snap.Wallet
	if wallet == nil {
		t.Fatal("wallet missing")
	}
	// amount and amountLeft are fixed-point micro-units: divided by 1e6 they
	// are 20 and 7.5 cents, and the fractional cent rounds up.
	if wallet.TotalCents != 20 || wallet.LeftCents != 8 {
		t.Fatalf("balance cents = %d/%d", wallet.TotalCents, wallet.LeftCents)
	}
	if wallet.LimitCents != 5000 || wallet.UsedCents != 1200 || !wallet.LimitEnabled {
		t.Fatalf("limit = %+v", wallet)
	}
	if wallet.Currency != "CNY" {
		t.Fatalf("currency = %q", wallet.Currency)
	}
	if snap.Base != "https://api.kimi.com/coding/v1" {
		t.Fatalf("base = %q", snap.Base)
	}
	if got := FormatPercent(42); got != "42" {
		t.Fatalf("format = %q", got)
	}
}

func TestParseUsagesClampsAndDropsBrokenValues(t *testing.T) {
	body := `{"usages":{"limit_5h":{"used_ratio":250},"limit_7d":{"used_ratio":-3},"limit_month_total":{"used_ratio":null}},"boosterWallet":null}`
	snap, err := ParseUsages([]byte(body), "https://api.kimi.ai/coding/v1", time.Time{})
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if snap.Usages.Limit5h == nil || snap.Usages.Limit5h.UsedPercent != 100 {
		t.Fatalf("high ratio = %+v", snap.Usages.Limit5h)
	}
	if snap.Usages.Limit7d == nil || snap.Usages.Limit7d.UsedPercent != 0 {
		t.Fatalf("low ratio = %+v", snap.Usages.Limit7d)
	}
	if snap.Usages.MonthTotal != nil || snap.Usages.MonthCode != nil {
		t.Fatal("a null ratio must drop the window")
	}
	if snap.Wallet != nil {
		t.Fatal("a null wallet must stay null")
	}
	if _, err := ParseUsages([]byte("not json"), "b", time.Time{}); !errors.Is(err, ErrResponse) {
		t.Fatalf("broken body error = %v", err)
	}
	if _, err := ParseUsages([]byte(`{"boosterWallet":null}`), "b", time.Time{}); !errors.Is(err, ErrResponse) {
		t.Fatalf("missing usages error = %v", err)
	}
}

func TestFetchUsagesReportsStatusAndTransportErrors(t *testing.T) {
	ctx := context.Background()
	now := time.Now()
	cases := []struct {
		name string
		doer Doer
		want error
	}{
		{"unauthorized", stubDoer{fn: func(*http.Request) (*http.Response, error) {
			return bodyResponse(http.StatusUnauthorized, `{"error":{"message":"nope"}}`), nil
		}}, ErrUnauthorized},
		{"forbidden", stubDoer{fn: func(*http.Request) (*http.Response, error) {
			return bodyResponse(http.StatusForbidden, ""), nil
		}}, ErrUnauthorized},
		{"transport", stubDoer{fn: func(*http.Request) (*http.Response, error) {
			return nil, errors.New("dial failed")
		}}, ErrNetwork},
		{"no client", nil, ErrNoClient},
	}
	for _, tc := range cases {
		_, err := FetchUsages(ctx, tc.doer, "https://usage.test/coding/v1", "token", now)
		if !errors.Is(err, tc.want) {
			t.Fatalf("%s: err = %v", tc.name, err)
		}
	}
	_, err := FetchUsages(ctx, stubDoer{fn: func(*http.Request) (*http.Response, error) {
		return bodyResponse(http.StatusServiceUnavailable, ""), nil
	}}, "https://usage.test/coding/v1", "token", now)
	var status *StatusError
	if !errors.As(err, &status) || status.Code != http.StatusServiceUnavailable {
		t.Fatalf("status error = %v", err)
	}
	if !strings.Contains(err.Error(), "503") {
		t.Fatalf("status text = %q", err.Error())
	}
}

func TestFetchUsagesSendsBearerAndAccept(t *testing.T) {
	var seen http.Header
	doer := stubDoer{fn: func(req *http.Request) (*http.Response, error) {
		seen = req.Header.Clone()
		if req.Method != http.MethodGet || req.URL.Path != "/coding/v1/usages" {
			t.Errorf("request = %s %s", req.Method, req.URL.Path)
		}
		return bodyResponse(http.StatusOK, sampleUsage), nil
	}}
	if _, err := FetchUsages(context.Background(), doer, "https://usage.test/coding/v1/", "tok", time.Now()); err != nil {
		t.Fatalf("fetch: %v", err)
	}
	if got := seen.Get("Authorization"); got != "Bearer tok" {
		t.Fatalf("authorization = %q", got)
	}
	if got := seen.Get("Accept"); got != "application/json" {
		t.Fatalf("accept = %q", got)
	}
}

// tempHome isolates a test from the developer machine's ~/.kimi-code.
func tempHome(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	t.Setenv(EnvHome, dir)
	t.Setenv(EnvBaseURL, "")
	t.Setenv(EnvOAuthHost, "")
	return dir
}

func writeCredentials(t *testing.T, dir, body string) string {
	t.Helper()
	path := filepath.Join(dir, "credentials", CredentialsFileName)
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestFetchReportsEveryState(t *testing.T) {
	now := time.Date(2026, 11, 13, 8, 0, 0, 0, time.UTC)
	usageOK := stubDoer{fn: func(*http.Request) (*http.Response, error) {
		return bodyResponse(http.StatusOK, sampleUsage), nil
	}}
	usageDown := stubDoer{fn: func(*http.Request) (*http.Response, error) {
		return bodyResponse(http.StatusBadGateway, ""), nil
	}}
	refreshRejected := stubDoer{fn: func(req *http.Request) (*http.Response, error) {
		if req.URL.Path == "/api/oauth/token" {
			return bodyResponse(http.StatusBadRequest, `{"error":"invalid_grant"}`), nil
		}
		return bodyResponse(http.StatusOK, sampleUsage), nil
	}}

	t.Run("no credential", func(t *testing.T) {
		tempHome(t)
		rep := Fetch(context.Background(), Config{Client: usageOK, Now: func() time.Time { return now }})
		if rep.State != StateNoCredential || !errors.Is(rep.Err, ErrNoCredential) {
			t.Fatalf("report = %+v", rep)
		}
		if rep.Base != BaseMainland {
			t.Fatalf("base = %q", rep.Base)
		}
	})

	t.Run("expired and rejected", func(t *testing.T) {
		dir := tempHome(t)
		writeCredentials(t, dir, `{"access_token":"a","refresh_token":"r","expires_at":1,"expires_in":900,"scope":"kimi-code","token_type":"Bearer"}`)
		rep := Fetch(context.Background(), Config{
			Client: refreshRejected, BaseURL: "https://usage.test/coding/v1",
			OAuthURL: "https://auth.test/api/oauth/token", Now: func() time.Time { return now },
		})
		if rep.State != StateExpired || !errors.Is(rep.Err, ErrRefreshReject) {
			t.Fatalf("report = %+v", rep)
		}
	})

	t.Run("endpoint fails", func(t *testing.T) {
		dir := tempHome(t)
		writeCredentials(t, dir, `{"access_token":"a","refresh_token":"r","expires_at":0}`)
		rep := Fetch(context.Background(), Config{
			Client: usageDown, BaseURL: "https://usage.test/coding/v1", Now: func() time.Time { return now },
		})
		if rep.State != StateFailed || rep.Err == nil {
			t.Fatalf("report = %+v", rep)
		}
	})

	t.Run("ready", func(t *testing.T) {
		dir := tempHome(t)
		writeCredentials(t, dir, `{"access_token":"a","refresh_token":"r","expires_at":0}`)
		rep := Fetch(context.Background(), Config{
			Client: usageOK, BaseURL: "https://usage.test/coding/v1", Now: func() time.Time { return now },
		})
		if rep.State != StateReady || rep.Err != nil {
			t.Fatalf("report = %+v", rep)
		}
		if rep.Usages.Limit5h == nil || rep.FetchedAt.IsZero() {
			t.Fatalf("reading = %+v", rep)
		}
		if rep.HasExpiry != (!rep.ExpiresAt.IsZero()) {
			t.Fatalf("expiry = %+v", rep)
		}
	})
}

func TestRefreshRotatesTheTokenAndKeepsUnknownFields(t *testing.T) {
	now := time.Date(2026, 11, 13, 8, 0, 0, 0, time.UTC)
	dir := tempHome(t)
	path := writeCredentials(t, dir, `{"access_token":"old-access","refresh_token":"old-refresh","expires_at":1,"expires_in":900,"scope":"kimi-code","token_type":"Bearer","device_id":"KEEP-ME"}`)
	var form map[string]string
	doer := stubDoer{fn: func(req *http.Request) (*http.Response, error) {
		switch req.URL.Path {
		case "/api/oauth/token":
			raw, _ := io.ReadAll(req.Body)
			if ct := req.Header.Get("Content-Type"); ct != "application/x-www-form-urlencoded" {
				t.Errorf("content type = %q", ct)
			}
			form = map[string]string{}
			for _, part := range strings.Split(string(raw), "&") {
				key, value, _ := strings.Cut(part, "=")
				form[key] = value
			}
			return bodyResponse(http.StatusOK, `{"access_token":"new-access","refresh_token":"new-refresh","expires_in":900,"token_type":"Bearer","scope":"kimi-code"}`), nil
		default:
			if got := req.Header.Get("Authorization"); got != "Bearer new-access" {
				t.Errorf("usage authorization = %q", got)
			}
			return bodyResponse(http.StatusOK, sampleUsage), nil
		}
	}}
	rep := Fetch(context.Background(), Config{
		Client: doer, BaseURL: "https://usage.test/coding/v1",
		OAuthURL: "https://auth.test/api/oauth/token", Now: func() time.Time { return now },
	})
	if rep.State != StateReady {
		t.Fatalf("report = %+v", rep)
	}
	if form["grant_type"] != "refresh_token" || form["client_id"] != ClientID || form["refresh_token"] != "old-refresh" {
		t.Fatalf("form = %v", form)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var stored map[string]any
	if err := json.Unmarshal(data, &stored); err != nil {
		t.Fatalf("stored json: %v", err)
	}
	if stored["access_token"] != "new-access" || stored["refresh_token"] != "new-refresh" {
		t.Fatalf("stored tokens = %v", stored)
	}
	if stored["device_id"] != "KEEP-ME" {
		t.Fatalf("unknown field dropped: %v", stored)
	}
	if got, ok := stored["expires_at"].(float64); !ok || int64(got) != now.Unix()+900 {
		t.Fatalf("expires_at = %v", stored["expires_at"])
	}
	if strings.Contains(string(data), "old-") {
		t.Fatal("the rotated file still carries an old token")
	}
}

func TestEnsureFreshLeavesAFreshTokenAlone(t *testing.T) {
	now := time.Date(2026, 11, 13, 8, 0, 0, 0, time.UTC)
	dir := tempHome(t)
	path := writeCredentials(t, dir, `{"access_token":"a","refresh_token":"r","expires_at":1814500000,"expires_in":900}`)
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	calls := 0
	doer := stubDoer{fn: func(req *http.Request) (*http.Response, error) {
		if req.URL.Path == "/api/oauth/token" {
			calls++
			return bodyResponse(http.StatusBadRequest, ""), nil
		}
		return bodyResponse(http.StatusOK, sampleUsage), nil
	}}
	rep := Fetch(context.Background(), Config{
		Client: doer, BaseURL: "https://usage.test/coding/v1",
		OAuthURL: "https://auth.test/api/oauth/token", Now: func() time.Time { return now },
	})
	if rep.State != StateReady {
		t.Fatalf("report = %+v", rep)
	}
	if calls != 0 {
		t.Fatalf("a fresh token was refreshed %d times", calls)
	}
	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(before) != string(after) {
		t.Fatal("a fresh credential file was rewritten")
	}
}

func TestScanCostsSumsUsageRecords(t *testing.T) {
	dir := tempHome(t)
	write := func(session string, lines ...string) {
		path := filepath.Join(dir, "sessions", "wd_app_1", session, "agents", "main", "wire.jsonl")
		if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(strings.Join(lines, nl)+nl), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	write("session_a",
		`{"type":"usage.record","model":"kimi-code/k3.1-pro","usage":{"inputOther":4357,"output":141,"inputCacheRead":16384,"inputCacheCreation":0},"time":1793431204566}`,
		`{"type":"token_counting.measured","tokens":20882,"time":1793431204566}`,
		`{"type":"usage.record","model":"kimi-code/k3.1-pro","usage":{"inputOther":100,"output":9,"inputCacheRead":0,"inputCacheCreation":50},"time":1793431304566}`,
	)
	write("session_b",
		`{"type":"usage.record","model":"","usage":{"inputOther":-5,"output":7,"inputCacheRead":0,"inputCacheCreation":0},"time":1793431404566}`,
	)
	costs := ScanCosts(filepath.Join(dir, ".."))
	if costs.Sessions != 2 || costs.Records != 3 {
		t.Fatalf("costs = %+v", costs)
	}
	if !costs.Present() || costs.Truncated {
		t.Fatalf("costs = %+v", costs)
	}
	if len(costs.Models) != 2 {
		t.Fatalf("models = %+v", costs.Models)
	}
	first := costs.Models[0]
	if first.Model != "kimi-code/k3.1-pro" {
		t.Fatalf("order = %+v", costs.Models)
	}
	if first.InputOther != 4457 || first.CacheRead != 16384 || first.CacheWrite != 50 || first.Output != 150 {
		t.Fatalf("totals = %+v", first)
	}
	if first.Records != 2 || first.LastAt.IsZero() {
		t.Fatalf("record = %+v", first)
	}
	if costs.Models[1].Model != "unknown" || costs.Models[1].InputOther != 0 {
		t.Fatalf("unknown model = %+v", costs.Models[1])
	}
}

func TestScanCostsToleratesAMissingDirectory(t *testing.T) {
	tempHome(t)
	costs := ScanCosts(t.TempDir())
	if costs.Present() || costs.Sessions != 0 || costs.Models != nil {
		t.Fatalf("costs = %+v", costs)
	}
}

func TestCostRowsMarkAModelWithoutPrice(t *testing.T) {
	costs := Costs{
		Models:  []ModelUsage{{Model: "kimi-code/k3.1-pro", InputOther: 1_000_000, Output: 1_000_000, Records: 1}},
		Records: 1,
	}
	rows := CostRows(costs, DefaultPrices())
	if len(rows) != 1 || !rows[0].Missing || rows[0].Amount != "" {
		t.Fatalf("rows = %+v", rows)
	}
	table := PriceTable{"k3.1-pro": Price{
		Input: decimal.MustParse("1"), CacheRead: decimal.MustParse("0.1"), Output: decimal.MustParse("2"),
	}}
	rows = CostRows(costs, table)
	if len(rows) != 1 || rows[0].Missing {
		t.Fatalf("rows = %+v", rows)
	}
	if rows[0].Amount != "$3" {
		t.Fatalf("amount = %q", rows[0].Amount)
	}
}

func TestBaseURLFollowsRegionAndEnvironment(t *testing.T) {
	dir := tempHome(t)
	if got := BaseURL(dir); got != BaseMainland {
		t.Fatalf("default base = %q", got)
	}
	if err := os.WriteFile(filepath.Join(dir, "region"), []byte("global"+nl), 0o600); err != nil {
		t.Fatal(err)
	}
	if got := Region(dir); got != RegionGlobal {
		t.Fatalf("region = %q", got)
	}
	if got := BaseURL(dir); got != BaseGlobal {
		t.Fatalf("global base = %q", got)
	}
	t.Setenv(EnvBaseURL, "https://relay.test/coding/v1/")
	if got := BaseURL(dir); got != "https://relay.test/coding/v1" {
		t.Fatalf("override = %q", got)
	}
	if got := UsageURL("https://relay.test/coding/v1"); got != "https://relay.test/coding/v1/usages" {
		t.Fatalf("usage url = %q", got)
	}
}

func TestCredentialsWithoutTokensIsAnError(t *testing.T) {
	dir := tempHome(t)
	writeCredentials(t, dir, `{"access_token":"only"}`)
	if _, err := LoadCredentials(dir); !errors.Is(err, ErrCredential) {
		t.Fatalf("err = %v", err)
	}
	t.Setenv(EnvHome, filepath.Join(t.TempDir(), "absent"))
	if _, err := LoadCredentials(t.TempDir()); !errors.Is(err, ErrNoCredential) {
		t.Fatalf("missing err = %v", err)
	}
}
