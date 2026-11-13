package kimi

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/licoy/quosparr/internal/claude"
	"github.com/licoy/quosparr/internal/grok"
	"github.com/licoy/quosparr/internal/usage"
	"github.com/licoy/quosparr/internal/usage/decimal"
)

const (
	// maxSessions bounds how many session logs one scan opens.
	maxSessions = 256
	// maxScanBytes is the shared read budget of one scan.
	maxScanBytes int64 = 32 << 20
	// maxWireLine skips a log line longer than any usage event.
	maxWireLine = 1 << 20
	// usageRecordType is the only event type that carries token counts.
	usageRecordType = "usage.record"
)

var recordToken = []byte("\"" + usageRecordType + "\"")

// ModelUsage is the summed token usage of one model in the session logs.
type ModelUsage struct {
	Model      string
	InputOther int64
	CacheRead  int64
	CacheWrite int64
	Output     int64
	Records    int
	LastAt     time.Time
}

// Total is every token of the row, the cached part included.
func (m ModelUsage) Total() int64 {
	return m.InputOther + m.CacheRead + m.CacheWrite + m.Output
}

// Costs is the local token usage of the CLI. It comes from the session logs
// only: the CLI reports tokens and never money.
type Costs struct {
	// Dir is the sessions directory the scan looked at.
	Dir      string
	Models   []ModelUsage
	Sessions int
	Records  int
	// Truncated reports that the scan stopped at its session or byte budget.
	Truncated bool
}

// Present reports whether any usage.record event was found.
func (c Costs) Present() bool { return c.Records > 0 }

// ScanCosts sums the usage.record events of the CLI session logs under
// <home>/sessions/wd_*/session_*/agents/main/wire.jsonl. It never returns an
// error: a missing or unreadable log is simply not counted.
func ScanCosts(home string) Costs {
	dir := SessionsDir(home)
	out := Costs{Dir: dir}
	if dir == "" {
		return out
	}
	paths, err := filepath.Glob(filepath.Join(dir, "wd_*", "session_*", "agents", "main", "wire.jsonl"))
	if err != nil || len(paths) == 0 {
		return out
	}
	sort.Strings(paths)
	if len(paths) > maxSessions {
		paths, out.Truncated = paths[:maxSessions], true
	}
	budget := maxScanBytes
	byModel := map[string]*ModelUsage{}
	for _, path := range paths {
		if budget <= 0 {
			out.Truncated = true
			break
		}
		records, ok := scanWire(path, &budget, byModel)
		if !ok {
			continue
		}
		out.Sessions++
		out.Records += records
	}
	out.Models = make([]ModelUsage, 0, len(byModel))
	for _, m := range byModel {
		out.Models = append(out.Models, *m)
	}
	sort.SliceStable(out.Models, func(i, j int) bool {
		if a, b := out.Models[i].Total(), out.Models[j].Total(); a != b {
			return a > b
		}
		return out.Models[i].Model < out.Models[j].Model
	})
	return out
}

// scanWire reads one JSONL log. The read budget is shared by the whole scan.
func scanWire(path string, budget *int64, byModel map[string]*ModelUsage) (int, bool) {
	f, err := os.Open(path)
	if err != nil {
		return 0, false
	}
	defer f.Close()
	if info, err := f.Stat(); err != nil || info.IsDir() {
		return 0, false
	}
	reader := bufio.NewReaderSize(f, 64<<10)
	records := 0
	for {
		line, err := reader.ReadBytes('\n')
		if len(line) > 0 {
			*budget -= int64(len(line))
			if len(line) <= maxWireLine && bytes.Contains(line, recordToken) {
				if event, ok := parseRecord(line); ok {
					mergeUsage(byModel, event)
					records++
				}
			}
		}
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return records, true
		}
		if *budget <= 0 {
			break
		}
	}
	return records, true
}

// wireEvent is the subset of a session log line this package reads. Unknown
// event types and unknown fields are ignored, so a newer CLI still works.
type wireEvent struct {
	Type  string `json:"type"`
	Model string `json:"model"`
	Usage struct {
		InputOther         int64 `json:"inputOther"`
		InputCacheRead     int64 `json:"inputCacheRead"`
		InputCacheCreation int64 `json:"inputCacheCreation"`
		Output             int64 `json:"output"`
	} `json:"usage"`
	Time int64 `json:"time"`
}

func parseRecord(line []byte) (wireEvent, bool) {
	var event wireEvent
	if err := json.Unmarshal(line, &event); err != nil {
		return event, false
	}
	if event.Type != usageRecordType {
		return event, false
	}
	return event, true
}

