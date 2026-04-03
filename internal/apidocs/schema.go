package apidocs

import (
	"reflect"
	"slices"
	"strings"
	"time"

	"github.com/p-n-ai/pai-bot/internal/adminapi"
	"github.com/p-n-ai/pai-bot/internal/auth"
)

type schemaRegistry struct {
	schemas map[string]*Schema
	seen    map[reflect.Type]bool
}

func newSchemaRegistry() *schemaRegistry {
	return &schemaRegistry{
		schemas: map[string]*Schema{},
		seen:    map[reflect.Type]bool{},
	}
}

func (r *schemaRegistry) refFor(value any) *Schema {
	t := indirectType(reflect.TypeOf(value))
	name := schemaName(t)
	r.ensure(t)
	return &Schema{Ref: "#/components/schemas/" + name}
}

func (r *schemaRegistry) ensure(t reflect.Type) {
	t = indirectType(t)
	if t == nil || isInlineSchemaType(t) || r.seen[t] {
		return
	}
	r.seen[t] = true
	r.schemas[schemaName(t)] = r.schemaForType(t)
}

func (r *schemaRegistry) schemaForField(t reflect.Type) *Schema {
	t = indirectType(t)
	if t == nil {
		return &Schema{Type: "string"}
	}
	if isInlineSchemaType(t) {
		return r.schemaForType(t)
	}
	return r.refFor(reflect.New(t).Elem().Interface())
}

func (r *schemaRegistry) schemaForType(t reflect.Type) *Schema {
	t = indirectType(t)
	if t == nil {
		return &Schema{Type: "string"}
	}

	if enum := enumValues(t); len(enum) > 0 {
		return &Schema{Type: "string", Enum: enum}
	}

	if t == reflect.TypeOf(time.Time{}) {
		return &Schema{Type: "string", Format: "date-time"}
	}

	switch t.Kind() {
	case reflect.Bool:
		return &Schema{Type: "boolean"}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return &Schema{Type: "integer"}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return &Schema{Type: "integer"}
	case reflect.Float32, reflect.Float64:
		return &Schema{Type: "number"}
	case reflect.String:
		return &Schema{Type: "string"}
	case reflect.Slice, reflect.Array:
		return &Schema{
			Type:  "array",
			Items: r.schemaForField(t.Elem()),
		}
	case reflect.Map:
		schema := &Schema{Type: "object"}
		if t.Key().Kind() == reflect.String {
			schema.AdditionalProperties = r.schemaForField(t.Elem())
		} else {
			schema.AdditionalProperties = true
		}
		return schema
	case reflect.Interface:
		return &Schema{Type: "object", AdditionalProperties: true}
	case reflect.Struct:
		return r.objectSchema(t)
	default:
		return &Schema{Type: "string"}
	}
}

func (r *schemaRegistry) objectSchema(t reflect.Type) *Schema {
	schema := &Schema{
		Type:                 "object",
		Properties:           map[string]*Schema{},
		AdditionalProperties: false,
	}

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		if !field.IsExported() {
			continue
		}

		name, omitEmpty, ok := jsonFieldName(field)
		if !ok {
			continue
		}

		schema.Properties[name] = r.schemaForField(field.Type)
		applyFieldMetadata(t, field.Name, schema.Properties[name])
		if omitEmpty || optionalFieldOverride(t, field.Name) || isOptionalField(field.Type) {
			continue
		}
		schema.Required = append(schema.Required, name)
	}

	slices.Sort(schema.Required)
	if len(schema.Required) == 0 {
		schema.Required = nil
	}
	return schema
}

func indirectType(t reflect.Type) reflect.Type {
	for t != nil && t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	return t
}

func isInlineSchemaType(t reflect.Type) bool {
	t = indirectType(t)
	if t == nil {
		return true
	}
	if t == reflect.TypeOf(time.Time{}) {
		return true
	}
	if enum := enumValues(t); len(enum) > 0 {
		return true
	}
	switch t.Kind() {
	case reflect.Bool, reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64, reflect.String, reflect.Interface:
		return true
	case reflect.Slice, reflect.Array, reflect.Map:
		return true
	default:
		return false
	}
}

func schemaName(t reflect.Type) string {
	t = indirectType(t)
	if t.Name() != "" {
		return t.Name()
	}
	return strings.ReplaceAll(t.String(), ".", "")
}

func jsonFieldName(field reflect.StructField) (name string, omitEmpty bool, ok bool) {
	tag := field.Tag.Get("json")
	if tag == "-" {
		return "", false, false
	}
	if tag == "" {
		return field.Name, false, true
	}
	parts := strings.Split(tag, ",")
	if parts[0] == "" {
		return field.Name, slices.Contains(parts[1:], "omitempty"), true
	}
	return parts[0], slices.Contains(parts[1:], "omitempty"), true
}

func isOptionalField(t reflect.Type) bool {
	if t == nil {
		return true
	}
	switch t.Kind() {
	case reflect.Pointer, reflect.Interface:
		return true
	default:
		return false
	}
}

func enumValues(t reflect.Type) []any {
	switch t {
	case reflect.TypeOf(auth.Role("")):
		return []any{
			string(auth.RoleStudent),
			string(auth.RoleTeacher),
			string(auth.RoleParent),
			string(auth.RoleAdmin),
			string(auth.RolePlatformAdmin),
		}
	}
	return nil
}

func applyFieldMetadata(parent reflect.Type, fieldName string, schema *Schema) {
	if schema == nil {
		return
	}
	if fieldName == "Email" && schema.Ref == "" && schema.Type == "string" {
		schema.Format = "email"
	}
	if description := fieldDescription(parent, fieldName); description != "" {
		schema.Description = description
	}
}

func optionalFieldOverride(parent reflect.Type, fieldName string) bool {
	switch parent {
	case reflect.TypeOf(auth.LoginRequest{}):
		return fieldName == "TenantID"
	}
	return false
}

func fieldDescription(parent reflect.Type, fieldName string) string {
	switch parent {
	case reflect.TypeOf(adminapi.AIDailyUsagePoint{}):
		if fieldName == "Date" {
			return "Calendar date in YYYY-MM-DD format."
		}
	case reflect.TypeOf(adminapi.UpsertTokenBudgetWindowRequest{}):
		switch fieldName {
		case "PeriodStart", "PeriodEnd":
			return "UTC timestamp in RFC3339 format."
		}
	}
	return ""
}
