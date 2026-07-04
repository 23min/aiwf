package config

import (
	"reflect"
	"strings"
)

// SchemaField describes one yaml-tagged field in the aiwf.yaml schema,
// contributed by walking the Config struct tree via reflection (E-0057).
type SchemaField struct {
	// Path is the dotted key path as it appears (or would appear) in
	// aiwf.yaml, e.g. "tdd.strict". A slice-of-struct field's elements use a
	// "[]" path segment (e.g. "areas.members[].name"); a map-of-struct
	// field's elements use "<key>" as a placeholder for the dynamic map key
	// (e.g. "agents.<key>.model").
	Path string
	// Type is the field's Go type rendered for display (e.g. "bool",
	// "[]string", "config.TDD" for a nested block).
	Type string
}

// Schema walks the Config struct tree and returns one SchemaField per
// yaml-tagged field, in struct-declaration order (depth-first). A
// struct-typed, slice-of-struct, or map-of-struct field contributes an entry
// for itself (the block) and then recurses into its element type. Fields
// whose Go name starts with "Legacy" are excluded: they are decode-only
// migration shims (see the package doc), never a documented,
// hand-authorable key.
func Schema() []SchemaField {
	var fields []SchemaField
	walkSchema(reflect.TypeFor[Config](), "", &fields)
	return fields
}

func walkSchema(t reflect.Type, prefix string, out *[]SchemaField) {
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		if strings.HasPrefix(f.Name, "Legacy") {
			continue
		}
		key, ok := yamlKey(f.Tag.Get("yaml"))
		if !ok {
			continue
		}
		path := key
		if prefix != "" {
			path = prefix + "." + key
		}
		*out = append(*out, SchemaField{Path: path, Type: f.Type.String()})

		switch f.Type.Kind() {
		case reflect.Struct:
			walkSchema(f.Type, path, out)
		case reflect.Slice:
			if elem := f.Type.Elem(); elem.Kind() == reflect.Struct {
				walkSchema(elem, path+"[]", out)
			}
		case reflect.Map:
			if elem := f.Type.Elem(); elem.Kind() == reflect.Struct {
				walkSchema(elem, path+".<key>", out)
			}
		}
	}
}

// yamlKey extracts the key name from a yaml struct tag (the part before the
// first comma), reporting false for an absent or "-" (explicitly skipped)
// tag.
func yamlKey(tag string) (string, bool) {
	if tag == "" || tag == "-" {
		return "", false
	}
	if i := strings.IndexByte(tag, ','); i >= 0 {
		tag = tag[:i]
	}
	return tag, true
}
