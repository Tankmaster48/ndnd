package codegen

import (
	"fmt"
	"strings"
	"text/template"
)

// Generates a code snippet that appends the number of bytes required to encode a TLV type number in NDN, based on its value range (1, 3, 5, or 9 bytes).
func GenTypeNumLen(typeNum uint64) (string, error) {
	var ret uint
	switch {
	case typeNum <= 0xfc:
		ret = 1
	case typeNum <= 0xffff:
		ret = 3
	case typeNum <= 0xffffffff:
		ret = 5
	default:
		ret = 9
	}
	return fmt.Sprintf("\tl += %d", ret), nil
}

// Generates Go code to encode a type number into a byte buffer using variable-length encoding (1, 3, 5, or 9 bytes) based on its value, following TLV format conventions for efficient serialization.
func GenEncodeTypeNum(typeNum uint64) (string, error) {
	ret := ""
	switch {
	case typeNum <= 0xfc:
		ret += fmt.Sprintf("\tbuf[pos] = byte(%d)\n", typeNum)
		ret += fmt.Sprintf("\tpos += %d", 1)
	case typeNum <= 0xffff:
		ret += fmt.Sprintf("\tbuf[pos] = %d\n", 0xfd)
		ret += fmt.Sprintf("\tbinary.BigEndian.PutUint16(buf[pos+1:], uint16(%d))\n", typeNum)
		ret += fmt.Sprintf("\tpos += %d", 3)
	case typeNum <= 0xffffffff:
		ret += fmt.Sprintf("\tbuf[pos] = %d\n", 0xfe)
		ret += fmt.Sprintf("\tbinary.BigEndian.PutUint32(buf[pos+1:], uint32(%d))\n", typeNum)
		ret += fmt.Sprintf("\tpos += %d", 5)
	default:
		ret += fmt.Sprintf("\tbuf[pos] = %d\n", 0xff)
		ret += fmt.Sprintf("\tbinary.BigEndian.PutUint64(buf[pos+1:], uint64(%d))\n", typeNum)
		ret += fmt.Sprintf("\tpos += %d", 9)
	}
	return ret, nil
}

// Generates Go code to calculate the encoding length of a natural number field, either as TLV-encoded length (if `isTlv=true`) or with a 1-byte header plus value length (if `isTlv=false`), using the provided variable name.  

Example: For `code="n"` and `isTlv=true`, returns `l += uint(enc.TLNum(n).EncodingLength())`.
func GenNaturalNumberLen(code string, isTlv bool) (string, error) {
	var temp string
	if isTlv {
		temp = `l += uint(enc.TLNum({{.}}).EncodingLength())`
	} else {
		temp = `l += uint(1 + enc.Nat({{.}}).EncodingLength())`
	}
	t := template.Must(template.New("NaturalNumberLen").Parse(temp))
	b := strings.Builder{}
	err := t.Execute(&b, code)
	return b.String(), err
}

// Generates Go code for encoding a natural number into a byte buffer using either TLV (Type-Length-Value) format or variable-length encoding, based on the provided `isTlv` flag.
func GenNaturalNumberEncode(code string, isTlv bool) (string, error) {
	var temp string
	if isTlv {
		temp = `pos += uint(enc.TLNum({{.}}).EncodeInto(buf[pos:]))`
	} else {
		temp = `
			buf[pos] = byte(enc.Nat({{.}}).EncodeInto(buf[pos+1:]))
			pos += uint(1 + buf[pos])
		`
	}
	t := template.Must(template.New("NaturalNumberEncode").Parse(temp))
	b := strings.Builder{}
	err := t.Execute(&b, code)
	return b.String(), err
}

// Generates a code snippet for decoding a TLV number into a specified variable with error handling, returning the generated code and any template execution errors.
func GenTlvNumberDecode(code string) (string, error) {
	const Temp = `{{.}}, err = reader.ReadTLNum()
	if err != nil {
		return nil, enc.ErrFailToParse{TypeNum: 0, Err: err}
	}`
	t := template.Must(template.New("TlvNumberDecode").Parse(Temp))
	b := strings.Builder{}
	err := t.Execute(&b, code)
	return b.String(), err
}

// Generates Go code to decode a natural number (unsigned integer) from a byte stream into the specified variable, handling byte-by-byte reading, bit shifting, and error checking for unexpected EOF.
func GenNaturalNumberDecode(code string) (string, error) {
	const Temp = `{{.}} = uint64(0)
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
			{{.}} = uint64({{.}}<<8) | uint64(x)
		}
	}`
	t := template.Must(template.New("NaturalNumberDecode").Parse(Temp))
	b := strings.Builder{}
	err := t.Execute(&b, code)
	return b.String(), err
}

// Generates a code snippet for appending a value `l` to a wire plan and resetting `l` to zero, typically used in encoding or packet construction workflows.
func GenSwitchWirePlan() (string, error) {
	return `wirePlan = append(wirePlan, l)
	l = 0`, nil
}

// Generates code to advance to the next buffer in a wire array, resetting the position counter and handling end-of-wire conditions by setting the buffer to nil when the index exceeds the wire length.
func GenSwitchWire() (string, error) {
	return `wireIdx ++
	pos = 0
	if wireIdx < len(wire) {
		buf = wire[wireIdx]
	} else {
		buf = nil
	}`, nil
}
