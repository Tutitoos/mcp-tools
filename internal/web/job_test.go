package web

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

// TestJobSubscribeReplaysHistoryBeforeLive reproduces B10: a subscriber must
// see every historical event, in order, before any live event — never a
// live event spliced ahead of history, and never a duplicate or drop at the
// history/live boundary. A concurrent publish is fired right as subscribe()
// runs to try to hit that boundary either way (live-0 lands in the replayed
// snapshot or arrives right after — both are correct outcomes, and either
// way the assertions below hold).
func TestJobSubscribeReplaysHistoryBeforeLive(t *testing.T) {
	j := &Job{subs: map[chan jobEvent]struct{}{}}
	for i := range 20 {
		j.publish("stdout", fmt.Sprintf("hist-%d", i))
	}

	go j.publish("stdout", "live-0")

	ch, cancel := j.subscribe()
	defer cancel()

	var got []string
	timeout := time.After(2 * time.Second)
	for len(got) < 21 {
		select {
		case ev := <-ch:
			got = append(got, ev.line)
		case <-timeout:
			t.Fatalf("timed out after %d/21 events: %v", len(got), got)
		}
	}

	for i := range 20 {
		want := fmt.Sprintf("hist-%d", i)
		if got[i] != want {
			t.Fatalf("event %d = %q, want %q (history out of order or corrupted): %v", i, got[i], want, got)
		}
	}
	if got[20] != "live-0" {
		t.Fatalf("event 20 = %q, want %q (live event spliced before end of history, or duplicated): %v", got[20], "live-0", got)
	}
}

// TestJobSubscribeNoDuplicateAtRaceBoundary stresses the exact window the
// naive B10 fix (snapshot history under j.mu, release it, then re-lock a
// SEPARATE j.subMu to register the channel) still gets wrong: a publish
// landing in that gap is neither in the snapshot (already taken) nor seen
// by broadcast (channel not registered yet) — permanently lost — or, if the
// snapshot and the broadcast subMu-window straddle differently, delivered
// twice. This repo's fix uses one lock across the whole
// snapshot+push+register critical section (see subscribe/publish in
// job.go) specifically to rule both out. Assert: whatever a subscriber
// racing an active publisher receives is strictly increasing — no gaps
// mid-sequence, no repeats.
func TestJobSubscribeNoDuplicateAtRaceBoundary(t *testing.T) {
	for iter := range 50 {
		j := &Job{subs: map[chan jobEvent]struct{}{}}
		const n = 30
		var wg sync.WaitGroup
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := range n {
				j.publish("stdout", fmt.Sprintf("%d", i))
			}
		}()

		time.Sleep(time.Duration(iter%5) * time.Microsecond) // hit varying points in the race window
		ch, cancel := j.subscribe()

		var got []int
		timeout := time.After(500 * time.Millisecond)
	drain:
		for {
			select {
			case ev := <-ch:
				var v int
				fmt.Sscanf(ev.line, "%d", &v)
				got = append(got, v)
				if v == n-1 {
					break drain
				}
			case <-timeout:
				break drain
			}
		}
		cancel()
		wg.Wait()

		for i := 1; i < len(got); i++ {
			if got[i] <= got[i-1] {
				t.Fatalf("iter %d: out-of-order/duplicate event at index %d: %v", iter, i, got)
			}
		}
	}
}

// resetJobBus clears the global job bus so tests don't see jobs left over
// from other tests in this package.
func resetJobBus(t *testing.T) {
	t.Cleanup(func() {
		jobs.mu.Lock()
		jobs.jobs = map[string]*Job{}
		jobs.mu.Unlock()
	})
}

// TestJobBusSnapshotIncludesLabelAndStatus confirms a running job's
// snapshot carries its label, a "running" status, at least one history
// line, a non-zero StartedAt, and no FinishedAt yet.
func TestJobBusSnapshotIncludesLabelAndStatus(t *testing.T) {
	resetJobBus(t)
	foo := jobs.start("foo")
	bar := jobs.start("bar")
	foo.publish("stdout", "hello")
	bar.publish("stdout", "world")

	for _, tc := range []struct {
		job   *Job
		label string
	}{{foo, "foo"}, {bar, "bar"}} {
		s := tc.job.snapshot()
		if s.Status != "running" {
			t.Errorf("job %s status = %q, want running", tc.label, s.Status)
		}
		if s.Label != tc.label {
			t.Errorf("job %s label = %q, want %q", tc.label, s.Label, tc.label)
		}
		if s.Lines < 1 {
			t.Errorf("job %s lines = %d, want >= 1", tc.label, s.Lines)
		}
		if s.StartedAt.IsZero() {
			t.Errorf("job %s StartedAt is zero", tc.label)
		}
		if s.FinishedAt != nil {
			t.Errorf("job %s FinishedAt = %v, want nil", tc.label, s.FinishedAt)
		}
	}
}

