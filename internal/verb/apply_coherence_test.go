package verb_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/gitops"
	"github.com/23min/aiwf/internal/verb"
)

// TestApply_RefusesAnIncoherentTrailerSet covers M-0291/AC-1 at the seam
// itself.
//
// The integration table in internal/cli/integration drives the real
// binary, which is what proves the refusal reaches an operator — but a
// subprocess contributes nothing to the coverage profile, so the guard's
// own lines read as untested. This is the direct call: same property,
// one layer down, and the arm that returns is exercised in-process.
func TestApply_RefusesAnIncoherentTrailerSet(t *testing.T) {
	t.Parallel()
	r := newApplyTestRepo(t)

	// force + a non-human actor: sovereign, and the actor is not human.
	p := unrelatedPlan()
	p.Trailers = []gitops.Trailer{
		{Key: gitops.TrailerVerb, Value: "promote"},
		{Key: gitops.TrailerEntity, Value: "E-0099"},
		{Key: gitops.TrailerActor, Value: "ai/claude"},
		{Key: gitops.TrailerPrincipal, Value: "human/peter"},
		{Key: gitops.TrailerForce, Value: "escalation"},
	}

	before := headSHA(t, r.root)
	sha, err := verb.Apply(r.ctx, r.root, p)

	if err == nil {
		t.Fatalf("Apply accepted an incoherent trailer set (sha %s); want refusal", sha)
	}
	if sha != "" {
		t.Errorf("Apply returned sha %q alongside an error; the contract is sha-empty-on-error", sha)
	}
	if _, rule := verb.AsCoherenceError(err); rule == "" {
		t.Errorf("Apply refused with %v, which is not a coherence error; the refusal came from some other guard", err)
	}
	if after := headSHA(t, r.root); after != before {
		t.Errorf("HEAD moved %s -> %s; the guard must refuse before committing", before[:8], after[:8])
	}
	// The plan's write must not have landed either. A guard that refuses
	// after touching the worktree leaves the operator to clean up.
	if _, statErr := os.Stat(filepath.Join(r.root, "work/epics/E-0099-unrelated/epic.md")); !os.IsNotExist(statErr) {
		t.Errorf("the plan's write landed despite the refusal; the guard must run before any filesystem work")
	}
}

// TestApply_AcceptsACoherentTrailerSet pins the other arm. Without it a
// guard that refused everything would pass the test above while breaking
// every verb in the kernel.
func TestApply_AcceptsACoherentTrailerSet(t *testing.T) {
	t.Parallel()
	r := newApplyTestRepo(t)

	p := unrelatedPlan()
	p.Trailers = []gitops.Trailer{
		{Key: gitops.TrailerVerb, Value: "promote"},
		{Key: gitops.TrailerEntity, Value: "E-0099"},
		{Key: gitops.TrailerActor, Value: "human/peter"},
	}

	sha, err := verb.Apply(r.ctx, r.root, p)
	if err != nil {
		t.Fatalf("Apply refused a coherent trailer set: %v", err)
	}
	if sha == "" {
		t.Error("Apply returned an empty sha with no error")
	}
	body, logErr := runGit(r.ctx, r.root, "log", "-1", "--format=%B")
	if logErr != nil {
		t.Fatalf("git log: %v", logErr)
	}
	if !strings.Contains(body, "aiwf-actor: human/peter") {
		t.Errorf("commit does not carry the plan's trailers:\n%s", body)
	}
}
