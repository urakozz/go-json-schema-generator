// Copyright Kozyrev Yury
// MIT license.
package generator

import (
	"fmt"
	"strings"
	"reflect"
	"encoding/json"
)

//func main() {
//	s := &Document{}
//	s.Read(&domain.Order{})
//	fmt.Println(s)
//}

const DEFAULT_SCHEMA = "http://json-schema.org/schema#"

type Document struct {
	Schema string `json:"$schema,omitempty"`
	property
}

// Reads the variable structure into the JSON-Schema Document
func (d *Document) Read(variable interface{}) {
	d.setDefaultSchema()

	value := reflect.ValueOf(variable)
	d.read(value.Type())
}

func (d *Document) setDefaultSchema() {
	if d.Schema == "" {
		d.Schema = DEFAULT_SCHEMA
	}
}

// Marshal returns the JSON encoding of the Document
func (d *Document) Marshal() ([]byte, error) {
	return json.MarshalIndent(d, "", "    ")
}

// String return the JSON encoding of the Document as a string
func (d *Document) String() string {
	json, _ := d.Marshal()
	return string(json)
}

type property struct {
	Type                 string               `json:"type,omitempty"`
	Format               string               `json:"format,omitempty"`
	Items                *property            `json:"items,omitempty"`
	Properties           map[string]*property `json:"properties,omitempty"`
	Required             []string             `json:"required,omitempty"`
	AdditionalProperties bool                 `json:"additionalProperties,omitempty"`
	Description          string               `json:"description,omitempty"`
	AnyOf                []*property          `json:"anyOf,omitempty"`
}
func (p *property) read(t reflect.Type) {
	jsType, format, kind := getTypeFromMapping(t)
	if jsType != "" {
		p.Type = jsType
	}
	if format != "" {
		p.Format = format
	}

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
			{Type:p.Type},
			{Type:"null"},
		}
		p.Type = ""
	}
}

func (p *property) readFromSlice(t reflect.Type) {
	jsType, _, kind := getTypeFromMapping(t.Elem())
	if kind == reflect.Uint8 {
		p.Type = "string"
	} else if jsType != "" || kind == reflect.Ptr {
		p.Items = &property{}
		p.Items.read(t.Elem())
	}
}

func (p *property) readFromMap(t reflect.Type) {
	jsType, format, _ := getTypeFromMapping(t.Elem())

	if jsType != "" {
		p.Properties = make(map[string]*property, 0)
		p.Properties[".*"] = &property{Type: jsType, Format: format}
	} else {
		p.AdditionalProperties = true
	}
}

func (p *property) readFromStruct(t reflect.Type) {
	p.Type = "object"
	p.Properties = make(map[string]*property, 0)
	p.AdditionalProperties = false

	count := t.NumField()
	for i := 0; i < count; i++ {
		field := t.Field(i)

		tag := field.Tag.Get("json")
		schemaTag := structTag(field.Tag.Get("schema"))
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

		if opts.Contains("omitempty") || !schemaTag.Contains("required") {
			continue
		}
		p.Required = append(p.Required, name)
		//if !opts.Contains("omitempty") {
		//	p.Required = append(p.Required, name)
		//}
	}
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
		return tag[:idx], structTag(tag[idx + 1:])
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
			s, next = s[:i], s[i + 1:]
		}
		if s == optionName {
			return true
		}
		s = next
	}
	return false
}
