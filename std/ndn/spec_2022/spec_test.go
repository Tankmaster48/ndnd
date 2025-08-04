package spec_2022_test

import (
	"crypto/sha256"
	"encoding/hex"
	"testing"
	"time"

	enc "github.com/named-data/ndnd/std/encoding"
	"github.com/named-data/ndnd/std/ndn"
	"github.com/named-data/ndnd/std/ndn/spec_2022"
	sig "github.com/named-data/ndnd/std/security/signer"
	"github.com/named-data/ndnd/std/types/optional"
	"github.com/named-data/ndnd/std/utils"
	tu "github.com/named-data/ndnd/std/utils/testutils"
	"github.com/stretchr/testify/require"
)

// Constructs a Data packet with the specified name, configuration (including content type), optional content, and signer, producing a wire-encoded packet with computed signature if signing is enabled.
func TestMakeDataBasic(t *testing.T) {
	tu.SetT(t)

	spec := spec_2022.Spec{}

	data, err := spec.MakeData(
		tu.NoErr(enc.NameFromStr("/local/ndn/prefix")),
		&ndn.DataConfig{
			ContentType: optional.Some(ndn.ContentTypeBlob),
		},
		nil,
		sig.NewSha256Signer(),
	)
	require.NoError(t, err)
	require.Equal(t, []byte(
		"\x06\x42\x07\x14\x08\x05local\x08\x03ndn\x08\x06prefix"+
			"\x14\x03\x18\x01\x00"+
			"\x16\x03\x1b\x01\x00"+
			"\x17 \x7f1\xe4\t\xc5z/\x1d\r\xdaVh8\xfd\xd9\x94"+
			"\xd8'S\x13[\xd7\x15\xa5\x9d%^\x80\xf2\xab\xf0\xb5"),
		data.Wire.Join())

	data, err = spec.MakeData(
		tu.NoErr(enc.NameFromStr("/local/ndn/prefix")),
		&ndn.DataConfig{
			ContentType: optional.Some(ndn.ContentTypeBlob),
		},
		enc.Wire{[]byte("01020304")},
		sig.NewSha256Signer(),
	)
	require.NoError(t, err)
	require.Equal(t, []byte(
		"\x06L\x07\x14\x08\x05local\x08\x03ndn\x08\x06prefix"+
			"\x14\x03\x18\x01\x00"+
			"\x15\x0801020304"+
			"\x16\x03\x1b\x01\x00"+
			"\x17 \x94\xe9\xda\x91\x1a\x11\xfft\x02i:G\x0cO\xdd!"+
			"\xe0\xc7\xb6\xfd\x8f\x9cn\xc5\x93{\x93\x04\xe0\xdf\xa6S"),
		data.Wire.Join())

	data, err = spec.MakeData(
		tu.NoErr(enc.NameFromStr("/local/ndn/prefix")),
		&ndn.DataConfig{
			ContentType: optional.Some(ndn.ContentTypeBlob),
		},
		nil,
		nil,
	)
	require.NoError(t, err)
	require.Equal(t, []byte(
		"\x06\x1b\x07\x14\x08\x05local\x08\x03ndn\x08\x06prefix"+
			"\x14\x03\x18\x01\x00"),
		data.Wire.Join())

	data, err = spec.MakeData(
		tu.NoErr(enc.NameFromStr("/E")),
		&ndn.DataConfig{
			ContentType: optional.None[ndn.ContentType](),
		},
		enc.Wire{},
		sig.NewSha256Signer(),
	)
	require.NoError(t, err)
	require.Equal(t, tu.NoErr(hex.DecodeString(
		"06300703080145"+
			"1400150016031b0100"+
			"1720f965ee682c6973c3cbaa7b69e4c7063680f83be93a46be2ccc98686134354b66")),
		data.Wire.Join())
}

