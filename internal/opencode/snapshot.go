package opencode

import (
	"context"
	"errors"
	"os"
	"time"

	"github.com/egoist/mygo/plugins/sqlite"
	"github.com/licoy/quosparr/internal/sqliteruntime"
)

var (
	// ErrNoCredential means the CLI has no OpenCode Console login here.
	ErrNoCredential = errors.New("opencode: no console login")
	// ErrCredential means a stored login could not be read.
	ErrCredential = errors.New("opencode: credential unreadable")
	// ErrNoDatabase means the CLI database is not on this machine. It is the
	// normal answer when the CLI was never installed.
	ErrNoDatabase = errors.New("opencode: no local database")
	// ErrDatabase means the database exists but could not be read. This
	// package never creates, migrates, or repairs it.
	ErrDatabase = errors.New("opencode: local database unreadable")
)

// queryer is the one SQLite operation this package uses.
type queryer interface {
	Query(ctx context.Context, sql string, args ...any) (sqlite.Rows, error)
}

// Rows is the result set of the SQLite plugin, so a caller (or a test) can
// build one without a database.
type Rows = sqlite.Rows

// credentialQuery picks the active Console login, newest first.
const credentialQuery = `SELECT value FROM credential
 WHERE integration_id = 'opencode' AND COALESCE(active, 1) <> 0
 ORDER BY time_updated DESC
 LIMIT 1`

// State is the overall answer of one refresh.
type State int

const (
	// StateUnknown is a refresh that has not finished yet.
	StateUnknown State = iota
	// StateReady has a usable login and, possibly, a reading.
	StateReady
	// StateNoCredential has no stored login.
	StateNoCredential
	// StateExpired has a login that is past its expiry.
	StateExpired
	// StateUnauthorized has a login the endpoint no longer accepts.
	StateUnauthorized
	// StateError could not read the local login at all.
	StateError
)

// Identity is the account line of the panel. It never carries a token.
type Identity struct {
	OrgID    string
	OrgName  string
	Email    string
	UsageURL string
}

// Snapshot is one reading of the panel. It holds no token, so it is safe to
// keep for the life of the app.
type Snapshot struct {
	FetchedAt time.Time
	State     State
	Identity  Identity

	Usage    Usage
	HasUsage bool
	// UsageOff is a 403 from the usage endpoint: the organization has no Go
	// usage page. UsageErr is any other failure.
	UsageOff bool
	UsageErr error

	Budgets      []Budget
	BudgetsKnown bool
	// BudgetsOff is a 403 from the budget endpoint.
	BudgetsOff bool
	BudgetsErr error

	Cost    Cost
	HasCost bool

	DatabasePath string
	// Err is the failure that stops the quota section: no readable login.
	Err error
	// LocalErr is a database that exists but could not be read. The cost
	// section reports it and the quota section still works.
	LocalErr error
}

// Local is what the CLI data directory holds.
type Local struct {
	DatabasePath  string
	DatabaseRead  bool
	Credential    Credential
	HasCredential bool
	Cost          Cost
	CostRead      bool
	Err           error
}

// Read reads the local database, falling back to the legacy auth file for the
// login. A missing database is not an error; an unreadable one is Err.
func Read(ctx context.Context, home string) Local {
	out := Local{DatabasePath: DatabasePath(home)}
	db, err := openDatabase(ctx, out.DatabasePath)
	if db != nil {
		defer func() { _ = db.Close() }()
		out.DatabaseRead = true
		if cred, ok := readCredential(ctx, db); ok {
			out.Credential, out.HasCredential = cred, true
		}
		if cost, err := readCost(ctx, db); err == nil {
			out.Cost, out.CostRead = cost, true
		} else {
			out.Err = err
		}
	} else if !errors.Is(err, ErrNoDatabase) {
		out.Err = err
	}
	if !out.HasCredential {
		if cred, ok := readAuthFile(home); ok {
			out.Credential, out.HasCredential = cred, true
		}
	}
	return out
}

// openDatabase opens the CLI database read only. The CLI may have it open, so
// this package never writes to it, never migrates it, and never copies it.
func openDatabase(ctx context.Context, path string) (*sqlite.DB, error) {
	info, err := os.Lstat(path)
	if err != nil || !info.Mode().IsRegular() {
		return nil, ErrNoDatabase
	}
	if err := sqliteruntime.Load(); err != nil {
		return nil, ErrDatabase
	}
	db, err := sqlite.Open(ctx, path, sqlite.OpenOptions{ReadOnly: true})
	if err != nil {
		return nil, ErrDatabase
	}
	return db, nil
}

func readCredential(ctx context.Context, db queryer) (Credential, bool) {
	rows, err := db.Query(ctx, credentialQuery)
	if err != nil || len(rows.Values) != 1 || len(rows.Values[0]) != 1 {
		return Credential{}, false
	}
	raw, ok := rows.Values[0][0].(string)
	if !ok {
		return Credential{}, false
	}
	cred, err := parseCredential([]byte(raw))
	if err != nil {
		return Credential{}, false
	}
	return cred, true
}

// Load reads the local data and, when a usable login exists, both endpoints.
// It never returns an error: the snapshot says what happened, because the
// panel has to render every outcome.
func Load(ctx context.Context, home string, client Client, now time.Time) Snapshot {
	out := Snapshot{FetchedAt: now}
	local := Read(ctx, home)
	out.DatabasePath = local.DatabasePath
	out.LocalErr = local.Err
	out.Cost, out.HasCost = local.Cost, local.CostRead
	switch {
	case !local.HasCredential:
		out.State = StateNoCredential
		return out
	case local.Credential.Expired(now):
		out.State = StateExpired
		return out
	}
	cred := local.Credential
	out.Identity = identityOf(cred)
	out.State = StateReady
	usage, ok, err := client.Usage(ctx, cred)
	switch {
	case errors.Is(err, ErrUnauthorized):
		out.State, out.Err = StateUnauthorized, err
		return out
	case err != nil:
		out.UsageErr = err
	default:
		out.Usage, out.HasUsage, out.UsageOff = usage, ok, !ok
	}
	budgets, ok, err := client.Budgets(ctx, cred)
	switch {
	case errors.Is(err, ErrUnauthorized):
		// Keep the usage that did arrive; the panel asks for a new login.
		out.Err = err
	case err != nil:
		out.BudgetsErr = err
	default:
		out.Budgets, out.BudgetsKnown, out.BudgetsOff = budgets, ok, !ok
	}
	return out
}

// identityOf is the account line: the organization name, else its id, plus
// the Console page of that organization.
func identityOf(cred Credential) Identity {
	id := Identity{OrgID: cred.OrgID, OrgName: cred.OrgName, Email: cred.Email}
	if cred.OrgID != "" {
		id.UsageURL = "https://opencode.ai/console/" + cred.OrgID + "/activity/overview"
	}
	return id
}

// Name is the account label of the panel.
func (i Identity) Name() string {
	switch {
	case i.OrgName != "":
		return i.OrgName
	case i.OrgID != "":
		return i.OrgID
	case i.Email != "":
		return i.Email
	}
	return ""
}
