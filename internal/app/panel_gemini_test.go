package app

import (
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/licoy/quosparr/internal/gemini"
	"github.com/licoy/quosparr/internal/l10n"
	"github.com/licoy/quosparr/internal/store"
)

// geminiPanelFor renders the Gemini tab of a real preferences store with one
// injected reading. That reading is the whole network stand-in: no test touches
// a credential file or an endpoint.
func geminiPanelFor(t *testing.T, reading gemini.Reading) (*App, *ui.Tester, l10n.Localizer) {
	t.Helper()
	l := l10n.New(l10n.English)
	a, tt := newPanel(t, func(p *store.Preferences) {
		p.Language = store.LanguageEnglish
		p.SelectedProvider = store.ProviderGemini
	}, panelFixture(), PanelActions{})
	a.setGeminiReading(t.TempDir(), reading)
	tt.Frame()
	if got := a.settings.Prefs().SelectedProvider; got != store.ProviderGemini {
		t.Fatalf("selected %s", got)
	}
	return a, tt, l
}

func readyReading(now time.Time) gemini.Reading {
	return gemini.Reading{
		Status:    gemini.StatusReady,
		FetchedAt: now,
		Tier:      gemini.Tier{ID: "standard-tier", Name: "Gemini Code Assist"},
		Project:   "example-project",
		Buckets: []gemini.Bucket{{
			Model: "gemini-2.5-pro", Remaining: 42, Limit: 100, HasLimit: true,
			Fraction: 0.42, HasFraction: true,
			ResetAt: now.Add(3 * time.Hour), HasReset: true,
		}},
		Cost: gemini.Ledger{
			Sessions: 2,
			Models:   []gemini.Usage{{Model: "gemini-2.5-pro", Input: 1_000_000, Requests: 3}},
		},
	}
}

func TestGeminiPanelRendersTheQuotaWithUsedPercentAndReset(t *testing.T) {
	t.Setenv(gemini.HomeEnvVar, "")
	_, tt, l := geminiPanelFor(t, readyReading(time.Now()))

	if !hasExact(tt, tx(l, xGeminiTitle)) {
		t.Fatalf("the quota title is missing: %q", tt.Texts())
	}
	if !tt.HasText("Gemini Code Assist") {
		t.Fatalf("the identity row is missing: %q", tt.Texts())
	}
	if !hasExact(tt, tx(l, xGrokBotUsed, "58")) {
		t.Fatalf("the used percentage is missing: %q", tt.Texts())
	}
	if !hasExact(tt, tx(l, xGeminiRequests, "42", "100")) {
		t.Fatalf("the request counts are missing: %q", tt.Texts())
	}
	if !hasExact(tt, "gemini-2.5-pro") {
		t.Fatalf("the model name is missing: %q", tt.Texts())
	}
	if !tt.HasText(l.Text(l10n.KeyNextResetIn)) {
		t.Fatalf("the reset time is missing: %q", tt.Texts())
	}
	if !hasExact(tt, "$1.25") {
		t.Fatalf("the cost is missing: %q", tt.Texts())
	}
	if !hasExact(tt, tx(l, xGeminiSessions, "2")) {
		t.Fatalf("the session count is missing: %q", tt.Texts())
	}
	if hasExact(tt, tx(l, xGeminiErrorTitle)) || hasExact(tt, tx(l, xGeminiNoSubscriptionTitle)) {
		t.Fatalf("a ready reading drew another state: %q", tt.Texts())
	}
}

