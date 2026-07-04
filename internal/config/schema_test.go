package config

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"gopkg.in/yaml.v3"
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

	// Description is out of scope here (AC-2's fieldDescriptions registry
	// owns it, pinned by TestSchema_EveryFieldHasDescription) — comparing it
	// too would duplicate that registry's content into a second place.
	ignoreDescription := cmpopts.IgnoreFields(SchemaField{}, "Description")

	// Order asserted as-is (no sort): Schema's doc comment promises
	// struct-declaration order, and want above is written in that order.
	if diff := cmp.Diff(want, got, ignoreDescription); diff != "" {
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

// TestSchema_EveryFieldHasDescription is the anti-drift backbone (M-0231
// AC-2): every path Schema() returns must have a non-empty entry in
// fieldDescriptions. A newly-added yaml field with no registry entry fails
// this test, rather than silently shipping an undocumented block.
func TestSchema_EveryFieldHasDescription(t *testing.T) {
	t.Parallel()
	for _, f := range Schema() {
		if f.Description == "" {
			t.Errorf("Schema() field %q has no description in fieldDescriptions", f.Path)
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

// uncommentYAML strips a leading "# " (after any indentation) from every
// line, simulating a consumer deleting the comment markers on a block they
// want to activate. It is test-only: the real interaction is a human
// manually uncommenting the lines they care about, never a programmatic step.
func uncommentYAML(s string) string {
	lines := strings.Split(s, "\n")
	for i, line := range lines {
		trimmed := strings.TrimLeft(line, " ")
		indent := line[:len(line)-len(trimmed)]
		if rest, ok := strings.CutPrefix(trimmed, "# "); ok {
			lines[i] = indent + rest
		}
	}
	return strings.Join(lines, "\n")
}

// TestGenerateExample_ProducesValidReparseableYAML pins M-0231 AC-3: once
// every comment marker is stripped, GenerateExample's output must be valid
// YAML that decodes into Config, and the nested/list/map rendering paths
// (areas.members' list-of-struct, agents' map-of-struct) must actually
// reconstruct the right structure — not just parse as *something*.
func TestGenerateExample_ProducesValidReparseableYAML(t *testing.T) {
	t.Parallel()
	uncommented := uncommentYAML(GenerateExample())

	var cfg Config
	if err := yaml.Unmarshal([]byte(uncommented), &cfg); err != nil {
		t.Fatalf("uncommented output does not decode into Config: %v\n---\n%s", err, uncommented)
	}

	var generic map[string]any
	if err := yaml.Unmarshal([]byte(uncommented), &generic); err != nil {
		t.Fatalf("uncommented output does not parse as a YAML mapping: %v", err)
	}

	tdd, ok := generic["tdd"].(map[string]any)
	if !ok {
		t.Fatalf("tdd is not a mapping: %#v", generic["tdd"])
	}
	if !hasKey(tdd, "strict") {
		t.Error("tdd.strict missing from parsed output")
	}

	areas, ok := generic["areas"].(map[string]any)
	if !ok {
		t.Fatalf("areas is not a mapping: %#v", generic["areas"])
	}
	members, ok := areas["members"].([]any)
	if !ok || len(members) == 0 {
		t.Fatalf("areas.members is not a non-empty list: %#v", areas["members"])
	}
	firstMember, ok := members[0].(map[string]any)
	if !ok {
		t.Fatalf("areas.members[0] is not a mapping: %#v", members[0])
	}
	if !hasKey(firstMember, "name") {
		t.Error("areas.members[0].name missing")
	}
	if !hasKey(firstMember, "paths") {
		t.Error("areas.members[0].paths missing")
	}

	agents, ok := generic["agents"].(map[string]any)
	if !ok {
		t.Fatalf("agents is not a mapping: %#v", generic["agents"])
	}
	exampleAgent, ok := agents["<key>"].(map[string]any)
	if !ok {
		t.Fatalf("agents.<key> is not a mapping: %#v", agents["<key>"])
	}
	if !hasKey(exampleAgent, "model") {
		t.Error("agents.<key>.model missing")
	}
	if !hasKey(exampleAgent, "effort") {
		t.Error("agents.<key>.effort missing")
	}
}

// TestGenerateExample_ScalarDefaultsPassLoaderValidation narrows AC-3's
// "uncommenting a block yields a value the loader accepts" constraint to
// what is actually true. Config.Validate() only checks two things —
// Areas.validate() and validateAgents() — so this test's real guarantee is
// narrower than "every scalar field individually validated": it confirms
// the decoded config, once the two example-item placeholder blocks
// (areas.members, agents.<key>) are cleared, passes the loader's full
// validation with no error from either check. The scalar fields themselves
// have no per-field Validate() logic to exercise; their claim rests on
// decoding without error (TestGenerateExample_ProducesValidReparseableYAML)
// plus each one being a real accessor-resolved default
// (TestDefaultFor_ResolverPaths), not on this test.
func TestGenerateExample_ScalarDefaultsPassLoaderValidation(t *testing.T) {
	t.Parallel()
	uncommented := uncommentYAML(GenerateExample())

	var cfg Config
	if err := yaml.Unmarshal([]byte(uncommented), &cfg); err != nil {
		t.Fatalf("uncommented output does not decode into Config: %v", err)
	}

	cfg.Areas.Members = nil
	cfg.Agents = nil

	if err := cfg.Validate(); err != nil {
		t.Errorf("Validate() rejected the scalar-default fields: %v", err)
	}
}

// TestGenerateExample_AreasExampleItemRejectedVerbatim documents, rather
// than silently ships, one of the two known example-item limitations: an
// independent fresh review of the first fix found this test's original name
// ("ExampleItemPlaceholdersRejectedVerbatim", plural) overclaimed — only
// areas.members actually fails Validate() verbatim (an empty member name is
// invalid); agents.<key> does not (see
// TestGenerateExample_AgentsExampleItemSilentlyAcceptedVerbatim below). If
// this test starts passing, the placeholder has silently become a valid
// default, which is worth a second look, not a quiet win.
func TestGenerateExample_AreasExampleItemRejectedVerbatim(t *testing.T) {
	t.Parallel()
	uncommented := uncommentYAML(GenerateExample())

	var cfg Config
	if err := yaml.Unmarshal([]byte(uncommented), &cfg); err != nil {
		t.Fatalf("uncommented output does not decode into Config: %v", err)
	}

	err := cfg.Validate()
	if err == nil {
		t.Fatal("Validate() unexpectedly accepted the areas.members example item verbatim — the placeholder-rejection contract may have changed")
	}
	if !strings.Contains(err.Error(), "empty member") {
		t.Errorf("Validate() error = %q, want it to name the empty example member", err.Error())
	}
}

// TestGenerateExample_AgentsExampleItemSilentlyAcceptedVerbatim pins the
// other half of the asymmetry: uncommenting agents.<key> verbatim (without
// replacing the placeholder key name) decodes to an Agent literally named
// "<key>" with empty Model/Effort, and Validate() accepts it — Agent.Model/
// Effort are optional by design (see the Agent doc comment), and an agent
// NAME is never validated against a closed set (config cannot enumerate the
// shipped agent set without importing skills), so nothing rejects the
// nonsense name. This is accepted-but-useless, not rejected — the opposite
// failure mode from areas.members, and worth pinning precisely rather than
// leaving as an assumption.
func TestGenerateExample_AgentsExampleItemSilentlyAcceptedVerbatim(t *testing.T) {
	t.Parallel()
	uncommented := uncommentYAML(GenerateExample())

	var cfg Config
	if err := yaml.Unmarshal([]byte(uncommented), &cfg); err != nil {
		t.Fatalf("uncommented output does not decode into Config: %v", err)
	}
	cfg.Areas.Members = nil // isolate the agents block from areas' own rejection

	if _, ok := cfg.Agents["<key>"]; !ok {
		t.Fatalf("expected the example agent literally named %q, got %#v", "<key>", cfg.Agents)
	}
	if err := cfg.Validate(); err != nil {
		t.Errorf("Validate() = %v, want nil — agents.<key> is expected to be silently accepted, not rejected", err)
	}
}

// hasKey reports whether m contains key, avoiding a govet shadow warning
// from repeated `if _, ok := m[key]; !ok` blocks in the same function scope.
func hasKey(m map[string]any, key string) bool {
	_, ok := m[key]
	return ok
}

// TestIsSliceOfStruct_MapOfStruct_StructContainer drives the three type
// predicates directly, including the map-of-a-plain-scalar shape ("map[string]int")
// that the real Config schema doesn't currently exercise — Agents is the
// only map field, and it happens to be map[string]config.Agent.
func TestIsSliceOfStruct_MapOfStruct_StructContainer(t *testing.T) {
	t.Parallel()
	cases := []struct {
		typ             string
		wantSliceStruct bool
		wantMapStruct   bool
		wantStructTop   bool
	}{
		{"[]config.Member", true, false, false},
		{"[]string", false, false, false},
		{"map[string]config.Agent", false, true, false},
		{"map[string]int", false, false, false},
		{"config.TDD", false, false, true},
		{"bool", false, false, false},
		{"string", false, false, false},
	}
	for _, c := range cases {
		if got := isSliceOfStruct(c.typ); got != c.wantSliceStruct {
			t.Errorf("isSliceOfStruct(%q) = %v, want %v", c.typ, got, c.wantSliceStruct)
		}
		if got := isMapOfStruct(c.typ); got != c.wantMapStruct {
			t.Errorf("isMapOfStruct(%q) = %v, want %v", c.typ, got, c.wantMapStruct)
		}
		if got := isStructContainer(c.typ); got != c.wantStructTop {
			t.Errorf("isStructContainer(%q) = %v, want %v", c.typ, got, c.wantStructTop)
		}
	}
}

// TestDefaultFor_HandlesAllLeafTypes drives defaultFor directly, including
// the "*bool" leaf shape that the real schema never reaches through this
// switch — both real *bool leaf fields (status_md.auto_update,
// guidance.wire_claudemd) have a fieldDefaultResolvers override and return
// before the switch runs.
func TestDefaultFor_HandlesAllLeafTypes(t *testing.T) {
	t.Parallel()
	cases := []struct {
		field SchemaField
		want  string
	}{
		{SchemaField{Path: "no.such.path", Type: "*bool"}, ""},
		{SchemaField{Path: "no.such.path", Type: "*int"}, ""},
		{SchemaField{Path: "no.such.path", Type: "[]string"}, "[]"},
		{SchemaField{Path: "no.such.path", Type: "bool"}, "false"},
		{SchemaField{Path: "no.such.path", Type: "string"}, `""`},
	}
	for _, c := range cases {
		if got := defaultFor(c.field); got != c.want {
			t.Errorf("defaultFor(%+v) = %q, want %q", c.field, got, c.want)
		}
	}
}

// TestDefaultFor_ResolverPaths pins every fieldDefaultResolvers entry to the
// value its real accessor actually returns (not a hand-copied literal) —
// each is compared against the getter/constant it wraps, so a resolver that
// silently stopped calling through would be caught here.
func TestDefaultFor_ResolverPaths(t *testing.T) {
	t.Parallel()
	zero := &Config{}
	wantTrunk, _ := zero.AllocateTrunkRef()
	cases := []struct {
		path string
		want string
	}{
		{"allocate.trunk", wantTrunk},
		{"html.out_dir", zero.HTMLOutDir()},
		{"entities.title_max_length", fmt.Sprintf("%d", zero.EntityTitleMaxLength())},
		{"worktree.dir", zero.WorktreeDir()},
		{"status_md.auto_update", fmt.Sprintf("%t", zero.StatusMdAutoUpdate())},
		{"guidance.wire_claudemd", fmt.Sprintf("%t", zero.WireClaudeMd())},
	}
	for _, c := range cases {
		got := defaultFor(SchemaField{Path: c.path, Type: "string"})
		if got != c.want {
			t.Errorf("defaultFor(%q) = %q, want %q (from the real accessor)", c.path, got, c.want)
		}
	}
}

// TestAcceptedKeys_MatchesSchemaPaths pins M-0231 AC-4: AcceptedKeys() is
// derived from Schema(), not a parallel hand-maintained list — the exact
// single-source guarantee G-0307's strict-decode guard is meant to consume
// (see G-0307's "Coordinate with E-0057" section).
func TestAcceptedKeys_MatchesSchemaPaths(t *testing.T) {
	t.Parallel()
	want := map[string]bool{}
	for _, f := range Schema() {
		want[f.Path] = true
	}

	got := AcceptedKeys()

	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("AcceptedKeys() mismatch vs Schema() paths (-want +got):\n%s", diff)
	}
}

// TestAcceptedKeys_MembershipChecks drives AcceptedKeys() through the actual
// consumer use case a strict-decode guard needs: exact membership at the
// top level and inside a nested block, and rejection of both a made-up
// top-level key and a typo'd nested key.
func TestAcceptedKeys_MembershipChecks(t *testing.T) {
	t.Parallel()
	keys := AcceptedKeys()

	cases := []struct {
		key  string
		want bool
	}{
		{"tdd", true},
		{"tdd.strict", true},
		{"tdd.stict", false}, // the exact typo G-0307 cites
		{"araes", false},     // the exact typo G-0307 cites
		{"agents.<key>.model", true},
	}
	for _, c := range cases {
		if got := keys[c.key]; got != c.want {
			t.Errorf("AcceptedKeys()[%q] = %v, want %v", c.key, got, c.want)
		}
	}
}

// TestFieldRegistries_NoOrphanKeys pins the reverse of the direction
// TestSchema_EveryFieldHasDescription already covers: not just "every real
// path has a registry entry," but "every registry entry is a real path."
// Without this, renaming or removing a yaml tag leaves the OLD registry key
// silently orphaned in fieldDescriptions or fieldDefaultResolvers — no
// finding, no test failure, just a resolver that will never match again and
// a description nobody reads (surfaced in the M-0231 design-quality review).
func TestFieldRegistries_NoOrphanKeys(t *testing.T) {
	t.Parallel()
	valid := AcceptedKeys()

	for path := range fieldDescriptions {
		if !valid[path] {
			t.Errorf("fieldDescriptions has orphan key %q — no such Schema() path", path)
		}
	}
	for path := range fieldDefaultResolvers {
		if !valid[path] {
			t.Errorf("fieldDefaultResolvers has orphan key %q — no such Schema() path", path)
		}
	}
}
