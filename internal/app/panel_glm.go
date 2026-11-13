package app

import (
	"errors"
	"strings"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/licoy/quosparr/internal/glm"
	"github.com/licoy/quosparr/internal/l10n"
	"github.com/licoy/quosparr/internal/store"
)

// panelGLM is the GLM Coding Plan quota page. The quota is a live vendor
// reading and the cost is priced from the local ZCode database; the
// credential that both use stays in the ZCode credential store.
func panelGLM(c *ui.Context, a *App, l l10n.Localizer, prefs store.Preferences, st *panelState) {
	s := a.glmScreen()
	first := true
	glmIdentityRow(c, l, s)
	sectionBlock(c, &first, func() {
		drawSectionHead(c, l, st, sectionSpec{
			title:      tx(l, xGLMTitle),
			caption:    s.Quota.Level,
			refreshing: s.Running,
			refresh:    func() { a.refreshGLM() },
		})
		switch {
		case errors.Is(s.QuotaErr, glm.ErrNoCredentials):
			glmMessage(c, l, tx(l, xGLMEmptyTitle), tx(l, xGLMEmptyDetail, s.CredPath), "")
		case errors.Is(s.QuotaErr, glm.ErrUnauthorized):
			glmMessage(c, l, tx(l, xGLMExpiredTitle), tx(l, xGLMExpiredDetail), "")
		case s.QuotaErr != nil:
			glmMessage(c, l, tx(l, xGLMErrorTitle), glmErrText(l, s.QuotaErr), s.CredPath)
		case !s.ready():
			// A success envelope with no bucket is not a reading; it must not
			// draw an empty bar as if nothing were used.
			glmMessage(c, l, tx(l, xGLMErrorTitle), tx(l, xGLMErrDecode), s.CredPath)
		default:
			glmReading(c, l, s, st.resetSwitch(prefs.QuotaResetTimeFormat))
		}
	})
	glmCostSection(c, l, s, &first)
}

// glmIdentityRow names the product and the credential the panel read. The
// quota tabs that are not the Grok family draw their own identity row here,
// because panelHeader leaves the body to them.
func glmIdentityRow(c *ui.Context, l l10n.Localizer, s glmScreen) {
	t := c.Theme()
	parts := []string{glmCredentialText(l, s.Candidate)}
	if s.Candidate.Key != "" {
		parts = append(parts, s.Candidate.Region.Host())
	}
	if s.Candidate.Organization != "" {
		parts = append(parts, s.Candidate.Organization)
	}
	ui.Row(c).Gap(6).AlignItems(ui.Center).Children(func() {
		ui.Icon(c, icCpu).Size(14, 14).TextColor(inkSecondary(t))
		ui.Text(c, tx(l, xGLMTab)).FontSize(fsCallout).FontWeight(600).SingleLine()
		ui.Text(c, strings.Join(parts, " · ")).FontSize(fsCaption).TextColor(inkSecondary(t)).SingleLine().Ellipsis("…").Grow(1).Basis(0)
	})
}

// glmCredentialText names where the credential came from.
func glmCredentialText(l l10n.Localizer, cand glm.Candidate) string {
	switch cand.Kind {
	case glm.KindEnvKey:
		return tx(l, xGLMCredentialEnv)
	case glm.KindCodingPlan:
		return tx(l, xGLMCredentialCodingPlan)
	case glm.KindOAuth:
		return tx(l, xGLMCredentialOAuth)
	}
	return tx(l, xGLMCredentialNone)
}

// glmMessage is the sunken card of a missing or failed reading.
func glmMessage(c *ui.Context, l l10n.Localizer, title, detail, path string) {
	t := c.Theme()
	cardColumn(c, surfaceSunkenKind, radiusCard).Gap(7).Padding(12).Children(func() {
		ui.Row(c).Gap(6).AlignItems(ui.Center).Children(func() {
			ui.Icon(c, icCpu).Size(13, 13).TextColor(inkSecondary(t))
			ui.Text(c, title).FontSize(fsCallout).FontWeight(600)
		})
		dimText(c, detail)
		if path != "" {
			dimText(c, path)
		}
		dimText(c, tx(l, xGLMSourceNote))
	})
}

// glmReading is the quota share of one accepted reading.
func glmReading(c *ui.Context, l l10n.Localizer, s glmScreen, rs resetSwitch) {
	quota := s.Quota
	now := time.Now()
	used := glm.FormatPercent(quota.Primary.Percent)
	left := glm.FormatPercent(100 - quota.Primary.Percent)
	t := c.Theme()
	ui.Column(c).Gap(10).Children(func() {
		dimText(c, tx(l, xGLMQuotaSource)+" · "+quota.Endpoint)
		ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
			ui.Text(c, tx(l, xGrokBotUsed, used)).FontSize(fsCallout).FontWeight(600).Grow(1).Basis(0)
			ui.Text(c, tx(l, xGrokBotLeft, left)).FontSize(fsCallout).FontWeight(600).FontFeatures("tnum").TextColor(healthInk(t, grokBotHealth(quota.Primary.Percent))).SingleLine()
		})
		usedFractionBar(c, quota.Primary.Percent/100, grokBotHealth(quota.Primary.Percent))
		if quota.Primary.HasReset {
			ui.Row(c).Children(func() {
				ui.Spacer(c).Grow(1)
				resetLine(c, l, "reset-glm", quota.Primary.ResetAt, now, rs)
			})
		}
		if quota.Primary.Derived {
			dimText(c, tx(l, xGLMDerived))
		}
		glmOtherBuckets(c, l, quota)
		dimText(c, tx(l, xGLMFetched, s.FetchedAt.Local().Format("2006-01-02 15:04")))
	})
}

// glmOtherBuckets lists the buckets other than the headline one, by the type
// the endpoint spelled.
func glmOtherBuckets(c *ui.Context, l l10n.Localizer, quota glm.Quota) {
	t := c.Theme()
	for _, limit := range quota.Limits {
		if limit.Type == quota.Primary.Type {
			continue
		}
		ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
			ui.Text(c, limit.Type).FontSize(fsCaption).TextColor(inkSecondary(t)).SingleLine().Grow(1).Basis(0)
			ui.Text(c, tx(l, xGrokBotUsed, glm.FormatPercent(limit.Percent))).FontSize(fsCaption).FontFeatures("tnum").SingleLine()
		})
	}
}

// glmErrText explains a failed reading without quoting the response body.
func glmErrText(l l10n.Localizer, err error) string {
	switch {
	case errors.Is(err, glm.ErrUnauthorized):
		return tx(l, xGLMExpiredDetail)
	case errors.Is(err, glm.ErrNetwork):
		return tx(l, xGLMErrNetwork)
	case errors.Is(err, glm.ErrBadResponse):
		return tx(l, xGLMErrHTTP)
	case errors.Is(err, glm.ErrBusinessCode):
		return tx(l, xGLMErrCode)
	case errors.Is(err, glm.ErrCredentialFormat):
		return tx(l, xGLMErrStore)
	default:
		return tx(l, xGLMErrDecode)
	}
}
