package opencode

import (
	"encoding/json"
	"math"
	"strconv"
	"time"
)

// Window is one quota window of the Zen/Go usage answer.
type Window struct {
	// Percent is the used share, clamped to 0-100. It is only meaningful
	// when HasPercent is set.
	Percent    float64
	HasPercent bool
	// Limited is the "rate-limited" status: the window is exhausted now.
	Limited  bool
	ResetsAt time.Time
	HasReset bool
}

// Usage is the Zen/OpenCode Go usage answer: three rolling windows.
type Usage struct {
	Rolling Window
	Weekly  Window
	Monthly Window
}

type rawWindow struct {
	Status   string   `json:"status"`
	Percent  *float64 `json:"percent"`
	ResetsAt string   `json:"resetsAt"`
}

type rawUsage struct {
	Usage *struct {
		Rolling *rawWindow `json:"rolling"`
		Weekly  *rawWindow `json:"weekly"`
		Monthly *rawWindow `json:"monthly"`
	} `json:"usage"`
}

// decodeUsage reads the usage answer. An answer without a usage object is
// ErrDecode; a missing window is left without a percentage instead of being
// invented as 0.
func decodeUsage(data []byte) (Usage, error) {
	var raw rawUsage
	if err := json.Unmarshal(data, &raw); err != nil || raw.Usage == nil {
		return Usage{}, ErrDecode
	}
	return Usage{
		Rolling: decodeWindow(raw.Usage.Rolling),
		Weekly:  decodeWindow(raw.Usage.Weekly),
		Monthly: decodeWindow(raw.Usage.Monthly),
	}, nil
}

func decodeWindow(raw *rawWindow) Window {
	if raw == nil {
		return Window{}
	}
	w := Window{Limited: raw.Status == "rate-limited"}
	if raw.Percent != nil && !math.IsNaN(*raw.Percent) && !math.IsInf(*raw.Percent, 0) {
		w.Percent, w.HasPercent = clampPercent(*raw.Percent), true
	}
	if at, ok := parseTime(raw.ResetsAt); ok {
		w.ResetsAt, w.HasReset = at, true
	}
	return w
}

// Any reports whether at least one window carried a percentage.
func (u Usage) Any() bool {
	return u.Rolling.HasPercent || u.Weekly.HasPercent || u.Monthly.HasPercent
}

// clampPercent bounds a used share to 0-100.
func clampPercent(v float64) float64 {
	return math.Min(100, math.Max(0, v))
}

// parseTime reads an ISO 8601 timestamp with optional fractional seconds.
func parseTime(s string) (time.Time, bool) {
	if s == "" {
		return time.Time{}, false
	}
	at, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return time.Time{}, false
	}
	return at, true
}

// FormatPercent renders a 0-100 value with at most one decimal place.
func FormatPercent(v float64) string {
	rounded := math.Round(v*10) / 10
	if rounded == math.Trunc(rounded) {
		return strconv.FormatFloat(rounded, 'f', 0, 64)
	}
	return strconv.FormatFloat(rounded, 'f', 1, 64)
}

// FormatUSD renders a non-negative dollar amount with a leading $ and up to
// two decimals.
func FormatUSD(v float64) string {
	if math.IsNaN(v) || math.IsInf(v, 0) || v < 0 {
		v = 0
	}
	rounded := math.Round(v*100) / 100
	s := strconv.FormatFloat(rounded, 'f', 2, 64)
	for len(s) > 0 && s[len(s)-1] == '0' {
		s = s[:len(s)-1]
	}
	s = trimDot(s)
	return "$" + s
}

func trimDot(s string) string {
	if s != "" && s[len(s)-1] == '.' {
		return s[:len(s)-1]
	}
	if s == "" {
		return "0"
	}
	return s
}
