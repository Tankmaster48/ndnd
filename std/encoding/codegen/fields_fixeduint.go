package codegen

import (
	"fmt"
	"strings"
	"text/template"
)

// FixedUintField represents a fixed-length unsigned integer.
type FixedUintField struct {
	BaseTlvField

	opt bool
	l   uint
}

// Constructs a fixed-size unsigned integer TLV field with the specified name, type number, and annotation defining its size (byte, uint16, uint32, uint64) and optional status.
func NewFixedUintField(name string, typeNum uint64, annotation string, _ *TlvModel) (TlvField, error) {
	if annotation == "" {
		return nil, ErrInvalidField
	}
	strs := strings.Split(annotation, ":")
	optional := false
	if len(strs) >= 2 && strs[1] == "optional" {
		optional = true
	}
	l := uint(0)
	switch strs[0] {
	case "byte":
		l = 1
	case "uint16":
		l = 2
	case "uint32":
		l = 4
	case "uint64":
		l = 8
	}
	return &FixedUintField{
		BaseTlvField: BaseTlvField{
			name:    name,
			typeNum: typeNum,
		},
		opt: optional,
		l:   l,
	}, nil
}

// Generates code to calculate the encoding length for a fixed-size unsigned integer field, including optional presence checks and the sum of type number length, a fixed overhead (1 byte), and the field's pre-defined length (f.l).
func (f *FixedUintField) GenEncodingLength() (string, error) {
	g := strErrBuf{}
	if f.opt {
		g.printlnf("if value.%s.IsSet() {", f.name)
		g.printlne(GenTypeNumLen(f.typeNum))
		g.printlnf("l += 1 + %d", f.l)
		g.printlnf("}")
	} else {
		g.printlne(GenTypeNumLen(f.typeNum))
		g.printlnf("l += 1 + %d", f.l)
	}
	return g.output()
}

// Generates the wire encoding plan for the FixedUintField by returning its calculated encoding length as a string representation.
func (f *FixedUintField) GenEncodingWirePlan() (string, error) {
	return f.GenEncodingLength()
}

// Generates code to encode a fixed-size unsigned integer field into a TLV-encoded buffer, handling both optional and non-optional values by writing the type, length, and big-endian value bytes according to the specified field size (1, 2, 4, or 8 bytes).
func (f *FixedUintField) GenEncodeInto() (string, error) {
	g := strErrBuf{}

	gen := func(name string) (string, error) {
		gi := strErrBuf{}
		switch f.l {
		case 1:
			gi.printlnf("buf[pos] = 1")
			gi.printlnf("buf[pos+1] = byte(%s)", name)
			gi.printlnf("pos += %d", 2)
		case 2:
			gi.printlnf("buf[pos] = 2")
			gi.printlnf("binary.BigEndian.PutUint16(buf[pos+1:], uint16(%s))", name)
			gi.printlnf("pos += %d", 3)
		case 4:
			gi.printlnf("buf[pos] = 4")
			gi.printlnf("binary.BigEndian.PutUint32(buf[pos+1:], uint32(%s))", name)
			gi.printlnf("pos += %d", 5)
		case 8:
			gi.printlnf("buf[pos] = 8")
			gi.printlnf("binary.BigEndian.PutUint64(buf[pos+1:], uint64(%s))", name)
			gi.printlnf("pos += %d", 9)
		}
		return gi.output()
	}

	if f.opt {
		g.printlnf("if optval, ok := value.%s.Get(); ok {", f.name)
		g.printlne(GenEncodeTypeNum(f.typeNum))
		g.printlne(gen("optval"))
		g.printlnf("}")
	} else {
		g.printlne(GenEncodeTypeNum(f.typeNum))
		g.printlne(gen("value." + f.name))
	}

	return g.output()
}

// Generates Go code to read a fixed-size unsigned integer field from an io.Reader, handling optional fields and different integer sizes (1, 2, 4, or 8 bytes).
func (f *FixedUintField) GenReadFrom() (string, error) {
	g := strErrBuf{}
	digit := ""
	switch f.l {
	case 1:
		digit = "byte"
	case 2:
		digit = "uint16"
	case 4:
		digit = "uint32"
	case 8:
		digit = "uint64"
	}

	gen := func(name string) {
		if f.l == 1 {
			g.printlnf("%s, err = reader.ReadByte()", name)
			g.printlnf("if err == io.EOF {")
			g.printlnf("err = io.ErrUnexpectedEOF")
			g.printlnf("}")
		} else {
			const Temp = `{{.Name}} = {{.Digit}}(0)
			{
				for i := 0; i < int(l); i++ {
					x := byte(0)
					x, err = reader.ReadByte()
					if err != nil {
						if err == io.EOF {
							err = io.ErrUnexpectedEOF
						}
						break
					}
					{{.Name}} = {{.Digit}}({{.Name}}<<8) | {{.Digit}}(x)
				}
			}
			`
			t := template.Must(template.New("FixedUintDecode").Parse(Temp))
			g.executeTemplate(t, struct {
				Name  string
				Digit string
			}{
				Name:  name,
				Digit: digit,
			})
		}
	}

	if f.opt {
		g.printlnf("{")
		g.printlnf("optval := %s(0)", digit)
		gen("optval")
		g.printlnf("value.%s.Set(optval)", f.name)
		g.printlnf("}")
	} else {
		gen("value." + f.name)
	}
	return g.output()
}

// Generates code to skip processing an optional unsigned integer field by unsetting it if optional, or returns an error message for required fields that must be present during encoding.
func (f *FixedUintField) GenSkipProcess() (string, error) {
	if f.opt {
		return fmt.Sprintf("value.%s.Unset()", f.name), nil
	} else {
		return fmt.Sprintf("err = enc.ErrSkipRequired{Name: \"%s\", TypeNum: %d}", f.name, f.typeNum), nil
	}
}
