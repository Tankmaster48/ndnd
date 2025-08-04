package codegen

import (
	"fmt"
	"strings"
	"text/template"
)

// StructField represents a struct field of another TlvModel.
type StructField struct {
	BaseTlvField

	StructType  string
	innerNoCopy bool
}

// Constructs a TLV struct field with the given name, type number, and annotation, validating nested structure type and nocopy flag compatibility with the provided model.
func NewStructField(name string, typeNum uint64, annotation string, model *TlvModel) (TlvField, error) {
	if annotation == "" {
		return nil, ErrInvalidField
	}
	strs := strings.Split(annotation, ":")
	structType := strs[0]
	innerNoCopy := false
	if len(strs) > 1 && strs[1] == "nocopy" {
		innerNoCopy = true
	}
	if !model.NoCopy && innerNoCopy {
		return nil, ErrInvalidField
	}
	return &StructField{
		BaseTlvField: BaseTlvField{
			name:    name,
			typeNum: typeNum,
		},
		StructType:  structType,
		innerNoCopy: innerNoCopy,
	}, nil
}

// Generates a variable declaration for an encoder struct associated with the field, combining the field's name with "_encoder" and the struct type with "Encoder".  

Example: For a field named "data" of type "MyStruct", returns "data_encoder MyStructEncoder".
func (f *StructField) GenEncoderStruct() (string, error) {
	return fmt.Sprintf("%s_encoder %sEncoder", f.name, f.StructType), nil
}

// Generates code to initialize an encoder for a struct field, ensuring the field is non-nil before calling the encoder's Init method.
func (f *StructField) GenInitEncoder() (string, error) {
	var templ = template.Must(template.New("StructInitEncoder").Parse(`
		if value.{{.}} != nil {
			encoder.{{.}}_encoder.Init(value.{{.}})
		}
	`))
	var g strErrBuf
	g.executeTemplate(templ, f.name)
	return g.output()
}

// Generates a parsing context struct declaration by combining the field's name and type, e.g., "fieldName_context FieldTypeParsingContext" for use in data parsing logic.
func (f *StructField) GenParsingContextStruct() (string, error) {
	return fmt.Sprintf("%s_context %sParsingContext", f.name, f.StructType), nil
}

// Generates a method call to initialize a context-specific struct field using the field's name.  

Example: Returns "context.example_context.Init()" for a field named "example", facilitating context initialization in generated code.
func (f *StructField) GenInitContext() (string, error) {
	return fmt.Sprintf("context.%s_context.Init()", f.name), nil
}

// Generates code to calculate the TLV encoding length for a struct field, including type number, length field, and encoded value when non-nil.
func (f *StructField) GenEncodingLength() (string, error) {
	var g strErrBuf
	g.printlnf("if value.%s != nil {", f.name)
	g.printlne(GenTypeNumLen(f.typeNum))
	g.printlne(GenNaturalNumberLen(fmt.Sprintf("encoder.%s_encoder.Length", f.name), true))
	g.printlnf("l += encoder.%s_encoder.Length", f.name)
	g.printlnf("}")
	return g.output()
}

// Generates Go code to compute the wire encoding length for a struct field, handling nested encodings with multiple buffers or Wire fields while avoiding unnecessary buffer creation when the inner structure is self-contained.
func (f *StructField) GenEncodingWirePlan() (string, error) {
	if f.innerNoCopy {
		var g strErrBuf
		g.printlnf("if value.%s != nil {", f.name)
		g.printlne(GenTypeNumLen(f.typeNum))
		g.printlne(GenNaturalNumberLen(fmt.Sprintf("encoder.%s_encoder.Length", f.name), true))
		g.printlnf("if encoder.%s_encoder.Length > 0 {", f.name)
		// wirePlan[0] is always nonzero.
		g.printlnf("l += encoder.%s_encoder.wirePlan[0]", f.name)
		g.printlnf("for i := 1; i < len(encoder.%s_encoder.wirePlan); i ++ {", f.name)
		g.printlne(GenSwitchWirePlan())
		g.printlnf("l = encoder.%s_encoder.wirePlan[i]", f.name)
		g.printlnf("}")
		// If l == 0 then inner struct ends with a Wire. So we cannot continue.
		// Otherwise, continue on the last part of the inner wire.
		// Therefore, if the inner structure only uses 1 buf (i.e. with no Wire field),
		// the outer structure will not create extra buffers.
		g.printlnf("if l == 0 {")
		g.printlne(GenSwitchWirePlan())
		g.printlnf("}")
		g.printlnf("}")
		g.printlnf("}")
		return g.output()
	} else {
		return f.GenEncodingLength()
	}
}

// Generates Go code to encode a struct field's value into a buffer, using either a direct copy method or a non-copying approach with wire plan indexing depending on the field's configuration.
func (f *StructField) GenEncodeInto() (string, error) {
	var g strErrBuf
	g.printlnf("if value.%s != nil {", f.name)
	g.printlne(GenEncodeTypeNum(f.typeNum))
	g.printlne(GenNaturalNumberEncode(fmt.Sprintf("encoder.%s_encoder.Length", f.name), true))
	g.printlnf("if encoder.%s_encoder.Length > 0 {", f.name)
	if !f.innerNoCopy {
		g.printlnf("encoder.%s_encoder.EncodeInto(value.%s, buf[pos:])", f.name, f.name)
		g.printlnf("pos += encoder.%s_encoder.Length", f.name)
	} else {
		templ := template.Must(template.New("StructEncodeInto").Parse(`{
			subWire := make(enc.Wire, len(encoder.{{.}}_encoder.wirePlan))
			subWire[0] = buf[pos:]
			for i := 1; i < len(subWire); i ++ {
				subWire[i] = wire[wireIdx + i]
			}
			encoder.{{.}}_encoder.EncodeInto(value.{{.}}, subWire)
			for i := 1; i < len(subWire); i ++ {
				wire[wireIdx + i] = subWire[i]
			}
			if lastL := encoder.{{.}}_encoder.wirePlan[len(subWire)-1]; lastL > 0 {
				wireIdx += len(subWire) - 1
				if len(subWire) > 1 {
					pos = lastL
				} else {
					pos += lastL
				}
			} else {
				wireIdx += len(subWire)
				pos = 0
			}
			if wireIdx < len(wire) {
				buf = wire[wireIdx]
			} else {
				buf = nil
			}
		}`))
		g.executeTemplate(templ, f.name)
	}
	g.printlnf("}")
	g.printlnf("}")
	return g.output()
}

// Generates a code snippet that sets the struct field to nil, indicating it should be skipped during processing.
func (f *StructField) GenSkipProcess() (string, error) {
	return fmt.Sprintf("value.%s = nil", f.name), nil
}

// Generates code to parse a struct field's value from a data stream using the corresponding context's Parse method, with length delegation and critical flag handling.
func (f *StructField) GenReadFrom() (string, error) {
	return fmt.Sprintf(
		"value.%[1]s, err = context.%[1]s_context.Parse(reader.Delegate(int(l)), ignoreCritical)",
		f.name,
	), nil
}
