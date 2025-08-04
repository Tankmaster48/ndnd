package codegen

import (
	"fmt"
	"text/template"
)

// WireField represents a binary string field of type Wire or [][]byte.
type WireField struct {
	BaseTlvField

	noCopy bool
}

// Constructs a new TLV WireField with the specified name, type number, and model's noCopy setting.
func NewWireField(name string, typeNum uint64, _ string, model *TlvModel) (TlvField, error) {
	return &WireField{
		BaseTlvField: BaseTlvField{
			name:    name,
			typeNum: typeNum,
		},
		noCopy: model.NoCopy,
	}, nil
}

// Generates a Go struct field declaration for tracking the length of a wire-encoded field, in the format "{FieldName}_length uint", used during NDN packet serialization.
func (f *WireField) GenEncoderStruct() (string, error) {
	return fmt.Sprintf("%s_length uint", f.name), nil
}

// Generates code to calculate and set the total length of elements in a non-nil field for encoding purposes.
func (f *WireField) GenInitEncoder() (string, error) {
	templ := template.Must(template.New("WireInitEncoder").Parse(`
		if value.{{.}} != nil {
			encoder.{{.}}_length = 0
			for _, c := range value.{{.}} {
				encoder.{{.}}_length += uint(len(c))
			}
		}
	`))

	var g strErrBuf
	g.executeTemplate(templ, f.name)
	return g.output()
}

// Generates Go code to compute the total encoding length for a non-nil NDN wire field, including type number, length field, and value size.  

**Explanation:**  
This function produces code that calculates the total byte length required to encode a specific NDN wire field when it is non-nil. The generated code adds the length of the type number, the variable-length encoding of the length field itself (using natural number encoding), and the accumulated length of the field's value. This is essential for precomputing buffer sizes during NDN packet serialization.
func (f *WireField) GenEncodingLength() (string, error) {
	g := strErrBuf{}
	g.printlnf("if value.%s != nil {", f.name)
	g.printlne(GenTypeNumLen(f.typeNum))
	g.printlne(GenNaturalNumberLen("encoder."+f.name+"_length", true))
	g.printlnf("l += encoder.%s_length", f.name)
	g.printlnf("}")
	return g.output()
}

// Generates code for encoding a field's wire format plan when `noCopy` is true, or calculates the encoding length otherwise, depending on whether the field requires non-copying (in-place) encoding logic.
func (f *WireField) GenEncodingWirePlan() (string, error) {
	if f.noCopy {
		g := strErrBuf{}
		g.printlnf("if value.%s != nil {", f.name)
		g.printlne(GenTypeNumLen(f.typeNum))
		g.printlne(GenNaturalNumberLen("encoder."+f.name+"_length", true))
		g.printlne(GenSwitchWirePlan())
		g.printlnf("for range value.%s {", f.name)
		g.printlne(GenSwitchWirePlan())
		g.printlnf("}")
		g.printlnf("}")
		return g.output()
	} else {
		return f.GenEncodingLength()
	}
}

// Generates code to encode a wire field's value into a buffer, handling type numbering, length encoding, and either copying data into the buffer or preserving references based on the `noCopy` flag.
func (f *WireField) GenEncodeInto() (string, error) {
	g := strErrBuf{}
	g.printlnf("if value.%s != nil {", f.name)
	g.printlne(GenEncodeTypeNum(f.typeNum))
	g.printlne(GenNaturalNumberEncode("encoder."+f.name+"_length", true))
	if f.noCopy {
		g.printlne(GenSwitchWire())
		g.printlnf("for _, w := range value.%s {", f.name)
		g.printlnf("wire[wireIdx] = w")
		g.printlne(GenSwitchWire())
		g.printlnf("}")
	} else {
		g.printlnf("for _, w := range value.%s {", f.name)
		g.printlnf("copy(buf[pos:], w)")
		g.printlnf("pos += uint(len(w))")
		g.printlnf("}")
	}
	g.printlnf("}")
	return g.output()
}

// Generates code to read the wire field's value into the corresponding struct field using the provided reader and specified length.
func (f *WireField) GenReadFrom() (string, error) {
	g := strErrBuf{}
	g.printlnf("value.%s, err = reader.ReadWire(int(l))", f.name)
	return g.output()
}

// Generates a code snippet that sets the field's value to nil, indicating it should be skipped during processing.
func (f *WireField) GenSkipProcess() (string, error) {
	return "value." + f.name + " = nil", nil
}
