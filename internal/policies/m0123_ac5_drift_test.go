package policies

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/entity"
	"github.com/23min/aiwf/internal/workflows/spec"
)

// AC-5 — bidirectional drift policy between spec.Rules()/AntiRules() and
// the impl-side kernel symbols.
//
// The two arms:
//
//   - impl → spec: every (Kind, FromState) the kernel FSM recognizes is
//     referenced by ≥1 spec Rule; every top-level Cobra verb is referenced
//     OR allowlisted as "no per-entity legal semantics".
//
//   - spec → impl: every Rule's Kind / FromState / Verb / ExpectedErrorCode
//     resolves to a real impl symbol (or, for ExpectedErrorCode, an
//     allowlisted deferred-impl entry citing the D-NNNN that tracks the
//     missing implementation).
//
// What's deliberately out of scope for AC-5:
//
// The "impl → spec finding-code coverage" arm (every legality-pertinent
// finding code emitted by the impl is referenced by ≥1 illegal-outcome
// Rule) is genuinely hard without an impl-side classifier distinguishing
// "verb-time legality" findings from "structural integrity" findings.
// Today the impl emits codes from both classes through the same Finding
// struct shape. Classification would require either tagging each Code at
// declaration time (adding metadata to internal/check/) or maintaining a
// parallel closed-set enumeration here — both wider than M-0123. A
// follow-up gap at wrap captures the deferral.

// TestM0123_AC5_ImplToSpec_EntityFSMCovered asserts every (Kind, FromState)
// in the kernel's entity FSM is referenced by ≥1 Rule. Walks
// entity.AllKinds() × entity.AllowedStatuses(kind) — the exported
// enumeration surfaces, so a new kind or status added to the FSM grows
// the test's input space automatically.
//
// Strengthens AC-2's TestM0123_AC2_EveryEntityFSMFromStateCovered (which
// hardcoded the state lists in canonicalFromStates) by sourcing the truth
// from entity package's exported enumerators.
func TestM0123_AC5_ImplToSpec_EntityFSMCovered(t *testing.T) {
	t.Parallel()

	covered := buildSpecCoverageMap()

	for _, k := range entity.AllKinds() {
		for _, fs := range entity.AllowedStatuses(k) {
			if !covered[k][fs] {
				t.Errorf("spec.Rules() missing coverage for (Kind=%q, FromState=%q): no cell references this FSM position", k, fs)
			}
		}
	}
}

// TestM0123_AC5_ImplToSpec_ACFSMCovered asserts every status in
// entity.AllowedACStatuses() is referenced by ≥1 Rule with Kind=KindAC.
// Mirrors the entity-FSM coverage arm for the AC sub-FSM.
func TestM0123_AC5_ImplToSpec_ACFSMCovered(t *testing.T) {
	t.Parallel()

	covered := buildSpecCoverageMap()

	for _, fs := range entity.AllowedACStatuses() {
		if !covered[spec.KindAC][fs] {
			t.Errorf("spec.Rules() missing coverage for (Kind=KindAC, FromState=%q): no cell references this AC FSM position", fs)
		}
	}
}

// TestM0123_AC5_ImplToSpec_TDDPhaseFSMCovered asserts every TDD phase
// (including the "" pre-cycle entry state) is referenced by ≥1 Rule with
// Kind=KindTDDPhase. The "" entry state is part of the FSM by design
// (transition.go:159: tddPhaseTransitions[""] = {"red"}) so cells must
// cover it.
func TestM0123_AC5_ImplToSpec_TDDPhaseFSMCovered(t *testing.T) {
	t.Parallel()

	covered := buildSpecCoverageMap()

	// The "" pre-cycle state is not in AllowedTDDPhases (which returns
	// the canonical post-entry phases). Include it explicitly per the
	// FSM declaration.
	tddPhases := append([]string{""}, entity.AllowedTDDPhases()...)

	for _, fs := range tddPhases {
		if !covered[spec.KindTDDPhase][fs] {
			t.Errorf("spec.Rules() missing coverage for (Kind=KindTDDPhase, FromState=%q): no cell references this TDD-phase FSM position", fs)
		}
	}
}

