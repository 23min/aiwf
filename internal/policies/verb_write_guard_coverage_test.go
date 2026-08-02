package policies

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestPolicy_VerbWriteGuardCoverage is the live assertion: this repo's
// own verb entry points all carry a recorded guard treatment.
func TestPolicy_VerbWriteGuardCoverage(t *testing.T) {
	t.Parallel()
	runPolicy(t, PolicyVerbWriteGuardCoverage)
}

// writeGuardFixture materializes a temp root carrying one internal/verb
// file declaring the given source, so a firing case can present an entry
// point the ledger does not know about.
func writeGuardFixture(t *testing.T, src string) string {
	t.Helper()
	root := t.TempDir()
	dir := filepath.Join(root, "internal", "verb")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "fixture.go"), []byte(src), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	return root
}

// TestPolicyVerbWriteGuardCoverage_UnrecordedEntryPointFires is the
// drift this policy exists to catch: a verb added without anyone
// deciding how the guard treats it.
func TestPolicyVerbWriteGuardCoverage_UnrecordedEntryPointFires(t *testing.T) {
	t.Parallel()
	src := `package verb

type Result struct{}

func BrandNewVerb(id string) (*Result, error) { return nil, nil }
`
	vs, err := PolicyVerbWriteGuardCoverage(writeGuardFixture(t, src))
	if err != nil {
		t.Fatalf("policy returned error: %v", err)
	}
	var matched bool
	for _, v := range vs {
		if strings.Contains(v.Detail, "BrandNewVerb has no recorded decision") {
			matched = true
		}
	}
	if !matched {
		t.Errorf("no violation names the unrecorded verb; got:\n%s", joinDetails(vs))
	}
}

// TestPolicyVerbWriteGuardCoverage_StaleLedgerEntryFires covers the
// other direction. A fixture tree declaring exactly one known entry
// point leaves every other ledger name unmatched, which is the shape a
// removed or renamed verb produces.
func TestPolicyVerbWriteGuardCoverage_StaleLedgerEntryFires(t *testing.T) {
	t.Parallel()
	src := `package verb

type Result struct{}

func SetPriority(id string) (*Result, error) { return nil, nil }
`
	vs, err := PolicyVerbWriteGuardCoverage(writeGuardFixture(t, src))
	if err != nil {
		t.Fatalf("policy returned error: %v", err)
	}
	var matched bool
	for _, v := range vs {
		if strings.Contains(v.Detail, "is recorded in verbGuardTreatments but is no longer an exported") {
			matched = true
		}
	}
	if !matched {
		t.Errorf("no violation names a stale ledger entry; got:\n%s", joinDetails(vs))
	}
}

// TestPolicyVerbWriteGuardCoverage_EmptyScanFailsClosed pins the
// fail-closed arm. A tree with no verb package must not read as "every
// route is recorded" — it means the policy is scanning nothing.
func TestPolicyVerbWriteGuardCoverage_EmptyScanFailsClosed(t *testing.T) {
	t.Parallel()
	vs, err := PolicyVerbWriteGuardCoverage(t.TempDir())
	if err != nil {
		t.Fatalf("policy returned error: %v", err)
	}
	if len(vs) != 1 || !strings.Contains(vs[0].Detail, "scanning nothing") {
		t.Errorf("expected a single fail-closed violation, got:\n%s", joinDetails(vs))
	}
}

// TestValidateGuardTreatments_RejectsMalformedEntries drives the
// ledger-shape arms with input the real ledger will never carry. An
// unknown treatment and a blank reason each defeat the ledger's purpose
// in their own way: one records a decision nothing in the guard
// implements, the other records a label with nothing to check it against.
func TestValidateGuardTreatments_RejectsMalformedEntries(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name       string
		ledger     []guardTreatment
		wantDetail string
	}{
		{
			name:       "a treatment outside the closed set",
			ledger:     []guardTreatment{{Func: "SomeVerb", Treatment: "mostly-fine", Reason: "a reason"}},
			wantDetail: `records treatment "mostly-fine"`,
		},
		{
			name:       "a treatment with no reason",
			ledger:     []guardTreatment{{Func: "SomeVerb", Treatment: guardTreatmentGuarded, Reason: "   "}},
			wantDetail: "records a treatment with no reason",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, vs := validateGuardTreatments(tc.ledger)
			var matched bool
			for _, v := range vs {
				if strings.Contains(v.Detail, tc.wantDetail) {
					matched = true
				}
			}
			if !matched {
				t.Errorf("no violation contains %q; got:\n%s", tc.wantDetail, joinDetails(vs))
			}
		})
	}
}

