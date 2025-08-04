package basic_test

import (
	"testing"
	"time"

	enc "github.com/named-data/ndnd/std/encoding"
	basic_engine "github.com/named-data/ndnd/std/engine/basic"
	"github.com/named-data/ndnd/std/engine/face"
	"github.com/named-data/ndnd/std/ndn"
	"github.com/named-data/ndnd/std/ndn/spec_2022"
	sig "github.com/named-data/ndnd/std/security/signer"
	"github.com/named-data/ndnd/std/types/optional"
	tu "github.com/named-data/ndnd/std/utils/testutils"
	"github.com/stretchr/testify/require"
)

// Sets up a test environment with a dummy face and timer, runs the provided test logic function with these components, and ensures proper cleanup of the engine.
func executeTest(t *testing.T, main func(*face.DummyFace, *basic_engine.Engine, *basic_engine.DummyTimer)) {
	tu.SetT(t)

	face := face.NewDummyFace()
	timer := basic_engine.NewDummyTimer()
	engine := basic_engine.NewEngine(face, timer)
	require.NoError(t, engine.Start())

	main(face, engine, timer)

	require.NoError(t, engine.Stop())
}

// "Executes a test scenario to verify the correct initialization and startup behavior of the engine using dummy face and timer components."
func TestEngineStart(t *testing.T) {
	executeTest(t, func(face *face.DummyFace, engine *basic_engine.Engine, timer *basic_engine.DummyTimer) {
	})
}

// Sends an Interest with freshness and lifetime requirements, verifies receipt of a matching Data packet with expected name, freshness, and content via a callback.
func TestConsumerBasic(t *testing.T) {
	executeTest(t, func(face *face.DummyFace, engine *basic_engine.Engine, timer *basic_engine.DummyTimer) {
		hitCnt := 0

		spec := engine.Spec()
		name := tu.NoErr(enc.NameFromStr("/example/testApp/randomData/t=1570430517101"))
		config := &ndn.InterestConfig{
			MustBeFresh: true,
			CanBePrefix: false,
			Lifetime:    optional.Some(6 * time.Second),
		}
		interest, err := spec.MakeInterest(name, config, nil, nil)
		require.NoError(t, err)
		err = engine.Express(interest, func(args ndn.ExpressCallbackArgs) {
			hitCnt += 1
			require.Equal(t, ndn.InterestResultData, args.Result)
			require.True(t, args.Data.Name().Equal(name))
			require.Equal(t, 1*time.Second, args.Data.Freshness().Unwrap())
			require.Equal(t, []byte("Hello, world!"), args.Data.Content().Join())
		})
		require.NoError(t, err)
		buf := tu.NoErr(face.Consume())
		require.Equal(t, enc.Buffer(
			"\x050\x07(\x08\x07example\x08\x07testApp\x08\nrandomData"+
				"\x38\x08\x00\x00\x01m\xa4\xf3\xffm\x12\x00\x0c\x02\x17p"),
			buf)
		timer.MoveForward(500 * time.Millisecond)
		require.NoError(t, face.FeedPacket(enc.Buffer(
			"\x06B\x07(\x08\x07example\x08\x07testApp\x08\nrandomData"+
				"\x38\x08\x00\x00\x01m\xa4\xf3\xffm\x14\x07\x18\x01\x00\x19\x02\x03\xe8"+
				"\x15\rHello, world!",
		)))

		require.Equal(t, 1, hitCnt)
	})
}

// TODO: TestInterestCancel

