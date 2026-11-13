package app

import (
	"context"
	"sync"
	"time"

	"github.com/licoy/quosparr/internal/kimi"
	"github.com/licoy/quosparr/internal/netx"
	"github.com/licoy/quosparr/internal/platform"
)

const (
	// kimiPollInterval is the poll of the vendor usage endpoint. The endpoint
	// is a live API, so it is asked at most once a minute.
	kimiPollInterval = 60 * time.Second
	// kimiCostInterval rescans the local session logs at most this often: they
	// change only while the CLI runs.
	kimiCostInterval = 5 * time.Minute
	// kimiFetchTimeout bounds one reading, request included.
	kimiFetchTimeout = 30 * time.Second
)

// kimiHold is the last Kimi Code reading. The watcher and the manual refresh
// are its only writers. It never stores a token.
type kimiHold struct {
	mu      sync.Mutex
	home    string
	report  kimi.Report
	costs   kimi.Costs
	costsAt time.Time
	prices  kimi.PriceTable
	loaded  bool
	loading bool
}

// kimiScreen is one consistent snapshot of the hold for the panel.
type kimiScreen struct {
	Report  kimi.Report
	Costs   kimi.Costs
	Prices  kimi.PriceTable
	Loaded  bool
	Loading bool
}

func (s kimiScreen) ready() bool {
	return s.Loaded && s.Report.State == kimi.StateReady
}

func (a *App) kimiScreen() kimiScreen {
	a.kimi.mu.Lock()
	s := kimiScreen{
		Report: a.kimi.report, Costs: a.kimi.costs, Prices: a.kimi.prices,
		Loaded: a.kimi.loaded, Loading: a.kimi.loading,
	}
	a.kimi.mu.Unlock()
	if s.Prices == nil {
		s.Prices = kimi.DefaultPrices()
	}
	return s
}

// kimiHome is the user home the reading resolves its paths from.
func (a *App) kimiHome() string {
	a.kimi.mu.Lock()
	home := a.kimi.home
	a.kimi.mu.Unlock()
	if home != "" {
		return home
	}
	if home, err := platform.Home(); err == nil {
		return home
	}
	return ""
}

// watchKimi keeps the Kimi Code reading current until ctx is done. The first
// reading happens at once, so the panel is never empty on startup.
func (a *App) watchKimi(ctx context.Context, home string) {
	a.kimi.mu.Lock()
	a.kimi.home = home
	a.kimi.mu.Unlock()
	a.refreshKimi()
	ticker := time.NewTicker(kimiPollInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			a.refreshKimi()
		}
	}
}

// refreshKimi refetches the quota and the local cost reading now, off the UI
// thread, and repaints on the main thread when the answer lands. A refresh that
// is already running wins: the endpoint is not asked twice in parallel.
func (a *App) refreshKimi() {
	a.kimi.mu.Lock()
	if a.kimi.loading {
		a.kimi.mu.Unlock()
		return
	}
	a.kimi.loading = true
	scan := a.kimi.costsAt.IsZero() || time.Since(a.kimi.costsAt) >= kimiCostInterval
	prices := a.kimi.prices
	a.kimi.mu.Unlock()
	if prices == nil {
		prices = kimi.DefaultPrices()
	}
	runOnMain(func() { a.paintKimi() })
	go a.fetchKimi(scan, prices)
}

// fetchKimi does one reading. The hold's loading flag is set by refreshKimi and
// cleared here, so the panel shows the wait exactly once.
func (a *App) fetchKimi(scanCosts bool, prices kimi.PriceTable) {
	home := a.kimiHome()
	ctx, cancel := context.WithTimeout(context.Background(), kimiFetchTimeout)
	defer cancel()
	var report kimi.Report
	client, err := netx.ClientFor(ctx, netx.WithoutCookies)
	if err != nil {
		report = kimi.Report{State: kimi.StateFailed, Err: err, Base: kimi.BaseURL(home), Region: kimi.Region(home)}
	} else {
		report = kimi.Fetch(ctx, kimi.Config{Home: home, Client: client})
	}
	var costs kimi.Costs
	scannedAt := time.Time{}
	if scanCosts {
		costs = kimi.ScanCosts(home)
		scannedAt = time.Now()
	}
	a.kimi.mu.Lock()
	a.kimi.report = report
	a.kimi.prices = prices
	if scanCosts {
		a.kimi.costs = costs
		a.kimi.costsAt = scannedAt
	}
	a.kimi.loaded = true
	a.kimi.loading = false
	a.kimi.mu.Unlock()
	runOnMain(func() { a.paintKimi() })
}

func (a *App) paintKimi() {
	if a.panel != nil {
		a.panel.Invalidate()
	}
}