func TestGeminiPanelStates(t *testing.T) {
	t.Setenv(gemini.HomeEnvVar, "")
	cases := []struct {
		name    string
		reading gemini.Reading
		want    []textKey
		absent  []textKey
		detail  string
	}{
		{
			name:    "no credential",
			reading: gemini.Reading{Status: gemini.StatusNoCreds},
			want:    []textKey{xGeminiNoCredsTitle},
			absent:  []textKey{xGeminiErrorTitle, xGeminiNoSubscriptionTitle, xGeminiExpiredTitle},
			detail:  "oauth_creds.json",
		},
		{
			name:    "expired sign-in",
			reading: gemini.Reading{Status: gemini.StatusExpired, Err: gemini.ErrRefreshRejected},
			want:    []textKey{xGeminiExpiredTitle},
			absent:  []textKey{xGeminiErrorTitle, xGeminiNoSubscriptionTitle, xGeminiNoCredsTitle},
			detail:  "oauth_creds.json",
		},
		{
			name:    "expired without a configured client",
			reading: gemini.Reading{Status: gemini.StatusExpired, Err: gemini.ErrNoClientSecret},
			want:    []textKey{xGeminiExpiredTitle},
			absent:  []textKey{xGeminiErrorTitle, xGeminiNoCredsTitle, xGeminiNoSubscriptionTitle},
			detail:  "GEMINI_OAUTH_CLIENT_ID",
		},
		{
			name:    "no subscription",
			reading: gemini.Reading{Status: gemini.StatusNoSubscription, Reason: "SUBSCRIPTION_REQUIRED"},
			want:    []textKey{xGeminiNoSubscriptionTitle},
			absent:  []textKey{xGeminiErrorTitle, xGeminiNoCredsTitle, xGeminiExpiredTitle},
			detail:  "Reason SUBSCRIPTION_REQUIRED",
		},
		{
			name:    "network failure",
			reading: gemini.Reading{Status: gemini.StatusError, Err: gemini.ErrNetwork},
			want:    []textKey{xGeminiErrorTitle, xGeminiErrNetwork},
			absent:  []textKey{xGeminiNoSubscriptionTitle, xGeminiNoCredsTitle},
		},
		{
			name:    "http failure",
			reading: gemini.Reading{Status: gemini.StatusError, Err: &gemini.HTTPError{Status: 502}},
			want:    []textKey{xGeminiErrorTitle},
			absent:  []textKey{xGeminiNoSubscriptionTitle},
			detail:  "502",
		},
		{
			name:    "unreadable credential",
			reading: gemini.Reading{Status: gemini.StatusError, Err: gemini.ErrBadJSON},
			want:    []textKey{xGeminiErrorTitle, xGeminiErrCreds},
			absent:  []textKey{xGeminiNoSubscriptionTitle},
		},
		{
			name:    "unknown state",
			reading: gemini.Reading{Status: gemini.StatusUnknown},
			absent:  []textKey{xGeminiErrorTitle, xGeminiNoCredsTitle, xGeminiNoSubscriptionTitle},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, tt, l := geminiPanelFor(t, tc.reading)
			for _, key := range tc.want {
				if !hasExact(tt, tx(l, key)) {
					t.Fatalf("%s missing %q: %q", tc.name, key, tt.Texts())
				}
			}
			for _, key := range tc.absent {
				if hasExact(tt, tx(l, key)) {
					t.Fatalf("%s also drew %q: %q", tc.name, key, tt.Texts())
				}
			}
			if tc.detail != "" && !tt.HasText(tc.detail) {
				t.Fatalf("%s missing %q: %q", tc.name, tc.detail, tt.Texts())
			}
			if !hasExact(tt, tx(l, xGrokBotCostTitle)) {
				t.Fatalf("%s lost the cost section: %q", tc.name, tt.Texts())
			}
			if !hasExact(tt, tx(l, xGeminiCostEmpty)) {
				t.Fatalf("%s lost the empty-cost note: %q", tc.name, tt.Texts())
			}
		})
	}
}

