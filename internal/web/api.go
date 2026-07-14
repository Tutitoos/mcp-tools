package web

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/Tutitoos/mcp-tools/internal/config"
	"github.com/Tutitoos/mcp-tools/internal/docker"
	"github.com/Tutitoos/mcp-tools/internal/orchestrator"
	"github.com/Tutitoos/mcp-tools/internal/state"
	"github.com/Tutitoos/mcp-tools/internal/tools"
)

// svcNameRe restricts docker-compose service keys accepted by
// handleLogsStream to safe characters — rejects shell metacharacters,
// spaces, and path traversal before the value reaches exec.Command.
var svcNameRe = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)

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
		deps := t.Deps
		if deps == nil {
			deps = []string{}
		}
		row := map[string]any{
			"key":        t.Key,
			"label":      t.Label,
			"summary":    t.Summary,
			"deploy":     t.Deploy.String(),
			"default_on": t.DefaultOn,
			"deps":       deps,
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
	env := redactEnv(loadEnvOrEmpty(config.EnvFile()))
	mem0 := redactEnv(loadEnvOrEmpty(config.EnvMem0File()))
	services, dockerRunning := listComposeServicesCached()
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

// secretKeyRe matches env keys whose values must never leave the host via
// the API: API keys, tokens, secrets, passwords. Neither generated env file
// contains such a key today, but /api/status dumps both files verbatim, so
// this is the guard that keeps a future TOKEN/KEY addition from leaking.
var secretKeyRe = regexp.MustCompile(`(?i)(KEY|TOKEN|SECRET|PASS)`)

func loadEnvOrEmpty(path string) map[string]string {
	m, _ := config.LoadEnv(path)
	return m
}

// redactEnv returns env with secret-shaped values masked. The key stays
// visible (the settings UI lists it); only the value is replaced.
func redactEnv(env map[string]string) map[string]string {
	for k, v := range env {
		if v != "" && secretKeyRe.MatchString(k) {
			env[k] = "••••••••"
		}
	}
	return env
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
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				writeError(w, http.StatusBadRequest, "invalid json: "+err.Error())
				return
			}
		}
		job := jobs.start(verb + " " + key)
		log := func(line string) { job.publish("stdout", line) }
		safeGo(job, func(ctx context.Context) error {
			switch verb {
			case "install":
				_, err := orchestrator.InstallSingle(ctx, key, log)
				return err
			case "upgrade":
				return orchestrator.UpgradeSingle(ctx, key, log)
			case "uninstall":
				return orchestrator.Uninstall(ctx, key, body.Force, false, log)
			default:
				return fmt.Errorf("unknown verb %q", verb)
			}
		})
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
	log := func(line string) { job.publish("stdout", line) }
	safeGo(job, func(ctx context.Context) error {
		_, err := orchestrator.Configure(ctx, prev, body.Selected, false, log)
		return err
	})
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
	for k, v := range body.Values {
		if strings.ContainsAny(v, "\n\r") {
			slog.Warn("web: stripped embedded newline from env value", "key", k)
			body.Values[k] = strings.NewReplacer("\n", "", "\r", "").Replace(v)
		}
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
		safeGo(job, func(_ context.Context) error {
			streamOllamaPull(job, body.Tag)
			return nil
		})
		writeJSON(w, http.StatusAccepted, map[string]any{"ok": true, "job_id": job.ID})
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

// handleModels returns parsed `docker exec mcp-tools-ollama ollama list` rows.
// Returns [] when the ollama container is down so the SPA can render its
// empty state instead of crashing on .map. Failure is logged, not surfaced
// in the response body.
func handleModels(w http.ResponseWriter, _ *http.Request) {
	rows, err := listOllamaModels()
	if err != nil {
		slog.Warn("web: ollama list failed", "err", err)
		writeJSON(w, http.StatusOK, []map[string]string{})
		return
	}
	if rows == nil {
		rows = []map[string]string{}
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
	safeGo(job, func(_ context.Context) error {
		streamOllamaExec(job, verb, body.Tag)
		return nil
	})
	writeJSON(w, http.StatusAccepted, map[string]any{"ok": true, "job_id": job.ID})
}

// handleServices returns `docker compose ps --format json`.
func handleServices(w http.ResponseWriter, _ *http.Request) {
	svcs, _ := listComposeServicesCached()
	if svcs == nil {
		svcs = []map[string]string{}
	}
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
		safeGo(job, func(_ context.Context) error {
			streamComposeAction(job, verb, name)
			return nil
		})
		writeJSON(w, http.StatusAccepted, map[string]any{"ok": true, "job_id": job.ID})
	}
}

// handleLogsStream streams `docker compose logs --no-log-prefix --no-color
// --tail N [-f] <svc>` as SSE frames.
// query params: tail (int, default 200), follow (0|1, default 1).
func handleLogsStream(w http.ResponseWriter, r *http.Request) {
	svc := chi.URLParam(r, "service")
	if svc == "" {
		writeError(w, http.StatusBadRequest, "service required")
		return
	}
	if !svcNameRe.MatchString(svc) {
		writeError(w, http.StatusBadRequest, "invalid service name")
		return
	}
	tail := 200
	follow := "1"
	if t := r.URL.Query().Get("tail"); t != "" {
		n, err := strconv.Atoi(t)
		if err != nil || n < 10 || n > 5000 {
			writeError(w, http.StatusBadRequest, "tail must be int 10..5000")
			return
		}
		tail = n
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
	args := []string{"logs", "--no-log-prefix", "--no-color", "--tail", strconv.Itoa(tail)}
	if follow != "0" {
		args = append(args, "-f")
	}
	args = append(args, svc)
	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()
	cmd := docker.ComposeCmdContext(ctx, []string{"dockers/compose.yaml"}, args...)
	if err := streamCmdSSE(ctx, w, flusher, cmd); err != nil {
		fmt.Fprintf(w, "data: docker: %s\n\n", err.Error())
		flusher.Flush()
	}
}

// handleSync dispatches /api/{skills|rules|mcp-config}/sync to the
// matching orchestrator helper.
func handleSync(kind string) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		job := jobs.start("sync " + kind)
		log := func(line string) { job.publish("stdout", line) }
		safeGo(job, func(ctx context.Context) error {
			switch kind {
			case "skills":
				return orchestrator.InstallSkills(ctx, false, log)
			case "rules":
				return orchestrator.InstallRules(ctx, false, log)
			case "mcp-config":
				return orchestrator.RefreshMcpConfig(ctx, false, log)
			default:
				return errors.New("unknown sync kind " + kind)
			}
		})
		writeJSON(w, http.StatusAccepted, map[string]any{"ok": true, "job_id": job.ID})
	}
}

