package app

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/licoy/quosparr/internal/glm"
	"github.com/licoy/quosparr/internal/l10n"
	"github.com/licoy/quosparr/internal/store"
)

// glmFakeQuota is a reading of the documented shape: the share is a used
// percentage from 0 to 100 and the reset is a millisecond epoch.
func glmFakeQuota() glm.Quota {
	reset := time.Date(2026, 12, 11, 12, 0, 0, 0, time.UTC)
	return glm.Quota{
		Level: "glm-coding-plan",
		Limits: []glm.Limit{
			{Type: glm.PrimaryLimitType, Percent: 63, HasPercent: true, ResetAt: reset, HasReset: true},
			{Type: "MCP_USAGE_LIMIT", Percent: 12, HasPercent: true},
		},
		Primary:    glm.Limit{Type: glm.PrimaryLimitType, Percent: 63, HasPercent: true, ResetAt: reset, HasReset: true},
		HasPrimary: true,
		Endpoint:   glm.BigModelQuotaURL,
		FetchedAt:  time.Date(2026, 11, 2, 10, 0, 0, 0, time.UTC),
	}
}

func glmFakeCost() glm.Cost {
	return glm.Cost{
		Rows: []glm.Row{
			{Model: "GLM-5.3", Requests: 1793, Input: 376635945, Output: 1495567, CacheRead: 365827072, HasPrice: true, USD: 629.13},
			{Model: "GLM-5.3-Highspeed", Requests: 4, Input: 100, Output: 10, CacheRead: 0},
		},
		Unpriced: 1,
		TotalUSD: 629.13,
	}
}

func glmPanel(t *testing.T, mutate func(*glmHold)) (*App, *ui.Tester, l10n.Localizer) {
	t.Helper()
	a, tt := newPanel(t, func(p *store.Preferences) {
		p.Language = store.LanguageSimplifiedChinese
		p.SelectedProvider = store.ProviderGLM
	}, panelFixture(), PanelActions{})
	a.glm.mu.Lock()
	a.glm.credPath = filepath.Join(t.TempDir(), "credentials.json")
	a.glm.dbPath = filepath.Join(t.TempDir(), "db.sqlite")
	if mutate != nil {
		mutate(&a.glm)
	}
	a.glm.mu.Unlock()
	tt.Frame()
	return a, tt, l10n.New(l10n.SimplifiedChinese)
}

func TestGLMPanelWithoutACredential(t *testing.T) {
	_, tt, l := glmPanel(t, func(h *glmHold) {
		h.quotaErr = glm.ErrNoCredentials
	})
	for _, want := range []string{
		tx(l, xGLMTitle), tx(l, xGLMEmptyTitle), tx(l, xGLMCredentialNone),
		tx(l, xGLMCostTitle), tx(l, xGLMSourceNote),
	} {
		if !tt.HasText(want) {
			t.Fatalf("missing %q in %q", want, tt.Texts())
		}
	}
	if tt.HasText(l.Text(l10n.KeyNextResetIn)) || hasExact(tt, tx(l, xGrokBotUsed, "63")) {
		t.Fatalf("a missing credential drew a quota bar: %q", tt.Texts())
	}
}

func TestGLMPanelWithAnExpiredCredential(t *testing.T) {
	_, tt, l := glmPanel(t, func(h *glmHold) {
		h.candidate = glm.Candidate{Key: "stale", Kind: glm.KindCodingPlan, Plan: "bigmodel-individual-coding-plan"}
		h.quotaErr = glm.ErrUnauthorized
	})
	for _, want := range []string{
		tx(l, xGLMExpiredTitle), tx(l, xGLMExpiredDetail), tx(l, xGLMCredentialCodingPlan), "bigmodel.cn",
		tx(l, xGLMCostTitle),
	} {
		if !tt.HasText(want) {
			t.Fatalf("missing %q in %q", want, tt.Texts())
		}
	}
}

func TestGLMPanelWithAnEndpointError(t *testing.T) {
	_, tt, l := glmPanel(t, func(h *glmHold) {
		h.candidate = glm.Candidate{Key: "k", Kind: glm.KindEnvKey, Plan: "BIGMODEL_USAGE_API_KEY"}
		h.quotaErr = fmt.Errorf("%w: 502", glm.ErrBadResponse)
	})
	for _, want := range []string{tx(l, xGLMErrorTitle), tx(l, xGLMErrHTTP), tx(l, xGLMCredentialEnv)} {
		if !tt.HasText(want) {
			t.Fatalf("missing %q in %q", want, tt.Texts())
		}
	}
	if tt.HasText(tx(l, xGLMExpiredDetail)) {
		t.Fatal("an HTTP error was reported as an expired credential")
	}
}

func TestGLMPanelRendersTheQuotaAndTheCost(t *testing.T) {
	fetched := time.Date(2026, 11, 2, 10, 0, 0, 0, time.UTC)
	_, tt, l := glmPanel(t, func(h *glmHold) {
		h.candidate = glm.Candidate{Key: "k", Kind: glm.KindCodingPlan, Plan: "bigmodel-individual-coding-plan", Organization: "368617"}
		h.quota = glmFakeQuota()
		h.cost = glmFakeCost()
		h.fetchedAt = fetched
	})
	for _, want := range []string{
		"已用 63%",
		"剩余 37%",
		"glm-coding-plan",
		"MCP_USAGE_LIMIT",
		"已用 12%",
		l.Text(l10n.KeyNextResetIn),
		tx(l, xGLMFetched, fetched.Local().Format("2006-01-02 15:04")),
		tx(l, xGLMCostTitle),
		tx(l, xGLMCostTotal, "$629.13"),
		tx(l, xGLMCostUnpriced, "1"),
		"GLM-5.3",
		"GLM-5.3-Highspeed",
		tx(l, xGrokBotCostMissingPrice),
		tx(l, xGLMCredentialCodingPlan),
		"368617",
		glm.BigModelQuotaURL,
	} {
		if !tt.HasText(want) {
			t.Fatalf("missing %q in %q", want, tt.Texts())
		}
	}
	if !hasExact(tt, "已用 63%") {
		t.Fatalf("the used share is not its own label: %q", tt.Texts())
	}
	if tt.HasText(tx(l, xGLMEmptyTitle)) || tt.HasText(tx(l, xGLMErrorTitle)) {
		t.Fatalf("a reading drew an empty or error card: %q", tt.Texts())
	}
}

