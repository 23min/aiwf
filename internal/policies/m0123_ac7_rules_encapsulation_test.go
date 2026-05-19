package policies

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// AC-7 — encapsulation: LookupRules is the public access surface for the
// spec table. Production code outside internal/workflows/spec must reach
// for LookupRules to answer "is verb V legal on (Kind, FromState)?"; it
// must not iterate spec.Rules() / spec.AntiRules() directly.
//
// The encapsulation rule's purpose:
//
//   - Single point of refactor: any future shape change to the table
//     ripples through LookupRules' signature, not through every consumer's
//     filter loop.
//   - Single point of optimization: LookupRules can later add indexing,
//     caching, or precondition pruning; consumers don't need to know.
//   - Single point of contract: LookupRules names the (kind, fromState,
//     verb) query shape. Consumers that iterate the slice are free to
//     misuse the table (e.g., filter on just Verb, ignoring the kind).
//
// The chokepoint is this test. The drift tests in internal/policies/
// (AC-2, AC-5, AC-6) legitimately iterate spec.Rules() — they police the
// table's well-formedness rather than answering "is X legal?". Test files
// (_test.go) under any package are exempted on the same reasoning: tests
// often need full-table iteration to drive coverage or assertions.

// allowedSpecRulesAccessPrefixes lists the path prefixes (relative to repo
// root, slash-separated) where references to spec.Rules / spec.AntiRules
// are legitimate. Two entries by design — the spec package itself
// (production callers can reach for the unexported state via the package
// boundary) and the drift-policy package (which polices spec
// well-formedness).
var allowedSpecRulesAccessPrefixes = []string{
	"internal/workflows/spec",
	"internal/policies",
}

// TestM0123_AC7_RulesNotReferencedFromProduction asserts no .go file
// under internal/ outside the allowlisted prefixes references
// spec.Rules or spec.AntiRules. Test files (_test.go) are exempt globally
// — test code often needs full-table iteration to assert invariants and
// is not the consumer-facing access path.
//
// Walks the live repo. Currently zero production callers (LookupRules is
// the only public surface in use); the test's bite is for future PRs
// that might reach for the slice.
func TestM0123_AC7_RulesNotReferencedFromProduction(t *testing.T) {
	t.Parallel()

	internalDir := filepath.Join(repoRoot(t), "internal")
	refs, err := findProductionSpecRulesRefs(internalDir, allowedSpecRulesAccessPrefixes)
	if err != nil {
		t.Fatalf("findProductionSpecRulesRefs: %v", err)
	}
	for _, ref := range refs {
		t.Errorf("%s:%d: production code references spec.%s — use spec.LookupRules instead (encapsulation per M-0123/AC-7)",
			ref.File, ref.Line, ref.Selector)
	}
}

