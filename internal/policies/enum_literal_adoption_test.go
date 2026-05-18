package policies

import (
	"os"
	"path/filepath"
	"testing"
)

// TestEnumerateEntityStatusConstants_LiveTree pins AC-1: the policy
// reads internal/entity/entity.go at run time and builds a map from
// status string literal value (e.g., "open") to constant identifier
// (e.g., "StatusOpen"). The live tree's Status* constants are the
// authoritative input — adding a new status must auto-extend the
// rule with no second source of truth.
func TestEnumerateEntityStatusConstants_LiveTree(t *testing.T) {
	t.Parallel()
	consts, err := enumerateEntityStatusConstants(repoRoot(t))
	if err != nil {
		t.Fatalf("enumerateEntityStatusConstants: %v", err)
	}
	// Spot-check the known canonical entries. Exhaustiveness check
	// would couple the test to entity.go's contents; the spot check
	// proves the enumeration runs and recognises representative
	// shapes (epic, milestone, gap, adr, AC status values).
	want := map[string]string{
		"open":      "StatusOpen",
		"active":    "StatusActive",
		"done":      "StatusDone",
		"cancelled": "StatusCancelled",
		"draft":     "StatusDraft",
		"met":       "StatusMet",
		"deferred":  "StatusDeferred",
		"addressed": "StatusAddressed",
	}
	for value, wantName := range want {
		gotName, ok := consts[value]
		if !ok {
			t.Errorf("expected %q in enumerated set; got map %+v", value, consts)
			continue
		}
		if gotName != wantName {
			t.Errorf("for %q got %q, want %q", value, gotName, wantName)
		}
	}
}

// TestPolicyEnumLiteralAdoption_FiresOnBinaryExpr pins AC-2: a
// production source file with `s == "open"` outside internal/entity/
// produces a violation pointing at the literal's line.
func TestPolicyEnumLiteralAdoption_FiresOnBinaryExpr(t *testing.T) {
	t.Parallel()
	root := buildSyntheticTreeForEnumPolicy(t, "drift", `package drift

func bad(status string) bool {
	return status == "open"
}
`)
	violations, err := PolicyEnumLiteralAdoption(root)
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
		t.Errorf("expected violation at drift.go:4 (== \"open\"); got %+v", violations)
	}
}

// TestPolicyEnumLiteralAdoption_FiresOnBangEq covers the `!=` branch.
func TestPolicyEnumLiteralAdoption_FiresOnBangEq(t *testing.T) {
	t.Parallel()
	root := buildSyntheticTreeForEnumPolicy(t, "drift", `package drift

func bad(status string) bool {
	return status != "done"
}
`)
	violations, err := PolicyEnumLiteralAdoption(root)
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
		t.Errorf("expected violation at drift.go:4 (!= \"done\"); got %+v", violations)
	}
}

// buildSyntheticTreeForEnumPolicy creates a tempdir with a synthetic
// internal/entity/entity.go (carrying Status constants for the
// enumerator to read) plus a drift file at
// internal/cli/<pkgName>/drift.go containing the supplied body.
// Returns the tempdir as the policy's root.
func buildSyntheticTreeForEnumPolicy(t *testing.T, pkgName, body string) string {
	t.Helper()
	root := t.TempDir()
	entityDir := filepath.Join(root, "internal", "entity")
	if err := os.MkdirAll(entityDir, 0o755); err != nil {
		t.Fatal(err)
	}
	const entityGo = `package entity

const (
	StatusOpen      = "open"
	StatusActive    = "active"
	StatusDone      = "done"
	StatusCancelled = "cancelled"
	StatusDraft     = "draft"
	StatusMet       = "met"
)
`
	if err := os.WriteFile(filepath.Join(entityDir, "entity.go"), []byte(entityGo), 0o644); err != nil {
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

// TestPolicyEnumLiteralAdoption_FiresOnSwitchCase pins AC-3: a
// switch statement with a literal case clause matching a known
// status fires a violation pointing at the case expression's line.
func TestPolicyEnumLiteralAdoption_FiresOnSwitchCase(t *testing.T) {
	t.Parallel()
	root := buildSyntheticTreeForEnumPolicy(t, "drift", `package drift

func bad(status string) string {
	switch status {
	case "active":
		return "yes"
	}
	return "no"
}
`)
	violations, err := PolicyEnumLiteralAdoption(root)
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
		t.Errorf("expected violation at drift.go:5 (case \"active\"); got %+v", violations)
	}
}

// TestPolicyEnumLiteralAdoption_IgnoreSuppressesBinaryExpr pins AC-4:
// a `//enums:ignore <reason>` comment on the same line as the literal
// suppresses the violation.
func TestPolicyEnumLiteralAdoption_IgnoreSuppressesBinaryExpr(t *testing.T) {
	t.Parallel()
	root := buildSyntheticTreeForEnumPolicy(t, "drift", `package drift

func bad(status string) bool {
	return status == "open" //enums:ignore parsing legacy fixture
}
`)
	violations, err := PolicyEnumLiteralAdoption(root)
	if err != nil {
		t.Fatalf("policy: %v", err)
	}
	for _, v := range violations {
		if v.File == "internal/cli/drift/drift.go" && v.Line == 4 {
			t.Errorf("expected violation suppressed by //enums:ignore; got %+v", v)
		}
	}
}

// TestPolicyEnumLiteralAdoption_IgnoreSuppressesSwitchCase mirrors
// AC-4 across the switch/case detector. Same line-suffix comment
// shape.
func TestPolicyEnumLiteralAdoption_IgnoreSuppressesSwitchCase(t *testing.T) {
	t.Parallel()
	root := buildSyntheticTreeForEnumPolicy(t, "drift", `package drift

func bad(status string) string {
	switch status {
	case "active": //enums:ignore parsing legacy fixture
		return "yes"
	}
	return "no"
}
`)
	violations, err := PolicyEnumLiteralAdoption(root)
	if err != nil {
		t.Fatalf("policy: %v", err)
	}
	for _, v := range violations {
		if v.File == "internal/cli/drift/drift.go" && v.Line == 5 {
			t.Errorf("expected violation suppressed by //enums:ignore; got %+v", v)
		}
	}
}

// TestPolicyEnumLiteralAdoption_AcceptsConstantReference pins the
// no-false-positive shape: when the comparison uses the constant
// reference (not a literal), no violation fires.
func TestPolicyEnumLiteralAdoption_AcceptsConstantReference(t *testing.T) {
	t.Parallel()
	root := buildSyntheticTreeForEnumPolicy(t, "ok", `package ok

import "github.com/23min/aiwf/internal/entity"

func good(status string) bool {
	return status == entity.StatusOpen
}
`)
	violations, err := PolicyEnumLiteralAdoption(root)
	if err != nil {
		t.Fatalf("policy: %v", err)
	}
	for _, v := range violations {
		if v.File == "internal/cli/ok/drift.go" {
			t.Errorf("expected no violation on constant reference; got %+v", v)
		}
	}
}

