package opencode

import (
	"errors"
	"testing"
)

func TestCostFromRowsSumsGroupsAndTotals(t *testing.T) {
	rows := Rows{
		Columns: []string{"provider", "model", "cost", "input", "output", "reasoning", "cache_read", "cache_write", "sessions"},
		Values: [][]any{
			{"opencode-go", "kimi-k3.1-pro", 3.277566, int64(10217), int64(40503), int64(1105323), int64(467710541), int64(18798966), int64(41)},
			{"opencode", "deepseek-v4.1-flash-free", 6.3e-05, int64(78532), int64(8337), int64(0), int64(0), int64(0), int64(12)},
		},
	}
	cost, err := costFromRows(rows)
	if err != nil {
		t.Fatal(err)
	}
	if len(cost.Models) != 2 || cost.Sessions != 53 {
		t.Fatalf("cost = %+v", cost)
	}
	if cost.Tokens.Input != 88749 || cost.Tokens.CacheRead != 467710541 {
		t.Fatalf("tokens = %+v", cost.Tokens)
	}
	if cost.TotalUSD < 3.2776 || cost.TotalUSD > 3.2777 {
		t.Fatalf("total = %v", cost.TotalUSD)
	}
	if cost.Models[0].Provider != "opencode-go" || cost.Models[0].Model != "kimi-k3.1-pro" {
		t.Fatalf("first group = %+v", cost.Models[0])
	}
}

func TestCostFromRowsRejectsAnUnexpectedShape(t *testing.T) {
	if _, err := costFromRows(Rows{Columns: []string{"a"}}); !errors.Is(err, ErrDecode) {
		t.Errorf("short column list: err = %v", err)
	}
	bad := Rows{
		Columns: make([]string, costColumns),
		Values:  [][]any{{"provider", 1, 2.0}},
	}
	if _, err := costFromRows(bad); !errors.Is(err, ErrDecode) {
		t.Errorf("short row: err = %v", err)
	}
}

func TestCostFromRowsAcceptsAnEmptyDatabase(t *testing.T) {
	cost, err := costFromRows(Rows{Columns: make([]string, costColumns)})
	if err != nil {
		t.Fatal(err)
	}
	if cost.Sessions != 0 || cost.TotalUSD != 0 || len(cost.Models) != 0 {
		t.Fatalf("cost = %+v", cost)
	}
}
