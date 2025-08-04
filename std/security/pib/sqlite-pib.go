package sqlitepib

import (
	"database/sql"

	_ "github.com/mattn/go-sqlite3"
	enc "github.com/named-data/ndnd/std/encoding"
	"github.com/named-data/ndnd/std/log"
	"github.com/named-data/ndnd/std/ndn"
	spec "github.com/named-data/ndnd/std/ndn/spec_2022"
)

type SqliteCert struct {
	pib        *SqlitePib
	rowId      uint
	name       enc.Name
	certBits   []byte
	isDefault  bool
	keyLocator enc.Name
}

type SqliteKey struct {
	pib       *SqlitePib
	rowId     uint
	name      enc.Name
	keyBits   []byte
	isDefault bool
}

type SqliteIdent struct {
	pib       *SqlitePib
	rowId     uint
	name      enc.Name
	isDefault bool
}

type SqlitePib struct {
	db  *sql.DB
	tpm Tpm
}

// Returns a string representation of the SqlitePib object, which is the identifier "sqlite-pib".
func (pib *SqlitePib) String() string {
	return "sqlite-pib"
}

// Returns the TPM instance associated with this SqlitePib.
func (pib *SqlitePib) Tpm() Tpm {
	return pib.tpm
}

// Retrieves an identity with the specified name from the SQLite-based persistence information base (PIB), returning an Identity object if found.
func (pib *SqlitePib) GetIdentity(name enc.Name) Identity {
	nameWire := name.Bytes()
	rows, err := pib.db.Query("SELECT id, is_default FROM identities WHERE identity=?", nameWire)
	if err != nil {
		return nil
	}
	defer rows.Close()
	if !rows.Next() {
		return nil
	}
	ret := &SqliteIdent{
		pib:  pib,
		name: name,
	}
	err = rows.Scan(&ret.rowId, &ret.isDefault)
	if err != nil {
		return nil
	}
	return ret
}

// Retrieves a key from the SQLite-based PIB (Public Information Base) by its name, returning a Key object if found or nil if the key does not exist or an error occurs.
func (pib *SqlitePib) GetKey(keyName enc.Name) Key {
	nameWire := keyName.Bytes()
	rows, err := pib.db.Query("SELECT id, key_bits, is_default FROM keys WHERE key_name=?", nameWire)
	if err != nil {
		return nil
	}
	defer rows.Close()
	if !rows.Next() {
		return nil
	}
	ret := &SqliteKey{
		pib:  pib,
		name: keyName,
	}
	err = rows.Scan(&ret.rowId, &ret.keyBits, &ret.isDefault)
	if err != nil {
		return nil
	}
	return ret
}

// Retrieves a certificate with the specified name from the SQLite-based Persistent Identity Bundle (PIB), parses its data, and returns a Cert object containing its metadata and key locator information if it exists.
func (pib *SqlitePib) GetCert(certName enc.Name) Cert {
	nameWire := certName.Bytes()
	rows, err := pib.db.Query(
		"SELECT id, certificate_data, is_default FROM certificates WHERE certificate_name=?",
		nameWire,
	)
	if err != nil {
		return nil
	}
	defer rows.Close()
	if !rows.Next() {
		return nil
	}
	ret := &SqliteCert{
		pib:  pib,
		name: certName,
	}
	err = rows.Scan(&ret.rowId, &ret.certBits, &ret.isDefault)
	if err != nil {
		return nil
	}
	// Parse the certificate and get the signer
	data, _, err := spec.Spec{}.ReadData(enc.NewBufferView(ret.certBits))
	if err != nil || data.Signature() == nil {
		return nil
	}
	ret.keyLocator = data.Signature().KeyName()
	return ret
}

// Returns a signer for the specified certificate name by retrieving the corresponding key from the TPM, or nil if the certificate name is invalid.
func (pib *SqlitePib) GetSignerForCert(certName enc.Name) ndn.Signer {
	l := len(certName)
	if l < 2 {
		return nil
	}
	return pib.tpm.GetSigner(certName[:l-2], certName)
}

// Returns the name of the identity stored in the SqliteIdent instance.
func (iden *SqliteIdent) Name() enc.Name {
	return iden.name
}

// Retrieves a cryptographic key with the given name from the identity's PIB (Private Information Base).
func (iden *SqliteIdent) GetKey(keyName enc.Name) Key {
	return iden.pib.GetKey(keyName)
}

