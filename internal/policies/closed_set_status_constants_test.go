package policies

import (
	"os"
	"path/filepath"
	"testing"
)

// TestPolicyClosedSetStatusViaConstants_FiresOnPriorityLiteral pins
// G-0078/E-0066/M-0261 AC-3: a `.Priority ==` comparison against a
// known priority literal fires closed-set-status-via-constants,
// mirroring the policy's existing `.Status ==` coverage.
func TestPolicyClosedSetStatusViaConstants_FiresOnPriorityLiteral(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	dir := filepath.Join(root, "internal", "cli", "drift")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	const body = `package drift

type entity struct {
	Priority string
}

func bad(e entity) bool {
	return e.Priority == "urgent"
}
`
	if err := os.WriteFile(filepath.Join(dir, "drift.go"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	violations, err := PolicyClosedSetStatusViaConstants(root)
	if err != nil {
		t.Fatalf("policy: %v", err)
	}
	found := false
	for _, v := range violations {
		if v.Policy == "closed-set-status-via-constants" && v.Line == 8 {
			found = true
		}
	}
	if !found {
		t.Errorf("expected a closed-set-status-via-constants violation at line 8; got %+v", violations)
	}
}

// TestPolicyClosedSetStatusViaConstants_FiresOnPriorityBangEq mirrors
// the `.Status !=` context pattern for `.Priority !=`, added purely
// for symmetry with the `==` pattern above.
func TestPolicyClosedSetStatusViaConstants_FiresOnPriorityBangEq(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	dir := filepath.Join(root, "internal", "cli", "drift")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	const body = `package drift

type entity struct {
	Priority string
}

func bad(e entity) bool {
	return e.Priority != "low"
}
`
	if err := os.WriteFile(filepath.Join(dir, "drift.go"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	violations, err := PolicyClosedSetStatusViaConstants(root)
	if err != nil {
		t.Fatalf("policy: %v", err)
	}
	found := false
	for _, v := range violations {
		if v.Policy == "closed-set-status-via-constants" && v.Line == 8 {
			found = true
		}
	}
	if !found {
		t.Errorf("expected a closed-set-status-via-constants violation at line 8; got %+v", violations)
	}
}

// TestPolicyClosedSetStatusViaConstants_FiresOnPriorityStructLiteral
// mirrors the `Status:` struct-literal-assignment context pattern for
// `Priority:`.
func TestPolicyClosedSetStatusViaConstants_FiresOnPriorityStructLiteral(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	dir := filepath.Join(root, "internal", "cli", "drift")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	const body = `package drift

type entity struct {
	Priority string
}

var _ = entity{Priority: "high"}
`
	if err := os.WriteFile(filepath.Join(dir, "drift.go"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	violations, err := PolicyClosedSetStatusViaConstants(root)
	if err != nil {
		t.Fatalf("policy: %v", err)
	}
	found := false
	for _, v := range violations {
		if v.Policy == "closed-set-status-via-constants" && v.Line == 7 {
			found = true
		}
	}
	if !found {
		t.Errorf("expected a closed-set-status-via-constants violation at line 7; got %+v", violations)
	}
}
