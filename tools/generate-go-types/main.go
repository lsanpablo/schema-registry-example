package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode"

	"github.com/dave/jennifer/jen"
)

type Type string

const (
	TypeNull    Type = "null"
	TypeBoolean Type = "boolean"
	TypeObject  Type = "object"
	TypeArray   Type = "array"
	TypeNumber  Type = "number"
	TypeString  Type = "string"
	TypeInteger Type = "integer"
)

type TypeSet []Type

func (ts *TypeSet) UnmarshalJSON(b []byte) error {
	if b[0] == '[' {
		type rawTypeSet TypeSet
		out := (*rawTypeSet)(ts)
		return json.Unmarshal(b, out)
	} else {
		var t Type
		err := json.Unmarshal(b, &t)
		if err != nil {
			*ts = nil
		} else {
			*ts = []Type{t}
		}
		return err
	}
}

type Schema struct {
	// Core
	Schema     string            `json:"$schema"`
	Vocabulary map[string]bool   `json:"$vocabulary"`
	ID         string            `json:"$id"`
	Ref        string            `json:"$ref"`
	DynamicRef string            `json:"$dynamicRef"`
	Defs       map[string]Schema `json:"$defs"`
	Comment    string            `json:"$comment"`

	// Applying subschemas with logic
	AllOf []Schema `json:"allOf"`
	AnyOf []Schema `json:"anyOf"`
	OneOf []Schema `json:"oneOf"`
	Not   []Schema `json:"not"`

	// Applying subschemas conditionally
	If               *Schema           `json:"if"`
	Then             *Schema           `json:"then"`
	Else             *Schema           `json:"else"`
	DependentSchemas map[string]Schema `json:"dependentSchemas"`

	// Applying subschemas to arrays
	PrefixItems []Schema `json:"prefixItems"`
	Items       *Schema  `json:"items"`
	Contains    *Schema  `json:"contains"`

	// Applying subschemas to objects
	Properties           map[string]Schema `json:"properties"`
	PatternProperties    map[string]Schema `json:"patternProperties"`
	AdditionalProperties *Schema           `json:"additionalProperties"`
	PropertyNames        *Schema           `json:"propertyNames"`

	// Validation
	Type  TypeSet       `json:"type"`
	Enum  []interface{} `json:"enum"`
	Const interface{}   `json:"const"`

	// Validation for numbers
	MultipleOf       json.Number `json:"multipleOf"`
	Maximum          json.Number `json:"maximum"`
	ExclusiveMaximum json.Number `json:"exclusiveMaximum"`
	Minimum          json.Number `json:"minimum"`
	ExclusiveMinimum json.Number `json:"exclusiveMinimum"`

	// Validation for strings
	MaxLength int    `json:"maxLength"`
	MinLength int    `json:"minLength"`
	Pattern   string `json:"pattern"`

	// Validation for arrays
	MaxItems    int  `json:"maxItems"`
	MinItems    int  `json:"minItems"`
	UniqueItems bool `json:"uniqueItems"`
	MaxContains int  `json:"maxContains"`
	MinContains int  `json:"minContains"`

	// Validation for objects
	MaxProperties     int                 `json:"maxProperties"`
	MinProperties     int                 `json:"minProperties"`
	Required          []string            `json:"required"`
	DependentRequired map[string][]string `json:"dependentRequired"`

	// Basic metadata annotations
	Title       string        `json:"title"`
	Description string        `json:"description"`
	Default     interface{}   `json:"default"`
	Deprecated  bool          `json:"deprecated"`
	ReadOnly    bool          `json:"readOnly"`
	WriteOnly   bool          `json:"writeOnly"`
	Examples    []interface{} `json:"examples"`
}

func (schema *Schema) UnmarshalJSON(b []byte) error {
	if bytes.Equal(b, []byte("true")) {
		*schema = Schema{}
	} else if bytes.Equal(b, []byte("false")) {
		*schema = Schema{Not: []Schema{
			Schema{},
		}}
	} else {
		type rawSchema Schema
		var out rawSchema
		if err := json.Unmarshal(b, &out); err != nil {
			return err
		}
		*schema = Schema(out)
	}
	return nil
}

