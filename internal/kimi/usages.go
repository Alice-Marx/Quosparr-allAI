package kimi

import (
	"context"
	"encoding/json"
	"io"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// Bucket is one quota window of the usage answer.
type Bucket struct {
	// UsedPercent is the normalized 0-100 share of the window, clamped.
	UsedPercent float64
	// RawRatio is used_ratio exactly as the endpoint sent it, so a reader can
	// tell which unit the answer used.
	RawRatio float64
	// HasReset is false when the window carried no parseable reset_time.
	HasReset bool
	ResetAt  time.Time
}

// Usages are the four quota windows of the answer. Every window is optional: a
// missing one is neither 0% used nor full.
type Usages struct {
	Limit5h    *Bucket
	Limit7d    *Bucket
	MonthTotal *Bucket
	MonthCode  *Bucket
}

// Any reports whether the answer carried at least one window.
func (u Usages) Any() bool {
	return u.Limit5h != nil || u.Limit7d != nil || u.MonthTotal != nil || u.MonthCode != nil
}

// Wallet is the booster wallet of the answer. Its two amount sources use
// different units: TotalCents and LeftCents came from fixed-point micro-units
// (divided by 1e6), while LimitCents and UsedCents were already cents.
type Wallet struct {
	TotalCents   int64
	LeftCents    int64
	LimitEnabled bool
	LimitCents   int64
	UsedCents    int64
	Currency     string
}

// Snapshot is one usage answer.
type Snapshot struct {
	Base      string
	Usages    Usages
	Wallet    *Wallet
	FetchedAt time.Time
}

// UsageURL is {base}/usages with the trailing slash of base removed.
func UsageURL(base string) string {
	return trimBase(base) + "/usages"
}

// FetchUsages asks the endpoint for the quota. A 401 or 403 is ErrUnauthorized,
// which means the user has to sign in again; every other failure is
// ErrNetwork, a StatusError or ErrResponse.
func FetchUsages(ctx context.Context, client Doer, base, token string, now time.Time) (Snapshot, error) {
	if client == nil {
		return Snapshot{}, ErrNoClient
	}
	if strings.TrimSpace(token) == "" {
		return Snapshot{}, ErrUnauthorized
	}
	ctx, cancel := context.WithTimeout(ctx, RequestTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, UsageURL(base), nil)
	if err != nil {
		return Snapshot{}, ErrResponse
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return Snapshot{}, ErrNetwork
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxBodyBytes))
	if err != nil {
		return Snapshot{}, ErrNetwork
	}
	switch {
	case resp.StatusCode == http.StatusUnauthorized, resp.StatusCode == http.StatusForbidden:
		return Snapshot{}, ErrUnauthorized
	case resp.StatusCode != http.StatusOK:
		return Snapshot{}, &StatusError{Code: resp.StatusCode}
	}
	return ParseUsages(body, base, now)
}

// ParseUsages decodes a usage answer. The body has no envelope: the windows are
// under "usages" and the wallet under "boosterWallet".
func ParseUsages(body []byte, base string, now time.Time) (Snapshot, error) {
	if now.IsZero() {
		now = time.Now()
	}
	var raw rawUsage
	if err := json.Unmarshal(body, &raw); err != nil {
		return Snapshot{}, ErrResponse
	}
	if raw.Usages == nil {
		return Snapshot{}, ErrResponse
	}
	return Snapshot{
		Base:      trimBase(base),
		Usages:    raw.Usages.parse(),
		Wallet:    raw.BoosterWallet.parse(),
		FetchedAt: now,
	}, nil
}

type rawUsage struct {
	Usages        *rawBuckets `json:"usages"`
	BoosterWallet *rawWallet  `json:"boosterWallet"`
}

// rawBuckets keeps the wire key names: the four windows are snake_case and all
// optional.
type rawBuckets struct {
	Limit5h    *rawEntry `json:"limit_5h"`
	Limit7d    *rawEntry `json:"limit_7d"`
	MonthTotal *rawEntry `json:"limit_month_total"`
	MonthCode  *rawEntry `json:"limit_month_code"`
}

type rawEntry struct {
	UsedRatio json.Number `json:"used_ratio"`
	ResetTime *string     `json:"reset_time"`
}

func (r *rawBuckets) parse() Usages {
	return Usages{
		Limit5h:    r.Limit5h.parse(),
		Limit7d:    r.Limit7d.parse(),
		MonthTotal: r.MonthTotal.parse(),
		MonthCode:  r.MonthCode.parse(),
	}
}

// parse drops an entry whose ratio is missing or not a finite number, as the
// CLI does, rather than reporting 0%.
func (e *rawEntry) parse() *Bucket {
	if e == nil {
		return nil
	}
	ratio, err := e.UsedRatio.Float64()
	if err != nil || math.IsNaN(ratio) || math.IsInf(ratio, 0) {
		return nil
	}
	b := &Bucket{RawRatio: ratio, UsedPercent: normalizeRatio(ratio)}
	if e.ResetTime != nil {
		if at, ok := parseReset(*e.ResetTime); ok {
			b.HasReset, b.ResetAt = true, at
		}
	}
	return b
}

