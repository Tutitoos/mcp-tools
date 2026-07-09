package web

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/Tutitoos/mcp-tools/internal/plugins"
)

// handlePlugins returns the workspace plugins list merged with lockfile state.
func handlePlugins(w http.ResponseWriter, _ *http.Request) {
	views, err := plugins.Scan()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if views == nil {
		views = []plugins.View{}
	}
	writeJSON(w, http.StatusOK, views)
}

// handlePluginAction returns a handler for /api/plugins/{name}/{link|unlink|enable|disable}.
// Every action shells `omp plugin <verb> <arg>` and streams stdout+stderr to the job's SSE.
// 404 if the workspace scan has no plugin with that name; 503 if `omp` isn't in PATH.
func handlePluginAction(verb string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		name := chi.URLParam(r, "name")
		view, err := findPluginByName(name)
		if err != nil {
			writeError(w, http.StatusNotFound, err.Error())
			return
		}
		ompBin, err := exec.LookPath("omp")
		if err != nil {
			writeError(w, http.StatusServiceUnavailable, "omp CLI no está en PATH")
			return
		}
		// Argument per verb: link → absolute plugin path; the rest → plugin name.
		var arg string
		switch verb {
		case "link":
			arg = view.Path
		case "unlink", "enable", "disable":
			arg = view.Name
		default:
			writeError(w, http.StatusBadRequest, "unknown verb "+verb)
			return
		}
		// `unlink` shells to `omp plugin uninstall` (that's what the CLI calls it).
		cliVerb := verb
		if verb == "unlink" {
			cliVerb = "uninstall"
		}
		job := jobs.start("plugin " + verb + " " + view.Name)
		safeGo(job, func(ctx context.Context) error {
			cctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
			defer cancel()
			cmd := exec.CommandContext(cctx, ompBin, "plugin", cliVerb, arg)
			cmd.Env = os.Environ()
			// safeGo already publishes "ERROR <msg>" to stdout and calls
			// job.setError() when this closure returns non-nil; no extra
			// publish() needed. See internal/web/job.go:152-155.
			return streamCmdToJob(job, cmd)
		})
		writeJSON(w, http.StatusAccepted, map[string]any{"ok": true, "job_id": job.ID})
	}
}

// findPluginByName is a shared 404-guard for the four action handlers.
func findPluginByName(name string) (plugins.View, error) {
	views, err := plugins.Scan()
	if err != nil {
		return plugins.View{}, err
	}
	for _, v := range views {
		if v.Name == name {
			return v, nil
		}
	}
	return plugins.View{}, fmt.Errorf("plugin %q not found in workspace", name)
}