// TestJobBusSnapshotAfterFinish confirms a finished job's snapshot reports
// "error" status, the recorded error message, and a FinishedAt after
// StartedAt.
func TestJobBusSnapshotAfterFinish(t *testing.T) {
	resetJobBus(t)
	j := jobs.start("boom-job")
	time.Sleep(time.Millisecond)
	j.setError(errors.New("boom"))
	jobs.finish(j.ID)

	s := j.snapshot()
	if s.Status != "error" {
		t.Fatalf("status = %q, want error", s.Status)
	}
	if s.Error != "boom" {
		t.Fatalf("error = %q, want boom", s.Error)
	}
	if s.FinishedAt == nil {
		t.Fatalf("FinishedAt is nil, want set")
	}
	if !s.FinishedAt.After(s.StartedAt) {
		t.Fatalf("FinishedAt %v not after StartedAt %v", s.FinishedAt, s.StartedAt)
	}
}

// TestAPIJobsEndpointSortedDesc confirms GET /api/jobs returns every job
// in the bus, most-recently-started first.
func TestAPIJobsEndpointSortedDesc(t *testing.T) {
	resetJobBus(t)
	var ids []string
	for _, label := range []string{"first", "second", "third"} {
		j := jobs.start(label)
		ids = append(ids, j.ID)
		time.Sleep(2 * time.Millisecond)
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/jobs", nil)
	req.RemoteAddr = "127.0.0.1:1234"
	NewRouter().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", rec.Code, rec.Body.String())
	}
	var out []JobSummary
	if err := json.NewDecoder(rec.Body).Decode(&out); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(out) != 3 {
		t.Fatalf("len(out) = %d, want 3; body=%s", len(out), rec.Body.String())
	}
	for i := 1; i < len(out); i++ {
		if out[i-1].StartedAt.Before(out[i].StartedAt) {
			t.Fatalf("not sorted desc: entry %d (%v) before entry %d (%v)", i-1, out[i-1].StartedAt, i, out[i].StartedAt)
		}
	}
	if out[0].ID != ids[2] {
		t.Fatalf("out[0].ID = %s, want %s (last-started job first)", out[0].ID, ids[2])
	}
}

// TestAPIJobCancelUnknown confirms cancelling an unknown/expired job ID
// returns 404.
func TestAPIJobCancelUnknown(t *testing.T) {
	resetJobBus(t)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/jobs/does-not-exist/cancel", nil)
	req.RemoteAddr = "127.0.0.1:1234"
	NewRouter().ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404; body=%s", rec.Code, rec.Body.String())
	}
}

// TestAPIJobCancelRunning confirms cancelling a running job (one blocked
// on <-ctx.Done()) makes it terminate — status ends up "ok" or "error";
// either proves the bus observed completion rather than hanging forever.
func TestAPIJobCancelRunning(t *testing.T) {
	resetJobBus(t)
	job := jobs.start("cancel-me")
	safeGo(job, func(ctx context.Context) error {
		<-ctx.Done()
		return ctx.Err()
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/jobs/"+job.ID+"/cancel", nil)
	req.RemoteAddr = "127.0.0.1:1234"
	NewRouter().ServeHTTP(rec, req)
	if rec.Code != http.StatusAccepted {
		t.Fatalf("status = %d, want 202; body=%s", rec.Code, rec.Body.String())
	}

	deadline := time.Now().Add(200 * time.Millisecond)
	s := job.snapshot()
	for s.Status == "running" && time.Now().Before(deadline) {
		time.Sleep(5 * time.Millisecond)
		s = job.snapshot()
	}
	if s.Status == "running" {
		t.Fatalf("job still running 200ms after cancel; status=%q", s.Status)
	}
}