// Tests the handling of a NACK response (specifically NackReasonNoRoute) for an expressed Interest by verifying callback invocation, packet encoding, and correct result codes.
func TestInterestNack(t *testing.T) {
	executeTest(t, func(face *face.DummyFace, engine *basic_engine.Engine, timer *basic_engine.DummyTimer) {
		hitCnt := 0

		spec := engine.Spec()
		name := tu.NoErr(enc.NameFromStr("/localhost/nfd/faces/events"))
		config := &ndn.InterestConfig{
			MustBeFresh: true,
			CanBePrefix: true,
			Lifetime:    optional.Some(1 * time.Second),
		}
		interest, err := spec.MakeInterest(name, config, nil, nil)
		require.NoError(t, err)
		err = engine.Express(interest, func(args ndn.ExpressCallbackArgs) {
			hitCnt += 1
			require.Equal(t, ndn.InterestResultNack, args.Result)
			require.Equal(t, spec_2022.NackReasonNoRoute, args.NackReason)
		})
		require.NoError(t, err)
		buf := tu.NoErr(face.Consume())
		require.Equal(t, enc.Buffer(
			"\x05)\x07\x1f\x08\tlocalhost\x08\x03nfd\x08\x05faces\x08\x06events"+
				"\x21\x00\x12\x00\x0c\x02\x03\xe8"),
			buf)
		timer.MoveForward(500 * time.Millisecond)
		require.NoError(t, face.FeedPacket(enc.Buffer(
			"\x64\x36\xfd\x03\x20\x05\xfd\x03\x21\x01\x96"+
				"\x50\x2b\x05)\x07\x1f\x08\tlocalhost\x08\x03nfd\x08\x05faces\x08\x06events"+
				"\x21\x00\x12\x00\x0c\x02\x03\xe8",
		)))

		require.Equal(t, 1, hitCnt)
	})
}

// Tests that expressing an Interest with a short lifetime results in a timeout callback when no Data is received before the timer expires, even if a Data packet is subsequently sent.
func TestInterestTimeout(t *testing.T) {
	executeTest(t, func(face *face.DummyFace, engine *basic_engine.Engine, timer *basic_engine.DummyTimer) {
		hitCnt := 0

		spec := engine.Spec()
		name := tu.NoErr(enc.NameFromStr("not important"))
		config := &ndn.InterestConfig{
			Lifetime: optional.Some(10 * time.Millisecond),
		}
		interest, err := spec.MakeInterest(name, config, nil, nil)
		require.NoError(t, err)
		err = engine.Express(interest, func(args ndn.ExpressCallbackArgs) {
			hitCnt += 1
			require.Equal(t, ndn.InterestResultTimeout, args.Result)
		})
		require.NoError(t, err)
		buf := tu.NoErr(face.Consume())
		require.Equal(t, enc.Buffer("\x05\x14\x07\x0f\x08\rnot important\x0c\x01\x0a"), buf)
		timer.MoveForward(50 * time.Millisecond)

		data, err := spec.MakeData(name, &ndn.DataConfig{}, enc.Wire{enc.Buffer("\x0a")}, sig.NewSha256Signer())
		require.NoError(t, err)
		require.NoError(t, face.FeedPacket(data.Wire.Join()))

		require.Equal(t, 1, hitCnt)
	})
}

