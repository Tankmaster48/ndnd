package codegen

import "fmt"

// StringField represents a UTF-8 encoded string.
type StringField struct {
	BaseTlvField

	opt bool
}

// Constructs a TLV string field with the given name, type number, and optional status determined by the "optional" annotation.
func NewStringField(name string, typeNum uint64, annotation string, _ *TlvModel) (TlvField, error) {
	return &StringField{
		BaseTlvField: BaseTlvField{
			name:    name,
			typeNum: typeNum,
		},
		opt: annotation == "optional",
	}, nil
}

// Generates code to calculate the total encoding length for a string field, including type number, natural number length encoding, and the string's byte length, with optional presence checking if the field is marked optional.
func (f *StringField) GenEncodingLength() (string, error) {
	g := strErrBuf{}
	if f.opt {
		g.printlnf("if optval, ok := value.%s.Get(); ok {", f.name)
		g.printlne(GenTypeNumLen(f.typeNum))
		g.printlne(GenNaturalNumberLen("len(optval)", true))
		g.printlnf("l += uint(len(optval))")
		g.printlnf("}")
	} else {
		g.printlne(GenTypeNumLen(f.typeNum))
		g.printlne(GenNaturalNumberLen("len(value."+f.name+")", true))
		g.printlnf("l += uint(len(value.%s))", f.name)
	}
	return g.output()
}

// Generates the wire encoding plan for a string field by calculating and returning its length as part of the encoding process.
func (f *StringField) GenEncodingWirePlan() (string, error) {
	return f.GenEncodingLength()
}

// Generates code to encode a string field (optional or required) into a buffer, including type number, length, and value, for Named-Data Networking data structures.
func (f *StringField) GenEncodeInto() (string, error) {
	g := strErrBuf{}
	if f.opt {
		g.printlnf("if optval, ok := value.%s.Get(); ok {", f.name)
		g.printlne(GenEncodeTypeNum(f.typeNum))
		g.printlne(GenNaturalNumberEncode("len(optval)", true))
		g.printlnf("copy(buf[pos:], optval)")
		g.printlnf("pos += uint(len(optval))")
		g.printlnf("}")
	} else {
		g.printlne(GenEncodeTypeNum(f.typeNum))
		g.printlne(GenNaturalNumberEncode("len(value."+f.name+")", true))
		g.printlnf("copy(buf[pos:], value.%s)", f.name)
		g.printlnf("pos += uint(len(value.%s))", f.name)
	}
	return g.output()
}

// Generates code to read a string field from a data stream into a struct, handling optional fields and using a specified length for the read operation.
func (f *StringField) GenReadFrom() (string, error) {
	g := strErrBuf{}
	g.printlnf("{")
	g.printlnf("var builder strings.Builder")
	g.printlnf("_, err = reader.CopyN(&builder, int(l))")
	g.printlnf("if err == nil {")
	if f.opt {
		g.printlnf("value.%s.Set(builder.String())", f.name)
	} else {
		g.printlnf("value.%s = builder.String()", f.name)
	}
	g.printlnf("}")
	g.printlnf("}")
	return g.output()
}

// Generates code to handle skipping an optional string field (by unsetting it) or returns an error for skipping a required field, including the field name and type number.
func (f *StringField) GenSkipProcess() (string, error) {
	if f.opt {
		return fmt.Sprintf("value.%s.Unset()", f.name), nil
	} else {
		return fmt.Sprintf("err = enc.ErrSkipRequired{Name: \"%s\", TypeNum: %d}", f.name, f.typeNum), nil
	}
}
