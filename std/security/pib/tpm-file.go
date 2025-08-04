package sqlitepib

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"os"
	"path"

	enc "github.com/named-data/ndnd/std/encoding"
	"github.com/named-data/ndnd/std/log"
	"github.com/named-data/ndnd/std/ndn"
)

type FileTpm struct {
	path string
}

// Returns a string representation of the FileTpm instance, including its associated file path in the format "file-tpm (path)".
func (tpm *FileTpm) String() string {
	return fmt.Sprintf("file-tpm (%s)", tpm.path)
}

// Generates a filename for a private key file by computing the SHA-256 hash of the input key name bytes, encoding it in hexadecimal, and appending the `.privkey` extension.
func (tpm *FileTpm) ToFileName(keyNameBytes []byte) string {
	h := sha256.New()
	h.Write(keyNameBytes)
	return hex.EncodeToString(h.Sum(nil)) + ".privkey"
}

// Retrieves and decodes a private key file to construct an NDN signer for the specified key name and key locator, supporting RSA or ECC key formats.
func (tpm *FileTpm) GetSigner(keyName enc.Name, keyLocatorName enc.Name) ndn.Signer {
	keyNameBytes := keyName.Bytes()
	fileName := path.Join(tpm.path, tpm.ToFileName(keyNameBytes))

	text, err := os.ReadFile(fileName)
	if err != nil {
		log.Error(tpm, "Unable to read private key file", "file", fileName, "err", err)
		return nil
	}

	blockLen := base64.StdEncoding.DecodedLen(len(text))
	block := make([]byte, blockLen)
	n, err := base64.StdEncoding.Decode(block, text)
	if err != nil {
		log.Error(tpm, "Unable to base64 decode private key file", "file", fileName, "err", err)
		return nil
	}
	block = block[:n]

	// There are only two formats: PKCS1 encoded RSA, or EC
	// eckbits, err := x509.ParseECPrivateKey(block)
	// if err == nil {
	// 	// ECC Key
	// 	// TODO: Handle for Interest
	// 	return sec.NewEccSigner(false, false, 0, eckbits, keyLocatorName)
	// }

	// rsabits, err := x509.ParsePKCS1PrivateKey(block)
	// if err == nil {
	// 	// RSA Key
	// 	// TODO: Handle for Interest
	// 	return sec.NewRsaSigner(false, false, 0, rsabits, keyLocatorName)
	// }

	log.Error(tpm, "Unrecognized private key format", "file", fileName)
	return nil
}

// Generates a cryptographic key of the specified type and size for the given key name in a file-based TPM implementation, returning the key material as an encoded buffer.
func (tpm *FileTpm) GenerateKey(keyName enc.Name, keyType string, keySize uint64) enc.Buffer {
	panic("not implemented")
}

// Checks if a key with the given name exists in the TPM's storage by converting the name to a filename and verifying the corresponding file's existence.
func (tpm *FileTpm) KeyExist(keyName enc.Name) bool {
	keyNameBytes := keyName.Bytes()
	fileName := path.Join(tpm.path, tpm.ToFileName(keyNameBytes))
	_, err := os.Stat(fileName)
	return err == nil
}

// "Deletes the key associated with the provided name from the TPM."
func (tpm *FileTpm) DeleteKey(keyName enc.Name) {
	panic("not implemented")
}

// Constructs a new FileTpm instance that uses the specified file path for TPM storage operations.
func NewFileTpm(path string) Tpm {
	return &FileTpm{
		path: path,
	}
}
