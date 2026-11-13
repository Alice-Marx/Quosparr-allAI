package opencode

import (
	"os"
	"path/filepath"
)

// IntegrationID is the Console integration the quota endpoints belong to.
const IntegrationID = "opencode"

const (
	// DatabaseName is the SQLite file that holds credentials and sessions.
	DatabaseName = "opencode.db"
	// AuthName is the legacy credential file, still readable for older CLIs.
	AuthName = "auth.json"
)

// DataDir is the CLI data directory: $XDG_DATA_HOME/opencode, else
// ~/.local/share/opencode. The CLI uses that layout on every platform it
// supports, Windows and macOS included.
func DataDir(home string) string {
	if dir := os.Getenv("XDG_DATA_HOME"); dir != "" {
		return filepath.Join(dir, "opencode")
	}
	return filepath.Join(home, ".local", "share", "opencode")
}

// DatabasePath is the CLI database of home.
func DatabasePath(home string) string {
	return filepath.Join(DataDir(home), DatabaseName)
}

// authPaths are the legacy credential files, newest layout first.
func authPaths(home string) []string {
	if dir := os.Getenv("XDG_DATA_HOME"); dir != "" {
		return []string{filepath.Join(dir, "opencode", AuthName)}
	}
	return []string{
		filepath.Join(home, ".local", "share", "opencode", AuthName),
		filepath.Join(home, ".opencode", AuthName),
	}
}
