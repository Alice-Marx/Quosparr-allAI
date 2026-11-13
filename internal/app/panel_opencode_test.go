package app

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/licoy/quosparr/internal/l10n"
	"github.com/licoy/quosparr/internal/opencode"
	"github.com/licoy/quosparr/internal/store"
)

// openCodeReadySnap is a signed-in reading with all three sections filled.
func openCodeReadySnap() opencode.Snapshot {
	at := time.Date(2026, 11, 2, 10, 0, 0, 0, time.Local)
	return opencode.Snapshot{
		FetchedAt: at,
		State:     opencode.StateReady,
		Identity:  opencode.Identity{OrgID: "org_1", OrgName: "Team Console", UsageURL: "https://opencode.ai/console/org_1/activity/overview"},
		Usage: opencode.Usage{
			Rolling: opencode.Window{Percent: 1, HasPercent: true, ResetsAt: at.Add(3 * time.Hour), HasReset: true},
			Weekly:  opencode.Window{Percent: 8.25, HasPercent: true, ResetsAt: at.Add(72 * time.Hour), HasReset: true},
			Monthly: opencode.Window{Percent: 75, HasPercent: true, Limited: true, ResetsAt: at.Add(240 * time.Hour), HasReset: true},
		},
		HasUsage:     true,
		BudgetsKnown: true,
		Budgets: []opencode.Budget{{
			ID: "r1", LimitUSD: 50, SpentUSD: 12.34, HasLimit: true, Scope: "org", Per: "month",
			ResetsAt: at.Add(240 * time.Hour), HasReset: true,
		}},
		HasCost: true,
		Cost: opencode.Cost{
			TotalUSD: 3.2776, Sessions: 53,
			Tokens: opencode.Tokens{Input: 88749, Output: 48840, Reasoning: 1105323, CacheRead: 467710541, CacheWrite: 18798966},
			Models: []opencode.ModelCost{
				{Provider: "opencode-go", Model: "kimi-k3.1-pro", CostUSD: 3.277566, Sessions: 41},
				{Provider: "opencode", Model: "deepseek-v4.1-flash-free", CostUSD: 6.3e-05, Sessions: 12},
			},
		},
	}
}

func openCodePanel(t *testing.T) (*App, *ui.Tester) {
	t.Helper()
	a, tt := newPanel(t, func(p *store.Preferences) {
		p.Language = store.LanguageSimplifiedChinese
		p.SelectedProvider = store.ProviderOpenCode
	}, panelFixture(), PanelActions{})
	return a, tt
}

func setOpenCodeSnap(a *App, snap opencode.Snapshot) {
	a.opencode.mu.Lock()
	a.opencode.snap = snap
	a.opencode.loaded = true
	a.opencode.mu.Unlock()
}

func TestOpenCodePanelRendersIdentityQuotaBudgetAndCost(t *testing.T) {
	a, tt := openCodePanel(t)
	l := l10n.New(l10n.SimplifiedChinese)
	tt.Frame()
	if !hasExact(tt, tx(l, xOpenCodeTitle)) {
		t.Fatalf("title missing: %q", tt.Texts())
	}
	setOpenCodeSnap(a, openCodeReadySnap())
	tt.Frame()
	for _, want := range []string{
		"Team Console",
		tx(l, xOpenCodeRolling),
		tx(l, xOpenCodeWeekly),
		tx(l, xOpenCodeMonthly),
		tx(l, xOpenCodeUsed, "75"),
		tx(l, xOpenCodeUsed, "8.3"),
		tx(l, xOpenCodeLimited),
		tx(l, xOpenCodeBudgetSpent, "$12.34", "$50"),
		tx(l, xOpenCodeBudgetRule, "org", "month"),
		tx(l, xOpenCodeCostSessions, "53"),
		"$3.28",
		"opencode-go/kimi-k3.1-pro",
		tx(l, xOpenCodeCostTokens, "88749", "48840", "1105323", "467710541", "18798966"),
	} {
		if !tt.HasText(want) {
			t.Fatalf("missing %q in %q", want, tt.Texts())
		}
	}
	for _, want := range []string{tx(l, xOpenCodeTitle), tx(l, xOpenCodeBudgetTitle), tx(l, xOpenCodeCostTitle)} {
		if !hasExact(tt, want) {
			t.Fatalf("section title %q missing: %q", want, tt.Texts())
		}
	}
}

