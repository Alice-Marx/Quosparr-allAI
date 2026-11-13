package app

import (
	"errors"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/licoy/quosparr/internal/kimi"
	"github.com/licoy/quosparr/internal/l10n"
	"github.com/licoy/quosparr/internal/store"
)

// kimiFixture is one reading with three of the four windows, a wallet and a
// local cost reading, so every section has something to render.
func kimiFixture(at time.Time) (kimi.Report, kimi.Costs) {
	reset := at.Add(90 * time.Minute)
	report := kimi.Report{
		State:     kimi.StateReady,
		Base:      kimi.BaseMainland,
		Region:    kimi.RegionMainland,
		FetchedAt: at,
		HasExpiry: true,
		ExpiresAt: at.Add(12 * time.Minute),
		Usages: kimi.Usages{
			Limit5h:    &kimi.Bucket{UsedPercent: 42, RawRatio: 0.42, HasReset: true, ResetAt: reset},
			Limit7d:    &kimi.Bucket{UsedPercent: 13, RawRatio: 0.13, HasReset: true, ResetAt: reset.Add(24 * time.Hour)},
			MonthTotal: &kimi.Bucket{UsedPercent: 5.5, RawRatio: 0.055, HasReset: true, ResetAt: reset.Add(48 * time.Hour)},
		},
		Wallet: &kimi.Wallet{
			TotalCents: 20, LeftCents: 8, LimitEnabled: true,
			LimitCents: 5000, UsedCents: 1200, Currency: "USD",
		},
	}
	costs := kimi.Costs{
		Dir:      "kimi-sessions",
		Sessions: 2,
		Records:  3,
		Models: []kimi.ModelUsage{{
			Model: "kimi-code/k3.1-pro", InputOther: 4457, CacheRead: 16384,
			CacheWrite: 50, Output: 150, Records: 2,
		}},
	}
	return report, costs
}

func newKimiPanel(t *testing.T) (*App, *ui.Tester) {
	t.Helper()
	return newPanel(t, func(p *store.Preferences) {
		p.Language = store.LanguageSimplifiedChinese
		p.SelectedProvider = store.ProviderKimi
	}, panelFixture(), PanelActions{})
}

func wantKimiText(t *testing.T, tt *ui.Tester, want string) {
	t.Helper()
	if !tt.HasText(want) {
		t.Fatalf("missing %q in %q", want, tt.Texts())
	}
}

func TestKimiPanelRendersTheReading(t *testing.T) {
	at := time.Now()
	l := l10n.New(l10n.SimplifiedChinese)
	a, tt := newKimiPanel(t)
	report, costs := kimiFixture(at)
	a.kimi.mu.Lock()
	a.kimi.loaded = true
	a.kimi.report = report
	a.kimi.costs = costs
	a.kimi.costsAt = at
	a.kimi.prices = kimi.DefaultPrices()
	a.kimi.mu.Unlock()
	tt.Frame()
	if !hasExact(tt, tx(l, xKimiTab)) {
		t.Fatalf("identity row: %q", tt.Texts())
	}
	for _, want := range []string{
		tx(l, xKimiRegionMainland),
		kimi.BaseMainland,
		tx(l, xKimiQuotaTitle),
		tx(l, xKimiLimit5h),
		tx(l, xKimiLimit7d),
		tx(l, xKimiLimitMonthTotal),
		tx(l, xKimiUsed, "42"),
		tx(l, xKimiUsed, "13"),
		tx(l, xKimiUsed, "5.5"),
		tx(l, xKimiUnknown),
		l.Text(l10n.KeyNextResetIn),
		tx(l, xKimiCredentialValid, report.ExpiresAt.Local().Format("2006-01-02 15:04")),
		tx(l, xKimiWalletBalance, "0.08 USD", "0.20 USD"),
		tx(l, xKimiWalletLimit, "50.00 USD", "12.00 USD"),
		tx(l, xKimiCostTitle),
		"kimi-code/k3.1-pro",
		tx(l, xKimiCostTokens, "4457", "16384", "50", "150"),
		tx(l, xKimiCostMissingPrice),
		tx(l, xKimiCostScanned, "2", "3"),
		tx(l, xKimiNotesCredential),
	} {
		wantKimiText(t, tt, want)
	}
	if !hasExact(tt, tx(l, xKimiUsed, "42")) {
		t.Fatalf("the percent line is not its own text: %q", tt.Texts())
	}
	if tt.HasText(tx(l, xKimiResetMissing)) {
		t.Fatalf("a window with a reset claimed to have none: %q", tt.Texts())
	}
}

