package codegen

import (
	"fmt"
	"strings"
	"text/template"
)

// SequenceField represents a slice field of another supported field type.
type SequenceField struct {
	BaseTlvField

	SubField  TlvField
	FieldType string
}

// Constructs a TLV sequence field with the given name, type number, and sub-field definition parsed from the annotation, using the provided model to create the nested sub-field.
func NewSequenceField(name string, typeNum uint64, annotation string, model *TlvModel) (TlvField, error) {
	strs := strings.SplitN(annotation, ":", 3)
	if len(strs) < 2 {
		return nil, ErrInvalidField
	}
	subFieldType := strs[0]
	subFieldClass := strs[1]
	if len(strs) >= 3 {
		annotation = strs[2]
	} else {
		annotation = ""
	}
	subField, err := CreateField(subFieldClass, name, typeNum, annotation, model)
	if err != nil {
		return nil, err
	}
	return &SequenceField{
		BaseTlvField: BaseTlvField{
			name:    name,
			typeNum: typeNum,
		},
		SubField:  subField,
		FieldType: subFieldType,
	}, nil
}

// Generates a Go struct for encoding the sequence field, nesting its subfield's encoder within a slice of structs.
func (f *SequenceField) GenEncoderStruct() (string, error) {
	g := strErrBuf{}
	g.printlnf("%s_subencoder []struct{", f.name)
	g.printlne(f.SubField.GenEncoderStruct())
	g.printlnf("}")
	return g.output()
}

// Generates Go code to initialize an encoder for a sequence field by iterating over its elements, creating pseudo-encoders and values for each, and recursively initializing subfield encoders.
func (f *SequenceField) GenInitEncoder() (string, error) {
	// Sequence uses faked encoder variable to embed the subfield.
	// I have verified that the Go compiler can optimize this in simple cases.
	templ := template.Must(template.New("SeqInitEncoder").Parse(`
		{
			{{.Name}}_l := len(value.{{.Name}})
			encoder.{{.Name}}_subencoder = make([]struct{
				{{.SubField.GenEncoderStruct}}
			}, {{.Name}}_l)
			for i := 0; i < {{.Name}}_l; i ++ {
				pseudoEncoder := &encoder.{{.Name}}_subencoder[i]
				pseudoValue := struct {
					{{.Name}} {{.FieldType}}
				}{
					{{.Name}}: value.{{.Name}}[i],
				}
				{
					encoder := pseudoEncoder
					value := &pseudoValue
					{{.SubField.GenInitEncoder}}
					_ = encoder
					_ = value
				}
			}
		}
	`))

	var g strErrBuf
	g.executeTemplate(templ, f)
	return g.output()
}

// Generates a parsing context structure for the sub-field type of a sequence, deferring to the sub-field's parsing logic since the number of elements in the sequence is unknown until parsing begins.
func (f *SequenceField) GenParsingContextStruct() (string, error) {
	// This is not a slice, because the number of elements is unknown before parsing.
	return f.SubField.GenParsingContextStruct()
}

// Generates the initialization context string by delegating to the subfield's GenInitContext method.
func (f *SequenceField) GenInitContext() (string, error) {
	return f.SubField.GenInitContext()
}

// Generates code to encode a sequence field by iterating over each element, creating a pseudo-encoder and value struct, and invoking the specified sub-field encoding function for each item in the sequence.
func (f *SequenceField) encodingGeneral(funcName string) (string, error) {
	templ := template.Must(template.New("SequenceEncodingGeneral").Parse(
		fmt.Sprintf(`if value.{{.Name}} != nil {
			for seq_i, seq_v := range value.{{.Name}} {
			pseudoEncoder := &encoder.{{.Name}}_subencoder[seq_i]
			pseudoValue := struct {
				{{.Name}} {{.FieldType}}
			}{
				{{.Name}}: seq_v,
			}
			{
				encoder := pseudoEncoder
				value := &pseudoValue
				{{.SubField.%s}}
				_ = encoder
				_ = value
			}
		}
	}
	`, funcName)))

	var g strErrBuf
	g.executeTemplate(templ, f)
	return g.output()
}

// Generates the encoded length value for the sequence field by invoking a general encoding method with the operation "GenEncodingLength", returning the result as a string and potential error.
func (f *SequenceField) GenEncodingLength() (string, error) {
	return f.encodingGeneral("GenEncodingLength")
}

// Generates a wire encoding plan for the SequenceField using the "GenEncodingWirePlan" encoding strategy.
func (f *SequenceField) GenEncodingWirePlan() (string, error) {
	return f.encodingGeneral("GenEncodingWirePlan")
}

// Generates the code for encoding the SequenceField into a TLV-encoded format using the general "GenEncodeInto" strategy.
func (f *SequenceField) GenEncodeInto() (string, error) {
	return f.encodingGeneral("GenEncodeInto")
}

// Generates code to read a sequence of elements into a slice field by initializing the slice if empty, using a pseudo-struct to hold each element's value during parsing, and appending the parsed element to the slice after processing its subfield.
func (f *SequenceField) GenReadFrom() (string, error) {
	templ := template.Must(template.New("NameEncodeInto").Parse(`
		if value.{{.Name}} == nil {
			value.{{.Name}} = make([]{{.FieldType}}, 0)
		}
		{
			pseudoValue := struct {
				{{.Name}} {{.FieldType}}
			}{}
			{
				value := &pseudoValue
				{{.SubField.GenReadFrom}}
				_ = value
			}
			value.{{.Name}} = append(value.{{.Name}}, pseudoValue.{{.Name}})
		}
		progress --
	`))

	var g strErrBuf
	g.executeTemplate(templ, f)
	return g.output()
}

// Returns a comment string "// sequence - skip" to indicate that processing of the sequence field is skipped after all elements have been parsed, ensuring no nil assignment.
func (f *SequenceField) GenSkipProcess() (string, error) {
	// Skip is called after all elements are parsed, so we should not assign nil.
	return "// sequence - skip", nil
}
