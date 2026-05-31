package policies

import (
	"os"
	"path/filepath"
	"testing"
)

// TestEnumerateCheckFindingCodeConstants_LiveTree pins that the
// enumerator reads internal/check/*.go at run time and builds a map
// from finding-code string literal value (e.g., "acs-shape") to
// constant identifier (e.g., "CodeACsShape"). The live tree's Code*
// constants are the authoritative input — adding a new typed
// finding-code must auto-extend the policy with no second source of
// truth.
func TestEnumerateCheckFindingCodeConstants_LiveTree(t *testing.T) {
	t.Parallel()
	consts, err := enumerateCheckFindingCodeConstants(repoRoot(t))
	if err != nil {
		t.Fatalf("enumerateCheckFindingCodeConstants: %v", err)
	}
	// Spot-check entries from each file family. Exhaustiveness check
	// would couple the test to internal/check/'s contents; the spot
	// check proves the enumeration runs and recognises representative
	// shapes across acs.go, archive_rules.go, check.go, entity_body.go,
	// entity_id_narrow_width.go, epic_active_drafts.go,
	// fsm_history_consistent.go, tree_discipline.go, and provenance.go.
	want := map[string]string{
		"acs-shape":                         "CodeACsShape",
		"archive-sweep-pending":             "CodeArchiveSweepPending",
		"ids-unique":                        "CodeIDsUnique",
		"entity-body-empty":                 "CodeEntityBodyEmpty",
		"entity-id-narrow-width":            "CodeEntityIDNarrowWidth",
		"epic-active-no-drafted-milestones": "CodeEpicActiveNoDraftedMilestones",
		"fsm-history-consistent":            "CodeFSMHistoryConsistent",
		"unexpected-tree-file":              "CodeUnexpectedTreeFile",
		"provenance-trailer-incoherent":     "CodeProvenanceTrailerIncoherent",
	}
	for value, wantName := range want {
		gotName, ok := consts[value]
		if !ok {
			t.Errorf("expected %q in enumerated set; got map size %d", value, len(consts))
			continue
		}
		if gotName != wantName {
			t.Errorf("for %q got %q, want %q", value, gotName, wantName)
		}
	}
}

