package sync

import (
	"testing"

	"github.com/named-data/ndnd/std/encoding"
	tu "github.com/named-data/ndnd/std/utils/testutils"
	"github.com/stretchr/testify/require"
)

// This function tests a hierarchical name-based publisher-subscriber system, verifying subscription matching (including prefix and root subscriptions), publication routing, and unsubscription behavior for NDN-like name patterns.
func TestSimplePs(t *testing.T) {
	tu.SetT(t)

	ps := NewSimplePs[int]()

	// Test SubscribePublisher
	val1, val2, val3, val4 := 0, 0, 0, 0
	ps.Subscribe(tu.NoErr(encoding.NameFromStr("/a/b")), func(v int) { val1 = v })
	ps.Subscribe(tu.NoErr(encoding.NameFromStr("/a/b/c")), func(v int) { val2 = v })
	ps.Subscribe(tu.NoErr(encoding.NameFromStr("/d")), func(v int) { val3 = v })

	require.True(t, ps.HasSub(tu.NoErr(encoding.NameFromStr("/a/b"))))
	require.True(t, ps.HasSub(tu.NoErr(encoding.NameFromStr("/d/e/f"))))
	require.False(t, ps.HasSub(tu.NoErr(encoding.NameFromStr("/"))))
	require.False(t, ps.HasSub(tu.NoErr(encoding.NameFromStr("/e/f/g"))))

	// Broad subscription
	ps.Subscribe(tu.NoErr(encoding.NameFromStr("/")), func(v int) { val4 = v })
	require.True(t, ps.HasSub(tu.NoErr(encoding.NameFromStr("/"))))
	require.True(t, ps.HasSub(tu.NoErr(encoding.NameFromStr("/e/f/g"))))

	// Test Publish
	ps.Publish(tu.NoErr(encoding.NameFromStr("/a/b/hello")), 1)
	require.Equal(t, 1, val1)
	require.Equal(t, 0, val2)
	require.Equal(t, 0, val3)
	require.Equal(t, 1, val4)

	ps.Publish(tu.NoErr(encoding.NameFromStr("/a/b/c/hello")), 2)
	require.Equal(t, 2, val1)
	require.Equal(t, 2, val2)
	require.Equal(t, 0, val3)
	require.Equal(t, 2, val4)

	ps.Publish(tu.NoErr(encoding.NameFromStr("/d/hello")), 3)
	require.Equal(t, 2, val1)
	require.Equal(t, 2, val2)
	require.Equal(t, 3, val3)
	require.Equal(t, 3, val4)

	ps.Publish(tu.NoErr(encoding.NameFromStr("/hello")), 4)
	require.Equal(t, 2, val1)
	require.Equal(t, 2, val2)
	require.Equal(t, 3, val3)
	require.Equal(t, 4, val4)

	// Test UnsubscribePublisher
	ps.Unsubscribe(tu.NoErr(encoding.NameFromStr("/a/b")))

	ps.Publish(tu.NoErr(encoding.NameFromStr("/a/b/hello")), 5)
	require.Equal(t, 2, val1)
	require.Equal(t, 2, val2)
	require.Equal(t, 3, val3)
	require.Equal(t, 5, val4)

	ps.Publish(tu.NoErr(encoding.NameFromStr("/a/b/c/hello")), 6)
	require.Equal(t, 2, val1)
	require.Equal(t, 6, val2)
	require.Equal(t, 3, val3)
	require.Equal(t, 6, val4)
}
