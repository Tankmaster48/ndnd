package gen_basic_test

import (
	"testing"
	"time"

	enc "github.com/named-data/ndnd/std/encoding"
	"github.com/named-data/ndnd/std/encoding/tests/gen_basic"
	"github.com/named-data/ndnd/std/types/optional"
	tu "github.com/named-data/ndnd/std/utils/testutils"
	"github.com/stretchr/testify/require"
)

// This function tests the serialization and deserialization of a `FakeMetaInfo` structure using TLV (Type-Length-Value) encoding, validating correct byte output, parsing behavior, and error handling for invalid or malformed inputs in the Named-Data Networking (NDN) context.
func TestFakeMetaInfo(t *testing.T) {
	tu.SetT(t)

	f := gen_basic.FakeMetaInfo{
		Number: 1,
		Time:   2 * time.Second,
		Binary: []byte{3, 4, 5},
	}
	buf := f.Bytes()
	require.Equal(t,
		[]byte{
			0x18, 0x01, 0x01,
			0x19, 0x02, 0x07, 0xd0,
			0x1a, 0x03, 0x03, 0x04, 0x05,
		},
		buf)

	f2 := tu.NoErr(gen_basic.ParseFakeMetaInfo(enc.NewBufferView(buf), false))
	require.Equal(t, f, *f2)

	buf2 := []byte{
		0x19, 0x02, 0x07, 0xd0,
		0x1a, 0x03, 0x03, 0x04, 0x05,
	}
	tu.Err(gen_basic.ParseFakeMetaInfo(enc.NewBufferView(buf2), false))

	buf2 = []byte{
		0x18, 0x01, 0x01,
		0x19, 0x02, 0x07, 0xd0,
		0x1a, 0x08, 0x03, 0x04, 0x05,
	}
	tu.Err(gen_basic.ParseFakeMetaInfo(enc.NewBufferView(buf2), false))

	buf2 = []byte{
		0x18, 0x01, 0x01,
		0x19, 0x02, 0x07, 0xd0,
		0x1a, 0x03, 0x03, 0x04, 0x05,
		0x30, 0x01, 0x00,
	}
	f2 = tu.NoErr(gen_basic.ParseFakeMetaInfo(enc.NewBufferView(buf2), false))
	require.Equal(t, f, *f2)

	buf2 = []byte{
		0x18, 0x01, 0x01,
		0x19, 0x02, 0x07, 0xd0,
		0x1a, 0x03, 0x03, 0x04, 0x05,
		0x31, 0x01, 0x00,
	}
	f2 = tu.NoErr(gen_basic.ParseFakeMetaInfo(enc.NewBufferView(buf2), true))
	require.Equal(t, f, *f2)

	buf2 = []byte{
		0x18, 0x01, 0x01,
		0x19, 0x02, 0x07, 0xd0,
		0x1a, 0x03, 0x03, 0x04, 0x05,
		0x31, 0x01, 0x00,
	}
	tu.Err(gen_basic.ParseFakeMetaInfo(enc.NewBufferView(buf2), false))
}

// This function tests the serialization and deserialization of a `gen_basic.OptField` structure, verifying that optional fields (Number, Time, Binary, Bool) are correctly encoded into a TLV (Type-Length-Value) format and that parsing the resulting bytes reconstructs the original structure, including handling of presence/absence and zero values.
func TestOptField(t *testing.T) {
	tu.SetT(t)

	f := gen_basic.OptField{
		Number: optional.Some[uint64](1),
		Time:   optional.Some(2 * time.Second),
		Binary: []byte{3, 4, 5},
		Bool:   true,
	}
	buf := f.Bytes()
	require.Equal(t,
		[]byte{
			0x18, 0x01, 0x01,
			0x19, 0x02, 0x07, 0xd0,
			0x1a, 0x03, 0x03, 0x04, 0x05,
			0x30, 0x00,
		},
		buf)
	f2 := tu.NoErr(gen_basic.ParseOptField(enc.NewBufferView(buf), false))
	require.Equal(t, f, *f2)

	f = gen_basic.OptField{
		Number: optional.None[uint64](),
		Time:   optional.None[time.Duration](),
		Binary: nil,
		Bool:   false,
	}
	buf = f.Bytes()
	require.Equal(t,
		[]byte{},
		buf)
	f2 = tu.NoErr(gen_basic.ParseOptField(enc.NewBufferView(buf), false))
	require.Equal(t, f, *f2)

	f = gen_basic.OptField{
		Number: optional.Some[uint64](0),
		Time:   optional.Some(0 * time.Second),
		Binary: []byte{},
	}
	buf = f.Bytes()
	require.Equal(t,
		[]byte{
			0x18, 0x01, 0x00,
			0x19, 0x01, 0x00,
			0x1a, 0x00,
		},
		buf)
	f2 = tu.NoErr(gen_basic.ParseOptField(enc.NewBufferView(buf), false))
	require.Equal(t, f, *f2)
}

