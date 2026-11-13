package glm

import (
	"context"
	"errors"
	"math"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

var fixedNow = time.Date(2026, 11, 2, 10, 0, 0, 0, time.UTC)

// successBody is the shape of the specification: a TIME_LIMIT bucket whose
// percentage is the used share from 0 to 100 and whose reset is in
// milliseconds.
const successBody = `{
  "code": 0,
  "success": true,
  "data": {
    "level": "glm-coding-plan",
    "limits": [
      {"type": "TIME_LIMIT", "number": 1234, "usage": 456, "remaining": 778,
       "percentage": 63, "nextResetTime": 1794362501515,
       "usageDetails": [{"modelCode": "GLM-5.3", "displayName": "GLM-5.3", "usage": 400}]},
      {"type": "", "percentage": 5},
      {"type": "MCP_USAGE_LIMIT", "percentage": 12}
    ]
  }
}`

func TestDecodeQuotaReadsThePrimaryBucket(t *testing.T) {
	quota, err := DecodeQuota([]byte(successBody), fixedNow)
	if err != nil {
		t.Fatal(err)
	}
	if quota.Level != "glm-coding-plan" {
		t.Fatalf("level = %q", quota.Level)
	}
	if len(quota.Limits) != 2 {
		t.Fatalf("limits = %d; the empty type must be dropped", len(quota.Limits))
	}
	if !quota.HasPrimary || quota.Primary.Type != PrimaryLimitType {
		t.Fatalf("primary = %+v", quota.Primary)
	}
	if quota.Primary.Percent != 63 {
		t.Fatalf("used = %v", quota.Primary.Percent)
	}
	if !quota.Primary.HasReset || !quota.Primary.ResetAt.Equal(time.UnixMilli(1794362501515).UTC()) {
		t.Fatalf("reset = %v %v", quota.Primary.ResetAt, quota.Primary.HasReset)
	}
}

func TestDecodeQuotaRejectsAnExpiredEnvelopeUnderHTTP200(t *testing.T) {
	body := `{"code":401,"msg":"令牌已过期或验证不正确","success":false}`
	if _, err := DecodeQuota([]byte(body), fixedNow); !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("err = %v", err)
	}
	english := `{"code":401,"msg":"token expired or incorrect","success":false}`
	if _, err := DecodeQuota([]byte(english), fixedNow); !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("err = %v", err)
	}
}

func TestDecodeQuotaRejectsOtherBusinessCodes(t *testing.T) {
	for _, body := range []string{
		`{"code":500,"success":false}`,
		`{"code":1001,"success":true,"data":{"limits":[]}}`,
		`{"success":false}`,
	} {
		if _, err := DecodeQuota([]byte(body), fixedNow); !errors.Is(err, ErrBusinessCode) {
			t.Fatalf("%s: err = %v", body, err)
		}
	}
}

func TestDecodeQuotaToleratesMissingFields(t *testing.T) {
	for _, body := range []string{
		`{"code":0,"success":true,"data":{}}`,
		`{"code":0,"success":true,"data":{"limits":null}}`,
		`{"code":0,"success":true,"data":{"limits":[{}]}}`,
		`{"code":0,"success":true,"data":{"limits":[{"type":"TIME_LIMIT"}]}}`,
		`{"code":0,"success":true,"data":{"limits":[{"type":"TIME_LIMIT","percentage":63}]},"extra":{"a":1}}`,
	} {
		if _, err := DecodeQuota([]byte(body), fixedNow); err != nil {
			t.Fatalf("%s: %v", body, err)
		}
	}
	if _, err := DecodeQuota([]byte("not json"), fixedNow); !errors.Is(err, ErrDecode) {
		t.Fatalf("err = %v", err)
	}
	if _, err := DecodeQuota([]byte(`{"code":0,"success":true}`), fixedNow); !errors.Is(err, ErrDecode) {
		t.Fatalf("missing data must not read as an empty quota")
	}
}

