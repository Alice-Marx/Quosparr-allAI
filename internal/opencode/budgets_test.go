package opencode

import (
	"errors"
	"testing"
	"time"
)

func TestDecodeBudgetsReadsDollarFields(t *testing.T) {
	body := `[{"ruleID":"r1","limitDollars":50,"spentDollars":12.34,"scope":"org","per":"month",
 "coverage":"all","picked":true,"shared":false,"resetsAt":"2026-12-01T00:00:00.000Z"}]`
	budgets, err := decodeBudgets([]byte(body))
	if err != nil {
		t.Fatal(err)
	}
	if len(budgets) != 1 {
		t.Fatalf("budgets = %+v", budgets)
	}
	b := budgets[0]
	if b.ID != "r1" || b.LimitUSD != 50 || b.SpentUSD != 12.34 || b.Scope != "org" || b.Per != "month" {
		t.Fatalf("budget = %+v", b)
	}
	if !b.HasReset || !b.ResetsAt.Equal(time.Date(2026, 12, 1, 0, 0, 0, 0, time.UTC)) {
		t.Fatalf("reset = %v", b.ResetsAt)
	}
	if got := b.Fraction(); got < 0.24 || got > 0.25 {
		t.Errorf("fraction = %v", got)
	}
}

func TestDecodeBudgetsAcceptsTheEmptyList(t *testing.T) {
	budgets, err := decodeBudgets([]byte("[]"))
	if err != nil {
		t.Fatal(err)
	}
	if len(budgets) != 0 {
		t.Fatalf("budgets = %+v", budgets)
	}
	if _, err := decodeBudgets([]byte(`{"_tag":"Unauthorized"}`)); !errors.Is(err, ErrDecode) {
		t.Errorf("an error body is not a budget list: %v", err)
	}
}

func TestBudgetFractionClampsToAFullBar(t *testing.T) {
	over := Budget{LimitUSD: 10, SpentUSD: 25, HasLimit: true}
	if got := over.Fraction(); got != 1 {
		t.Errorf("over budget fraction = %v", got)
	}
	if got := (Budget{}).Fraction(); got != 0 {
		t.Errorf("limitless fraction = %v", got)
	}
	negative := Budget{LimitUSD: 10, SpentUSD: -3, HasLimit: true}
	if got := negative.Fraction(); got != 0 {
		t.Errorf("negative fraction = %v", got)
	}
}
