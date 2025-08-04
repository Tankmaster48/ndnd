package arc_test

import (
	"testing"

	"github.com/named-data/ndnd/std/types/arc"
	"github.com/stretchr/testify/require"
)

// Tests the ArcPool's reference counting and object reuse by verifying that objects are correctly initialized, modified, and reset upon retrieval and release, ensuring proper reference tracking and deterministic value restoration.
func TestArcPool(t *testing.T) {

	pool := arc.NewArcPool(
		func() *int { return new(int) },
		func(v *int) { *v = 42 })

	arc := pool.Get()
	ref := arc.Load()
	require.Equal(t, 42, *ref)

	*ref = 43
	arc.Inc()
	arc.Inc()
	require.Equal(t, int32(1), arc.Dec())

	arc2 := pool.Get()
	require.Equal(t, 42, *arc2.Load())
	require.False(t, ref == arc2.Load())

	require.Equal(t, int32(0), arc.Dec()) // release
	arc3 := pool.Get()
	require.Equal(t, 42, *arc3.Load())
	require.Equal(t, 42, *ref) // reused (not deteministic though)
	require.True(t, ref == arc3.Load())
}