// mergeUsage adds one record. A negative counter is clamped to zero rather than
// pulling a total down.
func mergeUsage(byModel map[string]*ModelUsage, event wireEvent) {
	model := strings.TrimSpace(event.Model)
	if model == "" {
		model = "unknown"
	}
	m, ok := byModel[model]
	if !ok {
		m = &ModelUsage{Model: model}
		byModel[model] = m
	}
	m.InputOther += nonNegative(event.Usage.InputOther)
	m.CacheRead += nonNegative(event.Usage.InputCacheRead)
	m.CacheWrite += nonNegative(event.Usage.InputCacheCreation)
	m.Output += nonNegative(event.Usage.Output)
	m.Records++
	if event.Time > 0 {
		if at := time.UnixMilli(event.Time); at.After(m.LastAt) {
			m.LastAt = at
		}
	}
}

func nonNegative(v int64) int64 {
	if v < 0 {
		return 0
	}
	return v
}

// Price is the USD price of one million tokens of a model.
type Price struct {
	Input      decimal.Decimal
	CacheRead  decimal.Decimal
	CacheWrite decimal.Decimal
	Output     decimal.Decimal
}

// PriceTable maps a model name, the part of it after the last "/", or the part
// before the first "/" to a price.
type PriceTable map[string]Price

// DefaultPrices carries no entry on purpose. Kimi Code bills a subscription
// window, the CLI ships no price list, and no per-token price is known for its
// model names, so a row is reported as unpriced instead of being priced by
// guesswork. A caller that verifies a price adds it here.
func DefaultPrices() PriceTable { return PriceTable{} }

func (t PriceTable) lookup(model string) (Price, bool) {
	for _, key := range priceKeys(model) {
		if p, ok := t[key]; ok {
			return p, true
		}
	}
	return Price{}, false
}

func priceKeys(model string) []string {
	model = strings.TrimSpace(model)
	if model == "" {
		return nil
	}
	keys := []string{model}
	if i := strings.LastIndex(model, "/"); i >= 0 && i+1 < len(model) {
		keys = append(keys, model[i+1:])
	}
	if i := strings.Index(model, "/"); i > 0 {
		keys = append(keys, model[:i])
	}
	return keys
}

// cost prices the four counters. A cache write falls back to the input rate
// when the table has no rate of its own for it.
func (p Price) cost(input, cacheRead, cacheWrite, output int64) decimal.Decimal {
	million := decimal.FromInt(1_000_000)
	part := func(tokens int64, rate decimal.Decimal) decimal.Decimal {
		return decimal.FromInt64(tokens).Div(million).Mul(rate)
	}
	writeRate := p.CacheWrite
	if writeRate.IsZero() {
		writeRate = p.Input
	}
	total := part(input, p.Input)
	total = total.Add(part(cacheRead, p.CacheRead))
	total = total.Add(part(cacheWrite, writeRate))
	return total.Add(part(output, p.Output))
}

// CostRow is one model of the cost section. Amount is empty when the model has
// no known price.
type CostRow struct {
	Model      string
	InputOther int64
	CacheRead  int64
	CacheWrite int64
	Output     int64
	Amount     string
	Missing    bool
}

// CostRows prices the scanned models. A model whose price is unknown keeps its
// token counts and is marked Missing.
func CostRows(costs Costs, table PriceTable) []CostRow {
	rows := make([]CostRow, 0, len(costs.Models))
	for _, m := range costs.Models {
		row := CostRow{
			Model: m.Model, InputOther: m.InputOther, CacheRead: m.CacheRead,
			CacheWrite: m.CacheWrite, Output: m.Output,
		}
		if amount, ok := LookupPrice(table, m); ok {
			row.Amount = amount
		} else {
			row.Missing = true
		}
		rows = append(rows, row)
	}
	return rows
}

// LookupPrice prices one model: the supplied table first, then the price tables
// this repository already ships, so a model those know is still priced. It
// returns the amount formatted in USD.
func LookupPrice(table PriceTable, m ModelUsage) (string, bool) {
	if p, ok := table.lookup(m.Model); ok {
		return formatDecimal(p.cost(m.InputOther, m.CacheRead, m.CacheWrite, m.Output)), true
	}
	input := m.InputOther + m.CacheWrite
	if amount, ok := usage.CostForUsage(m.Model, usage.TokenUsage{
		InputTokens: input + m.CacheRead, CachedInputTokens: m.CacheRead, OutputTokens: m.Output,
	}); ok {
		return formatDecimal(amount), true
	}
	if amount, ok := claude.PricingCost(m.Model, claude.TokenParts{
		Input: input, CacheRead: m.CacheRead, Output: m.Output,
	}, ""); ok {
		return formatDecimal(amount), true
	}
	if input <= math.MaxInt && m.CacheRead <= math.MaxInt && m.Output <= math.MaxInt {
		if amount, ok := grok.PricingCost(m.Model, int(input), int(m.CacheRead), int(m.Output)); ok {
			return formatDecimal(amount), true
		}
	}
	return "", false
}

func formatDecimal(d decimal.Decimal) string {
	return "$" + d.String()
}
