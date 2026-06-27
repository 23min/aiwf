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

// TestIsValidAreaValue pins AC-1 of M-0184: the single definition of "valid
// non-empty area value" — the reserved AreaGlobal sentinel, or any declared
// member. Empty is not valid here (absence is area-required's concern). The
// global sentinel is valid regardless of the declared set, including a nil
// member list.
func TestIsValidAreaValue(t *testing.T) {
	t.Parallel()
	members := []string{"platform", "billing"}
	cases := []struct {
		name    string
		value   string
		members []string
		want    bool
	}{
		{"reserved global sentinel", AreaGlobal, members, true},
		{"declared member", "platform", members, true},
		{"another declared member", "billing", members, true},
		{"undeclared value", "tooling", members, false},
		{"empty value", "", members, false},
		// Position A (M-0184): with no declared members the area dimension
		// is inert (M-0171), so NOTHING is valid — not even the reserved
		// global sentinel. The predicate is THE definition of "valid area
		// value", so it gates global here rather than relying on each
		// caller's pre-guard.
		{"global with nil members is inert", AreaGlobal, nil, false},
		{"global with empty members is inert", AreaGlobal, []string{}, false},
		{"member-looking value with nil members", "platform", nil, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := IsValidAreaValue(tc.value, tc.members); got != tc.want {
				t.Errorf("IsValidAreaValue(%q, %v) = %v, want %v", tc.value, tc.members, got, tc.want)
			}
		})
	}
}

// TestAreaGlobal_Value pins the reserved sentinel literal so a rename of the
// constant surfaces here rather than silently shifting the wire value every
// other call site routes through.
func TestAreaGlobal_Value(t *testing.T) {
	t.Parallel()
	if AreaGlobal != "global" {
		t.Errorf("AreaGlobal = %q, want %q", AreaGlobal, "global")
	}
}

// TestCarriesOwnArea pins the single-source-of-truth predicate for which
// kinds store their own `area` versus derive it from a parent (E-0043):
// a milestone derives from its parent epic and never self-tags; the five
// other root kinds carry their own area.
func TestCarriesOwnArea(t *testing.T) {
	t.Parallel()
	if CarriesOwnArea(KindMilestone) {
		t.Errorf("CarriesOwnArea(KindMilestone) = true, want false (derives from parent epic)")
	}
	for _, k := range []Kind{KindEpic, KindGap, KindADR, KindDecision, KindContract} {
		if !CarriesOwnArea(k) {
			t.Errorf("CarriesOwnArea(%v) = false, want true (self-tagging root kind)", k)
		}
	}
}
