package glm

import (
	"strconv"
	"strings"
)

// Price is the USD price of one million tokens.
type Price struct {
	Input     float64
	Output    float64
	CacheRead float64
}

// prices are the models.dev USD prices per million tokens of the models the
// ZCode CLI bills. Cache writes are free on this vendor and are not part of
// the table.
var prices = map[string]Price{
	"glm-5.3":       {Input: 1.4, Output: 4.4, CacheRead: 0.26},
	"glm-5.3-flash": {Input: 0.15, Output: 0.5, CacheRead: 0.03},
	"glm-5.2":       {Input: 1.4, Output: 4.4, CacheRead: 0.26},
	"glm-5-turbo":   {Input: 1.2, Output: 4.0, CacheRead: 0.24},
	"glm-4.7":       {Input: 0.6, Output: 2.2, CacheRead: 0.11},
	"glm-4.6v":      {Input: 0.3, Output: 0.9, CacheRead: 0.05},
}

// Lookup returns the rate of a model id. The upstream table prices the
// *-highspeed variants at zero, which is a data gap rather than a free
// model, so they are reported as unpriced.
func Lookup(model string) (Price, bool) {
	rate, ok := prices[normalizeModel(model)]
	return rate, ok
}

// normalizeModel lower-cases a model id and drops a vendor prefix.
func normalizeModel(model string) string {
	name := strings.ToLower(strings.TrimSpace(model))
	if _, rest, ok := strings.Cut(name, "/"); ok {
		name = rest
	}
	return name
}

// Cost prices one usage row. Input excludes cache reads, which are priced
// separately: pricing a cache read as input overstates the bill.
func (p Price) Cost(input, output, cacheRead int64) float64 {
	const million = 1e6
	return float64(input)/million*p.Input +
		float64(output)/million*p.Output +
		float64(cacheRead)/million*p.CacheRead
}

// FormatUSD is the panel form of a cost.
func FormatUSD(v float64) string { return "$" + formatFixed(v, 2) }

// formatFixed formats v with at most decimals fraction digits.
func formatFixed(v float64, decimals int) string {
	return strconv.FormatFloat(v, 'f', decimals, 64)
}
