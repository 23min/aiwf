package check

import (
	"context"

	"github.com/23min/aiwf/internal/check"
	"github.com/23min/aiwf/internal/cli/cliutil"
	"github.com/23min/aiwf/internal/tree"
)

// RunInstructionFenceCheck runs the instruction-file fence over the range
// the provenance audit judges, resolved by ResolveUntrailedRange: --since
// when given, else the upstream. Where no range resolves, the provenance
// audit reports the scope advisory once and the fence reports nothing.
// Commits already on t's trunk, when it names a remote-tracking ref, are
// not judged, and a commit's entity trailer resolves against t.
func RunInstructionFenceCheck(ctx context.Context, root, since string, t *tree.Tree) ([]check.Finding, error) {
	if !cliutil.HasCommits(ctx, root) {
		return nil, nil
	}
	rangeArg, advisory, err := ResolveUntrailedRange(ctx, root, since)
	if err != nil || advisory != nil {
		return nil, err
	}
	commits, err := check.WalkInstructionCommits(ctx, root, rangeArg, trunkRefOf(t))
	if err != nil { //coverage:ignore the range was just verified by ResolveUntrailedRange and the trunk ref resolved by the tree load, so git log over them fails only on a repository broken mid-run
		return nil, err
	}
	return check.RunInstructionFence(commits, func(id string) bool { return check.EntityResolves(t, id) }), nil
}
