package policies

import (
	"go/ast"
	"go/parser"
	"go/token"
	"strings"
	"testing"
)

// parseViolationSnippet parses a synthetic policy source for the pure core.
func parseViolationSnippet(t *testing.T, src string) (*token.FileSet, *ast.File) {
	t.Helper()
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "synthetic.go", src, 0)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	return fset, f
}

// TestNonLiteralPolicyIDs drives the pure core. The ban's whole value is
// that it refuses the shapes the firing-fixture inventory cannot see, so
// each non-literal spelling gets its own case.
func TestNonLiteralPolicyIDs(t *testing.T) {
	t.Parallel()

	const header = "package policies\n\nconst someID = \"x\"\n\nvar v = "

	tests := []struct {
		name  string
		expr  string
		wantN int
	}{
		{
			name:  "a string literal is allowed",
			expr:  `Violation{Policy: "my-policy", File: "f"}`,
			wantN: 0,
		},
		{
			name:  "a named constant is refused",
			expr:  `Violation{Policy: someID, File: "f"}`,
			wantN: 1,
		},
		{
			name:  "a qualified constant is refused",
			expr:  `Violation{Policy: other.ID, File: "f"}`,
			wantN: 1,
		},
		{
			name:  "a concatenation is refused",
			expr:  `Violation{Policy: "my-" + someID, File: "f"}`,
			wantN: 1,
		},
		{
			name:  "a call result is refused",
			expr:  `Violation{Policy: idOf(), File: "f"}`,
			wantN: 1,
		},
		{
			name:  "a package-qualified Violation is still scanned",
			expr:  `policies.Violation{Policy: someID}`,
			wantN: 1,
		},
		{
			name:  "a composite literal of another type is ignored",
			expr:  `Finding{Policy: someID}`,
			wantN: 0,
		},
		{
			name:  "a Violation with no Policy field is ignored",
			expr:  `Violation{File: "f"}`,
			wantN: 0,
		},
		{
			name:  "a positional Violation carries no key to inspect",
			expr:  `Violation{"a", "b", "c"}`,
			wantN: 0,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			fset, f := parseViolationSnippet(t, header+tt.expr+"\n")
			got := nonLiteralPolicyIDs(fset, f, "internal/policies/synthetic.go")
			if len(got) != tt.wantN {
				t.Fatalf("got %d violations, want %d: %+v", len(got), tt.wantN, got)
			}
			for _, v := range got {
				if v.Policy != "violation-policy-id-literal" {
					t.Errorf("violation Policy = %q, want violation-policy-id-literal", v.Policy)
				}
				if !strings.Contains(v.Detail, "not a string literal") {
					t.Errorf("Detail must say what is wrong; got %q", v.Detail)
				}
			}
		})
	}
}

// TestPolicy_ViolationPolicyIDLiteral is the CI gate entry point: every
// Violation in this package's own production sources spells its policy
// id inline, so PolicyFiringFixturePresence's line-matching inventory
// sees all of them.
func TestPolicy_ViolationPolicyIDLiteral(t *testing.T) {
	t.Parallel()
	runPolicy(t, PolicyViolationPolicyIDLiteral)
}

// TestPolicyViolationPolicyIDLiteral_ReadsRealSources confirms the walk
// reaches this package's sources rather than silently scanning nothing —
// the failure mode that would make the gate above vacuous. A tree with no
// internal/policies directory is an error, not silence.
func TestPolicyViolationPolicyIDLiteral_ReadsRealSources(t *testing.T) {
	t.Parallel()

	// The real tree parses and yields no violation.
	if _, err := PolicyViolationPolicyIDLiteral(repoRoot(t)); err != nil {
		t.Fatalf("scanning the real tree: %v", err)
	}
	// Proof the scan is not vacuous: the same walk over a tree with no
	// internal/policies reports an error rather than an empty pass.
	if _, err := PolicyViolationPolicyIDLiteral(t.TempDir()); err == nil {
		t.Error("want an error when internal/policies is absent, got nil")
	}
}
