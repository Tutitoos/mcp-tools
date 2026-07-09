package web

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
)

// Job is one async orchestrator action. Lines are streamed to all SSE
// listeners; on completion the final error (if any) is published via a
// "done" frame.
//
// Events published before a subscriber attaches are kept in `history` so
// late subscribers (e.g. a UI that opens the SSE stream after a POST
// returns) still see the full transcript. The history is bounded to
// avoid unbounded growth on long-running jobs.
type Job struct {
	ID         string
	Label      string
	StartedAt  time.Time
	FinishedAt time.Time // zero mientras corre; fijado en jobBus.finish
	jobCtx     context.Context
	mu         sync.Mutex // guards err, done, history, subs — see publish/subscribe.
	cancel     context.CancelFunc
	err        error
	done       bool
	history    []jobEvent
	subs       map[chan jobEvent]struct{}
	expires    time.Time
}

type jobEvent struct {
	stream string
	line   string
	done   bool
	err    string
}

// maxHistory caps the per-job event replay buffer. Long-running jobs
// (e.g. multi-step installs) are bounded so memory stays predictable.
const maxHistory = 512

// jobBus keeps every Job in memory; subscribers attach per job ID. The
// default retention is 5 minutes after completion (configurable via
// MCP_TOOLS_JOB_TTL).
type jobBus struct {
	mu      sync.Mutex
	jobs    map[string]*Job
	ttl     time.Duration
	tickNow func() time.Time
}

var jobs = newJobBus()

func newJobBus() *jobBus {
	return &jobBus{
		jobs:    map[string]*Job{},
		ttl:     parseJobTTL(),
		tickNow: time.Now,
	}
}

func parseJobTTL() time.Duration {
	if v := strings.TrimSpace(envOr("MCP_TOOLS_JOB_TTL", "")); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	return 5 * time.Minute
}

func envOr(key, fallback string) string {
	if v := strings.TrimSpace(getenv(key)); v != "" {
		return v
	}
	return fallback
}

// getenv is a tiny indirection so tests can override env.
var getenv = os.Getenv

// start allocates a fresh Job with a unique ID and a cancellable context.
func (b *jobBus) start(label string) *Job {
	id := randHex(8)
	ctx, cancel := context.WithCancel(context.Background())
	j := &Job{
		ID:        id,
		Label:     label,
		StartedAt: b.tickNow(),
		jobCtx:    ctx,
		cancel:    cancel,
		subs:      map[chan jobEvent]struct{}{},
		expires:   b.tickNow().Add(b.ttl),
	}
	b.mu.Lock()
	b.jobs[id] = j
	b.mu.Unlock()
	j.publish("system", "── "+label)
	return j
}

// finish marks the Job as done, broadcasts a final event, and schedules
// its removal from the bus.
func (b *jobBus) finish(id string) {
	b.mu.Lock()
	j, ok := b.jobs[id]
	if !ok {
		b.mu.Unlock()
		return
	}
	j.mu.Lock()
	j.FinishedAt = b.tickNow()
	j.done = true
	errMsg := ""
	if j.err != nil {
		errMsg = j.err.Error()
	}
	j.broadcastLocked(jobEvent{done: true, err: errMsg})
	j.mu.Unlock()
	go func() {
		<-time.After(b.ttl)
		b.mu.Lock()
		delete(b.jobs, id)
		b.mu.Unlock()
	}()
	b.mu.Unlock()
}

