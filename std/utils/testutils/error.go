package utils

import (
	"testing"

	"github.com/stretchr/testify/require"
)

var testT *testing.T

// Sets the global test instance for the package, enabling testing utilities to access the provided `testing.T` for logging and failure reporting during test execution.
func SetT(t *testing.T) {
	testT = t
}

// Asserts that an error is nil during testing, returning the corresponding value if no error occurred.
func NoErr[T any](v T, err error) T {
	require.NoError(testT, err)
	return v
}

// Asserts that the provided error is non-nil (using `require.Error`) and returns it, typically used in tests to verify error conditions while discarding an associated unused value of type T.
func Err[T any](_ T, err error) error {
	require.Error(testT, err)
	return err
}

// Panics if the provided error is non-nil, otherwise returns the given value.
func NoErrB[T any](v T, err error) T {
	if err != nil {
		panic(err)
	}
	return v
}

// Panics if the provided error is nil, otherwise returns it; the first argument of type T is ignored and used for context in testing scenarios.
func ErrB[T any](_ T, err error) error {
	if err == nil {
		panic("expected error")
	}
	return err
}
