package policies

import (
	"go/ast"
	"go/parser"
	"testing"

	"github.com/23min/aiwf/internal/codes"
	"github.com/23min/aiwf/internal/workflows/spec"
)

// TestFindingClass_LegalityEnumerable is M-0140 / AC-1's evidence: the
// legality-pertinent kernel codes carry a structural Class marker that
// the AST scanner enumerates from their declarations (D-0011). Because
// the class is intrinsic to the descriptor value, the closed legality
// set derives from the same source it classifies — no parallel allowlist.
//
// The assertions exercise every arm the descriptor scanner feeds:
//
//   - the two real legality codes classify ClassLegality (the descriptor
//     arm reading `Class: ClassLegality` / `codes.ClassLegality`);
//   - a real structural-integrity code is present but classifies
//     ClassStructural (the bare-string const arm's default);
//   - the legality *set* derived by filtering on the marker contains the
//     legality codes and excludes the structural one — the exact
//     enumeration AC-5's fourth arm will consume.
func TestFindingClass_LegalityEnumerable(t *testing.T) {
	t.Parallel()

	m, err := collectImplFindingCodes(repoRoot(t))
	if err != nil {
		t.Fatalf("collectImplFindingCodes: %v", err)
	}

	// The two legality codes are named by their descriptors' Class field.
	legalityCodes := []string{
		"fsm-transition-illegal",
		"authorize-kind-not-allowed",
	}
	for _, code := range legalityCodes {
		class, ok := m[code]
		if !ok {
			t.Errorf("legality code %q not found in impl scan", code)
			continue
		}
		if class != codes.ClassLegality {
			t.Errorf("code %q classified %v, want ClassLegality", code, class)
		}
	}

	// A structural-integrity code is present but is NOT legality — it
	// stays a bare string const and defaults to ClassStructural.
	const structuralCode = "provenance-trailer-incoherent"
	class, ok := m[structuralCode]
	if !ok {
		t.Errorf("structural code %q not found in impl scan", structuralCode)
	} else if class != codes.ClassStructural {
		t.Errorf("code %q classified %v, want ClassStructural", structuralCode, class)
	}

	// Derive the legality set by filtering on the marker — the
	// enumeration the AC-5 fourth arm uses. It must contain the legality
	// codes and must not contain the structural one.
	legalitySet := map[string]bool{}
	for code, cls := range m {
		if cls == codes.ClassLegality {
			legalitySet[code] = true
		}
	}
	for _, code := range legalityCodes {
		if !legalitySet[code] {
			t.Errorf("derived legality set is missing %q", code)
		}
	}
	if legalitySet[structuralCode] {
		t.Errorf("derived legality set wrongly contains structural code %q", structuralCode)
	}
}

// TestM0140_AC3_M0138LegalityCodesRoundTrip is M-0140/AC-3: the two
// legality codes that exist at this milestone round-trip end to end —
// each is codes.ClassLegality on the impl side (the AC-1 descriptor
// marker) AND is the ExpectedErrorCode of >=1 OutcomeIllegal spec Rule on
// the spec side. It pins the concrete codes' full loop, so the AC-1
// descriptor migration or a spec edit that broke either half fails here.
//
// The epic's AC-3 also names the two cancel codes; those are emitted by
// M-0139 and certified there (the AC-2 fourth arm auto-includes them once
// M-0139 classifies them as ClassLegality) — chokepoint-first ordering,
// per D-0011 and the AC-3 spec note.
func TestM0140_AC3_M0138LegalityCodesRoundTrip(t *testing.T) {
	t.Parallel()

	implCodes, err := collectImplFindingCodes(repoRoot(t))
	if err != nil {
		t.Fatalf("collectImplFindingCodes: %v", err)
	}

	rules := spec.Rules()
	for _, code := range []string{"fsm-transition-illegal", "authorize-kind-not-allowed"} {
		// Impl side: the descriptor marks it ClassLegality.
		if got := implCodes[code]; got != codes.ClassLegality {
			t.Errorf("impl side: code %q classified %v, want ClassLegality", code, got)
		}
		// Spec side: >=1 illegal-outcome Rule names it as ExpectedErrorCode.
		n := 0
		for i := range rules {
			r := &rules[i]
			if r.Outcome == spec.OutcomeIllegal && r.ExpectedErrorCode == code {
				n++
			}
		}
		if n == 0 {
			t.Errorf("spec side: code %q is the ExpectedErrorCode of no illegal-outcome spec Rule", code)
		}
	}
}

// parseExpr parses a Go expression into an *ast.CompositeLit for the
// descriptor-arm branch tests. Fails the test if the source isn't a
// composite literal.
func parseCompositeLit(t *testing.T, src string) *ast.CompositeLit {
	t.Helper()
	e, err := parser.ParseExpr(src)
	if err != nil {
		t.Fatalf("parser.ParseExpr(%q): %v", src, err)
	}
	lit, ok := e.(*ast.CompositeLit)
	if !ok {
		t.Fatalf("ParseExpr(%q) = %T, want *ast.CompositeLit", src, e)
	}
	return lit
}

