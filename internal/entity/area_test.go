package entity

import (
	"os"
	"strings"
	"testing"
)

// TestParse_AreaOnRootKinds pins AC-1 of M-0171: the five root kinds (epic,
// ADR, gap, decision, contract) accept an optional `area` frontmatter field,
// and absent/empty parses clean — no error, empty Area, no default written.
// Milestones derive their area from the parent epic (AC-3) and are not
// exercised here.
func TestParse_AreaOnRootKinds(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		fm   string
		want string
	}{
		{"epic", "id: E-0001\ntitle: Foo\nstatus: active\narea: platform", "platform"},
		{"adr", "id: ADR-0001\ntitle: Foo\nstatus: accepted\narea: platform", "platform"},
		{"gap", "id: G-0001\ntitle: Foo\nstatus: open\narea: tooling", "tooling"},
		{"decision", "id: D-0001\ntitle: Foo\nstatus: open\narea: tooling", "tooling"},
		{"contract", "id: C-0001\ntitle: Foo\nstatus: accepted\narea: api", "api"},
		{"absent", "id: E-0002\ntitle: Foo\nstatus: active", ""},
		{"empty", "id: E-0003\ntitle: Foo\nstatus: active\narea: \"\"", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			content := []byte("---\n" + tc.fm + "\n---\n")
			got, err := Parse(tc.name+".md", content)
			if err != nil {
				t.Fatalf("Parse: %v", err)
			}
			if got.Area != tc.want {
				t.Errorf("Area = %q, want %q", got.Area, tc.want)
			}
		})
	}
}

// TestAreaField_DocumentsForwardCompat pins the AC-5-folded forward-compat note
// on AC-1: the `area` field site must document that a pre-`area` binary rejects
// a file using it via the generic KnownFields(true) strict-decoder window. The
// assertion is scoped to the field's own contiguous doc comment, not a flat
// file grep (per the repo's "substring assertions are not structural
// assertions" rule).
func TestAreaField_DocumentsForwardCompat(t *testing.T) {
	t.Parallel()
	src, err := os.ReadFile("entity.go")
	if err != nil {
		t.Fatalf("read entity.go: %v", err)
	}
	lines := strings.Split(string(src), "\n")
	fieldIdx := -1
	for i, l := range lines {
		if strings.Contains(l, `yaml:"area,omitempty"`) {
			fieldIdx = i
			break
		}
	}
	if fieldIdx < 0 {
		t.Fatal(`no field with yaml:"area,omitempty" found in entity.go`)
	}
	var comment []string
	for i := fieldIdx - 1; i >= 0; i-- {
		s := strings.TrimSpace(lines[i])
		if strings.HasPrefix(s, "//") {
			comment = append([]string{s}, comment...)
			continue
		}
		break
	}
	block := strings.ToLower(strings.Join(comment, " "))
	if !strings.Contains(block, "knownfields") {
		t.Errorf("area field doc comment must name the KnownFields strict-decoder window; got:\n%s", strings.Join(comment, "\n"))
	}
}

// TestParse_AreaKnownButUnknownSiblingRejected is the behavior half of AC-1's
// forward-compat evidence: `area` is a KNOWN field (parses), but the strict
// decoder still rejects an unknown sibling — the KnownFields(true) window the
// field-site note documents. Pairs with TestAreaField_DocumentsForwardCompat
// (which pins the prose) so both the claim and the guarantee are covered.
func TestParse_AreaKnownButUnknownSiblingRejected(t *testing.T) {
	t.Parallel()
	content := []byte("---\nid: E-0001\ntitle: Foo\nstatus: active\narea: platform\nbogus: nope\n---\n")
	_, err := Parse("x.md", content)
	if err == nil || !strings.Contains(err.Error(), "bogus") {
		t.Errorf("err = %v, want a 'field bogus' rejection", err)
	}
}
