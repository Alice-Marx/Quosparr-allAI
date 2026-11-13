package glm

import (
	"context"
	"errors"
	"os"
	"sort"
	"time"

	"github.com/egoist/mygo/plugins/sqlite"
	"github.com/licoy/quosparr/internal/sqliteruntime"
)

var (
	// ErrNoUsage means the ZCode usage database has no model_usage rows.
	ErrNoUsage = errors.New("glm: usage database is missing")
	// ErrDatabase means the database exists but could not be read.
	ErrDatabase = errors.New("glm: usage database could not be read")
)

// costQuery sums the per-request usage of the database. input_tokens and the
// cache read column are separate (see the package doc); reasoning tokens are
// already inside output_tokens and are not charged again.
const costQuery = `SELECT model_id, COUNT(*),
       COALESCE(SUM(input_tokens), 0),
       COALESCE(SUM(output_tokens), 0),
       COALESCE(SUM(cache_read_input_tokens), 0)
  FROM model_usage
 GROUP BY model_id`

// Row is one model line of the local usage database, summed over every
// provider of the store.
type Row struct {
	Model     string
	Requests  int64
	Input     int64
	Output    int64
	CacheRead int64
	// HasPrice is false for a model without a rate; it is not free.
	HasPrice bool
	USD      float64
}

// Cost is the priced usage of one ZCode database.
type Cost struct {
	Path     string
	Rows     []Row
	Unpriced int
	TotalUSD float64
}

// ReadCost reads the ZCode usage database of home read-only and prices every
// model. A missing file or table is ErrNoUsage; an open or query failure is
// ErrDatabase.
func ReadCost(ctx context.Context, home string) (Cost, error) {
	cost := Cost{Path: DatabasePath(home)}
	if cost.Path == "" {
		return cost, ErrNoUsage
	}
	info, err := os.Stat(cost.Path)
	if err != nil || !info.Mode().IsRegular() {
		return cost, ErrNoUsage
	}
	if err := sqliteruntime.Load(); err != nil {
		return cost, ErrDatabase
	}
	db, err := sqlite.Open(ctx, cost.Path, sqlite.OpenOptions{ReadOnly: true, BusyTimeout: 5 * time.Second})
	if err != nil {
		return cost, ErrDatabase
	}
	rows, queryErr := db.Query(ctx, costQuery)
	closeErr := db.Close()
	if queryErr != nil || closeErr != nil {
		return cost, ErrDatabase
	}
	for _, values := range rows.Values {
		row, ok := parseCostRow(values)
		if !ok {
			return Cost{Path: cost.Path}, ErrDatabase
		}
		rate, priced := Lookup(row.Model)
		row.HasPrice = priced
		if priced {
			row.USD = rate.Cost(row.Input, row.Output, row.CacheRead)
			cost.TotalUSD += row.USD
		} else {
			cost.Unpriced++
		}
		cost.Rows = append(cost.Rows, row)
	}
	sortRows(cost.Rows)
	return cost, nil
}

// parseCostRow reads one row of costQuery. A missing number reads as zero, so
// a NULL column never fails the whole panel.
func parseCostRow(values []any) (Row, bool) {
	if len(values) < 5 {
		return Row{}, false
	}
	model, ok := values[0].(string)
	if !ok {
		return Row{}, false
	}
	return Row{
		Model:     model,
		Requests:  countOf(values[1]),
		Input:     countOf(values[2]),
		Output:    countOf(values[3]),
		CacheRead: countOf(values[4]),
	}, true
}

// countOf reads an integer column, which SQLite may also return as a float.
func countOf(value any) int64 {
	switch number := value.(type) {
	case int64:
		return number
	case float64:
		return int64(number)
	}
	return 0
}

// sortRows puts the priced models first, most expensive at the top.
func sortRows(rows []Row) {
	sort.SliceStable(rows, func(i, j int) bool {
		a, b := rows[i], rows[j]
		if a.USD != b.USD {
			return a.USD > b.USD
		}
		return a.Model < b.Model
	})
}