func randHex(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// safeGo runs fn under jobCtx with panic recovery. On panic, the recovery
// value is published as a stderr line, recorded as the job error, and the
// job is finished — so subscribers never see a silent hang.
func safeGo(job *Job, fn func(ctx context.Context) error) {
	go func() {
		defer jobs.finish(job.ID)
		defer func() {
			if r := recover(); r != nil {
				slog.Error("web: job panic", "err", r, "job", job.ID)
				job.publish("stderr", fmt.Sprintf("PANIC %v", r))
				job.setError(fmt.Errorf("panic: %v", r))
			}
		}()
		if err := fn(job.Ctx()); err != nil {
			job.publish("stdout", "ERROR "+err.Error())
			job.setError(err)
		}
	}()
}

// Ctx returns the Job's cancellable context. Each goroutine that runs an
// orchestrator action receives this so cancellation propagates.
func (j *Job) Ctx() context.Context {
	if j.jobCtx != nil {
		return j.jobCtx
	}
	return context.Background()
}

// setError records the final error if not already set.
func (j *Job) setError(err error) {
	j.mu.Lock()
	defer j.mu.Unlock()
	if j.err == nil {
		j.err = err
	}
}

// publish fans a (stream, line) frame out to every subscriber.
func (j *Job) publish(stream, line string) {
	ev := jobEvent{stream: stream, line: line}
	j.mu.Lock()
	if len(j.history) < maxHistory {
		j.history = append(j.history, ev)
	}
	j.broadcastLocked(ev)
	j.mu.Unlock()
}

// broadcastLocked fans ev out to every current subscriber. Caller MUST
// hold j.mu — publish and finish both append to history/set done and
// broadcast under the SAME lock acquisition (never two separate ones) so
// a subscribe() racing either of them can't see the event twice (once via
// history replay, once via a live send) or miss it entirely. See B10.
func (j *Job) broadcastLocked(ev jobEvent) {
	for ch := range j.subs {
		select {
		case ch <- ev:
		default:
			// drop on slow subscriber; they'll resync via reconnection
		}
	}
}

// subscribe attaches a channel that receives every event for this job.
// The caller MUST call the returned cancel func to detach.
//
// Snapshot + push + register happen under one unbroken j.mu hold, atomic
// with publish/finish's own critical section (append/done + broadcast
// under that same lock). Whichever side wins the lock runs to completion
// before the other starts, so an event is never skipped (the old code
// released the lock between snapshotting history and registering the
// channel, so a publish landing in that window was neither in the
// snapshot nor seen by broadcast — silently lost) or delivered twice.
func (j *Job) subscribe() (<-chan jobEvent, func()) {
	// Replay the buffered history to the new subscriber. If the job
	// already finished, include the synthetic done frame so the late
	// subscriber sees the final result (orchestrator errors live in
	// j.err — setError runs before finish). Buffered events are bounded
	// by maxHistory, so this is a copy of at most a few KB.
	j.mu.Lock()
	pending := append([]jobEvent(nil), j.history...)
	if j.done {
		errStr := ""
		if j.err != nil {
			errStr = j.err.Error()
		}
		pending = append(pending, jobEvent{done: true, err: errStr})
	}

	// Safety ceiling: an absurdly long history (shouldn't happen given
	// maxHistory=512, but the +1 synthetic done frame and future
	// changes could push past it) gets truncated to the tail, with a
	// marker so the subscriber knows lines were dropped.
	const safetyCeiling = 1024
	if len(pending) > safetyCeiling {
		dropped := len(pending) - 512
		marker := jobEvent{stream: "system", line: fmt.Sprintf("── history truncated (%d older lines dropped)", dropped)}
		pending = append([]jobEvent{marker}, pending[len(pending)-512:]...)
	}

	ch := make(chan jobEvent, 64+len(pending)+1)
	for _, ev := range pending {
		ch <- ev // never blocks: ch's capacity was sized for exactly this.
	}
	j.subs[ch] = struct{}{}
	j.mu.Unlock()

	cancel := func() {
		j.mu.Lock()
		delete(j.subs, ch)
		j.mu.Unlock()
	}
	return ch, cancel
}

// handleJobEvents is the SSE endpoint for /api/jobs/{jobID}/events.
func handleJobEvents(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "jobID")
	jobs.mu.Lock()
	j, ok := jobs.jobs[id]
	jobs.mu.Unlock()
	if !ok {
		writeError(w, http.StatusNotFound, "job not found or expired")
		return
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
	flusher.Flush()

	ch, cancel := j.subscribe()
	defer cancel()

	// Send a synthetic hello so the client can confirm the stream is open.
	fmt.Fprintf(w, "event: hello\ndata: {\"job_id\":\"%s\"}\n\n", id)
	flusher.Flush()

	for {
		select {
		case <-r.Context().Done():
			return
		case ev, ok := <-ch:
			if !ok {
				return
			}
			if ev.done {
				errStr := ""
				if ev.err != "" {
					errStr = `, "error":"` + jsonEscape(ev.err) + `"`
				}
				fmt.Fprintf(w, "event: done\ndata: {\"ok\":%t%s}\n\n", ev.err == "", errStr)
				flusher.Flush()
				return
			}
			fmt.Fprintf(w, "data: %s %s\n\n", ev.stream, ev.line)
			flusher.Flush()
		}
	}
}

// jsonEscape escapes a string for embedding in a JSON value (lightweight;
// we never accept arbitrary input here).
func jsonEscape(s string) string {
	var b strings.Builder
	for _, r := range s {
		switch r {
		case '"':
			b.WriteString(`\"`)
		case '\\':
			b.WriteString(`\\`)
		case '\n':
			b.WriteString(`\n`)
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}

// JobSummary is the JSON shape returned by GET /api/jobs.
type JobSummary struct {
	ID         string     `json:"id"`
	Label      string     `json:"label"`
	Status     string     `json:"status"` // "running" | "ok" | "error"
	StartedAt  time.Time  `json:"started_at"`
	FinishedAt *time.Time `json:"finished_at,omitempty"`
	Error      string     `json:"error,omitempty"`
	Lines      int        `json:"lines"` // len(history) snapshot
}

// snapshot returns the JobSummary for j. Caller MUST NOT hold j.mu.
func (j *Job) snapshot() JobSummary {
	j.mu.Lock()
	defer j.mu.Unlock()
	s := JobSummary{
		ID:        j.ID,
		Label:     j.Label,
		StartedAt: j.StartedAt,
		Lines:     len(j.history),
	}
	switch {
	case !j.done:
		s.Status = "running"
	case j.err != nil:
		s.Status = "error"
		s.Error = j.err.Error()
	default:
		s.Status = "ok"
	}
	if !j.FinishedAt.IsZero() {
		t := j.FinishedAt
		s.FinishedAt = &t
	}
	return s
}

// handleJobs is GET /api/jobs — snapshot of every job currently in the bus
// (live + finished-within-TTL), sorted by StartedAt descending. Cheap:
// bounded by TTL; each entry is a fixed-size struct.
func handleJobs(w http.ResponseWriter, _ *http.Request) {
	jobs.mu.Lock()
	js := make([]*Job, 0, len(jobs.jobs))
	for _, j := range jobs.jobs {
		js = append(js, j)
	}
	jobs.mu.Unlock()

	out := make([]JobSummary, 0, len(js))
	for _, j := range js {
		out = append(out, j.snapshot())
	}
	sort.Slice(out, func(i, k int) bool { return out[i].StartedAt.After(out[k].StartedAt) })
	writeJSON(w, http.StatusOK, out)
}

// handleJobCancel is POST /api/jobs/{jobID}/cancel — invokes the job's
// cancel func. Idempotent: cancelling a finished job is a no-op.
func handleJobCancel(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "jobID")
	jobs.mu.Lock()
	j, ok := jobs.jobs[id]
	jobs.mu.Unlock()
	if !ok {
		writeError(w, http.StatusNotFound, "job not found or expired")
		return
	}
	if j.cancel != nil {
		j.cancel()
	}
	writeJSON(w, http.StatusAccepted, map[string]any{"ok": true})
}
