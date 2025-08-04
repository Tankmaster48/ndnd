package codegen

import (
	"fmt"
	"strings"
	"text/template"
)

// SignatureField represents SignatureValue field
// It handles the signature covered part, and the position of the signature.
// Requires estimated length of the signature as input, which should be >= the real length.
// When estimated length is 0, the signature is not encoded.
type SignatureField struct {
	BaseTlvField

	sigCovered string
	startPoint string
	noCopy     bool
}

// Generates Go struct fields for encoding a signature field, including an integer `wireIdx` for tracking the wire format position and an unsigned integer `estLen` for estimating encoded length.
func (f *SignatureField) GenEncoderStruct() (string, error) {
	g := strErrBuf{}
	g.printlnf("%s_wireIdx int", f.name)
	g.printlnf("%s_estLen uint", f.name)
	return g.output()
}

// Generates encoder initialization code for a signature field, setting its wire index to -1 to indicate an uninitialized state in the encoding process.
func (f *SignatureField) GenInitEncoder() (string, error) {
	// SignatureInfo is set in Data/Interest.Encode()
	// {{.}}_estLen is required as an input to the encoder
	var g strErrBuf
	g.execTemplS("SignatureInitEncoder", `
		encoder.{{.}}_wireIdx = -1
	`, f.name)
	return g.output()
}

// Generates a parsing context struct for the SignatureField, returning an empty string as no additional context is required for this field.
func (f *SignatureField) GenParsingContextStruct() (string, error) {
	return "", nil
}

// Generates a code snippet initializing a signature field's context variable as an empty `enc.Wire` slice for encoding/decoding operations.
func (f *SignatureField) GenInitContext() (string, error) {
	return fmt.Sprintf("context.%s = make(enc.Wire, 0)", f.sigCovered), nil
}

// Generates code to calculate the total encoding length for a signature field, including type number, variable-length encoding of the estimated length, and the field's content length.  

Example:  
`GenEncodingLength() generates Go code that computes the TLV-encoded length for a signature field, incorporating its type, variable-length size, and content length.`
func (f *SignatureField) GenEncodingLength() (string, error) {
	var g strErrBuf
	g.printlnf("if encoder.%s_estLen > 0 {", f.name)
	g.printlne(GenTypeNumLen(f.typeNum))
	g.printlne(GenNaturalNumberLen(fmt.Sprintf("encoder.%s_estLen", f.name), true))
	g.printlnf("l += encoder.%s_estLen", f.name)
	g.printlnf("}")
	return g.output()
}

// Generates code for encoding a signature field's wire plan by calculating type/length components and updating the wire plan index, returning the generated code string or error. 

Example: Generates code to compute and append wire encoding plan steps for a signature field's type number, length, and value segments.
func (f *SignatureField) GenEncodingWirePlan() (string, error) {
	var g strErrBuf
	g.printlnf("if encoder.%s_estLen > 0 {", f.name)
	g.printlne(GenTypeNumLen(f.typeNum))
	g.printlne(GenNaturalNumberLen("encoder."+f.name+"_estLen", true))
	g.printlne(GenSwitchWirePlan())
	g.printlnf("encoder.%s_wireIdx = len(wirePlan)", f.name)
	g.printlne(GenSwitchWirePlan())
	g.printlnf("}")
	return g.output()
}

// Generates code to encode a signature field into a TLV-encoded NDN packet, handling the covered data (data to be signed) by either copying or referencing existing buffer regions based on the `noCopy` flag.
func (f *SignatureField) GenEncodeInto() (string, error) {
	g := strErrBuf{}
	g.printlnf("if encoder.%s_estLen > 0 {", f.name)
	g.printlnf("startPos := int(pos)")
	g.printlne(GenEncodeTypeNum(f.typeNum))
	g.printlne(GenNaturalNumberEncode("encoder."+f.name+"_estLen", true))
	if f.noCopy {
		// Capture the covered part from encoder.startPoint to startPos
		g.printlnf("if encoder.%s_wireIdx == int(wireIdx) {", f.startPoint)
		g.printlnf("coveredPart := buf[encoder.%s:startPos]", f.startPoint)
		g.printlnf("encoder.%s = append(encoder.%s, coveredPart)", f.sigCovered, f.sigCovered)
		g.printlnf("} else {")
		g.printlnf("coverStart := wire[encoder.%s_wireIdx][encoder.%s:]", f.startPoint, f.startPoint)
		g.printlnf("encoder.%s = append(encoder.%s, coverStart)", f.sigCovered, f.sigCovered)
		g.printlnf("for i := encoder.%s_wireIdx + 1; i < int(wireIdx); i++ {", f.startPoint)
		g.printlnf("encoder.%s = append(encoder.%s, wire[i])", f.sigCovered, f.sigCovered)
		g.printlnf("}")
		g.printlnf("coverEnd := buf[:startPos]")
		g.printlnf("encoder.%s = append(encoder.%s, coverEnd)", f.sigCovered, f.sigCovered)
		g.printlnf("}")

		// The outside encoder calculates the signature, so we simply
		// mark the buffer and shuffle the wire.
		g.printlne(GenSwitchWire())
		g.printlne(GenSwitchWire())
	} else {
		g.printlnf("coveredPart := buf[encoder.%s:startPos]", f.startPoint)
		g.printlnf("encoder.%s = append(encoder.%s, coveredPart)", f.sigCovered, f.sigCovered)

		g.printlnf("pos += encoder.%s_estLen", f.name)
	}
	g.printlnf("}")
	return g.output()
}

