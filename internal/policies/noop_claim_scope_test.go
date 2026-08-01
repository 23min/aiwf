package policies

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestPolicy_NoOpClaimScope is the live assertion: every same-state NoOp
// this repo constructs has a recorded claim scope.
func TestPolicy_NoOpClaimScope(t *testing.T) {
	t.Parallel()
	runPolicy(t, PolicyNoOpClaimScope)
}

// claimScopeFixture materializes a temp root carrying one internal/verb
// file, so a firing case can present a NoOp site the ledger does not know
// about.
func claimScopeFixture(t *testing.T, src string) string {
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

// TestPolicyNoOpClaimScope_UnrecordedSiteFires is the drift this policy
// exists to catch: a verb that converges without anyone deciding what its
// claim is about. An omission and a deliberate "no guard needed" look
// identical in the code; only the ledger tells them apart.
func TestPolicyNoOpClaimScope_UnrecordedSiteFires(t *testing.T) {
	t.Parallel()
	src := `package verb

type Result struct {
	NoOp        bool
	NoOpMessage string
}

func BrandNewVerb(id string) (*Result, error) {
	return &Result{NoOp: true, NoOpMessage: "already there"}, nil
}
`
	vs, err := PolicyNoOpClaimScope(claimScopeFixture(t, src))
	if err != nil {
		t.Fatalf("policy returned error: %v", err)
	}
	var matched bool
	for _, v := range vs {
		if strings.Contains(v.Detail, "BrandNewVerb converges with no recorded claim scope") {
			matched = true
		}
	}
	if !matched {
		t.Errorf("no violation names the unrecorded site; got:\n%s", joinDetails(vs))
	}
}

// TestPolicyNoOpClaimScope_StaleLedgerEntryFires covers the other
// direction: a ledger entry naming a function that no longer converges is
// a recorded decision about nothing.
func TestPolicyNoOpClaimScope_StaleLedgerEntryFires(t *testing.T) {
	t.Parallel()
	src := `package verb

type Result struct {
	NoOp        bool
	NoOpMessage string
}

func SetPriority(id string) (*Result, error) {
	return &Result{NoOp: true, NoOpMessage: "already there"}, nil
}
`
	vs, err := PolicyNoOpClaimScope(claimScopeFixture(t, src))
	if err != nil {
		t.Fatalf("policy returned error: %v", err)
	}
	var matched bool
	for _, v := range vs {
		if strings.Contains(v.Detail, "is recorded in noOpClaimScopes but no longer converges") {
			matched = true
		}
	}
	if !matched {
		t.Errorf("no violation names a stale ledger entry; got:\n%s", joinDetails(vs))
	}
}

// TestPolicyNoOpClaimScope_EmptyScanFailsClosed pins the fail-closed arm.
// A tree with no verb package must not read as "every site is recorded" —
// it means the policy is scanning nothing.
func TestPolicyNoOpClaimScope_EmptyScanFailsClosed(t *testing.T) {
	t.Parallel()
	vs, err := PolicyNoOpClaimScope(t.TempDir())
	if err != nil {
		t.Fatalf("policy returned error: %v", err)
	}
	if len(vs) != 1 || !strings.Contains(vs[0].Detail, "scanning nothing") {
		t.Errorf("expected a single fail-closed violation, got:\n%s", joinDetails(vs))
	}
}

// TestValidateClaimScopes_RejectsMalformedEntries drives the ledger-shape
// arms. A scope outside the closed set records a decision nothing
// implements; a blank reason records a label with nothing to check it
// against — and for the `none` scope the reason is the entire content,
// since there is no guard to inspect.
func TestValidateClaimScopes_RejectsMalformedEntries(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name       string
		ledger     []claimScope
		wantDetail string
	}{
		{
			name:       "a scope outside the closed set",
			ledger:     []claimScope{{Func: "SomeVerb", Scope: "vibes", Reason: "a reason"}},
			wantDetail: `records scope "vibes"`,
		},
		{
			name:       "a scope with no reason",
			ledger:     []claimScope{{Func: "SomeVerb", Scope: claimScopeTargetEntity, Reason: "  "}},
			wantDetail: "records a claim scope with no reason",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, vs := validateClaimScopes(tc.ledger)
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

// TestValidateClaimScopes_AcceptsAWellFormedLedger is the negative control.
func TestValidateClaimScopes_AcceptsAWellFormedLedger(t *testing.T) {
	t.Parallel()
	recorded, vs := validateClaimScopes([]claimScope{
		{Func: "SomeVerb", Scope: claimScopeTargetEntity, Reason: "the claim reads the target's stored field"},
	})
	if len(vs) != 0 {
		t.Errorf("a well-formed ledger should produce no violations, got:\n%s", joinDetails(vs))
	}
	if recorded["SomeVerb"] != claimScopeTargetEntity {
		t.Errorf("recorded[SomeVerb] = %q, want %q", recorded["SomeVerb"], claimScopeTargetEntity)
	}
}

// TestNoOpClaimScopes_AreWellFormed checks the real ledger's own shape
// without a filesystem, so a malformed entry fails on its own terms rather
// than riding on a verb package being present to scan.
//
// Every one of the four scopes must be populated. A ledger that had
// collapsed to a single scope would satisfy every other assertion here
// while having lost the distinction the scoping exists to make.
func TestNoOpClaimScopes_AreWellFormed(t *testing.T) {
	t.Parallel()
	seen := map[string]bool{}
	populated := map[string]bool{}
	for _, entry := range noOpClaimScopes {
		switch entry.Scope {
		case claimScopeTargetEntity, claimScopeConfigFile, claimScopeSweepSelection, claimScopeNone:
		default:
			t.Errorf("%s: scope %q is outside the closed set", entry.Func, entry.Scope)
		}
		if strings.TrimSpace(entry.Reason) == "" {
			t.Errorf("%s: empty reason", entry.Func)
		}
		if seen[entry.Func] {
			t.Errorf("%s: recorded twice", entry.Func)
		}
		seen[entry.Func] = true
		populated[entry.Scope] = true
	}
	for _, scope := range []string{
		claimScopeTargetEntity, claimScopeConfigFile, claimScopeSweepSelection, claimScopeNone,
	} {
		if !populated[scope] {
			t.Errorf("no ledger entry carries scope %q; the four scopes must stay distinguishable", scope)
		}
	}
}

// TestPolicyNoOpClaimScope_OnlyNoOpTrueLiteralsCount pins what the scan
// treats as a converging site. A Result built positionally, and one that
// explicitly sets NoOp: false, are both Result literals — neither is a
// same-state convergence, and counting either would demand a ledger entry
// for a claim that is not being made.
func TestPolicyNoOpClaimScope_OnlyNoOpTrueLiteralsCount(t *testing.T) {
	t.Parallel()
	src := `package verb

type Result struct {
	NoOp        bool
	NoOpMessage string
}

// The one genuine convergence, so the scan is not empty.
func SetPriority(id string) (*Result, error) {
	return &Result{NoOp: true, NoOpMessage: "already there"}, nil
}

func positionalResult() *Result { return &Result{false, "positional"} }

func explicitlyNotConverging() *Result { return &Result{NoOp: false, NoOpMessage: ""} }
`
	vs, err := PolicyNoOpClaimScope(claimScopeFixture(t, src))
	if err != nil {
		t.Fatalf("policy returned error: %v", err)
	}
	for _, v := range vs {
		if strings.Contains(v.Detail, "positionalResult") || strings.Contains(v.Detail, "explicitlyNotConverging") {
			t.Errorf("a non-converging Result literal was counted as a site:\n%s", v.Detail)
		}
	}
	// SetPriority is in the real ledger, so the fixture's one genuine site
	// is recorded and the scan is non-empty — no fail-closed violation.
	for _, v := range vs {
		if strings.Contains(v.Detail, "scanning nothing") {
			t.Errorf("fail-closed fired despite a real NoOp site:\n%s", v.Detail)
		}
	}
}
