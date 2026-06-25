package aiwfyaml

import (
	"strings"
	"testing"
)

// TestSetAreas_RenameMemberPreservesEverythingElse pins the core
// area-member rewrite (E-0044, M-0177): SetAreas splices a rewritten
// areas block back, renaming one member while preserving member order,
// the leading top-level keys, and comments outside the block.
func TestSetAreas_RenameMemberPreservesEverythingElse(t *testing.T) {
	t.Parallel()
	src := `# top-of-file comment
hosts: [claude-code]

# the workstream areas
areas:
  members:
    - platform
    - billing
`
	doc, _, err := ReadBytes([]byte(src))
	if err != nil {
		t.Fatalf("ReadBytes: %v", err)
	}
	if err := doc.SetAreas([]string{"infra", "billing"}, ""); err != nil {
		t.Fatalf("SetAreas: %v", err)
	}
	got := string(doc.Bytes())
	want := `# top-of-file comment
hosts: [claude-code]

# the workstream areas
areas:
  members:
    - infra
    - billing
`
	if got != want {
		t.Errorf("SetAreas rewrote unexpected bytes\n got: %q\nwant: %q", got, want)
	}
}

// TestSetAreas_PreservesDefaultLabel pins that the `default:` display
// label survives a member rename when carried back through SetAreas.
func TestSetAreas_PreservesDefaultLabel(t *testing.T) {
	t.Parallel()
	src := `areas:
  members:
    - platform
    - billing
  default: untagged
`
	doc, _, err := ReadBytes([]byte(src))
	if err != nil {
		t.Fatalf("ReadBytes: %v", err)
	}
	if err := doc.SetAreas([]string{"infra", "billing"}, "untagged"); err != nil {
		t.Fatalf("SetAreas: %v", err)
	}
	got := string(doc.Bytes())
	if !strings.Contains(got, "default: untagged") {
		t.Errorf("default label dropped:\n%s", got)
	}
	if !strings.Contains(got, "- infra") || strings.Contains(got, "- platform") {
		t.Errorf("member not renamed:\n%s", got)
	}
}

// TestSetAreas_PreservesTrailingKeysAndComments pins that a top-level
// key and its comment AFTER the areas block survive the splice — the
// byte-range replacement must stop at the next top-level key.
func TestSetAreas_PreservesTrailingKeysAndComments(t *testing.T) {
	t.Parallel()
	src := `areas:
  members:
    - platform
    - billing

# trailing comment belongs to html
html:
  out_dir: site
`
	doc, _, err := ReadBytes([]byte(src))
	if err != nil {
		t.Fatalf("ReadBytes: %v", err)
	}
	if err := doc.SetAreas([]string{"infra", "billing"}, ""); err != nil {
		t.Fatalf("SetAreas: %v", err)
	}
	got := string(doc.Bytes())
	for _, want := range []string{"# trailing comment belongs to html", "html:", "out_dir: site", "- infra"} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q after splice:\n%s", want, got)
		}
	}
	if strings.Contains(got, "- platform") {
		t.Errorf("old member still present:\n%s", got)
	}
}

// TestSetAreas_NoAreasBlockErrors pins the refusal path: SetAreas on a
// Doc with no areas block returns an error rather than fabricating one.
// The verb never reaches this (it only renames declared members), but
// the guard keeps the API honest.
func TestSetAreas_NoAreasBlockErrors(t *testing.T) {
	t.Parallel()
	doc, _, err := ReadBytes([]byte("hosts: [claude-code]\n"))
	if err != nil {
		t.Fatalf("ReadBytes: %v", err)
	}
	if err := doc.SetAreas([]string{"platform"}, ""); err == nil {
		t.Fatal("SetAreas on a doc with no areas block should error")
	}
}

// TestSetAreas_QuotesMemberNeedingQuoting pins that a member whose value
// would be misread unquoted (a YAML reserved word, a leading digit) is
// emitted quoted, matching the contracts writer's yamlScalar behavior.
func TestSetAreas_QuotesMemberNeedingQuoting(t *testing.T) {
	t.Parallel()
	src := `areas:
  members:
    - platform
`
	doc, _, err := ReadBytes([]byte(src))
	if err != nil {
		t.Fatalf("ReadBytes: %v", err)
	}
	if err := doc.SetAreas([]string{`"true"`}, ""); err != nil {
		t.Fatalf("SetAreas: %v", err)
	}
	// The member value is the literal string `"true"` — yamlScalar
	// double-quotes any value containing a quote character, so the
	// emitted member must be quoted, not a bare YAML boolean.
	got := string(doc.Bytes())
	if !strings.Contains(got, `- "`) {
		t.Errorf("member needing quoting was emitted bare:\n%s", got)
	}
}

// TestReadBytes_DetectsAreasWithoutContracts pins that the areas block
// is detected even when no contracts: block is present — the contracts
// path returns early on a missing contracts key, so areas detection
// must not depend on it.
func TestReadBytes_DetectsAreasWithoutContracts(t *testing.T) {
	t.Parallel()
	src := `hosts: [claude-code]
areas:
  members:
    - platform
    - billing
`
	doc, contracts, err := ReadBytes([]byte(src))
	if err != nil {
		t.Fatalf("ReadBytes: %v", err)
	}
	if contracts != nil {
		t.Errorf("contracts = %+v, want nil", contracts)
	}
	// SetAreas succeeding proves the areas block was located.
	if err := doc.SetAreas([]string{"infra", "billing"}, ""); err != nil {
		t.Fatalf("SetAreas after areas-only read: %v", err)
	}
	if !strings.Contains(string(doc.Bytes()), "- infra") {
		t.Errorf("areas block not rewritten:\n%s", doc.Bytes())
	}
}

// TestSetAreas_AreasBeforeContracts pins that when both blocks exist,
// rewriting areas leaves the later contracts block intact — the
// areas byte range must stop at the contracts key.
func TestSetAreas_AreasBeforeContracts(t *testing.T) {
	t.Parallel()
	src := `areas:
  members:
    - platform
    - billing
contracts:
  validators: {}
  entries: []
`
	doc, _, err := ReadBytes([]byte(src))
	if err != nil {
		t.Fatalf("ReadBytes: %v", err)
	}
	if err := doc.SetAreas([]string{"infra", "billing"}, ""); err != nil {
		t.Fatalf("SetAreas: %v", err)
	}
	got := string(doc.Bytes())
	for _, want := range []string{"- infra", "contracts:", "validators: {}", "entries: []"} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q after areas splice:\n%s", want, got)
		}
	}
}
