package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
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

func generateStruct(schema *Schema, root *Schema) jen.Code {
	var names []string
	for name := range schema.Properties {
		names = append(names, name)
	}
	sort.Strings(names)

	var fields []jen.Code
	for _, name := range names {
		prop := schema.Properties[name]
		id := formatId(name)
		required := isRequired(schema, name)
		t := generateSchemaType(&prop, root, required)
		jsonTag := name
		if !required {
			jsonTag += ",omitempty"
		}
		tags := map[string]string{"json": jsonTag}
		fields = append(fields, jen.Id(id).Add(t).Tag(tags))
	}
	return jen.Struct(fields...)
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

func generateSchemaType(schema *Schema, root *Schema, required bool) jen.Code {
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
		return jen.Op("*").Add(generateSchemaType(subschema, root, true))
	}

	switch schemaType(schema) {
	case TypeNull:
		return jen.Struct()
	case TypeBoolean:
		return jen.Bool()
	case TypeArray:
		return jen.Index().Add(generateSchemaType(schema.Items, root, required))
	case TypeNumber:
		return jen.Qual("encoding/json", "Number")
	case TypeString:
		return jen.String()
	case TypeInteger:
		return jen.Int64()
	case TypeObject:
		noAdditionalProps := noAdditionalProps(schema)
		if noAdditionalProps && len(schema.PatternProperties) == 0 {
			t := generateStruct(schema, root)
			if !required {
				t = jen.Op("*").Add(t)
			}
			return t
		} else if patternProp := singlePatternProp(schema); noAdditionalProps && patternProp != nil {
			return jen.Map(jen.String()).Add(generateSchemaType(patternProp, root, true))
		} else {
			return jen.Map(jen.String()).Add(generateSchemaType(schema.AdditionalProperties, root, true))
		}
	default:
		return jen.Qual("encoding/json", "RawMessage")
	}
}

func generateDef(schema *Schema, root *Schema, f *jen.File, name string) {
	id := formatId(name)

	if schema.Ref == "" && schemaType(schema) == "" {
		f.Type().Id(id).Struct(
			jen.Qual("encoding/json", "RawMessage"),
		).Line()

		var children []Schema
		for _, child := range schema.AllOf {
			children = append(children, child)
		}
		for _, child := range schema.AnyOf {
			children = append(children, child)
		}
		for _, child := range schema.OneOf {
			children = append(children, child)
		}
		if schema.Then != nil {
			children = append(children, *schema.Then)
		}
		if schema.Else != nil {
			children = append(children, *schema.Else)
		}
		for _, child := range schema.DependentSchemas {
			children = append(children, child)
		}

		for _, child := range children {
			refName := refName(child.Ref)
			if refName == "" {
				continue
			}

			t := generateSchemaType(&child, root, false)

			f.Func().Params(
				jen.Id("v").Id(id),
			).Id(formatId(refName)).Params().Params(
				t,
				jen.Id("error"),
			).Block(
				jen.Var().Id("out").Add(t),
				jen.Id("err").Op(":=").Qual("encoding/json", "Unmarshal").Params(
					jen.Id("v").Op(".").Id("RawMessage"),
					jen.Op("&").Id("out"),
				),
				jen.Return(
					jen.Id("out"),
					jen.Id("err"),
				),
			).Line()
		}
	} else {
		f.Type().Id(id).Add(generateSchemaType(schema, root, true)).Line()
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
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, usage)
	}
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

	if schema.Ref == "" {
		generateDef(schema, schema, f, "root")
	}

	var names []string
	for name := range schema.Defs {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		def := schema.Defs[name]
		generateDef(&def, schema, f, name)
	}

	if err := f.Save(outputFilename); err != nil {
		log.Fatalf("failed to save output file: %v", err)
	}
}
