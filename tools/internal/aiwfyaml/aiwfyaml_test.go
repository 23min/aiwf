package aiwfyaml

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
)

const baseConfig = `aiwf_version: 0.1.0
actor: human/peter
`

func TestRead_NoContractsBlock(t *testing.T) {
	_, c, err := ReadBytes([]byte(baseConfig))
	if err != nil {
		t.Fatalf("ReadBytes: %v", err)
	}
	if c != nil {
		t.Errorf("Contracts = %+v, want nil for file with no contracts: block", c)
	}
}

func TestRead_BasicBlock(t *testing.T) {
	src := baseConfig + `
contracts:
  validators:
    cue:
      command: cue
      args: [vet, "{{schema}}", "{{fixture}}"]
  entries:
    - id: C-001
      validator: cue
      schema: docs/schemas/opspec/schema.cue
      fixtures: docs/schemas/opspec/fixtures
`
	_, c, err := ReadBytes([]byte(src))
	if err != nil {
		t.Fatalf("ReadBytes: %v", err)
	}
	if c == nil {
		t.Fatal("Contracts is nil")
	}
	wantValidators := map[string]Validator{
		"cue": {Command: "cue", Args: []string{"vet", "{{schema}}", "{{fixture}}"}},
	}
	if diff := cmp.Diff(wantValidators, c.Validators); diff != "" {
		t.Errorf("validators mismatch (-want +got):\n%s", diff)
	}
	wantEntries := []Entry{{
		ID:        "C-001",
		Validator: "cue",
		Schema:    "docs/schemas/opspec/schema.cue",
		Fixtures:  "docs/schemas/opspec/fixtures",
	}}
	if diff := cmp.Diff(wantEntries, c.Entries); diff != "" {
		t.Errorf("entries mismatch (-want +got):\n%s", diff)
	}
}

func TestRead_RejectsAnchor(t *testing.T) {
	src := baseConfig + `
contracts:
  validators:
    base: &base
      command: cue
      args: [vet]
    cue: *base
  entries: []
`
	_, _, err := ReadBytes([]byte(src))
	if err == nil {
		t.Fatal("expected error for anchor inside contracts:")
	}
	if !strings.Contains(err.Error(), "anchor") {
		t.Errorf("error %q does not mention anchors", err)
	}
}

func TestRead_RejectsAlias(t *testing.T) {
	// Alias in another section is fine; only aliases *inside* contracts:
	// are rejected. We exercise the alias-inside-contracts path by
	// declaring an anchor outside and aliasing inside.
	src := `aiwf_version: 0.1.0
actor: human/peter
shared: &cue_args
  - vet
  - "{{schema}}"
contracts:
  validators:
    cue:
      command: cue
      args: *cue_args
  entries: []
`
	_, _, err := ReadBytes([]byte(src))
	if err == nil {
		t.Fatal("expected error for alias inside contracts:")
	}
	if !strings.Contains(err.Error(), "alias") && !strings.Contains(err.Error(), "anchor") {
		t.Errorf("error %q does not mention alias/anchor", err)
	}
}

func TestRead_RejectsUnknownField(t *testing.T) {
	src := baseConfig + `
contracts:
  validators:
    cue:
      command: cue
      args: [vet]
  entries:
    - id: C-001
      validator: cue
      schema: s
      fixtures: f
      mystery: nope
`
	_, _, err := ReadBytes([]byte(src))
	if err == nil {
		t.Fatal("expected error for unknown field 'mystery' in entry")
	}
	if !strings.Contains(err.Error(), "mystery") {
		t.Errorf("error %q does not name the unknown field", err)
	}
}

func TestRead_RejectsBadID(t *testing.T) {
	src := baseConfig + `
contracts:
  validators:
    cue:
      command: cue
      args: [vet]
  entries:
    - id: X-001
      validator: cue
      schema: s
      fixtures: f
`
	_, _, err := ReadBytes([]byte(src))
	if err == nil {
		t.Fatal("expected error for non-C-NNN id")
	}
}