func TestGeminiPanelNeverRendersACredentialValue(t *testing.T) {
	t.Setenv(gemini.HomeEnvVar, "")
	t.Setenv(gemini.EnvClientID, "test-client-id-value")
	t.Setenv(gemini.EnvClientSecret, "test-client-secret-value")
	_, tt, _ := geminiPanelFor(t, gemini.Reading{Status: gemini.StatusExpired, Err: gemini.ErrNoClientSecret})
	for _, text := range tt.Texts() {
		for _, value := range []string{"test-client-id-value", "test-client-secret-value"} {
			if strings.Contains(text, value) {
				t.Fatalf("%q reached the panel in %q", value, text)
			}
		}
	}
	// The guidance names the variables, never their values.
	if !tt.HasText(gemini.EnvClientID) || !tt.HasText(gemini.EnvClientSecret) {
		t.Fatalf("the actionable text does not name the variables: %q", tt.Texts())
	}
	if !tt.HasText("oauth_creds.json") {
		t.Fatalf("the guidance does not name the credential file: %q", tt.Texts())
	}
}

func TestGeminiPanelCostFlagsUnpricedAndEstimatedModels(t *testing.T) {
	t.Setenv(gemini.HomeEnvVar, "")
	reading := gemini.Reading{
		Status: gemini.StatusNoSubscription,
		Reason: "SUBSCRIPTION_REQUIRED",
		Cost: gemini.Ledger{Models: []gemini.Usage{
			{Model: "gemini-3.8-flash", Input: 1_000_000, Requests: 1},
			{Model: "gemini-mystery", Input: 10, Requests: 1},
		}},
	}
	_, tt, l := geminiPanelFor(t, reading)
	if !hasExact(tt, "$0.30") {
		t.Fatalf("the family estimate is missing: %q", tt.Texts())
	}
	if !hasExact(tt, tx(l, xGeminiCostEstimate)) {
		t.Fatalf("the estimate is not flagged: %q", tt.Texts())
	}
	if !hasExact(tt, tx(l, xGrokBotCostMissingPrice)) {
		t.Fatalf("the unpriced model is not flagged: %q", tt.Texts())
	}
	if !hasExact(tt, "—") {
		t.Fatalf("an unpriced model has no amount: %q", tt.Texts())
	}
	if hasExact(tt, tx(l, xGeminiCostEmpty)) {
		t.Fatalf("priced rows and the empty note must not both appear: %q", tt.Texts())
	}
}

func TestGeminiHoldKeepsOneFetchInFlight(t *testing.T) {
	s := NewSettings(store.New(store.NewMemoryDomain(), nil), System{}, nil)
	s.prefs = store.DefaultPreferences()
	a := New(Options{Version: "test", Settings: s})
	if !a.beginGeminiFetch() {
		t.Fatal("the first fetch must start")
	}
	if a.beginGeminiFetch() {
		t.Fatal("a second fetch started while one was in flight")
	}
	if screen := a.geminiScreen(); !screen.Busy || screen.Fetched {
		t.Fatalf("screen = %+v", screen)
	}
	a.setGeminiReading("/tmp/home", readyReading(time.Now()))
	screen := a.geminiScreen()
	if screen.Busy || !screen.Fetched || !screen.Reading.Ready() || screen.Home != "/tmp/home" {
		t.Fatalf("screen = %+v", screen)
	}
	if !a.beginGeminiFetch() {
		t.Fatal("a finished reading must let the next fetch start")
	}
}

func TestGeminiTextsAreLocalizedInEveryLanguage(t *testing.T) {
	for _, lang := range l10n.Languages() {
		loc := l10n.New(lang)
		for _, key := range []textKey{
			xGeminiTitle, xGeminiNoCredsTitle, xGeminiNoCredsDetail, xGeminiExpiredTitle,
			xGeminiExpiredDetail, xGeminiNoSubscriptionTitle, xGeminiNoSubscriptionDetail,
			xGeminiErrorTitle, xGeminiErrNetwork, xGeminiErrHTTP, xGeminiErrResponse,
			xGeminiErrCreds, xGeminiErrSignIn, xGeminiErrGeneric, xGeminiReason,
			xGeminiNoBuckets, xGeminiRequests, xGeminiCostEmpty, xGeminiCostEstimate,
			xGeminiSessions, xGeminiSourceNote,
		} {
			if text := tx(loc, key); strings.TrimSpace(text) == "" || text == string(key) {
				t.Fatalf("%s/%s = %q", lang, key, text)
			}
		}
	}
}
