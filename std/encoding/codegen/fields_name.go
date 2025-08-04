package codegen

import (
	"fmt"
	"text/template"
)

// NameField represents a name field.
type NameField struct {
	BaseTlvField
}

// Constructs a new NameField TLV field with the given name and type number for use in NDN data structures.
func NewNameField(name string, typeNum uint64, _ string, _ *TlvModel) (TlvField, error) {
	return &NameField{
		BaseTlvField: BaseTlvField{
			name:    name,
			typeNum: typeNum,
		},
	}, nil
}

// Generates a struct field declaration for the length of the name field as an unsigned integer, used in TLV encoding of NDN packets.
func (f *NameField) GenEncoderStruct() (string, error) {
	return fmt.Sprintf("%s_length uint", f.name), nil
}

// Generates Go code to initialize an encoder for a NameField by calculating the total length of its components by summing each component's encoding length.
func (f *NameField) GenInitEncoder() (string, error) {
	var g strErrBuf
	const Temp = `if value.{{.}} != nil {
		encoder.{{.}}_length = 0
		for _, c := range value.{{.}} {
			encoder.{{.}}_length += uint(c.EncodingLength())
		}
	}
	`
	t := template.Must(template.New("NameInitEncoder").Parse(Temp))
	g.executeTemplate(t, f.name)
	return g.output()
}

// Generates code to calculate the total encoding length for a non-nil field by summing the TLV type number length and the variable-length natural number encoding of the field's length in NDN.
func (f *NameField) GenEncodingLength() (string, error) {
	g := strErrBuf{}
	g.printlnf("if value.%s != nil {", f.name)
	g.printlne(GenTypeNumLen(f.typeNum))
	g.printlne(GenNaturalNumberLen(fmt.Sprintf("encoder.%s_length", f.name), true))
	g.printlnf("l += encoder.%s_length", f.name)
	g.printlnf("}")
	return g.output()
}

// Generates the wire encoding plan for the NameField by returning its encoded length as a string representation.
func (f *NameField) GenEncodingWirePlan() (string, error) {
	return f.GenEncodingLength()
}

// Generates Go code to encode a variable-length array field (e.g., Name components) into a TLV-encoded NDN packet by writing the type, length, and sequentially encoding each array element.
func (f *NameField) GenEncodeInto() (string, error) {
	g := strErrBuf{}
	g.printlnf("if value.%s != nil {", f.name)
	g.printlne(GenEncodeTypeNum(f.typeNum))
	g.printlne(GenNaturalNumberEncode("encoder."+f.name+"_length", true))
	g.printlnf("for _, c := range value.%s {", f.name)
	g.printlnf("pos += uint(c.EncodeInto(buf[pos:]))")
	g.printlnf("}")
	g.printlnf("}")
	return g.output()
}

// Generates code to read a Name component from a reader into a struct field using a delegate.
func (f *NameField) GenReadFrom() (string, error) {
	const Temp = `
		delegate:=reader.Delegate(int(l))
		value.{{.Name}}, err = delegate.ReadName()
	`
	t := template.Must(template.New("NameEncodeInto").Parse(Temp))
	var g strErrBuf
	g.executeTemplate(t, f)
	return g.output()
}

// Generates a code snippet that assigns `nil` to the struct field specified by the `NameField`'s name, effectively skipping its processing in the `value` struct.
func (f *NameField) GenSkipProcess() (string, error) {
	return "value." + f.name + " = nil", nil
}