// nonLegalityVerbAllowlist names top-level Cobra verbs that don't drive
// per-entity FSM transitions and therefore aren't referenced by Rules().
// Each entry carries a one-line rationale so a contributor adding a new
// verb has to explicitly classify it: either wire it into spec.Rules()
// (FSM-driving / legality-pertinent) or add an allowlist entry.
//
// Drift is policed by TestM0123_AC5_ImplToSpec_VerbsCovered.
var nonLegalityVerbAllowlist = map[string]string{
	"version":    "metadata-only; no entity mutation",
	"check":      "validation engine; emits findings rather than driving FSM transitions",
	"add":        "creation verb; new entities enter at the kind's initial status with no legality choice",
	"rename":     "slug-only mutation; FSM state is preserved",
	"retitle":    "title-only mutation; FSM state is preserved",
	"edit-body":  "body-only mutation; FSM state is preserved",
	"move":       "branch-cross / file-move mutation; FSM state is preserved",
	"reallocate": "id renumber; FSM state is preserved",
	"rewidth":    "id-width canonicalization (ADR-0008); FSM state is preserved",
	"archive":    "terminal-state sweep; status is already terminal before the verb runs (ADR-0004)",
	"init":       "framework bootstrap in consumer repo; no entity state",
	"update":     "framework artifact refresh in consumer repo; no entity state",
	"upgrade":    "self-upgrade of the aiwf binary; no entity state",
	"history":    "read-only git-log query",
	"doctor":     "consumer-repo health report; no entity mutation",
	"render":     "read-only HTML / status surface generator",
	"import":     "creation verb; same initial-status reasoning as add",
	"whoami":     "identity query; no entity state",
	"status":     "read-only tree state snapshot",
	"list":       "read-only filtered listing",
	"schema":     "read-only schema introspection",
	"show":       "read-only entity inspection",
	"template":   "scaffold-prose query; no entity mutation",
	"contract":   "topical verb group; sub-verbs handle their own non-FSM lifecycle",
	"milestone":  "topical verb group for sub-verbs (depends-on); each sub-verb has its own non-FSM mutation",
}

// TestM0123_AC5_ImplToSpec_VerbsCovered asserts every top-level Cobra verb
// is either referenced by ≥1 Rule (FSM-driving / legality-pertinent) or
// listed in nonLegalityVerbAllowlist with a rationale.
//
// The verb set is sourced via findTopLevelVerbs (the existing AST walker
// from skill_coverage.go) so a new AddCommand call in root.go grows the
// test's input space automatically.
func TestM0123_AC5_ImplToSpec_VerbsCovered(t *testing.T) {
	t.Parallel()

	verbs, err := findTopLevelVerbs(repoRoot(t))
	if err != nil {
		t.Fatalf("findTopLevelVerbs: %v", err)
	}

	verbsInSpec := map[string]bool{}
	for _, r := range spec.Rules() {
		verbsInSpec[r.Verb] = true
	}

	for verb := range verbs {
		if verbsInSpec[verb] {
			continue
		}
		if _, allowlisted := nonLegalityVerbAllowlist[verb]; allowlisted {
			continue
		}
		t.Errorf("top-level Cobra verb %q is not referenced by any spec.Rules() cell AND has no nonLegalityVerbAllowlist entry — either wire it into spec.Rules() with a Kind/FromState cell or add an allowlist entry with a one-line rationale", verb)
	}
}