func TestRead_RejectsUndeclaredValidator(t *testing.T) {
	src := baseConfig + `
contracts:
  validators:
    cue:
      command: cue
      args: [vet]
  entries:
    - id: C-001
      validator: ghost
      schema: s
      fixtures: f
`
	_, _, err := ReadBytes([]byte(src))
	if err == nil {
		t.Fatal("expected error for undeclared validator name")
	}
	if !strings.Contains(err.Error(), "ghost") {
		t.Errorf("error %q does not mention the undeclared validator", err)
	}
}

func TestSetContracts_AppendsWhenAbsent(t *testing.T) {
	d, c, err := ReadBytes([]byte(baseConfig))
	if err != nil {
		t.Fatalf("ReadBytes: %v", err)
	}
	if c != nil {
		t.Fatal("Contracts should be nil before SetContracts")
	}

	in := &Contracts{
		Validators: map[string]Validator{
			"cue": {Command: "cue", Args: []string{"vet", "{{schema}}", "{{fixture}}"}},
		},
		Entries: []Entry{{
			ID: "C-001", Validator: "cue", Schema: "s.cue", Fixtures: "f",
		}},
	}
	if err = d.SetContracts(in); err != nil {
		t.Fatalf("SetContracts: %v", err)
	}

	got := string(d.Bytes())
	if !strings.HasPrefix(got, baseConfig) {
		t.Errorf("base content lost; got:\n%s", got)
	}
	if !strings.Contains(got, "contracts:") {
		t.Errorf("contracts: block missing; got:\n%s", got)
	}

	// Round-trip: parse the post-write bytes and compare.
	_, back, err := ReadBytes(d.Bytes())
	if err != nil {
		t.Fatalf("re-read: %v", err)
	}
	if diff := cmp.Diff(in, back); diff != "" {
		t.Errorf("round-trip mismatch (-want +got):\n%s", diff)
	}
}

func TestSetContracts_PreservesOuterCommentsAndOrder(t *testing.T) {
	src := `# Top-of-file comment
aiwf_version: 0.1.0
actor: human/peter # actor comment
hosts: [claude-code]

contracts:
  validators:
    cue:
      command: cue
      args: [vet]
  entries:
    - id: C-001
      validator: cue
      schema: old/schema.cue
      fixtures: old/fixtures
`
	d, c, err := ReadBytes([]byte(src))
	if err != nil {
		t.Fatalf("ReadBytes: %v", err)
	}
	if c == nil {
		t.Fatal("Contracts is nil")
	}

	// Mutate the schema path and write back.
	c.Entries[0].Schema = "new/schema.cue"
	if err := d.SetContracts(c); err != nil {
		t.Fatalf("SetContracts: %v", err)
	}

	got := string(d.Bytes())

	// Outside the contracts: block: every byte from the original up to
	// the "contracts:" line must be byte-identical.
	idx := strings.Index(src, "contracts:")
	if idx < 0 {
		t.Fatal("source has no contracts: token (test setup wrong)")
	}
	wantPrefix := src[:idx]
	if !strings.HasPrefix(got, wantPrefix) {
		t.Errorf("outer content changed.\nwant prefix:\n%q\ngot:\n%q", wantPrefix, got[:min(len(got), len(wantPrefix)+64)])
	}

	// Updated path made it through.
	if !strings.Contains(got, "new/schema.cue") {
		t.Errorf("updated schema path missing from output:\n%s", got)
	}
	if strings.Contains(got, "old/schema.cue") {
		t.Errorf("stale schema path still present:\n%s", got)
	}
}

