package glm

import (
	"context"
	"errors"
	"math"
	"os"
	"path/filepath"
	"testing"

	"github.com/egoist/mygo/plugins/sqlite"
	"github.com/licoy/quosparr/internal/sqliteruntime"
)

// requireSQLite skips when the bundled SQLite library cannot be loaded, as on
// an offline machine with a cold cache.
func requireSQLite(t *testing.T) {
	t.Helper()
	if err := sqliteruntime.Load(); err != nil {
		t.Skipf("sqlite native library unavailable: %v", err)
	}
}

// usageStore writes a usage database into a temporary home. Only the columns
// the panel reads exist.
func usageStore(t *testing.T, statements ...string) string {
	t.Helper()
	requireSQLite(t)
	home := t.TempDir()
	path := DatabasePath(home)
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	db, err := sqlite.Open(context.Background(), path, sqlite.OpenOptions{})
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	ctx := context.Background()
	schema := []string{
		`CREATE TABLE model_usage (
		  id INTEGER PRIMARY KEY,
		  provider_id TEXT,
		  model_id TEXT,
		  input_tokens INTEGER,
		  output_tokens INTEGER,
		  reasoning_tokens INTEGER,
		  cache_creation_input_tokens INTEGER,
		  cache_read_input_tokens INTEGER)`,
	}
	for _, stmt := range append(schema, statements...) {
		if _, err := db.Execute(ctx, stmt); err != nil {
			t.Fatalf("%v", err)
		}
	}
	return home
}

func TestReadCostPricesEveryModelOfTheDatabase(t *testing.T) {
	home := usageStore(t,
		`INSERT INTO model_usage (provider_id, model_id, input_tokens, output_tokens, reasoning_tokens, cache_creation_input_tokens, cache_read_input_tokens)
		 VALUES ('account:bigmodel-individual-coding-plan', 'GLM-5.3', 1000000, 0, 0, 0, 0)`,
		`INSERT INTO model_usage (provider_id, model_id, input_tokens, output_tokens, reasoning_tokens, cache_creation_input_tokens, cache_read_input_tokens)
		 VALUES ('account:bigmodel-individual-coding-plan', 'GLM-5.3', 0, 1000000, 500000, 0, 0)`,
		`INSERT INTO model_usage (provider_id, model_id, input_tokens, output_tokens, reasoning_tokens, cache_creation_input_tokens, cache_read_input_tokens)
		 VALUES ('account:bigmodel-start-plan', 'GLM-5.3', 0, 0, 0, 0, 1000000)`,
	)
	cost, err := ReadCost(context.Background(), home)
	if err != nil {
		t.Fatal(err)
	}
	if cost.Path != DatabasePath(home) {
		t.Fatalf("path = %q", cost.Path)
	}
	if len(cost.Rows) != 1 {
		t.Fatalf("rows = %d; a model must aggregate over its providers' requests", len(cost.Rows))
	}
	if DatabasePath(home) != cost.Path {
		t.Fatalf("path = %q", cost.Path)
	}
	row := cost.Rows[0]
	if row.Model != "GLM-5.3" || row.Requests != 3 {
		t.Fatalf("row = %+v", row)
	}
	if row.Input != 1000000 || row.Output != 1000000 || row.CacheRead != 1000000 {
		t.Fatalf("row = %+v", row)
	}
	// 1M input + 1M output + 1M cache reads at the glm-5.3 rates.
	want := 1.4 + 4.4 + 0.26
	if !row.HasPrice || math.Abs(row.USD-want) > 1e-9 || math.Abs(cost.TotalUSD-want) > 1e-9 {
		t.Fatalf("row = %+v, total = %v", row, cost.TotalUSD)
	}
	if cost.Unpriced != 0 {
		t.Fatalf("unpriced = %d", cost.Unpriced)
	}
}

func TestReadCostChargesCacheReadsAtTheCacheRate(t *testing.T) {
	home := usageStore(t,
		`INSERT INTO model_usage (provider_id, model_id, input_tokens, output_tokens, reasoning_tokens, cache_creation_input_tokens, cache_read_input_tokens)
		 VALUES ('bigmodel-standard-api', 'GLM-5.3', 100, 1, 0, 0, 100000000)`,
	)
	cost, err := ReadCost(context.Background(), home)
	if err != nil {
		t.Fatal(err)
	}
	// 100M cache reads are $26 at the cache rate and $140 if they were
	// priced as input, the 65% overstatement the specification warns about.
	want := 26 + 100*1.4/1e6 + 4.4/1e6
	if math.Abs(cost.TotalUSD-want) > 1e-9 {
		t.Fatalf("total = %v", cost.TotalUSD)
	}
}

func TestReadCostMarksAnUnpricedModel(t *testing.T) {
	home := usageStore(t,
		`INSERT INTO model_usage (provider_id, model_id, input_tokens, output_tokens, reasoning_tokens, cache_creation_input_tokens, cache_read_input_tokens)
		 VALUES ('account:bigmodel-individual-coding-plan', 'GLM-5.3-Highspeed', 1000000, 1000000, 0, 0, 1000000)`,
		`INSERT INTO model_usage (provider_id, model_id, input_tokens, output_tokens, reasoning_tokens, cache_creation_input_tokens, cache_read_input_tokens)
		 VALUES ('account:bigmodel-individual-coding-plan', 'GLM-5.3-Flash', 1000000, 0, 0, 0, 0)`,
	)
	cost, err := ReadCost(context.Background(), home)
	if err != nil {
		t.Fatal(err)
	}
	if len(cost.Rows) != 2 {
		t.Fatalf("rows = %d", len(cost.Rows))
	}
	first := cost.Rows[0]
	if first.Model != "GLM-5.3-Flash" || !first.HasPrice {
		t.Fatalf("the priced row must sort first: %+v", cost.Rows)
	}
	last := cost.Rows[1]
	if last.HasPrice || last.Model != "GLM-5.3-Highspeed" {
		t.Fatalf("highspeed must be unpriced, not free: %+v", last)
	}
	if cost.Unpriced != 1 || math.Abs(cost.TotalUSD-0.15) > 1e-9 {
		t.Fatalf("unpriced = %d, total = %v", cost.Unpriced, cost.TotalUSD)
	}
}

func TestReadCostWithoutADatabaseIsNoUsage(t *testing.T) {
	home := t.TempDir()
	cost, err := ReadCost(context.Background(), home)
	if !errors.Is(err, ErrNoUsage) {
		t.Fatalf("err = %v", err)
	}
	if cost.Path != DatabasePath(home) {
		t.Fatalf("path = %q", cost.Path)
	}
	if _, err := ReadCost(context.Background(), ""); !errors.Is(err, ErrNoUsage) {
		t.Fatalf("empty home = %v", err)
	}
}

func TestReadCostWithoutTheUsageTableIsADatabaseError(t *testing.T) {
	requireSQLite(t)
	home := t.TempDir()
	path := DatabasePath(home)
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	db, err := sqlite.Open(context.Background(), path, sqlite.OpenOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Execute(context.Background(), "CREATE TABLE other (id INTEGER)"); err != nil {
		t.Fatal(err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := ReadCost(context.Background(), home); !errors.Is(err, ErrDatabase) {
		t.Fatalf("err = %v", err)
	}
}