func TestDecodeQuotaDerivesAndClampsTheShare(t *testing.T) {
	body := `{"code":0,"success":true,"data":{"limits":[
	  {"type":"TIME_LIMIT","number":200,"remaining":150},
	  {"type":"CLAMP_HIGH","percentage":140},
	  {"type":"CLAMP_LOW","percentage":-3}
	]}}`
	quota, err := DecodeQuota([]byte(body), fixedNow)
	if err != nil {
		t.Fatal(err)
	}
	if !quota.Primary.HasPercent || !quota.Primary.Derived {
		t.Fatalf("primary = %+v", quota.Primary)
	}
	if math.Abs(quota.Primary.Percent-25) > 1e-9 {
		t.Fatalf("derived percent = %v", quota.Primary.Percent)
	}
	if quota.Limits[1].Percent != 100 || quota.Limits[2].Percent != 0 {
		t.Fatalf("clamped = %v %v", quota.Limits[1].Percent, quota.Limits[2].Percent)
	}
}

func TestPickPrimaryFallsBackInTheSourceOrder(t *testing.T) {
	primary, ok := PickPrimary([]Limit{{Type: "OTHER", HasPercent: true, Percent: 4}, {Type: PrimaryLimitType, Percent: 9}})
	if !ok || primary.Type != PrimaryLimitType {
		t.Fatalf("primary = %+v", primary)
	}
	primary, ok = PickPrimary([]Limit{{Type: "OTHER"}, {Type: "WITH_SHARE", HasPercent: true, Percent: 4}})
	if !ok || primary.Type != "WITH_SHARE" {
		t.Fatalf("primary = %+v", primary)
	}
	primary, ok = PickPrimary([]Limit{{Type: "ONLY"}})
	if !ok || primary.Type != "ONLY" {
		t.Fatalf("primary = %+v", primary)
	}
	if _, ok := PickPrimary(nil); ok {
		t.Fatal("no bucket reported a primary")
	}
}

func TestUnixTimeCorrectsSecondsAndDropsNonsense(t *testing.T) {
	millis := 1794362501515.0
	at, ok := UnixTime(&millis)
	if !ok || at.UnixMilli() != 1794362501515 {
		t.Fatalf("millis = %v %v", at, ok)
	}
	seconds := 1794362501.0
	at, ok = UnixTime(&seconds)
	if !ok || !at.Equal(time.Unix(1794362501, 0).UTC()) {
		t.Fatalf("seconds = %v %v", at, ok)
	}
	for _, value := range []float64{0, -1, 1000, math.NaN()} {
		value := value
		if _, ok := UnixTime(&value); ok {
			t.Fatalf("%v was accepted as an epoch", value)
		}
	}
	if _, ok := UnixTime(nil); ok {
		t.Fatal("a missing reset time was accepted")
	}
}

// server returns a client bound to a handler and records the last request.
func server(t *testing.T, handler http.HandlerFunc) (Client, *http.Request) {
	t.Helper()
	var seen http.Request
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen = *r
		handler(w, r)
	}))
	t.Cleanup(srv.Close)
	client := Client{HTTP: srv.Client(), QuotaURL: srv.URL, Now: func() time.Time { return fixedNow }}
	return client, &seen
}

func TestFetchSendsTheCredentialWithoutABearerPrefix(t *testing.T) {
	client, seen := server(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("content-type", "application/json")
		_, _ = w.Write([]byte(successBody))
	})
	cand := Candidate{Key: "raw-credential", Kind: KindCodingPlan, Plan: "bigmodel-individual-coding-plan", Region: BigModel}
	quota, err := client.Fetch(context.Background(), cand)
	if err != nil {
		t.Fatal(err)
	}
	if quota.Endpoint == "" || strings.Contains(quota.Endpoint, "raw-credential") {
		t.Fatalf("endpoint = %q", quota.Endpoint)
	}
	if got := seen.Header.Get("authorization"); got != "raw-credential" {
		t.Fatalf("authorization = %q; the monitor API takes the credential verbatim", got)
	}
	if got := seen.Header.Get("accept"); got != "application/json" {
		t.Fatalf("accept = %q", got)
	}
	if seen.Header.Get("bigmodel-organization") != "" || strings.Contains(seen.URL.RawQuery, "type=2") {
		t.Fatal("a personal request carried the team context")
	}
}

