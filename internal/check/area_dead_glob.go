package check

import (
	"fmt"
	"os"

	"github.com/23min/aiwf/internal/areamatch"
	"github.com/23min/aiwf/internal/tree"
)

// CodeAreaDeadGlob is the finding code emitted by AreaDeadGlob. Typed per
// G-0129 so the compiler closes on rename / retire across the emit site and
// tests.
const CodeAreaDeadGlob = "area-dead-glob"

// AreaPaths is the config-agnostic projection of a declared area's name and
// its path globs. The path-axis checks take it from the CLI seam so the
// check package stays config-agnostic (the M-0171/AC-4 boundary): the pure
// check.Run never reads aiwf.yaml.
type AreaPaths struct {
	Name  string
	Paths []string
}

// AreaDeadGlob (warning) reports any declared area path glob that matches no
// real file or directory under the repo root — dead config: a renamed,
// deleted, or typo'd project path leaving that area's path oracle empty. The
// check is per-glob: each declared glob must locate at least one path, and
// each dead glob fires its own finding naming the area and the glob.
//
// Reads the filesystem read-only through the area-glob matcher (areamatch,
// the SSOT introduced in M-0180/AC-1) and never fails on IO: an empty or
// unreadable root, or a glob that errors during the walk, is silently
// skipped (the roadmapCaseCollision precedent). Malformed globs are owned by
// config-load validation (Tier 1), not re-reported here.
//
// Composed at the CLI layer (internal/cli/check) with the declared areas
// sourced from config — the same seam AreaUnknown uses — so the pure
// check.Run stays config-agnostic. Inert when no area declares a `paths:`
// glob (label-only / legacy string-form config), so a paths-less areas block
// fires nothing on the path axis (M-0180/AC-5). Severity is warning,
// escalated to error under areas.required by ApplyAreaRequiredStrict.
func AreaDeadGlob(t *tree.Tree, areas []AreaPaths) []Finding {
	if t.Root == "" {
		return nil
	}
	// Never fail on IO: a missing or unreadable root yields no findings
	// rather than firing dead-glob for every area (the roadmapCaseCollision
	// precedent). doublestar.Glob over a non-existent root returns empty
	// without erroring, so without this guard every glob would read as dead.
	if _, err := os.Stat(t.Root); err != nil {
		return nil
	}
	fsys := os.DirFS(t.Root)
	var findings []Finding
	for _, a := range areas {
		for _, glob := range a.Paths {
			matched, err := areamatch.MatchesAny(fsys, glob)
			if err != nil {
				// Never fail on IO. A malformed glob cannot reach here —
				// areamatch.Validate rejects it at config load (Tier 1), so an
				// error here is a filesystem walk failure, skipped as
				// indeterminate (the roadmapCaseCollision precedent).
				continue
			}
			if matched {
				continue
			}
			findings = append(findings, Finding{
				Code:     CodeAreaDeadGlob,
				Severity: SeverityWarning,
				Message: fmt.Sprintf(
					"area %q declares path glob %q which matches no file or directory",
					a.Name, glob),
				Field: "areas.members.paths",
			})
		}
	}
	return findings
}
