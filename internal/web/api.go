package web

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/Tutitoos/mcp-tools/internal/config"
	"github.com/Tutitoos/mcp-tools/internal/orchestrator"
	"github.com/Tutitoos/mcp-tools/internal/state"
	"github.com/Tutitoos/mcp-tools/internal/tools"
)

// handleTools returns the full registry annotated with each tool's
// installation status and whether it's in the persisted selection.
func handleTools(w http.ResponseWriter, _ *http.Request) {
	st, err := state.Load()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	reg := tools.Registry()
	rows := make([]map[string]any, 0, len(reg))
	for _, t := range reg {
		row := map[string]any{
			"key":        t.Key,
			"label":      t.Label,
			"summary":    t.Summary,
			"deploy":     t.Deploy.String(),
			"default_on": t.DefaultOn,
			"deps":       t.Deps,
			"selected":   st.Has(t.Key),
		}
		if t.Status != nil {
			p, err := t.Status()
			if err == nil {
				row["installed"] = p.Installed
				row["version"] = p.Version
				if p.Extra != nil {
					row["extra"] = p.Extra
				}
			} else {
				row["installed"] = false
				row["extra"] = map[string]any{"error": err.Error()}
			}
		}
		rows = append(rows, row)
	}
	writeJSON(w, http.StatusOK, rows)
}

// handleStatus returns the snapshot used by the dashboard: persisted
// state, .env contents, .env.mem0 contents, docker ps, and a high-level
// docker-running flag.
func handleStatus(w http.ResponseWriter, _ *http.Request) {
	st, err := state.Load()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	env, _ := config.LoadEnv(config.EnvFile())
	mem0, _ := config.LoadEnv(config.EnvMem0File())
	services, dockerRunning := listComposeServices()
	resp := map[string]any{
		"state": map[string]any{
			"selected":   st.Selected,
			"versions":   st.Versions,
			"updated_at": st.UpdatedAt.Format(time.RFC3339),
		},
		"env":              env,
		"env_mem0":         mem0,
		"compose_services": services,
		"docker_running":   dockerRunning,
	}
	writeJSON(w, http.StatusOK, resp)
}

// handleToolAction returns a handler for /api/tools/{key}/{install|upgrade|uninstall}.
// Force is read from the JSON body for uninstall.
func handleToolAction(verb string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		key := chi.URLParam(r, "key")
		if _, err := tools.Get(key); err != nil {
			writeError(w, http.StatusNotFound, err.Error())
			return
		}
		var body struct {
			Force bool `json:"force"`
		}
		if r.Body != nil && r.ContentLength != 0 {
			_ = json.NewDecoder(r.Body).Decode(&body)
		}
		job := jobs.start(verb + " " + key)
		go func() {
			defer jobs.finish(job.ID)
			log := func(line string) { job.publish("stdout", line) }
		ctx := job.Ctx()
			var err error
			switch verb {
			case "install":
				_, err = orchestrator.InstallSingle(ctx, key, log)
			case "upgrade":
				err = orchestrator.UpgradeSingle(ctx, key, log)
			case "uninstall":
				err = orchestrator.Uninstall(ctx, key, body.Force, false, log)
			default:
				err = fmt.Errorf("unknown verb %q", verb)
			}
			if err != nil {
				log("ERROR " + err.Error())
				job.setError(err)
			}
		}()
		writeJSON(w, http.StatusAccepted, map[string]any{"ok": true, "job_id": job.ID})
	}
}

