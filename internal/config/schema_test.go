package config

import (
	"reflect"
	"testing"

	"github.com/google/go-cmp/cmp"
)

// TestSchema_EnumeratesEveryYAMLField pins Schema()'s coverage of the full
// aiwf.yaml field set (E-0057/M-0231 AC-1): every yaml-tagged field on the
// Config struct tree — including nested-struct, slice-of-struct, and
// map-of-struct fields — contributes exactly one SchemaField. The two legacy
// migration-shim fields (LegacyAiwfVersion, LegacyActor) are deliberately
// excluded: they are decode-only compatibility fields, never a documented,
// hand-authorable key (see the config package doc).
func TestSchema_EnumeratesEveryYAMLField(t *testing.T) {
	t.Parallel()
	want := []SchemaField{
		{Path: "hosts", Type: "[]string"},
		{Path: "status_md", Type: "config.StatusMd"},
		{Path: "status_md.auto_update", Type: "*bool"},
		{Path: "tdd", Type: "config.TDD"},
		{Path: "tdd.require_test_metrics", Type: "bool"},
		{Path: "tdd.strict", Type: "bool"},
		{Path: "html", Type: "config.HTML"},
		{Path: "html.out_dir", Type: "string"},
		{Path: "html.commit_output", Type: "bool"},
		{Path: "allocate", Type: "config.Allocate"},
		{Path: "allocate.trunk", Type: "string"},
		{Path: "tree", Type: "config.Tree"},
		{Path: "tree.allow_paths", Type: "[]string"},
		{Path: "tree.strict", Type: "bool"},
		{Path: "archive", Type: "config.Archive"},
		{Path: "archive.sweep_threshold", Type: "*int"},
		{Path: "entities", Type: "config.Entities"},
		{Path: "entities.title_max_length", Type: "*int"},
		{Path: "guidance", Type: "config.Guidance"},
		{Path: "guidance.wire_claudemd", Type: "*bool"},
		{Path: "areas", Type: "config.Areas"},
		{Path: "areas.members", Type: "[]config.Member"},
		{Path: "areas.members[].name", Type: "string"},
		{Path: "areas.members[].paths", Type: "[]string"},
		{Path: "areas.default", Type: "string"},
		{Path: "areas.required", Type: "bool"},
		{Path: "areas.coverage_roots", Type: "[]string"},
		{Path: "worktree", Type: "config.Worktree"},
		{Path: "worktree.dir", Type: "string"},
		{Path: "agents", Type: "map[string]config.Agent"},
		{Path: "agents.<key>.model", Type: "string"},
		{Path: "agents.<key>.effort", Type: "string"},
	}

	got := Schema()

	// Order asserted as-is (no sort): Schema's doc comment promises
	// struct-declaration order, and want above is written in that order.
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("Schema() mismatch (-want +got):\n%s", diff)
	}
}

// TestSchema_ExcludesLegacyFields pins that the two decode-only legacy
// migration fields never surface in the generated schema reference, even
// though they carry yaml tags and decode on read.
func TestSchema_ExcludesLegacyFields(t *testing.T) {
	t.Parallel()
	for _, f := range Schema() {
		if f.Path == "aiwf_version" || f.Path == "actor" {
			t.Errorf("Schema() includes legacy field %q; want excluded", f.Path)
		}
	}
}

// walkSchemaFixtureChild and walkSchemaFixtureRoot exist only to drive
// walkSchema field shapes the real Config struct doesn't currently exercise:
// an untagged field, an explicitly "-"-tagged field, and a map field whose
// value type is not a struct. Every real Config map field (Agents) happens
// to have a struct value, so that branch is otherwise unreached.
type walkSchemaFixtureChild struct {
	Inner string `yaml:"inner"`
}

type walkSchemaFixtureRoot struct {
	Untagged  string
	Excluded  string                            `yaml:"-"`
	LegacyOne string                            `yaml:"legacy_one"`
	Scalar    string                            `yaml:"scalar"`
	Nested    walkSchemaFixtureChild            `yaml:"nested"`
	Ints      []int                             `yaml:"ints"`
	Children  []walkSchemaFixtureChild          `yaml:"children"`
	Counts    map[string]int                    `yaml:"counts"`
	ChildMap  map[string]walkSchemaFixtureChild `yaml:"child_map"`
}

// TestWalkSchema_HandlesAllFieldShapes drives walkSchema directly against a
// fixture type covering every field shape the switch in walkSchema
// branches on: an untagged field and an explicit `yaml:"-"` field (both must
// be skipped), a field named with the "Legacy" exclusion prefix, a plain
// scalar, a nested struct, a slice of a non-struct and of a struct, and a map
// of a non-struct and of a struct. The real Config struct never exercises
// the untagged/"-"/non-struct-map cases, so this fixture is the only path
// that reaches them.
func TestWalkSchema_HandlesAllFieldShapes(t *testing.T) {
	t.Parallel()
	var got []SchemaField
	walkSchema(reflect.TypeFor[walkSchemaFixtureRoot](), "", &got)

	want := []SchemaField{
		{Path: "scalar", Type: "string"},
		{Path: "nested", Type: "config.walkSchemaFixtureChild"},
		{Path: "nested.inner", Type: "string"},
		{Path: "ints", Type: "[]int"},
		{Path: "children", Type: "[]config.walkSchemaFixtureChild"},
		{Path: "children[].inner", Type: "string"},
		{Path: "counts", Type: "map[string]int"},
		{Path: "child_map", Type: "map[string]config.walkSchemaFixtureChild"},
		{Path: "child_map.<key>.inner", Type: "string"},
	}

	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("walkSchema mismatch (-want +got):\n%s", diff)
	}
}