// Constructs a signed Data packet with the specified name, metadata (ContentType, Freshness, FinalBlockID), and empty content using SHA-256 signing.
func TestMakeDataMetaInfo(t *testing.T) {
	tu.SetT(t)
	spec := spec_2022.Spec{}

	data, err := spec.MakeData(
		tu.NoErr(enc.NameFromStr("/local/ndn/prefix/37=%00")),
		&ndn.DataConfig{
			ContentType:  optional.Some(ndn.ContentTypeBlob),
			Freshness:    optional.Some(1000 * time.Millisecond),
			FinalBlockID: optional.Some(enc.NewSequenceNumComponent(2)),
		},
		nil,
		sig.NewSha256Signer(),
	)
	require.NoError(t, err)
	require.Equal(t, []byte(
		"\x06\x4e\x07\x17\x08\x05local\x08\x03ndn\x08\x06prefix\x25\x01\x00"+
			"\x14\x0c\x18\x01\x00\x19\x02\x03\xe8\x1a\x03\x3a\x01\x02"+
			"\x16\x03\x1b\x01\x00"+
			"\x17 \x0f^\xa1\x0c\xa7\xf5Fb\xf0\x9cOT\xe0FeC\x8f92\x04\x9d\xabP\x80o'\x94\xaa={hQ"),
		data.Wire.Join())
}

type testSigner struct{}

// Returns a fixed Name object representing "/KEY", used to identify the test signer's key in the NDN namespace.
func (testSigner) KeyName() enc.Name {
	name, _ := enc.NameFromStr("/KEY")
	return name
}

// Returns the key name associated with this signer as its KeyLocator.
func (t testSigner) KeyLocator() enc.Name {
	return t.KeyName()
}

// Returns the signature type associated with the test signer, which is defined as 200 for testing purposes.
func (testSigner) Type() ndn.SigType {
	return ndn.SigType(200)
}

// Returns an estimated size of 10 for the test signer's signature.
func (testSigner) EstimateSize() uint {
	return 10
}

// Returns a dummy 5-byte zero signature for testing, ignoring the input Wire and not performing actual cryptographic signing.
func (testSigner) Sign(enc.Wire) ([]byte, error) {
	return []byte{0, 0, 0, 0, 0}, nil
}

// Returns nil and an error indicating no public key is available, serving as a test implementation for scenarios where public key retrieval is not required or supported.
func (testSigner) Public() ([]byte, error) {
	return nil, ndn.ErrNoPubKey
}

// Constructs a Data packet with the specified name and ContentTypeBlob, using a test signer that generates placeholder signature fields, resulting in an unsigned packet with minimal signature data.
func TestMakeDataShrink(t *testing.T) {
	tu.SetT(t)
	spec := spec_2022.Spec{}

	data, err := spec.MakeData(
		tu.NoErr(enc.NameFromStr("/test")),
		&ndn.DataConfig{
			ContentType: optional.Some(ndn.ContentTypeBlob),
		},
		nil,
		testSigner{},
	)
	require.NoError(t, err)
	require.Equal(t, []byte{
		0x6, 0x22, 0x7, 0x6, 0x8, 0x4, 0x74, 0x65, 0x73, 0x74, 0x14, 0x3, 0x18, 0x1, 0x0,
		0x16, 0xc, 0x1b, 0x1, 0xc8, 0x1c, 0x7, 0x7, 0x5, 0x8, 0x3, 0x4b, 0x45, 0x59,
		0x17, 0x5, 0x0, 0x0, 0x0, 0x0, 0x0},
		data.Wire.Join())
}

