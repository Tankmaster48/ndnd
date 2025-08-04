package encoding

import (
	"bytes"
	"io"
	"os"
	"slices"
	"strconv"
	"strings"
)

const (
	TypeInvalidComponent                TLNum = 0x00
	TypeImplicitSha256DigestComponent   TLNum = 0x01
	TypeParametersSha256DigestComponent TLNum = 0x02
	TypeGenericNameComponent            TLNum = 0x08
	TypeKeywordNameComponent            TLNum = 0x20
	TypeSegmentNameComponent            TLNum = 0x32
	TypeByteOffsetNameComponent         TLNum = 0x34
	TypeVersionNameComponent            TLNum = 0x36
	TypeTimestampNameComponent          TLNum = 0x38
	TypeSequenceNumNameComponent        TLNum = 0x3a
)

const (
	ParamShaNameConvention  = "params-sha256"
	DigestShaNameConvention = "sha256digest"
)

var (
	HEX_LOWER = []rune("0123456789abcdef")
	HEX_UPPER = []rune("0123456789ABCDEF")
)

var DISABLE_ALT_URI = os.Getenv("NDN_NAME_ALT_URI") == "0"

type Component struct {
	Typ TLNum
	Val []byte
}

// This method returns the component as a `ComponentPattern` interface, enabling its use as a pattern-matching component in NDN name operations.
func (c Component) ComponentPatternTrait() ComponentPattern {
	return c
}

// Creates a deep copy of the Component by duplicating its Val slice.
func (c Component) Clone() Component {
	return Component{
		Typ: c.Typ,
		Val: slices.Clone(c.Val),
	}
}

// Returns the length of the component's value as a TLNum for TLV encoding.
func (c Component) Length() TLNum {
	return TLNum(len(c.Val))
}

// Returns the string representation of the component by writing its contents to a strings.Builder.
func (c Component) String() string {
	sb := strings.Builder{}
	c.WriteTo(&sb)
	return sb.String()
}

// Serializes the component's type and value into the provided string builder in a format that may use alternative URI representations for the type, returning the total number of bytes written.
func (c Component) WriteTo(sb *strings.Builder) int {
	size := 0

	vFmt := compValFmt(compValFmtText{})
	if conv, ok := compConvByType[c.Typ]; !DISABLE_ALT_URI && ok {
		vFmt = conv.vFmt
		typ := conv.name
		sb.WriteString(typ)
		sb.WriteRune('=')
		size += len(typ) + 1
	} else if DISABLE_ALT_URI || c.Typ != TypeGenericNameComponent {
		typ := strconv.FormatUint(uint64(c.Typ), 10)
		sb.WriteString(typ)
		sb.WriteRune('=')
		size += len(typ) + 1
	}

	size += vFmt.WriteTo(c.Val, sb)
	return size
}

// Constructs the canonical string representation of a name component, appending the component's type as "type=" (if non-generic) followed by the formatted value.
func (c Component) CanonicalString() string {
	sb := strings.Builder{}
	if c.Typ != TypeGenericNameComponent {
		sb.WriteString(strconv.FormatUint(uint64(c.Typ), 10))
		sb.WriteRune('=')
	}
	compValFmtText{}.WriteTo(c.Val, &sb)
	return sb.String()
}

// Constructs a new Name by appending the specified components to the initial component.
func (c Component) Append(rest ...Component) Name {
	return Name{c}.Append(rest...)
}

// Returns the total number of bytes required to encode the component, summing the encoded lengths of its type, the length of its value (as a natural number), and the value itself.
func (c Component) EncodingLength() int {
	l := len(c.Val)
	return c.Typ.EncodingLength() + Nat(l).EncodingLength() + l
}

// Encodes the component's type and variable-length value into the provided buffer, returning the total number of bytes written (type encoding + value length encoding + value data).
func (c Component) EncodeInto(buf Buffer) int {
	p1 := c.Typ.EncodeInto(buf)
	p2 := Nat(len(c.Val)).EncodeInto(buf[p1:])
	copy(buf[p1+p2:], c.Val)
	return p1 + p2 + len(c.Val)
}

// Encodes the component into a byte slice by allocating a buffer of the appropriate size and writing the encoded data into it.
func (c Component) Bytes() []byte {
	buf := make([]byte, c.EncodingLength())
	c.EncodeInto(buf)
	return buf
}

// Compares a Component with another ComponentPattern, returning -1, 0, or 1 based on type hierarchy, value length, and byte-wise comparison of their values, with Components considered less than non-Component patterns.
func (c Component) Compare(rhs ComponentPattern) int {
	rc, ok := rhs.(Component)
	if !ok {
		p, ok := rhs.(*Component)
		if !ok {
			// Component is always less than pattern
			return -1
		}
		rc = *p
	}
	if c.Typ != rc.Typ {
		if c.Typ < rc.Typ {
			return -1
		} else {
			return 1
		}
	}
	if len(c.Val) != len(rc.Val) {
		if len(c.Val) < len(rc.Val) {
			return -1
		} else {
			return 1
		}
	}
	return bytes.Compare(c.Val, rc.Val)
}