// handleConfigure applies a new selection; equivalent to `mcp-tools configure`
// driven from the multi-select checklist on /configure.
func handleConfigure(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Selected []string `json:"selected"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json: "+err.Error())
		return
	}
	prev, err := state.Load()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	job := jobs.start("configure")
	go func() {
		defer jobs.finish(job.ID)
			log := func(line string) { job.publish("stdout", line) }
		ctx := job.Ctx()
		if _, err := orchestrator.Configure(ctx, prev, false, log); err != nil {
			log("ERROR " + err.Error())
			job.setError(err)
		}
	}()
	writeJSON(w, http.StatusAccepted, map[string]any{"ok": true, "job_id": job.ID})
}

// handleEnv rewrites the .env file via config.UpdateEnv.
func handleEnv(w http.ResponseWriter, r *http.Request) {
	updateEnvHandler(config.EnvFile(), w, r)
}

// handleEnvMem0 rewrites .env.mem0.
func handleEnvMem0(w http.ResponseWriter, r *http.Request) {
	updateEnvHandler(config.EnvMem0File(), w, r)
}

func updateEnvHandler(path string, w http.ResponseWriter, r *http.Request) {
	var body struct {
		Values map[string]string `json:"values"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json: "+err.Error())
		return
	}
	if err := config.UpdateEnv(path, body.Values); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

// handleSelectModel updates MEM0_LLM_MODEL or MEM0_EMBED_MODEL in
// .env.mem0 and (optionally) triggers an `ollama pull` for the new tag.
func handleSelectModel(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Slot string `json:"slot"` // "llm" or "embed"
		Tag  string `json:"tag"`
		Pull bool   `json:"pull"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json: "+err.Error())
		return
	}
	if body.Tag == "" || (body.Slot != "llm" && body.Slot != "embed") {
		writeError(w, http.StatusBadRequest, "slot must be llm|embed and tag non-empty")
		return
	}
	key := "MEM0_LLM_MODEL"
	if body.Slot == "embed" {
		key = "MEM0_EMBED_MODEL"
	}
	if err := config.UpdateEnv(config.EnvMem0File(), map[string]string{key: body.Tag}); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if body.Pull {
		job := jobs.start("ollama pull " + body.Tag)
		go func() {
			defer jobs.finish(job.ID)
			streamOllamaPull(job, body.Tag)
		}()
		writeJSON(w, http.StatusAccepted, map[string]any{"ok": true, "job_id": job.ID})
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

// handleModels returns parsed `docker exec mcp-tools-ollama ollama list` rows.
func handleModels(w http.ResponseWriter, _ *http.Request) {
	rows, err := listOllamaModels()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, rows)
}

func handleModelPull(w http.ResponseWriter, r *http.Request) {
	enqueueOllamaAction(w, r, "pull")
}

func handleModelRm(w http.ResponseWriter, r *http.Request) {
	enqueueOllamaAction(w, r, "rm")
}

func enqueueOllamaAction(w http.ResponseWriter, r *http.Request, verb string) {
	var body struct {
		Tag string `json:"tag"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json: "+err.Error())
		return
	}
	if body.Tag == "" {
		writeError(w, http.StatusBadRequest, "tag required")
		return
	}
	job := jobs.start("ollama " + verb + " " + body.Tag)
	go func() {
		defer jobs.finish(job.ID)
		streamOllamaExec(job, verb, body.Tag)
	}()
	writeJSON(w, http.StatusAccepted, map[string]any{"ok": true, "job_id": job.ID})
}

// handleServices returns `docker compose ps --format json`.
func handleServices(w http.ResponseWriter, _ *http.Request) {
	svcs, _ := listComposeServices()
	writeJSON(w, http.StatusOK, svcs)
}

// handleServiceAction handles up/stop/restart for a compose service.
func handleServiceAction(verb string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		name := chi.URLParam(r, "name")
		if name == "" {
			writeError(w, http.StatusBadRequest, "service name required")
			return
		}
		job := jobs.start("docker compose " + verb + " " + name)
		go func() {
			defer jobs.finish(job.ID)
			streamComposeAction(job, verb, name)
		}()
		writeJSON(w, http.StatusAccepted, map[string]any{"ok": true, "job_id": job.ID})
	}
}