// This function tests the NDN Data packet parsing logic by reading various TLV-encoded Data packets (with different fields like name, content, and signature) using the 2022 specification and verifying their correctness through field checks and signature validation.
func TestReadDataBasic(t *testing.T) {
	tu.SetT(t)
	spec := spec_2022.Spec{}

	data, covered, err := spec.ReadData(enc.NewBufferView([]byte(
		"\x06\x42\x07\x14\x08\x05local\x08\x03ndn\x08\x06prefix" +
			"\x14\x03\x18\x01\x00" +
			"\x16\x03\x1b\x01\x00" +
			"\x17 \x7f1\xe4\t\xc5z/\x1d\r\xdaVh8\xfd\xd9\x94" +
			"\xd8'S\x13[\xd7\x15\xa5\x9d%^\x80\xf2\xab\xf0\xb5"),
	))
	require.NoError(t, err)
	require.Equal(t, "/local/ndn/prefix", data.Name().String())
	require.Equal(t, ndn.ContentTypeBlob, data.ContentType().Unwrap())
	require.False(t, data.Freshness().IsSet())
	require.False(t, data.FinalBlockID().IsSet())
	require.Equal(t, ndn.SignatureDigestSha256, data.Signature().SigType())
	require.True(t, data.Content() == nil)
	h := sha256.New()
	for _, c := range covered {
		h.Write(c)
	}
	sig := h.Sum(nil)
	require.Equal(t, sig, data.Signature().SigValue())

	data, covered, err = spec.ReadData(enc.NewBufferView([]byte(
		"\x06L\x07\x14\x08\x05local\x08\x03ndn\x08\x06prefix" +
			"\x14\x03\x18\x01\x00" +
			"\x15\x0801020304" +
			"\x16\x03\x1b\x01\x00" +
			"\x17 \x94\xe9\xda\x91\x1a\x11\xfft\x02i:G\x0cO\xdd!" +
			"\xe0\xc7\xb6\xfd\x8f\x9cn\xc5\x93{\x93\x04\xe0\xdf\xa6S"),
	))
	require.NoError(t, err)
	require.Equal(t, "/local/ndn/prefix", data.Name().String())
	require.Equal(t, ndn.ContentTypeBlob, data.ContentType().Unwrap())
	require.False(t, data.Freshness().IsSet())
	require.False(t, data.FinalBlockID().IsSet())
	require.Equal(t, ndn.SignatureDigestSha256, data.Signature().SigType())
	require.Equal(t, []byte("01020304"), data.Content().Join())
	h = sha256.New()
	for _, c := range covered {
		h.Write(c)
	}
	sig = h.Sum(nil)
	require.Equal(t, sig, data.Signature().SigValue())

	data, _, err = spec.ReadData(enc.NewBufferView([]byte(
		"\x06\x1b\x07\x14\x08\x05local\x08\x03ndn\x08\x06prefix" +
			"\x14\x03\x18\x01\x00"),
	))
	require.NoError(t, err)
	require.Equal(t, "/local/ndn/prefix", data.Name().String())
	require.Equal(t, ndn.ContentTypeBlob, data.ContentType().Unwrap())
	require.False(t, data.Freshness().IsSet())
	require.False(t, data.FinalBlockID().IsSet())
	require.Equal(t, ndn.SignatureNone, data.Signature().SigType())
	require.True(t, data.Content() == nil)
	require.True(t, data.Signature().SigValue() == nil)

	data, covered, err = spec.ReadData(enc.NewBufferView(tu.NoErr(hex.DecodeString(
		"06300703080145" +
			"1400150016031b0100" +
			"1720f965ee682c6973c3cbaa7b69e4c7063680f83be93a46be2ccc98686134354b66"),
	)))
	require.NoError(t, err)
	require.Equal(t, "/E", data.Name().String())
	require.False(t, data.ContentType().IsSet())
	require.False(t, data.Freshness().IsSet())
	require.False(t, data.FinalBlockID().IsSet())
	require.Equal(t, ndn.SignatureDigestSha256, data.Signature().SigType())
	require.Equal(t, 0, len(data.Content().Join()))
	h = sha256.New()
	for _, c := range covered {
		h.Write(c)
	}
	sig = h.Sum(nil)
	require.Equal(t, sig, data.Signature().SigValue())
}

// Verifies the correct parsing of a Data packet with metadata fields (name, content type, freshness, final block ID, and SHA-256 digest signature) and validates the signature against the covered TLV-encoded fields.
func TestReadDataMetaInfo(t *testing.T) {
	tu.SetT(t)
	spec := spec_2022.Spec{}

	data, covered, err := spec.ReadData(enc.NewBufferView([]byte(
		"\x06\x4e\x07\x17\x08\x05local\x08\x03ndn\x08\x06prefix\x25\x01\x00" +
			"\x14\x0c\x18\x01\x00\x19\x02\x03\xe8\x1a\x03\x3a\x01\x02" +
			"\x16\x03\x1b\x01\x00" +
			"\x17 \x0f^\xa1\x0c\xa7\xf5Fb\xf0\x9cOT\xe0FeC\x8f92\x04\x9d\xabP\x80o'\x94\xaa={hQ"),
	))
	require.NoError(t, err)
	require.Equal(t, "/local/ndn/prefix/37=%00", data.Name().String())
	require.Equal(t, ndn.ContentTypeBlob, data.ContentType().Unwrap())
	require.Equal(t, 1000*time.Millisecond, data.Freshness().Unwrap())
	require.Equal(t, enc.NewSequenceNumComponent(2), data.FinalBlockID().Unwrap())
	require.Equal(t, ndn.SignatureDigestSha256, data.Signature().SigType())
	require.True(t, data.Content() == nil)
	h := sha256.New()
	for _, c := range covered {
		h.Write(c)
	}
	sig := h.Sum(nil)
	require.Equal(t, sig, data.Signature().SigValue())
}

