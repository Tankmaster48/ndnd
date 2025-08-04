package ndn

import (
	"errors"
	"fmt"
)

type ErrInvalidValue struct {
	Item  string
	Value any
}

// Returns an error message indicating an invalid value for a specific item, including the item name and invalid value in the formatted string.
func (e ErrInvalidValue) Error() string {
	return fmt.Sprintf("invalid value for %s: %v", e.Item, e.Value)
}

type ErrNotSupported struct {
	Item string
}

// Returns an error string indicating that the specified field (e.Item) is not supported, formatted as "not supported field: {Item}".
func (e ErrNotSupported) Error() string {
	return fmt.Sprintf("not supported field: %s", e.Item)
}

var ErrCancelled = errors.New("operation cancelled")
var ErrNetwork = errors.New("network error")
var ErrProtocol = errors.New("protocol error")
var ErrSecurity = errors.New("security error")

// ErrFailedToEncode is returned when encoding fails but the input arguments are valid.
var ErrFailedToEncode = errors.New("failed to encode an NDN packet")

// ErrWrongType is returned when the type of the packet to parse is not expected.
var ErrWrongType = errors.New("packet to parse is not of desired type")

// ErrMultipleHandlers is returned when multiple handlers are attached to the same prefix.
var ErrMultipleHandlers = errors.New("multiple handlers attached to the same prefix")

// ErrDeadlineExceed is returned when the deadline of the Interest passed.
var ErrDeadlineExceed = errors.New("interest deadline exceeded")

// ErrFaceDown is returned when the face is closed.
var ErrFaceDown = errors.New("face is down. Unable to send packet")

// ErrNoPubKey is returned when the public key does not exist.
var ErrNoPubKey = errors.New("public key does not exist")