// normalizeRatio maps used_ratio onto 0-100. The unit is undocumented: the CLI
// forwards the value without conversion, so a value above 1 is read as a
// percentage and anything else as a fraction. Both are clamped, so a broken
// answer can never paint a bar outside its track.
func normalizeRatio(ratio float64) float64 {
	percent := ratio * 100
	if ratio > 1 {
		percent = ratio
	}
	switch {
	case percent < 0:
		return 0
	case percent > 100:
		return 100
	}
	return percent
}

// parseReset reads the ISO8601 reset_time. An unparseable value leaves the
// window without a reset time instead of inventing one.
func parseReset(raw string) (time.Time, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return time.Time{}, false
	}
	for _, layout := range []string{time.RFC3339Nano, time.RFC3339, "2006-01-02T15:04:05Z0700"} {
		if at, err := time.Parse(layout, raw); err == nil {
			return at, true
		}
	}
	return time.Time{}, false
}

type rawWallet struct {
	Balance struct {
		Type       string      `json:"type"`
		Amount     json.Number `json:"amount"`
		AmountLeft json.Number `json:"amountLeft"`
	} `json:"balance"`
	MonthlyChargeLimitEnabled bool      `json:"monthlyChargeLimitEnabled"`
	MonthlyChargeLimit        *rawMoney `json:"monthlyChargeLimit"`
	MonthlyUsed               *rawMoney `json:"monthlyUsed"`
}

type rawMoney struct {
	PriceInCents json.Number `json:"priceInCents"`
	Currency     string      `json:"currency"`
}

// parse keeps the wallet only for a booster balance that holds something. The
// monthly limit fields are read when the endpoint enables them; a balance is
// never dropped just because the limit is off.
func (w *rawWallet) parse() *Wallet {
	if w == nil || !strings.EqualFold(strings.TrimSpace(w.Balance.Type), "BOOSTER") {
		return nil
	}
	total, err := w.Balance.Amount.Float64()
	if err != nil || total <= 0 {
		return nil
	}
	left, _ := w.Balance.AmountLeft.Float64()
	out := &Wallet{
		TotalCents:   fixedPointToCents(total),
		LeftCents:    fixedPointToCents(left),
		LimitEnabled: w.MonthlyChargeLimitEnabled,
		Currency:     walletCurrency(w),
	}
	if w.MonthlyChargeLimit != nil {
		out.LimitCents = centsValue(w.MonthlyChargeLimit.PriceInCents)
	}
	if w.MonthlyUsed != nil {
		out.UsedCents = centsValue(w.MonthlyUsed.PriceInCents)
	}
	return out
}

// walletCurrency is monthlyChargeLimit.currency, then monthlyUsed.currency, then
// USD: the answer has no top-level currency.
func walletCurrency(w *rawWallet) string {
	if w.MonthlyChargeLimit != nil {
		if c := strings.TrimSpace(w.MonthlyChargeLimit.Currency); c != "" {
			return strings.ToUpper(c)
		}
	}
	if w.MonthlyUsed != nil {
		if c := strings.TrimSpace(w.MonthlyUsed.Currency); c != "" {
			return strings.ToUpper(c)
		}
	}
	return "USD"
}

// fixedPointToCents divides the fixed-point micro-unit amounts of the balance by
// 1e6. A positive fraction of a cent rounds up to one cent, as the CLI does.
func fixedPointToCents(micro float64) int64 {
	if math.IsNaN(micro) || math.IsInf(micro, 0) {
		return 0
	}
	cents := micro / 1e6
	if cents > 0 && cents < 1 {
		return 1
	}
	return int64(math.Round(cents))
}

// centsValue reads a priceInCents field, which is already in cents.
func centsValue(n json.Number) int64 {
	v, err := n.Float64()
	if err != nil || math.IsNaN(v) || v < 0 {
		return 0
	}
	return int64(math.Round(v))
}

// FormatPercent renders a 0-100 value with at most one decimal place.
func FormatPercent(v float64) string {
	if math.IsNaN(v) || math.IsInf(v, 0) {
		v = 0
	}
	rounded := math.Round(v*10) / 10
	if rounded == math.Trunc(rounded) {
		return strconv.FormatFloat(rounded, 'f', 0, 64)
	}
	return strconv.FormatFloat(rounded, 'f', 1, 64)
}

// FormatMoney renders an amount in cents with its currency code, because the
// panel has no currency table to pick a symbol from.
func FormatMoney(cents int64, currency string) string {
	if strings.TrimSpace(currency) == "" {
		currency = "USD"
	}
	return strconv.FormatFloat(float64(cents)/100, 'f', 2, 64) + " " + currency
}
