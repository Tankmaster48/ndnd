package trust_schema

import (
	enc "github.com/named-data/ndnd/std/encoding"
	"github.com/named-data/ndnd/std/ndn"
	"github.com/named-data/ndnd/std/security/signer"
)

// NullSchema is a trust schema that allows everything.
type NullSchema struct{}

// Constructs a new NullSchema instance representing an empty schema with no validation or structure enforcement.
func NewNullSchema() *NullSchema {
	return &NullSchema{}
}

// Permits any packet and certificate names by always returning true, indicating no validation is enforced.
func (*NullSchema) Check(pkt enc.Name, cert enc.Name) bool {
	return true
}

// Returns a SHA-256 signer that ignores the provided name and keychain parameters.
func (*NullSchema) Suggest(enc.Name, ndn.KeyChain) ndn.Signer {
	return signer.NewSha256Signer()
}
