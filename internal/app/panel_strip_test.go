package app

import (
	"testing"

	"github.com/licoy/quosparr/internal/l10n"
	"github.com/licoy/quosparr/internal/store"
)

// TestProviderStripKeepsEveryLabelReadable locks the tab geometry: with nine
// providers a single row left every label a few DIP wide, and the strip
// rendered an ellipsis instead of the name. Each name must fit, and the
// quota tabs must sit on a wrapped second row.
func TestProviderStripKeepsEveryLabelReadable(t *testing.T) {
	l := l10n.New(l10n.English)
	_, tt := newPanel(t, nil, panelFixture(), PanelActions{})
	all := append(store.Providers(), store.SnapshotProviders()...)
	for _, p := range all {
		name := providerLabel(l, p)
		rect, ok := tt.Find(name)
		if !ok {
			t.Fatalf("the %q tab renders no label", name)
		}
		// A clipped label collapses to an ellipsis a few DIP wide; a real one
		// needs at least the width of its shortest word.
		if rect.W < 30 {
			t.Fatalf("the %q tab is only %v DIP wide, too narrow to read", name, rect.W)
		}
	}
	first, ok := tt.Find(providerLabel(l, store.ProviderCodex))
	if !ok {
		t.Fatal("the CLI row is missing")
	}
	second, ok := tt.Find(providerLabel(l, store.ProviderKimi))
	if !ok {
		t.Fatal("the quota row is missing")
	}
	if second.Y <= first.Y {
		t.Fatalf("the quota tabs are not on a second row: %v then %v", first.Y, second.Y)
	}
	// Both rows share one column width, so the same column keeps one centre
	// across the two rows: Codex sits over Kimi Code, Grok over OpenCode.
	for _, pair := range [][2]store.Provider{
		{store.ProviderCodex, store.ProviderKimi},
		{store.ProviderGrok, store.ProviderOpenCode},
	} {
		above, _ := tt.Find(providerLabel(l, pair[0]))
		below, _ := tt.Find(providerLabel(l, pair[1]))
		// Find returns the glyph box, not the tab box, so a long label can
		// sit a few DIP off its column centre. A ragged row would be far
		// larger, which is what this catches.
		diff := (above.X + above.W/2) - (below.X + below.W/2)
		if diff > 8 || diff < -8 {
			t.Fatalf("column of %q and %q is off by %v DIP",
				providerLabel(l, pair[0]), providerLabel(l, pair[1]), diff)
		}
	}
}
