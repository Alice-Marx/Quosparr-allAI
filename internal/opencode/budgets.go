package opencode

import (
	"encoding/json"
	"math"
	"time"
)

// Budget is one Console budget rule.
type Budget struct {
	ID string
	// LimitUSD and SpentUSD are dollars, not cents.
	LimitUSD float64
	SpentUSD float64
	HasLimit bool
	Scope    string
	Per      string
	ResetsAt time.Time
	HasReset bool
}

type rawBudget struct {
	RuleID       string   `json:"ruleID"`
	LimitDollars *float64 `json:"limitDollars"`
	SpentDollars *float64 `json:"spentDollars"`
	Scope        string   `json:"scope"`
	Per          string   `json:"per"`
	ResetsAt     string   `json:"resetsAt"`
}

// decodeBudgets reads the budget answer. An empty array is the normal answer
// for an organization without budget rules, so it is not an error.
func decodeBudgets(data []byte) ([]Budget, error) {
	var raw []rawBudget
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, ErrDecode
	}
	out := make([]Budget, 0, len(raw))
	for _, item := range raw {
		b := Budget{ID: item.RuleID, Scope: item.Scope, Per: item.Per}
		if item.LimitDollars != nil {
			b.LimitUSD, b.HasLimit = nonNegative(*item.LimitDollars), true
		}
		if item.SpentDollars != nil {
			b.SpentUSD = nonNegative(*item.SpentDollars)
		}
		if at, ok := parseTime(item.ResetsAt); ok {
			b.ResetsAt, b.HasReset = at, true
		}
		out = append(out, b)
	}
	return out, nil
}

// Fraction is the spent share of the limit, clamped to 0-1. Over-budget
// spending is real and reported as a full bar.
func (b Budget) Fraction() float64 {
	if !b.HasLimit || b.LimitUSD <= 0 {
		return 0
	}
	return math.Min(1, math.Max(0, b.SpentUSD/b.LimitUSD))
}

func nonNegative(v float64) float64 {
	if math.IsNaN(v) || math.IsInf(v, 0) || v < 0 {
		return 0
	}
	return v
}
