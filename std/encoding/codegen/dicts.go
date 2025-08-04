package codegen

import "text/template"

// Generates code to add a field's value to a dictionary, conditionally handling optional fields by checking their presence before insertion.
func (f *NaturalField) GenToDict() (string, error) {
	g := strErrBuf{}
	if f.opt {
		g.printlnf("if optval, ok := value.%s.Get(); ok {", f.name)
		g.printlnf("\tdict[\"%s\"] = optval", f.name)
		g.printlnf("}")
	} else {
		g.printlnf("dict[\"%s\"] = value.%s", f.name, f.name)
	}
	return g.output()
}

// Generates Go code to parse a dictionary entry into a struct field, performing type checks for uint64 values, handling optional fields, and returning appropriate errors for missing or incompatible data.
func (f *NaturalField) GenFromDict() (string, error) {
	g := strErrBuf{}
	g.printlnf("if vv, ok := dict[\"%s\"]; ok {", f.name)
	g.printlnf("\tif v, ok := vv.(uint64); ok {")
	if f.opt {
		g.printlnf("\t\tvalue.%s.Set(v)", f.name)
	} else {
		g.printlnf("\t\tvalue.%s = v", f.name)
	}
	g.printlnf("\t} else {")
	g.printlnf("\t\terr = enc.ErrIncompatibleType{Name: \"%s\", TypeNum: %d, ValType: \"uint64\", Value: vv}", f.name, f.typeNum)
	g.printlnf("\t}")
	g.printlnf("} else {")
	if f.opt {
		g.printlnf("\tvalue.%s.Unset()", f.name)
	} else {
		g.printlnf("err = enc.ErrSkipRequired{Name: \"%s\", TypeNum: %d}", f.name, f.typeNum)
	}
	g.printlnf("}")
	return g.output()
}

// Generates code to populate a dictionary with a field's value, conditionally handling optional fields by checking their presence before assignment.
func (f *StringField) GenToDict() (string, error) {
	g := strErrBuf{}
	if f.opt {
		g.printlnf("if optval, ok := value.%s.Get(); ok {", f.name)
		g.printlnf("\tdict[\"%s\"] = optval", f.name)
		g.printlnf("}")
	} else {
		g.printlnf("dict[\"%s\"] = value.%s", f.name, f.name)
	}
	return g.output()
}

// Generates Go code to extract a string field from a dictionary, handling optional/required semantics and type compatibility checks.
func (f *StringField) GenFromDict() (string, error) {
	g := strErrBuf{}
	g.printlnf("if vv, ok := dict[\"%s\"]; ok {", f.name)
	g.printlnf("\tif v, ok := vv.(string); ok {")
	if f.opt {
		g.printlnf("\t\tvalue.%s.Set(v)", f.name)
	} else {
		g.printlnf("\t\tvalue.%s = v", f.name)
	}
	g.printlnf("\t} else {")
	g.printlnf("\t\terr = enc.ErrIncompatibleType{Name: \"%s\", TypeNum: %d, ValType: \"string\", Value: vv}", f.name, f.typeNum)
	g.printlnf("\t}")
	g.printlnf("} else {")
	if f.opt {
		g.printlnf("\tvalue.%s.Unset()", f.name)
	} else {
		g.printlnf("err = enc.ErrSkipRequired{Name: \"%s\", TypeNum: %d}", f.name, f.typeNum)
	}
	g.printlnf("}")
	return g.output()
}

// Generates Go code that conditionally adds a non-nil field to a dictionary map using the field's name as both key and value.
func (f *BinaryField) GenToDict() (string, error) {
	g := strErrBuf{}
	g.printlnf("if value.%s != nil {", f.name)
	g.printlnf("\tdict[\"%s\"] = value.%s", f.name, f.name)
	g.printlnf("}")
	return g.output()
}

// Generates code to safely extract a binary slice field from a dictionary into a struct, handling type mismatches and missing keys with appropriate error reporting.
func (f *BinaryField) GenFromDict() (string, error) {
	g := strErrBuf{}
	g.printlnf("if vv, ok := dict[\"%s\"]; ok {", f.name)
	g.printlnf("\tif v, ok := vv.([]byte); ok {")
	g.printlnf("\t\tvalue.%s = v", f.name)
	g.printlnf("\t} else {")
	g.printlnf("\t\terr = enc.ErrIncompatibleType{Name: \"%s\", TypeNum: %d, ValType: \"[]byte\", Value: vv}", f.name, f.typeNum)
	g.printlnf("\t}")
	g.printlnf("} else {")
	g.printlnf("\tvalue.%s = nil", f.name)
	g.printlnf("}")
	return g.output()
}

// Generates a code snippet that assigns a boolean field's value to a dictionary entry using the field's name as both the key and the source struct field.
func (f *BoolField) GenToDict() (string, error) {
	g := strErrBuf{}
	g.printlnf("dict[\"%s\"] = value.%s", f.name, f.name)
	return g.output()
}

// Generates Go code to extract a boolean field from a map[string]interface{} dictionary, performing type checking, setting a default value of false if the key is missing, and reporting errors for incompatible types.
func (f *BoolField) GenFromDict() (string, error) {
	g := strErrBuf{}
	g.printlnf("if vv, ok := dict[\"%s\"]; ok {", f.name)
	g.printlnf("\tif v, ok := vv.(bool); ok {")
	g.printlnf("\t\tvalue.%s = v", f.name)
	g.printlnf("\t} else {")
	g.printlnf("\t\terr = enc.ErrIncompatibleType{Name: \"%s\", TypeNum: %d, ValType: \"bool\", Value: vv}", f.name, f.typeNum)
	g.printlnf("\t}")
	g.printlnf("} else {")
	g.printlnf("\tvalue.%s = false", f.name)
	g.printlnf("}")
	return g.output()
}

