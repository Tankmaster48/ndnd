package ndncert

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/binary"
	"io"
	"math"

	"github.com/named-data/ndnd/std/security/ndncert/tlv"
)

const AeadSizeNonce = 12
const AeadSizeTag = 16
const AeadSizeRand = 8

type AeadMessage struct {
	IV         [AeadSizeNonce]byte
	AuthTag    [AeadSizeTag]byte
	CipherText []byte
}

// Constructs a TLV-encoded CipherMsg from the AEAD message, including the initialization vector, authentication tag, and cipher text.
func (m *AeadMessage) TLV() *tlv.CipherMsg {
	return &tlv.CipherMsg{
		InitVec:  m.IV[:],
		AuthNTag: m.AuthTag[:],
		Payload:  m.CipherText,
	}
}

// Initializer for AEAD message using parameters from a TLV cipher structure.  

**Description:**  
Initializes the AEAD message with the provided TLV cipher message's initialization vector, authentication tag, and encrypted payload.  

**Function Signature:**  
```go
func (m *AeadMessage) FromTLV(t *tlv.CipherMsg)
```  

**Semantics:**  
- Converts the TLV `InitVec` and `AuthNTag` fields into fixed-size byte arrays (`[AeadSizeNonce]byte` and `[AeadSizeTag]byte`, respectively).  
- Assigns the `Payload` field of the TLV structure to the `CipherText` field of the AEAD message.  
- Assumes the input TLV structure is properly formatted and does not perform validation.  

**Example Use Case:**  
Reconstructing an AEAD-encrypted message from raw TLV-encoded data.
func (m *AeadMessage) FromTLV(t *tlv.CipherMsg) {
	m.IV = [AeadSizeNonce]byte(t.InitVec)
	m.AuthTag = [AeadSizeTag]byte(t.AuthNTag)
	m.CipherText = t.Payload
}

type AeadCounter struct {
	block  uint32
	random [AeadSizeRand]byte
}

// Constructs a new AEAD counter instance initialized with a cryptographically secure random nonce and a zero-based block counter for encryption operations.
func NewAeadCounter() *AeadCounter {
	randomBytes := make([]byte, AeadSizeRand)
	if _, randReadErr := io.ReadFull(rand.Reader, randomBytes); randReadErr != nil {
		panic(randReadErr.Error())
	}
	return &AeadCounter{
		block:  0,
		random: [AeadSizeRand]byte(randomBytes),
	}
}

// Encrypts plaintext using AES-GCM with the provided key and additional authenticated data, generating an IV from the counter and returning an AEAD message containing the ciphertext, authentication tag, and IV.
func AeadEncrypt(
	key [AeadSizeTag]byte,
	plaintext []byte,
	info []byte,
	counter *AeadCounter,
) (*AeadMessage, error) {
	block, err := aes.NewCipher(key[:])
	if err != nil {
		return nil, err
	}

	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	// Make initialization vector
	counter.block += uint32(math.Ceil(float64(float32(len(plaintext)) / float32(AeadSizeTag))))
	cblock := make([]byte, 4)
	binary.LittleEndian.PutUint32(cblock, counter.block)
	iv := append(counter.random[:], cblock...)

	// Encrypt and seal
	output := aesgcm.Seal(nil, iv, plaintext, info)

	return &AeadMessage{
		IV:         [AeadSizeNonce]byte(iv),
		AuthTag:    ([AeadSizeTag]byte)(output[len(plaintext):]),
		CipherText: output[:len(plaintext)],
	}, nil
}

// Decrypts an AES-GCM encrypted message using the provided key, message's IV and ciphertext, and additional authentication data (info), returning the plaintext or an error if decryption fails.
func AeadDecrypt(
	key [AeadSizeTag]byte,
	message AeadMessage,
	info []byte,
) ([]byte, error) {
	block, err := aes.NewCipher(key[:])
	if err != nil {
		return nil, err
	}

	nonce := message.IV[:]
	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	ciphertext := append(message.CipherText, message.AuthTag[:]...)

	plaintext, err := aesgcm.Open(nil, nonce, ciphertext, info[:])
	if err != nil {
		return nil, err
	}

	return plaintext, nil
}
