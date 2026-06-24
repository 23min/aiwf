package policies

import (
	"fmt"
	"go/parser"
	"go/token"
	"path"
	"strings"
)

// PolicyLayeringDirection pins the kernel's import-direction invariant
// mechanically: a production package may import packages in the same or
// a lower layer, never a higher one. An "upward" import — a domain
// package reaching back into verb/cli/cmd, or any lower tier importing a
// higher tier — fails CI (G-0227 item 5).
//
// The layering is the one §A3 of the health scorecard verified ("no
// upward dependencies"): arrows point uniformly downward
//
//	cmd → cli → verb → check/render/htmlrender/initrepo →
//	tree/scope/trunk/contractcheck → entity/gitops/aiwfyaml/config →
//	codes/pathutil
//
// The tiers below are that doctrine *corrected to the real DAG*. The
// scorecard prose groups packages into loose altitude bands that do not
// survive literal enforcement — the arrow shows contractcheck in the
// tree/scope band, but contractcheck actually imports check, so layerTier
// places it *above* check (tier 3, between verb and the check/render
// band); render imports check and tree imports trunk likewise. The prose
// bands are an approximation, not a partition. layerTier encodes the
// corrected assignment under which every current edge is downward or
// sideways.
//
// Same-tier ("sideways") imports are deliberately allowed. The invariant
// the scorecard validated, and the only one that is false-positive-free,
// is no-upward. A sideways edge cannot invert an *existing* dependency
// (Go's own cycle ban already prevents that), so it cannot flip the
// architecture upside down. It can, however, introduce a brand-new
// directional coupling between two currently-independent same-tier
// packages (e.g. a future entity→gitops) — that is the accepted blind
// spot of the coarse-band design, not a defect. A finer per-package total
// order (forbidding sideways too) was considered and rejected as brittle:
// it would cry wolf on legitimate same-tier refactors of a still-growing
// tree.
//
// Two test-only packages are allowlisted by name + rationale: they sit
// outside the production dependency spine (consumed only by *_test.go),
// so their imports legitimately reach into any layer for fixture setup.
//
// Scope is every production file WalkGoFiles returns (internal/ + cmd/;
// _test.go and internal/policies/ are excluded). The import-path walk is
// complete for what it scans — every import form (named, dot, blank)
// carries its path literal in ast.File.Imports, so there is no aliasing
// blind spot of the kind the os.* write policies have. The one unscanned
// production package is internal/policies/ itself (skipped by
// construction; it imports only entity, a leaf-ward edge).
func PolicyLayeringDirection(root string) ([]Violation, error) {
	files, err := WalkGoFiles(root, true)
	if err != nil {
		return nil, err //coverage:ignore WalkGoFiles errors only on a filesystem walk failure; not reachable with a valid tree root.
	}
	const modulePrefix = "github.com/23min/aiwf/"
	var out []Violation
	reported := map[string]bool{} // dedupe per-package / per-edge findings
	fset := token.NewFileSet()
	for _, f := range files {
		srcPkg := path.Dir(f.Path)
		if _, ok := layeringAllowlist[srcPkg]; ok {
			continue
		}
		srcTier, known := layerTier(srcPkg)
		if !known {
			if !reported[srcPkg] {
				reported[srcPkg] = true
				out = append(out, Violation{
					Policy: "layering-direction",
					File:   f.Path,
					Detail: srcPkg + " has no layering tier; add it to layerTier (or allowlist it as test-only)",
				})
			}
			continue
		}
		astFile, perr := parser.ParseFile(fset, f.AbsPath, f.Contents, parser.ImportsOnly)
		if perr != nil {
			continue
		}
		for _, imp := range astFile.Imports {
			ipath := strings.Trim(imp.Path.Value, `"`)
			if !strings.HasPrefix(ipath, modulePrefix) {
				continue
			}
			tgtPkg := strings.TrimPrefix(ipath, modulePrefix)
			if _, ok := layeringAllowlist[tgtPkg]; ok {
				continue
			}
			tgtTier, tok := layerTier(tgtPkg)
			if !tok {
				key := srcPkg + "=>" + tgtPkg
				if !reported[key] {
					reported[key] = true
					out = append(out, Violation{
						Policy: "layering-direction",
						File:   f.Path,
						Line:   fset.Position(imp.Pos()).Line,
						Detail: "import of untiered package " + tgtPkg + "; add it to layerTier",
					})
				}
				continue
			}
			if tgtTier < srcTier {
				out = append(out, Violation{
					Policy: "layering-direction",
					File:   f.Path,
					Line:   fset.Position(imp.Pos()).Line,
					Detail: fmt.Sprintf("%s (tier %d) imports %s (tier %d): upward dependency — imports must point to the same or a lower layer",
						srcPkg, srcTier, tgtPkg, tgtTier),
				})
			}
		}
	}
	return out, nil
}

// layeringAllowlist names production packages exempt from the
// import-direction check. Key is the module-relative package path; value
// is the rationale (kept beside the exemption so they travel together).
// Both are test-only: consumed solely by *_test.go, outside the
// production binary closure, so reaching into any layer for fixture
// setup is legitimate.
var layeringAllowlist = map[string]string{
	"internal/cellcoverage": "test-fixture helper consumed only by *_test.go under internal/policies; imports verb+cliutil by design, excluded from the production binary closure (scorecard §A3)",
	"internal/testsupport":  "test-only git/env hardening glue consumed solely by TestMains; not part of the production dependency spine",
}

// layerTier returns the pinned layer tier for a module-relative package
// path (e.g. "internal/verb") and whether the path is known to the map.
// Lower numbers are higher altitude: an import is legal iff the target
// tier is greater than or equal to the source tier. The assignment is
// the scorecard §A3 doctrine corrected so every current edge is downward
// or sideways (see PolicyLayeringDirection).
func layerTier(pkg string) (tier int, known bool) {
	switch pkg {
	case "cmd/aiwf":
		return 0, true
	case "internal/verb":
		return 2, true
	case "internal/contractcheck", "internal/contractverify":
		return 3, true
	case "internal/check", "internal/render", "internal/htmlrender",
		"internal/initrepo", "internal/roadmap", "internal/contractconfig":
		return 4, true
	case "internal/tree", "internal/scope", "internal/trunk",
		"internal/manifest", "internal/recipe", "internal/skills":
		return 5, true
	case "internal/entity", "internal/gitops", "internal/aiwfyaml", "internal/config":
		return 6, true
	case "internal/codes", "internal/pathutil", "internal/version",
		"internal/branchparse", "internal/repolock", "internal/pluginstate",
		"internal/areagroup":
		return 7, true
	}
	// Prefix bands: the CLI ring (cli + every cli/* adapter, incl. the
	// cliutil hub) is one tier; the workflows spec catalog is one tier.
	if pkg == "internal/cli" || strings.HasPrefix(pkg, "internal/cli/") {
		return 1, true
	}
	if pkg == "internal/workflows" || strings.HasPrefix(pkg, "internal/workflows/") {
		return 3, true
	}
	return 0, false
}
