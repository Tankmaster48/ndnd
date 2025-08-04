package sync

import (
	"errors"
	"fmt"

	enc "github.com/named-data/ndnd/std/encoding"
)

var ErrSnapshot = errors.New("snapshot error")

type ErrSync struct {
	Publisher enc.Name
	BootTime  uint64
	Err       error
}

// Returns a formatted error message for a sync error, including the publisher, boot time, and underlying error.
func (e *ErrSync) Error() string {
	return fmt.Sprintf("sync error [%s][%d]: %v", e.Publisher, e.BootTime, e.Err)
}

// Returns the underlying error wrapped by this ErrSync instance, enabling error unwrapping as part of Go's error handling pattern.
func (e *ErrSync) Unwrap() error {
	return e.Err
}