// This function tests the serialization and deserialization of a `WireNameField` struct, verifying correct TLV encoding of a combined wire format and NDN name, including edge cases like empty inputs.
func TestWireName(t *testing.T) {
	tu.SetT(t)

	f := gen_basic.WireNameField{
		Wire: enc.Wire{
			[]byte{1, 2, 3},
			[]byte{4, 5, 6},
		},
		Name: tu.NoErr(enc.NameFromStr("/A/B/C")),
	}
	buf := f.Bytes()
	require.Equal(t,
		[]byte{
			0x01, 0x06, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06,
			0x02, 0x09, 0x08, 0x01, 'A', 0x08, 0x01, 'B', 0x08, 0x01, 'C',
		},
		buf)
	f2 := tu.NoErr(gen_basic.ParseWireNameField(enc.NewBufferView(buf), false))
	require.True(t, f.Name.Equal(f2.Name))
	require.Equal(t, f.Wire.Join(), f2.Wire.Join())

	f2 = tu.NoErr(gen_basic.ParseWireNameField(enc.NewBufferView([]byte{}), false))
	require.Equal(t, enc.Name(nil), f2.Name)
	require.Equal(t, enc.Wire(nil), f2.Wire)

	f2 = tu.NoErr(gen_basic.ParseWireNameField(enc.NewBufferView(
		[]byte{
			0x01, 0x00, 0x02, 0x00,
		}), false))
	require.Equal(t, enc.Name{}, f2.Name)
	require.Equal(t, []byte{}, f2.Wire.Join())
}

// Tests the correct encoding and parsing of a `Markers` structure containing wire data and a name, verifying that the parsed result matches the original input after serialization.
func TestMarkers(t *testing.T) {
	tu.SetT(t)

	f := gen_basic.Markers{
		Wire: enc.Wire{
			[]byte{1, 2, 3},
			[]byte{4, 5, 6},
		},
		Name: tu.NoErr(enc.NameFromStr("/A/B/C")),
	}
	buf := f.Encode(1)
	require.Equal(t,
		[]byte{
			0x01, 0x06, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06,
			0x02, 0x09, 0x08, 0x01, 'A', 0x08, 0x01, 'B', 0x08, 0x01, 'C',
		},
		buf)
	f2 := gen_basic.ParseMarkers(buf, 2)
	require.True(t, f.Name.Equal(f2.Name))
	require.Equal(t, f.Wire.Join(), f2.Wire.Join())
}

// Tests that a NoCopyStruct is encoded and parsed correctly without copying underlying byte slices, ensuring data integrity and efficient memory usage during TLV encoding/decoding.
func TestNoCopy(t *testing.T) {
	tu.SetT(t)

	f := gen_basic.NoCopyStruct{
		Wire1: enc.Wire{
			[]byte{1, 2, 3},
			[]byte{4, 5, 6},
		},
		Number: 1,
		Wire2: enc.Wire{
			[]byte{7, 8, 9},
			[]byte{10, 11, 12},
		},
	}
	wire := f.Encode()
	require.Equal(t, []byte{0x01, 0x06}, []byte(wire[0]))
	require.Equal(t, []byte{0x01, 0x02, 0x03}, []byte(wire[1]))
	require.Equal(t, []byte{0x04, 0x05, 0x06}, []byte(wire[2]))
	require.Equal(t, []byte{0x02, 0x01, 0x01, 0x03, 0x06}, []byte(wire[3]))
	require.Equal(t, []byte{0x07, 0x08, 0x09}, []byte(wire[4]))
	require.Equal(t, []byte{0x0a, 0x0b, 0x0c}, []byte(wire[5]))
	for i := 6; i < len(wire); i++ {
		require.Equal(t, enc.Buffer(nil), wire[i])
	}
	f2 := tu.NoErr(gen_basic.ParseNoCopyStruct(enc.NewWireView(wire), true))
	require.Equal(t, f.Wire1.Join(), f2.Wire1.Join())
	require.Equal(t, f.Number, f2.Number)
	require.Equal(t, f.Wire2.Join(), f2.Wire2.Join())
}