func TestSetContracts_ReplaceMidFile(t *testing.T) {
	// `contracts:` is *not* the last top-level key; verify the splice
	// stops at the line of the next key (`hosts:`) and content after
	// it survives.
	src := `aiwf_version: 0.1.0
actor: human/peter
contracts:
  validators:
    cue:
      command: cue
      args: [vet]
  entries:
    - id: C-001
      validator: cue
      schema: old/schema.cue
      fixtures: old/fixtures
hosts:
  - claude-code
`
	d, c, err := ReadBytes([]byte(src))
	if err != nil {
		t.Fatalf("ReadBytes: %v", err)
	}
	c.Entries[0].Schema = "new.cue"
	if err := d.SetContracts(c); err != nil {
		t.Fatalf("SetContracts: %v", err)
	}
	got := string(d.Bytes())

	// `hosts:` block must still be present and intact at the tail.
	if !strings.Contains(got, "hosts:\n  - claude-code\n") {
		t.Errorf("trailing hosts: block damaged:\n%s", got)
	}
	if !strings.Contains(got, "new.cue") {
		t.Errorf("new schema path missing:\n%s", got)
	}
	if strings.Contains(got, "old/schema.cue") {
		t.Errorf("stale schema path retained:\n%s", got)
	}
}

func TestSetContracts_RemovesWithNil(t *testing.T) {
	src := baseConfig + `
contracts:
  validators:
    cue:
      command: cue
      args: [vet]
  entries: []
`
	d, c, err := ReadBytes([]byte(src))
	if err != nil {
		t.Fatalf("ReadBytes: %v", err)
	}
	if c == nil {
		t.Fatal("Contracts is nil pre-remove")
	}
	if err = d.SetContracts(nil); err != nil {
		t.Fatalf("SetContracts(nil): %v", err)
	}
	got := string(d.Bytes())
	if strings.Contains(got, "contracts:") {
		t.Errorf("contracts: block still present after remove:\n%s", got)
	}
	// The base content must still be there and parseable.
	_, back, err := ReadBytes(d.Bytes())
	if err != nil {
		t.Fatalf("re-read after remove: %v", err)
	}
	if back != nil {
		t.Errorf("Contracts non-nil after remove: %+v", back)
	}
}

func TestSetContracts_RoundTripIsStable(t *testing.T) {
	in := &Contracts{
		Validators: map[string]Validator{
			"cue":        {Command: "cue", Args: []string{"vet", "{{schema}}", "{{fixture}}"}},
			"jsonschema": {Command: "ajv", Args: []string{"validate", "-s", "{{schema}}", "-d", "{{fixture}}"}},
		},
		Entries: []Entry{
			{ID: "C-001", Validator: "cue", Schema: "a.cue", Fixtures: "fa"},
			{ID: "C-002", Validator: "jsonschema", Schema: "b.json", Fixtures: "fb"},
		},
	}
	d, _, err := ReadBytes([]byte(baseConfig))
	if err != nil {
		t.Fatalf("ReadBytes: %v", err)
	}
	if err = d.SetContracts(in); err != nil {
		t.Fatalf("SetContracts: %v", err)
	}
	first := append([]byte(nil), d.Bytes()...)

	d2, c, err := ReadBytes(first)
	if err != nil {
		t.Fatalf("re-read: %v", err)
	}
	if err := d2.SetContracts(c); err != nil {
		t.Fatalf("SetContracts second pass: %v", err)
	}
	if !cmp.Equal(first, d2.Bytes()) {
		t.Errorf("second SetContracts not stable.\nfirst:\n%s\nsecond:\n%s", first, d2.Bytes())
	}
}

func TestWrite_AtomicAndIdempotent(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "aiwf.yaml")
	d, _, err := ReadBytes([]byte(baseConfig))
	if err != nil {
		t.Fatalf("ReadBytes: %v", err)
	}
	if err := d.Write(path); err != nil {
		t.Fatalf("Write: %v", err)
	}
	// Write again with the same bytes — must succeed.
	if err := d.Write(path); err != nil {
		t.Fatalf("Write second: %v", err)
	}
}

func TestValidate_RejectsEmptyCommand(t *testing.T) {
	c := &Contracts{
		Validators: map[string]Validator{
			"empty": {Command: "", Args: nil},
		},
	}
	if err := c.Validate(); err == nil {
		t.Error("expected error for empty command")
	}
}

func TestValidate_RejectsEmptyValidatorKey(t *testing.T) {
	c := &Contracts{
		Validators: map[string]Validator{
			"": {Command: "x", Args: nil},
		},
	}
	if err := c.Validate(); err == nil {
		t.Error("expected error for empty validator key")
	}
}
