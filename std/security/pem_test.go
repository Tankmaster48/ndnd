package security_test

import (
	"encoding/base64"
	"testing"

	"github.com/named-data/ndnd/std/security"
	tu "github.com/named-data/ndnd/std/utils/testutils"
	"github.com/stretchr/testify/require"
)

// Verifies that the PEM encoding of a decoded base64 certificate matches the expected PEM-formatted string.
func TestPemEncodeCert(t *testing.T) {
	tu.SetT(t)

	cert, _ := base64.StdEncoding.DecodeString(CERT_ROOT)
	res := tu.NoErr(security.PemEncode(cert))
	require.Equal(t, CERT_ROOT_PEM, string(res))
}

// Verifies that the `PemDecode` function correctly decodes a PEM-formatted certificate string into its raw byte representation, ensuring the output matches a precomputed base64-decoded certificate.
func TestPemDecodeCert(t *testing.T) {
	tu.SetT(t)

	cert, _ := base64.StdEncoding.DecodeString(CERT_ROOT)
	res := security.PemDecode([]byte(CERT_ROOT_PEM))
	require.Equal(t, cert, res[0])
}
