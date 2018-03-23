// Copyright Kozyrev Yury
// MIT license.
package generator

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
	"strings"
)

//func main() {
//	s := &Document{}
//	s.Read(&domain.Order{})
//	fmt.Println(s)
//}

const DEFAULT_SCHEMA = "http://json-schema.org/schema#"

var rTypeInt64, rTypeFloat64 = reflect.TypeOf(int64(0)), reflect.TypeOf(float64(0))

type Document struct {
	Schema string `json:"$schema,omitempty"`
	property
}

func Generate(v interface{}) string {
	return new(Document).Read(v).String()
}

// Reads the variable structure into the JSON-Schema Document
func (d *Document) Read(variable interface{}) *Document {
	d.setDefaultSchema()

	value := reflect.ValueOf(variable)
	d.read(value.Type())
	return d
}

func (d *Document) setDefaultSchema() {
	if d.Schema == "" {
		d.Schema = DEFAULT_SCHEMA
	}
}

// String return the JSON encoding of the Document as a string
func (d *Document) String() string {
	json, _ := json.MarshalIndent(d, "", "    ")
	return string(json)
}

type property struct {
	Type                 []string             `json:"type,omitempty"`
	Format               string               `json:"format,omitempty"`
	Items                *property            `json:"items,omitempty"`
	Properties           map[string]*property `json:"properties,omitempty"`
	Required             []string             `json:"required,omitempty"`
	AdditionalProperties bool                 `json:"additionalProperties,omitempty"`
	Description          string               `json:"description,omitempty"`
	AnyOf                []*property          `json:"anyOf,omitempty"`

	// validation keywords:
	// For any number-valued fields, we're making them pointers, because
	// we want empty values to be omitted, but for numbers, 0 is seen as empty.

	// numbers validators
	MultipleOf       *float64 `json:"multipleOf,omitempty"`
	Maximum          *float64 `json:"maximum,omitempty"`
	Minimum          *float64 `json:"minimum,omitempty"`
	ExclusiveMaximum *float64 `json:"exclusiveMaximum,omitempty"`
	ExclusiveMinimum *float64 `json:"exclusiveMinimum,omitempty"`
	// string validators
	MaxLength *int64 `json:"maxLength,omitempty"`
	MinLength *int64 `json:"minLength,omitempty"`
	Pattern   string `json:"pattern,omitempty"`
	// Enum is defined for arbitrary types, but I'm currently just implementing it for strings.
	Enum []string `json:"enum,omitempty"`

	// Implemented for strings and numbers
	Const interface{} `json:"const,omitempty"`
}

func (p *property) read(t reflect.Type) {
	jsType, format, kind := getTypeFromMapping(t)
	if jsType != "" {
		p.Type = append(p.Type, jsType)
	}
	if format != "" {
		p.Format = format
	}
	p.Type = append(p.Type, "null")

	switch kind {
	case reflect.Slice:
		p.readFromSlice(t)
	case reflect.Map:
		p.readFromMap(t)
	case reflect.Struct:
		p.readFromStruct(t)
	case reflect.Ptr:
		p.read(t.Elem())
	}

	// say we have *int
	if kind == reflect.Ptr && isPrimitive(t.Elem().Kind()) {
		p.AnyOf = []*property{
			{Type: p.Type},
			{Type: []string{"null"}},
		}
		p.Type = []string{"null"}
	}
}

func (p *property) readFromSlice(t reflect.Type) {
	jsType, _, kind := getTypeFromMapping(t.Elem())
	if kind == reflect.Uint8 {
		p.Type = []string{"string"}
	} else if jsType != "" || kind == reflect.Ptr {
		p.Items = &property{}
		p.Items.read(t.Elem())
	}
}

func (p *property) readFromMap(t reflect.Type) {
	jsType, format, _ := getTypeFromMapping(t.Elem())

	if jsType != "" {
		p.Properties = make(map[string]*property, 0)
		p.Properties[".*"] = &property{Type: []string{jsType}, Format: format}
	} else {
		p.AdditionalProperties = true
	}
}

func (p *property) readFromStruct(t reflect.Type) {
	p.Type = []string{"object"}
	p.Properties = make(map[string]*property, 0)
	p.AdditionalProperties = false

	count := t.NumField()
	for i := 0; i < count; i++ {
		field := t.Field(i)

		tag := field.Tag.Get("json")
		_, required := field.Tag.Lookup("required")
		name, opts := parseTag(tag)
		if name == "" {
			name = field.Name
		}
		if name == "-" {
			continue
		}

		p.Properties[name] = &property{}
		p.Properties[name].read(field.Type)
		p.Properties[name].Description = field.Tag.Get("description")
		p.Properties[name].addValidatorsFromTags(&field.Tag)

		if opts.Contains("omitempty") || !required {
			continue
		}
		p.Required = append(p.Required, name)
	}
}