func TestGLMPanelWithoutAQuotaBucket(t *testing.T) {
	_, tt, l := glmPanel(t, func(h *glmHold) {
		h.candidate = glm.Candidate{Key: "k", Kind: glm.KindCodingPlan, Plan: "bigmodel-individual-coding-plan"}
		h.quota = glm.Quota{Level: "glm-coding-plan"}
	})
	if !tt.HasText(tx(l, xGLMErrorTitle)) {
		t.Fatalf("an empty envelope drew a reading: %q", tt.Texts())
	}
	if hasExact(tt, tx(l, xGrokBotUsed, "0")) {
		t.Fatalf("an empty envelope claimed nothing was used: %q", tt.Texts())
	}
}

func TestGLMPanelOffersTheRefreshControl(t *testing.T) {
	_, tt, l := glmPanel(t, func(h *glmHold) {
		h.quota = glmFakeQuota()
		h.cost = glmFakeCost()
		h.fetchedAt = h.quota.FetchedAt
	})
	if _, found := tt.Find(l.Text(l10n.KeyRefresh)); !found {
		t.Fatalf("no refresh control in %q", tt.Texts())
	}
}

// glmSuccessBody is the documented success envelope.
const glmSuccessBody = `{"code":0,"success":true,"data":{"level":"glm-coding-plan","limits":[
  {"type":"TIME_LIMIT","percentage":63,"nextResetTime":1794362501515}]}}`

func TestGLMLoadReadsTheStoreAndTheEndpoint(t *testing.T) {
	clearGLMEnvKeys(t)
	home := t.TempDir()
	storePath := glm.CredentialPath(home)
	if err := os.MkdirAll(filepath.Dir(storePath), 0o700); err != nil {
		t.Fatal(err)
	}
	storeBody := `{"account-provider:coding-plan:account:bigmodel-individual-coding-plan:account:42:api-key":"secret-value"}`
	if err := os.WriteFile(storePath, []byte(storeBody), 0o600); err != nil {
		t.Fatal(err)
	}
	var seen string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen = r.Header.Get("authorization")
		_, _ = w.Write([]byte(glmSuccessBody))
	}))
	t.Cleanup(srv.Close)
	t.Cleanup(setGLMClient(glm.Client{HTTP: srv.Client(), QuotaURL: srv.URL}))
	a, _ := newPanel(t, func(p *store.Preferences) {
		p.Language = store.LanguageSimplifiedChinese
		p.SelectedProvider = store.ProviderGLM
	}, panelFixture(), PanelActions{})
	a.loadGLM(context.Background(), home)
	s := a.glmScreen()
	if s.QuotaErr != nil {
		t.Fatalf("quota error: %v", s.QuotaErr)
	}
	if !s.ready() || s.Quota.Primary.Percent != 63 {
		t.Fatalf("quota = %+v", s.Quota)
	}
	if s.Candidate.Kind != glm.KindCodingPlan || s.Candidate.Organization != "42" {
		t.Fatalf("candidate = %s", s.Candidate)
	}
	if seen != "secret-value" {
		t.Fatalf("authorization = %q; the monitor API takes the credential verbatim", seen)
	}
	if !errors.Is(s.CostErr, glm.ErrNoUsage) {
		t.Fatalf("cost error = %v, want a missing database", s.CostErr)
	}
	if s.CredPath != storePath || s.DBPath != glm.DatabasePath(home) {
		t.Fatalf("paths = %q %q", s.CredPath, s.DBPath)
	}
}

func TestGLMLoadWithoutACredentialNeverCallsTheEndpoint(t *testing.T) {
	clearGLMEnvKeys(t)
	home := t.TempDir()
	sent := false
	t.Cleanup(setGLMClient(glm.Client{HTTP: &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		sent = true
		return nil, errors.New("no request expected")
	})}}))
	a, _ := newPanel(t, func(p *store.Preferences) {
		p.Language = store.LanguageSimplifiedChinese
		p.SelectedProvider = store.ProviderGLM
	}, panelFixture(), PanelActions{})
	a.loadGLM(context.Background(), home)
	s := a.glmScreen()
	if !errors.Is(s.QuotaErr, glm.ErrNoCredentials) {
		t.Fatalf("quota error = %v", s.QuotaErr)
	}
	if s.Candidate.Key != "" {
		t.Fatal("a candidate appeared without a credential")
	}
	if sent {
		t.Fatal("a request was sent without a credential")
	}
}

// setGLMClient replaces the panel client and returns what restores it.
func setGLMClient(client glm.Client) (restore func()) {
	old := newGLMClient
	newGLMClient = func() glm.Client { return client }
	return func() { newGLMClient = old }
}

// clearGLMEnvKeys hides the environment overrides of the credential store.
func clearGLMEnvKeys(t *testing.T) {
	t.Helper()
	t.Setenv("ZCODE_BIGMODEL_USAGE_API_KEY", "")
	t.Setenv("BIGMODEL_USAGE_API_KEY", "")
}