// Generates code to read a signature field from a reader, appending the covered data to the context's signature coverage buffer if reading is successful.
func (f *SignatureField) GenReadFrom() (string, error) {
	g := strErrBuf{}
	g.printlnf("value.%s, err = reader.ReadWire(int(l))", f.name)
	g.printlnf("if err == nil {")
	g.printlnf("coveredPart := reader.Range(context.%s, startPos)", f.startPoint)
	g.printlnf("context.%s = append(context.%s, coveredPart...)", f.sigCovered, f.sigCovered)
	g.printlnf("}")
	return g.output()
}

// Generates code to skip processing a specific field by setting it to nil in the data structure, typically used to exclude it during signature calculation.
func (f *SignatureField) GenSkipProcess() (string, error) {
	return "value." + f.name + " = nil", nil
}

// Constructs a new Signature TLV field with specified name, type number, and annotation-derived start point/covered data, using the model's NoCopy setting.
func NewSignatureField(name string, typeNum uint64, annotation string, model *TlvModel) (TlvField, error) {
	strs := strings.Split(annotation, ":")
	if len(strs) < 2 || strs[0] == "" || strs[1] == "" {
		return nil, ErrInvalidField
	}
	return &SignatureField{
		BaseTlvField: BaseTlvField{
			name:    name,
			typeNum: typeNum,
		},
		startPoint: strs[0],
		sigCovered: strs[1],
		noCopy:     model.NoCopy,
	}, nil
}

// InterestNameField represents the Name field in an Interest, which may contain a ParametersSha256DigestComponent.
// Requires needDigest as input, indicating whether ParametersSha256Digest component is required.
// It will modify the input Name value and generate a final Name value.
type InterestNameField struct {
	BaseTlvField

	sigCovered string
}

// Generates a struct definition with fields for tracking encoding state (length, digest need, wire index, and position) during Interest name encoding in Named-Data Networking.
func (f *InterestNameField) GenEncoderStruct() (string, error) {
	g := strErrBuf{}
	g.printlnf("%s_length uint", f.name)
	g.printlnf("%s_needDigest bool", f.name)
	g.printlnf("%s_wireIdx int", f.name)
	g.printlnf("%s_pos uint", f.name)
	return g.output()
}

// Initializes the encoder for the Interest Name field by adjusting components (removing a trailing SHA-256 digest component if present, adding a new one if required) and calculating the total encoded length for wire transmission.
func (f *InterestNameField) GenInitEncoder() (string, error) {
	var g strErrBuf
	const Temp = `
	encoder.{{.}}_wireIdx = -1
	encoder.{{.}}_length = 0
	if value.{{.}} != nil {
		if len(value.{{.}}) > 0 && value.{{.}}[len(value.{{.}})-1].Typ == enc.TypeParametersSha256DigestComponent {
			value.{{.}} = value.{{.}}[:len(value.{{.}})-1]
		}
		if encoder.{{.}}_needDigest {
			value.{{.}} = append(value.{{.}}, enc.Component{
				Typ: enc.TypeParametersSha256DigestComponent,
				Val: make([]byte, 32),
			})
		}
		for _, c := range value.{{.}} {
			encoder.{{.}}_length += uint(c.EncodingLength())
		}
	}
	`
	t := template.Must(template.New("InterestNameInitEncoder").Parse(Temp))
	g.executeTemplate(t, f.name)
	return g.output()
}

// Generates a struct definition for parsing context, including a wire index and position field, specific to the InterestNameField.
func (f *InterestNameField) GenParsingContextStruct() (string, error) {
	g := strErrBuf{}
	g.printlnf("%s_wireIdx int", f.name)
	g.printlnf("%s_pos uint", f.name)
	return g.output()
}