func (schema *Schema) IsTrue() bool {
	return len(schema.AllOf) == 0 && len(schema.AnyOf) == 0 && len(schema.OneOf) == 0 && len(schema.Not) == 0 && schema.If == nil && schema.Then == nil && schema.Else == nil && len(schema.DependentSchemas) == 0 && len(schema.PrefixItems) == 0 && schema.Items == nil && schema.Contains == nil && len(schema.Properties) == 0 && len(schema.PatternProperties) == 0 && schema.AdditionalProperties == nil && schema.PropertyNames == nil
}

func (schema *Schema) IsFalse() bool {
	for _, not := range schema.Not {
		if not.IsTrue() {
			return true
		}
	}
	return false
}

var nestedTypes = make(map[string]*Schema)

func formatId(s string) string {
	fields := strings.FieldsFunc(s, func(c rune) bool {
		return !unicode.IsLetter(c) && !unicode.IsNumber(c)
	})
	for i, v := range fields {
		fields[i] = strings.Title(v)
	}
	return strings.Join(fields, "")
}

func refName(ref string) string {
	prefix := "#/$defs/"
	if !strings.HasPrefix(ref, prefix) {
		return ""
	}
	return strings.TrimPrefix(ref, prefix)
}

func resolveRef(def *Schema, root *Schema) *Schema {
	if def.Ref == "" {
		return def
	}

	name := refName(def.Ref)
	if name == "" {
		log.Fatalf("unsupported $ref %q", def.Ref)
	}

	result, ok := root.Defs[name]
	if !ok {
		log.Fatalf("invalid $ref %q", def.Ref)
	}
	return &result
}

func schemaType(schema *Schema) Type {
	switch {
	case len(schema.Type) == 1:
		return schema.Type[0]
	case len(schema.Type) > 0:
		return ""
	}

	var v interface{}
	if schema.Const != nil {
		v = schema.Const
	} else if len(schema.Enum) > 0 {
		v = schema.Enum[0]
	}

	switch v.(type) {
	case bool:
		return TypeBoolean
	case map[string]interface{}:
		return TypeObject
	case []interface{}:
		return TypeArray
	case float64:
		return TypeNumber
	case string:
		return TypeString
	default:
		return ""
	}
}

func isRequired(schema *Schema, propName string) bool {
	for _, name := range schema.Required {
		if name == propName {
			return true
		}
	}
	return false
}

func generateStruct(name string, schema *Schema, root *Schema, f *jen.File) jen.Code {
	// record nested type
	nestedTypes[name] = schema

	// build fields
	var propNames []string
	for p := range schema.Properties {
		propNames = append(propNames, p)
	}
	sort.Strings(propNames)
	var fields []jen.Code
	for _, propName := range propNames {
		prop := schema.Properties[propName]
		id := formatId(propName)
		required := isRequired(schema, propName)

		// determine Go type
		t := generateSchemaType(propName, &prop, root, required, f)

		// json tag
		jsonTag := propName
		if !required {
			jsonTag += ",omitempty"
		}
		// zog tag: lower case field name without underscores
		zogTag := strings.ToLower(strings.ReplaceAll(propName, "_", ""))
		tags := map[string]string{"json": jsonTag, "zog": zogTag}
		fields = append(fields, jen.Id(id).Add(t).Tag(tags))
	}

	// emit type
	f.Type().Id(name).Struct(fields...).Line()
	return jen.Id(name)
}

func singlePatternProp(schema *Schema) *Schema {
	if len(schema.PatternProperties) != 1 {
		return nil
	}
	for _, prop := range schema.PatternProperties {
		return &prop
	}
	return nil
}

func noAdditionalProps(schema *Schema) bool {
	return schema.AdditionalProperties != nil && schema.AdditionalProperties.IsFalse()
}

// unwrapNullableSchema unwraps a schema in the form:
//
//	{
//		"oneOf": {
//			{ "type": "null" },
//			<sub-schema>
//		}
//	}
func unwrapNullableSchema(schema *Schema) (*Schema, bool) {
	for _, choices := range [][]Schema{schema.AnyOf, schema.OneOf} {
		if len(choices) != 2 {
			continue
		}

		nullIndex := -1
		for i, choice := range choices {
			if len(choice.Type) == 1 && choice.Type[0] == TypeNull {
				nullIndex = i
				break
			}
		}
		if nullIndex < 0 {
			continue
		}

		otherIndex := (nullIndex + 1) % 2
		return &choices[otherIndex], true
	}
	return nil, false
}

