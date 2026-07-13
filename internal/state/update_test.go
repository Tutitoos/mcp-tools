package state

import (
	"fmt"
	"sync"
	"testing"
)

// TestUpdateConcurrentMutationsNoLostUpdates reproduces F5 (AUDIT-2026-07-11):
// the bare Load → mutate → Save pattern lost updates under concurrency
// because saveMu only made the final write atomic. state.Update must make
// the whole read-modify-write cycle atomic: N concurrent single-key adds
// must all survive.
func TestUpdateConcurrentMutationsNoLostUpdates(t *testing.T) {
	withDataDir(t)
	seed := State{Selected: []string{"base"}}
	if err := seed.Save(); err != nil {
		t.Fatalf("seed save: %v", err)
	}

	const n = 50
	var wg sync.WaitGroup
	for i := range n {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			key := fmt.Sprintf("tool-%02d", i)
			if _, err := Update(func(s *State) error {
				s.Selected = append(s.Selected, key)
				return nil
			}); err != nil {
				t.Errorf("Update(%s): %v", key, err)
			}
		}(i)
	}
	wg.Wait()

	final, err := Load()
	if err != nil {
		t.Fatalf("final load: %v", err)
	}
	if len(final.Selected) != n+1 {
		t.Fatalf("lost updates: got %d selected keys, want %d", len(final.Selected), n+1)
	}
	for i := range n {
		key := fmt.Sprintf("tool-%02d", i)
		if !final.Has(key) {
			t.Errorf("missing %s after concurrent Update calls", key)
		}
	}
}
