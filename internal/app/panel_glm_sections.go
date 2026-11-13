package app

import (
	"errors"
	"strconv"

	"github.com/egoist/mygo/ui"
	"github.com/licoy/quosparr/internal/glm"
	"github.com/licoy/quosparr/internal/l10n"
)

// glmCostRowLimit keeps the section inside one screen; the rows are already
// ordered most expensive first.
const glmCostRowLimit = 6

// glmCostSection is the API-equivalent cost of the local ZCode usage.
func glmCostSection(c *ui.Context, l l10n.Localizer, s glmScreen, first *bool) {
	sectionBlock(c, first, func() {
		drawSectionHead(c, l, nil, sectionSpec{title: tx(l, xGLMCostTitle)})
		switch {
		case errors.Is(s.CostErr, glm.ErrNoUsage), s.CostErr == nil && len(s.Cost.Rows) == 0:
			dimText(c, tx(l, xGLMCostEmpty))
			dimText(c, s.DBPath)
			return
		case s.CostErr != nil:
			dimText(c, tx(l, xGLMErrDatabase))
			dimText(c, s.DBPath)
			return
		}
		ui.Text(c, tx(l, xGLMCostTotal, glm.FormatUSD(s.Cost.TotalUSD))).FontSize(fsCallout).FontWeight(600).FontFeatures("tnum")
		if s.Cost.Unpriced > 0 {
			dimText(c, tx(l, xGLMCostUnpriced, strconv.Itoa(s.Cost.Unpriced)))
		}
		for _, row := range glmCostRows(s.Cost.Rows) {
			glmCostLine(c, l, row)
		}
		dimText(c, tx(l, xGLMCostSourceNote))
	})
}

// glmCostLine is one model of the cost section.
func glmCostLine(c *ui.Context, l l10n.Localizer, row glm.Row) {
	amount := "—"
	if row.HasPrice {
		amount = glm.FormatUSD(row.USD)
	}
	ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
		ui.Text(c, row.Model).FontSize(fsCaption).FontWeight(600).Grow(1).Basis(0).SingleLine().Ellipsis("…")
		ui.Text(c, amount).FontSize(fsCaption).FontFeatures("tnum").SingleLine()
	})
	dimText(c, tx(l, xGLMCostTokensLine, strconv.FormatInt(row.Input, 10), strconv.FormatInt(row.Output, 10),
		strconv.FormatInt(row.CacheRead, 10), strconv.FormatInt(row.Requests, 10)))
	if !row.HasPrice {
		dimText(c, tx(l, xGrokBotCostMissingPrice))
	}
}

// glmCostRows trims the priced rows to what the section shows.
func glmCostRows(rows []glm.Row) []glm.Row {
	if len(rows) > glmCostRowLimit {
		return rows[:glmCostRowLimit]
	}
	return rows
}