// Constructs an Interest packet with a specified name and configuration parameters such as lifetime, freshness, hop limit, nonce, and forwarding hints, encoding them into the NDN TLV wire format.
func TestMakeIntBasic(t *testing.T) {
	tu.SetT(t)
	spec := spec_2022.Spec{}

	interest, err := spec.MakeInterest(
		tu.NoErr(enc.NameFromStr("/local/ndn/prefix")),
		&ndn.InterestConfig{
			Lifetime: optional.Some(4 * time.Second),
		},
		nil,
		nil,
	)
	require.NoError(t, err)
	require.Equal(t, "/local/ndn/prefix", interest.FinalName.String())
	require.Equal(t, []byte("\x05\x1a\x07\x14\x08\x05local\x08\x03ndn\x08\x06prefix\x0c\x02\x0f\xa0"),
		interest.Wire.Join())

	interest, err = spec.MakeInterest(
		tu.NoErr(enc.NameFromStr("/local/ndn/prefix")),
		&ndn.InterestConfig{
			CanBePrefix: true,
			MustBeFresh: true,
			Lifetime:    optional.Some(10 * time.Millisecond),
			HopLimit:    utils.IdPtr[byte](1),
			Nonce:       optional.Some[uint32](0),
		},
		nil,
		nil,
	)
	require.NoError(t, err)
	require.Equal(t, []byte(
		"\x05\x26\x07\x14\x08\x05local\x08\x03ndn\x08\x06prefix"+
			"\x21\x00\x12\x00\x0a\x04\x00\x00\x00\x00\x0c\x01\x0a\x22\x01\x01"),
		interest.Wire.Join())

	interest, err = spec.MakeInterest(
		tu.NoErr(enc.NameFromStr("/local/ndn/prefix")),
		&ndn.InterestConfig{
			Lifetime: optional.Some(4 * time.Second),
			Nonce:    optional.Some[uint32](0x01020304),
			ForwardingHint: []enc.Name{
				tu.NoErr(enc.NameFromStr("/name/A")),
				tu.NoErr(enc.NameFromStr("/ndn/B")),
				tu.NoErr(enc.NameFromBytes([]byte("\x07\x0d\x08\x0bshekkuenseu"))),
			},
		},
		nil,
		nil,
	)
	require.NoError(t, err)
	require.Equal(t, []byte(
		"\x05\x46\x07\x14\x08\x05local\x08\x03ndn\x08\x06prefix\x1e\x24"+
			"\x07\x09\x08\x04name\x08\x01A"+
			"\x07\x08\x08\x03ndn\x08\x01B"+
			"\x07\r\x08\x0bshekkuenseu"+
			"\x0a\x04\x01\x02\x03\x04\x0c\x02\x0f\xa0"),
		interest.Wire.Join())
}

// Constructs and verifies an NDN Interest packet with a large (384-byte) application-specific parameter and a 4-second lifetime, ensuring correct encoding/decoding and name validation.
func TestMakeIntLargeAppParam(t *testing.T) {
	tu.SetT(t)
	spec := spec_2022.Spec{}

	appParam := make([]byte, 384)
	for i := range appParam {
		appParam[i] = byte(i & 0xff)
	}
	encoded, err := spec.MakeInterest(
		tu.NoErr(enc.NameFromStr("/interest/with/large/prefix")),
		&ndn.InterestConfig{
			Lifetime: optional.Some(4 * time.Second),
		},
		enc.Wire{appParam},
		sig.NewHmacSigner([]byte("temp-hmac-key")),
	)
	require.NoError(t, err)

	interest, _, err := spec.ReadInterest(enc.NewWireView(encoded.Wire))
	require.NoError(t, err)
	require.Equal(t, appParam, interest.AppParam().Join())
	require.True(t, interest.Name().Equal(encoded.FinalName))
}

