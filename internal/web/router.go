package web

import (
	"embed"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
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
//	GET  /api/plugins             → workspace plugins + lockfile state
//	POST /api/plugins/{name}/{link|unlink|enable|disable} → enqueue omp plugin job
//	POST /api/services/{name}/up|stop|restart → enqueue compose action
//	GET  /api/logs/{service}      → SSE stream of `docker compose logs -f`
//	POST /api/skills/sync         → re-run RunSkills
//	POST /api/rules/sync          → re-run RunRules
//	POST /api/mcp-config/sync     → re-run RunMcpConfig
//	GET  /api/jobs/{id}/events    → SSE log + done frame
//	GET  /api/jobs                → snapshot list of jobs in the bus
//	POST /api/jobs/{id}/cancel    → cancel a running job
//
// Anything else under /api/* returns 404. Non-/api/* paths fall through
// to the SSR handler (or SPA fallback when SSR is unavailable).
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
	r.Get("/api/plugins", handlePlugins)
	r.Get("/api/jobs/{jobID}/events", handleJobEvents)
	r.Get("/api/jobs", handleJobs)

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
	r.Post("/api/plugins/{name}/link", handlePluginAction("link"))
	r.Post("/api/plugins/{name}/unlink", handlePluginAction("unlink"))
	r.Post("/api/plugins/{name}/enable", handlePluginAction("enable"))
	r.Post("/api/plugins/{name}/disable", handlePluginAction("disable"))
	r.Post("/api/jobs/{jobID}/cancel", handleJobCancel)
	r.Get("/api/logs/{service}", handleLogsStream)
	r.Post("/api/skills/sync", handleSync("skills"))
	r.Post("/api/rules/sync", handleSync("rules"))
	r.Post("/api/mcp-config/sync", handleSync("mcp-config"))

	// SPA + SSR fallback: anything not /api/* serves SSR (when the
	// engine is initialised) or the embedded SPA shell.
	r.NotFound(ssrHandler(SPAAssets, ssrEngineOrNil()))

	return r
}

func handleVersion(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{
		"version": version.Version,
		"commit":  version.Commit,
		"date":    version.Date,
	})
}

// ssrEngineOrNil returns the package-level SSR engine, or nil if it has
// not been initialised (e.g. the bundle wasn't built). Callers handle
// the nil case by serving the SPA fallback directly.
func ssrEngineOrNil() *ssrEngine {
	return ssr
}

// ssrHandler returns an http.Handler that serves SSR-rendered HTML for
// routes matched by the React Router, or the embedded SPA shell as a
// fallback. Asset paths (with an extension) are served from the embed
// without consulting the SSR engine.
func ssrHandler(assets embed.FS, ssr *ssrEngine) http.HandlerFunc {
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
		// Asset path (has extension)? serve from embed directly.
		if path.Ext(r.URL.Path) != "" {
			fileServer.ServeHTTP(w, r)
			return
		}
		// Try SSR. On non-match (route table returned Response/404)
		// or any error, fall back to the SPA shell so the client-side
		// router still boots.
		if ssr != nil {
			if html, err := ssr.render(r.URL.Path); err == nil && html != "" {
				// The SSR bundle returns just the <body> content
				// (Shell + matched route). Inject it into the
				// document template so the client only has to
				// hydrate the body, not the full <html>.
				w.Header().Set("Cache-Control", "no-cache, must-revalidate")
				w.Header().Set("Content-Type", "text/html; charset=utf-8")
				w.WriteHeader(http.StatusOK)
				_, _ = io.WriteString(w, injectBody(shellTemplate, html))
				return
			}
		}
		// Fallback: serve the SPA shell (no client-side router has loaded
		// any routes yet, but HydratedRouter will take over).
		w.Header().Set("Cache-Control", "no-cache, must-revalidate")
		r2 := r.Clone(r.Context())
		r2.URL.Path = "/"
		fileServer.ServeHTTP(w, r2)
	}
}

// InitSSR is the wiring point called once at startup (from the CLI's
// serve command). It extracts the embedded SSR bundle to a temp file
// so each request can spawn `node <bundle> <url>` cheaply. Returns an
// error when node is not in PATH or the bundle is missing; callers
// decide whether to abort or fall back to SPA-only mode.
func InitSSR(assets embed.FS) error {
	if err := ensureNode(); err != nil {
		return err
	}
	if err := ensureSSRBundleExists(assets, "build/server/index.js"); err != nil {
		return err
	}
	// The document template lives alongside the SPA shell. If it's
	// missing we still proceed: injectBody falls back to appending
	// before </body>, which keeps the route working even when the
	// template was generated without the marker.
	if err := loadShellTemplate(assets, "build/client/index.html"); err != nil {
		fmt.Fprintf(os.Stderr, "ssr: shell template missing (%v); SSR may render without <head>\n", err)
	}
	eng, err := newSSREngine(assets, "build/server/index.js")
	if err != nil {
		return err
	}
	ssr = eng
	return nil
}

// ShutdownSSR cleans up the temp dir holding the SSR bundle. Safe to
// call multiple times.
func ShutdownSSR() {
	if ssr != nil {
		_ = ssr.Close()
		ssr = nil
	}
}

// ssr is the package-level engine initialised by InitSSR.
var ssr *ssrEngine
