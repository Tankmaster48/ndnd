package codegen

import "strings"

// ProcedureArgument is a variable used during encoding and decoding procedure.
type ProcedureArgument struct {
	BaseTlvField

	argType string
}

// Generates a string representation of an encoder struct by concatenating the argument's name and type for use in encoding procedures.
func (f *ProcedureArgument) GenEncoderStruct() (string, error) {
	return f.name + " " + f.argType, nil
}

// Generates a parsing context struct definition as a string by combining the argument's name and type.
func (f *ProcedureArgument) GenParsingContextStruct() (string, error) {
	return f.name + " " + f.argType, nil
}

// NewProcedureArgument creates a ProcedureArgument field.
func NewProcedureArgument(name string, _ uint64, annotation string, _ *TlvModel) (TlvField, error) {
	return &ProcedureArgument{
		BaseTlvField: BaseTlvField{
			name:    name,
			typeNum: 0,
		},
		argType: annotation,
	}, nil
}

// OffsetMarker is a marker that marks a position in the wire.
type OffsetMarker struct {
	BaseTlvField

	noCopy bool
}

// Generates a Go struct definition for an encoder, including fields for the offset marker's position and optional wire index based on configuration.
func (f *OffsetMarker) GenEncoderStruct() (string, error) {
	g := strErrBuf{}
	g.printlnf("%s int", f.name)
	if f.noCopy {
		g.printlnf("%s_wireIdx int", f.name)
	}
	g.printlnf("%s_pos int", f.name)
	return g.output()
}

// Generates a struct field declaration string for a parsing context using the OffsetMarker's name and an integer type.  

Example: If the OffsetMarker's name is "offset", returns "offset int".
func (f *OffsetMarker) GenParsingContextStruct() (string, error) {
	return f.name + " " + "int", nil
}

// Generates code to skip processing during data reading by delegating to `GenSkipProcess`.
func (f *OffsetMarker) GenReadFrom() (string, error) {
	return f.GenSkipProcess()
}

// Generates a code snippet assigning the integer value of `startPos` to a context variable named after the marker's identifier.
func (f *OffsetMarker) GenSkipProcess() (string, error) {
	return "context." + f.name + " = int(startPos)", nil
}

// Generates a code snippet assigning the integer length `l` to an encoder field named after the OffsetMarker's associated data field.
func (f *OffsetMarker) GenEncodingLength() (string, error) {
	return "encoder." + f.name + " = int(l)", nil
}

// Generates code to encode offset information into an encoder struct, setting the wire index if copying is enabled and always setting the position.
func (f *OffsetMarker) GenEncodeInto() (string, error) {
	g := strErrBuf{}
	if f.noCopy {
		g.printlnf("encoder.%s_wireIdx = int(wireIdx)", f.name)
	}
	g.printlnf("encoder.%s_pos = int(pos)", f.name)
	return g.output()
}

// NewOffsetMarker creates an offset marker field.
func NewOffsetMarker(name string, _ uint64, _ string, model *TlvModel) (TlvField, error) {
	return &OffsetMarker{
		BaseTlvField: BaseTlvField{
			name:    name,
			typeNum: 0,
		},
		noCopy: model.NoCopy,
	}, nil
}

// RangeMarker is a marker that catches a range in the wire from an OffsetMarker to current position.
// It is necessary because the offset given by OffsetMarker is not necessarily from the beginning of the outmost TLV,
// when parsing. It is the same with OffsetMarker for encoding.
type RangeMarker struct {
	BaseTlvField

	noCopy     bool
	startPoint string
	sigCovered string
}

// Generates a Go struct definition as a string with fields for encoding, including an integer field for the named range, an optional wire index field if noCopy is enabled, and a position field.
func (f *RangeMarker) GenEncoderStruct() (string, error) {
	g := strErrBuf{}
	g.printlnf("%s int", f.name)
	if f.noCopy {
		g.printlnf("%s_wireIdx int", f.name)
	}
	g.printlnf("%s_pos int", f.name)
	return g.output()
}

// Generates a string to assign the integer length value `l` to the encoder's field corresponding to this RangeMarker's name.
func (f *RangeMarker) GenEncodingLength() (string, error) {
	return "encoder." + f.name + " = int(l)", nil
}

// Generates code to set encoder state variables (wire index and position) for a RangeMarker during encoding, conditionally avoiding data copying when configured.  

Example:  
`Generates code to update encoder state with wire index and position for efficient data encoding.`
func (f *RangeMarker) GenEncodeInto() (string, error) {
	g := strErrBuf{}
	if f.noCopy {
		g.printlnf("encoder.%s_wireIdx = int(wireIdx)", f.name)
	}
	g.printlnf("encoder.%s_pos = int(pos)", f.name)
	return g.output()
}

// Generates a Go struct field declaration for a parsing context with the specified name and integer type.
func (f *RangeMarker) GenParsingContextStruct() (string, error) {
	g := strErrBuf{}
	g.printlnf("%s int", f.name)
	return g.output()
}

// Delegates to `GenSkipProcess()` to generate a string representation for skipping processing steps, typically used in data segmentation or range-based reading scenarios.
func (f *RangeMarker) GenReadFrom() (string, error) {
	return f.GenSkipProcess()
}

// Generates code to record the start position in the context and process a range using the reader's Range method for skipping data.
func (f *RangeMarker) GenSkipProcess() (string, error) {
	g := strErrBuf{}
	g.printlnf("context.%s = int(startPos)", f.name)
	g.printlnf("context.%s = reader.Range(context.%s, startPos)", f.sigCovered, f.startPoint)
	return g.output()
}

// NewOffsetMarker creates an range marker field.
func NewRangeMarker(name string, typeNum uint64, annotation string, model *TlvModel) (TlvField, error) {
	strs := strings.Split(annotation, ":")
	if len(strs) < 2 || strs[0] == "" || strs[1] == "" {
		return nil, ErrInvalidField
	}
	return &RangeMarker{
		BaseTlvField: BaseTlvField{
			name:    name,
			typeNum: typeNum,
		},
		startPoint: strs[0],
		sigCovered: strs[1],
		noCopy:     model.NoCopy,
	}, nil
}
