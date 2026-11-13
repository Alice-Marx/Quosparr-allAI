package app

import (
	"errors"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/licoy/quosparr/internal/l10n"
	"github.com/licoy/quosparr/internal/opencode"
	"github.com/licoy/quosparr/internal/runway"
	"github.com/licoy/quosparr/internal/store"
)

// panelOpenCode is the OpenCode page: the Console account, the Zen/Go usage
// windows, the Console budget and the cost the CLI recorded. Every state the
// reading can reach has a rendering, because a signed-out machine is the
// ordinary case and not a failure.
func panelOpenCode(c *ui.Context, a *App, l l10n.Localizer, prefs store.Preferences, st *panelState) {
	s := a.openCodeScreen()
	now := time.Now()
	rs := st.resetSwitch(prefs.QuotaResetTimeFormat)
	first := true
	sectionBlock(c, &first, func() {
		drawSectionHead(c, l, st, sectionSpec{
			title:      tx(l, xOpenCodeTitle),
			caption:    openCodeCaption(l, s.Snap),
			refreshing: s.Loading,
			refresh:    func() { a.refreshOpenCode() },
		})
		switch {
		case s.Snap.State == opencode.StateReady:
			openCodeReading(c, l, s, now, rs)
		case s.Snap.State == opencode.StateNoCredential:
			openCodeMessage(c, l, tx(l, xOpenCodeNoCredential), tx(l, xOpenCodeNoCredentialDetail))
		case s.Snap.State == opencode.StateExpired, s.Snap.State == opencode.StateUnauthorized:
			openCodeMessage(c, l, tx(l, xOpenCodeExpired), tx(l, xOpenCodeExpiredDetail))
		case s.Snap.State == opencode.StateError:
			openCodeMessage(c, l, tx(l, xOpenCodeError), openCodeErrText(l, s.Snap.Err))
		default:
			sectionEmpty(c, l, s.Loading)
		}
	})
	openCodeSections(c, l, s, now, rs, &first)
}

// openCodeCaption is the header caption: the organization of a signed-in
// account.
func openCodeCaption(l l10n.Localizer, snap opencode.Snapshot) string {
	if snap.State != opencode.StateReady {
		return ""
	}
	if name := snap.Identity.Name(); name != "" {
		return name
	}
	return tx(l, xOpenCodeConsole)
}

func openCodeMessage(c *ui.Context, l l10n.Localizer, title, detail string) {
	t := c.Theme()
	cardColumn(c, surfaceSunkenKind, radiusCard).Gap(7).Padding(12).Children(func() {
		ui.Row(c).Gap(6).AlignItems(ui.Center).Children(func() {
			ui.Icon(c, icWarnCircle).Size(13, 13).TextColor(inkSecondary(t))
			ui.Text(c, title).FontSize(fsCallout).FontWeight(600)
		})
		dimText(c, detail)
		dimText(c, tx(l, xOpenCodeSourceNote))
	})
}

// openCodeReading is the identity line and the three usage windows.
func openCodeReading(c *ui.Context, l l10n.Localizer, s openCodeScreen, now time.Time, rs resetSwitch) {
	snap := s.Snap
	t := c.Theme()
	ui.Column(c).Gap(10).Children(func() {
		if name := snap.Identity.Name(); name != "" {
			ui.Row(c).Gap(6).AlignItems(ui.Center).Children(func() {
				ui.Icon(c, icKey).Size(13, 13).TextColor(inkSecondary(t))
				ui.Text(c, name).FontSize(fsCallout).FontWeight(600).SingleLine().Ellipsis("…").MaxWidth(240)
			})
		}
		switch {
		case snap.UsageErr != nil:
			dimText(c, tx(l, xOpenCodeUsageFailed)+" · "+openCodeErrText(l, snap.UsageErr))
		case snap.UsageOff:
			dimText(c, tx(l, xOpenCodeUsageOff))
		case !snap.HasUsage || !snap.Usage.Any():
			dimText(c, tx(l, xOpenCodeNoUsageData))
		default:
			openCodeWindowRow(c, l, tx(l, xOpenCodeRolling), snap.Usage.Rolling, now, rs, "reset-opencode-rolling")
			openCodeWindowRow(c, l, tx(l, xOpenCodeWeekly), snap.Usage.Weekly, now, rs, "reset-opencode-weekly")
			openCodeWindowRow(c, l, tx(l, xOpenCodeMonthly), snap.Usage.Monthly, now, rs, "reset-opencode-monthly")
		}
		if !snap.FetchedAt.IsZero() {
			dimText(c, tx(l, xOpenCodeFetched, snap.FetchedAt.Local().Format("2006-01-02 15:04")))
		}
	})
}

// openCodeWindowRow is one quota window: label, used share, bar and reset.
func openCodeWindowRow(c *ui.Context, l l10n.Localizer, title string, w opencode.Window, now time.Time, rs resetSwitch, key string) {
	t := c.Theme()
	ui.Column(c).Gap(4).Children(func() {
		ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
			ui.Text(c, title).FontSize(fsCaption).TextColor(inkSecondary(t)).Grow(1).Basis(0)
			if w.Limited {
				ui.Text(c, tx(l, xOpenCodeLimited)).FontSize(fsCaption).TextColor(healthInk(t, runway.HealthRed)).SingleLine()
			}
			if w.HasPercent {
				ui.Text(c, tx(l, xOpenCodeUsed, opencode.FormatPercent(w.Percent))).FontSize(fsCaption).FontWeight(600).
					FontFeatures("tnum").TextColor(healthInk(t, openCodeHealth(w.Percent))).SingleLine()
			} else {
				ui.Text(c, tx(l, xOpenCodeNoData)).FontSize(fsCaption).TextColor(inkSecondary(t)).SingleLine()
			}
		})
		if w.HasPercent {
			usedFractionBar(c, w.Percent/100, openCodeHealth(w.Percent))
		}
		if w.HasReset {
			ui.Row(c).Children(func() {
				ui.Spacer(c).Grow(1)
				resetLine(c, l, key, w.ResetsAt, now, rs)
			})
		}
	})
}

func openCodeHealth(used float64) runway.Health {
	rounded := int(used + 0.5)
	if rounded < 0 {
		rounded = 0
	}
	if rounded > 100 {
		rounded = 100
	}
	return runway.HealthForUsedPercent(rounded)
}

// openCodeErrText names a failure without repeating a header or a token.
func openCodeErrText(l l10n.Localizer, err error) string {
	switch {
	case err == nil:
		return ""
	case errors.Is(err, opencode.ErrDecode):
		return tx(l, xOpenCodeErrDecode)
	case errors.Is(err, opencode.ErrUnauthorized):
		return tx(l, xOpenCodeExpiredDetail)
	case errors.Is(err, opencode.ErrDatabase), errors.Is(err, opencode.ErrCredential):
		return tx(l, xOpenCodeDatabaseFailed)
	default:
		return tx(l, xOpenCodeErrRequest)
	}
}
