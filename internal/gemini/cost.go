package gemini

import (
	"bufio"
	"encoding/json"
	"io"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

// Limits that keep one reading cheap on a directory a long session can fill.
const (
	maxSessionFiles       = 400
	maxSessionBytes int64 = 8 << 20
	maxMessages           = 20000
	maxCostRows           = 12
)

// Usage is the token spend of one model across the local chat logs.
type Usage struct {
	// Model is the model the CLI recorded for the answer, which is the name
	// the upstream service used, not the requested alias.
	Model string
	// Input is promptTokenCount. It already includes Cached and Tool.
	Input int64
	// Cached is cachedContentTokenCount, a subset of Input.
	Cached int64
	// Output is candidatesTokenCount.
	Output int64
	// Thoughts is thoughtsTokenCount; it is priced as output.
	Thoughts int64
	// Tool is toolUsePromptTokenCount, a subset of Input.
	Tool int64
	// Requests counts the answers that carried a token record.
	Requests int64
}

// Ledger is the token spend of every chat log this package could read.
type Ledger struct {
	// Models is the spend per model, largest first.
	Models []Usage
	// Sessions counts the chat logs that were read.
	Sessions int
	// Truncated reports that a bound stopped the scan early.
	Truncated bool
}

// HasAny reports whether any chat log recorded a token count.
func (l Ledger) HasAny() bool { return len(l.Models) > 0 }

// ScanCost reads the chat logs under <home>/.gemini/tmp/<workspace>/chats and
// sums their token records per model. It never fails: an unreadable log is
// skipped and marks the ledger truncated.
func ScanCost(home string) Ledger {
	pattern := filepath.Join(TempDir(home), "*", ChatDirName, SessionFilePrefix+"*.jsonl")
	files, err := filepath.Glob(pattern)
	if err != nil || len(files) == 0 {
		return Ledger{}
	}
	sort.Strings(files)
	var ledger Ledger
	byModel := make(map[string]*Usage)
	for _, path := range files {
		if ledger.Sessions >= maxSessionFiles {
			ledger.Truncated = true
			break
		}
		messages, truncated := readSession(path)
		ledger.Sessions++
		if truncated {
			ledger.Truncated = true
		}
		for _, message := range messages {
			if message.Type != "gemini" || message.Tokens == nil {
				continue
			}
			usage := byModel[message.Model]
			if usage == nil {
				usage = &Usage{Model: message.Model}
				byModel[message.Model] = usage
			}
			usage.Input += message.Tokens.Input
			usage.Cached += message.Tokens.Cached
			usage.Output += message.Tokens.Output
			usage.Thoughts += message.Tokens.Thoughts
			usage.Tool += message.Tokens.Tool
			usage.Requests++
		}
	}
	for _, usage := range byModel {
		ledger.Models = append(ledger.Models, *usage)
	}
	sort.SliceStable(ledger.Models, func(i, j int) bool {
		left, right := ledger.Models[i], ledger.Models[j]
		leftTotal, rightTotal := left.total(), right.total()
		if leftTotal != rightTotal {
			return leftTotal > rightTotal
		}
		return left.Model < right.Model
	})
	return ledger
}

func (u Usage) total() int64 { return u.Input + u.Output + u.Thoughts }

// sessionMessage is one message of a chat log.
type sessionMessage struct {
	ID     string         `json:"id"`
	Type   string         `json:"type"`
	Model  string         `json:"model"`
	Tokens *sessionTokens `json:"tokens"`
}

// sessionTokens is the recorded usage of one answer.
type sessionTokens struct {
	Input    int64 `json:"input"`
	Output   int64 `json:"output"`
	Cached   int64 `json:"cached"`
	Thoughts int64 `json:"thoughts"`
	Tool     int64 `json:"tool"`
	Total    int64 `json:"total"`
}

// readSession folds one JSONL chat log into the messages it ends with. The file
// is a patch stream, not a message list: a line either replaces the whole
// messages array, rewinds to a message, or carries one message.
func readSession(path string) ([]sessionMessage, bool) {
	f, err := os.Open(path)
	if err != nil {
		return nil, true
	}
	defer f.Close()
	reader := bufio.NewReaderSize(io.LimitReader(f, maxSessionBytes+1), 64*1024)
	var (
		current   []sessionMessage
		index     = make(map[string]int)
		read      int64
		truncated bool
	)
	for {
		line, err := reader.ReadString('\n')
		read += int64(len(line))
		if read > maxSessionBytes {
			truncated = true
			break
		}
		if trimmed := strings.TrimSpace(line); trimmed != "" {
			current, index = foldLine([]byte(trimmed), current, index)
			if len(current) > maxMessages {
				current = current[:maxMessages]
				index = messageIndex(current)
				truncated = true
			}
		}
		if err != nil {
			if err != io.EOF {
				truncated = true
			}
			break
		}
	}
	return current, truncated
}

// sessionPatch is the envelope of one line: a $set, a rewind, or a message.
type sessionPatch struct {
	Set      *sessionSet `json:"$set"`
	RewindTo string      `json:"$rewindTo"`
}

type sessionSet struct {
	Messages json.RawMessage `json:"messages"`
}

// foldLine applies one line to the messages folded so far.
func foldLine(line []byte, current []sessionMessage, index map[string]int) ([]sessionMessage, map[string]int) {
	var patch sessionPatch
	if json.Unmarshal(line, &patch) != nil {
		return current, index
	}
	if patch.Set != nil && len(patch.Set.Messages) > 0 {
		var batch []sessionMessage
		if json.Unmarshal(patch.Set.Messages, &batch) != nil {
			return current, index
		}
		return batch, messageIndex(batch)
	}
	if patch.RewindTo != "" {
		if at, ok := index[patch.RewindTo]; ok {
			cut := current[:at+1]
			return cut, messageIndex(cut)
		}
		return current, index
	}
	var message sessionMessage
	if json.Unmarshal(line, &message) != nil {
		return current, index
	}
	if message.Tokens == nil && message.Type == "" {
		// A header line or a scratchpad update: not a message.
		return current, index
	}
	if message.ID != "" {
		if at, ok := index[message.ID]; ok {
			current[at] = message
			return current, index
		}
	}
	current = append(current, message)
	if message.ID != "" {
		index[message.ID] = len(current) - 1
	}
	return current, index
}

func messageIndex(messages []sessionMessage) map[string]int {
	index := make(map[string]int, len(messages))
	for at, message := range messages {
		if message.ID != "" {
			index[message.ID] = at
		}
	}
	return index
}

// Price is a published rate in US dollars per million tokens. Three kinds of
// prompt token are priced apart because the provider bills them apart.
type Price struct {
	Input  float64
	Cached float64
	Output float64
}

// priceTable holds the published rates of the models this package knows. Every
// entry is an external published price, not something the CLI source carries.
var priceTable = map[string]Price{
	"gemini-2.5-pro":        {Input: 1.25, Cached: 0.125, Output: 10.00},
	"gemini-2.5-flash":      {Input: 0.30, Cached: 0.03, Output: 2.50},
	"gemini-2.5-flash-lite": {Input: 0.10, Cached: 0.01, Output: 0.40},
	"gemini-2.0-flash":      {Input: 0.10, Cached: 0.025, Output: 0.40},
}

// familyPrices is the fallback order, most specific first. A model matched here
// has no published price of its own and is reported as an estimate.
var familyPrices = []struct {
	match string
	price Price
}{
	{match: "flash-lite", price: Price{Input: 0.10, Cached: 0.01, Output: 0.40}},
	{match: "flash", price: Price{Input: 0.30, Cached: 0.03, Output: 2.50}},
	{match: "pro", price: Price{Input: 1.25, Cached: 0.125, Output: 10.00}},
}

// priceFor resolves a model. exact is the published rate of that model,
// family a rate borrowed from its family, and neither means unpriced.
func priceFor(model string) (price Price, exact, family bool) {
	if price, ok := priceTable[model]; ok {
		return price, true, false
	}
	lower := strings.ToLower(model)
	if lower == "" {
		return Price{}, false, false
	}
	for _, candidate := range familyPrices {
		if strings.Contains(lower, candidate.match) {
			return candidate.price, false, true
		}
	}
	return Price{}, false, false
}

// CostRow is one priced model of the ledger. Amount is empty when the model has
// no price at all.
type CostRow struct {
	Model string
	// Input is the uncached prompt token count.
	Input int64
	// Cached is the cached prompt token count, priced apart and never added to
	// Input.
	Cached int64
	// Output is the candidate count plus the thinking count.
	Output int64
	// Tool is the tool-prompt count, a subset of Input, shown but not billed
	// twice.
	Tool     int64
	Requests int64
	Amount   string
	// Missing reports a model with no published price and no family rate.
	Missing bool
	// Estimated reports a model priced by its family instead of exactly.
	Estimated bool
}

// CostRows prices the ledger. The bool reports whether there was anything to
// price.
func CostRows(ledger Ledger) ([]CostRow, bool) {
	if len(ledger.Models) == 0 {
		return nil, false
	}
	rows := make([]CostRow, 0, min(len(ledger.Models), maxCostRows))
	for _, usage := range ledger.Models {
		if len(rows) >= maxCostRows {
			break
		}
		rows = append(rows, costRow(usage))
	}
	return rows, true
}

func costRow(usage Usage) CostRow {
	cached := usage.Cached
	if cached < 0 {
		cached = 0
	}
	if cached > usage.Input {
		// The recorded counts nest cached inside input; a log that breaks that
		// nesting is clamped rather than double billed.
		cached = usage.Input
	}
	row := CostRow{
		Model:    usage.Model,
		Input:    usage.Input - cached,
		Cached:   cached,
		Output:   usage.Output + usage.Thoughts,
		Tool:     usage.Tool,
		Requests: usage.Requests,
	}
	price, exact, family := priceFor(usage.Model)
	if !exact && !family {
		row.Missing = true
		return row
	}
	row.Estimated = family
	amount := (float64(row.Input)*price.Input +
		float64(row.Cached)*price.Cached +
		float64(row.Output)*price.Output) / 1_000_000
	row.Amount = FormatUSD(amount)
	return row
}

// FormatUSD renders a dollar amount with four decimals at most and two at
// least.
func FormatUSD(v float64) string {
	if math.IsNaN(v) || math.IsInf(v, 0) || v < 0 {
		v = 0
	}
	text := strconv.FormatFloat(v, 'f', 4, 64)
	if dot := strings.IndexByte(text, '.'); dot >= 0 {
		end := len(text)
		for end > dot+3 && text[end-1] == '0' {
			end--
		}
		text = text[:end]
	}
	return "$" + text
}