// TestPolicyFindingCodeAdoption_FiresOnKeyValueExpr pins the emit-side
// chokepoint: a struct literal with `Code: "..."` keyed-field-value
// where the literal matches a known finding-code constant fires a
// violation pointing at the literal's line.
func TestPolicyFindingCodeAdoption_FiresOnKeyValueExpr(t *testing.T) {
	t.Parallel()
	root := buildSyntheticTreeForCodePolicy(t, "drift", `package drift

type Finding struct {
	Code string
}

func bad() Finding {
	return Finding{Code: "acs-shape"}
}
`)
	violations, err := PolicyFindingCodeAdoption(root)
	if err != nil {
		t.Fatalf("policy: %v", err)
	}
	found := false
	for _, v := range violations {
		if v.File == "internal/cli/drift/drift.go" && v.Line == 8 {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected violation at drift.go:8 (Code: \"acs-shape\"); got %+v", violations)
	}
}

// TestPolicyFindingCodeAdoption_FiresOnBinaryExpr covers the
// comparison-site detector: `code == "..."` matching a known
// finding-code constant fires a violation.
func TestPolicyFindingCodeAdoption_FiresOnBinaryExpr(t *testing.T) {
	t.Parallel()
	root := buildSyntheticTreeForCodePolicy(t, "drift", `package drift

func bad(code string) bool {
	return code == "acs-shape"
}
`)
	violations, err := PolicyFindingCodeAdoption(root)
	if err != nil {
		t.Fatalf("policy: %v", err)
	}
	found := false
	for _, v := range violations {
		if v.File == "internal/cli/drift/drift.go" && v.Line == 4 {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected violation at drift.go:4 (== \"acs-shape\"); got %+v", violations)
	}
}

// TestPolicyFindingCodeAdoption_FiresOnSwitchCase covers the
// switch/case detector for finding codes.
func TestPolicyFindingCodeAdoption_FiresOnSwitchCase(t *testing.T) {
	t.Parallel()
	root := buildSyntheticTreeForCodePolicy(t, "drift", `package drift

func bad(code string) string {
	switch code {
	case "acs-shape":
		return "yes"
	}
	return "no"
}
`)
	violations, err := PolicyFindingCodeAdoption(root)
	if err != nil {
		t.Fatalf("policy: %v", err)
	}
	found := false
	for _, v := range violations {
		if v.File == "internal/cli/drift/drift.go" && v.Line == 5 {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected violation at drift.go:5 (case \"acs-shape\"); got %+v", violations)
	}
}

// TestPolicyFindingCodeAdoption_IgnoreSuppressesKeyValueExpr pins the
// allowlist convention: `//enums:ignore <reason>` line-suffix comment
// suppresses the emit-side violation.
func TestPolicyFindingCodeAdoption_IgnoreSuppressesKeyValueExpr(t *testing.T) {
	t.Parallel()
	root := buildSyntheticTreeForCodePolicy(t, "drift", `package drift

type Finding struct {
	Code string
}

func bad() Finding {
	return Finding{Code: "acs-shape"} //enums:ignore intentional fixture literal
}
`)
	violations, err := PolicyFindingCodeAdoption(root)
	if err != nil {
		t.Fatalf("policy: %v", err)
	}
	for _, v := range violations {
		if v.File == "internal/cli/drift/drift.go" && v.Line == 8 {
			t.Errorf("expected violation suppressed by //enums:ignore; got %+v", v)
		}
	}
}

// TestPolicyFindingCodeAdoption_SilentOnTypedUsage pins that typed
// constant usage produces no violation — the rule is value-keyed
// against the constant table, so an `Ident`-shaped value (e.g.
// `check.CodeACsShape`) never matches because it isn't a BasicLit.
func TestPolicyFindingCodeAdoption_SilentOnTypedUsage(t *testing.T) {
	t.Parallel()
	root := buildSyntheticTreeForCodePolicy(t, "drift", `package drift

type Finding struct {
	Code string
}

const CodeACsShape = "acs-shape"

func ok() Finding {
	return Finding{Code: CodeACsShape}
}
`)
	violations, err := PolicyFindingCodeAdoption(root)
	if err != nil {
		t.Fatalf("policy: %v", err)
	}
	for _, v := range violations {
		if v.File == "internal/cli/drift/drift.go" {
			t.Errorf("typed constant usage should not fire; got %+v", v)
		}
	}
}

// buildSyntheticTreeForCodePolicy creates a tempdir with a synthetic
// internal/check/codes.go (carrying Code constants for the enumerator
// to read) plus a drift file at internal/cli/<pkgName>/drift.go
// containing the supplied body. Returns the tempdir as the policy's
// root.
func buildSyntheticTreeForCodePolicy(t *testing.T, pkgName, body string) string {
	t.Helper()
	root := t.TempDir()
	checkDir := filepath.Join(root, "internal", "check")
	if err := os.MkdirAll(checkDir, 0o755); err != nil {
		t.Fatal(err)
	}
	const checkGo = `package check

const (
	CodeACsShape          = "acs-shape"
	CodeIDsUnique         = "ids-unique"
	CodeEntityBodyEmpty   = "entity-body-empty"
	CodeFSMHistoryConsistent = "fsm-history-consistent"
)
`
	if err := os.WriteFile(filepath.Join(checkDir, "codes.go"), []byte(checkGo), 0o644); err != nil {
		t.Fatal(err)
	}
	driftDir := filepath.Join(root, "internal", "cli", pkgName)
	if err := os.MkdirAll(driftDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(driftDir, "drift.go"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return root
}
