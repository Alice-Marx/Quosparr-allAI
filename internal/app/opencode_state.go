package app

import (
	"context"
	"sync"
	"time"

	"github.com/licoy/quosparr/internal/opencode"
	"github.com/licoy/quosparr/internal/platform"
)

// openCodeRefreshInterval is how often the watcher rereads OpenCode. Both
// answers are windows of hours or days and the stored login only changes when
// the CLI signs in, so a shorter poll would cost requests and show nothing.
const openCodeRefreshInterval = 5 * time.Minute

// openCodeHold is the last OpenCode reading. The watcher and the manual
// refresh are its only writers. It never stores a token: opencode.Snapshot
// keeps the access token unexported.
type openCodeHold struct {
	mu      sync.Mutex
	home    string
	snap    opencode.Snapshot
	loaded  bool
	loading bool
}

// openCodeScreen is the panel's copy of the reading.
type openCodeScreen struct {
	Snap    opencode.Snapshot
	Loaded  bool
	Loading bool
}

func (a *App) openCodeScreen() openCodeScreen {
	a.opencode.mu.Lock()
	defer a.opencode.mu.Unlock()
	return openCodeScreen{Snap: a.opencode.snap, Loaded: a.opencode.loaded, Loading: a.opencode.loading}
}

func (a *App) openCodeHome() string {
	a.opencode.mu.Lock()
	home := a.opencode.home
	a.opencode.mu.Unlock()
	if home != "" {
		return home
	}
	if h, err := platform.Home(); err == nil {
		return h
	}
	return ""
}

// watchOpenCode keeps the OpenCode reading current until ctx is done: once at
// startup and then on an interval.
func (a *App) watchOpenCode(ctx context.Context, home string) {
	if home == "" {
		return
	}
	a.opencode.mu.Lock()
	a.opencode.home = home
	a.opencode.mu.Unlock()
	a.recordOpenCode(ctx, home)
	ticker := time.NewTicker(openCodeRefreshInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			a.recordOpenCode(ctx, home)
		}
	}
}

// recordOpenCode loads the reading and repaints. Both call sites run it off
// the UI thread, so the repaint is handed to the main thread.
func (a *App) recordOpenCode(ctx context.Context, home string) {
	snap := opencode.Load(ctx, home, opencode.Client{}, time.Now())
	a.opencode.mu.Lock()
	a.opencode.snap = snap
	a.opencode.loaded = true
	a.opencode.loading = false
	a.opencode.mu.Unlock()
	runOnMain(func() { a.paintOpenCode() })
}

// refreshOpenCode refetches now. The work runs off the UI thread, because the
// panel and the provider tab both call it while drawing.
func (a *App) refreshOpenCode() {
	home := a.openCodeHome()
	if home == "" {
		return
	}
	a.opencode.mu.Lock()
	if a.opencode.loading {
		a.opencode.mu.Unlock()
		return
	}
	a.opencode.home = home
	a.opencode.loading = true
	a.opencode.mu.Unlock()
	runOnMain(func() { a.paintOpenCode() })
	go a.recordOpenCode(context.Background(), home)
}

func (a *App) paintOpenCode() {
	if a.panel != nil {
		a.panel.Invalidate()
	}
}
