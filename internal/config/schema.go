package config

import (
	"fmt"
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
	// Description is a one-line, consumer-facing summary of the field,
	// looked up from fieldDescriptions by Path. Empty when the path has no
	// registry entry — a state the anti-drift test in schema_test.go treats
	// as a failure for every path Schema() actually returns.
	Description string
}

// fieldDescriptions is the explicit, hand-maintained registry of one-line
// descriptions keyed by schema Path (locked design decision, M-0231: doc
// comments in config.go attach at the struct level, not per field, so
// go/ast field-attachment isn't viable — see the milestone's Design notes).
// This is the anti-drift backbone: schema_test.go's
// TestSchema_EveryFieldHasDescription fails whenever Schema() returns a path
// with no entry here.
var fieldDescriptions = map[string]string{
	"hosts": "Supported host list; the PoC default and only supported value is claude-code.",

	"status_md":             "Opt-out for the pre-commit hook that keeps STATUS.md in sync with the entity tree.",
	"status_md.auto_update": "Whether the STATUS.md auto-update hook is installed (default true).",

	"tdd":                      "Opt-in governance for the acceptance-criteria TDD workflow.",
	"tdd.require_test_metrics": "Require an aiwf-tests: trailer on every AC promoted to done under tdd: required (default false).",
	"tdd.strict":               "Promote TDD-related warnings to errors so the pre-push hook blocks the push (default false).",

	"html":               "Settings for the static site rendered by aiwf render --format=html.",
	"html.out_dir":       "Directory the HTML renderer writes into (default \"site\").",
	"html.commit_output": "Commit the rendered HTML output instead of gitignoring it (default false).",

	"allocate":       "Configuration for the entity id allocator.",
	"allocate.trunk": "Git ref the allocator treats as trunk when scanning for existing ids (default refs/remotes/origin/main).",

	"tree":             "Policy for what may live under work/ alongside the entity tree.",
	"tree.allow_paths": "Repo-relative glob patterns exempted from the tree-discipline check.",
	"tree.strict":      "Promote unexpected-tree-file from a warning to a blocking error (default false).",

	"archive":                 "Drift-control configuration for the per-kind archive convention.",
	"archive.sweep_threshold": "Terminal-entity count past which archive-sweep-pending escalates from advisory to a blocking error (unset: always advisory).",

	"entities":                  "Policy for entity-shape constraints the kernel enforces when writing entity files.",
	"entities.title_max_length": "Maximum length for an entity title and slug (default 80).",

	"guidance":               "Opt-out for aiwf maintaining its per-turn LLM guidance import in the consumer's CLAUDE.md.",
	"guidance.wire_claudemd": "Whether aiwf wires and self-heals the guidance import in CLAUDE.md (default true).",

	"areas":                 "Declares the closed set of workstream area tags entities may carry.",
	"areas.members":         "Declared area members, each a name and optional source-path globs.",
	"areas.members[].name":  "The area tag entities carry in their area: frontmatter field.",
	"areas.members[].paths": "Repo-relative glob patterns locating this area's source in a monorepo.",
	"areas.default":         "Display label for the untagged complement in grouped views (never a member itself).",
	"areas.required":        "Require every self-tagging entity to declare a member area (default false).",
	"areas.coverage_roots":  "Directories whose immediate child directories must each be claimed by some area's paths.",

	"worktree":     "Default placement for the git worktrees the start rituals create.",
	"worktree.dir": "Repo-relative directory ritual worktrees are placed under (default .claude/worktrees).",

	"agents":              "Per-agent model tier and reasoning-effort overrides for shipped role agents.",
	"agents.<key>.model":  "Model alias this agent's card is materialized with (opus, sonnet, haiku, fable, inherit).",
	"agents.<key>.effort": "Reasoning-effort level this agent's card is materialized with (low, medium, high, xhigh, max).",
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
	for i := range fields {
		fields[i].Description = fieldDescriptions[fields[i].Path]
	}
	return fields
}

// AcceptedKeys returns the full set of accepted aiwf.yaml key paths (the
// same paths Schema() enumerates), as a set for O(1) membership checks. This
// is the single source G-0307's strict-decode guard is meant to validate
// against, rather than a hand-maintained parallel allowlist — see G-0307's
// "Coordinate with E-0057" section. Every call recomputes from Schema()
// (no cached package-level state), matching Schema()'s own no-caching shape.
func AcceptedKeys() map[string]bool {
	keys := make(map[string]bool)
	for _, f := range Schema() {
		keys[f.Path] = true
	}
	return keys
}

