package app

import (
	"context"
	"sync"
	"time"

	"github.com/licoy/quosparr/internal/gemini"
	"github.com/licoy/quosparr/internal/platform"
)

const (
	// geminiPollInterval is how often the watcher refetches. The service
	// caches its own answer for 30 seconds and the window resets daily, so a
	// slower poll is enough and keeps the endpoints quiet.
	geminiPollInterval = 5 * time.Minute
	// geminiFetchTimeout bounds one reading, including a token refresh.
	geminiFetchTimeout = 45 * time.Second
)

// geminiHold is the last Gemini reading. The watcher and the manual refresh
// are its only writers. It never stores a token.
type geminiHold struct {
	mu      sync.Mutex
	home    string
	reading gemini.Reading
	fetched bool
	busy    bool
}

// geminiScreen is the panel's view of the hold.
type geminiScreen struct {
	// Home is the home directory the reading was taken for, so the panel can
	// name the credential path without guessing.
	Home    string
	Reading gemini.Reading
	Fetched bool
	Busy    bool
}

func (a *App) geminiScreen() geminiScreen {
	a.gemini.mu.Lock()
	s := geminiScreen{Reading: a.gemini.reading, Fetched: a.gemini.fetched, Busy: a.gemini.busy}
	home := a.gemini.home
	a.gemini.mu.Unlock()
	if home == "" {
		if resolved, err := platform.Home(); err == nil {
			home = resolved
		}
	}
	s.Home = home
	return s
}

func (a *App) geminiHome() string {
	a.gemini.mu.Lock()
	home := a.gemini.home
	a.gemini.mu.Unlock()
	if home != "" {
		return home
	}
	resolved, err := platform.Home()
	if err != nil {
		return ""
	}
	return resolved
}

// setGeminiReading stores one reading and the home it was read for.
func (a *App) setGeminiReading(home string, reading gemini.Reading) {
	a.gemini.mu.Lock()
	a.gemini.home = home
	a.gemini.reading = reading
	a.gemini.fetched = true
	a.gemini.busy = false
	a.gemini.mu.Unlock()
}

// watchGemini keeps the quota current until ctx is done.
func (a *App) watchGemini(ctx context.Context, home string) {
	a.gemini.mu.Lock()
	a.gemini.home = home
	a.gemini.mu.Unlock()
	a.refreshGemini()
	ticker := time.NewTicker(geminiPollInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			a.refreshGemini()
		}
	}
}

// refreshGemini refetches the reading now and repaints. The fetch runs off the
// UI thread because it waits for the network; the panel draws its busy state
// in the meantime, and a second call while one is in flight is ignored.
func (a *App) refreshGemini() {
	if !a.beginGeminiFetch() {
		return
	}
	home := a.geminiHome()
	runOnMain(func() { a.paintGemini() })
	go func() {
		a.loadGemini(home)
		runOnMain(func() { a.paintGemini() })
	}()
}

func (a *App) beginGeminiFetch() bool {
	a.gemini.mu.Lock()
	defer a.gemini.mu.Unlock()
	if a.gemini.busy {
		return false
	}
	a.gemini.busy = true
	return true
}

func (a *App) loadGemini(home string) {
	ctx, cancel := context.WithTimeout(context.Background(), geminiFetchTimeout)
	defer cancel()
	a.setGeminiReading(home, gemini.Fetch(ctx, home, gemini.Options{}))
}

func (a *App) paintGemini() {
	if a.panel != nil {
		a.panel.Invalidate()
	}
}