func TestFetchAddsTheTeamContext(t *testing.T) {
	client, seen := server(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(successBody))
	})
	cand := Candidate{Key: "team-credential", Kind: KindCodingPlan, Plan: "bigmodel-team-coding-plan", Region: BigModel, Team: true, Organization: "42"}
	if _, err := client.Fetch(context.Background(), cand); err != nil {
		t.Fatal(err)
	}
	if seen.URL.Query().Get("type") != "2" {
		t.Fatalf("query = %q", seen.URL.RawQuery)
	}
	if got := seen.Header.Get("bigmodel-organization"); got != "42" {
		t.Fatalf("organization = %q", got)
	}
}

func TestFetchReportsAnHTTPError(t *testing.T) {
	client, _ := server(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
	})
	if _, err := client.Fetch(context.Background(), Candidate{Key: "k"}); !errors.Is(err, ErrBadResponse) {
		t.Fatalf("err = %v", err)
	}
}

func TestFetchNeverPutsTheCredentialInAnError(t *testing.T) {
	client, _ := server(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})
	cand := Candidate{Key: "SECRET-KEY", Plan: "bigmodel-individual-coding-plan"}
	_, err := client.Fetch(context.Background(), cand)
	if err == nil {
		t.Fatal("a 500 was accepted")
	}
	if strings.Contains(err.Error(), "SECRET-KEY") {
		t.Fatalf("error leaked the credential: %v", err)
	}
}

func TestFetchFirstSkipsARejectedCredential(t *testing.T) {
	var calls int
	client, _ := server(t, func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.Header.Get("authorization") == "stale" {
			_, _ = w.Write([]byte(`{"code":401,"success":false}`))
			return
		}
		_, _ = w.Write([]byte(successBody))
	})
	candidates := []Candidate{{Key: "stale", Plan: "bigmodel-individual-coding-plan"}, {Key: "fresh", Plan: "bigmodel-team-coding-plan", Team: true, Organization: "42"}}
	quota, winner, err := client.FetchFirst(context.Background(), candidates)
	if err != nil {
		t.Fatal(err)
	}
	if !quota.HasPrimary || winner.Plan != "bigmodel-team-coding-plan" || calls != 2 {
		t.Fatalf("winner = %s, calls = %d", winner, calls)
	}
}

func TestFetchFirstReportsTheLastRejection(t *testing.T) {
	client, _ := server(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"code":401,"success":false}`))
	})
	_, _, err := client.FetchFirst(context.Background(), []Candidate{{Key: "a"}, {Key: "b"}})
	if !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("err = %v", err)
	}
	if _, _, err := client.FetchFirst(context.Background(), nil); !errors.Is(err, ErrNoCredentials) {
		t.Fatalf("empty candidates = %v", err)
	}
}

func TestURLForUsesTheEnvironmentOverrideForTheDomesticHostOnly(t *testing.T) {
	t.Setenv("BIGMODEL_USAGE_QUOTA_URL", "")
	t.Setenv("ZCODE_BIGMODEL_USAGE_QUOTA_URL", "https://example.test/quota")
	client := Client{}
	if got := client.URLFor(BigModel); got != "https://example.test/quota" {
		t.Fatalf("domestic = %q", got)
	}
	if got := client.URLFor(Zai); got != ZaiQuotaURL {
		t.Fatalf("international = %q; the override belongs to the domestic host", got)
	}
	client.QuotaURL = "https://override.test/quota"
	if got := client.URLFor(Zai); got != "https://override.test/quota" {
		t.Fatalf("explicit = %q", got)
	}
}

func TestFetchWithoutACredentialIsRefused(t *testing.T) {
	client, _ := server(t, func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("a request was sent without a credential")
	})
	if _, err := client.Fetch(context.Background(), Candidate{}); !errors.Is(err, ErrNoCredentials) {
		t.Fatalf("err = %v", err)
	}
}
