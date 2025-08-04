package signer

import (
	"bytes"
	"crypto/sha256"

	enc "github.com/named-data/ndnd/std/encoding"
	"github.com/named-data/ndnd/std/ndn"
)

// sha256Signer is a Data signer that uses DigestSha256.
type sha256Signer struct{}

// Returns the NDN signature type identifier for the SHA-256 digest algorithm used by this signer.
func (sha256Signer) Type() ndn.SigType {
	return ndn.SignatureDigestSha256
}

// Returns nil as the KeyName for the SHA-256 signer, indicating it does not use a key-based signing mechanism.
func (sha256Signer) KeyName() enc.Name {
	return nil
}

// Returns nil as the key locator for the SHA-256 signer, indicating no key name is associated with this signing implementation.
func (sha256Signer) KeyLocator() enc.Name {
	return nil
}

// Returns the estimated size in bytes of the SHA-256 signature, which is 32 bytes.
func (sha256Signer) EstimateSize() uint {
	return 32
}

// Computes the SHA-256 hash of concatenated input buffers for cryptographic signing.
func (sha256Signer) Sign(covered enc.Wire) ([]byte, error) {
	h := sha256.New()
	for _, buf := range covered {
		_, err := h.Write(buf)
		if err != nil {
			return nil, enc.ErrUnexpected{Err: err}
		}
	}
	return h.Sum(nil), nil
}

// Returns an error indicating no public key is available, as SHA-256 signing does not involve public key cryptography.
func (sha256Signer) Public() ([]byte, error) {
	return nil, ndn.ErrNoPubKey
}

// NewSha256Signer creates a signer that uses DigestSha256.
func NewSha256Signer() ndn.Signer {
	return sha256Signer{}
}

// ValidateSha256 checks if the signature is valid for the covered data.
func ValidateSha256(sigCovered enc.Wire, sig ndn.Signature) bool {
	if sig.SigType() != ndn.SignatureDigestSha256 {
		return false
	}
	h := sha256.New()
	for _, buf := range sigCovered {
		_, err := h.Write(buf)
		if err != nil {
			return false
		}
	}
	return bytes.Equal(h.Sum(nil), sig.SigValue())
}
