package gen_map_test

import (
	"testing"

	enc "github.com/named-data/ndnd/std/encoding"
	def "github.com/named-data/ndnd/std/encoding/tests/gen_map"
	tu "github.com/named-data/ndnd/std/utils/testutils"
	"github.com/stretchr/testify/require"
)

// "Tests the correct serialization and deserialization of a StringMap, ensuring that parameter key-value pairs are accurately preserved when converted to and from byte representations."
func TestStringMap(t *testing.T) {
	tu.SetT(t)

	f := def.StringMap{
		Params: map[string][]byte{
			"a":  []byte("a"),
			"b":  []byte("bb"),
			"cc": []byte("cccc"),
		},
	}
	buf := f.Bytes()
	// Note: orders are not preserved, so we shouldn't check the result
	// require.Equal(t, []byte{0x85, 0x1, 0x61, 0x87, 0x1, 0x61,
	// 	0x85, 0x1, 0x62, 0x87, 0x2, 0x62, 0x62,
	// 	0x85, 0x2, 0x63, 0x63, 0x87, 0x4, 0x63, 0x63, 0x63, 0x63}, buf)
	f2 := tu.NoErr(def.ParseStringMap(enc.NewBufferView(buf), false))
	require.Equal(t, f, *f2)

	f = def.StringMap{
		Params: map[string][]byte{},
	}
	buf = f.Bytes()
	require.Equal(t, []byte{}, buf)
	f2 = tu.NoErr(def.ParseStringMap(enc.NewBufferView(buf), false))
	require.Equal(t, 0, len(f2.Params))
}

// This function tests the serialization and deserialization of an `IntStructMap` structure, ensuring that a map of unsigned integers to `Inner` objects can be correctly encoded to and decoded from bytes, including handling empty maps.
func TestIntStructMap(t *testing.T) {
	tu.SetT(t)

	f := def.IntStructMap{
		Params: map[uint64]*def.Inner{
			1: {1},
			2: {2},
			3: {3},
		},
	}
	buf := f.Bytes()
	// Note: orders are not preserved, so we shouldn't check the result
	// require.Equal(t, []byte{0x85, 0x1, 0x2, 0x87, 0x3, 0x1, 0x1, 0x2,
	// 	0x85, 0x1, 0x3, 0x87, 0x3, 0x1, 0x1, 0x3,
	// 	0x85, 0x1, 0x1, 0x87, 0x3, 0x1, 0x1, 0x1}, buf)
	f2 := tu.NoErr(def.ParseIntStructMap(enc.NewBufferView(buf), false))
	require.Equal(t, f, *f2)

	f = def.IntStructMap{
		Params: map[uint64]*def.Inner{},
	}
	buf = f.Bytes()
	require.Equal(t, []byte{}, buf)
	f2 = tu.NoErr(def.ParseIntStructMap(enc.NewBufferView(buf), false))
	require.Equal(t, 0, len(f2.Params))
}