// composeCache memoises listComposeServices for a short window. The
// dashboard shell polls /api/status every 5s and the services/logs views
// poll /api/services on the same cadence; without the cache each poll
// spawns a `docker compose ps` process (~300ms). 2.5s TTL keeps the UI
// honest (state changes surface within one poll) while collapsing
// concurrent pollers and multiple tabs into one docker invocation.
var composeCache struct {
	mu   sync.Mutex
	at   time.Time
	rows []map[string]string
	ok   bool
}

// listComposeServicesCached is the handler-facing entry point; it serves
// from composeCache when fresh and delegates to listComposeServices
// otherwise. The lock is held across the docker call on purpose: a cold
// cache with N concurrent pollers should run docker once, not N times.
func listComposeServicesCached() ([]map[string]string, bool) {
	composeCache.mu.Lock()
	defer composeCache.mu.Unlock()
	if time.Since(composeCache.at) < 2500*time.Millisecond {
		return composeCache.rows, composeCache.ok
	}
	rows, ok := listComposeServices()
	composeCache.at = time.Now()
	composeCache.rows, composeCache.ok = rows, ok
	return rows, ok
}

// listComposeServices invokes `docker compose ps --format json` and reduces
// each row to {name, state} — the shape web/app/lib/api.ts's ServiceView
// expects. `docker compose ps` output has non-string fields (ExitCode is a
// number), so decoding straight into map[string]string always failed
// silently here — every row was dropped and this endpoint returned `[]`
// unconditionally, regardless of what docker was actually running.
// Discovered while fixing B7 (verifying its "non-empty array" smoke test)
// — not one of the original B1-B15 findings, but in the same function B7
// already required editing and it blocked outright verifying B7.
func listComposeServices() ([]map[string]string, bool) {
	if _, err := exec.LookPath("docker"); err != nil {
		return nil, false
	}
	out, err := docker.Output("ps", "--format", "json")
	if err != nil {
		// Fallback: bounded `docker info` probe to distinguish "daemon up,
		// no compose stack" from "daemon down/hung".
		if info, ierr := docker.RunCmdWithTimeout(5*time.Second, "info").Output(); ierr == nil && len(info) > 0 {
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
		var row struct {
			Service string `json:"Service"` // compose.yaml service key — what `docker compose <verb> <name>` expects
			State   string `json:"State"`
		}
		if err := json.Unmarshal([]byte(line), &row); err != nil {
			continue
		}
		out2 = append(out2, map[string]string{"name": row.Service, "state": row.State})
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

// streamCmdSSE starts cmd and streams its stdout+stderr concurrently as SSE
// "data: <stream> <line>\n\n" frames to w. Both pipes are read by their own
// goroutine — reading them concurrently rather than draining stdout to EOF
// before touching stderr avoids a child stalling on a full stderr pipe
// buffer (see streamCmdToJob) — and every frame is serialised by writeMu,
// since w/flusher aren't safe for concurrent use (B8: the previous code had
// one goroutine writing stdout while the caller wrote stderr, with no lock
// between them — a guaranteed data race under -race).
func streamCmdSSE(ctx context.Context, w http.ResponseWriter, flusher http.Flusher, cmd *exec.Cmd) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("stdout pipe: %w", err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("stderr pipe: %w", err)
	}
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start: %w", err)
	}
	var writeMu sync.Mutex
	var wg sync.WaitGroup
	wg.Add(2)
	go func() { defer wg.Done(); streamPipeSSE(w, flusher, &writeMu, stdout, "stdout") }()
	go func() { defer wg.Done(); streamPipeSSE(w, flusher, &writeMu, stderr, "stderr") }()
	wg.Wait()
	return cmd.Wait()
}

// streamPipeSSE reads lines from r and writes each as an SSE `data:` frame
// to w, serialised by mu. Used by streamCmdSSE.
func streamPipeSSE(w http.ResponseWriter, flusher http.Flusher, mu *sync.Mutex, r io.Reader, stream string) {
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
				mu.Lock()
				fmt.Fprintf(w, "data: %s %s\n\n", stream, line)
				flusher.Flush()
				mu.Unlock()
			}
		}
		if err != nil {
			if len(carry) > 0 {
				mu.Lock()
				fmt.Fprintf(w, "data: %s %s\n\n", stream, string(carry))
				flusher.Flush()
				mu.Unlock()
			}
			return
		}
	}
}

