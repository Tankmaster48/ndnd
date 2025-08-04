package keychain

import (
	enc "github.com/named-data/ndnd/std/encoding"
	"github.com/named-data/ndnd/std/ndn"
	spec "github.com/named-data/ndnd/std/ndn/spec_2022"
	sec "github.com/named-data/ndnd/std/security"
)

// KeyChainMem is an in-memory keychain.
type KeyChainMem struct {
	identities []ndn.KeyChainIdentity
	certNames  []enc.Name
	pubStore   ndn.Store
}

// NewKeyChainMem creates a new in-memory keychain.
func NewKeyChainMem(pubStore ndn.Store) ndn.KeyChain {
	return &KeyChainMem{
		identities: make([]ndn.KeyChainIdentity, 0),
		certNames:  make([]enc.Name, 0),
		pubStore:   pubStore,
	}
}

// Returns the string representation "keychain-mem" for the KeyChainMem instance when converted to a string.
func (kc *KeyChainMem) String() string {
	return "keychain-mem"
}

// Returns the public key store managed by the key chain for accessing and managing cryptographic keys.
func (kc *KeyChainMem) Store() ndn.Store {
	return kc.pubStore
}

// Returns the slice of identities managed by the key chain.
func (kc *KeyChainMem) Identities() []ndn.KeyChainIdentity {
	return kc.identities
}

// Returns the KeyChainIdentity with the specified name if it exists in the key chain, otherwise returns nil.
func (kc *KeyChainMem) IdentityByName(name enc.Name) ndn.KeyChainIdentity {
	for _, id := range kc.identities {
		if id.Name().Equal(name) {
			return id
		}
	}
	return nil
}

// Inserts a cryptographic signer into the key chain under its associated identity, attaching any existing certificates and ensuring no duplicate keys are added.
func (kc *KeyChainMem) InsertKey(signer ndn.Signer) error {
	// Get key name
	keyName := signer.KeyName()
	idName, err := sec.GetIdentityFromKeyName(keyName)
	if err != nil {
		return err
	}

	// Check if signer already exists
	idObj, _ := kc.IdentityByName(idName).(*keyChainIdentity)
	if idObj != nil {
		for _, key := range idObj.Keys() {
			if key.KeyName().Equal(keyName) {
				return nil // not an error
			}
		}
	} else {
		// Create new identity if not exists
		idObj = &keyChainIdentity{name: idName}
		kc.identities = append(kc.identities, idObj)
	}

	// Attach any existing certificates to the signer
	key := &keyChainKey{signer: signer}
	for _, certName := range kc.certNames {
		if keyName.IsPrefix(certName) {
			key.insertCert(certName)
		}
	}

	// Insert signer to identity
	idObj.keyList = append(idObj.keyList, key)
	idObj.sort()

	return nil
}

// Inserts a key certificate into the key chain after validating its content type, name structure, and expiration, and updates associated identities.
func (kc *KeyChainMem) InsertCert(wire []byte) error {
	data, _, err := spec.Spec{}.ReadData(enc.NewBufferView(wire))
	if err != nil {
		return err
	}

	contentType, ok := data.ContentType().Get()
	if !ok || contentType != ndn.ContentTypeKey {
		return ndn.ErrInvalidValue{Item: "content type"}
	}

	// /<IdentityName>/KEY/<KeyId>/<IssuerId>/<Version>
	name := data.Name()
	if !name.At(-4).IsGeneric("KEY") {
		return ndn.ErrInvalidValue{Item: "KEY component"}
	}
	if !name.At(-1).IsVersion() {
		return ndn.ErrInvalidValue{Item: "version component"}
	}

	// Check if certificate is valid
	if sec.CertIsExpired(data) {
		return ndn.ErrInvalidValue{Item: "certificate expiry"}
	}

	// Check if certificate already exists
	for _, existing := range kc.certNames {
		if existing.Equal(name) {
			return nil // not an error
		}
	}
	kc.certNames = append(kc.certNames, name)

	// Insert certificate to public store
	if err := kc.pubStore.Put(name, wire); err != nil {
		return err
	}

	// Update identities with the new certificate
	for _, id := range kc.identities {
		id.(*keyChainIdentity).insertCert(name)
	}

	return nil
}
