package cli

import (
	"github.com/Tutitoos/mcp-tools/internal/systemd"
)

// modeValue is a tiny alias used by the legacy CLI subcommands (stop /
// restart / open / status-web) when they delegate to the systemd
// helpers. It exists so we don't re-implement the parseModeOverride
// switch in every file.
type modeValue = systemd.Mode

// detectMode is the same as systemd.DetectMode but takes a free-form
// override string ("user" / "system" / "" / anything else) and returns
// the parsed systemd.Mode.
func detectMode(override string) (modeValue, error) {
	return systemd.DetectMode(parseModeOverride(override))
}

// parseModeOverride maps a CLI flag value (user / system / "") to a
// systemd.Mode. Empty / unknown → Mode("") which systemd.DetectMode
// treats as "auto" (probe).
func parseModeOverride(s string) systemd.Mode {
	switch s {
	case "user":
		return systemd.ModeUser
	case "system":
		return systemd.ModeSystem
	}
	return systemd.Mode("")
}
