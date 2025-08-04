package signer_test

import (
	"crypto/ed25519"
	"crypto/x509"
	"testing"

	enc "github.com/named-data/ndnd/std/encoding"
	"github.com/named-data/ndnd/std/ndn"
	sig "github.com/named-data/ndnd/std/security/signer"
	tu "github.com/named-data/ndnd/std/utils/testutils"
	"github.com/stretchr/testify/require"
)

var TEST_KEY_NAME, _ = enc.NameFromStr("/KEY")

// Verifies the correctness of an Ed25519 signer's configuration and its ability to generate valid signatures by checking key properties, estimated size, and successful signature verification against provided data and public key.
func testEd25519Verify(t *testing.T, signer ndn.Signer, verifyKey []byte) bool {
	require.Equal(t, uint(ed25519.SignatureSize), signer.EstimateSize())
	require.Equal(t, ndn.SignatureEd25519, signer.Type())
	require.Equal(t, TEST_KEY_NAME, signer.KeyName())

	dataVal := enc.Wire{
		[]byte("\x07\x14\x08\x05local\x08\x03ndn\x08\x06prefix"),
		[]byte("\x14\x03\x18\x01\x00"),
	}
	sigValue := tu.NoErr(signer.Sign(dataVal))

	// For basic test, we use ed25519.Verify to verify the signature.
	verifyKeyBits := tu.NoErr(x509.ParsePKIXPublicKey(verifyKey)).(ed25519.PublicKey)
	return ed25519.Verify(verifyKeyBits, dataVal.Join(), sigValue)
}

// Constructs an Ed25519 signer using a predefined seed key and verifies its ability to generate a valid public key and signing capability.
func TestEd25519SignerNew(t *testing.T) {
	tu.SetT(t)

	edkeybits := ed25519.NewKeyFromSeed([]byte("01234567890123456789012345678901"))
	signer := sig.NewEd25519Signer(TEST_KEY_NAME, edkeybits)
	pub := tu.NoErr(signer.Public())
	require.True(t, testEd25519Verify(t, signer, pub))
}

// Tests the Ed25519 key generation function by creating two distinct key pairs, verifying each can sign and validate with their own public keys, and ensuring they cannot validate each other's signatures.
func TestEd25519Keygen(t *testing.T) {
	tu.SetT(t)

	signer1 := tu.NoErr(sig.KeygenEd25519(TEST_KEY_NAME))
	pub1 := tu.NoErr(signer1.Public())
	require.True(t, testEd25519Verify(t, signer1, pub1))

	signer2 := tu.NoErr(sig.KeygenEd25519(TEST_KEY_NAME))
	pub2 := tu.NoErr(signer2.Public())
	require.True(t, testEd25519Verify(t, signer2, pub2))

	// Check that the two signers are different.
	require.False(t, testEd25519Verify(t, signer2, pub1))
}

// **Function Description:**  
Tests parsing of Ed25519 signers from a secret key, verifies the parsed signer matches the original, and ensures parsing fails when using a public key.
func TestEd25519Parse(t *testing.T) {
	tu.SetT(t)

	edkeybits := ed25519.NewKeyFromSeed([]byte("01234567890123456789012345678901"))
	signer1 := sig.NewEd25519Signer(TEST_KEY_NAME, edkeybits)

	secret := tu.NoErr(sig.GetSecret(signer1))
	signer2 := tu.NoErr(sig.ParseEd25519(TEST_KEY_NAME, secret))

	// Check that the two signers are the same.
	pub1 := tu.NoErr(signer1.Public())
	require.True(t, testEd25519Verify(t, signer2, pub1))

	// Make sure parse fails with the public key.
	pub2 := tu.NoErr(signer1.Public())
	_, err := sig.ParseEd25519(TEST_KEY_NAME, pub2)
	require.Error(t, err)
}
