package check

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/23min/aiwf/internal/areamatch"
	"github.com/23min/aiwf/internal/tree"
)

// CodeAreaUnslotted is the finding code emitted by AreaCoverage for an
// unslotted project directory. Typed per G-0129 so the compiler closes on
// rename / retire across the emit site and tests.
const CodeAreaUnslotted = "area-unslotted"

// CodeAreaCoverageRootMissing is emitted by AreaCoverage for a declared
// coverage root that resolves to no directory (the path does not exist, or
// names a file) — dead config, the coverage analogue of area-dead-glob. A
// silently-skipped dead root gives the operator false confidence that
// coverage is active (M-0185/AC-8).
const CodeAreaCoverageRootMissing = "area-coverage-root-missing"

// CodeAreaCoverageNoPaths is emitted by AreaCoverage when coverage_roots is
// declared but no area declares any `paths:` — the operator opted into
// coverage but the path oracle is dormant, so the check is inert. Surfaced
// rather than silently no-op'd (M-0185/AC-8).
const CodeAreaCoverageNoPaths = "area-coverage-no-paths"

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
// Activation and the opted-in-but-undeliverable diagnostics:
//   - no coverage root declared — fully inert; the knob's presence is the
//     activation signal, so absence means the law does not apply (M-0185/AC-4);
//   - coverage root declared but no area declares any `paths:` — the path
//     oracle is dormant, so instead of silently doing nothing the check emits
//     one area-coverage-no-paths finding: the operator opted in but the
//     prerequisite is missing (M-0185/AC-8);
//   - a declared root that resolves to no directory (does not exist, or names
//     a file) emits area-coverage-root-missing — dead config, the coverage
//     analogue of area-dead-glob; a silently-skipped dead root would give
//     false confidence that coverage is active (M-0185/AC-8).
//
// Enumeration is single-level (one os.ReadDir per declared root) and reads the
// filesystem read-only; a transient/permission IO error yields no findings
// rather than failing (the roadmapCaseCollision precedent). Dot-prefixed
// immediate children (.git / .github / .claude / …) are skipped — tooling /
// VCS artifacts are never projects, so enumerating only the declared roots
// genuinely sidesteps the .git / node_modules / build-output noise a blanket
// walk would pick up, even when the declared root is "." (the repo root). The
// "is this directory claimed?" test routes through the areamatch SSOT
// (M-0180/AC-1), the same matcher the path-axis checks use — no second matcher.
//
// Composed at the CLI layer (internal/cli/check) with the declared areas and
// coverage roots sourced from config — the same seam AreaDeadGlob and
// AreaOverlap use — so the pure check.Run stays config-agnostic. Severity is
// warning, escalated to error under areas.required by ApplyAreaRequiredStrict.
func AreaCoverage(t *tree.Tree, areas []AreaPaths, coverageRoots []string) []Finding {
	if t.Root == "" {
		return nil
	}
	// Fully inert without a declared coverage root: the knob's presence is the
	// activation signal (M-0185/AC-4). Absent, the law does not apply.
	if len(coverageRoots) == 0 {
		return nil
	}
	// Opted into coverage but the path oracle is dormant (a label-only /
	// legacy areas block declares no `paths:`): surface it rather than
	// silently doing nothing — the operator took an affirmative action whose
	// prerequisite is missing (M-0185/AC-8). One finding, not per-root.
	if !AnyAreaHasPaths(areas) {
		return []Finding{{
			Code:     CodeAreaCoverageNoPaths,
			Severity: SeverityWarning,
			Message: "areas.coverage_roots is declared but no area declares paths — coverage is inert; " +
				"add paths to a member (areas.members[].paths) or remove the coverage roots",
			Field: "areas.coverage_roots",
		}}
	}
	var findings []Finding
	for _, root := range coverageRoots {
		rootAbs := filepath.Join(t.Root, root)
		// Resolve the declared root, distinguishing dead config (warn) from a
		// transient/permission IO error (skip), mirroring AreaDeadGlob's
		// os.Stat guard. A non-existent root or one that names a file is dead
		// config — false confidence that coverage is active (M-0185/AC-8); any
		// other stat error is indeterminate and skipped (never fail on IO, the
		// roadmapCaseCollision precedent).
		info, statErr := os.Stat(rootAbs)
		if statErr != nil {
			if errors.Is(statErr, fs.ErrNotExist) {
				findings = append(findings, coverageRootMissingFinding(root))
			}
			continue
		}
		if !info.IsDir() {
			findings = append(findings, coverageRootMissingFinding(root))
			continue
		}
		// Single-level, read-only enumeration of the root's immediate
		// children. Never fail on IO (the roadmapCaseCollision precedent).
		entries, readErr := os.ReadDir(rootAbs)
		if readErr != nil { //coverage:ignore os.ReadDir on a stat-confirmed directory only fails on a transient/permission error; the os.Stat guard above already handles not-exist and not-a-dir, and a dir the test user cannot read is not reproducible (root in the dev container bypasses dir perms)
			continue
		}
		for _, e := range entries {
			// Only directories are candidate projects; files at the root
			// (README, top-level config) are never flagged.
			if !e.IsDir() {
				continue
			}
			// Skip dot-prefixed directories (.git, .github, .claude, …): these
			// are tooling / VCS artifacts, never projects, and `.claude` is
			// materialized by aiwf itself. Flagging them is a never-actionable
			// false positive — this is what makes the "enumerating only
			// declared roots sidesteps .git/build noise" contract hold even
			// when the declared root is "." (the repo root).
			if strings.HasPrefix(e.Name(), ".") {
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

// coverageRootMissingFinding builds the dead-coverage-root finding for a
// declared root that resolves to no directory (M-0185/AC-8).
func coverageRootMissingFinding(root string) Finding {
	return Finding{
		Code:     CodeAreaCoverageRootMissing,
		Severity: SeverityWarning,
		Message: fmt.Sprintf(
			"coverage root %q (areas.coverage_roots) resolves to no directory — fix the path or remove the entry",
			root),
		Path:  root,
		Field: "areas.coverage_roots",
	}
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
