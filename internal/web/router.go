package web

import (
	"embed"
	"io/fs"
	"net/http"
	"path"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/Tutitoos/mcp-tools/internal/version"
)

// NewRouter builds the HTTP handler for the web admin panel. The router
// is intentionally minimal: every state-changing /api/* handler validates
// input, calls into the orchestrator, and enqueues a job so clients can
// stream the log via SSE.
//
// Routing tree:
//
//	GET  /api/version             → build metadata
//	GET  /api/tools               → registry snapshot + selected flags
//	GET  /api/status              → state + .env + docker compose ps
//	POST /api/tools/{key}/install → enqueue install job
//	POST /api/tools/{key}/upgrade → enqueue upgrade job
//	POST /api/tools/{key}/uninstall → enqueue uninstall job
//	POST /api/configure           → apply selection diff
//	POST /api/env                 → rewrite .env via config.UpdateEnv
//	POST /api/env-mem0            → rewrite .env.mem0
//	POST /api/select-model        → swap a model + (optional) pull
//	GET  /api/models              → parsed `ollama list`
//	POST /api/models/pull         → enqueue `ollama pull`
//	POST /api/models/rm           → enqueue `ollama rm`
//	GET  /api/services            → `docker compose ps --format json`
//	POST /api/services/{name}/up|stop|restart → enqueue compose action
//	GET  /api/logs/{service}      → SSE stream of `docker logs -f`
//	POST /api/skills/sync         → re-run RunSkills
//	POST /api/rules/sync          → re-run RunRules
//	POST /api/mcp-config/sync     → re-run RunMcpConfig
//	GET  /api/jobs/{id}/events    → SSE log + done frame
//
// Anything else under /api/* returns 404. Non-/api/* paths fall through
// to the embedded SPA (index.html for routes, asset file otherwise).
func NewRouter() http.Handler {
	r := chi.NewRouter()
	// Middleware stack: request logger + recoverer. The API is open
	// by design -- bind to 127.0.0.1 (or rely on firewall) to restrict
	// access. The bearer-token gate was removed: too much friction for
	// a self-hosted home tool.
	r.Use(requestLogger)
	r.Use(recoverer)

	// Public, unauthenticated health probe.
	r.Get("/api/version", handleVersion)

	// Read-only snapshots.
	r.Get("/api/tools", handleTools)
	r.Get("/api/status", handleStatus)
	r.Get("/api/services", handleServices)
	r.Get("/api/models", handleModels)
	r.Get("/api/jobs/{jobID}/events", handleJobEvents)

	// State-changing handlers.
	r.Post("/api/tools/{key}/install", handleToolAction("install"))
	r.Post("/api/tools/{key}/upgrade", handleToolAction("upgrade"))
	r.Post("/api/tools/{key}/uninstall", handleToolAction("uninstall"))
	r.Post("/api/configure", handleConfigure)
	r.Post("/api/env", handleEnv)
	r.Post("/api/env-mem0", handleEnvMem0)
	r.Post("/api/select-model", handleSelectModel)
	r.Post("/api/models/pull", handleModelPull)
	r.Post("/api/models/rm", handleModelRm)
	r.Post("/api/services/{name}/up", handleServiceAction("up"))
	r.Post("/api/services/{name}/stop", handleServiceAction("stop"))
	r.Post("/api/services/{name}/restart", handleServiceAction("restart"))
	r.Get("/api/logs/{service}", handleLogsStream)
	r.Post("/api/skills/sync", handleSync("skills"))
	r.Post("/api/rules/sync", handleSync("rules"))
	r.Post("/api/mcp-config/sync", handleSync("mcp-config"))

	// SPA fallback: anything not /api/* serves embedded assets or index.html.
	r.NotFound(spaHandler(SPAAssets))

	return r
}

func handleVersion(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{
		"version": version.Version,
		"commit":  version.Commit,
		"date":    version.Date,
	})
}

// spaHandler returns an http.Handler that serves files from the embedded
// SPA bundle. Requests for paths with an extension (assets) serve the
// matching file; everything else (client-side routes) gets index.html.
func spaHandler(assets embed.FS) http.HandlerFunc {
	sub, err := fs.Sub(assets, "build/client")
	if err != nil {
		return func(w http.ResponseWriter, _ *http.Request) {
			http.Error(w, "spa: embed sub failed: "+err.Error(), http.StatusInternalServerError)
		}
	}
	fileServer := http.FileServer(http.FS(sub))
	return func(w http.ResponseWriter, r *http.Request) {
		// Don't intercept /api/* — let upstream 404 handle it.
		if strings.HasPrefix(r.URL.Path, "/api/") {
			http.NotFound(w, r)
			return
		}
		// Asset path (has extension)? serve from embed; otherwise SPA fallback.
		if path.Ext(r.URL.Path) == "" {
			r2 := r.Clone(r.Context())
			r2.URL.Path = "/"
			fileServer.ServeHTTP(w, r2)
			return
		}
		fileServer.ServeHTTP(w, r)
	}
}