// handleLogsStream streams `docker logs --tail N -f` as SSE frames.
// query params: tail (int, default 200), follow (0|1, default 1).
func handleLogsStream(w http.ResponseWriter, r *http.Request) {
	svc := chi.URLParam(r, "service")
	if svc == "" {
		writeError(w, http.StatusBadRequest, "service required")
		return
	}
	tail := 200
	follow := "1"
	if t := r.URL.Query().Get("tail"); t != "" {
		fmt.Sscanf(t, "%d", &tail)
	}
	if f := r.URL.Query().Get("follow"); f != "" {
		follow = f
	}
	flusher, ok := w.(http.Flusher)
	if !ok {
		writeError(w, http.StatusInternalServerError, "streaming unsupported")
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.WriteHeader(http.StatusOK)
	args := []string{"logs", "--tail", fmt.Sprintf("%d", tail)}
	if follow != "0" {
		args = append(args, "-f")
	}
	args = append(args, svc)
	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()
	cmd := exec.CommandContext(ctx, "docker", args...)
	cmd.Env = os.Environ()
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		fmt.Fprintf(w, "data: docker pipe: %s\n\n", err.Error())
		flusher.Flush()
		return
	}
	stderr, _ := cmd.StderrPipe()
	if err := cmd.Start(); err != nil {
		fmt.Fprintf(w, "data: docker start: %s\n\n", err.Error())
		flusher.Flush()
		return
	}
	go func() {
		defer cancel()
		streamDockerPipe(w, flusher, stdout, "stdout")
	}()
	streamDockerPipe(w, flusher, stderr, "stderr")
	_ = cmd.Wait()
}

// handleSync dispatches /api/{skills|rules|mcp-config}/sync to the
// matching orchestrator helper.
func handleSync(kind string) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		job := jobs.start("sync " + kind)
		go func() {
			defer jobs.finish(job.ID)
			log := func(line string) { job.publish("stdout", line) }
			ctx := job.Ctx()
			var err error
			switch kind {
			case "skills":
				err = orchestrator.InstallSkills(ctx, false, log)
			case "rules":
				err = orchestrator.InstallRules(ctx, false, log)
			case "mcp-config":
				err = orchestrator.RefreshMcpConfig(ctx, false, log)
			default:
				err = errors.New("unknown sync kind " + kind)
			}
			if err != nil {
				log("ERROR " + err.Error())
				job.setError(err)
			}
		}()
		writeJSON(w, http.StatusAccepted, map[string]any{"ok": true, "job_id": job.ID})
	}
}

// listComposeServices invokes `docker compose ps --format json`. Each line
// of stdout is a JSON object; we decode each and return as a flat array.
func listComposeServices() ([]map[string]string, bool) {
	cmd := exec.Command("docker", "compose", "-f", "dockers/compose.yaml", "--env-file", ".env", "ps", "--format", "json")
	cmd.Env = os.Environ()
	if _, err := exec.LookPath("docker"); err != nil {
		return nil, false
	}
	out, err := cmd.Output()
	if err != nil {
		// Fallback: try with a shorter timeout via docker info to detect docker.
		if info, ierr := exec.Command("docker", "info").Output(); ierr == nil && len(info) > 0 {
			return nil, true
		}
		return nil, false
	}
	var out2 []map[string]string
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		var row map[string]string
		if err := json.Unmarshal([]byte(line), &row); err == nil {
			out2 = append(out2, row)
		}
	}
	return out2, true
}

// listOllamaModels parses `docker exec mcp-tools-ollama ollama list`.
func listOllamaModels() ([]map[string]string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "docker", "exec", "mcp-tools-ollama", "ollama", "list")
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("ollama list: %w", err)
	}
	var rows []map[string]string
	for _, line := range strings.Split(string(out), "\n") {
		if line == "" || strings.HasPrefix(line, "NAME") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 4 {
			continue
		}
		rows = append(rows, map[string]string{
			"tag":      fields[0],
			"size":     fields[2] + " " + fields[3],
			"modified": strings.Join(fields[4:], " "),
		})
	}
	return rows, nil
}

