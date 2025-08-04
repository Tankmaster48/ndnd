package encoding

import "time"

// VersionImmutable is the version number for immutable objects.
// A version number of 0 will be used on the wire.
const VersionImmutable = uint64(0)

// VersionUnixMicro is the version number for objects with a unix timestamp.
// A version number of microseconds since the unix epoch will be used on the wire.
// Current unix time must be positive, or usage will panic.
const VersionUnixMicro = uint64(1<<63 - 16)

// Constructs a new NDN name component with the specified type number and byte slice value.
func NewBytesComponent(typ TLNum, val []byte) Component {
	return Component{
		Typ: typ,
		Val: val,
	}
}

// Constructs a new Component with the specified TLNum type and string value, converting the string to a byte slice for storage.
func NewStringComponent(typ TLNum, val string) Component {
	return Component{
		Typ: typ,
		Val: []byte(val),
	}
}

// Constructs an NDN component of the specified TLNum type with the given unsigned 64-bit integer value, encoding the value as bytes in the component's payload.
func NewNumberComponent(typ TLNum, val uint64) Component {
	return Component{
		Typ: typ,
		Val: Nat(val).Bytes(),
	}
}

// Constructs a generic name component with the specified string value.
func NewGenericComponent(val string) Component {
	return NewStringComponent(TypeGenericNameComponent, val)
}

// Constructs a generic NDN name component from the provided byte slice using the generic name component type.
func NewGenericBytesComponent(val []byte) Component {
	return NewBytesComponent(TypeGenericNameComponent, val)
}

// Constructs a name component of keyword type using the provided string value.
func NewKeywordComponent(val string) Component {
	return NewStringComponent(TypeKeywordNameComponent, val)
}

// Constructs a segment name component with the given segment number for use in Named Data Networking names.
func NewSegmentComponent(seg uint64) Component {
	return NewNumberComponent(TypeSegmentNameComponent, seg)
}

// Constructs a byte offset name component with the given uint64 offset value for use in NDN names.
func NewByteOffsetComponent(off uint64) Component {
	return NewNumberComponent(TypeByteOffsetNameComponent, off)
}

// Constructs a name component of type sequence number using the provided unsigned 64-bit integer value.
func NewSequenceNumComponent(seq uint64) Component {
	return NewNumberComponent(TypeSequenceNumNameComponent, seq)
}

// Constructs an NDN Name component of type TypeVersionNameComponent with the specified numeric version value.
func NewVersionComponent(v uint64) Component {
	return NewNumberComponent(TypeVersionNameComponent, v)
}

// Constructs a timestamp name component using the specified 64-bit unsigned integer value.
func NewTimestampComponent(t uint64) Component {
	return NewNumberComponent(TypeTimestampNameComponent, t)
}

// Returns true if the component is a generic name component with the specified text value.
func (c Component) IsGeneric(text string) bool {
	return c.Typ == TypeGenericNameComponent && string(c.Val) == text
}

// Returns true if the component is of keyword type and its value matches the specified keyword string.
func (c Component) IsKeyword(keyword string) bool {
	return c.Typ == TypeKeywordNameComponent && string(c.Val) == keyword
}

// Returns true if the component is a segment name component.
func (c Component) IsSegment() bool {
	return c.Typ == TypeSegmentNameComponent
}

// Returns true if the component is of type TypeByteOffsetNameComponent, indicating it represents a byte offset within a name.
func (c Component) IsByteOffset() bool {
	return c.Typ == TypeByteOffsetNameComponent
}

// Returns true if the component is a sequence number type used in Named Data Networking names for ordered versioning.
func (c Component) IsSequenceNum() bool {
	return c.Typ == TypeSequenceNumNameComponent
}

// "Returns true if the component's type is TypeVersionNameComponent, indicating it represents a version identifier in an NDN name."
func (c Component) IsVersion() bool {
	return c.Typ == TypeVersionNameComponent
}

// Returns true if the component is of type TypeTimestampNameComponent.
func (c Component) IsTimestamp() bool {
	return c.Typ == TypeTimestampNameComponent
}

// WithVersion appends a version component to the name.
func (n Name) WithVersion(v uint64) Name {
	if n.At(-1).IsVersion() {
		n = n.Prefix(-1) // pop old version
	}
	switch v {
	case VersionImmutable:
		v = 0
	case VersionUnixMicro:
		if now := time.Now().UnixMicro(); now > 0 { // > 1970
			v = uint64(now)
		} else {
			panic("current unix time is negative")
		}
	}
	return n.Append(NewVersionComponent(v))
}
