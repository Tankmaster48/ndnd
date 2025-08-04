package signer

import (
	"crypto/hmac"
	"crypto/sha256"

	enc "github.com/named-data/ndnd/std/encoding"
	"github.com/named-data/ndnd/std/ndn"
)

// hmacSigner is a Data signer that uses a provided HMAC key.
type hmacSigner struct {
	key []byte
}

// Returns the signature type HMAC with SHA-256 used by this signer when generating or verifying signatures.
func (signer *hmacSigner) Type() ndn.SigType {
	return ndn.SignatureHmacWithSha256
}

// Returns nil, indicating that the HMAC signer does not have a key name associated with it, as HMAC uses symmetric keys rather than named NDN keys.
func (*hmacSigner) KeyName() enc.Name {
	return nil
}

// Returns nil, indicating that the HMAC signing mechanism does not require a key locator as it uses symmetric key authentication.
func (*hmacSigner) KeyLocator() enc.Name {
	return nil
}

// Returns the estimated size (in bytes) of the HMAC signature produced by this signer, which is fixed at 32 bytes for SHA-256-based signatures.
func (*hmacSigner) EstimateSize() uint {
	return 32
}

// Generates an HMAC-SHA256 signature over the concatenation of all buffers in the provided `enc.Wire` slice using the signer's key, returning the resulting signature bytes or an error.
func (signer *hmacSigner) Sign(covered enc.Wire) ([]byte, error) {
	mac := hmac.New(sha256.New, signer.key)
	for _, buf := range covered {
		_, err := mac.Write(buf)
		if err != nil {
			return nil, enc.ErrUnexpected{Err: err}
		}
	}
	return mac.Sum(nil), nil
}

// Returns nil and an error indicating no public key exists, as HMAC signing uses symmetric keys and does not involve public key cryptography.
func (*hmacSigner) Public() ([]byte, error) {
	return nil, ndn.ErrNoPubKey
}

// NewHmacSigner creates a Data signer that uses DigestSha256.
func NewHmacSigner(key []byte) ndn.Signer {
	return &hmacSigner{key}
}

// ValidateHmac verifies the signature with a known HMAC shared key.
func ValidateHmac(sigCovered enc.Wire, sig ndn.Signature, key []byte) bool {
	if sig.SigType() != ndn.SignatureHmacWithSha256 {
		return false
	}
	mac := hmac.New(sha256.New, []byte(key))
	for _, buf := range sigCovered {
		_, err := mac.Write(buf)
		if err != nil {
			return false
		}
	}
	return hmac.Equal(mac.Sum(nil), sig.SigValue())
}
