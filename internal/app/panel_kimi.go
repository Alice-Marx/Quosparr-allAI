package app

import (
	"errors"
	"strconv"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/licoy/quosparr/internal/kimi"
	"github.com/licoy/quosparr/internal/l10n"
	"github.com/licoy/quosparr/internal/store"
)

// panelKimi is the Kimi Code quota page. Unlike the snapshot tabs it reads a
// live vendor endpoint, so the refresh button and the wait are part of it.
func panelKimi(c *ui.Context, a *App, l l10n.Localizer, prefs store.Preferences, state *panelState) {
	s := a.kimiScreen()
	rs := state.resetSwitch(prefs.QuotaResetTimeFormat)
	first := true
	sectionBlock(c, &first, func() {
		drawSectionHead(c, l, state, sectionSpec{
			title:      tx(l, xKimiTitle),
			caption:    kimiRegionLabel(l, s.Report.Region),
			refreshing: s.Loading,
			refresh:    func() { a.refreshKimi() },
		})
		kimiIdentity(c, l, s)
		switch {
		case !s.Loaded && s.Loading:
			ui.Row(c).Gap(6).AlignItems(ui.Center).Children(func() {
				ui.Spinner(c).Size(14, 14).Label(tx(l, xKimiLoading))
				calloutDim(c, tx(l, xKimiLoading))
			})
		case !s.Loaded:
			calloutDim(c, l.Text(l10n.KeyNotLoaded))
		case s.Report.State == kimi.StateReady:
			dimText(c, tx(l, xKimiFetched, s.Report.FetchedAt.Local().Format("2006-01-02 15:04")))
		case s.Report.State == kimi.StateNoCredential:
			kimiMessage(c, l, tx(l, xKimiNoCredential), tx(l, xKimiNoCredentialDetail))
		case s.Report.State == kimi.StateExpired:
			kimiMessage(c, l, tx(l, xKimiUnauthorized), tx(l, xKimiUnauthorizedDetail))
		default:
			kimiMessage(c, l, tx(l, xKimiFetchFailed), kimiErrText(l, s.Report.Err))
		}
	})
	kimiSections(c, l, s, rs, &first)
}

// kimiIdentity is the identity line: the product name, the region the endpoint
// was chosen for, the endpoint itself and the credential expiry.
func kimiIdentity(c *ui.Context, l l10n.Localizer, s kimiScreen) {
	t := c.Theme()
	ui.Row(c).Padding(3, 6).Gap(6).Margin(8, -6, 0, -6).AlignItems(ui.Center).Children(func() {
		ui.Icon(c, icTerminal).Size(14, 14).TextColor(inkSecondary(t))
		ui.Text(c, tx(l, xKimiTab)).FontSize(fsCallout).FontWeight(600).SingleLine().Ellipsis("…").MaxWidth(240)
		ui.Text(c, kimiRegionLabel(l, s.Report.Region)).FontSize(fsCaption).TextColor(inkSecondary(t)).SingleLine()
	})
	if s.Report.Base != "" {
		dimText(c, tx(l, xKimiEndpoint, s.Report.Base))
	}
	switch {
	case s.Report.State == kimi.StateNoCredential:
	case s.Report.HasExpiry && !s.Report.ExpiresAt.After(time.Now()):
		ui.Text(c, tx(l, xKimiCredentialExpired)).FontSize(fsCaption).TextColor(c.Theme().Danger).NoWrap()
	case s.Report.HasExpiry:
		dimText(c, tx(l, xKimiCredentialValid, s.Report.ExpiresAt.Local().Format("2006-01-02 15:04")))
	default:
		dimText(c, tx(l, xKimiCredentialNoExpiry))
	}
}

// kimiMessage is the card of a state that has no reading to show.
func kimiMessage(c *ui.Context, l l10n.Localizer, title, detail string) {
	t := c.Theme()
	cardColumn(c, surfaceSunkenKind, radiusCard).Gap(7).Padding(12).Children(func() {
		ui.Row(c).Gap(6).AlignItems(ui.Center).Children(func() {
			ui.Icon(c, icTerminal).Size(13, 13).TextColor(inkSecondary(t))
			ui.Text(c, title).FontSize(fsCallout).FontWeight(600)
		})
		dimText(c, detail)
		dimText(c, tx(l, xKimiLiveNote))
	})
}

// kimiErrText explains a failed reading without ever printing a token or a raw
// vendor body.
func kimiErrText(l l10n.Localizer, err error) string {
	var status *kimi.StatusError
	switch {
	case err == nil:
		return tx(l, xKimiErrUnknown)
	case errors.As(err, &status):
		return tx(l, xKimiErrStatus, strconv.Itoa(status.Code))
	case errors.Is(err, kimi.ErrUnauthorized):
		return tx(l, xKimiUnauthorizedDetail)
	case errors.Is(err, kimi.ErrNetwork):
		return tx(l, xKimiErrNetwork)
	case errors.Is(err, kimi.ErrResponse):
		return tx(l, xKimiErrResponse)
	case errors.Is(err, kimi.ErrNoCredential), errors.Is(err, kimi.ErrCredential):
		return tx(l, xKimiErrCredential)
	default:
		return tx(l, xKimiErrUnknown)
	}
}

// kimiRegionLabel is the region marker of the CLI in words.
func kimiRegionLabel(l l10n.Localizer, region string) string {
	switch region {
	case kimi.RegionMainland:
		return tx(l, xKimiRegionMainland)
	case kimi.RegionGlobal:
		return tx(l, xKimiRegionGlobal)
	case "":
		return tx(l, xKimiRegionUnknown)
	default:
		return tx(l, xKimiRegion, region)
	}
}
