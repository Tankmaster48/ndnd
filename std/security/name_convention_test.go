package security_test

import (
	"testing"

	enc "github.com/named-data/ndnd/std/encoding"
	sec "github.com/named-data/ndnd/std/security"
	tu "github.com/named-data/ndnd/std/utils/testutils"
	"github.com/stretchr/testify/require"
)

// Verifies that a key name is correctly generated from an identity name by appending a unique key identifier, ensuring it extends the expected "KEY" prefix and that the original identity can be accurately extracted from the key name.
func TestKeyName(t *testing.T) {
	tu.SetT(t)

	id, _ := enc.NameFromStr("/my/test/identity")
	keyPfx, _ := enc.NameFromStr("/my/test/identity/KEY")

	keyName := sec.MakeKeyName(id)
	require.True(t, keyPfx.IsPrefix(keyName))
	require.Equal(t, len(keyPfx)+1, len(keyName))

	id2, _ := sec.GetIdentityFromKeyName(keyName)
	require.Equal(t, id, id2)
}

// Extracts the identity name from a key name by removing the "/KEY/<kid>" suffix and returns an error if the input name does not conform to the expected key name structure.
func TestGetIdentityFromKeyName(t *testing.T) {
	tu.SetT(t)

	name, err := sec.GetIdentityFromKeyName(tu.NoErr(enc.NameFromStr("/my/test/identity/KEY/kid")))
	require.NoError(t, err)
	require.Equal(t, tu.NoErr(enc.NameFromStr("/my/test/identity")), name)

	_, err = sec.GetIdentityFromKeyName(tu.NoErr(enc.NameFromStr("/some/components")))
	require.Error(t, err)

	_, err = sec.GetIdentityFromKeyName(tu.NoErr(enc.NameFromStr("/wrong/components/KEY/wrong/this")))
	require.Error(t, err)

	_, err = sec.GetIdentityFromKeyName(enc.Name{})
	require.Error(t, err)
}

// Constructs a certificate name by appending a certificate component and version to a valid key name, ensuring the key name contains a 'KEY' component.
func TestMakeCertName(t *testing.T) {
	tu.SetT(t)

	keyName := tu.NoErr(enc.NameFromStr("/my/test/identity/KEY/kid"))
	certName, err := sec.MakeCertName(keyName, enc.NewGenericComponent("Test"), 123)
	require.NoError(t, err)
	require.Equal(t, "/my/test/identity/KEY/kid/Test/v=123", certName.String())

	// invalid key name
	_, err = sec.MakeCertName(tu.NoErr(enc.NameFromStr("/my/test/identity")), // no KEY
		enc.NewGenericComponent("Test"), 123)
	require.Error(t, err)
}

// Extracts the key name from a certificate name by truncating at the key ID component, validating that the certificate name follows the expected format (KEY-based structure with optional version/digest components).
func TestGetKeyNameFromCertName(t *testing.T) {
	tu.SetT(t)

	certName := tu.NoErr(enc.NameFromStr("/my/test/identity/KEY/kid/Test/v=123"))
	keyName, err := sec.GetKeyNameFromCertName(certName)
	require.NoError(t, err)
	require.Equal(t, tu.NoErr(enc.NameFromStr("/my/test/identity/KEY/kid")), keyName)

	// implicit digest
	certName = tu.NoErr(enc.NameFromStr("/my/test/identity/KEY/kid/Test/v=123/1=implicit"))
	keyName, err = sec.GetKeyNameFromCertName(certName)
	require.NoError(t, err)
	require.Equal(t, tu.NoErr(enc.NameFromStr("/my/test/identity/KEY/kid")), keyName)

	// invalid cert names
	_, err = sec.GetKeyNameFromCertName(tu.NoErr(enc.NameFromStr("/my/test/identity/NOTKEY/kid/Test/v=123")))
	require.Error(t, err)

	_, err = sec.GetKeyNameFromCertName(tu.NoErr(enc.NameFromStr("/my/test/identity/KEY/kid/Test/v=123/but/extra")))
	require.Error(t, err)

	_, err = sec.GetKeyNameFromCertName(enc.Name{})
	require.Error(t, err)
}
