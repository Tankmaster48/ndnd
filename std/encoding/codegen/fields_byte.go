package codegen

import (
	"fmt"
)

// ByteField represents a pointer to the wire address of a byte.
type ByteField struct {
	BaseTlvField
}

// Constructs a new TLV ByteField with the specified name and type number, ignoring the annotation and model parameters.
func NewByteField(name string, typeNum uint64, annotation string, _ *TlvModel) (TlvField, error) {
	return &ByteField{
		BaseTlvField: BaseTlvField{
			name:    name,
			typeNum: typeNum,
		},
	}, nil
}

// Generates code to compute the encoding length for a non-nil TLV field, including the type's encoded length and an additional 2 bytes for the length field.
func (f *ByteField) GenEncodingLength() (string, error) {
	g := strErrBuf{}
	g.printlnf("if value.%s != nil {", f.name)
	g.printlne(GenTypeNumLen(f.typeNum))
	g.printlnf("l += 2")
	g.printlnf("}")
	return g.output()
}

// Generates the wire encoding plan for the byte field by determining its encoded length.
func (f *ByteField) GenEncodingWirePlan() (string, error) {
	return f.GenEncodingLength()
}

// Generates code to encode a byte field into a buffer, writing the type, length (1), and value only if the field is non-nil.  

*Example format:*  
`Generates code to encode a byte field into a buffer, writing the type, length (1), and value only if the field is non-nil.`
func (f *ByteField) GenEncodeInto() (string, error) {
	g := strErrBuf{}
	g.printlnf("if value.%s != nil {", f.name)
	g.printlne(GenEncodeTypeNum(f.typeNum))
	g.printlnf("buf[pos] = 1")
	g.printlnf("buf[pos+1] = byte(*value.%s)", f.name)
	g.printlnf("pos += 2")
	g.printlnf("}")
	return g.output()
}

// Generates code to read a single byte from an input stream into a struct field, converting EOF errors to unexpected EOF and storing the result in a pointer to the field's byte value.
func (f *ByteField) GenReadFrom() (string, error) {
	g := strErrBuf{}
	g.printlnf("{")
	g.printlnf("buf, err := reader.ReadBuf(1)")
	g.printlnf("if err == io.EOF {")
	g.printlnf("err = io.ErrUnexpectedEOF")
	g.printlnf("}")
	g.printlnf("value.%s = &buf[0]", f.name)
	g.printlnf("}")
	return g.output()
}

// Generates a code snippet that assigns `nil` to the field's corresponding struct member, effectively skipping its processing.
func (f *ByteField) GenSkipProcess() (string, error) {
	return fmt.Sprintf("value.%s = nil", f.name), nil
}