// Tests the behavior of the `CanBePrefix` flag in NDN Interests by verifying that setting `CanBePrefix: true` allows an Interest to match and receive a Data packet with a longer name, while `CanBePrefix: false` results in a timeout unless the Data name exactly matches.
func TestInterestCanBePrefix(t *testing.T) {
	executeTest(t, func(face *face.DummyFace, engine *basic_engine.Engine, timer *basic_engine.DummyTimer) {
		hitCnt := 0

		spec := engine.Spec()
		name1 := tu.NoErr(enc.NameFromStr("/not"))
		name2 := tu.NoErr(enc.NameFromStr("/not/important"))
		config1 := &ndn.InterestConfig{
			Lifetime:    optional.Some(5 * time.Millisecond),
			CanBePrefix: false,
		}
		config2 := &ndn.InterestConfig{
			Lifetime:    optional.Some(5 * time.Millisecond),
			CanBePrefix: true,
		}
		interest1, err := spec.MakeInterest(name1, config1, nil, nil)
		require.NoError(t, err)
		interest2, err := spec.MakeInterest(name1, config2, nil, nil)
		require.NoError(t, err)
		interest3, err := spec.MakeInterest(name2, config1, nil, nil)
		require.NoError(t, err)

		dataWire := []byte("\x06\x1d\x07\x10\x08\x03not\x08\timportant\x14\x03\x18\x01\x00\x15\x04test")

		err = engine.Express(interest1, func(args ndn.ExpressCallbackArgs) {
			hitCnt += 1
			require.Equal(t, ndn.InterestResultTimeout, args.Result)
		})
		require.NoError(t, err)

		err = engine.Express(interest2, func(args ndn.ExpressCallbackArgs) {
			hitCnt += 1
			require.Equal(t, ndn.InterestResultData, args.Result)
			require.True(t, args.Data.Name().Equal(name2))
			require.Equal(t, []byte("test"), args.Data.Content().Join())
			require.Equal(t, dataWire, args.RawData.Join())
		})
		require.NoError(t, err)

		err = engine.Express(interest3, func(args ndn.ExpressCallbackArgs) {
			hitCnt += 1
			require.Equal(t, ndn.InterestResultData, args.Result)
			require.True(t, args.Data.Name().Equal(name2))
			require.Equal(t, []byte("test"), args.Data.Content().Join())
			require.Equal(t, dataWire, args.RawData.Join())
		})
		require.NoError(t, err)

		buf := tu.NoErr(face.Consume())
		require.Equal(t, enc.Buffer("\x05\x0a\x07\x05\x08\x03not\x0c\x01\x05"), buf)
		buf = tu.NoErr(face.Consume())
		require.Equal(t, enc.Buffer("\x05\x0c\x07\x05\x08\x03not\x21\x00\x0c\x01\x05"), buf)
		buf = tu.NoErr(face.Consume())
		require.Equal(t, enc.Buffer("\x05\x15\x07\x10\x08\x03not\x08\timportant\x0c\x01\x05"), buf)

		timer.MoveForward(4 * time.Millisecond)
		require.NoError(t, face.FeedPacket(dataWire))
		require.Equal(t, 2, hitCnt)
		timer.MoveForward(1 * time.Second)
		require.Equal(t, 3, hitCnt)
	})
}

// Tests the handling of NDN Interests with implicit SHA-256 digest components by verifying timeout behavior for an invalid digest and successful data retrieval for a valid digest.
func TestImplicitSha256(t *testing.T) {
	executeTest(t, func(face *face.DummyFace, engine *basic_engine.Engine, timer *basic_engine.DummyTimer) {
		hitCnt := 0

		spec := engine.Spec()
		name1 := tu.NoErr(enc.NameFromStr(
			"/test/sha256digest=FFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF"))
		name2 := tu.NoErr(enc.NameFromStr(
			"/test/sha256digest=5488f2c11b566d49e9904fb52aa6f6f9e66a954168109ce156eea2c92c57e4c2"))
		config := &ndn.InterestConfig{
			Lifetime: optional.Some(5 * time.Millisecond),
		}
		interest1, err := spec.MakeInterest(name1, config, nil, nil)
		require.NoError(t, err)
		interest2, err := spec.MakeInterest(name2, config, nil, nil)
		require.NoError(t, err)

		err = engine.Express(interest1, func(args ndn.ExpressCallbackArgs) {
			hitCnt += 1
			require.Equal(t, ndn.InterestResultTimeout, args.Result)
		})
		require.NoError(t, err)
		err = engine.Express(interest2, func(args ndn.ExpressCallbackArgs) {
			hitCnt += 1
			require.Equal(t, ndn.InterestResultData, args.Result)
			require.True(t, args.Data.Name().Equal(tu.NoErr(enc.NameFromStr("/test"))))
			require.Equal(t, []byte("test"), args.Data.Content().Join())
		})
		require.NoError(t, err)

		buf := tu.NoErr(face.Consume())
		require.Equal(t, enc.Buffer(
			"\x05\x2d\x07\x28\x08\x04test\x01\x20"+
				"\xff\xff\xff\xff\xff\xff\xff\xff\xff\xff\xff\xff\xff\xff\xff\xff"+
				"\xff\xff\xff\xff\xff\xff\xff\xff\xff\xff\xff\xff\xff\xff\xff\xff"+
				"\x0c\x01\x05",
		), buf)
		buf = tu.NoErr(face.Consume())
		require.Equal(t, enc.Buffer(
			"\x05\x2d\x07\x28\x08\x04test\x01\x20"+
				"\x54\x88\xf2\xc1\x1b\x56\x6d\x49\xe9\x90\x4f\xb5\x2a\xa6\xf6\xf9"+
				"\xe6\x6a\x95\x41\x68\x10\x9c\xe1\x56\xee\xa2\xc9\x2c\x57\xe4\xc2"+
				"\x0c\x01\x05",
		), buf)

		timer.MoveForward(4 * time.Millisecond)
		require.NoError(t, face.FeedPacket(
			enc.Buffer("\x06\x13\x07\x06\x08\x04test\x14\x03\x18\x01\x00\x15\x04test"),
		))
		require.Equal(t, 1, hitCnt)
		timer.MoveForward(1 * time.Second)
		require.Equal(t, 2, hitCnt)
	})
}