// Constructs a signed Interest packet with the given name, configuration, application-specific data, and SHA-256 signer, appending the signature hash to the final name component as per the NDN 2022 specification.
func TestMakeIntSign(t *testing.T) {
	tu.SetT(t)
	spec := spec_2022.Spec{}

	interest, err := spec.MakeInterest(
		tu.NoErr(enc.NameFromStr("/local/ndn/prefix")),
		&ndn.InterestConfig{
			Lifetime: optional.Some(4 * time.Second),
		},
		enc.Wire{[]byte{1, 2, 3, 4}},
		nil,
	)
	require.NoError(t, err)
	require.Equal(t,
		"/local/ndn/prefix/params-sha256=47756f21fe0ee265149aa2be3c63c538a72378e9b0a58b39c5916367d35bda10",
		interest.FinalName.String())
	require.Equal(t, []byte(
		"\x05\x42\x07\x36\x08\x05local\x08\x03ndn\x08\x06prefix"+
			"\x02 \x47\x75\x6f\x21\xfe\x0e\xe2\x65\x14\x9a\xa2\xbe\x3c\x63\xc5\x38"+
			"\xa7\x23\x78\xe9\xb0\xa5\x8b\x39\xc5\x91\x63\x67\xd3\x5b\xda\x10"+
			"\x0c\x02\x0f\xa0\x24\x04\x01\x02\x03\x04"),
		interest.Wire.Join())

	// "/test/params-sha256=FFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF/ndn" is not supported yet

	interest, err = spec.MakeInterest(
		tu.NoErr(enc.NameFromStr("/local/ndn/prefix")),
		&ndn.InterestConfig{
			Lifetime: optional.Some(4 * time.Second),
			Nonce:    optional.Some[uint32](0x6c211166),
		},
		enc.Wire{[]byte{1, 2, 3, 4}},
		sig.NewSha256Signer(),
	)
	require.NoError(t, err)
	require.Equal(t,
		"/local/ndn/prefix/params-sha256=8e6e36d7eabcde43756140c90bda09d500d2a577f2f533b569f0441df0a7f9e2",
		interest.FinalName.String())
	require.Equal(t, []byte(
		"\x05\x6f\x07\x36\x08\x05local\x08\x03ndn\x08\x06prefix"+
			"\x02 \x8e\x6e\x36\xd7\xea\xbc\xde\x43\x75\x61\x40\xc9\x0b\xda\x09\xd5"+
			"\x00\xd2\xa5\x77\xf2\xf5\x33\xb5\x69\xf0\x44\x1d\xf0\xa7\xf9\xe2"+
			"\x0a\x04\x6c\x21\x11\x66\x0c\x02\x0f\xa0"+
			"\x24\x04\x01\x02\x03\x04"+
			"\x2c\x03\x1b\x01\x00"+
			"\x2e \xea\xa8\xf0\x99\x08\x63\x78\x95\x1d\xe0\x5f\xf1\xde\xbb\xc1\x18"+
			"\xb5\x21\x8b\x2f\xca\xa0\xb5\x1d\x18\xfa\xbc\x29\xf5\x4d\x58\xff"),
		interest.Wire.Join())

	interest, err = spec.MakeInterest(
		tu.NoErr(enc.NameFromStr("/local/ndn/prefix")),
		&ndn.InterestConfig{
			Lifetime: optional.Some(4 * time.Second),
			Nonce:    optional.Some[uint32](0x6c211166),
		},
		enc.Wire{},
		sig.NewSha256Signer(),
	)
	require.NoError(t, err)
	require.Equal(t,
		"/local/ndn/prefix/params-sha256=4077a57049d83848b525a423ab978e6480f96d5ca38a80a5e2d6e250a617be4f",
		interest.FinalName.String())
	require.Equal(t, []byte(
		"\x05\x6b\x07\x36\x08\x05local\x08\x03ndn\x08\x06prefix"+
			"\x02 \x40\x77\xa5\x70\x49\xd8\x38\x48\xb5\x25\xa4\x23\xab\x97\x8e\x64"+
			"\x80\xf9\x6d\x5c\xa3\x8a\x80\xa5\xe2\xd6\xe2\x50\xa6\x17\xbe\x4f"+
			"\x0a\x04\x6c\x21\x11\x66\x0c\x02\x0f\xa0"+
			"\x24\x00"+
			"\x2c\x03\x1b\x01\x00"+
			"\x2e \x09\x4e\x00\x9d\x74\x59\x82\x5c\xa0\x2d\xaa\xb7\xad\x60\x48\x30"+
			"\x39\x19\xd8\x99\x80\x25\xbe\xff\xa6\xf9\x96\x79\xd6\x5e\x9f\x62"),
		interest.Wire.Join())
}

