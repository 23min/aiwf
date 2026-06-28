package check

import (
	"fmt"
	"os"
	"path"
	"path/filepath"

	"github.com/23min/aiwf/internal/areamatch"
	"github.com/23min/aiwf/internal/tree"
)

// CodeAreaUnslotted is the finding code emitted by AreaCoverage. Typed per
// G-0129 so the compiler closes on rename / retire across the emit site and
// tests.
const CodeAreaUnslotted = "area-unslotted"

// AreaCoverage (warning) reports any immediate child directory of an
// operator-declared coverage root that is claimed by no declared area's
// `paths:` glob — an unslotted project (E-0044, M-0185). It is the *covering*
// law of the area↔directory partition: where AreaDeadGlob asserts no area
// column is empty and AreaOverlap asserts no directory row is claimed twice,
// coverage asserts every in-scope directory is claimed by some area. The
// monorepo-specific catch for a newly-added project that nobody slotted.
//
// The universe is scoped and opt-in: the operator names the coverage root(s)
// (aiwf.yaml: areas.coverage_roots), and only the immediate child directories
// of those roots must tile. Directories outside any declared root are
// unscoped and never flagged, so the single-project / semantic-section repo —
// which declares no coverage root — is never flagged wholesale. The model is
// deliberately not total-partition: the filesystem remainder (README, docs/,
// top-level config) is legitimately uncovered.
//
// Two guards make it inert (M-0185/AC-4):
//   - no coverage root declared — the knob's presence is the activation
//     signal, so absence means the law does not apply;
//   - no area declares any `paths:` — the path axis is dormant (a label-only
//     / legacy string-form areas block), so coverage has no oracle to test
//     against and stays silent rather than flagging every child.
//
// Enumeration is single-level (one os.ReadDir per declared root) and reads the
// filesystem read-only; a missing or unreadable root yields no findings rather
// than failing (the roadmapCaseCollision precedent). Enumerating only the
// declared roots — never a blanket walk — sidesteps the .git / node_modules /
// build-output noise a recursive walk would pick up. The "is this directory
// claimed?" test routes through the areamatch SSOT (M-0180/AC-1), the same
// matcher the path-axis checks use — no second matcher.
//
// Composed at the CLI layer (internal/cli/check) with the declared areas and
// coverage roots sourced from config — the same seam AreaDeadGlob and
// AreaOverlap use — so the pure check.Run stays config-agnostic. Severity is
// warning, escalated to error under areas.required by ApplyAreaRequiredStrict.
func AreaCoverage(t *tree.Tree, areas []AreaPaths, coverageRoots []string) []Finding {
	if t.Root == "" {
		return nil
	}
	// Inert without a declared coverage root: the knob's presence is the
	// activation signal (M-0185/AC-4). Absent, the law does not apply.
	if len(coverageRoots) == 0 {
		return nil
	}
	// Inert without any declared paths: the path axis is dormant for a
	// label-only / legacy areas block, so coverage has no oracle and stays
	// silent rather than flagging every child (M-0185/AC-4).
	if !AnyAreaHasPaths(areas) {
		return nil
	}
	var findings []Finding
	for _, root := range coverageRoots {
		// Single-level, read-only enumeration of the declared root's
		// immediate children. Never fail on IO: a missing or unreadable root
		// yields no findings (the roadmapCaseCollision precedent).
		entries, err := os.ReadDir(filepath.Join(t.Root, root))
		if err != nil {
			continue
		}
		for _, e := range entries {
			// Only directories are candidate projects; files at the root
			// (README, top-level config) are never flagged.
			if !e.IsDir() {
				continue
			}
			// The child's repo-relative, '/'-separated path — what the area
			// globs match against. path.Join (not filepath.Join) keeps the
			// separator '/' so the glob match is platform-independent.
			child := path.Join(root, e.Name())
			claimed, matchErr := claimedByAnyArea(areas, child)
			if matchErr != nil {
				// A malformed glob cannot reach here — areamatch.Validate
				// rejects it at config load (Tier 1) — so a match error is an
				// indeterminate artifact; skip the child rather than risk a
				// false unslotted finding (the never-fail-on-IO spirit).
				continue
			}
			if claimed {
				continue
			}
			findings = append(findings, Finding{
				Code:     CodeAreaUnslotted,
				Severity: SeverityWarning,
				Message: fmt.Sprintf(
					"directory %q under coverage root %q is claimed by no area's paths glob — slot it into an area (areas.members[].paths) or remove the coverage root",
					child, root),
				Path:  child,
				Field: "areas.coverage_roots",
			})
		}
	}
	return findings
}

// claimedByAnyArea reports whether the repo-relative directory path is matched
// by at least one declared area's path glob — "does some area claim this
// directory?". It routes through the areamatch SSOT (M-0180/AC-1),
// short-circuiting on the first match. A whole-project glob (`projects/app-a/**`)
// matches the bare project directory (`projects/app-a`), so the common
// area-paths shape claims its own root directory. Returns the first match
// error encountered (a malformed glob), which the caller treats as
// indeterminate.
func claimedByAnyArea(areas []AreaPaths, dir string) (bool, error) {
	for _, a := range areas {
		for _, glob := range a.Paths {
			ok, err := areamatch.Match(glob, dir)
			if err != nil {
				return false, err
			}
			if ok {
				return true, nil
			}
		}
	}
	return false, nil
}
