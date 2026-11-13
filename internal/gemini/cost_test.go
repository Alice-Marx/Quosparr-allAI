package gemini

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeSession puts one chat log into a temporary home and returns that home.
func writeSession(t *testing.T, home, workspace, name, body string) {
	t.Helper()
	dir := ChatsDir(home, workspace)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, SessionFilePrefix+name+".jsonl"), []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
}

const oneAnswer = `{"id":"m1","timestamp":"2026-10-10T12:04:06.527Z","type":"gemini","content":"hi","tokens":{"input":12886,"output":9,"cached":0,"thoughts":134,"tool":0,"total":13029},"model":"gemini-3.8-flash"}`

func TestScanCostFoldsThePatchStream(t *testing.T) {
	isolateEnv(t)
	home := t.TempDir()
	body := strings.Join([]string{
		`{"sessionId":"s1","projectHash":"h","startTime":"2026-10-10T12:00:00Z","kind":"main"}`,
		`{"$set":{"messages":[{"id":"u1","type":"user","content":"hello"}]}}`,
		`{"$set":{"lastUpdated":"2026-10-10T12:04:06Z"}}`,
		oneAnswer,
		`{"$set":{"memoryScratchpad":{"version":1}}}`,
		`{"id":"m2","type":"gemini","tokens":{"input":1000,"output":100,"cached":400,"thoughts":7,"tool":11,"total":1107},"model":"gemini-2.5-pro"}`,
	}, "\n") + "\n"
	writeSession(t, home, "workspace-a", "s1", body)

	ledger := ScanCost(home)
	if !ledger.HasAny() || ledger.Sessions != 1 || ledger.Truncated {
		t.Fatalf("ledger = %+v", ledger)
	}
	if len(ledger.Models) != 2 {
		t.Fatalf("models = %+v", ledger.Models)
	}
	byModel := map[string]Usage{}
	for _, usage := range ledger.Models {
		byModel[usage.Model] = usage
	}
	flash := byModel["gemini-3.8-flash"]
	if flash.Input != 12886 || flash.Output != 9 || flash.Thoughts != 134 || flash.Requests != 1 {
		t.Fatalf("flash = %+v", flash)
	}
	pro := byModel["gemini-2.5-pro"]
	if pro.Input != 1000 || pro.Cached != 400 || pro.Output != 100 || pro.Thoughts != 7 || pro.Tool != 11 {
		t.Fatalf("pro = %+v", pro)
	}
	// The $set replaced the whole array, so the user message is gone and only
	// the two answers count.
	if ledger.Models[0].Model != "gemini-3.8-flash" {
		t.Fatalf("the largest spend must come first: %+v", ledger.Models)
	}
}

func TestScanCostRewindsToTheMessage(t *testing.T) {
	isolateEnv(t)
	home := t.TempDir()
	body := strings.Join([]string{
		`{"$set":{"messages":[` + oneAnswer + `,
{"id":"m2","type":"gemini","tokens":{"input":50,"output":5},"model":"gemini-2.5-pro"}]}}`,
		`{"$rewindTo":"m1"}`,
		`{"id":"m1","type":"gemini","tokens":{"input":9000,"output":10},"model":"gemini-2.5-flash"}`,
	}, "\n") + "\n"
	writeSession(t, home, "workspace-a", "s1", body)

	ledger := ScanCost(home)
	if len(ledger.Models) != 1 {
		t.Fatalf("models = %+v", ledger.Models)
	}
	only := ledger.Models[0]
	if only.Model != "gemini-2.5-flash" || only.Input != 9000 || only.Output != 10 {
		t.Fatalf("the rewind must drop the second answer and keep the rewritten one: %+v", only)
	}
}

func TestScanCostOfAnEmptyHome(t *testing.T) {
	isolateEnv(t)
	ledger := ScanCost(t.TempDir())
	if ledger.HasAny() || ledger.Sessions != 0 {
		t.Fatalf("ledger = %+v", ledger)
	}
	if _, ok := CostRows(ledger); ok {
		t.Fatal("an empty ledger has no rows")
	}
}

func TestCostRowsPriceCachedApartFromInput(t *testing.T) {
	isolateEnv(t)
	ledger := Ledger{Models: []Usage{{
		Model: "gemini-2.5-pro", Input: 1_000_000, Cached: 500_000, Output: 0, Thoughts: 0, Requests: 1,
	}}}
	rows, ok := CostRows(ledger)
	if !ok || len(rows) != 1 {
		t.Fatalf("rows = %+v", rows)
	}
	row := rows[0]
	if row.Input != 500_000 || row.Cached != 500_000 || row.Missing || row.Estimated {
		t.Fatalf("row = %+v", row)
	}
	// 500k uncached at $1.25/M plus 500k cached at $0.125/M.
	if row.Amount != "$0.6875" {
		t.Fatalf("amount = %q", row.Amount)
	}
}

func TestCostRowsPriceThinkingAsOutput(t *testing.T) {
	isolateEnv(t)
	ledger := Ledger{Models: []Usage{{
		Model: "gemini-2.5-pro", Input: 0, Output: 500_000, Thoughts: 500_000, Requests: 1,
	}}}
	rows, _ := CostRows(ledger)
	if rows[0].Output != 1_000_000 {
		t.Fatalf("output = %d", rows[0].Output)
	}
	if rows[0].Amount != "$10.00" {
		t.Fatalf("amount = %q", rows[0].Amount)
	}
}

func TestCostRowsClampCachedToInput(t *testing.T) {
	isolateEnv(t)
	ledger := Ledger{Models: []Usage{{Model: "gemini-2.5-flash", Input: 100, Cached: 400, Requests: 1}}}
	rows, _ := CostRows(ledger)
	if rows[0].Cached != 100 || rows[0].Input != 0 {
		t.Fatalf("row = %+v", rows[0])
	}
}

func TestCostRowsWithoutAPrice(t *testing.T) {
	isolateEnv(t)
	unknown := Ledger{Models: []Usage{{Model: "gemini-nonexistent-1", Input: 10_000, Requests: 1}}}
	rows, _ := CostRows(unknown)
	if !rows[0].Missing || rows[0].Amount != "" {
		t.Fatalf("row = %+v", rows[0])
	}
	familyLedger := Ledger{Models: []Usage{{Model: "gemini-3.8-flash", Input: 1_000_000, Requests: 1}}}
	rows, _ = CostRows(familyLedger)
	if rows[0].Missing || !rows[0].Estimated || rows[0].Amount != "$0.30" {
		t.Fatalf("row = %+v", rows[0])
	}
	empty := Ledger{Models: []Usage{{Model: "", Input: 5}}}
	rows, _ = CostRows(empty)
	if !rows[0].Missing {
		t.Fatalf("a model the log never recorded is unpriced: %+v", rows[0])
	}
}

func TestFormatUSD(t *testing.T) {
	isolateEnv(t)
	for value, want := range map[float64]string{0: "$0.00", 1.25: "$1.25", 0.0123: "$0.0123", 2: "$2.00", 0.6875: "$0.6875"} {
		if got := FormatUSD(value); got != want {
			t.Errorf("FormatUSD(%v) = %q, want %q", value, got, want)
		}
	}
}