// Tests the parsing of NDN Interest packets according to the 2022 specification, validating correct handling of name components, lifetime, nonce, hop limit, application parameters, and cryptographic signatures (including SHA-256) across multiple test cases with both valid and invalid inputs.
func TestReadIntBasic(t *testing.T) {
	tu.SetT(t)
	spec := spec_2022.Spec{}

	interest, _, err := spec.ReadInterest(enc.NewBufferView([]byte(
		"\x05\x1a\x07\x14\x08\x05local\x08\x03ndn\x08\x06prefix\x0c\x02\x0f\xa0"),
	))
	require.NoError(t, err)
	require.Equal(t, "/local/ndn/prefix", interest.Name().String())
	require.Equal(t, 4*time.Second, interest.Lifetime().Unwrap())
	require.True(t, interest.AppParam() == nil)
	require.False(t, interest.CanBePrefix())
	require.False(t, interest.MustBeFresh())
	require.False(t, interest.Nonce().IsSet())
	require.True(t, interest.HopLimit() == nil)
	require.True(t, interest.Signature().SigType() == ndn.SignatureNone)

	interest, _, err = spec.ReadInterest(enc.NewBufferView([]byte(
		"\x05\x26\x07\x14\x08\x05local\x08\x03ndn\x08\x06prefix" +
			"\x21\x00\x12\x00\x0a\x04\x00\x00\x00\x00\x0c\x01\x0a\x22\x01\x01"),
	))
	require.NoError(t, err)
	require.Equal(t, "/local/ndn/prefix", interest.Name().String())
	require.Equal(t, 10*time.Millisecond, interest.Lifetime().Unwrap())
	require.True(t, interest.AppParam() == nil)
	require.True(t, interest.CanBePrefix())
	require.True(t, interest.MustBeFresh())
	require.Equal(t, uint32(0), interest.Nonce().Unwrap())
	require.Equal(t, uint(1), *interest.HopLimit())
	require.True(t, interest.Signature().SigType() == ndn.SignatureNone)

	interest, _, err = spec.ReadInterest(enc.NewBufferView([]byte(
		"\x05\x42\x07\x36\x08\x05local\x08\x03ndn\x08\x06prefix" +
			"\x02 \x47\x75\x6f\x21\xfe\x0e\xe2\x65\x14\x9a\xa2\xbe\x3c\x63\xc5\x38" +
			"\xa7\x23\x78\xe9\xb0\xa5\x8b\x39\xc5\x91\x63\x67\xd3\x5b\xda\x10" +
			"\x0c\x02\x0f\xa0\x24\x04\x01\x02\x03\x04"),
	))
	require.NoError(t, err)
	require.Equal(t,
		"/local/ndn/prefix/params-sha256=47756f21fe0ee265149aa2be3c63c538a72378e9b0a58b39c5916367d35bda10",
		interest.Name().String())
	require.Equal(t, 4*time.Second, interest.Lifetime().Unwrap())
	require.False(t, interest.CanBePrefix())
	require.False(t, interest.MustBeFresh())
	require.Equal(t, []byte{1, 2, 3, 4}, interest.AppParam().Join())
	require.True(t, interest.Signature().SigType() == ndn.SignatureNone)

	// Reject wrong digest
	_, _, err = spec.ReadInterest(enc.NewBufferView([]byte(
		"\x05\x42\x07\x36\x08\x05local\x08\x03ndn\x08\x06prefix" +
			"\x02 \x47\x75\x6f\x21\xfe\x0e\xe2\x65\x14\x9a\xa2\xbe\x3c\x63\xc5\x38" +
			"\xa7\x23\x78\xe9\xb0\xa5\x8b\x39\xc5\x91\x63\x67\xd3\x5b\x00\x00" +
			"\x0c\x02\x0f\xa0\x24\x04\x01\x02\x03\x04"),
	))
	require.Error(t, err)

	var covered enc.Wire
	interest, covered, err = spec.ReadInterest(enc.NewBufferView([]byte(
		"\x05\x6f\x07\x36\x08\x05local\x08\x03ndn\x08\x06prefix" +
			"\x02 \x8e\x6e\x36\xd7\xea\xbc\xde\x43\x75\x61\x40\xc9\x0b\xda\x09\xd5" +
			"\x00\xd2\xa5\x77\xf2\xf5\x33\xb5\x69\xf0\x44\x1d\xf0\xa7\xf9\xe2" +
			"\x0a\x04\x6c\x21\x11\x66\x0c\x02\x0f\xa0" +
			"\x24\x04\x01\x02\x03\x04" +
			"\x2c\x03\x1b\x01\x00" +
			"\x2e \xea\xa8\xf0\x99\x08\x63\x78\x95\x1d\xe0\x5f\xf1\xde\xbb\xc1\x18" +
			"\xb5\x21\x8b\x2f\xca\xa0\xb5\x1d\x18\xfa\xbc\x29\xf5\x4d\x58\xff"),
	))
	require.NoError(t, err)
	require.Equal(t,
		"/local/ndn/prefix/params-sha256=8e6e36d7eabcde43756140c90bda09d500d2a577f2f533b569f0441df0a7f9e2",
		interest.Name().String())
	require.Equal(t, uint32(0x6c211166), interest.Nonce().Unwrap())
	require.Equal(t, []byte{1, 2, 3, 4}, interest.AppParam().Join())
	require.True(t, interest.Signature().SigType() == ndn.SignatureDigestSha256)
	h := sha256.New()
	for _, c := range covered {
		h.Write(c)
	}
	sig := h.Sum(nil)
	require.Equal(t, sig, interest.Signature().SigValue())

	interest, covered, err = spec.ReadInterest(enc.NewBufferView([]byte(
		"\x05\x6b\x07\x36\x08\x05local\x08\x03ndn\x08\x06prefix" +
			"\x02 \x40\x77\xa5\x70\x49\xd8\x38\x48\xb5\x25\xa4\x23\xab\x97\x8e\x64" +
			"\x80\xf9\x6d\x5c\xa3\x8a\x80\xa5\xe2\xd6\xe2\x50\xa6\x17\xbe\x4f" +
			"\x0a\x04\x6c\x21\x11\x66\x0c\x02\x0f\xa0" +
			"\x24\x00" +
			"\x2c\x03\x1b\x01\x00" +
			"\x2e \x09\x4e\x00\x9d\x74\x59\x82\x5c\xa0\x2d\xaa\xb7\xad\x60\x48\x30" +
			"\x39\x19\xd8\x99\x80\x25\xbe\xff\xa6\xf9\x96\x79\xd6\x5e\x9f\x62"),
	))
	require.NoError(t, err)
	require.Equal(t,
		"/local/ndn/prefix/params-sha256=4077a57049d83848b525a423ab978e6480f96d5ca38a80a5e2d6e250a617be4f",
		interest.Name().String())
	require.Equal(t, uint32(0x6c211166), interest.Nonce().Unwrap())
	require.Equal(t, []byte{}, interest.AppParam().Join())
	require.True(t, interest.Signature().SigType() == ndn.SignatureDigestSha256)
	h = sha256.New()
	for _, c := range covered {
		h.Write(c)
	}
	sig = h.Sum(nil)
	require.Equal(t, sig, interest.Signature().SigValue())
}

// Tests that the 2022 spec's `ReadInterest` function correctly returns errors for various malformed or invalid Interest encodings, such as incorrect TLV types, invalid length fields, and insufficient data.
func TestReadIntErrors(t *testing.T) {
	tu.SetT(t)
	spec := spec_2022.Spec{}

	_, _, err := spec.ReadInterest(enc.NewBufferView([]byte(
		"\x05\x6b\x07\x36\x08\x05local\x08\x03ndn\x08\x06prefix"),
	))
	require.Error(t, err)

	_, _, err = spec.ReadInterest(enc.NewBufferView([]byte(
		"\x05\x6b\x07\x14\x08\x05local\x08\x03ndn\x08\x06prefix"),
	))
	require.Error(t, err)

	_, _, err = spec.ReadInterest(enc.NewBufferView([]byte(
		"\x06\x6b\x07\x36\x08\x05local\x08\x03ndn\x08\x06prefix"),
	))
	require.Error(t, err)

	_, _, err = spec.ReadInterest(enc.NewBufferView([]byte(
		"\x01\x00"),
	))
	require.Error(t, err)
}
