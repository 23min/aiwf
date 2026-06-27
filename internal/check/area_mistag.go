package check

import (
	"context"
	"os/exec"
	"strings"

	"github.com/23min/aiwf/internal/entity"
	"github.com/23min/aiwf/internal/gitops"
)

// GatherEntityPaths walks HEAD-reachable history and returns, per canonical
// root entity id, the set of repo-relative paths that entity's commits
// touched — gathered via the `aiwf-entity:` commit trailer. A commit
// contributes its full touched-file set to every entity it is trailered to;
// composite acceptance-criterion trailers (M-NNNN/AC-N) roll up to the parent
// milestone so an AC's code lands in its milestone's set.
//
// The gather is deliberately UNFILTERED: it unions every touched path,
// planning files and project code alike. The area-glob filtering that decides
// mistag lives downstream in AreaMistag (M-0181/AC-2); keeping gather and
// filter separate makes each a single testable unit.
//
// Returns nil for a non-git root or empty history. Trailer ids are
// canonicalized at ingest so a narrow-legacy trailer (`aiwf-entity: G-1`) and
// a canonical-width lookup (`G-0001`) agree.
func GatherEntityPaths(ctx context.Context, root string) map[string]map[string]bool {
	if root == "" || !hasGitCommits(ctx, root) {
		return nil
	}
	// One `git log --name-only` pass. Control chars (SOH \x01 starting each
	// commit, ETX \x03 closing its trailer block) delimit the structured
	// fields so the newlines in git's name-only file list can't confuse the
	// parse — neither char ever occurs in an id, path, or trailer value. The
	// full trailer block is parsed via gitops.ParseTrailers (the codebase
	// standard, robust against folding) rather than git's key= filtering.
	cmd := exec.CommandContext(ctx, "git", "log", "--no-renames", "--name-only",
		"--pretty=format:\x01%(trailers:unfold=true)\x03", "HEAD")
	cmd.Dir = root
	out, err := cmd.Output()
	if err != nil {
		return nil //coverage:ignore git log on a hasGitCommits-verified repo fails only on exotic IO; not deterministically reproducible in a test.
	}
	result := map[string]map[string]bool{}
	for _, chunk := range strings.Split(string(out), "\x01") {
		etx := strings.Index(chunk, "\x03")
		if etx < 0 {
			continue // the leading split fragment before the first commit
		}
		var roots []string
		for _, tr := range gitops.ParseTrailers(chunk[:etx]) {
			if tr.Key != gitops.TrailerEntity {
				continue
			}
			v := strings.TrimSpace(tr.Value)
			if v == "" {
				continue
			}
			// Roll a composite AC trailer up to its milestone, then
			// canonicalize, so a narrow-legacy trailer and a canonical-width
			// lookup agree (the WalkAcknowledgedSHAEntities ingest convention).
			roots = append(roots, entity.Canonicalize(entity.CompositeRoot(v)))
		}
		if len(roots) == 0 {
			continue // untrailered commit contributes no entity key
		}
		for _, line := range strings.Split(chunk[etx+1:], "\n") {
			p := strings.TrimSpace(line)
			if p == "" {
				continue
			}
			for _, r := range roots {
				if result[r] == nil {
					result[r] = map[string]bool{}
				}
				result[r][p] = true
			}
		}
	}
	if len(result) == 0 {
		return nil
	}
	return result
}