// Tests the serialization and deserialization of a `StrField` struct containing two string fields (one optional), verifying correct byte encoding, parsing, and round-trip equality for both populated and empty/omitted values.
func TestStrField(t *testing.T) {
	tu.SetT(t)

	f := gen_basic.StrField{
		Str1: "hello",
		Str2: optional.Some("world"),
	}
	buf := f.Bytes()
	require.Equal(t, []byte{0x01, 0x05, 'h', 'e', 'l', 'l', 'o', 0x02, 0x05, 'w', 'o', 'r', 'l', 'd'}, buf)
	f2 := tu.NoErr(gen_basic.ParseStrField(enc.NewBufferView(buf), false))
	require.Equal(t, f, *f2)

	f = gen_basic.StrField{
		Str1: "",
		Str2: optional.None[string](),
	}
	buf = f.Bytes()
	require.Equal(t, []byte{0x01, 0x00}, buf)
	f2 = tu.NoErr(gen_basic.ParseStrField(enc.NewBufferView(buf), false))
	require.Equal(t, f, *f2)

	buf = []byte{}
	tu.Err(gen_basic.ParseStrField(enc.NewBufferView(buf), false))
}

// This function tests the serialization and deserialization of a `FixedUintField` structure containing various unsigned integer fields (uint8, uint32, uint64, and pointer to uint8), verifying correct TLV encoding/decoding, optional field handling, and pointer behavior.
func TestFixedUintField(t *testing.T) {
	tu.SetT(t)

	f := gen_basic.FixedUintField{
		Byte:    1,
		U32:     optional.Some[uint32](2),
		U64:     optional.Some[uint64](3),
		BytePtr: nil,
	}
	buf := f.Bytes()
	require.Equal(t, []byte{
		0x01, 0x01, 0x01,
		0x02, 0x04, 0x00, 0x00, 0x00, 0x02,
		0x03, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x03,
	}, buf)
	f2 := tu.NoErr(gen_basic.ParseFixedUintField(enc.NewBufferView(buf), false))
	require.Equal(t, f, *f2)

	f = gen_basic.FixedUintField{
		Byte:    0,
		U32:     optional.None[uint32](),
		U64:     optional.None[uint64](),
		BytePtr: nil,
	}
	buf = f.Bytes()
	require.Equal(t, []byte{
		0x01, 0x01, 0x00,
	}, buf)
	f2 = tu.NoErr(gen_basic.ParseFixedUintField(enc.NewBufferView(buf), false))
	require.Equal(t, f, *f2)

	buf = []byte{}
	tu.Err(gen_basic.ParseFixedUintField(enc.NewBufferView(buf), false))

	bytePtr := new(byte)
	*bytePtr = 5
	f = gen_basic.FixedUintField{
		Byte:    7,
		BytePtr: bytePtr,
	}

	buf = f.Bytes()

	f2 = tu.NoErr(gen_basic.ParseFixedUintField(enc.NewBufferView(buf), false))
	require.Equal(t, uint8(7), f2.Byte)
	require.Equal(t, uint8(5), *f2.BytePtr)

	f2.Byte = 8
	*f2.BytePtr = 9
	f2 = tu.NoErr(gen_basic.ParseFixedUintField(enc.NewBufferView(buf), false))
	require.Equal(t, uint8(7), f2.Byte)     // unchanged
	require.Equal(t, uint8(9), *f2.BytePtr) // changed
}