// fieldDefaultResolvers overrides the effective default for the few leaf
// fields whose Go zero value would not match what config.Load actually
// applies (the locked design decision: call the real accessor, never
// hand-transcribe a duplicate literal — see M-0231's Design notes). Every
// other leaf field's zero value is already the true default, so defaultFor
// renders it generically from the field's Type.
var fieldDefaultResolvers = map[string]func() string{
	"allocate.trunk": func() string {
		ref, _ := (&Config{}).AllocateTrunkRef()
		return ref
	},
	"html.out_dir": func() string { return (&Config{}).HTMLOutDir() },
	"entities.title_max_length": func() string {
		return fmt.Sprintf("%d", (&Config{}).EntityTitleMaxLength())
	},
	"worktree.dir": func() string { return (&Config{}).WorktreeDir() },
	"status_md.auto_update": func() string {
		return fmt.Sprintf("%t", (&Config{}).StatusMdAutoUpdate())
	},
	"guidance.wire_claudemd": func() string {
		return fmt.Sprintf("%t", (&Config{}).WireClaudeMd())
	},
}

// defaultFor renders the effective default for a leaf schema field as it
// should appear after the YAML colon. A path in fieldDefaultResolvers calls
// the real accessor; every other leaf type covers exactly the five leaf
// shapes the current schema contains (*bool/*int, []string, bool, string) —
// see the milestone's Design notes for why each is provably zero-value-safe.
func defaultFor(f SchemaField) string {
	if resolve, ok := fieldDefaultResolvers[f.Path]; ok {
		return resolve()
	}
	switch f.Type {
	case "*bool", "*int":
		return ""
	case "[]string":
		return "[]"
	case "bool":
		return "false"
	default: // "string"
		return `""`
	}
}

// isSliceOfStruct reports whether a schema Type string is a slice whose
// element is a nested config struct (e.g. "[]config.Member"), as opposed to
// a slice of a plain scalar (e.g. "[]string").
func isSliceOfStruct(t string) bool {
	return strings.HasPrefix(t, "[]") && strings.Contains(t, "config.")
}

// isMapOfStruct reports whether a schema Type string is a map whose value
// is a nested config struct (e.g. "map[string]config.Agent"), as opposed to
// a map of a plain scalar.
func isMapOfStruct(t string) bool {
	return strings.HasPrefix(t, "map[string]") && strings.Contains(t, "config.")
}

// isStructContainer reports whether a schema Type string is a directly
// nested config struct block (e.g. "config.TDD").
func isStructContainer(t string) bool {
	return strings.HasPrefix(t, "config.")
}

// GenerateExample renders Schema() as fully-commented, reparseable YAML: a
// consumer reads it as reference and uncomments only the blocks they want to
// override, so every line — container and leaf alike — stays commented (the
// file is inert until acted on). A leaf field renders as
// "# key: <default>  # <description>"; a struct-typed field renders as
// "# key:  # <description>" with its children nested two spaces deeper. A
// slice-of-struct field's first child is dash-prefixed to form one example
// list item — this assumes a flat, non-nested element type, true of the
// current schema (Member has no struct-typed field of its own). A
// map-of-struct field gets a synthetic "<key>:" placeholder line between the
// map and its example entry's fields, since no real SchemaField represents
// that placeholder.
func GenerateExample() string {
	var b strings.Builder
	inListItem := false
	for _, f := range Schema() {
		depth := strings.Count(f.Path, ".")
		key := f.Path[strings.LastIndex(f.Path, ".")+1:]
		spaces := strings.Repeat("  ", depth)
		dash := ""
		if inListItem {
			spaces = strings.Repeat("  ", depth-1)
			dash = "- "
			inListItem = false
		}

		switch {
		case isSliceOfStruct(f.Type):
			fmt.Fprintf(&b, "%s# %s%s:  # %s\n", spaces, dash, key, f.Description)
			inListItem = true
		case isMapOfStruct(f.Type):
			fmt.Fprintf(&b, "%s# %s%s:  # %s\n", spaces, dash, key, f.Description)
			fmt.Fprintf(&b, "%s  # <key>:\n", spaces)
		case isStructContainer(f.Type):
			fmt.Fprintf(&b, "%s# %s%s:  # %s\n", spaces, dash, key, f.Description)
		default:
			fmt.Fprintf(&b, "%s# %s%s: %s  # %s\n", spaces, dash, key, defaultFor(f), f.Description)
		}
	}
	return b.String()
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
