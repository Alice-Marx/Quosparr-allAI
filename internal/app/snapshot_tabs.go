package app

import "github.com/licoy/quosparr/internal/store"

// refreshSnapshotTab refetches the quota reading of a panel-only tab and
// reports whether p was one. The header refresh and the preference listener
// both use it, so a new quota tab is wired once.
func (a *App) refreshSnapshotTab(p store.Provider) bool {
	switch p {
	case store.ProviderGrokBot:
		a.refreshGrokBot()
	case store.ProviderCursor:
		a.refreshCursor()
	case store.ProviderKimi:
		a.refreshKimi()
	case store.ProviderGLM:
		a.refreshGLM()
	case store.ProviderOpenCode:
		a.refreshOpenCode()
	case store.ProviderGemini:
		a.refreshGemini()
	default:
		return false
	}
	return true
}