func generateSchemaType(propName string, schema *Schema, root *Schema, required bool, f *jen.File) jen.Code {
	if schema == nil {
		schema = &Schema{}
	}

	refName := refName(schema.Ref)
	if refName != "" {
		schema = resolveRef(schema, root)
		t := jen.Id(formatId(refName))
		if !required && schemaType(schema) == TypeObject && noAdditionalProps(schema) && len(schema.PatternProperties) == 0 {
			t = jen.Op("*").Add(t)
		}
		return t
	}

	if subschema, ok := unwrapNullableSchema(schema); ok {
		return jen.Op("*").Add(generateSchemaType(propName, subschema, root, true, f))
	}

	switch schemaType(schema) {
	case TypeNull:
		return jen.Struct()
	case TypeBoolean:
		return jen.Bool()
	case TypeNumber:
		return jen.Qual("encoding/json", "Number")
	case TypeString:
		return jen.String()
	case TypeInteger:
		return jen.Int64()
	case TypeObject:
		// name nested struct as Parent + Field
		nestedName := "" // find last type defined in f to get context
		// For simplicity, derive a unique nested name
		nestedName = formatId(propName)
		// generate that nested struct
		generateStruct(nestedName, schema, root, f)
		typeCode := jen.Id(nestedName)
		if !required {
			typeCode = jen.Op("*").Add(typeCode)
		}
		return typeCode
	case TypeArray:
		return jen.Index().Add(generateSchemaType(propName, schema.Items, root, true, f))
	default:
		return jen.Qual("encoding/json", "RawMessage")
	}
}

func generateDef(schema *Schema, root *Schema, f *jen.File, name string) {
	id := formatId(name)
	// for root and defs, use named struct generator
	if schemaType(schema) == TypeObject {
		generateStruct(id, schema, root, f)
	} else {
		// unchanged: alias simple types
		f.Type().Id(id).Add(generateSchemaType(name, schema, root, true, f)).Line()
	}
}

func loadSchema(filename string) *Schema {
	f, err := os.Open(filename)
	if err != nil {
		log.Fatalf("failed to open schema file: %v", err)
	}
	defer f.Close()

	var schema Schema
	if err := json.NewDecoder(f).Decode(&schema); err != nil {
		log.Fatalf("failed to load schema JSON: %v", err)
	}

	return &schema
}

const usage = `usage: jsonschemagen -s <schema> -o <output> [options...]

Generate Go types and helpers for the specified JSON schema.

Options:

  -s <schema>    JSON schema filename. Required.
  -o <output>    Output filename for generated Go code. Required.
  -n <package>   Go package name, defaults to the dirname of the output file.
`

func main() {
	var schemaFilename, outputFilename, pkgName string
	flag.StringVar(&schemaFilename, "s", "", "schema filename")
	flag.StringVar(&outputFilename, "o", "", "output filename")
	flag.StringVar(&pkgName, "n", "", "package name")
	flag.Parse()

	if schemaFilename == "" || outputFilename == "" || len(flag.Args()) > 0 {
		flag.Usage()
		os.Exit(1)
	}

	if pkgName == "" {
		abs, err := filepath.Abs(outputFilename)
		if err != nil {
			log.Fatalf("failed to get absolute output filename: %v", err)
		}
		pkgName = filepath.Base(filepath.Dir(abs))
	}

	schema := loadSchema(schemaFilename)
	f := jen.NewFile(pkgName)

	// generate root and definitions
	if schema.Ref == "" {
		generateDef(schema, schema, f, "root")
	}
	var defNames []string
	for d := range schema.Defs {
		defNames = append(defNames, d)
	}
	sort.Strings(defNames)
	for _, d := range defNames {
		elem := schema.Defs[d]
		generateDef(&elem, schema, f, d)
	}

	// save file
	if err := f.Save(outputFilename); err != nil {
		log.Fatalf("failed to save output file: %v", err)
	}
}
