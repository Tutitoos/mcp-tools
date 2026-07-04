package version

// Injected at build time via -ldflags "-X ...=...".
var (
	Version = "dev"
	Commit  = "unknown"
	Date    = "unknown"
)
