package policies

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// entitySlugLiteralPat matches the leading shape of an entity slug
// directory or filename — e.g. "E-0027-trailered-...", "M-0090-...",
// "G-0102-...", "ADR-0007-...". Per ADR-0008 canonical width is 4
// digits; the parser tolerates narrower legacy widths (≥1 digit)
// so the regex matches the same widths.
var entitySlugLiteralPat = regexp.MustCompile(`^(E|M|G|D|C|ADR)-\d+-`)

// PolicyNoHardcodedEntityPaths flags Go source under
// internal/policies/ that builds entity-tree paths via literal
// filepath.Join segments naming an entity slug. The anti-pattern:
//
//	filepath.Join(root, "work", "epics",
//	    "E-0027-trailered-merge-commits-...",
//	    "M-0090-aiwfx-wrap-epic-emits-....md")
//
// Such paths break the moment `aiwf archive --apply` moves the
// entity into a per-kind archive/ subdir, per ADR-0004's
// uniform-archive-convention. Tests that need an entity's file
// must resolve through the loader:
//
//	tr, _, err := tree.Load(ctx, root)
//	e := tr.ByID("M-0090")
//	specPath := filepath.Join(root, e.Path)
//
// tree.Load resolves ids across active and archive, so the lookup
// survives sweeps indefinitely.
//
// The check fires on any string-literal arg to filepath.Join whose
// value starts with an entity-slug prefix (E-/M-/G-/D-/C-/ADR-)
// followed by digits and a dash. The first arg is exempt — it's
// almost always a `root` variable or an absolute path, matching the
// scope-narrowing precedent in PolicyFilepathJoinSegmentBySegment.
//
// Why scoped to internal/policies/: that package is where
// fixture-shaped tests over the planning tree live; it's the
// natural site for the bug. Other packages (e.g. internal/check,
// internal/htmlrender) test against synthetic tempdir trees, not
// the live repo's entity files, so they don't carry this risk.
// If the pattern leaks elsewhere, widen the scope here in one
// commit.
//
// The bug this prevents: M-0090's first archive sweep aborted
// because TestAiwfxWrapEpic_AC4_RitualsRepoSHARecordedAtWrap read
// the milestone spec via a literal path that the archive move
// invalidated. See `internal/policies/aiwfx_wrap_epic_test.go`'s
// fixed call site for the canonical loader-based resolution.
func PolicyNoHardcodedEntityPaths(root string) ([]Violation, error) {
	// WalkGoFiles deliberately skips internal/policies/ to keep
	// other policies from firing on their own assertion strings.
	// This policy's scope is *inside* that directory, so it walks
	// the directory directly with a focused listing.
	policiesDir := filepath.Join(root, "internal", "policies")
	entries, err := os.ReadDir(policiesDir)
	if err != nil {
		return nil, err
	}
	var out []Violation
	fset := token.NewFileSet()
	for _, ent := range entries {
		if ent.IsDir() || !strings.HasSuffix(ent.Name(), ".go") {
			continue
		}
		// The policy's own implementation file is always exempt:
		// the entitySlugLiteralPat regex source is a string that
		// would match itself if scanned through string-literal
		// inspection. (Belt-and-suspenders — the regex source
		// happens to be `^(E|M|G|D|C|ADR)-\d+-`, which does not
		// itself match the regex, but locking in the exemption
		// removes any chance of a future tweak self-firing.)
		if ent.Name() == "no_hardcoded_entity_paths.go" {
			continue
		}
		absPath := filepath.Join(policiesDir, ent.Name())
		relPath := filepath.ToSlash(filepath.Join("internal", "policies", ent.Name()))
		contents, rerr := os.ReadFile(absPath)
		if rerr != nil {
			return nil, rerr
		}
		astFile, perr := parser.ParseFile(fset, absPath, contents, parser.AllErrors)
		if perr != nil {
			continue
		}
		ast.Inspect(astFile, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok || len(call.Args) < 2 {
				return true
			}
			sel, ok := call.Fun.(*ast.SelectorExpr)
			if !ok {
				return true
			}
			pkg, ok := sel.X.(*ast.Ident)
			if !ok || pkg.Name != "filepath" || sel.Sel.Name != "Join" {
				return true
			}
			for i := 1; i < len(call.Args); i++ {
				lit, ok := call.Args[i].(*ast.BasicLit)
				if !ok || lit.Kind != token.STRING {
					continue
				}
				val, err := strconv.Unquote(lit.Value)
				if err != nil {
					continue
				}
				if !entitySlugLiteralPat.MatchString(val) {
					continue
				}
				out = append(out, Violation{
					Policy: "no-hardcoded-entity-paths",
					File:   relPath,
					Line:   fset.Position(call.Pos()).Line,
					Detail: "filepath.Join arg " + strconv.Quote(val) +
						" names an entity slug literally; this path breaks when `aiwf archive --apply` moves the entity into archive/ (per ADR-0004). Resolve the entity via tree.Load(ctx, root) + Tree.ByID + entity.Path instead — the loader resolves ids across active and archive, so the lookup survives sweeps.",
				})
			}
			return true
		})
	}
	return out, nil
}