func TestKimiPanelRendersEveryFailureState(t *testing.T) {
	l := l10n.New(l10n.SimplifiedChinese)
	cases := []struct {
		name string
		hold func(*kimiHold)
		want []string
	}{
		{
			name: "no credential",
			hold: func(h *kimiHold) {
				h.loaded = true
				h.report = kimi.Report{State: kimi.StateNoCredential, Err: kimi.ErrNoCredential, Base: kimi.BaseGlobal, Region: kimi.RegionGlobal}
			},
			want: []string{
				tx(l, xKimiNoCredential), tx(l, xKimiNoCredentialDetail), tx(l, xKimiRegionGlobal),
			},
		},
		{
			name: "expired credential",
			hold: func(h *kimiHold) {
				h.loaded = true
				h.report = kimi.Report{
					State: kimi.StateExpired, Err: kimi.ErrRefreshReject, Base: kimi.BaseMainland,
					Region: kimi.RegionMainland, HasExpiry: true, ExpiresAt: time.Now().Add(-time.Hour),
				}
			},
			want: []string{
				tx(l, xKimiUnauthorized), tx(l, xKimiUnauthorizedDetail), tx(l, xKimiCredentialExpired),
			},
		},
		{
			name: "endpoint error",
			hold: func(h *kimiHold) {
				h.loaded = true
				h.report = kimi.Report{State: kimi.StateFailed, Err: &kimi.StatusError{Code: 503}, Base: kimi.BaseMainland}
			},
			want: []string{tx(l, xKimiFetchFailed), tx(l, xKimiErrStatus, "503")},
		},
		{
			name: "network error",
			hold: func(h *kimiHold) {
				h.loaded = true
				h.report = kimi.Report{State: kimi.StateFailed, Err: kimi.ErrNetwork, Base: kimi.BaseMainland}
			},
			want: []string{tx(l, xKimiFetchFailed), tx(l, xKimiErrNetwork)},
		},
		{
			name: "never loaded",
			hold: func(h *kimiHold) {
				h.loading = false
			},
			want: []string{tx(l, xKimiRegionUnknown)},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			a, tt := newKimiPanel(t)
			a.kimi.mu.Lock()
			tc.hold(&a.kimi)
			a.kimi.mu.Unlock()
			tt.Frame()
			for _, want := range tc.want {
				wantKimiText(t, tt, want)
			}
			// The cost section reads local logs, so it is present in every state.
			wantKimiText(t, tt, tx(l, xKimiCostTitle))
		})
	}
}

func TestKimiPanelShowsTheLoadingState(t *testing.T) {
	l := l10n.New(l10n.SimplifiedChinese)
	a, tt := newKimiPanel(t)
	a.kimi.mu.Lock()
	a.kimi.loading = true
	a.kimi.mu.Unlock()
	tt.Frame()
	wantKimiText(t, tt, tx(l, xKimiLoading))
}

// TestKimiRefreshIsSingleFlight checks the guard that keeps a second refresh
// from asking the vendor endpoint while the first one is still running.
func TestKimiRefreshIsSingleFlight(t *testing.T) {
	t.Cleanup(setRunOnMain(func(fn func()) { fn() }))
	a, _ := newPanel(t, func(p *store.Preferences) {
		p.SelectedProvider = store.ProviderKimi
	}, panelFixture(), PanelActions{})
	a.kimi.mu.Lock()
	a.kimi.loading = true
	a.kimi.mu.Unlock()
	a.refreshKimi()
	a.kimi.mu.Lock()
	loading := a.kimi.loading
	a.kimi.mu.Unlock()
	if !loading {
		t.Fatal("a second refresh cleared the loading flag of the first one")
	}
}

func TestKimiScreenReportsReadiness(t *testing.T) {
	a, _ := newPanel(t, func(p *store.Preferences) {
		p.SelectedProvider = store.ProviderKimi
	}, panelFixture(), PanelActions{})
	if a.kimiScreen().ready() {
		t.Fatal("an empty hold is ready")
	}
	a.kimi.mu.Lock()
	a.kimi.loaded = true
	a.kimi.report = kimi.Report{State: kimi.StateReady}
	a.kimi.mu.Unlock()
	screen := a.kimiScreen()
	if !screen.ready() || screen.Prices == nil {
		t.Fatalf("screen = %+v", screen)
	}
	a.kimi.mu.Lock()
	a.kimi.report = kimi.Report{State: kimi.StateFailed, Err: errors.New("boom")}
	a.kimi.mu.Unlock()
	if a.kimiScreen().ready() {
		t.Fatal("a failed reading is ready")
	}
}
