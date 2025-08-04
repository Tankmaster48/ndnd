package codegen

import "fmt"

// TimeField represents a time field, recorded as milliseconds.
type TimeField struct {
	BaseTlvField

	opt bool
}

// Constructs a TimeField with the given name, type number, and optional status based on whether the annotation is "optional".
func NewTimeField(name string, typeNum uint64, annotation string, _ *TlvModel) (TlvField, error) {
	return &TimeField{
		BaseTlvField: BaseTlvField{
			name:    name,
			typeNum: typeNum,
		},
		opt: annotation == "optional",
	}, nil
}

// Generates code to compute the TLV encoding length for a time field, handling optional fields and converting the time value to milliseconds as a natural number.
func (f *TimeField) GenEncodingLength() (string, error) {
	g := strErrBuf{}
	if f.opt {
		g.printlnf("if optval, ok := value.%s.Get(); ok {", f.name)
		g.printlne(GenTypeNumLen(f.TypeNum()))
		g.printlne(GenNaturalNumberLen("uint64(optval/time.Millisecond)", false))
		g.printlnf("}")
	} else {
		g.printlne(GenTypeNumLen(f.TypeNum()))
		g.printlne(GenNaturalNumberLen("uint64(value."+f.name+"/time.Millisecond)", false))
	}
	return g.output()
}

// Generates the wire encoding plan for a TimeField by returning its encoded length as a string.
func (f *TimeField) GenEncodingWirePlan() (string, error) {
	return f.GenEncodingLength()
}

// Generates code to encode a time field as a natural number of milliseconds, handling optional fields by checking their presence before encoding.
func (f *TimeField) GenEncodeInto() (string, error) {
	g := strErrBuf{}
	if f.opt {
		g.printlnf("if optval, ok := value.%s.Get(); ok {", f.name)
		g.printlne(GenEncodeTypeNum(f.typeNum))
		g.printlne(GenNaturalNumberEncode("uint64(optval/time.Millisecond)", false))
		g.printlnf("}")
	} else {
		g.printlne(GenEncodeTypeNum(f.typeNum))
		g.printlne(GenNaturalNumberEncode("uint64(value."+f.name+"/time.Millisecond)", false))
	}
	return g.output()
}

// Generates Go code to decode a time field from input, converting a uint64 value to a time.Duration in milliseconds and assigning it to the target struct field (using `Set` for optional fields).
func (f *TimeField) GenReadFrom() (string, error) {
	g := strErrBuf{}
	g.printlnf("{")
	g.printlnf("timeInt := uint64(0)")
	g.printlne(GenNaturalNumberDecode("timeInt"))
	if f.opt {
		g.printlnf("optval := time.Duration(timeInt) * time.Millisecond")
		g.printlnf("value.%s.Set(optval)", f.name)
	} else {
		g.printlnf("value.%s = time.Duration(timeInt) * time.Millisecond", f.name)
	}
	g.printlnf("}")
	return g.output()
}

// Generates code to unset an optional time field or return an error for a required time field when skipping during encoding.
func (f *TimeField) GenSkipProcess() (string, error) {
	if f.opt {
		return fmt.Sprintf("value.%s.Unset()", f.name), nil
	} else {
		return fmt.Sprintf("err = enc.ErrSkipRequired{Name: \"%s\", TypeNum: %d}", f.name, f.typeNum), nil
	}
}
