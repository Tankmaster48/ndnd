package basic

import (
	"fmt"
	"sync"
	"time"
)

type dummyEvent struct {
	t time.Time
	f func()
}

type DummyTimer struct {
	now    time.Time
	events []dummyEvent
	// Lock is not an very important thing because:
	//   1. Basic engine itself is single-threaded
	//   2. This timer is for test only, and there is a low chance for race.
	lock sync.Mutex
}

// Constructs a new DummyTimer initialized to the Unix epoch time (1970-01-01T00:00:00Z) with an empty event schedule, typically used for deterministic timer testing.
func NewDummyTimer() *DummyTimer {
	now, err := time.Parse(time.RFC3339, "1970-01-01T00:00:00Z")
	if err != nil {
		return nil
	}
	return &DummyTimer{
		now:    now,
		events: make([]dummyEvent, 0),
	}
}

// Returns the current time stored in the DummyTimer instance.
func (tm *DummyTimer) Now() time.Time {
	return tm.now
}

// Advances the internal clock of the DummyTimer by the given duration and executes any scheduled events whose times are now in the past relative to the new time.
func (tm *DummyTimer) MoveForward(d time.Duration) {
	events := func() []dummyEvent {
		tm.lock.Lock()
		defer tm.lock.Unlock()
		tm.now = tm.now.Add(d)
		ret := make([]dummyEvent, len(tm.events))
		copy(ret, tm.events)
		return ret
	}()

	// Run events
	for i, e := range events {
		if e.f != nil {
			if e.t.Before(tm.now) {
				e.f()
				events[i].f = nil
			}
		}
	}

	func() {
		tm.lock.Lock()
		defer tm.lock.Unlock()
		tm.events = events
	}()
}

// Schedules a function to be executed after a specified duration and returns a cancellation function to cancel it before execution.
func (tm *DummyTimer) Schedule(d time.Duration, f func()) func() error {
	t := tm.now.Add(d)
	tm.lock.Lock()
	defer tm.lock.Unlock()

	idx := len(tm.events)
	for i := range tm.events {
		if tm.events[i].f == nil {
			idx = i
			break
		}
	}
	if idx == len(tm.events) {
		tm.events = append(tm.events, dummyEvent{
			t: t,
			f: f,
		})
	} else {
		tm.events[idx] = dummyEvent{
			t: t,
			f: f,
		}
	}

	return func() error {
		if t.Before(tm.now) {
			return nil // Already past
		}
		if idx < len(tm.events) && tm.events[idx].t.Equal(t) && tm.events[idx].f != nil {
			tm.lock.Lock()
			defer tm.lock.Unlock()
			tm.events[idx].f = nil
			return nil
		} else {
			return fmt.Errorf("event has already been canceled")
		}
	}
}

// Blocks the current goroutine for the specified duration using the DummyTimer's scheduling mechanism to signal completion via a channel.
func (tm *DummyTimer) Sleep(d time.Duration) {
	ch := make(chan struct{})
	tm.Schedule(d, func() {
		ch <- struct{}{}
		close(ch)
	})
	<-ch
}

// Returns a fixed 8-byte nonce value (`0x01` to `0x08`) for use with the DummyTimer in NDN operations, typically for testing or placeholder scenarios.
func (*DummyTimer) Nonce() []byte {
	return []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08}
}
