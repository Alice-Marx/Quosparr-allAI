package opencode

import "context"

// Tokens is a token count per kind, as the CLI records it.
type Tokens struct {
	Input      int64
	Output     int64
	Reasoning  int64
	CacheRead  int64
	CacheWrite int64
}

// ModelCost is one provider/model group of the recorded spend.
type ModelCost struct {
	Provider string
	Model    string
	CostUSD  float64
	Tokens   Tokens
	Sessions int64
}

// Cost is the spend the CLI recorded for the local sessions. The amounts are
// the ones the CLI wrote per session; this package does not re-price tokens.
type Cost struct {
	TotalUSD float64
	Tokens   Tokens
	Sessions int64
	Models   []ModelCost
}

// costQuery groups the session rows so one query yields both the groups and,
// by summing them, the total.
const costQuery = `SELECT COALESCE(json_extract(model, '$.providerID'), ''),
       COALESCE(json_extract(model, '$.id'), ''),
       COALESCE(SUM(cost), 0),
       COALESCE(SUM(tokens_input), 0),
       COALESCE(SUM(tokens_output), 0),
       COALESCE(SUM(tokens_reasoning), 0),
       COALESCE(SUM(tokens_cache_read), 0),
       COALESCE(SUM(tokens_cache_write), 0),
       COUNT(*)
 FROM session_v2
 GROUP BY 1, 2
 ORDER BY 3 DESC, 1, 2
 LIMIT 40`

const costColumns = 9

// readCost runs the grouped cost query.
func readCost(ctx context.Context, db queryer) (Cost, error) {
	rows, err := db.Query(ctx, costQuery)
	if err != nil {
		return Cost{}, ErrDatabase
	}
	return costFromRows(rows)
}

// costFromRows maps the cost rows. A row whose columns do not have the
// expected shape fails the whole reading rather than reporting a wrong total.
func costFromRows(rows Rows) (Cost, error) {
	out := Cost{}
	if len(rows.Columns) != costColumns {
		return Cost{}, ErrDecode
	}
	for _, row := range rows.Values {
		if len(row) != costColumns {
			return Cost{}, ErrDecode
		}
		provider, pok := row[0].(string)
		model, mok := row[1].(string)
		cost, cok := asFloat(row[2])
		in, iok := asInt(row[3])
		output, ook := asInt(row[4])
		reasoning, rok := asInt(row[5])
		cacheRead, crok := asInt(row[6])
		cacheWrite, cwok := asInt(row[7])
		sessions, sok := asInt(row[8])
		if !pok || !mok || !cok || !iok || !ook || !rok || !crok || !cwok || !sok {
			return Cost{}, ErrDecode
		}
		group := ModelCost{
			Provider: provider, Model: model, CostUSD: cost, Sessions: sessions,
			Tokens: Tokens{Input: in, Output: output, Reasoning: reasoning, CacheRead: cacheRead, CacheWrite: cacheWrite},
		}
		out.Models = append(out.Models, group)
		out.TotalUSD += group.CostUSD
		out.Sessions += group.Sessions
		out.Tokens.Input += group.Tokens.Input
		out.Tokens.Output += group.Tokens.Output
		out.Tokens.Reasoning += group.Tokens.Reasoning
		out.Tokens.CacheRead += group.Tokens.CacheRead
		out.Tokens.CacheWrite += group.Tokens.CacheWrite
	}
	return out, nil
}

func asFloat(v any) (float64, bool) {
	switch n := v.(type) {
	case float64:
		return n, true
	case int64:
		return float64(n), true
	}
	return 0, false
}

func asInt(v any) (int64, bool) {
	switch n := v.(type) {
	case int64:
		return n, true
	case float64:
		return int64(n), true
	}
	return 0, false
}
