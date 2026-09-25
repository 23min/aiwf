package entity

import (
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"gopkg.in/yaml.v3"
)

// TestSerialize_WritesReferencesAtCanonicalWidth: a reference field
// loaded or handed at a narrow width is written canonical, while the
// entity's own id, its prior_ids and its commit SHAs are written as they
// are, and the entity passed in is left unchanged.
func TestSerialize_WritesReferencesAtCanonicalWidth(t *testing.T) {
	t.Parallel()
	e := &Entity{
		ID:                "M-007",
		Title:             "Narrow references",
		Status:            "draft",
		PriorIDs:          []string{"M-003"},
		Parent:            "E-01",
		DependsOn:         []string{"M-002", "M-0005"},
		Supersedes:        []string{"ADR-0001"},
		SupersededBy:      "ADR-0002",
		DiscoveredIn:      "M-001",
		AddressedBy:       []string{"M-001"},
		AddressedByCommit: []string{"4b13a0f"},
		RelatesTo:         []string{"E-01", "G-055"},
		LinkedADRs:        []string{"ADR-0003"},
	}
	// Copy the slices too: a struct copy shares their arrays with e, so a
	// Serialize that rewrote a slice in place would change both alike.
	before := *e
	for _, f := range []*[]string{&before.PriorIDs, &before.DependsOn, &before.Supersedes, &before.AddressedBy, &before.AddressedByCommit, &before.RelatesTo, &before.LinkedADRs} {
		*f = slices.Clone(*f)
	}
	out, err := Serialize(e, nil)
	if err != nil {
		t.Fatal(err)
	}
	got, err := Parse("m.md", out)
	if err != nil {
		t.Fatalf("re-parse: %v", err)
	}
	want := before
	want.Parent = "E-0001"
	want.DependsOn = []string{"M-0002", "M-0005"}
	want.DiscoveredIn = "M-0001"
	want.AddressedBy = []string{"M-0001"}
	want.RelatesTo = []string{"E-0001", "G-0055"}
	want.Kind, want.Path = got.Kind, got.Path // set by Parse, not written by Serialize
	if diff := cmp.Diff(&want, got); diff != "" {
		t.Errorf("serialized entity mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(&before, e); diff != "" {
		t.Errorf("Serialize modified the entity it was given (-before +after):\n%s", diff)
	}
}

// notReferences names every string-valued frontmatter field that
// Serialize writes as it is, each with the reason it is not an id
// reference. Every other string-valued field is a reference and must be
// written canonical.
var notReferences = map[string]string{
	"id":                  "the entity's own id names its file; only reallocation changes it",
	"title":               "free text",
	"status":              "a status name",
	"area":                "an area name",
	"priority":            "a priority level",
	"tdd":                 "a policy name",
	"prior_ids":           "records the ids the entity carried before, at the width it carried them",
	"addressed_by_commit": "commit SHAs, not entity ids",
}

// TestSerialize_EveryStringFieldIsCanonicalizedOrNamedExempt walks the
// Entity struct, so a frontmatter field added later is judged without
// anyone remembering to list it: set to a narrow id, it must come back
// canonical unless notReferences names it with a reason.
func TestSerialize_EveryStringFieldIsCanonicalizedOrNamedExempt(t *testing.T) {
	t.Parallel()
	const narrow = "M-001"
	var e Entity
	v := reflect.ValueOf(&e).Elem()
	var fields []string
	for i := 0; i < v.NumField(); i++ {
		f := v.Type().Field(i)
		name, _, _ := strings.Cut(f.Tag.Get("yaml"), ",")
		if name == "" || name == "-" {
			continue
		}
		switch {
		case f.Type.Kind() == reflect.String:
			v.Field(i).SetString(narrow)
		case f.Type.Kind() == reflect.Slice && f.Type.Elem().Kind() == reflect.String:
			v.Field(i).Set(reflect.ValueOf([]string{narrow}))
		default:
			continue
		}
		fields = append(fields, name)
	}
	out, err := Serialize(&e, nil)
	if err != nil {
		t.Fatal(err)
	}
	fm, _, _ := Split(out)
	var written map[string]any
	if err := yaml.Unmarshal(fm, &written); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	for _, name := range fields {
		value := strings.Trim(strings.ReplaceAll(strings.TrimSpace(yamlString(written[name])), "\n", ""), "[]")
		_, exempt := notReferences[name]
		switch {
		case exempt && value != narrow:
			t.Errorf("field %s is named exempt but was rewritten to %q", name, value)
		case !exempt && value != Canonicalize(narrow):
			t.Errorf("field %s was written as %q; a reference field must be written canonical, or be named in notReferences with a reason", name, value)
		}
	}
	for name := range notReferences {
		if !slices.Contains(fields, name) {
			t.Errorf("notReferences names %s, which is no string-valued frontmatter field", name)
		}
	}
}

// yamlString renders a decoded scalar or one-element sequence as text.
func yamlString(v any) string {
	switch x := v.(type) {
	case string:
		return x
	case []any:
		if len(x) == 1 {
			if s, ok := x[0].(string); ok {
				return s
			}
		}
	}
	return ""
}
