package policies

import (
	"path/filepath"
	"testing"
)

// TestPolicySovereignDispatchers_UnreadableRootErrors pins that an
// unwalkable root surfaces as an error rather than as an empty result.
// The two are the same value to a caller that only reads violations,
// and reporting "no violations" for a tree nobody could read is the
// silent-pass shape this test excludes.
func TestPolicySovereignDispatchers_UnreadableRootErrors(t *testing.T) {
	t.Parallel()

	missing := filepath.Join(t.TempDir(), "no-such-tree")
	vs, err := PolicySovereignDispatchersGuardHumanActor(missing)
	if err == nil {
		t.Fatalf("policy returned nil error for unreadable root %q; got violations: %+v", missing, vs)
	}
	if vs != nil {
		t.Errorf("policy returned violations alongside an error: %+v", vs)
	}
}

// TestSovereignDispatchers_ScopeIsNotOrphaned asserts against the real
// repository that the scanned prefix still contains sovereign
// dispatchers.
//
// PolicySovereignDispatchersGuardHumanActor reports violations, so it
// is silent in two indistinguishable situations: every dispatcher is
// guarded, and no dispatcher was examined. A synthetic firing fixture
// cannot tell them apart either — it builds its own tree, so it passes
// whatever the real layout is. Only an assertion over the live tree
// catches a relocation that moves the dispatchers out from under the
// prefix.
func TestSovereignDispatchers_ScopeIsNotOrphaned(t *testing.T) {
	t.Parallel()

	dispatchers, err := sovereignDispatchers(repoRoot(t))
	if err != nil {
		t.Fatalf("scanning sovereign dispatchers: %v", err)
	}
	if len(dispatchers) == 0 {
		t.Fatalf("no sovereign dispatcher found under %q; the policy's scope is orphaned and it "+
			"cannot fire — repoint sovereignDispatcherPrefix at the layer that now parses --actor",
			sovereignDispatcherPrefix)
	}

	// Each trigger the scan declares should identify something. A
	// trigger matching nothing is dead weight that reads as coverage.
	seen := map[string]bool{}
	for _, d := range dispatchers {
		seen[d.Trigger] = true
	}
	for _, trigger := range []string{
		"declares --force + --reason",
		"declares --audit-only",
		"is the authorize dispatcher",
	} {
		if !seen[trigger] {
			t.Errorf("no dispatcher matched trigger %q; either the trigger is dead or the "+
				"dispatchers it describes moved out of %q", trigger, sovereignDispatcherPrefix)
		}
	}
}
