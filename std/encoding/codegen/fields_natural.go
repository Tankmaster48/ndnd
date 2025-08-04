package codegen

import "fmt"

// NaturalField represents a natural number field.
type NaturalField struct {
	BaseTlvField

	opt bool
}

// Constructs a new TLV NaturalField with the given name, type number, and optional status based on the annotation.
func NewNaturalField(name string, typeNum uint64, annotation string, _ *TlvModel) (TlvField, error) {
	return &NaturalField{
		BaseTlvField: BaseTlvField{
			name:    name,
			typeNum: typeNum,
		},
		opt: annotation == "optional",
	}, nil
}

// Generates code to calculate the total encoding length of a field, including its type number and natural number value, with conditional handling for optional fields that may not be present.
func (f *NaturalField) GenEncodingLength() (string, error) {
	g := strErrBuf{}
	if f.opt {
		g.printlnf("if optval, ok := value.%s.Get(); ok {", f.name)
		g.printlne(GenTypeNumLen(f.typeNum))
		g.printlne(GenNaturalNumberLen("optval", false))
		g.printlnf("}")
	} else {
		g.printlne(GenTypeNumLen(f.typeNum))
		g.printlne(GenNaturalNumberLen("value."+f.name, false))
	}
	return g.output()
}

// Generates the wire encoding plan for the field by returning its encoded length as a string.
func (f *NaturalField) GenEncodingWirePlan() (string, error) {
	return f.GenEncodingLength()
}

// Generates code to encode a natural number field (handling optional fields with a presence check) into a data structure, returning the generated code as a string and an error.
func (f *NaturalField) GenEncodeInto() (string, error) {
	g := strErrBuf{}
	if f.opt {
		g.printlnf("if optval, ok := value.%s.Get(); ok {", f.name)
		g.printlne(GenEncodeTypeNum(f.typeNum))
		g.printlne(GenNaturalNumberEncode("optval", false))
		g.printlnf("}")
	} else {
		g.printlne(GenEncodeTypeNum(f.typeNum))
		g.printlne(GenNaturalNumberEncode("value."+f.name, false))
	}
	return g.output()
}

// Generates Go code to decode a natural number field into a struct, using a temporary variable for optional fields to safely handle absence in input data.
func (f *NaturalField) GenReadFrom() (string, error) {
	if f.opt {
		g := strErrBuf{}
		g.printlnf("{")
		g.printlnf("optval := uint64(0)")
		g.printlne(GenNaturalNumberDecode("optval"))
		g.printlnf("value.%s.Set(optval)", f.name)
		g.printlnf("}")
		return g.output()
	} else {
		return GenNaturalNumberDecode("value." + f.name)
	}
}

// Generates code to either unset an optional field or produce an error when skipping a required field during encoding.
func (f *NaturalField) GenSkipProcess() (string, error) {
	if f.opt {
		return fmt.Sprintf("value.%s.Unset()", f.name), nil
	} else {
		return fmt.Sprintf("err = enc.ErrSkipRequired{Name: \"%s\", TypeNum: %d}", f.name, f.typeNum), nil
	}
}