// streamDockerPipe reads lines from r and writes them as SSE `data:` frames
// to w, flushing after each. Used by handleLogsStream.
func streamDockerPipe(w http.ResponseWriter, flusher http.Flusher, r io.Reader, stream string) {
	buf := make([]byte, 4096)
	var carry []byte
	for {
		n, err := r.Read(buf)
		if n > 0 {
			carry = append(carry, buf[:n]...)
			for {
				idx := strings.Index(string(carry), "\n")
				if idx < 0 {
					break
				}
				line := string(carry[:idx])
				carry = carry[idx+1:]
				fmt.Fprintf(w, "data: %s %s\n\n", stream, line)
				flusher.Flush()
			}
		}
		if err != nil {
			if len(carry) > 0 {
				fmt.Fprintf(w, "data: %s %s\n\n", stream, string(carry))
				flusher.Flush()
			}
			return
		}
	}
}

// streamOllamaPull runs `docker exec mcp-tools-ollama ollama pull <tag>`
// and streams progress lines to the job.
func streamOllamaPull(job *Job, tag string) {
	ctx, cancel := context.WithTimeout(job.Ctx(), 30*time.Minute)
	defer cancel()
	cmd := exec.CommandContext(ctx, "docker", "exec", "mcp-tools-ollama", "ollama", "pull", tag)
	cmd.Env = os.Environ()
	stdout, _ := cmd.StdoutPipe()
	stderr, _ := cmd.StderrPipe()
	if err := cmd.Start(); err != nil {
		job.publish("stderr", "ollama pull start: "+err.Error())
		job.setError(err)
		return
	}
	streamProcessLines(job, stdout, "stdout")
	streamProcessLines(job, stderr, "stderr")
	if err := cmd.Wait(); err != nil {
		job.setError(err)
	}
}

// streamOllamaExec wraps `docker exec mcp-tools-ollama ollama <verb> <tag>`.
func streamOllamaExec(job *Job, verb, tag string) {
	ctx, cancel := context.WithTimeout(job.Ctx(), 5*time.Minute)
	defer cancel()
	cmd := exec.CommandContext(ctx, "docker", "exec", "mcp-tools-ollama", "ollama", verb, tag)
	cmd.Env = os.Environ()
	stdout, _ := cmd.StdoutPipe()
	stderr, _ := cmd.StderrPipe()
	if err := cmd.Start(); err != nil {
		job.publish("stderr", "ollama "+verb+" start: "+err.Error())
		job.setError(err)
		return
	}
	streamProcessLines(job, stdout, "stdout")
	streamProcessLines(job, stderr, "stderr")
	if err := cmd.Wait(); err != nil {
		job.setError(err)
	}
}

// streamComposeAction runs `docker compose <verb> <service>`.
func streamComposeAction(job *Job, verb, name string) {
	args := []string{"compose", "-f", "dockers/compose.yaml", "--env-file", ".env", verb}
	if verb == "restart" {
		args = append(args, name)
	} else {
		args = append(args, verb, name)
	}
	cmd := exec.Command("docker", args...)
	cmd.Env = os.Environ()
	stdout, _ := cmd.StdoutPipe()
	stderr, _ := cmd.StderrPipe()
	if err := cmd.Start(); err != nil {
		job.publish("stderr", "compose "+verb+" start: "+err.Error())
		job.setError(err)
		return
	}
	streamProcessLines(job, stdout, "stdout")
	streamProcessLines(job, stderr, "stderr")
	if err := cmd.Wait(); err != nil {
		job.setError(err)
	}
}

// streamProcessLines reads r line-by-line and publishes each line to the job.
func streamProcessLines(job *Job, r io.Reader, stream string) {
	if r == nil {
		return
	}
	buf := make([]byte, 4096)
	var carry []byte
	for {
		n, err := r.Read(buf)
		if n > 0 {
			carry = append(carry, buf[:n]...)
			for {
				idx := strings.Index(string(carry), "\n")
				if idx < 0 {
					break
				}
				line := string(carry[:idx])
				carry = carry[idx+1:]
				job.publish(stream, line)
			}
		}
		if err != nil {
			if len(carry) > 0 {
				job.publish(stream, string(carry))
			}
			return
		}
	}
}