// TestDescriptorCode_Branches exercises every reachable arm of the
// scanner's descriptor recognizer that the live planning tree does not
// cover. The live tree declares both legality descriptors as
// `codes.Code{ID: ..., Class: codes.ClassLegality}` (a SelectorExpr type
// with a SelectorExpr Class value), so the bare-`Code` type form, the
// bare `ClassLegality` identifier, the explicit ClassStructural value,
// and the malformed/odd-shape rejection paths only run here.
func TestDescriptorCode_Branches(t *testing.T) {
	t.Parallel()

	t.Run("recognized descriptor cases", func(t *testing.T) {
		t.Parallel()
		cases := []struct {
			name     string
			src      string
			wantID   string
			wantClas codes.Class
		}{
			{
				// SelectorExpr type + SelectorExpr Class (the live-tree form).
				name:     "qualified type, qualified legality class",
				src:      `codes.Code{ID: "x-illegal", Class: codes.ClassLegality}`,
				wantID:   "x-illegal",
				wantClas: codes.ClassLegality,
			},
			{
				// Bare Ident type + bare Ident Class (same-package form).
				name:     "bare type, bare legality class",
				src:      `Code{ID: "y-illegal", Class: ClassLegality}`,
				wantID:   "y-illegal",
				wantClas: codes.ClassLegality,
			},
			{
				// Explicit structural class -> not legality.
				name:     "explicit structural class",
				src:      `Code{ID: "z-structural", Class: ClassStructural}`,
				wantID:   "z-structural",
				wantClas: codes.ClassStructural,
			},
			{
				// ID present, Class field absent -> defaults structural.
				name:     "no class field defaults structural",
				src:      `Code{ID: "no-class"}`,
				wantID:   "no-class",
				wantClas: codes.ClassStructural,
			},
			{
				// Class value is some other selector -> not legality.
				name:     "unrelated qualified class value",
				src:      `codes.Code{ID: "other", Class: codes.ClassStructural}`,
				wantID:   "other",
				wantClas: codes.ClassStructural,
			},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				id, class, ok := descriptorCode(parseCompositeLit(t, tc.src))
				if !ok {
					t.Fatalf("descriptorCode(%q) ok=false, want true", tc.src)
				}
				if id != tc.wantID {
					t.Errorf("id = %q, want %q", id, tc.wantID)
				}
				if class != tc.wantClas {
					t.Errorf("class = %v, want %v", class, tc.wantClas)
				}
			})
		}
	})

	t.Run("rejected (ok=false) cases", func(t *testing.T) {
		t.Parallel()
		cases := []struct {
			name string
			src  string
		}{
			// Type is not named Code -> typeNamedCode default arm.
			{"non-Code struct type", `Finding{Code: "f-1"}`},
			{"non-Code qualified type", `check.Finding{Code: "f-2"}`},
			// Named Code but no ID field -> gotID == "" reject.
			{"Code with only class, no ID", `Code{Class: ClassLegality}`},
			// ID present but not a string literal -> gotID stays empty.
			{"Code with non-string ID", `Code{ID: someVar, Class: ClassLegality}`},
			// Empty composite literal of a Code type -> no ID.
			{"empty Code literal", `Code{}`},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				if _, _, ok := descriptorCode(parseCompositeLit(t, tc.src)); ok {
					t.Errorf("descriptorCode(%q) ok=true, want false", tc.src)
				}
			})
		}
	})

	t.Run("class value branches", func(t *testing.T) {
		t.Parallel()
		// classValueIsLegality's non-ident default arm: a Class value that
		// is neither an Ident nor a SelectorExpr (e.g. a call expression)
		// reads as not-legality. descriptorCode then yields structural.
		id, class, ok := descriptorCode(parseCompositeLit(t, `Code{ID: "call-class", Class: classOf()}`))
		if !ok {
			t.Fatalf("ok=false, want true")
		}
		if id != "call-class" || class != codes.ClassStructural {
			t.Errorf("got (%q,%v), want (\"call-class\",ClassStructural)", id, class)
		}
	})

	t.Run("nil type is not a Code descriptor", func(t *testing.T) {
		t.Parallel()
		// typeNamedCode's nil/elided-type default arm — reachable via a
		// composite literal with no explicit type (as nested in a slice/map
		// literal). Build it directly since the source form `{ID: "x"}` is
		// not a standalone parseable expression.
		lit := &ast.CompositeLit{
			Type: nil,
			Elts: []ast.Expr{},
		}
		if _, _, ok := descriptorCode(lit); ok {
			t.Errorf("descriptorCode(nil-type lit) ok=true, want false")
		}
	})
}
