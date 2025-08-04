package basic

import (
	"crypto/rand"
	"fmt"
	"time"

	"github.com/named-data/ndnd/std/ndn"
)

type Timer struct{}

// Constructs a new Timer with default configuration.
func NewTimer() ndn.Timer {
	return Timer{}
}

// Sleeps for the given duration.
func (Timer) Sleep(d time.Duration) {
	time.Sleep(d)
}

// Schedules a function to execute after a specified duration and returns a cancellation function that stops the timer if it's still pending, or returns an error if the event has already been canceled or fired.
func (Timer) Schedule(d time.Duration, f func()) func() error {
	t := time.AfterFunc(d, f)
	return func() error {
		if t != nil {
			t.Stop()
			t = nil
			return nil
		} else {
			return fmt.Errorf("event has already been canceled")
		}
	}
}

// Returns the current time as a `time.Time` value.
func (Timer) Now() time.Time {
	return time.Now()
}

// Generates a random 8-byte nonce using the system's cryptographically secure random number generator.
func (Timer) Nonce() []byte {
	// After go1.20 rand.Seed does not need to be called manually.
	buf := make([]byte, 8)
	n, _ := rand.Read(buf) // Should always succeed
	return buf[:n]
}
