package sec

import (
	"fmt"
	"io"
	"os"

	"github.com/named-data/ndnd/std/security"
	"github.com/spf13/cobra"
)

type ToolPem struct{}

// Configures a Cobra command with PEM encoding/decoding utilities for converting NDN keys/certificates between TLV and PEM formats via stdin/stdout.
func (t *ToolPem) configure(cmd *cobra.Command) {
	cmd.AddGroup(&cobra.Group{
		ID:    "encoding",
		Title: "Encoding Utilities",
	})

	cmd.AddCommand(&cobra.Command{
		GroupID: "encoding",
		Use:     "pem-encode",
		Short:   "Encode an NDN key or cert to PEM",
		Long: `Encode a TLV NDN Key or Certificate to PEM.
Provide TLV data as input to stdin.`,
		Example: `  ndnd sec pem-encode < alice.tlv > alice.pem`,
		Args:    cobra.NoArgs,
		Run:     t.Encode,
	})

	cmd.AddCommand(&cobra.Command{
		GroupID: "encoding",
		Use:     "pem-decode",
		Short:   "Decode PEM to NDN TLV format",
		Long: `Decode a PEM file containing a single NDN TLV.
Provide PEM data as input to stdin.`,
		Example: `  ndnd sec pem-decode < alice.pem > alice.tlv`,
		Args:    cobra.NoArgs,
		Run:     t.Decode,
	})
}

// Encodes binary input read from standard input into PEM format using the security package and writes the resulting PEM-encoded data to standard output, exiting on errors.
func (t *ToolPem) Encode(_ *cobra.Command, args []string) {
	input, err := io.ReadAll(os.Stdin)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to read input: %+v\n", err)
		os.Exit(1)
		return
	}

	out, err := security.PemEncode(input)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to convert to text: %+v\n", err)
		os.Exit(1)
		return
	}

	os.Stdout.Write(out)
}

// Decodes a single PEM-encoded NDN data structure from standard input, validates it contains exactly one entry, and writes the decoded binary data to standard output.
func (t *ToolPem) Decode(_ *cobra.Command, args []string) {
	input, err := io.ReadAll(os.Stdin)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to read input from stdin: %+v\n", err)
		os.Exit(1)
		return
	}

	out := security.PemDecode(input)
	if len(out) == 0 {
		fmt.Fprintf(os.Stderr, "No valid NDN data found in stdin input\n")
		os.Exit(1)
		return
	}
	if len(out) > 1 {
		fmt.Fprintf(os.Stderr, "Multiple NDN data found in stdin input\n")
		os.Exit(1)
		return
	}

	os.Stdout.Write(out[0])
}
