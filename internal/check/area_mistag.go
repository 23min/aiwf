package check

import (
	"context"
	"fmt"
	"os/exec"
	"sort"
	"strings"

	"github.com/23min/aiwf/internal/areamatch"
	"github.com/23min/aiwf/internal/entity"
	"github.com/23min/aiwf/internal/gitops"
	"github.com/23min/aiwf/internal/tree"
)

// CodeAreaMistag is the finding code emitted by AreaMistag. Typed per G-0129
// so the compiler closes on rename / retire across the emit site and tests.
const CodeAreaMistag = "area-mistag"

// AreaMistag (warning) reports an entity whose linked commits' work landed in a
// DIFFERENT area's path territory than the one the entity is tagged to. It is
// the path-vs-tag consistency check: with `paths:` (M-0179) declared and the
// entity ↔ commit linkage aiwf records via the `aiwf-entity:` trailer
// (gathered by GatherEntityPaths), the touched-files-vs-glob comparison catches
// "filed against the wrong area, flew under the radar" — the failure label-only
// areas are blind to.
//
// The entity's effective area comes from t.ResolvedArea (so a milestone is
// judged against its parent epic's area), matched against the declared globs
// via the areamatch SSOT. Several guards make the rule inert where it can't
// speak:
//   - no area declares `paths:` → nothing to check against;
//   - the entity is untagged, or carries the reserved `global` sentinel
//     (inherently cross-cutting, ADR-0021) → skipped;
//   - the entity's own area declares no `paths:` → can't locate "inside";
//   - the entity has no linked commits / no touched paths → nothing to judge;
//   - archived entities are out of scope (ADR-0004 §"check shape rules").
//
// Severity is warning and NEVER escalates — unlike dead-glob / overlap, it is
// deliberately absent from ApplyAreaRequiredStrict, because legitimate
// cross-cutting work exists and the acknowledge path (M-0181/AC-6) is the
// sanctioned escape valve, not a strictness bump. Composed at the CLI layer
// with the gathered paths and declared areas, like the other path-axis rules.
func AreaMistag(t *tree.Tree, areas []AreaPaths, touchedByEntity map[string]map[string]bool) []Finding {
	globsByArea := map[string][]string{}
	for _, a := range areas {
		if len(a.Paths) > 0 {
			globsByArea[a.Name] = a.Paths
		}
	}
	if len(globsByArea) == 0 {
		return nil // no area declares paths → nothing to check against
	}
	var findings []Finding
	for _, e := range t.Entities {
		if entity.IsArchivedPath(e.Path) {
			continue
		}
		area := t.ResolvedArea(e)
		if area == "" || area == entity.AreaGlobal {
			continue // untagged, or the cross-cutting sentinel
		}
		if len(globsByArea[area]) == 0 {
			continue // the entity's own area declares no paths → can't check
		}
		touched := touchedByEntity[entity.Canonicalize(e.ID)]
		if len(touched) == 0 {
			continue // no linked commits / no touched paths
		}
		// Classify the touched paths against the declared areas. Paths matching
		// no declared glob (planning files, docs, unclaimed code) never
		// participate — the "area-claimed space" guard that keeps the rule from
		// firing on every entity's own planning commit. A path in the entity's
		// OWN area marks the work cross-cutting; foreign-area paths are
		// collected for the message.
		ownGlobs := globsByArea[area]
		foreign := map[string]bool{}
		insideOwn := false
		for p := range touched {
			if matchesAnyGlob(p, ownGlobs) {
				insideOwn = true
				continue // own-area path: can't also be foreign; next path
			}
			for name, globs := range globsByArea {
				if name == area {
					continue
				}
				if matchesAnyGlob(p, globs) {
					foreign[name] = true
				}
			}
		}
		// Cross-cutting is tolerated (M-0181/AC-3): if any work landed in the
		// entity's own area, do not fire even when other work landed elsewhere.
		// Otherwise fire only when some area-claimed work landed in a foreign
		// area (M-0181/AC-2).
		if insideOwn || len(foreign) == 0 {
			continue
		}
		foreignNames := make([]string, 0, len(foreign))
		for n := range foreign {
			foreignNames = append(foreignNames, n)
		}
		sort.Strings(foreignNames)
		findings = append(findings, Finding{
			Code:     CodeAreaMistag,
			Severity: SeverityWarning,
			Message: fmt.Sprintf(
				"%s resolves to area %q but its area-claimed work landed entirely in the %s area(s), none in its own — mis-tagged, or cross-cutting work to acknowledge",
				e.ID, area, strings.Join(foreignNames, ", ")),
			Path:     e.Path,
			EntityID: e.ID,
			Field:    "area",
		})
	}
	return findings
}

// AnyAreaHasPaths reports whether at least one declared area carries a `paths:`
// glob. The CLI seam uses it to gate the (full-history, multi-second) gather +
// AreaMistag entirely: with no paths-carrying area, mistag can produce nothing
// (AreaMistag early-returns nil), so walking history is pure waste. Gating here
// keeps `aiwf check` from eating a git-log walk for a feature the consumer
// hasn't opted into — the common default.
func AnyAreaHasPaths(areas []AreaPaths) bool {
	for _, a := range areas {
		if len(a.Paths) > 0 {
			return true
		}
	}
	return false
}

// matchesAnyGlob reports whether path matches at least one of globs via the
// areamatch SSOT. A glob that errors during matching is treated as no-match:
// malformed globs are rejected at config load (Tier 1, areamatch.Validate), so
// an error here is an indeterminate match, skipped (the AreaDeadGlob precedent).
func matchesAnyGlob(path string, globs []string) bool {
	for _, g := range globs {
		ok, err := areamatch.Match(g, path)
		if err != nil {
			continue
		}
		if ok {
			return true
		}
	}
	return false
}

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
// canonicalized at ingest so a narrow-legacy trailer (`aiwf-entity: G-123`, the
// pre-ADR-0008 gap width) and a canonical-width lookup (`G-0123`) agree.
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
	// A bespoke HEAD-scoped pass, deliberately NOT gitops.BulkRevwalk: that
	// walker is --all-scoped, but mistag must judge work landed on THIS branch
	// (the same DAG-scoping WalkAcknowledgedSHAs uses), and it omits
	// quotePath=false. Only the transport framing is bespoke — the semantic
	// SSOTs (gitops.ParseTrailers, entity.Canonicalize/CompositeRoot) are reused.
	//
	// `-c core.quotePath=false` so non-ASCII paths (projects/café/…) are
	// emitted literally rather than octal-escaped-and-quoted — a quoted path
	// would never match an area glob, silently hiding the entity's work (the
	// NUL-termination convention the gitops name-walkers use defeats the same
	// quoting; the custom \x01/\x03 framing precludes bare -z here). Paths with
	// embedded newlines remain git-quoted regardless and are out of scope.
	cmd := exec.CommandContext(ctx, "git", "-c", "core.quotePath=false",
		"log", "--no-renames", "--name-only",
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