// streamCmdToJob starts cmd and streams its stdout+stderr concurrently to
// job.publish (already goroutine-safe — see Job.publish). Reading both
// pipes concurrently, rather than draining stdout to EOF before touching
// stderr, avoids the classic pipe deadlock: a child that fills its stderr
// pipe buffer while only stdout is being read blocks forever on the write,
// so stdout never reaches EOF either (B9).
func streamCmdToJob(job *Job, cmd *exec.Cmd) error {
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return err
	}
	if err := cmd.Start(); err != nil {
		return err
	}
	var wg sync.WaitGroup
	wg.Add(2)
	go func() { defer wg.Done(); streamProcessLines(job, stdout, "stdout") }()
	go func() { defer wg.Done(); streamProcessLines(job, stderr, "stderr") }()
	wg.Wait()
	return cmd.Wait()
}

// streamOllamaPull runs `docker exec mcp-tools-ollama ollama pull <tag>`
// and streams progress lines to the job.
func streamOllamaPull(job *Job, tag string) {
	ctx, cancel := context.WithTimeout(job.Ctx(), 30*time.Minute)
	defer cancel()
	cmd := exec.CommandContext(ctx, "docker", "exec", "mcp-tools-ollama", "ollama", "pull", tag)
	cmd.Env = os.Environ()
	if err := streamCmdToJob(job, cmd); err != nil {
		job.publish("stderr", "ollama pull: "+err.Error())
		job.setError(err)
	}
}

// streamOllamaExec wraps `docker exec mcp-tools-ollama ollama <verb> <tag>`.
func streamOllamaExec(job *Job, verb, tag string) {
	ctx, cancel := context.WithTimeout(job.Ctx(), 5*time.Minute)
	defer cancel()
	cmd := exec.CommandContext(ctx, "docker", "exec", "mcp-tools-ollama", "ollama", verb, tag)
	cmd.Env = os.Environ()
	if err := streamCmdToJob(job, cmd); err != nil {
		job.publish("stderr", "ollama "+verb+": "+err.Error())
		job.setError(err)
	}
}

// streamComposeAction runs `docker compose <verb> <service>`.
func streamComposeAction(job *Job, verb, name string) {
	cmd := docker.ComposeWithFiles([]string{"dockers/compose.yaml"}, verb, name)
	cmd.Stdout, cmd.Stderr = nil, nil
	if err := streamCmdToJob(job, cmd); err != nil {
		job.publish("stderr", "compose "+verb+": "+err.Error())
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
