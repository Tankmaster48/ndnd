package optional_test

import (
	"testing"

	"github.com/named-data/ndnd/std/types/optional"
	"github.com/stretchr/testify/require"
)

// **Description:**  
Tests an optional value container's functionality for holding a value or being empty, including methods to check if a value is present, retrieve it with a presence flag, unwrap it (with panic if empty), or return a default value when empty.
func TestOptional(t *testing.T) {
	option := optional.Some[int](42)
	require.True(t, option.IsSet())
	val, ok := option.Get()
	require.Equal(t, 42, val)
	require.True(t, ok)
	require.Equal(t, 42, option.Unwrap())
	require.Equal(t, 42, option.GetOr(5))

	option = optional.None[int]()
	require.False(t, option.IsSet())
	val, ok = option.Get()
	require.Equal(t, 0, val)
	require.False(t, ok)
	require.Panics(t, func() { option.Unwrap() })
	require.Equal(t, 5, option.GetOr(5))

	option.Set(45)
	require.True(t, option.IsSet())
	val, ok = option.Get()
	require.Equal(t, 45, val)
	require.True(t, ok)
	require.Equal(t, 45, option.Unwrap())
	require.Equal(t, 45, option.GetOr(5))
}
