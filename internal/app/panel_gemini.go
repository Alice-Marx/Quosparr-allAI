package app

import (
	"errors"
	"strconv"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/licoy/quosparr/internal/gemini"
	"github.com/licoy/quosparr/internal/l10n"
	"github.com/licoy/quosparr/internal/store"
)

// panelGemini is the Gemini Code Assist quota page. Unlike the snapshot tabs it
// reads the vendor endpoints live, with the credential the Gemini CLI already
// stored; its cost section reads the CLI's local chat logs.
func panelGemini(c *ui.Context, a *App, l l10n.Localizer, prefs store.Preferences, st *panelState) {
	s := a.geminiScreen()
	rs := st.resetSwitch(prefs.QuotaResetTimeFormat)
	first := true
	sectionBlock(c, &first, func() {
		geminiIdentityRow(c, l, s)
		drawSectionHead(c, l, st, sectionSpec{
			title:      tx(l, xGeminiTitle),
			caption:    s.Reading.Tier.ID,
			refreshing: s.Busy,
			refresh:    func() { a.refreshGemini() },
		})
		geminiReading(c, l, s, rs)
	})
	geminiCostSection(c, l, s, &first)
}

// geminiIdentityRow names the product and the tier the service reported.
func geminiIdentityRow(c *ui.Context, l l10n.Localizer, s geminiScreen) {
	name := s.Reading.Tier.Name
	if name == "" {
		name = tx(l, xGeminiTab)
	}
	t := c.Theme()
	ui.Row(c).Gap(6).AlignItems(ui.Center).Children(func() {
		ui.Icon(c, icGlobe).Size(14, 14).TextColor(inkSecondary(t))
		ui.Text(c, name).FontSize(fsCallout).FontWeight(600).SingleLine().Ellipsis("…").MaxWidth(240)
	})
}

// geminiReading draws one state. Only StatusError takes the danger style: no
// credential, an expired sign-in and a missing subscription are normal answers.
func geminiReading(c *ui.Context, l l10n.Localizer, s geminiScreen, rs resetSwitch) {
	switch s.Reading.Status {
	case gemini.StatusReady:
		geminiBuckets(c, l, s.Reading.Buckets, rs)
		if !s.Reading.FetchedAt.IsZero() {
			dimText(c, tx(l, xGrokBotFetched, s.Reading.FetchedAt.Local().Format("2006-01-02 15:04")))
		}
	case gemini.StatusNoSubscription:
		geminiMessage(c, l, false, tx(l, xGeminiNoSubscriptionTitle), tx(l, xGeminiNoSubscriptionDetail))
		geminiReason(c, l, s.Reading.Reason)
	case gemini.StatusNoCreds:
		geminiMessage(c, l, false, tx(l, xGeminiNoCredsTitle), tx(l, xGeminiNoCredsDetail, gemini.CredsPath(s.Home)))
	case gemini.StatusExpired:
		geminiMessage(c, l, false, tx(l, xGeminiExpiredTitle), geminiExpiredDetail(l, s))
	case gemini.StatusError:
		geminiMessage(c, l, true, tx(l, xGeminiErrorTitle), geminiErrText(l, s.Reading.Err))
		geminiReason(c, l, s.Reading.Reason)
	default:
		if s.Busy || !s.Fetched {
			calloutDim(c, tx(l, xPanelLoading))
			return
		}
		calloutDim(c, l.Text(l10n.KeyNotLoaded))
	}
	dimText(c, tx(l, xGeminiSourceNote))
}

// geminiExpiredDetail names both ways out of an expired sign-in. Without a
// configured OAuth client this page will not refresh anything, so the text says
// what the user can do instead of hiding the reason behind "reading failed".
func geminiExpiredDetail(l l10n.Localizer, s geminiScreen) string {
	if errors.Is(s.Reading.Err, gemini.ErrNoClientSecret) {
		return tx(l, xGeminiNoSecretDetail, gemini.CredsPath(s.Home))
	}
	return tx(l, xGeminiExpiredDetail, gemini.CredsPath(s.Home))
}

func geminiMessage(c *ui.Context, l l10n.Localizer, danger bool, title, detail string) {
	t := c.Theme()
	titleInk := t.Text
	if danger {
		titleInk = t.Danger
	}
	cardColumn(c, surfaceSunkenKind, radiusCard).Gap(7).Padding(12).Children(func() {
		ui.Row(c).Gap(6).AlignItems(ui.Center).Children(func() {
			ui.Icon(c, icGlobe).Size(13, 13).TextColor(inkSecondary(t))
			ui.Text(c, title).FontSize(fsCallout).FontWeight(600).TextColor(titleInk)
		})
		dimText(c, detail)
	})
}

