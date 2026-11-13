package app

import (
	"strconv"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/licoy/quosparr/internal/l10n"
	"github.com/licoy/quosparr/internal/opencode"
)

func openCodeSections(c *ui.Context, l l10n.Localizer, s openCodeScreen, now time.Time, rs resetSwitch, first *bool) {
	openCodeBudgetSection(c, l, s, now, rs, first)
	openCodeCostSection(c, l, s, first)
}

// openCodeBudgetSection is the Console budget rules. An empty list and a
// switched-off feature both mean "nothing to show", not a failure.
func openCodeBudgetSection(c *ui.Context, l l10n.Localizer, s openCodeScreen, now time.Time, rs resetSwitch, first *bool) {
	snap := s.Snap
	sectionBlock(c, first, func() {
		drawSectionHead(c, l, nil, sectionSpec{title: tx(l, xOpenCodeBudgetTitle)})
		switch {
		case openCodeSignedOut(snap):
			dimText(c, tx(l, xOpenCodeBudgetSignedOut))
		case snap.BudgetsErr != nil:
			dimText(c, tx(l, xOpenCodeBudgetFailed)+" · "+openCodeErrText(l, snap.BudgetsErr))
		case snap.BudgetsOff:
			dimText(c, tx(l, xOpenCodeBudgetDisabled))
		case len(snap.Budgets) == 0:
			dimText(c, tx(l, xOpenCodeBudgetNone))
		default:
			for i, b := range snap.Budgets {
				openCodeBudgetRow(c, l, b, i, now, rs)
			}
		}
	})
}

func openCodeBudgetRow(c *ui.Context, l l10n.Localizer, b opencode.Budget, index int, now time.Time, rs resetSwitch) {
	t := c.Theme()
	limit := tx(l, xOpenCodeNoData)
	if b.HasLimit {
		limit = opencode.FormatUSD(b.LimitUSD)
	}
	ui.Column(c).Gap(4).Children(func() {
		ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
			ui.Text(c, tx(l, xOpenCodeBudgetSpent, opencode.FormatUSD(b.SpentUSD), limit)).
				FontSize(fsCaption).FontWeight(600).FontFeatures("tnum").Grow(1).Basis(0).SingleLine()
			if b.Per != "" {
				ui.Text(c, tx(l, xOpenCodeBudgetRule, openCodeBudgetScope(l, b.Scope), b.Per)).
					FontSize(fsCaption).TextColor(inkSecondary(t)).SingleLine()
			}
		})
		if b.HasLimit {
			usedFractionBar(c, b.Fraction(), openCodeHealth(b.Fraction()*100))
		}
		if b.HasReset {
			ui.Row(c).Children(func() {
				ui.Spacer(c).Grow(1)
				resetLine(c, l, "reset-opencode-budget-"+strconv.Itoa(index), b.ResetsAt, now, rs)
			})
		}
	})
}

// openCodeBudgetScope names the scope; an unknown or empty value is shown as
// the rule's own word, because inventing a label would be worse.
func openCodeBudgetScope(l l10n.Localizer, scope string) string {
	if scope == "" {
		return tx(l, xOpenCodeConsole)
	}
	return scope
}

// openCodeCostSection is the cost the CLI recorded. It is local, so it is
// shown even when the account is signed out.
func openCodeCostSection(c *ui.Context, l l10n.Localizer, s openCodeScreen, first *bool) {
	snap := s.Snap
	sectionBlock(c, first, func() {
		drawSectionHead(c, l, nil, sectionSpec{title: tx(l, xOpenCodeCostTitle)})
		if snap.LocalErr != nil {
			dimText(c, tx(l, xOpenCodeDatabaseFailed))
		}
		if !snap.HasCost || snap.Cost.Sessions == 0 {
			dimText(c, tx(l, xOpenCodeCostEmpty))
			return
		}
		t := c.Theme()
		ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
			ui.Text(c, opencode.FormatUSD(snap.Cost.TotalUSD)).FontSize(fsCallout).FontWeight(600).
				FontFeatures("tnum").Grow(1).Basis(0)
			ui.Text(c, tx(l, xOpenCodeCostSessions, strconv.FormatInt(snap.Cost.Sessions, 10))).
				FontSize(fsCaption).TextColor(inkSecondary(t)).SingleLine()
		})
		dimText(c, tx(l, xOpenCodeCostTokens,
			strconv.FormatInt(snap.Cost.Tokens.Input, 10),
			strconv.FormatInt(snap.Cost.Tokens.Output, 10),
			strconv.FormatInt(snap.Cost.Tokens.Reasoning, 10),
			strconv.FormatInt(snap.Cost.Tokens.CacheRead, 10),
			strconv.FormatInt(snap.Cost.Tokens.CacheWrite, 10)))
		models := snap.Cost.Models
		if len(models) > openCodeModelRows {
			models = models[:openCodeModelRows]
		}
		for _, m := range models {
			ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
				ui.Text(c, openCodeModelName(l, m)).FontSize(fsCaption).FontWeight(600).
					Grow(1).Basis(0).SingleLine().Ellipsis("…")
				ui.Text(c, opencode.FormatUSD(m.CostUSD)).FontSize(fsCaption).
					FontFeatures("tnum").SingleLine()
			})
		}
		dimText(c, tx(l, xOpenCodeCostNote))
		dimText(c, tx(l, xOpenCodeSourceNote))
	})
}

// openCodeModelRows bounds the per-model list of the cost section.
const openCodeModelRows = 6

// openCodeSignedOut is the reading without a usable login: no account, an
// expired one, an unreadable local one, or a login the endpoint rejected.
func openCodeSignedOut(snap opencode.Snapshot) bool {
	return snap.State != opencode.StateReady || snap.Err != nil
}

func openCodeModelName(l l10n.Localizer, m opencode.ModelCost) string {
	switch {
	case m.Provider != "" && m.Model != "":
		return m.Provider + "/" + m.Model
	case m.Model != "":
		return m.Model
	case m.Provider != "":
		return m.Provider
	}
	return tx(l, xOpenCodeCostUntitled)
}
