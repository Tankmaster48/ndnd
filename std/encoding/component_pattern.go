package encoding

import (
	"fmt"
	"strings"
)

type Matching map[string][]byte

type ComponentPattern interface {
	// ComponentPatternTrait returns the type trait of Component or Pattern
	// This is used to make ComponentPattern a union type of Component or Pattern
	// Component | Pattern does not work because we need a mixed list NamePattern
	ComponentPatternTrait() ComponentPattern

	// String returns the string of the component, with naming conventions.
	// Since naming conventions are not standardized, this should not be used for purposes other than logging.
	// please use CanonicalString() for stable string representation.
	String() string

	// CanonicalString returns the string representation of the component without naming conventions.
	CanonicalString() string

	// Compare returns an integer comparing two components lexicographically.
	// It compares the type number first, and then its value.
	// A component is always less than a pattern.
	// The result will be 0 if a == b, -1 if a < b, and +1 if a > b.
	Compare(ComponentPattern) int

	// Equal returns the two components/patterns are the same.
	Equal(ComponentPattern) bool

	// IsMatch returns if the Component value matches with the current component/pattern.
	IsMatch(value Component) bool

	// Match matches the current pattern/component with the value, and put the matching into the Matching map.
	Match(value Component, m Matching)

	// FromMatching initiates the pattern from the Matching map.
	FromMatching(m Matching) (*Component, error)
}

// Parses a string into a ComponentPattern, interpreting angle-bracket-enclosed strings with optional type=tag syntax (e.g., `<type=tag>` or `<tag>`) and falling back to regular components for non-pattern strings.
func ComponentPatternFromStr(s string) (ComponentPattern, error) {
	if len(s) <= 0 || s[0] != '<' {
		return ComponentFromStr(s)
	}
	if s[len(s)-1] != '>' {
		return nil, ErrFormat{"invalid component pattern: " + s}
	}
	s = s[1 : len(s)-1]
	strs := strings.Split(s, "=")
	if len(strs) > 2 {
		return nil, ErrFormat{"too many '=' in component pattern: " + s}
	}
	if len(strs) == 2 {
		typ, _, err := parseCompTypeFromStr(strs[0])
		if err != nil {
			return nil, err
		}
		return Pattern{
			Typ: typ,
			Tag: strs[1],
		}, nil
	} else {
		return Pattern{
			Typ: TypeGenericNameComponent,
			Tag: strs[0],
		}, nil
	}
}

type Pattern struct {
	Typ TLNum
	Tag string
}

// Returns a string representation of the Pattern, formatting it as `<tag>`, `<conversion=tag>`, or `<type=tag>` depending on its type and registered conversion name.
func (p Pattern) String() string {
	if p.Typ == TypeGenericNameComponent {
		return "<" + p.Tag + ">"
	} else if conv, ok := compConvByType[p.Typ]; ok {
		return "<" + conv.name + "=" + p.Tag + ">"
	} else {
		return fmt.Sprintf("<%d=%s>", p.Typ, p.Tag)
	}
}

// Returns a canonical string representation of the pattern, using the tag alone for generic components or the type and tag for other patterns.
func (p Pattern) CanonicalString() string {
	if p.Typ == TypeGenericNameComponent {
		return "<" + p.Tag + ">"
	} else {
		return fmt.Sprintf("<%d=%s>", p.Typ, p.Tag)
	}
}

// **Description:**  
Returns the receiver Pattern as a ComponentPattern, enabling type conversion or interface implementation for pattern-related operations.
func (p Pattern) ComponentPatternTrait() ComponentPattern {
	return p
}

// Compares this Pattern with another ComponentPattern, returning -1, 0, or 1 based on type hierarchy and tag string comparison, with Pattern types considered greater than non-Pattern components.
func (p Pattern) Compare(rhs ComponentPattern) int {
	rp, ok := rhs.(Pattern)
	if !ok {
		p, ok := rhs.(*Pattern)
		if !ok {
			// Pattern is always greater than component
			return 1
		}
		rp = *p
	}
	if p.Typ != rp.Typ {
		if p.Typ < rp.Typ {
			return -1
		} else {
			return 1
		}
	}
	return strings.Compare(p.Tag, rp.Tag)
}

// Compares two ComponentPattern instances for equality by checking if they are both Pattern (or *Pattern) and have the same Typ and Tag fields.
func (p Pattern) Equal(rhs ComponentPattern) bool {
	rp, ok := rhs.(Pattern)
	if !ok {
		p, ok := rhs.(*Pattern)
		if !ok {
			return false
		}
		rp = *p
	}
	return p.Typ == rp.Typ && p.Tag == rp.Tag
}

// Stores the byte value of the component in the matching map under the key specified by the pattern's tag.
func (p Pattern) Match(value Component, m Matching) {
	m[p.Tag] = make([]byte, len(value.Val))
	copy(m[p.Tag], value.Val)
}

// Constructs a Component using the pattern's type and the value associated with its tag from a matching result, or returns an error if the tag is not found.
func (p Pattern) FromMatching(m Matching) (*Component, error) {
	val, ok := m[p.Tag]
	if !ok {
		return nil, ErrNotFound{p.Tag}
	}
	return &Component{
		Typ: p.Typ,
		Val: []byte(val),
	}, nil
}

// Returns true if the given Component has the same type as the Pattern.
func (p Pattern) IsMatch(value Component) bool {
	return p.Typ == value.Typ
}