func (p *property) addValidatorsFromTags(tag *reflect.StructTag) {
	for _, item := range p.Type {
		switch item {
		case "string":
			p.addStringValidators(tag)
		case "number", "integer":
			p.addNumberValidators(tag)
		}
	}
}

// Some helper functions for not having to create temp variables all over the place
func int64ptr(i interface{}) *int64 {
	v := reflect.ValueOf(i)
	if !v.Type().ConvertibleTo(rTypeInt64) {
		return nil
	}
	j := v.Convert(rTypeInt64).Interface().(int64)
	return &j
}

func float64ptr(i interface{}) *float64 {
	v := reflect.ValueOf(i)
	if !v.Type().ConvertibleTo(rTypeFloat64) {
		return nil
	}
	j := v.Convert(rTypeFloat64).Interface().(float64)
	return &j
}

func (p *property) addStringValidators(tag *reflect.StructTag) {
	// min length
	mls := tag.Get("minLength")
	ml, err := strconv.ParseInt(mls, 10, 64)
	if err == nil {
		p.MinLength = int64ptr(ml)
	}
	// max length
	mls = tag.Get("maxLength")
	ml, err = strconv.ParseInt(mls, 10, 64)
	if err == nil {
		p.MaxLength = int64ptr(ml)
	}
	// pattern
	pat := tag.Get("pattern")
	if pat != "" {
		p.Pattern = pat
	}
	// enum
	en := tag.Get("enum")
	if en != "" {
		p.Enum = strings.Split(en, "|")
	}
	//const
	c := tag.Get("const")
	if c != "" {
		p.Const = c
	}
}

func (p *property) addNumberValidators(tag *reflect.StructTag) {
	m, err := strconv.ParseFloat(tag.Get("multipleOf"), 64)
	if err == nil {
		p.MultipleOf = float64ptr(m)
	}
	m, err = strconv.ParseFloat(tag.Get("min"), 64)
	if err == nil {
		p.Minimum = float64ptr(m)
	}
	m, err = strconv.ParseFloat(tag.Get("max"), 64)
	if err == nil {
		p.Maximum = float64ptr(m)
	}
	m, err = strconv.ParseFloat(tag.Get("exclusiveMin"), 64)
	if err == nil {
		p.ExclusiveMinimum = float64ptr(m)
	}
	m, err = strconv.ParseFloat(tag.Get("exclusiveMax"), 64)
	if err == nil {
		p.ExclusiveMaximum = float64ptr(m)
	}
	for _, item := range p.Type {
		c, err := parseType(tag.Get("const"), item)
		if err == nil {
			p.Const = c
		}
	}

}

func parseType(str, ty string) (interface{}, error) {
	var v interface{}
	var err error
	if ty == "number" {
		v, err = strconv.ParseFloat(str, 64)
	} else {
		v, err = strconv.ParseInt(str, 10, 64)
	}
	return v, err
}

var formatMapping = map[string][]string{
	"time.Time": []string{"string", "date-time"},
}

var kindMapping = map[reflect.Kind]string{
	reflect.Bool:    "boolean",
	reflect.Int:     "integer",
	reflect.Int8:    "integer",
	reflect.Int16:   "integer",
	reflect.Int32:   "integer",
	reflect.Int64:   "integer",
	reflect.Uint:    "integer",
	reflect.Uint8:   "integer",
	reflect.Uint16:  "integer",
	reflect.Uint32:  "integer",
	reflect.Uint64:  "integer",
	reflect.Float32: "number",
	reflect.Float64: "number",
	reflect.String:  "string",
	reflect.Slice:   "array",
	reflect.Struct:  "object",
	reflect.Map:     "object",
}

func isPrimitive(k reflect.Kind) bool {
	if v, ok := kindMapping[k]; ok {
		switch v {
		case "boolean":
		case "integer":
		case "number":
		case "string":
			return true
		}
	}
	return false
}

func getTypeFromMapping(t reflect.Type) (string, string, reflect.Kind) {
	if v, ok := formatMapping[t.String()]; ok {
		return v[0], v[1], reflect.String
	}

	if v, ok := kindMapping[t.Kind()]; ok {
		return v, "", t.Kind()
	}

	return "", "", t.Kind()
}

type structTag string

func parseTag(tag string) (string, structTag) {
	if idx := strings.Index(tag, ","); idx != -1 {
		return tag[:idx], structTag(tag[idx+1:])
	}
	return tag, structTag("")
}

func (o structTag) Contains(optionName string) bool {
	if len(o) == 0 {
		return false
	}

	s := string(o)
	for s != "" {
		var next string
		i := strings.Index(s, ",")
		if i >= 0 {
			s, next = s[:i], s[i+1:]
		}
		if s == optionName {
			return true
		}
		s = next
	}
	return false
}

var _ fmt.Stringer = (*Document)(nil)
