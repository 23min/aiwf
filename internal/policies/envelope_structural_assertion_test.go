package policies

import (
	"os"
	"path/filepath"
	"testing"
)

// TestPolicyEnvelopeStructuralAssertion_FiresOnRenamedField proves
// the policy catches the exact regression class M-0239/AC-4 exists
// to prevent: a future contributor renames an Envelope field (and
// its json tag) without updating this policy's pinned required-key
// set, silently breaking any downstream JSON consumer that reads the
// old key name. Per CLAUDE.md's "test the seam" discipline, a
// positive-only test against the live (clean) repo proves nothing
// about whether the policy actually fires on drift — this drives it
// against a synthetic fixture tree.
func TestPolicyEnvelopeStructuralAssertion_FiresOnRenamedField(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	dir := filepath.Join(root, "internal", "render")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	// "findings" renamed to "issues" — everything else unchanged.
	src := `package render

type Envelope struct {
	Tool     string          ` + "`json:\"tool\"`" + `
	Version  string          ` + "`json:\"version\"`" + `
	Status   string          ` + "`json:\"status\"`" + `
	Findings []check.Finding ` + "`json:\"issues\"`" + `
	Result   any             ` + "`json:\"result,omitempty\"`" + `
	Error    *EnvelopeError  ` + "`json:\"error,omitempty\"`" + `
	Metadata map[string]any  ` + "`json:\"metadata,omitempty\"`" + `
}
`
	if err := os.WriteFile(filepath.Join(dir, "render.go"), []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	violations, err := PolicyEnvelopeStructuralAssertion(root)
	if err != nil {
		t.Fatalf("policy: %v", err)
	}
	if len(violations) == 0 {
		t.Fatal("expected a violation on the renamed-field fixture; got none")
	}
	found := false
	for _, v := range violations {
		if v.Policy == "envelope-structural-assertion" && v.File == "internal/render/render.go" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected a envelope-structural-assertion violation on internal/render/render.go; got: %+v", violations)
	}
}

// TestPolicyEnvelopeStructuralAssertion_FiresOnMissingField proves
// the policy also catches a dropped field (not just a rename) — a
// field removed from the struct without the consuming policy being
// updated to match.
func TestPolicyEnvelopeStructuralAssertion_FiresOnMissingField(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	dir := filepath.Join(root, "internal", "render")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	// "metadata" dropped entirely.
	src := `package render

type Envelope struct {
	Tool     string          ` + "`json:\"tool\"`" + `
	Version  string          ` + "`json:\"version\"`" + `
	Status   string          ` + "`json:\"status\"`" + `
	Findings []check.Finding ` + "`json:\"findings\"`" + `
	Result   any             ` + "`json:\"result,omitempty\"`" + `
	Error    *EnvelopeError  ` + "`json:\"error,omitempty\"`" + `
}
`
	if err := os.WriteFile(filepath.Join(dir, "render.go"), []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	violations, err := PolicyEnvelopeStructuralAssertion(root)
	if err != nil {
		t.Fatalf("policy: %v", err)
	}
	if len(violations) == 0 {
		t.Fatal("expected a violation on the missing-field fixture; got none")
	}
}

// TestPolicyEnvelopeStructuralAssertion_AcceptsCurrentShape proves
// the policy accepts the Envelope struct's actual, current field
// tags without a false positive.
func TestPolicyEnvelopeStructuralAssertion_AcceptsCurrentShape(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	dir := filepath.Join(root, "internal", "render")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	src := `package render

type Envelope struct {
	Tool     string          ` + "`json:\"tool\"`" + `
	Version  string          ` + "`json:\"version\"`" + `
	Status   string          ` + "`json:\"status\"`" + `
	Findings []check.Finding ` + "`json:\"findings\"`" + `
	Result   any             ` + "`json:\"result,omitempty\"`" + `
	Error    *EnvelopeError  ` + "`json:\"error,omitempty\"`" + `
	Metadata map[string]any  ` + "`json:\"metadata,omitempty\"`" + `
}
`
	if err := os.WriteFile(filepath.Join(dir, "render.go"), []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	violations, err := PolicyEnvelopeStructuralAssertion(root)
	if err != nil {
		t.Fatalf("policy: %v", err)
	}
	if len(violations) != 0 {
		t.Errorf("policy fired on the current, correct shape: %+v", violations)
	}
}

// TestPolicyEnvelopeStructuralAssertion_MissingTypeEntirely proves
// the policy's own "no Envelope struct found" guard fires when the
// type has been renamed or removed outright, not just when its
// fields drift — a strictly larger break than a field rename.
func TestPolicyEnvelopeStructuralAssertion_MissingTypeEntirely(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	dir := filepath.Join(root, "internal", "render")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	src := `package render

type NotEnvelope struct {
	Tool string ` + "`json:\"tool\"`" + `
}
`
	if err := os.WriteFile(filepath.Join(dir, "render.go"), []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	violations, err := PolicyEnvelopeStructuralAssertion(root)
	if err != nil {
		t.Fatalf("policy: %v", err)
	}
	if len(violations) == 0 {
		t.Fatal("expected a violation when the Envelope type itself is missing; got none")
	}
}

// TestPolicyEnvelopeStructuralAssertion_NonStructEnvelope proves the
// policy treats `type Envelope <non-struct>` the same as a missing
// type — a type alias masquerading as the canonical envelope has no
// fields for the policy to check, which is itself the violation.
func TestPolicyEnvelopeStructuralAssertion_NonStructEnvelope(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	dir := filepath.Join(root, "internal", "render")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	src := `package render

type Envelope = map[string]any
`
	if err := os.WriteFile(filepath.Join(dir, "render.go"), []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	violations, err := PolicyEnvelopeStructuralAssertion(root)
	if err != nil {
		t.Fatalf("policy: %v", err)
	}
	if len(violations) == 0 {
		t.Fatal("expected a violation when Envelope is not a struct type; got none")
	}
}

// TestPolicyEnvelopeStructuralAssertion_ParseError proves a
// syntactically broken render.go surfaces as a Go error (not a
// silent pass or a panic) — distinct from the other policies in this
// package, which skip-and-continue past a parse error while scanning
// many files; this policy pins exactly one file, so a parse failure
// there is a real, visible problem worth surfacing rather than
// swallowing.
func TestPolicyEnvelopeStructuralAssertion_ParseError(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	dir := filepath.Join(root, "internal", "render")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	src := `package render

type Envelope struct {
`
	if err := os.WriteFile(filepath.Join(dir, "render.go"), []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := PolicyEnvelopeStructuralAssertion(root); err == nil {
		t.Fatal("expected a parse error on syntactically broken render.go; got nil")
	}
}

// TestPolicyEnvelopeStructuralAssertion_UnreadableFile proves a
// generic (non-NotExist) read failure surfaces as a Go error too —
// the sibling of the parse-error test above, covering os.ReadFile's
// other failure mode.
func TestPolicyEnvelopeStructuralAssertion_UnreadableFile(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	dir := filepath.Join(root, "internal", "render")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "render.go")
	if err := os.WriteFile(path, []byte("package render\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(path, 0o000); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chmod(path, 0o644) }() // restore so t.TempDir() cleanup can remove it
	if _, err := PolicyEnvelopeStructuralAssertion(root); err == nil {
		t.Fatal("expected a read error on an unreadable render.go; got nil")
	}
}

// TestPolicyEnvelopeStructuralAssertion_IgnoresUntaggedAndDashFields
// covers collectEnvelopeJSONTags's two field-skip branches: a field
// with no tag at all (a plausible mistake — a new field added without
// a json tag), and a field explicitly excluded via `json:"-"` (a
// deliberate, valid Go idiom this policy must not misread as a
// pinned key). Neither should appear in the found set, and since both
// happen alongside the correct current shape, the policy must not
// fire on this fixture.
func TestPolicyEnvelopeStructuralAssertion_IgnoresUntaggedAndDashFields(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	dir := filepath.Join(root, "internal", "render")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	src := `package render

type Envelope struct {
	Tool       string          ` + "`json:\"tool\"`" + `
	Version    string          ` + "`json:\"version\"`" + `
	Status     string          ` + "`json:\"status\"`" + `
	Findings   []check.Finding ` + "`json:\"findings\"`" + `
	Result     any             ` + "`json:\"result,omitempty\"`" + `
	Error      *EnvelopeError  ` + "`json:\"error,omitempty\"`" + `
	Metadata   map[string]any  ` + "`json:\"metadata,omitempty\"`" + `
	Untagged   string
	Unexported string ` + "`json:\"-\"`" + `
}
`
	if err := os.WriteFile(filepath.Join(dir, "render.go"), []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	violations, err := PolicyEnvelopeStructuralAssertion(root)
	if err != nil {
		t.Fatalf("policy: %v", err)
	}
	if len(violations) != 0 {
		t.Errorf("policy fired on untagged/dash-tagged fields alongside the correct required set: %+v", violations)
	}
}
