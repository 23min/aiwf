package policies

import (
	"context"
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/check"
	"github.com/23min/aiwf/internal/cli/cliutil"
)

// TestPolicy_ThisRepoTreeHasNoDanglingReference is where CI asserts
// this repo's planning tree resolves (G-0558).
//
// `aiwf check --fast`, `aiwf status`, `aiwf show` and `aiwf render`
// load without the cross-branch ref scan, so they report a total miss
// as the non-blocking unresolved-unverified subcode: they can see an id
// is absent here, not that it is absent everywhere. Only a load that
// consulted every tier can say that, and the surfaces that do are the
// full check and the pre-push hook — neither of which runs in CI
// against this repo's own tree.
//
// So this test builds the full view deliberately and asserts the
// stronger property those surfaces would: no reference in this repo's
// planning tree, structured or in prose, resolves at no tier at all. It
// pays the cross-branch scan once, in a suite already measured in
// minutes, and off every path an operator waits on.
//
// A reference to an id on an unmerged sibling branch is expected and
// passes — that classifies as cross-branch-pending, not unresolved.
func TestPolicy_ThisRepoTreeHasNoDanglingReference(t *testing.T) {
	t.Parallel()
	root := repoRoot(t)

	tr, loadErrs, err := cliutil.LoadTreeWithTrunk(context.Background(), root)
	if err != nil {
		t.Fatalf("loading tree with the cross-branch view: %v", err)
	}
	if !tr.CrossBranchScanned {
		t.Fatal("LoadTreeWithTrunk left CrossBranchScanned false — the assertion below would be about the wrong tier set")
	}

	// Both subcodes assert an id is allocated nowhere — `unresolved`
	// about the reference itself, `unresolved-milestone` about a
	// composite id's parent. `unresolved-ac` is not a dangling
	// reference: it means the parent resolved and lacks that AC.
	var dangling []check.Finding
	for _, f := range check.Run(tr, loadErrs) {
		if f.Code != "refs-resolve" && f.Code != "body-prose-id" {
			continue
		}
		if f.Subcode == "unresolved" || f.Subcode == "unresolved-milestone" {
			dangling = append(dangling, f)
		}
	}
	if len(dangling) > 0 {
		var lines []string
		for _, f := range dangling {
			lines = append(lines, "  "+f.Code+"/"+f.Subcode+": "+f.Message+" ("+f.Path+")")
		}
		t.Errorf("%d reference(s) resolve at no tier — not the working tree, not trunk, not any branch:\n%s",
			len(dangling), strings.Join(lines, "\n"))
	}
}
