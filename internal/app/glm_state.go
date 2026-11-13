package app

import (
	"context"
	"errors"
	"log"
	"sync"
	"time"

	"github.com/licoy/quosparr/internal/glm"
	"github.com/licoy/quosparr/internal/platform"
)

const (
	// glmReadTimeout bounds one refresh started from the panel.
	glmReadTimeout = 45 * time.Second
	// glmPollInterval keeps the live quota endpoint under the rate the ZCode
	// CLI itself uses (five minutes), so polling cannot trip vendor risk
	// control.
	glmPollInterval = 5 * time.Minute
)

// glmHold is the last GLM reading. The watcher and the manual refresh are its
// only writers. It stores the credential candidate the endpoint accepted, so
// the panel can name it, and never formats it anywhere.
type glmHold struct {
	mu        sync.Mutex
	credPath  string
	dbPath    string
	candidate glm.Candidate
	quota     glm.Quota
	quotaErr  error
	cost      glm.Cost
	costErr   error
	fetchedAt time.Time
	running   bool
}

// glmScreen is one reading of the hold.
type glmScreen struct {
	CredPath  string
	DBPath    string
	Candidate glm.Candidate
	Quota     glm.Quota
	QuotaErr  error
	Cost      glm.Cost
	CostErr   error
	FetchedAt time.Time
	Running   bool
}

// ready reports a usable quota reading.
func (s glmScreen) ready() bool { return s.QuotaErr == nil && s.Quota.HasPrimary }

// glmScreen copies the hold out from under its lock.
func (a *App) glmScreen() glmScreen {
	a.glm.mu.Lock()
	s := glmScreen{
		CredPath: a.glm.credPath, DBPath: a.glm.dbPath,
		Candidate: a.glm.candidate, Quota: a.glm.quota, QuotaErr: a.glm.quotaErr,
		Cost: a.glm.cost, CostErr: a.glm.costErr, FetchedAt: a.glm.fetchedAt,
		Running: a.glm.running,
	}
	a.glm.mu.Unlock()
	if s.CredPath == "" || s.DBPath == "" {
		if home, err := platform.Home(); err == nil {
			if s.CredPath == "" {
				s.CredPath = glm.CredentialPath(home)
			}
			if s.DBPath == "" {
				s.DBPath = glm.DatabasePath(home)
			}
		}
	}
	return s
}

// newGLMClient is the network client of the GLM panel. Tests replace it to
// point the fetcher at an httptest server.
var newGLMClient = func() glm.Client { return glm.Client{} }

// watchGLM keeps the GLM reading current until ctx is done. The first read
// runs at once, then every poll interval.
func (a *App) watchGLM(ctx context.Context, home string) {
	a.glm.mu.Lock()
	a.glm.credPath = glm.CredentialPath(home)
	a.glm.dbPath = glm.DatabasePath(home)
	a.glm.mu.Unlock()
	ticker := time.NewTicker(glmPollInterval)
	defer ticker.Stop()
	for {
		if a.beginGLM() {
			a.loadGLM(ctx, home)
			a.endGLM()
			runOnMain(func() { a.paintGLM() })
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

// refreshGLM refetches the GLM reading off the main thread and repaints when
// it lands; a request while one is in flight is dropped.
func (a *App) refreshGLM() {
	home, err := platform.Home()
	if err != nil {
		home = ""
	}
	if !a.beginGLM() {
		return
	}
	runOnMain(func() { a.paintGLM() })
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), glmReadTimeout)
		defer cancel()
		defer a.endGLM()
		a.loadGLM(ctx, home)
		runOnMain(func() { a.paintGLM() })
	}()
}

// beginGLM takes the running flag; a second caller is dropped.
func (a *App) beginGLM() bool {
	a.glm.mu.Lock()
	defer a.glm.mu.Unlock()
	if a.glm.running {
		return false
	}
	a.glm.running = true
	return true
}

func (a *App) endGLM() {
	a.glm.mu.Lock()
	a.glm.running = false
	a.glm.mu.Unlock()
}

// loadGLM reads the credential store, the quota and the local cost. It is the
// only writer of the hold and does the blocking work of the caller.
func (a *App) loadGLM(ctx context.Context, home string) {
	credPath, dbPath := glm.CredentialPath(home), glm.DatabasePath(home)
	store, storeErr := glm.Load(home)
	cost, costErr := glm.ReadCost(ctx, home)
	quota, candidate, quotaErr := fetchGLMQuota(ctx, newGLMClient(), store, storeErr)
	if errors.Is(quotaErr, glm.ErrCredentialFormat) {
		log.Println("glm credential store: unreadable")
	}
	if errors.Is(costErr, glm.ErrDatabase) {
		log.Println("glm usage database: unreadable")
	}
	a.glm.mu.Lock()
	a.glm.credPath, a.glm.dbPath = credPath, dbPath
	a.glm.candidate = candidate
	a.glm.quota, a.glm.quotaErr = quota, quotaErr
	a.glm.cost, a.glm.costErr = cost, costErr
	a.glm.fetchedAt = time.Now()
	a.glm.mu.Unlock()
}

// fetchGLMQuota tries the stored credentials in order. A store that could
// not be read is reported before any request is made.
func fetchGLMQuota(ctx context.Context, client glm.Client, store glm.Store, storeErr error) (glm.Quota, glm.Candidate, error) {
	if storeErr != nil {
		return glm.Quota{}, glm.Candidate{}, storeErr
	}
	if !store.Usable() {
		return glm.Quota{}, glm.Candidate{}, glm.ErrNoCredentials
	}
	return client.FetchFirst(ctx, store.Candidates)
}

// paintGLM repaints the panel on the main thread.
func (a *App) paintGLM() {
	if a.panel != nil {
		a.panel.Invalidate()
	}
}
