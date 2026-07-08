package web

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
)

// Job is one async orchestrator action. Lines are streamed to all SSE
// listeners; on completion the final error (if any) is published via a
// "done" frame.
type Job struct {
	ID      string
	jobCtx  context.Context
	mu      sync.Mutex
	cancel  context.CancelFunc
	err     error
	done    bool
	subs    map[chan jobEvent]struct{}
	subMu   sync.Mutex
	expires time.Time
}

type jobEvent struct {
	stream string
	line   string
	done   bool
	err    string
}

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
var getenv = func(key string) string {
	return ""
}

// start allocates a fresh Job with a unique ID and a cancellable context.
func (b *jobBus) start(label string) *Job {
	id := randHex(8)
	ctx, cancel := context.WithCancel(context.Background())
	j := &Job{
		ID:      id,
		jobCtx:  ctx,
		cancel:  cancel,
		subs:    map[chan jobEvent]struct{}{},
		expires: b.tickNow().Add(b.ttl),
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
	j.done = true
	errMsg := ""
	if j.err != nil {
		errMsg = j.err.Error()
	}
	j.mu.Unlock()
	j.broadcast(jobEvent{done: true, err: errMsg})
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
	j.broadcast(jobEvent{stream: stream, line: line})
}

func (j *Job) broadcast(ev jobEvent) {
	j.subMu.Lock()
	defer j.subMu.Unlock()
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
func (j *Job) subscribe() (<-chan jobEvent, func()) {
	ch := make(chan jobEvent, 64)
	j.subMu.Lock()
	j.subs[ch] = struct{}{}
	j.subMu.Unlock()
	cancel := func() {
		j.subMu.Lock()
		delete(j.subs, ch)
		j.subMu.Unlock()
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