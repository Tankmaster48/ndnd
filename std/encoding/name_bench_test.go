package encoding_test

import (
	"crypto/rand"
	"encoding/binary"
	"runtime"
	"testing"

	enc "github.com/named-data/ndnd/std/encoding"
	tu "github.com/named-data/ndnd/std/utils/testutils"
)

// Generates a slice of randomly created NDN names, each containing a specified number of components with randomly generated TLV types (minimum 1024) and byte values.
func randomNames(count int, size int) []enc.Name {
	names := make([]enc.Name, count)
	for i := 0; i < count; i++ {
		for j := 0; j < size; j++ {
			bytes := make([]byte, 12+j)
			rand.Read(bytes)
			typ := max(enc.TLNum(uint16(binary.BigEndian.Uint16(bytes[:4])-1024)), 1024)
			names[i] = append(names[i], enc.NewBytesComponent(typ, bytes[4:]))
		}
	}
	return names
}

// This function benchmarks the performance of encoding randomly generated names of a specified size using the provided encoding function, executing the encode operation `b.N` times as part of a Go testing benchmark.
func benchmarkNameEncode(b *testing.B, size int, encode func(name enc.Name)) {
	runtime.GC()
	names := randomNames(b.N, size)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		encode(names[i])
	}
}

// "Benchmarks the performance of hashing a Name with 20 components by repeatedly encoding and computing its cryptographic hash."
func BenchmarkNameHash(b *testing.B) {
	benchmarkNameEncode(b, 20, func(name enc.Name) { _ = name.Hash() })
}

// Benchmarks the performance of computing a prefix hash for a name with 20 components by encoding and discarding the result.
func BenchmarkNameHashPrefix(b *testing.B) {
	benchmarkNameEncode(b, 20, func(name enc.Name) { _ = name.PrefixHash() })
}

// Benchmarks the performance of converting an enc.Name object with 20 components to a string representation using its String() method.
func BenchmarkNameStringEncode(b *testing.B) {
	benchmarkNameEncode(b, 20, func(name enc.Name) { _ = name.String() })
}

// Benchmarks encoding a Name with 20 components into a TLV-formatted string representation.
func BenchmarkNameTlvStrEncode(b *testing.B) {
	benchmarkNameEncode(b, 20, func(name enc.Name) { _ = name.TlvStr() })
}

// **Description:**  
This benchmark function measures the performance of encoding an NDN name to its byte representation using the `Bytes()` method, testing a name with 20 components over multiple iterations.
func BenchmarkNameBytesEncode(b *testing.B) {
	benchmarkNameEncode(b, 20, func(name enc.Name) { _ = name.Bytes() })
}

// Benchmarks the performance of converting a single NameComponent to a string representation using the String() method.
func BenchmarkNameComponentStringEncode(b *testing.B) {
	benchmarkNameEncode(b, 1, func(name enc.Name) { _ = name[0].String() })
}

// Benchmarks encoding the first component of an NDN Name to a TLV-formatted string.
func BenchmarkNameComponentTlvStrEncode(b *testing.B) {
	benchmarkNameEncode(b, 1, func(name enc.Name) { _ = name[0].TlvStr() })
}

// **Description:**  
Runs a benchmark to measure the performance of cloning a name by encoding and cloning it 20 times using the `benchmarkNameEncode` helper.
func BenchmarkNameClone(b *testing.B) {
	benchmarkNameEncode(b, 20, func(name enc.Name) { _ = name.Clone() })
}

// Benchmarks the decoding of pre-encoded names using a specified encoding type and decode function, measuring performance across multiple randomly generated names of a given size.
func benchmarkNameDecode[T any](b *testing.B, size int, encode func(name enc.Name) T, decode func(e T)) {
	names := randomNames(b.N, size)
	nameEncs := make([]T, b.N)
	for i := 0; i < b.N; i++ {
		nameEncs[i] = encode(names[i])
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		decode(nameEncs[i])
	}
}

// This function benchmarks the performance of converting an `enc.Name` to a string representation using `String()` and decoding it back using `enc.NameFromStr`, measuring the time for 20-component names.
func BenchmarkNameStringDecode(b *testing.B) {
	benchmarkNameDecode(b, 20,
		func(name enc.Name) string { return name.String() },
		func(s string) { _ = tu.NoErrB(enc.NameFromStr(s)) })
}

// Benchmarks the decoding of a name encoded in TLV string format using helper functions for encoding and decoding.
func BenchmarkNameTlvStrDecode(b *testing.B) {
	benchmarkNameDecode(b, 20,
		func(name enc.Name) string { return name.TlvStr() },
		func(s string) { _ = tu.NoErrB(enc.NameFromTlvStr(s)) })
}

// Benchmarks the performance of converting a name component to a string and decoding it back into a component using `ComponentFromStr`, ensuring error-free operation.
func BenchmarkNameComponentStringDecode(b *testing.B) {
	benchmarkNameDecode(b, 1,
		func(name enc.Name) string { return name[0].String() },
		func(s string) { _ = tu.NoErrB(enc.ComponentFromStr(s)) })
}

// This function benchmarks the performance of decoding a single Name component from its TLV (Type-Length-Value) string representation using `ComponentFromTlvStr`, by generating test data with `TlvStr()` and measuring the decode operation's speed.
func BenchmarkNameComponentTlvStrDecode(b *testing.B) {
	benchmarkNameDecode(b, 1,
		func(name enc.Name) string { return name[0].TlvStr() },
		func(s string) { _ = tu.NoErrB(enc.ComponentFromTlvStr(s)) })
}
