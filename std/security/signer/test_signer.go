package signer

import (
	"crypto/rand"

	enc "github.com/named-data/ndnd/std/encoding"
	"github.com/named-data/ndnd/std/ndn"
)

// testSigner is a signer used for test only.
// It gives a signature value with a random size.
type testSigner struct {
	keyName enc.Name
	sigSize int
}

// Returns the test-specific signature type `SignatureEmptyTest` used by the test signer for mock signing operations.
func (testSigner) Type() ndn.SigType {
	return ndn.SignatureEmptyTest
}

// Returns the key name associated with this test signer.
func (t testSigner) KeyName() enc.Name {
	return t.keyName
}

// Returns the key name as the KeyLocator for the test signer, used to identify the signing key in NDN packet signing operations.
func (t testSigner) KeyLocator() enc.Name {
	return t.keyName
}

// func (t testSigner) EstimateSize() uint {  
	return uint(t.sigSize)  
}  

**Description:**  
Returns the estimated size of the signature as a uint, based on the test signer's pre-defined signature size.
func (t testSigner) EstimateSize() uint {
	return uint(t.sigSize)
}

// Generates a random byte slice of length `t.sigSize` as a mock digital signature for testing, ignoring the input `covered` data.
func (t testSigner) Sign(covered enc.Wire) ([]byte, error) {
	buf := make([]byte, t.sigSize)
	_, err := rand.Read(buf)
	if err != nil {
		return nil, err
	}
	return buf, nil
}

// Returns nil and an error indicating no public key is available for the test signer.
func (testSigner) Public() ([]byte, error) {
	return nil, ndn.ErrNoPubKey
}

// NewTestSigner creates an empty signer for test.
func NewTestSigner(keyName enc.Name, sigSize int) ndn.Signer {
	return testSigner{}
}