// TestM0123_AC5_SpecToImpl_KindsResolve asserts every Rule's Kind is a
// recognized kind value — either one of entity.AllKinds() or one of the
// two spec.Kind* extensions (KindAC, KindTDDPhase).
func TestM0123_AC5_SpecToImpl_KindsResolve(t *testing.T) {
	t.Parallel()

	recognized := map[entity.Kind]bool{
		spec.KindAC:       true,
		spec.KindTDDPhase: true,
	}
	for _, k := range entity.AllKinds() {
		recognized[k] = true
	}

	for i, r := range spec.Rules() {
		if !recognized[r.Kind] {
			t.Errorf("Rules()[%d]: Kind=%q does not resolve to entity.AllKinds() or spec.Kind{AC,TDDPhase}", i, r.Kind)
		}
	}
}

// TestM0123_AC5_SpecToImpl_FromStatesResolve asserts every Rule's
// FromState is recognized for its Kind. Sources of truth:
//
//   - entity kinds: entity.AllowedStatuses(kind)
//   - spec.KindAC: entity.AllowedACStatuses()
//   - spec.KindTDDPhase: entity.AllowedTDDPhases() ∪ {""}
func TestM0123_AC5_SpecToImpl_FromStatesResolve(t *testing.T) {
	t.Parallel()

	stateSet := func(states []string) map[string]bool {
		out := make(map[string]bool, len(states))
		for _, s := range states {
			out[s] = true
		}
		return out
	}

	allowedByKind := map[entity.Kind]map[string]bool{
		spec.KindAC:       stateSet(entity.AllowedACStatuses()),
		spec.KindTDDPhase: stateSet(append([]string{""}, entity.AllowedTDDPhases()...)),
	}
	for _, k := range entity.AllKinds() {
		allowedByKind[k] = stateSet(entity.AllowedStatuses(k))
	}

	for i, r := range spec.Rules() {
		allowed, ok := allowedByKind[r.Kind]
		if !ok {
			// KindsResolve will fail on the same Rule; skip here.
			continue
		}
		if !allowed[r.FromState] {
			t.Errorf("Rules()[%d] (Kind=%q): FromState=%q is not a recognized state for this kind", i, r.Kind, r.FromState)
		}
	}
}

// TestM0123_AC5_SpecToImpl_VerbsResolve asserts every Rule's Verb is a
// real top-level Cobra verb. Drift would mean a spec cell pointing at a
// fiction verb — e.g., a typo'd "promot" or a stale name after a verb
// rename.
func TestM0123_AC5_SpecToImpl_VerbsResolve(t *testing.T) {
	t.Parallel()

	verbs, err := findTopLevelVerbs(repoRoot(t))
	if err != nil {
		t.Fatalf("findTopLevelVerbs: %v", err)
	}

	for i, r := range spec.Rules() {
		if _, ok := verbs[r.Verb]; !ok {
			t.Errorf("Rules()[%d] (Kind=%q, FromState=%q): Verb=%q does not resolve to a real top-level Cobra verb", i, r.Kind, r.FromState, r.Verb)
		}
	}
}

// deferredImplErrorCodes names spec ExpectedErrorCodes whose impl-side
// emission hasn't landed yet, with the D-NNNN that tracks the missing
// implementation. The drift test treats these as resolved.
//
// When the impl lands, the deferred entry comes out and the same code is
// expected to appear as a `Code: "..."` literal in internal/, picked up
// by codeAppearsInImplSource. Forgetting to remove the entry is the
// only failure mode and it's a one-line edit.
var deferredImplErrorCodes = map[string]string{
	"fsm-transition-illegal":            "entity.ValidateTransition returns free-form errors today; structured-code wrap is a follow-up gap (filed at M-0123 wrap)",
	"epic-cancel-non-terminal-children": "D-0003 cancel-cascade impl is a follow-up gap (filed at M-0123 wrap)",
	"milestone-cancel-non-terminal-acs": "D-0004 cancel-cascade impl is a follow-up gap (filed at M-0123 wrap)",
	"ac-evidence-missing":               "D-0005 AC mechanical-evidence mechanism is a follow-up gap (filed at M-0123 wrap)",
	"authorize-kind-not-allowed":        "D-0007 authorize kind restriction impl is a follow-up gap (filed at M-0123 wrap)",
}