// Generates an empty initialization context string for the InterestNameField, indicating no additional data is needed for context initialization.
func (f *InterestNameField) GenInitContext() (string, error) {
	return "", nil
}

// Generates code to calculate the total encoding length of an InterestNameField, including type number and natural number length, when the field is non-nil.
func (f *InterestNameField) GenEncodingLength() (string, error) {
	g := strErrBuf{}
	g.printlnf("if value.%s != nil {", f.name)
	g.printlne(GenTypeNumLen(f.typeNum))
	g.printlne(GenNaturalNumberLen("encoder."+f.name+"_length", true))
	g.printlnf("l += encoder.%s_length", f.name)
	g.printlnf("}")
	return g.output()
}

// Generates the wire encoding plan for the InterestNameField by computing and returning the hexadecimal string representation of its encoded length.
func (f *InterestNameField) GenEncodingWirePlan() (string, error) {
	return f.GenEncodingLength()
}

// Generates code to encode an Interest Name field into a binary buffer, handling component-wise encoding, signature coverage tracking, and buffer position management for NDN packet serialization.
func (f *InterestNameField) GenEncodeInto() (string, error) {
	g := strErrBuf{}
	g.printlnf("if value.%s != nil {", f.name)
	g.printlne(GenEncodeTypeNum(f.typeNum))
	g.printlne(GenNaturalNumberEncode("encoder."+f.name+"_length", true))
	g.printlnf("sigCoverStart := pos")

	g.execTemplS("InterestNameEncodeInto", `
		i := 0
		for i = 0; i < len(value.{{.}}) - 1; i ++ {
			c := value.{{.}}[i]
			pos += uint(c.EncodeInto(buf[pos:]))
		}
		sigCoverEnd := pos
		encoder.{{.}}_wireIdx = int(wireIdx)
		if len(value.{{.}}) > 0 {
			encoder.{{.}}_pos = pos + 2
			c := value.{{.}}[i]
			pos += uint(c.EncodeInto(buf[pos:]))
			if !encoder.{{.}}_needDigest {
				sigCoverEnd = pos
			}
		}
	`, f.name)

	g.printlnf("encoder.%s = append(encoder.%s, buf[sigCoverStart:sigCoverEnd])", f.sigCovered, f.sigCovered)
	g.printlnf("}")
	return g.output()
}

// Generates Go code to read a Name field from a binary stream into a struct, parsing each component's type and value while handling errors and tracking signature coverage for validation.
func (f *InterestNameField) GenReadFrom() (string, error) {
	var g strErrBuf

	g.printlnf("{")

	g.execTemplS("NameEncodeInto", `
		value.{{.Name}} = make(enc.Name, l/2+1)
		startName := reader.Pos()
		endName := startName + int(l)
		sigCoverEnd := endName
		for j := range value.{{.Name}} {
			var err1, err3 error
			startComponent := reader.Pos()
			if startComponent >= endName {
				value.{{.Name}} = value.{{.Name}}[:j]
				break
			}
			value.{{.Name}}[j].Typ, err1 = reader.ReadTLNum()
			l, err2 := reader.ReadTLNum()
			value.{{.Name}}[j].Val, err3 = reader.ReadBuf(int(l))
			if err1 != nil || err2 != nil || err3 != nil {
				err = io.ErrUnexpectedEOF
				break
			}
			if value.{{.Name}}[j].Typ == enc.TypeParametersSha256DigestComponent {
				sigCoverEnd = startComponent
			}
		}
		if err == nil && reader.Pos() != endName {
			err = enc.ErrBufferOverflow
		}
	`, f)

	g.printlnf("if err == nil {")
	g.printlnf("coveredPart := reader.Range(startName, sigCoverEnd)")
	g.printlnf("context.%[1]s = append(context.%[1]s, coveredPart...)", f.sigCovered)
	g.printlnf("}")
	g.printlnf("}")
	return g.output()
}

// Generates a string assignment to set the Interest Name field to nil, effectively skipping its processing during packet generation.
func (f *InterestNameField) GenSkipProcess() (string, error) {
	return fmt.Sprintf("value.%s = nil", f.name), nil
}

// Constructs an InterestNameField TLV field with the given name, type number, and non-empty annotation for signature coverage.
func NewInterestNameField(name string, typeNum uint64, annotation string, _ *TlvModel) (TlvField, error) {
	if annotation == "" {
		return nil, ErrInvalidField
	}
	return &InterestNameField{
		BaseTlvField: BaseTlvField{
			name:    name,
			typeNum: typeNum,
		},
		sigCovered: annotation,
	}, nil
}