// Generates code to conditionally add a non-nil field to a dictionary with its name as the key.
func (f *NameField) GenToDict() (string, error) {
	g := strErrBuf{}
	g.printlnf("if value.%s != nil {", f.name)
	g.printlnf("\tdict[\"%s\"] = value.%s", f.name, f.name)
	g.printlnf("}")
	return g.output()
}

// Generates code to extract and validate a Name field from a dictionary, assigning it to the struct field if the type is correct or returning an error if incompatible.
func (f *NameField) GenFromDict() (string, error) {
	g := strErrBuf{}
	g.printlnf("if vv, ok := dict[\"%s\"]; ok {", f.name)
	g.printlnf("\tif v, ok := vv.(enc.Name); ok {")
	g.printlnf("\t\tvalue.%s = v", f.name)
	g.printlnf("\t} else {")
	g.printlnf("\t\terr = enc.ErrIncompatibleType{Name: \"%s\", TypeNum: %d, ValType: \"Name\", Value: vv}", f.name, f.typeNum)
	g.printlnf("\t}")
	g.printlnf("} else {")
	g.printlnf("\tvalue.%s = nil", f.name)
	g.printlnf("}")
	return g.output()
}

// Generates code to conditionally add a non-nil struct field to a dictionary map by calling the field's `ToDict()` method.
func (f *StructField) GenToDict() (string, error) {
	g := strErrBuf{}
	g.printlnf("if value.%s != nil {", f.name)
	g.printlnf("\tdict[\"%s\"] = value.%s.ToDict()", f.name, f.name)
	g.printlnf("}")
	return g.output()
}

// Generates Go code to safely extract and assign a struct field from a dictionary, performing type checks and error handling if the value is incompatible or missing.
func (f *StructField) GenFromDict() (string, error) {
	g := strErrBuf{}
	g.printlnf("if vv, ok := dict[\"%s\"]; ok {", f.name)
	g.printlnf("\tif v, ok := vv.(*%s); ok {", f.StructType)
	g.printlnf("\t\tvalue.%s = v", f.name)
	g.printlnf("\t} else {")
	g.printlnf("\t\terr = enc.ErrIncompatibleType{Name: \"%s\", TypeNum: %d, ValType: \"*%s\", Value: vv}",
		f.name, f.typeNum, f.StructType)
	g.printlnf("\t}")
	g.printlnf("} else {")
	g.printlnf("\tvalue.%s = nil", f.name)
	g.printlnf("}")
	return g.output()
}

// Generates Go code to convert a sequence field into a dictionary entry by iterating over each element, applying the subfield's `GenToDict` logic to populate a map, and aggregating the results into a slice stored under the field's name in the output dictionary.
func (f *SequenceField) GenToDict() (string, error) {
	// Sequence uses faked encoder variable to embed the subfield.
	// I have verified that the Go compiler can optimize this in simple cases.
	t := template.Must(template.New("SeqInitEncoder").Parse(`{
		{{.Name}}_l := len(value.{{.Name}})
		dictSeq = make([]{{.FieldType}}, {{.Name}}_l)
		for i := 0; i < {{.Name}}_l; i ++ {
			pseudoValue := struct {
				{{.Name}} {{.FieldType}}
			}{
				{{.Name}}: value.{{.Name}}[i],
			}
			pseudoMap = make(map[string]interface{})
			{
				dict := pseudoMap
				value := &pseudoValue
				{{.SubField.GenToDict}}
				_ = dict
				_ = value
			}
			dictSeq[i] = pseudoMap[{{.Name}}]
		}
		dict[\"{{.Name}}\"] = dictSeq
	}
	`))

	var g strErrBuf
	g.executeTemplate(t, f)
	return g.output()
}

// Generates a string representation of the SequenceField from a dictionary, currently returning an unimplemented error message.
func (f *SequenceField) GenFromDict() (string, error) {
	return "ERROR = \"Unimplemented yet!\"", nil
}

// Generates code to add a uint field to a dictionary, including a nil check if the field is optional.
func (f *FixedUintField) GenToDict() (string, error) {
	g := strErrBuf{}
	if f.opt {
		g.printlnf("if value.%s != nil {", f.name)
		g.printlnf("\tdict[\"%s\"] = *value.%s", f.name, f.name)
		g.printlnf("}")
	} else {
		g.printlnf("dict[\"%s\"] = value.%s", f.name, f.name)
	}
	return g.output()
}

// Generates code to parse a fixed-size unsigned integer field from a dictionary into a struct, handling type validation, optional/required constraints, and error reporting.
func (f *FixedUintField) GenFromDict() (string, error) {
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

	g := strErrBuf{}
	g.printlnf("if vv, ok := dict[\"%s\"]; ok {", f.name)
	g.printlnf("\tif v, ok := vv.(%s); ok {", digit)
	if f.opt {
		g.printlnf("\t\tvalue.%s = &v", f.name)
	} else {
		g.printlnf("\t\tvalue.%s = v", f.name)
	}
	g.printlnf("\t} else {")
	g.printlnf("\t\terr = enc.ErrIncompatibleType{Name: \"%s\", TypeNum: %d, ValType: \"%s\", Value: vv}",
		f.name, f.typeNum, digit)
	g.printlnf("\t}")
	g.printlnf("} else {")
	if f.opt {
		g.printlnf("\tvalue.%s = nil", f.name)
	} else {
		g.printlnf("err = enc.ErrSkipRequired{Name: \"%s\", TypeNum: %d}", f.name, f.typeNum)
	}
	g.printlnf("}")
	return g.output()
}