// TestValidateGuardTreatments_AcceptsAWellFormedLedger is the negative
// control for the cases above.
func TestValidateGuardTreatments_AcceptsAWellFormedLedger(t *testing.T) {
	t.Parallel()
	recorded, vs := validateGuardTreatments([]guardTreatment{
		{Func: "SomeVerb", Treatment: guardTreatmentGuarded, Reason: "writes entity bytes through Apply"},
	})
	if len(vs) != 0 {
		t.Errorf("a well-formed ledger should produce no violations, got:\n%s", joinDetails(vs))
	}
	if recorded["SomeVerb"] != guardTreatmentGuarded {
		t.Errorf("recorded[SomeVerb] = %q, want %q", recorded["SomeVerb"], guardTreatmentGuarded)
	}
}

// TestVerbGuardTreatments_AreWellFormed checks the ledger's own shape
// without a filesystem: every entry names one of the closed set of
// treatments, carries a reason, and appears once.
//
// The policy reports these too, but only while a verb package is present
// to scan. Asserting them directly means a malformed entry fails on its
// own terms rather than riding on that precondition.
func TestVerbGuardTreatments_AreWellFormed(t *testing.T) {
	t.Parallel()
	seen := map[string]bool{}
	for _, entry := range verbGuardTreatments {
		switch entry.Treatment {
		case guardTreatmentGuarded, guardTreatmentAdopts, guardTreatmentRecordOnly:
		default:
			t.Errorf("%s: treatment %q is outside the closed set", entry.Func, entry.Treatment)
		}
		if strings.TrimSpace(entry.Reason) == "" {
			t.Errorf("%s: empty reason", entry.Func)
		}
		if seen[entry.Func] {
			t.Errorf("%s: recorded twice", entry.Func)
		}
		seen[entry.Func] = true
	}
	if len(routesOutsideTheEntryPointScan) == 0 {
		t.Error("routesOutsideTheEntryPointScan is empty; the routes no scan can derive are the ones most worth writing down")
	}
	for _, r := range routesOutsideTheEntryPointScan {
		if strings.TrimSpace(r.Reason) == "" {
			t.Errorf("%s: empty reason", r.Route)
		}
	}
}

// TestCheckAdoptsFlagOwnership_FiresOutsideTheOwningFile pins the guard's
// one bypass lever. A verb that starts setting AdoptsWorkingCopy moves
// itself into the exempt class, where the uncommitted-change guard stops
// comparing what it writes — a behaviour change the treatment ledger
// cannot detect, because the ledger records names.
func TestCheckAdoptsFlagOwnership_FiresOutsideTheOwningFile(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	dir := filepath.Join(root, "internal", "verb")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	src := `package verb

type FileOp struct{ AdoptsWorkingCopy bool }

func newlyExemptVerb() FileOp { return FileOp{AdoptsWorkingCopy: true} }
`
	if err := os.WriteFile(filepath.Join(dir, "greedy.go"), []byte(src), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	files, err := WalkGoFiles(root, true)
	if err != nil {
		t.Fatalf("walk: %v", err)
	}
	vs := checkAdoptsFlagOwnership(files)
	var matched bool
	for _, v := range vs {
		if strings.Contains(v.Detail, "sets AdoptsWorkingCopy: true outside") {
			matched = true
		}
	}
	if !matched {
		t.Errorf("no violation names the unauthorized setter; got:\n%s", joinDetails(vs))
	}
}

// TestCheckAdoptsFlagOwnership_AllowsTheOwningFile is the negative
// control: the real setter must not trip its own rule.
func TestCheckAdoptsFlagOwnership_AllowsTheOwningFile(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	dir := filepath.Join(root, "internal", "verb")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	src := `package verb

type FileOp struct{ AdoptsWorkingCopy bool }

func blessMode() FileOp { return FileOp{AdoptsWorkingCopy: true} }
`
	if err := os.WriteFile(filepath.Join(dir, "editbody.go"), []byte(src), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	files, err := WalkGoFiles(root, true)
	if err != nil {
		t.Fatalf("walk: %v", err)
	}
	if vs := checkAdoptsFlagOwnership(files); len(vs) != 0 {
		t.Errorf("the owning file must not trip its own rule, got:\n%s", joinDetails(vs))
	}
}