// TestM0123_AC5_SpecToImpl_ErrorCodesResolve asserts every illegal Rule's
// non-empty ExpectedErrorCode resolves to either an impl-side `Code: "X"`
// literal anywhere under internal/ (excluding the spec package itself and
// test files) OR an entry in deferredImplErrorCodes.
//
// The walk: parse every non-test .go file under internal/ (excluding
// internal/workflows/spec) and collect string literals appearing in
// `Code: "..."` composite-literal fields. The set is the canonical
// impl-side surface for finding codes; AC-5's spec→impl arm closes the
// reverse direction.
func TestM0123_AC5_SpecToImpl_ErrorCodesResolve(t *testing.T) {
	t.Parallel()

	implCodes, err := collectImplFindingCodes(repoRoot(t))
	if err != nil {
		t.Fatalf("collectImplFindingCodes: %v", err)
	}

	for i, r := range spec.Rules() {
		if r.Outcome != spec.OutcomeIllegal || r.ExpectedErrorCode == "" {
			continue
		}
		code := r.ExpectedErrorCode
		if implCodes[code] {
			continue
		}
		if _, deferred := deferredImplErrorCodes[code]; deferred {
			continue
		}
		t.Errorf("Rules()[%d] (Kind=%q, FromState=%q, Verb=%q): ExpectedErrorCode=%q resolves to neither an impl `Code: \"...\"` literal nor a deferredImplErrorCodes entry — implement the code or add a deferred entry with the tracking D-NNNN reason",
			i, r.Kind, r.FromState, r.Verb, code)
	}
}

// buildSpecCoverageMap walks spec.Rules() once and returns the set of
// (Kind, FromState) positions referenced by at least one cell. Used by
// the three impl-FSM-coverage tests.
func buildSpecCoverageMap() map[entity.Kind]map[string]bool {
	covered := map[entity.Kind]map[string]bool{}
	rules := spec.Rules()
	for i := range rules {
		r := &rules[i]
		if covered[r.Kind] == nil {
			covered[r.Kind] = map[string]bool{}
		}
		covered[r.Kind][r.FromState] = true
	}
	return covered
}

// collectImplFindingCodes walks every non-test .go file under
// <root>/internal (excluding internal/workflows/spec, which is what the
// drift test resolves against) and returns the set of distinct string
// literals appearing as the value of a `Code` field in a composite
// literal (any struct shape — check.Finding plus any pseudo-finding type
// used in tests/fixtures, all use the same field name).
//
// Mirrors the AST walk in finding_hints.go but returns a set instead of
// a slice of (file, line) tuples — AC-5 needs membership, not source
// position.
func collectImplFindingCodes(root string) (map[string]bool, error) {
	internalDir := filepath.Join(root, "internal")
	out := map[string]bool{}
	fset := token.NewFileSet()

	err := filepath.Walk(internalDir, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if info.IsDir() {
			// Exclude the spec package itself — the table is what we're
			// resolving against; codes mentioned there are inputs, not
			// impl-side declarations.
			if filepath.Base(path) == "spec" && strings.HasSuffix(filepath.Dir(path), "workflows") {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		astFile, parseErr := parser.ParseFile(fset, path, nil, parser.AllErrors)
		if parseErr != nil {
			return nil
		}
		ast.Inspect(astFile, func(n ast.Node) bool {
			cl, ok := n.(*ast.CompositeLit)
			if !ok {
				return true
			}
			for _, elt := range cl.Elts {
				kv, ok := elt.(*ast.KeyValueExpr)
				if !ok {
					continue
				}
				ident, ok := kv.Key.(*ast.Ident)
				if !ok || ident.Name != "Code" {
					continue
				}
				if bl, ok := kv.Value.(*ast.BasicLit); ok && bl.Kind == token.STRING {
					if s, err := strconv.Unquote(bl.Value); err == nil && s != "" {
						out[s] = true
					}
				}
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
