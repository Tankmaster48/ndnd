package encoding_test

import (
	"crypto/rand"
	"testing"
	"time"

	enc "github.com/named-data/ndnd/std/encoding"
	"github.com/named-data/ndnd/std/ndn"
	spec "github.com/named-data/ndnd/std/ndn/spec_2022"
	"github.com/named-data/ndnd/std/types/optional"
	tu "github.com/named-data/ndnd/std/utils/testutils"
)

// Generates `count` unsigned Data packets with random names of length `nameSize`, random payloads of size `payloadSize`, and predefined MetaInfo (Blob content type, 4s freshness) and SignatureInfo (Ed25519, key locator).
func encodeDataCases(count int, payloadSize int, nameSize int) (ret []*spec.Data) {
	keyName := tu.NoErrB(enc.NameFromStr("/go-ndn/bench/signer/KEY"))

	ret = make([]*spec.Data, count)
	for i := 0; i < count; i++ {
		ret[i] = &spec.Data{
			NameV:    randomNames(1, nameSize)[0],
			ContentV: enc.Wire{make([]byte, payloadSize)},
			MetaInfo: &spec.MetaInfo{
				ContentType:     optional.Some(uint64(ndn.ContentTypeBlob)),
				FreshnessPeriod: optional.Some(4000 * time.Millisecond),
			},
			SignatureInfo: &spec.SignatureInfo{
				SignatureType: uint64(ndn.SignatureEd25519),
				KeyLocator:    &spec.KeyLocator{Name: keyName},
			},
		}
		rand.Read(ret[i].ContentV[0])
	}
	return ret
}

// Constructs a Data packet with the provided data and initializes a 32-byte signature buffer in the wire format for subsequent signing.
func encodeData(data *spec.Data) enc.Wire {
	packet := &spec.Packet{Data: data}
	encoder := spec.PacketEncoder{
		Data_encoder: spec.DataEncoder{
			SignatureValue_estLen: 32,
		},
	}
	encoder.Init(packet)
	wire := encoder.Encode(packet)

	wire[encoder.Data_encoder.SignatureValue_wireIdx] = make([]byte, 32)
	return wire
}

// Benchmarks the encoding of multiple Data packets with specified payload and name sizes by generating test cases and measuring the performance of the encoding operation.
func encodeDataBench(b *testing.B, payloadSize, nameSize int) {
	count := b.N
	data := encodeDataCases(count, payloadSize, nameSize)

	b.ResetTimer()
	for i := 0; i < count; i++ {
		encodeData(data[i])
	}
}

// **Description:**  
Performs a benchmark test for encoding a small Data packet with a specified size and parameters.
func BenchmarkDataEncodeSmall(b *testing.B) {
	encodeDataBench(b, 100, 5)
}

// "Runs a benchmark for encoding Data packets with a medium-sized payload (1000 bytes) and 10 name components."
func BenchmarkDataEncodeMedium(b *testing.B) {
	encodeDataBench(b, 1000, 10)
}

// **Description:**  
Benchmarks the encoding performance of a Data packet with a medium-long name (1000 components) and 20-byte content.
func BenchmarkDataEncodeMediumLongName(b *testing.B) {
	encodeDataBench(b, 1000, 20)
}

// **Description:**  
Benchmark test that measures the performance of encoding a large Data packet with a specified size and content length.
func BenchmarkDataEncodeLarge(b *testing.B) {
	encodeDataBench(b, 8000, 10)
}

// This function benchmarks the decoding of NDN Data packets with specified payload and name sizes by generating encoded test data, then measuring the performance of reading and validating those packets.
func decodeDataBench(b *testing.B, payloadSize, nameSize int) {
	count := b.N
	buffers := make([][]byte, count)

	for i, data := range encodeDataCases(count, payloadSize, nameSize) {
		buffers[i] = encodeData(data).Join()
	}

	b.ResetTimer()
	for i := 0; i < count; i++ {
		p, _, err := spec.ReadPacket(enc.NewBufferView(buffers[i]))
		if err != nil || p.Data == nil {
			b.Fatal(err)
		}
	}
}

// This function benchmarks the decoding of a small Data packet with a 100-byte content size and 5 name components.
func BenchmarkDataDecodeSmall(b *testing.B) {
	decodeDataBench(b, 100, 5)
}

// "Runs a benchmark test for decoding a medium-sized Data packet with 1000 iterations and 10 samples using the decodeDataBench helper function."
func BenchmarkDataDecodeMedium(b *testing.B) {
	decodeDataBench(b, 1000, 10)
}

// Runs a benchmark for decoding a Data packet with a name of 20 components and 1000-byte content to measure performance.
func BenchmarkDataDecodeMediumLongName(b *testing.B) {
	decodeDataBench(b, 1000, 20)
}

// Benchmarks decoding a large Data packet with 8000-byte content and 10 segments.
func BenchmarkDataDecodeLarge(b *testing.B) {
	decodeDataBench(b, 8000, 10)
}