// TestM0123_AC7_PolicyFiresOnViolation is the negative-case companion to
// the live-repo test above (CLAUDE.md §"Test untested code paths" —
// every reachable branch needs a test that traverses it). Drives the
// walker with a synthetic .go fixture that imports spec and references
// Rules(); asserts the walker reports the violation. Without this test,
// the policy could silently fail to detect the case it exists to catch.
func TestM0123_AC7_PolicyFiresOnViolation(t *testing.T) {
	t.Parallel()

	tmp := t.TempDir()
	// Mirror the internal/<pkg> shape so the prefix-exemption logic
	// receives realistic input.
	pkgDir := filepath.Join(tmp, "internal", "consumer")
	if err := os.MkdirAll(pkgDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	src := `package consumer

import "github.com/23min/aiwf/internal/workflows/spec"

func bad() int {
	rules := spec.Rules()
	return len(rules)
}

func alsoBad() int {
	return len(spec.AntiRules())
}
`
	if err := os.WriteFile(filepath.Join(pkgDir, "consumer.go"), []byte(src), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	internalDir := filepath.Join(tmp, "internal")
	refs, err := findProductionSpecRulesRefs(internalDir, allowedSpecRulesAccessPrefixes)
	if err != nil {
		t.Fatalf("findProductionSpecRulesRefs: %v", err)
	}

	if len(refs) != 2 {
		t.Fatalf("want 2 violations (Rules + AntiRules), got %d: %+v", len(refs), refs)
	}
	sawRules, sawAntiRules := false, false
	for _, r := range refs {
		switch r.Selector {
		case "Rules":
			sawRules = true
		case "AntiRules":
			sawAntiRules = true
		}
	}
	if !sawRules {
		t.Error("walker missed the spec.Rules reference")
	}
	if !sawAntiRules {
		t.Error("walker missed the spec.AntiRules reference")
	}
}

// TestM0123_AC7_PolicyHonorsAlias asserts the walker tracks alias imports
// (`specalias "github.com/23min/aiwf/internal/workflows/spec"`) and
// catches `specalias.Rules` references. A naive walker that just looks
// for `spec.Rules` would miss the alias case — and the violation would
// ship.
func TestM0123_AC7_PolicyHonorsAlias(t *testing.T) {
	t.Parallel()

	tmp := t.TempDir()
	pkgDir := filepath.Join(tmp, "internal", "consumer")
	if err := os.MkdirAll(pkgDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	src := `package consumer

import specalias "github.com/23min/aiwf/internal/workflows/spec"

func bad() int {
	return len(specalias.Rules())
}
`
	if err := os.WriteFile(filepath.Join(pkgDir, "consumer.go"), []byte(src), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	internalDir := filepath.Join(tmp, "internal")
	refs, err := findProductionSpecRulesRefs(internalDir, allowedSpecRulesAccessPrefixes)
	if err != nil {
		t.Fatalf("findProductionSpecRulesRefs: %v", err)
	}
	if len(refs) != 1 || refs[0].Selector != "Rules" {
		t.Errorf("want 1 violation for aliased spec.Rules, got %+v", refs)
	}
}

// TestM0123_AC7_PolicyExemptsLookupRules asserts LookupRules references
// are NOT flagged. Without this, the policy could be overly broad and
// trip on the legitimate consumer access path.
func TestM0123_AC7_PolicyExemptsLookupRules(t *testing.T) {
	t.Parallel()

	tmp := t.TempDir()
	pkgDir := filepath.Join(tmp, "internal", "consumer")
	if err := os.MkdirAll(pkgDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	src := `package consumer

import (
	"github.com/23min/aiwf/internal/entity"
	"github.com/23min/aiwf/internal/workflows/spec"
)

func legit() int {
	return len(spec.LookupRules(entity.KindEpic, "proposed", "promote"))
}
`
	if err := os.WriteFile(filepath.Join(pkgDir, "consumer.go"), []byte(src), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	internalDir := filepath.Join(tmp, "internal")
	refs, err := findProductionSpecRulesRefs(internalDir, allowedSpecRulesAccessPrefixes)
	if err != nil {
		t.Fatalf("findProductionSpecRulesRefs: %v", err)
	}
	if len(refs) != 0 {
		t.Errorf("LookupRules reference should NOT be flagged, got violations: %+v", refs)
	}
}

// TestM0123_AC7_PolicyExemptsTestFiles asserts _test.go files are
// exempt regardless of directory. Drift tests under internal/policies/
// are already covered by the prefix exemption; this case covers test
// files elsewhere (e.g., a consumer package's unit test that exercises
// the full table for coverage).
func TestM0123_AC7_PolicyExemptsTestFiles(t *testing.T) {
	t.Parallel()

	tmp := t.TempDir()
	pkgDir := filepath.Join(tmp, "internal", "consumer")
	if err := os.MkdirAll(pkgDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	src := `package consumer

import "github.com/23min/aiwf/internal/workflows/spec"

func TestSomething() {
	_ = spec.Rules()
}
`
	if err := os.WriteFile(filepath.Join(pkgDir, "consumer_test.go"), []byte(src), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	internalDir := filepath.Join(tmp, "internal")
	refs, err := findProductionSpecRulesRefs(internalDir, allowedSpecRulesAccessPrefixes)
	if err != nil {
		t.Fatalf("findProductionSpecRulesRefs: %v", err)
	}
	if len(refs) != 0 {
		t.Errorf("test file should be exempt, got violations: %+v", refs)
	}
}

// specRulesRef records a single production-side reference to a
// disallowed spec selector. File is repo-relative slash-separated; Line
// is 1-indexed.
type specRulesRef struct {
	File     string
	Line     int
	Selector string
}

// findProductionSpecRulesRefs walks internalDir for .go files outside
// the allowed prefixes (and outside _test.go), parses each, tracks the
// local name of the workflows/spec import (default "spec"; alias if
// declared), and records every SelectorExpr where the X identifier is
// the spec local name and the Sel name is "Rules" or "AntiRules".
//
// Returned File paths are relative to internalDir's parent (so they read
// as "internal/<pkg>/<file>.go"), matching the prefix list's shape.
func findProductionSpecRulesRefs(internalDir string, allowedPrefixes []string) ([]specRulesRef, error) {
	parent := filepath.Dir(internalDir)
	const specImportPath = "github.com/23min/aiwf/internal/workflows/spec"

	var out []specRulesRef
	err := filepath.Walk(internalDir, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if info.IsDir() {
			return nil
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		rel, err := filepath.Rel(parent, path)
		if err != nil {
			return err
		}
		relSlash := filepath.ToSlash(rel)
		for _, prefix := range allowedPrefixes {
			if strings.HasPrefix(relSlash, prefix+"/") || relSlash == prefix {
				return nil
			}
		}

		fset := token.NewFileSet()
		astFile, parseErr := parser.ParseFile(fset, path, nil, parser.AllErrors)
		if parseErr != nil {
			return nil
		}

		// Find the local name of the workflows/spec import. Default to
		// "spec" (package name); use the alias if set. Dot-imports
		// (`. "..."`) would make Rules() callable bare-named; this code
		// treats them as not-imported (no SelectorExpr to find).
		localName := ""
		for _, imp := range astFile.Imports {
			if imp.Path == nil {
				continue
			}
			pathLit := strings.Trim(imp.Path.Value, `"`)
			if pathLit != specImportPath {
				continue
			}
			if imp.Name != nil {
				if imp.Name.Name == "." || imp.Name.Name == "_" {
					// Dot/blank imports: not tracked.
					localName = ""
				} else {
					localName = imp.Name.Name
				}
			} else {
				localName = "spec"
			}
			break
		}
		if localName == "" {
			return nil
		}

		ast.Inspect(astFile, func(n ast.Node) bool {
			sel, ok := n.(*ast.SelectorExpr)
			if !ok {
				return true
			}
			ident, ok := sel.X.(*ast.Ident)
			if !ok || ident.Name != localName {
				return true
			}
			if sel.Sel.Name == "Rules" || sel.Sel.Name == "AntiRules" {
				out = append(out, specRulesRef{
					File:     relSlash,
					Line:     fset.Position(sel.Pos()).Line,
					Selector: sel.Sel.Name,
				})
			}
			return true
		})
		return nil
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}
