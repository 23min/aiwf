package policies

import (
	"path/filepath"
	"strings"
	"testing"
)

// TestPolicyApplyCallersAcquireLock_ScopeIsNotOrphaned asserts against
// the real repository that the scanned prefix still holds dispatchers
// reaching verb.Apply.
//
// The policy reports violations, so it is silent both when every
// dispatcher takes the lock and when no dispatcher was examined. A
// synthetic fixture cannot separate the two — it builds its own tree
// and passes whatever the real layout is — so only a live-tree
// assertion catches a relocation that empties the prefix.
func TestPolicyApplyCallersAcquireLock_ScopeIsNotOrphaned(t *testing.T) {
	t.Parallel()

	dispatchers, err := applyReachingDispatchers(repoRoot(t))
	if err != nil {
		t.Fatalf("scanning dispatchers: %v", err)
	}
	if len(dispatchers) == 0 {
		t.Fatalf("no dispatcher under %q reaches verb.Apply; the policy's scope is orphaned and it "+
			"cannot fire — repoint lockDispatcherPrefix at the layer that now takes the repo lock",
			lockDispatcherPrefix)
	}

	// Both dispatcher spellings must be represented. A verb's entry
	// point is Run and a subverb's is run<Sub>; a scan that reaches
	// only one spelling covers half the population while reporting
	// success over all of it.
	var sawRun, sawSubverb bool
	for _, d := range dispatchers {
		switch {
		case d.Func == "Run":
			sawRun = true
		case strings.HasPrefix(d.Func, "run"):
			sawSubverb = true
		}
	}
	if !sawRun {
		t.Errorf("no dispatcher named Run reaches verb.Apply; the verb entry points are unexamined")
	}
	if !sawSubverb {
		t.Errorf("no dispatcher named run<Sub> reaches verb.Apply; the subverb entry points are unexamined")
	}
}

// TestPolicyApplyCallersAcquireLock_UnreadableRootErrors pins that an
// unwalkable root surfaces as an error rather than as an empty result.
// To a caller that only reads violations the two are the same value,
// and reporting "no violations" for a tree nobody could read is the
// silent pass this policy is being held to.
func TestPolicyApplyCallersAcquireLock_UnreadableRootErrors(t *testing.T) {
	t.Parallel()

	missing := filepath.Join(t.TempDir(), "no-such-tree")
	vs, err := PolicyApplyCallersAcquireLock(missing)
	if err == nil {
		t.Fatalf("policy returned nil error for unreadable root %q; got violations: %+v", missing, vs)
	}
	if vs != nil {
		t.Errorf("policy returned violations alongside an error: %+v", vs)
	}
}

// TestPolicyApplyCallersAcquireLock_HelperLayerExempt pins that the
// shared helper layer is excluded.
//
// cliutil's finish helpers reach verb.Apply on behalf of a dispatcher
// that has already taken the lock. Without the exemption the policy
// reports them, and the only way to silence it would be to take the
// lock twice — asserting the opposite of what the lock is for.
func TestPolicyApplyCallersAcquireLock_HelperLayerExempt(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	// A helper that reaches Apply without the lock: exempt by path.
	mustWrite(t, filepath.Join(root, "internal", "cli", "cliutil", "apply.go"),
		"package cliutil\n\nfunc FinishVerbOutcome() { verb.Apply() }\n")
	// The same shape one directory over: not exempt.
	mustWrite(t, filepath.Join(root, "internal", "cli", "foo", "foo.go"),
		"package foo\n\nfunc Run() { verb.Apply() }\n")

	vs, err := PolicyApplyCallersAcquireLock(root)
	if err != nil {
		t.Fatalf("policy returned error: %v", err)
	}
	if len(vs) != 1 {
		t.Fatalf("want exactly 1 violation (the non-helper dispatcher), got %d: %+v", len(vs), vs)
	}
	if !strings.HasPrefix(vs[0].File, "internal/cli/foo/") {
		t.Errorf("violation names %q; want the non-helper dispatcher — the helper layer must stay exempt", vs[0].File)
	}
}