func TestOpenCodePanelRendersEveryState(t *testing.T) {
	l := l10n.New(l10n.SimplifiedChinese)
	usageFailed := openCodeReadySnap()
	usageFailed.UsageErr = opencode.ErrDecode
	usageOff := openCodeReadySnap()
	usageOff.UsageOff = true
	usageOff.HasUsage = false
	budgetsOff := openCodeReadySnap()
	budgetsOff.BudgetsOff = true
	budgetsOff.Budgets = nil

	cases := []struct {
		name string
		snap opencode.Snapshot
		want string
		gone string
	}{
		{"signed in", openCodeReadySnap(), tx(l, xOpenCodeBudgetSpent, "$12.34", "$50"), tx(l, xOpenCodeNoCredential)},
		{"no credential", opencode.Snapshot{State: opencode.StateNoCredential}, tx(l, xOpenCodeNoCredential), tx(l, xOpenCodeExpired)},
		{"expired", opencode.Snapshot{State: opencode.StateExpired}, tx(l, xOpenCodeExpired), tx(l, xOpenCodeNoCredential)},
		{"unauthorized", opencode.Snapshot{State: opencode.StateUnauthorized, Err: opencode.ErrUnauthorized}, tx(l, xOpenCodeExpired), ""},
		{"local failure", opencode.Snapshot{State: opencode.StateError, Err: errors.New("boom")}, tx(l, xOpenCodeError), ""},
		{"usage failed", usageFailed, tx(l, xOpenCodeUsageFailed), ""},
		{"usage off", usageOff, tx(l, xOpenCodeUsageOff), ""},
		{"budgets off", budgetsOff, tx(l, xOpenCodeBudgetDisabled), ""},
		{"no reading yet", opencode.Snapshot{}, tx(l, xOpenCodeNoCredential), ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			a, tt := openCodePanel(t)
			setOpenCodeSnap(a, tc.snap)
			tt.Frame()
			want := tc.want
			if tc.name == "no reading yet" {
				want = l.Text(l10n.KeyNotLoaded)
			}
			if !tt.HasText(want) {
				t.Fatalf("missing %q in %q", want, tt.Texts())
			}
			if tc.gone != "" && hasExact(tt, tc.gone) {
				t.Fatalf("%q must not be shown in %s: %q", tc.gone, tc.name, tt.Texts())
			}
			// The cost section stays readable without a login: the figures are
			// local.
			if tc.name == "no credential" {
				if hasExact(tt, tx(l, xOpenCodeCostTitle)) == false {
					t.Fatalf("cost section missing when signed out: %q", tt.Texts())
				}
			}
		})
	}
}

func TestOpenCodeRefreshReadsTheLocalDatabase(t *testing.T) {
	t.Cleanup(setRunOnMain(func(fn func()) { fn() }))
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	a, _ := openCodePanel(t)
	home := t.TempDir()
	a.opencode.mu.Lock()
	a.opencode.home = home
	a.opencode.mu.Unlock()
	a.recordOpenCode(context.Background(), home)
	s := a.openCodeScreen()
	if s.Snap.State != opencode.StateNoCredential {
		t.Fatalf("state = %v err=%v", s.Snap.State, s.Snap.Err)
	}
	if !s.Loaded || s.Loading {
		t.Fatalf("screen = %+v", s)
	}
	if !strings.HasSuffix(s.Snap.DatabasePath, "opencode.db") {
		t.Fatalf("database path = %q", s.Snap.DatabasePath)
	}
}

// refreshOpenCode is the entry the section refresh and the provider tab share:
// it must reread off the UI thread and replace a stale reading.
func TestOpenCodeRefreshReloadsTheAccount(t *testing.T) {
	t.Cleanup(setRunOnMain(func(fn func()) { fn() }))
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	a, _ := openCodePanel(t)
	setOpenCodeSnap(a, openCodeReadySnap())
	a.opencode.mu.Lock()
	a.opencode.home = t.TempDir()
	a.opencode.mu.Unlock()
	a.refreshOpenCode()
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		if a.openCodeScreen().Snap.State == opencode.StateNoCredential {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("refresh kept the stale reading: %+v", a.openCodeScreen().Snap)
}
