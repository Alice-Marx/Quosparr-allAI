package app

import (
	"strconv"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/licoy/quosparr/internal/kimi"
	"github.com/licoy/quosparr/internal/l10n"
	"github.com/licoy/quosparr/internal/runway"
)

func kimiSections(c *ui.Context, l l10n.Localizer, s kimiScreen, rs resetSwitch, first *bool) {
	if s.ready() {
		kimiQuotaSection(c, l, s, rs, first)
		kimiWalletSection(c, l, s, first)
	}
	kimiCostSection(c, l, s, first)
	kimiNoteSection(c, l, first)
}

// kimiQuotaSection is one bar per quota window. A window the answer omits is
// shown as unknown: it is neither 0% used nor full.
func kimiQuotaSection(c *ui.Context, l l10n.Localizer, s kimiScreen, rs resetSwitch, first *bool) {
	sectionBlock(c, first, func() {
		drawSectionHead(c, l, nil, sectionSpec{title: tx(l, xKimiQuotaTitle)})
		now := time.Now()
		t := c.Theme()
		windows := []struct {
			key    string
			label  textKey
			bucket *kimi.Bucket
		}{
			{"limit-5h", xKimiLimit5h, s.Report.Usages.Limit5h},
			{"limit-7d", xKimiLimit7d, s.Report.Usages.Limit7d},
			{"limit-month-total", xKimiLimitMonthTotal, s.Report.Usages.MonthTotal},
			{"limit-month-code", xKimiLimitMonthCode, s.Report.Usages.MonthCode},
		}
		for _, w := range windows {
			bucket := w.bucket
			ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
				ui.Text(c, tx(l, w.label)).FontSize(fsCaption).FontWeight(600).Grow(1).Basis(0).SingleLine().Ellipsis("…")
				if bucket == nil {
					ui.Text(c, tx(l, xKimiUnknown)).FontSize(fsCaption).TextColor(inkSecondary(t)).SingleLine()
					return
				}
				ui.Text(c, tx(l, xKimiUsed, kimi.FormatPercent(bucket.UsedPercent))).FontSize(fsCaption).FontWeight(600).FontFeatures("tnum").SingleLine()
			})
			if bucket == nil {
				continue
			}
			usedFractionBar(c, bucket.UsedPercent/100, kimiHealth(bucket.UsedPercent))
			if bucket.HasReset {
				ui.Row(c).Children(func() {
					ui.Spacer(c).Grow(1)
					resetLine(c, l, "reset-kimi-"+w.key, bucket.ResetAt, now, rs)
				})
			} else {
				dimText(c, tx(l, xKimiResetMissing))
			}
		}
		dimText(c, tx(l, xKimiQuotaNote))
	})
}

func kimiWalletSection(c *ui.Context, l l10n.Localizer, s kimiScreen, first *bool) {
	sectionBlock(c, first, func() {
		drawSectionHead(c, l, nil, sectionSpec{title: tx(l, xKimiWalletTitle)})
		wallet := s.Report.Wallet
		if wallet == nil {
			dimText(c, tx(l, xKimiWalletMissing))
			return
		}
		ui.Text(c, tx(l, xKimiWalletBalance,
			kimi.FormatMoney(wallet.LeftCents, wallet.Currency),
			kimi.FormatMoney(wallet.TotalCents, wallet.Currency))).FontSize(fsCallout).FontWeight(600).FontFeatures("tnum")
		if wallet.LimitEnabled {
			dimText(c, tx(l, xKimiWalletLimit,
				kimi.FormatMoney(wallet.LimitCents, wallet.Currency),
				kimi.FormatMoney(wallet.UsedCents, wallet.Currency)))
			return
		}
		dimText(c, tx(l, xKimiWalletLimitOff))
	})
}

// kimiCostSection prices the token counts of the CLI session logs. The logs
// carry no money, so a model without a known price is marked, not guessed at.
func kimiCostSection(c *ui.Context, l l10n.Localizer, s kimiScreen, first *bool) {
	sectionBlock(c, first, func() {
		drawSectionHead(c, l, nil, sectionSpec{title: tx(l, xKimiCostTitle)})
		if !s.Costs.Present() {
			dimText(c, tx(l, xKimiCostEmpty, s.Costs.Dir))
			return
		}
		for _, row := range kimi.CostRows(s.Costs, s.Prices) {
			name := row.Model
			amount := "—"
			if row.Amount != "" {
				amount = row.Amount
			}
			ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
				ui.Text(c, name).FontSize(fsCaption).FontWeight(600).Grow(1).Basis(0).SingleLine().Ellipsis("…")
				ui.Text(c, amount).FontSize(fsCaption).FontFeatures("tnum").SingleLine()
			})
			dimText(c, tx(l, xKimiCostTokens,
				strconv.FormatInt(row.InputOther, 10),
				strconv.FormatInt(row.CacheRead, 10),
				strconv.FormatInt(row.CacheWrite, 10),
				strconv.FormatInt(row.Output, 10)))
			if row.Missing {
				dimText(c, tx(l, xKimiCostMissingPrice))
			}
		}
		dimText(c, tx(l, xKimiCostScanned, strconv.Itoa(s.Costs.Sessions), strconv.Itoa(s.Costs.Records)))
		dimText(c, tx(l, xKimiCostNote))
	})
}

// kimiNoteSection states what the panel may touch: it reads the vendor endpoint
// and rewrites the credential file only when a refresh rotates the token.
func kimiNoteSection(c *ui.Context, l l10n.Localizer, first *bool) {
	sectionBlock(c, first, func() {
		drawSectionHead(c, l, nil, sectionSpec{title: tx(l, xKimiNotesTitle)})
		dimText(c, tx(l, xKimiNotesCredential))
		dimText(c, tx(l, xKimiLiveNote))
	})
}

func kimiHealth(used float64) runway.Health {
	rounded := int(used + 0.5)
	if rounded < 0 {
		rounded = 0
	}
	if rounded > 100 {
		rounded = 100
	}
	return runway.HealthForUsedPercent(rounded)
}
