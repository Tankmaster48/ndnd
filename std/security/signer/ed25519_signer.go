package signer

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/x509"
	"fmt"

	enc "github.com/named-data/ndnd/std/encoding"
	"github.com/named-data/ndnd/std/ndn"
)

// ed25519Signer is a signer that uses Ed25519 key to sign packets.
type ed25519Signer struct {
	name enc.Name
	key  ed25519.PrivateKey
}

// Returns the Ed25519 signature type for the signer.
func (s *ed25519Signer) Type() ndn.SigType {
	return ndn.SignatureEd25519
}

// Returns the name of the Ed25519 key associated with this signer.
func (s *ed25519Signer) KeyName() enc.Name {
	return s.name
}

// Returns the NDN name of the Ed25519 key used as a KeyLocator for signed packets.
func (s *ed25519Signer) KeyLocator() enc.Name {
	return s.name
}

// Returns the size in bytes of the Ed25519 signature produced by this signer.
func (s *ed25519Signer) EstimateSize() uint {
	return ed25519.SignatureSize
}

// Signs the concatenated wire-encoded data using the Ed25519 private key, returning the digital signature.
func (s *ed25519Signer) Sign(covered enc.Wire) ([]byte, error) {
	return ed25519.Sign(s.key, covered.Join()), nil
}

// Returns the public key associated with the ED25519 signer, encoded in PKIX/X.509 format as a byte slice.
func (s *ed25519Signer) Public() ([]byte, error) {
	return x509.MarshalPKIXPublicKey(s.key.Public())
}

// Returns the Ed25519 private key in PKCS#8 encoding for cryptographic operations.
func (s *ed25519Signer) Secret() ([]byte, error) {
	return x509.MarshalPKCS8PrivateKey(s.key)
}

// NewEd25519Signer creates a signer using ed25519 key
func NewEd25519Signer(name enc.Name, key ed25519.PrivateKey) ndn.Signer {
	return &ed25519Signer{name, key}
}

// Ed25519Keygen creates a signer using a new Ed25519 key
func KeygenEd25519(name enc.Name) (ndn.Signer, error) {
	_, sk, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, err
	}
	return NewEd25519Signer(name, sk), nil
}

// ParseEd25519 parses a signer from a byte slice.
func ParseEd25519(name enc.Name, key []byte) (ndn.Signer, error) {
	pkey, err := x509.ParsePKCS8PrivateKey(key)
	if err != nil {
		return nil, err
	}
	sk, ok := pkey.(ed25519.PrivateKey)
	if !ok {
		return nil, fmt.Errorf("invalid key type")
	}
	return NewEd25519Signer(name, sk), nil
}

// validateEd25519 verifies the signature with a known ed25519 public key.
// ndn-cxx's PIB does not support this, but a certificate is supposed to use ASN.1 DER format.
// Use x509.ParsePKIXPublicKey to parse. Note: ed25519.PublicKey is defined to be a pointer type without '*'.
func validateEd25519(sigCovered enc.Wire, sig ndn.Signature, pubKey ed25519.PublicKey) bool {
	if sig.SigType() != ndn.SignatureEd25519 {
		return false
	}
	return ed25519.Verify(pubKey, sigCovered.Join(), sig.SigValue())
}
