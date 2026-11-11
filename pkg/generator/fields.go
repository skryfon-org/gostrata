package generator

import (
	"fmt"
	"go/token"
	"strings"

	"github.com/skryfon-org/gostrata/pkg/strutil"
)

// Field describes one entity column declared on the command line (e.g. "price:float").
type Field struct {
	Name    string // raw name as typed
	Pascal  string // Go field name
	Snake   string // json key and SQL column
	Type    string // gostrata type keyword
	GoType  string
	PtrType string
	SQLType string
	Binding string // gin binding tag for create requests
}

// GormTag returns the gorm struct tag value for the field.
func (f Field) GormTag() string {
	if f.Type == "string" {
		return "size:255;not null"
	}
	return "not null"
}

type fieldType struct{ goType, binding string }

var fieldTypes = map[string]fieldType{
	"string": {"string", "required"},
	"text":   {"string", "required"},
	"int":    {"int", ""},
	"int64":  {"int64", ""},
	"float":  {"float64", ""},
	"bool":   {"bool", ""},
	"time":   {"time.Time", ""},
	"uuid":   {"uuid.UUID", ""},
}

// SupportedFieldTypes lists the type keywords accepted in field definitions.
func SupportedFieldTypes() []string {
	return []string{"string", "text", "int", "int64", "float", "bool", "time", "uuid"}
}

var reservedColumns = map[string]bool{"id": true, "created_at": true, "updated_at": true, "deleted_at": true}

// ParseFields parses definitions of the form name:type (type defaults to string).
func ParseFields(defs []string) ([]Field, error) {
	var fields []Field
	seen := map[string]bool{}
	for _, def := range defs {
		name, typ, _ := strings.Cut(def, ":")
		typ = strings.ToLower(strings.TrimSpace(typ))
		if typ == "" {
			typ = "string"
		}
		if !strutil.IsValidIdentifier(name) {
			return nil, fmt.Errorf("invalid field name %q in %q", name, def)
		}
		ft, ok := fieldTypes[typ]
		if !ok {
			return nil, fmt.Errorf("unknown field type %q in %q (supported: %s)", typ, def, strings.Join(SupportedFieldTypes(), ", "))
		}
		snake := strutil.ToSnakeCase(name)
		if reservedColumns[snake] {
			return nil, fmt.Errorf("field %q is reserved (generated automatically)", name)
		}
		if seen[snake] {
			return nil, fmt.Errorf("duplicate field %q", name)
		}
		seen[snake] = true
		fields = append(fields, Field{
			Name:    name,
			Pascal:  strutil.ToPascalCase(name),
			Snake:   snake,
			Type:    typ,
			GoType:  ft.goType,
			PtrType: "*" + ft.goType,
			Binding: ft.binding,
		})
	}
	return fields, nil
}

func isGoKeyword(s string) bool { return token.IsKeyword(s) }

// sqlTypeFor maps a gostrata field type to a column type for the given database engine.
func sqlTypeFor(db, typ string) string {
	switch typ {
	case "string":
		return "VARCHAR(255)"
	case "text":
		return "TEXT"
	case "int":
		return "INTEGER"
	case "int64":
		return "BIGINT"
	case "bool":
		return "BOOLEAN"
	case "float":
		switch db {
		case DBMySQL:
			return "DOUBLE"
		case DBSQLite:
			return "REAL"
		}
		return "DOUBLE PRECISION"
	case "time":
		switch db {
		case DBMySQL:
			return "DATETIME(6)"
		case DBSQLite:
			return "DATETIME"
		}
		return "TIMESTAMP WITH TIME ZONE"
	case "uuid":
		switch db {
		case DBMySQL:
			return "CHAR(36)"
		case DBSQLite:
			return "TEXT"
		}
		return "UUID"
	}
	return "TEXT"
}
