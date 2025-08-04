package encoding

import (
	"fmt"
	"strconv"
	"strings"
	"unicode"
)

type compValFmt interface {
	WriteTo(val []byte, sb *strings.Builder) int
	FromString(s string) ([]byte, error)
	ToMatching(val []byte) any
	FromMatching(m any) ([]byte, error)
}

type compValFmtInvalid struct{}
type compValFmtText struct{}
type compValFmtDec struct{}
type compValFmtHex struct{}

// Handles invalid component value formatting by writing nothing to the strings.Builder and returning 0 bytes written.
func (compValFmtInvalid) WriteTo(val []byte, sb *strings.Builder) int {
	return 0
}

// Returns an error indicating invalid component format when attempting to convert a string to bytes.
func (compValFmtInvalid) FromString(s string) ([]byte, error) {
	return nil, ErrFormat{"Invalid component format"}
}

// Returns nil for invalid components, indicating no valid match can be constructed from the provided byte slice.
func (compValFmtInvalid) ToMatching(val []byte) any {
	return nil
}

// Returns an error indicating invalid component format when attempting to convert input to bytes.
func (compValFmtInvalid) FromMatching(m any) ([]byte, error) {
	return nil, ErrFormat{"Invalid component format"}
}

// **Function Description:**  
Encodes a byte slice into a percent-encoded text format, appending the result to a strings.Builder, with non-printable characters represented as %XX sequences in uppercase hexadecimal.
func (compValFmtText) WriteTo(val []byte, sb *strings.Builder) int {
	size := 0
	for _, b := range val {
		if isLegalCompText(b) {
			sb.WriteByte(b)
			size += 1
		} else {
			sb.WriteRune('%')
			sb.WriteRune(HEX_UPPER[b>>4])
			sb.WriteRune(HEX_UPPER[b&0x0F])
			size += 3
		}
	}
	return size
}

// Converts a string to a byte slice using percent-encoding, validating and handling special characters (`%`, `=`, `/`, `\`) according to NDN component formatting rules, returning an error for invalid sequences or unescaped special characters.
func (compValFmtText) FromString(valStr string) ([]byte, error) {
	hasSpecialChar := false
	for _, c := range valStr {
		if c == '%' || c == '=' || c == '/' || c == '\\' {
			hasSpecialChar = true
			break
		}
	}
	if !hasSpecialChar {
		return []byte(valStr), nil
	}

	val := make([]byte, 0, len(valStr))
	for i := 0; i < len(valStr); {
		if isLegalCompText(valStr[i]) {
			val = append(val, valStr[i])
			i++
		} else if valStr[i] == '%' && i+2 < len(valStr) {
			v, err := strconv.ParseUint(valStr[i+1:i+3], 16, 8)
			if err != nil {
				return nil, ErrFormat{"invalid component value: " + valStr}
			}
			val = append(val, byte(v))
			i += 3
		} else {
			// Gracefully accept invalid character
			if valStr[i] != '%' && valStr[i] != '=' && valStr[i] != '/' && valStr[i] != '\\' {
				val = append(val, valStr[i])
				i++
			} else {
				return nil, ErrFormat{"invalid component value: " + valStr}
			}
		}
	}
	return val, nil
}

// Converts a byte slice to the text component value format, returning the input bytes unchanged as the corresponding text-formatted value.
func (compValFmtText) ToMatching(val []byte) any {
	return val
}

// Validates and converts an input value to a byte slice for a text component, returning an error if the input is not a valid byte slice.
func (compValFmtText) FromMatching(m any) ([]byte, error) {
	ret, ok := m.([]byte)
	if !ok {
		return nil, ErrFormat{"invalid text component value: " + fmt.Sprintf("%v", m)}
	} else {
		return ret, nil
	}
}

// Converts the input byte slice to a big-endian unsigned integer, appends its decimal string representation to the provided strings.Builder, and returns the number of characters written.
func (compValFmtDec) WriteTo(val []byte, sb *strings.Builder) int {
	x := uint64(0)
	for _, b := range val {
		x = (x << 8) | uint64(b)
	}
	vstr := strconv.FormatUint(x, 10)
	sb.WriteString(vstr)
	return len(vstr)
}

// Converts a decimal string representation of a component value into its binary encoded form, returning an error if the input is not a valid decimal number.
func (compValFmtDec) FromString(s string) ([]byte, error) {
	x, err := strconv.ParseUint(s, 10, 64)
	if err != nil {
		return nil, ErrFormat{"invalid decimal component value: " + s}
	}
	ret := make([]byte, Nat(x).EncodingLength())
	Nat(x).EncodeInto(ret)
	return ret, nil
}