// NumberVal returns the value of the component as a number
func (c Component) NumberVal() uint64 {
	ret := uint64(0)
	for _, v := range c.Val {
		ret = (ret << 8) | uint64(v)
	}
	return ret
}

// Hash returns the hash of the component
func (c Component) Hash() uint64 {
	xx := xxHashPool.Get()
	defer xxHashPool.Put(xx)

	size := c.EncodingLength()
	xx.buffer.Grow(size)
	buf := xx.buffer.AvailableBuffer()[:size]
	c.EncodeInto(buf)

	xx.hash.Write(buf)
	return xx.hash.Sum64()
}

// Compares two NDN name components for equality by checking their type and byte value, handling both value and pointer interface implementations.
func (c Component) Equal(rhs ComponentPattern) bool {
	// Go's strange design leads the the result that both Component and *Component implements this interface
	// And it is nearly impossible to predict what is what.
	// So we have to try to cast twice to get the correct result.
	rc, ok := rhs.(Component)
	if !ok {
		p, ok := rhs.(*Component)
		if !ok {
			return false
		}
		rc = *p
	}
	if c.Typ != rc.Typ || len(c.Val) != len(rc.Val) {
		return false
	}
	return bytes.Equal(c.Val, rc.Val)
}

// Implements component matching logic by comparing the receiver with the given value using the specified Matching rules.
func (Component) Match(value Component, m Matching) {}

// Returns a pointer to the existing Component without modifying or utilizing the provided Matching parameter.
func (c Component) FromMatching(m Matching) (*Component, error) {
	return &c, nil
}

// Returns true if the component is equal to the specified component.
func (c Component) IsMatch(value Component) bool {
	return c.Equal(value)
}

// Parses a string into an NDN name Component, returning an error if the input is invalid.
func ComponentFromStr(s string) (Component, error) {
	ret := Component{}
	err := componentFromStrInto(s, &ret)
	if err != nil {
		return Component{}, err
	} else {
		return ret, nil
	}
}

// Parses a name component from the provided byte slice, returning the decoded component and any error encountered during parsing.
func ComponentFromBytes(buf []byte) (Component, error) {
	r := NewBufferView(buf)
	return r.ReadComponent()
}

// Parses a component from the buffer by reading type and length fields, then extracting the corresponding value, returning the component and the total number of bytes consumed.
func ParseComponent(buf Buffer) (Component, int) {
	typ, p1 := ParseTLNum(buf)
	l, p2 := ParseTLNum(buf[p1:])
	start := p1 + p2
	end := start + int(l)
	return Component{
		Typ: typ,
		Val: buf[start:end],
	}, end
}

// Reads a Component from the wire format by parsing its type, length, and value, returning the component and any error encountered.
func (r *WireView) ReadComponent() (Component, error) {
	typ, err := r.ReadTLNum()
	if err != nil {
		return Component{}, err
	}
	l, err := r.ReadTLNum()
	if err != nil {
		if err == io.EOF {
			err = io.ErrUnexpectedEOF
		}
		return Component{}, err
	}
	val, err := r.ReadBuf(int(l))
	if err != nil {
		return Component{}, err
	}
	return Component{
		Typ: typ,
		Val: val,
	}, nil
}

// Parses a component type string into a TL number and value format, supporting named types (e.g., "NAME") via a predefined mapping or numeric types, returning errors for invalid or unrecognized inputs.
func parseCompTypeFromStr(s string) (TLNum, compValFmt, error) {
	if IsAlphabet(rune(s[0])) {
		if conv, ok := compConvByStr[s]; ok {
			return conv.typ, conv.vFmt, nil
		} else {
			return 0, compValFmtInvalid{}, ErrFormat{"unknown component type: " + s}
		}
	} else {
		typInt, err := strconv.ParseUint(s, 10, 64)
		if err != nil {
			return 0, compValFmtInvalid{}, ErrFormat{"invalid component type: " + s}
		}
		return TLNum(typInt), compValFmtText{}, nil
	}
}

// Parses a string into a Component, allowing an optional type prefix separated by '=', and populates the provided Component struct with the parsed type and value.
func componentFromStrInto(s string, ret *Component) error {
	var err error
	hasEq := false
	typStr := ""
	valStr := s
	for i, c := range s {
		if c == '=' {
			if !hasEq {
				typStr = s[:i]
				valStr = s[i+1:]
			} else {
				return ErrFormat{"too many '=' in component: " + s}
			}
			hasEq = true
		}
	}
	ret.Typ = TypeGenericNameComponent
	vFmt := compValFmt(compValFmtText{})
	ret.Val = []byte(nil)
	if hasEq {
		ret.Typ, vFmt, err = parseCompTypeFromStr(typStr)
		if err != nil {
			return err
		}
		if ret.Typ <= TypeInvalidComponent || ret.Typ > 0xffff {
			return ErrFormat{"invalid component type: " + valStr}
		}
	}
	ret.Val, err = vFmt.FromString(valStr)
	if err != nil {
		return err
	}
	return nil
}
