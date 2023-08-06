package rule

import (
	"sync"
	"testing"
	"time"
)

func TestRule_stateDisabled(t *testing.T) {
	state := NewRuleState(-1)
	e := state.getLast()
	if !e.At.IsZero() {
		t.Fatalf("expected entry to be zero")
	}

	state.Add(StateEntry{At: time.Now()})
	state.Add(StateEntry{At: time.Now()})
	state.Add(StateEntry{At: time.Now()})

	if len(state.getAll()) != 1 {
		// state should store at least one update at any circumstances
		t.Fatalf("expected for state to have %d entries; got %d",
			1, len(state.getAll()),
		)
	}
}
func TestRule_state(t *testing.T) {
	stateEntriesN := 20
	state := NewRuleState(stateEntriesN)
	e := state.getLast()
	if !e.At.IsZero() {
		t.Fatalf("expected entry to be zero")
	}

	now := time.Now()
	state.Add(StateEntry{At: now})

	e = state.getLast()
	if e.At != now {
		t.Fatalf("expected entry at %v to be equal to %v",
			e.At, now)
	}

	time.Sleep(time.Millisecond)
	now2 := time.Now()
	state.Add(StateEntry{At: now2})

	e = state.getLast()
	if e.At != now2 {
		t.Fatalf("expected entry at %v to be equal to %v",
			e.At, now2)
	}

	if len(state.getAll()) != 2 {
		t.Fatalf("expected for state to have 2 entries only; got %d",
			len(state.getAll()),
		)
	}

	var last time.Time
	for i := 0; i < stateEntriesN*2; i++ {
		last = time.Now()
		state.Add(StateEntry{At: last})
	}

	e = state.getLast()
	if e.At != last {
		t.Fatalf("expected entry at %v to be equal to %v",
			e.At, last)
	}

	if len(state.getAll()) != stateEntriesN {
		t.Fatalf("expected for state to have %d entries only; got %d",
			stateEntriesN, len(state.getAll()),
		)
	}
}

// TestRule_stateConcurrent supposed to test concurrent
// execution of state updates.
// Should be executed with -race flag
func TestRule_stateConcurrent(t *testing.T) {
	state := NewRuleState(20)

	const workers = 50
	const iterations = 100
	wg := sync.WaitGroup{}
	wg.Add(workers)
	for i := 0; i < workers; i++ {
		go func() {
			defer wg.Done()
			for i := 0; i < iterations; i++ {
				state.Add(StateEntry{At: time.Now()})
				state.getAll()
				state.getLast()
			}
		}()
	}
	wg.Wait()
}