// No need to test AppParam for expression. If `spec.MakeInterest` works, `engine.Express` will.

// This function tests that an Interest routed to a registered handler produces a correctly formatted Data packet with a Blob content type and test signature when processed by the NDN engine.
func TestRoute(t *testing.T) {
	executeTest(t, func(face *face.DummyFace, engine *basic_engine.Engine, timer *basic_engine.DummyTimer) {
		hitCnt := 0
		spec := engine.Spec()

		handler := func(args ndn.InterestHandlerArgs) {
			hitCnt += 1
			require.Equal(t, []byte(
				"\x05\x15\x07\x10\x08\x03not\x08\timportant\x0c\x01\x05",
			), args.RawInterest.Join())
			require.True(t, args.Interest.Signature().SigType() == ndn.SignatureNone)
			data, err := spec.MakeData(
				args.Interest.Name(),
				&ndn.DataConfig{
					ContentType: optional.Some(ndn.ContentTypeBlob),
				},
				enc.Wire{[]byte("test")},
				sig.NewTestSigner(enc.Name{}, 0))
			require.NoError(t, err)
			args.Reply(data.Wire)
		}

		prefix := tu.NoErr(enc.NameFromStr("/not"))
		engine.AttachHandler(prefix, handler)
		face.FeedPacket([]byte("\x05\x15\x07\x10\x08\x03not\x08\timportant\x0c\x01\x05"))
		require.Equal(t, 1, hitCnt)
		buf := tu.NoErr(face.Consume())
		require.Equal(t, enc.Buffer(
			"\x06\x22\x07\x10\x08\x03not\x08\timportant\x14\x03\x18\x01\x00\x15\x04test"+
				"\x16\x03\x1b\x01\xc8",
		), buf)
	})
}

// Constructs and processes an Interest packet for "/not/important", generating a signed Data packet with "test" content and verifying PIT token handling via a dummy face and engine.
func TestPitToken(t *testing.T) {
	executeTest(t, func(face *face.DummyFace, engine *basic_engine.Engine, timer *basic_engine.DummyTimer) {
		hitCnt := 0
		spec := engine.Spec()

		handler := func(args ndn.InterestHandlerArgs) {
			hitCnt += 1
			data, err := spec.MakeData(
				args.Interest.Name(),
				&ndn.DataConfig{
					ContentType: optional.Some(ndn.ContentTypeBlob),
				},
				enc.Wire{[]byte("test")},
				sig.NewTestSigner(enc.Name{}, 0))
			require.NoError(t, err)
			args.Reply(data.Wire)
		}

		prefix := tu.NoErr(enc.NameFromStr("/not"))
		engine.AttachHandler(prefix, handler)
		face.FeedPacket([]byte(
			"\x64\x1f\x62\x04\x01\x02\x03\x04\x50\x17" +
				"\x05\x15\x07\x10\x08\x03not\x08\timportant\x0c\x01\x05",
		))
		buf := tu.NoErr(face.Consume())
		require.Equal(t, enc.Buffer(
			"\x64\x2c\x62\x04\x01\x02\x03\x04\x50\x24"+
				"\x06\x22\x07\x10\x08\x03not\x08\timportant\x14\x03\x18\x01\x00\x15\x04test"+
				"\x16\x03\x1b\x01\xc8",
		), buf)
	})
}
