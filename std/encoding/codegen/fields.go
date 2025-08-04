package codegen

import (
	"errors"
	"strings"
)

var ErrInvalidField = errors.New("invalid TlvField. Please check the annotation (including type and arguments)")
var ErrWrongTypeNumber = errors.New("invalid type number")

type TlvField interface {
	Name() string
	TypeNum() uint64

	// codegen encoding length of the field
	//   - in(value): struct being encoded
	//   - out(l): length variable to update
	GenEncodingLength() (string, error)
	GenEncodingWirePlan() (string, error)
	GenEncodeInto() (string, error)
	GenEncoderStruct() (string, error)
	GenInitEncoder() (string, error)
	GenParsingContextStruct() (string, error)
	GenInitContext() (string, error)
	GenReadFrom() (string, error)
	GenSkipProcess() (string, error)
	GenToDict() (string, error)
	GenFromDict() (string, error)
}

// BaseTlvField is a base class for all TLV fields.
// Golang's inheritance is not supported, so we use this class to disable
// optional functions.
type BaseTlvField struct {
	name    string
	typeNum uint64
}

// Returns the name of the TLV field.
func (f *BaseTlvField) Name() string {
	return f.name
}

// Returns the TLV type number associated with this field.
func (f *BaseTlvField) TypeNum() uint64 {
	return f.typeNum
}

// Generates the Length component of the TLV encoding for the field, returning an empty string by default as this base implementation expects subclasses to override with specific length calculation logic.
func (*BaseTlvField) GenEncodingLength() (string, error) {
	return "", nil
}

// Generates a wire encoding plan for TLV (Type-Length-Value) serialization, but as the base implementation returns an empty string and no error, it serves as a placeholder for derived types to override with their specific encoding logic.
func (*BaseTlvField) GenEncodingWirePlan() (string, error) {
	return "", nil
}

// Generates an empty TLV-encoded string for the base field, intended to be overridden by derived classes implementing specific encoding logic.
func (*BaseTlvField) GenEncodeInto() (string, error) {
	return "", nil
}

// Generates the encoder structure for the TLV field, returning an empty string in the base implementation; intended to be overridden by derived types.
func (*BaseTlvField) GenEncoderStruct() (string, error) {
	return "", nil
}

// Generates initialization code for a TLV encoder, typically overridden by specific field types to provide custom encoder setup logic.
func (*BaseTlvField) GenInitEncoder() (string, error) {
	return "", nil
}

// Generates a parsing context struct for TLV decoding, returning an empty string as the base class has no specific parsing context requirements.
func (*BaseTlvField) GenParsingContextStruct() (string, error) {
	return "", nil
}

// Generates an initialization context string for TLV encoding/decoding, returning an empty string by default as a base implementation to be overridden by derived classes.
func (*BaseTlvField) GenInitContext() (string, error) {
	return "", nil
}

// Generates code for reading data from a TLV (Type-Length-Value) field, returning an empty string by default to indicate this base class should be overridden by specific implementations.
func (*BaseTlvField) GenReadFrom() (string, error) {
	return "", nil
}

// Returns a comment indicating that processing for the base TLV field should be skipped, allowing subclasses to handle their own TLV encoding/decoding logic.
func (*BaseTlvField) GenSkipProcess() (string, error) {
	return "// base - skip", nil
}

// Generates an empty dictionary representation for a TLV field, intended to be overridden by derived classes for specific encoding logic.
func (*BaseTlvField) GenToDict() (string, error) {
	return "", nil
}

// Generates a string representation of the TLV field from a dictionary, returning an empty string as this base implementation is intended to be overridden by derived types.
func (*BaseTlvField) GenFromDict() (string, error) {
	return "", nil
}

// Creates a TLV field of the specified class type (e.g., "natural", "string", "struct") with the given name, type number, annotation, and model configuration.
func CreateField(className string, name string, typeNum uint64, annotation string, model *TlvModel) (TlvField, error) {
	fieldList := map[string]func(string, uint64, string, *TlvModel) (TlvField, error){
		"natural":           NewNaturalField,
		"byte":              NewByteField,
		"fixedUint":         NewFixedUintField,
		"time":              NewTimeField,
		"binary":            NewBinaryField,
		"string":            NewStringField,
		"wire":              NewWireField,
		"name":              NewNameField,
		"bool":              NewBoolField,
		"procedureArgument": NewProcedureArgument,
		"offsetMarker":      NewOffsetMarker,
		"rangeMarker":       NewRangeMarker,
		"sequence":          NewSequenceField,
		"struct":            NewStructField,
		"signature":         NewSignatureField,
		"interestName":      NewInterestNameField,
		"map":               NewMapField,
	}

	for k, f := range fieldList {
		if strings.HasPrefix(className, k) {
			return f(name, typeNum, annotation, model)
		}
	}
	return nil, ErrInvalidField
}
