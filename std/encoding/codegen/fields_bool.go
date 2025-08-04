package codegen

import "fmt"

// BoolField represents a boolean field.
type BoolField struct {
	BaseTlvField
}

// Constructs a new Boolean TLV field with the specified name and type number, ignoring additional unused parameters.
func NewBoolField(name string, typeNum uint64, _ string, _ *TlvModel) (TlvField, error) {
	return &BoolField{
		BaseTlvField: BaseTlvField{
			name:    name,
			typeNum: typeNum,
		},
	}, nil
}

// Generates code that calculates the total encoding length for a boolean field when it is true, including the lengths of its type number and a zero type value.
func (f *BoolField) GenEncodingLength() (string, error) {
	g := strErrBuf{}
	g.printlnf("if value.%s {", f.name)
	g.printlne(GenTypeNumLen(f.typeNum))
	g.printlne(GenTypeNumLen(0))
	g.printlnf("}")
	return g.output()
}

// Generates the wire encoding plan for a boolean field by returning its calculated encoding length as a string representation.
func (f *BoolField) GenEncodingWirePlan() (string, error) {
	return f.GenEncodingLength()
}

// Generates code to conditionally encode a boolean field into a binary format by outputting its type number and a zero when the field is true.
func (f *BoolField) GenEncodeInto() (string, error) {
	g := strErrBuf{}
	g.printlnf("if value.%s {", f.name)
	g.printlne(GenEncodeTypeNum(f.typeNum))
	g.printlne(GenEncodeTypeNum(0))
	g.printlnf("}")
	return g.output()
}

// Generates code to read a boolean field from a data stream, setting the field to true and skipping the field's length.
func (f *BoolField) GenReadFrom() (string, error) {
	g := strErrBuf{}
	g.printlnf("value.%s = true", f.name)
	g.printlnf("err = reader.Skip(int(l))")
	return g.output()
}

// Generates a code snippet that sets the boolean field's value to false using its name. 

Example: `value.IsProcessed = false` (assuming field name is "IsProcessed").
func (f *BoolField) GenSkipProcess() (string, error) {
	return fmt.Sprintf("value.%s = false", f.name), nil
}