// Searches for a certificate associated with this identity's keys that satisfies the provided predicate function, returning the first matching certificate found.
func (iden *SqliteIdent) FindCert(check func(Cert) bool) Cert {
	rows, err := iden.pib.db.Query(
		"SELECT id, key_name, key_bits, is_default FROM keys WHERE identity_id=?",
		iden.rowId,
	)
	if err != nil {
		return nil
	}
	defer rows.Close()
	for rows.Next() {
		ret := &SqliteKey{
			pib: iden.pib,
		}
		var keyNameWire []byte
		err = rows.Scan(&ret.rowId, &keyNameWire, &ret.keyBits, &ret.isDefault)
		if err != nil {
			continue
		}
		ret.name, err = enc.NameFromBytes(keyNameWire)
		if err != nil {
			continue
		}
		cert := ret.FindCert(check)
		if cert != nil {
			return cert
		}
	}
	return nil

}

// Returns the name of the certificate as an `enc.Name`.
func (cert *SqliteCert) Name() enc.Name {
	return cert.name
}

// **Description:**  
Returns the key locator `Name` associated with this certificate, indicating the location of the cryptographic key used for validation.
func (cert *SqliteCert) KeyLocator() enc.Name {
	return cert.keyLocator
}

// Returns the key associated with this certificate by removing the last two components of the certificate's name and retrieving the corresponding key from the PIB.
func (cert *SqliteCert) Key() Key {
	l := len(cert.name)
	if l < 2 {
		return nil
	}
	return cert.pib.GetKey(cert.name[:l-2])
}

// Returns the raw certificate data as a byte slice.
func (cert *SqliteCert) Data() []byte {
	return cert.certBits
}

// Returns the NDN Signer associated with this certificate by retrieving it from the PIB using the certificate's name.
func (cert *SqliteCert) AsSigner() ndn.Signer {
	return cert.pib.GetSignerForCert(cert.name)
}

// Returns the name of the key as an `enc.Name`.
func (key *SqliteKey) Name() enc.Name {
	return key.name
}

// Returns the Identity associated with this key by truncating the last two characters of the key's name and retrieving the corresponding identity from the PIB.
func (key *SqliteKey) Identity() Identity {
	l := len(key.name)
	if l < 2 {
		return nil
	}
	return key.pib.GetIdentity(key.name[:l-2])
}

// Returns the cryptographic key bits stored in the SqliteKey instance as a byte slice.
func (key *SqliteKey) KeyBits() []byte {
	return key.keyBits
}

// Returns the self-signed certificate associated with this key by checking if the certificate's name contains a "self" component in the second-to-last position.
func (key *SqliteKey) SelfSignedCert() Cert {
	return key.FindCert(func(cert Cert) bool {
		l := len(cert.Name())
		selfComp := enc.NewGenericComponent("self")
		return l > 2 && cert.Name()[l-2].Equal(selfComp)
	})
}

// Retrieves the certificate with the given name from the Public Information Base (PIB) managed by this key.
func (key *SqliteKey) GetCert(certName enc.Name) Cert {
	return key.pib.GetCert(certName)
}

// Finds and returns a certificate associated with this key that satisfies the provided check function, or returns nil if no matching certificate is found.
func (key *SqliteKey) FindCert(check func(Cert) bool) Cert {
	rows, err := key.pib.db.Query(
		"SELECT id, certificate_name, certificate_data, is_default FROM certificates WHERE key_id=?",
		key.rowId,
	)
	if err != nil {
		return nil
	}
	defer rows.Close()
	for rows.Next() {
		ret := &SqliteCert{
			pib: key.pib,
		}
		var certNameWire []byte
		err = rows.Scan(&ret.rowId, &certNameWire, &ret.certBits, &ret.isDefault)
		if err != nil {
			continue
		}
		ret.name, err = enc.NameFromBytes(certNameWire)
		if err != nil {
			continue
		}
		// Parse the certificate and get the signer
		data, _, err := spec.Spec{}.ReadData(enc.NewBufferView(ret.certBits))
		if err != nil || data.Signature() == nil {
			continue
		}
		ret.keyLocator = data.Signature().KeyName()
		if check(ret) {
			return ret
		}
	}
	return nil
}

// Constructs a new SqlitePib instance using the specified SQLite database path and TPM for cryptographic operations.
func NewSqlitePib(path string, tpm Tpm) *SqlitePib {
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		log.Error(nil, "Unable to connect to sqlite PIB", "err", err)
		return nil
	}
	return &SqlitePib{
		db:  db,
		tpm: tpm,
	}
}
