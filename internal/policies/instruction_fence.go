package policies

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/23min/aiwf/internal/check"
	"github.com/23min/aiwf/internal/config"
	"github.com/23min/aiwf/internal/tree"
)

// PolicyInstructionFence runs aiwf check's instruction-file fence
// (ADR-0053) over this repository's commits between AIWF_COVERAGE_BASE and
// HEAD.
//
// The pre-push hook runs the same rule through the installed aiwf, which
// can predate it. This policy is compiled from the tree under test, so a
// stale binary cannot skip it — the reason PolicyCoauthorTrailerBan backs
// up the commit-msg hook. Like that policy it no-ops when no base is set,
// and it reads the repository's guidance.fence_instruction_commits, so the
// two layers cannot disagree about whether the fence is on.
func PolicyInstructionFence(root string) ([]Violation, error) {
	return instructionFenceViolations(root, os.Getenv("AIWF_COVERAGE_BASE"))
}

// instructionFenceViolations is the IO core: it walks baseRef..HEAD with the
// kernel walker and maps the kernel rule's findings to violations.
func instructionFenceViolations(root, baseRef string) ([]Violation, error) {
	baseRef = strings.TrimSpace(baseRef)
	if baseRef == "" || baseRef == zeroSHA {
		return nil, nil
	}
	// A missing or unreadable aiwf.yaml leaves the fence at its default,
	// on: a config fault is reported by the verbs that read config, and it
	// must not switch a gate off.
	var cfg *config.Config
	if loaded, err := config.Load(root); err == nil {
		cfg = loaded
	}
	if !cfg.FenceInstructionCommits() {
		return nil, nil
	}
	ctx := context.Background()
	// No trunk is excluded: the bases CI and the Makefile compute — a
	// push's before-SHA, a pull request's merge-base with main — already
	// leave trunk commits outside the range. A base set by hand to a
	// commit older than the branch's last trunk merge judges those too.
	commits, err := check.WalkInstructionCommits(ctx, root, baseRef+"..HEAD", "")
	if err != nil {
		return nil, err
	}
	t, _, err := tree.Load(ctx, root)
	if err != nil {
		return nil, fmt.Errorf("loading entity tree at %s: %w", root, err)
	}
	findings := check.RunInstructionFence(commits, func(id string) bool { return check.EntityResolves(t, id) })
	var out []Violation
	for i := range findings {
		out = append(out, Violation{
			Policy: "instruction-fence",
			File:   findings[i].Path,
			Detail: findings[i].Code + ": " + findings[i].Message,
		})
	}
	return out, nil
}
