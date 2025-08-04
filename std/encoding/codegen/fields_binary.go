package codegen

import "fmt"

// BinaryField represents a binary string field of type Buffer or []byte.
// BinaryField always makes a copy during encoding.
type BinaryField struct {
	BaseTlvField
}

// Constructs a BinaryField TLV field with the given name and type number, ignoring additional unused parameters.
func NewBinaryField(name string, typeNum uint64, _ string, _ *TlvModel) (TlvField, error) {
	return &BinaryField{
		BaseTlvField: BaseTlvField{
			name:    name,
			typeNum: typeNum,
		},
	}, nil
}

// Generates code to compute the total encoded length of a binary field, including type number, natural number length encoding, and data size, when the field is non-nil.
func (f *BinaryField) GenEncodingLength() (string, error) {
	g := strErrBuf{}
	g.printlnf("if value.%s != nil {", f.name)
	g.printlne(GenTypeNumLen(f.typeNum))
	g.printlne(GenNaturalNumberLen(fmt.Sprintf("len(value.%s)", f.name), true))
	g.printlnf("l += uint(len(value.%s))", f.name)
	g.printlnf("}")
	return g.output()
}

// Generates the wire encoding plan for the binary field, returning the encoded length as a string.
func (f *BinaryField) GenEncodingWirePlan() (string, error) {
	return f.GenEncodingLength()
}

// Generates code to encode a binary field into a buffer by writing its type number, length as a natural number, and value data if non-nil.
func (f *BinaryField) GenEncodeInto() (string, error) {
	g := strErrBuf{}
	g.printlnf("if value.%s != nil {", f.name)
	g.printlne(GenEncodeTypeNum(f.typeNum))
	g.printlne(GenNaturalNumberEncode("len(value."+f.name+")", true))
	g.printlnf("copy(buf[pos:], value.%s)", f.name)
	g.printlnf("pos += uint(len(value.%s))", f.name)
	g.printlnf("}")
	return g.output()
}

// Generates Go code to read a binary field into a byte slice of a specified length from a reader, returning the generated code and any error.
func (f *BinaryField) GenReadFrom() (string, error) {
	g := strErrBuf{}
	g.printlnf("value.%s = make([]byte, l)", f.name)
	g.printlnf("_, err = reader.ReadFull(value.%s)", f.name)
	return g.output()
}

// Generates a code snippet that assigns `nil` to the field's name in the value, indicating it should be skipped during processing.
func (f *BinaryField) GenSkipProcess() (string, error) {
	return "value." + f.name + " = nil", nil
}