func geminiReason(c *ui.Context, l l10n.Localizer, reason string) {
	if reason == "" {
		return
	}
	dimText(c, tx(l, xGeminiReason, reason))
}

// geminiBuckets draws one bar per model bucket, worst first. A bucket without
// a remainingFraction has no percentage in the answer, so it keeps its request
// count and no bar.
func geminiBuckets(c *ui.Context, l l10n.Localizer, buckets []gemini.Bucket, rs resetSwitch) {
	if len(buckets) == 0 {
		dimText(c, tx(l, xGeminiNoBuckets))
		return
	}
	now := time.Now()
	t := c.Theme()
	ui.Column(c).Gap(10).Children(func() {
		for _, bucket := range buckets {
			used, hasUsed := bucket.UsedPercent()
			if hasUsed {
				health := grokBotHealth(used)
				ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
					ui.Text(c, bucket.Model).FontSize(fsCaption).FontWeight(600).Grow(1).Basis(0).SingleLine().Ellipsis("…")
					ui.Text(c, tx(l, xGrokBotUsed, gemini.FormatPercent(used))).FontSize(fsCaption).FontWeight(600).FontFeatures("tnum").TextColor(healthInk(t, health)).SingleLine()
				})
				usedFractionBar(c, used/100, health)
			} else {
				ui.Text(c, bucket.Model).FontSize(fsCaption).FontWeight(600).SingleLine().Ellipsis("…")
			}
			if bucket.HasLimit {
				dimText(c, tx(l, xGeminiRequests, strconv.FormatInt(bucket.Remaining, 10), strconv.FormatInt(bucket.Limit, 10)))
			}
			if bucket.HasReset {
				ui.Row(c).Children(func() {
					ui.Spacer(c).Grow(1)
					resetLine(c, l, "reset-gemini-"+bucket.Model, bucket.ResetAt, now, rs)
				})
			}
		}
	})
}

// geminiCostSection prices the local token spend. It is drawn in every state:
// the chat logs are readable even when the quota endpoint refuses.
func geminiCostSection(c *ui.Context, l l10n.Localizer, s geminiScreen, first *bool) {
	sectionBlock(c, first, func() {
		drawSectionHead(c, l, nil, sectionSpec{title: tx(l, xGrokBotCostTitle)})
		rows, ok := gemini.CostRows(s.Reading.Cost)
		if !ok {
			dimText(c, tx(l, xGeminiCostEmpty))
			return
		}
		for _, row := range rows {
			name := row.Model
			if name == "" {
				name = tx(l, xGrokBotCostUntitled)
			}
			amount := "—"
			if row.Amount != "" {
				amount = row.Amount
			}
			ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
				ui.Text(c, name).FontSize(fsCaption).FontWeight(600).Grow(1).Basis(0).SingleLine().Ellipsis("…")
				ui.Text(c, amount).FontSize(fsCaption).FontFeatures("tnum").SingleLine()
			})
			dimText(c, tx(l, xGrokBotCostTokens,
				strconv.FormatInt(row.Input, 10),
				strconv.FormatInt(row.Output, 10),
				strconv.FormatInt(row.Cached, 10)))
			if row.Missing {
				dimText(c, tx(l, xGrokBotCostMissingPrice))
			}
			if row.Estimated {
				dimText(c, tx(l, xGeminiCostEstimate))
			}
		}
		dimText(c, tx(l, xGeminiSessions, strconv.Itoa(s.Reading.Cost.Sessions)))
	})
}

func geminiErrText(l l10n.Localizer, err error) string {
	var httpErr *gemini.HTTPError
	switch {
	case errors.As(err, &httpErr):
		return tx(l, xGeminiErrHTTP, strconv.Itoa(httpErr.Status))
	case errors.Is(err, gemini.ErrNetwork):
		return tx(l, xGeminiErrNetwork)
	case errors.Is(err, gemini.ErrUnauthorized), errors.Is(err, gemini.ErrRefreshRejected), errors.Is(err, gemini.ErrNoRefreshToken):
		return tx(l, xGeminiErrSignIn)
	case errors.Is(err, gemini.ErrBadResponse):
		return tx(l, xGeminiErrResponse)
	case errors.Is(err, gemini.ErrUnreadable), errors.Is(err, gemini.ErrBadJSON), errors.Is(err, gemini.ErrTooLarge):
		return tx(l, xGeminiErrCreds)
	case errors.Is(err, gemini.ErrRefreshFailed):
		return tx(l, xGeminiErrGeneric)
	default:
		return tx(l, xGeminiErrGeneric)
	}
}