// Converts a byte slice to a big-endian unsigned integer (uint64) for component value matching in NDN.
func (compValFmtDec) ToMatching(val []byte) any {
	x := uint64(0)
	for _, b := range val {
		x = (x << 8) | uint64(b)
	}
	return x
}

// Encodes a uint64 value into a byte slice using decimal-based NDN component encoding, returning an error if the input is invalid.
func (compValFmtDec) FromMatching(m any) ([]byte, error) {
	x, ok := m.(uint64)
	if !ok {
		return nil, ErrFormat{"invalid decimal component value: " + fmt.Sprintf("%v", m)}
	}
	ret := make([]byte, Nat(x).EncodingLength())
	Nat(x).EncodeInto(ret)
	return ret, nil
}

// Writes the hexadecimal representation of the byte slice to the strings.Builder using lowercase characters, with each byte encoded as two hex digits.
func (compValFmtHex) WriteTo(val []byte, sb *strings.Builder) int {
	for _, b := range val {
		sb.WriteRune(HEX_LOWER[b>>4])
		sb.WriteRune(HEX_LOWER[b&0x0F])
	}
	return len(val) * 2
}

// Converts a hexadecimal string to its corresponding byte slice representation, returning an error if the input is not a valid even-length hexadecimal string.
func (compValFmtHex) FromString(s string) ([]byte, error) {
	if len(s)%2 != 0 {
		return nil, ErrFormat{"invalid hexadecimal component value: " + s}
	}
	l := len(s) / 2
	val := make([]byte, l)
	for i := 0; i < l; i++ {
		b, err := strconv.ParseUint(s[i*2:i*2+2], 16, 8)
		if err != nil {
			return nil, ErrFormat{"invalid hexadecimal component value: " + s}
		}
		val[i] = byte(b)
	}
	return val, nil
}

// Returns the input byte slice as-is, indicating that the value is already in the correct hexadecimal format for matching operations.
func (compValFmtHex) ToMatching(val []byte) any {
	return val
}

// Validates and converts an input value to a byte slice for hexadecimal formatting in a component value, returning an error if the input is not a byte slice.
func (compValFmtHex) FromMatching(m any) ([]byte, error) {
	ret, ok := m.([]byte)
	if !ok {
		return nil, ErrFormat{"invalid text component value: " + fmt.Sprintf("%v", m)}
	} else {
		return ret, nil
	}
}

type componentConvention struct {
	typ  TLNum
	name string
	vFmt compValFmt
}

var (
	compConvByType = map[TLNum]*componentConvention{
		TypeImplicitSha256DigestComponent: {
			typ:  TypeImplicitSha256DigestComponent,
			name: DigestShaNameConvention,
			vFmt: compValFmtHex{},
		},
		TypeParametersSha256DigestComponent: {
			typ:  TypeParametersSha256DigestComponent,
			name: ParamShaNameConvention,
			vFmt: compValFmtHex{},
		},
		TypeSegmentNameComponent: {
			typ:  TypeSegmentNameComponent,
			name: "seg",
			vFmt: compValFmtDec{},
		},
		TypeByteOffsetNameComponent: {
			typ:  TypeByteOffsetNameComponent,
			name: "off",
			vFmt: compValFmtDec{},
		},
		TypeVersionNameComponent: {
			typ:  TypeVersionNameComponent,
			name: "v",
			vFmt: compValFmtDec{},
		},
		TypeTimestampNameComponent: {
			typ:  TypeTimestampNameComponent,
			name: "t",
			vFmt: compValFmtDec{},
		},
		TypeSequenceNumNameComponent: {
			typ:  TypeSequenceNumNameComponent,
			name: "seq",
			vFmt: compValFmtDec{},
		},
	}
	compConvByStr map[string]*componentConvention
)

// Initializes a map (`compConvByStr`) that indexes component conventions by their string names for efficient lookup.
func initComponentConventions() {
	compConvByStr = make(map[string]*componentConvention, len(compConvByType))
	for _, c := range compConvByType {
		compConvByStr[c.name] = c
	}
}

// **Description:**  
Returns true if the given byte is a valid character for a URI component, including alphabetic characters, digits, and the special characters '-', '_', '.', and '~'.
func isLegalCompText(b byte) bool {
	return IsAlphabet(rune(b)) || unicode.IsDigit(rune(b)) || b == '-' || b == '_' || b == '.' || b == '~'
}
