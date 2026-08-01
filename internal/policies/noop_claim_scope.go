package policies

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"sort"
	"strings"
)

// Claim scopes. The set is closed: a converging verb takes one of these,
// or the set grows deliberately — which is a design change rather than a
// list edit.
//
// The four are not interchangeable, and collapsing them is the specific
// failure this ledger exists to make visible. Scoping everything to "the
// target entity" was measured to leave the guard inert at the three sites
// that splice a working-copy aiwf.yaml, which have no target entity at all.
const (
	// claimScopeTargetEntity means the verb's convergence rests on the
	// stored state of the entity it was handed — its frontmatter field,
	// its body heading, its path — so the guard compares that entity's
	// file. A composite (<id>/AC-N) claim reads its parent milestone's
	// file, which is the same scope named from a sub-element.
	claimScopeTargetEntity = "target-entity"

	// claimScopeConfigFile means convergence rests on what aiwf.yaml
	// declares — a contract binding, a validator, the area vocabulary —
	// so the guard compares that file rather than any entity.
	claimScopeConfigFile = "aiwf-yaml"

	// claimScopeSweepDeciders means the verb has no target by
	// construction — its claim is derived from scanning the tree — and
	// the comparison is per candidate rather than per verb: a candidate
	// whose verdict rests on a mid-edit file is declined and reported,
	// while the rest of the sweep proceeds. Refusing the whole verb
	// instead would block unrelated work on any in-progress edit, and
	// exempting it would commit verdicts the record contradicts.
	claimScopeSweepDeciders = "sweep-deciders"

	// claimScopeNone means no comparison is wired, for one of two
	// reasons the entry has to state: the working copy cannot contradict
	// the claim at all (the baseline is git history, or the verb already
	// compares against HEAD itself), or it can and the divergence costs
	// nothing recoverable. The two are not interchangeable — the first
	// never needs revisiting, the second needs revisiting whenever its
	// cost changes — so the reason field carries the whole content of
	// such an entry: there is no guard to inspect, and the justification
	// is the only thing distinguishing a reasoned exemption from an
	// omission.
	claimScopeNone = "none"
)

// claimScope records what one converging verb's same-state claim is
// about.
type claimScope struct {
	Func   string
	Scope  string
	Reason string
}

// noOpClaimScopes records the claim scope of every internal/verb function
// that constructs a same-state NoOp. The set of such functions is derived
// from the source by the AST scan below rather than from this list, so a
// verb that starts converging without a recorded scope fails.
//
// What this list is for: a guard scoped to the wrong file is
// indistinguishable, in the code, from a guard scoped to the right one —
// both compile, both run, and the wrong one silently permits exactly the
// claims it was added to refuse. Writing the scope down per verb makes a
// change of scope a visible edit rather than a one-line substitution.
//
// What it is NOT: evidence that a scope is correct. It records the
// answer, not its justification-by-measurement — that is read at review,
// the same bound the sibling write-guard ledger states about itself.
var noOpClaimScopes = []claimScope{
	// Target entity: the claim reads the entity the verb was handed.
	{"Promote", claimScopeTargetEntity, "converges on the target's stored status; --superseded-by widens the scope to the superseding ADR, which the same claim reads"},
	{"Cancel", claimScopeTargetEntity, "converges on the target's stored status being terminal"},
	{"Move", claimScopeTargetEntity, "converges on the target's parent field and its file's location, both read from that file"},
	{"Rename", claimScopeTargetEntity, "converges on the target's path, derived from the file the loader read it from"},
	{"Retitle", claimScopeTargetEntity, "converges on the target's stored title and its body H1, both in that file"},
	{"SetPriority", claimScopeTargetEntity, "converges on the target's stored priority field"},
	{"SetArea", claimScopeTargetEntity, "converges on the target's stored area field"},
	{"MilestoneTDD", claimScopeTargetEntity, "converges on the milestone's stored tdd policy"},
	{"MilestoneDependsOn", claimScopeTargetEntity, "converges on the milestone's stored depends_on list"},
	{"promoteAC", claimScopeTargetEntity, "converges on an AC's stored status, which lives in the parent milestone's file"},
	{"cancelAC", claimScopeTargetEntity, "converges on an AC's stored status being terminal, read from the parent milestone's file"},
	{"renameAC", claimScopeTargetEntity, "converges on an AC's stored title and its body heading, both in the parent milestone's file"},
	{"retitleAC", claimScopeTargetEntity, "converges on an AC's stored title and its body heading, both in the parent milestone's file"},

	// aiwf.yaml: the claim reads the config, and there is no entity whose
	// file would answer it.
	{"ContractBind", claimScopeConfigFile, "converges on the binding already recorded in aiwf.yaml, not on the contract entity it names"},
	{"RecipeInstall", claimScopeConfigFile, "converges on the validator already declared in aiwf.yaml"},
	{"RenameArea", claimScopeConfigFile, "converges on the area vocabulary declared in aiwf.yaml; the entity retags follow from it rather than deciding it"},

	// Sweep deciders: no target entity, and the comparison is per
	// candidate move rather than per verb.
	{"Archive", claimScopeSweepDeciders, "each candidate move is declined when a file its verdict rests on is mid-edit — the entity's own file, anything beneath it for a directory-shaped kind, or an entity whose committed body links into the move. The link case is why the comparison reads HEAD rather than the working copy: a draft that dropped the link is invisible to a working-copy scan, and the resulting move commits a reference to a path absent at HEAD that no later run can repair, since an archived target leaves the scan for good. An entity terminal at HEAD but not on disk never becomes a candidate, so it is reported rather than passed over in silence"},

	// None: convergence rests on something a working copy cannot
	// contradict, or on a comparison the verb already makes itself.
	{"Rewidth", claimScopeNone, "the converging path writes nothing, and a masked rewrite is re-emitted by the next run because planRewidthRewrites rescans every active markdown independently of the rename set — so the cost is a rewrite deferred, not lost. The direction that would launder is carried by a move Apply guards"},
	{"AcknowledgeIllegal", claimScopeNone, "ackAlreadyRecorded walks git history, so its baseline is already the record rather than the working copy"},
	{"editBodyExplicit", claimScopeNone, "explicitBodySettled already compares the requested content against HEAD, and the verb exists to commit a divergent working copy — a guard refusing divergence would block the route every other refusal recommends"},
}

// noOpResultPositions returns the position of every Result{NoOp: true}
// composite literal in fn's body, in source order.
//
// It is the one place that decides what a same-state NoOp construction
// looks like. The ledger's key set below is derived from it, and
// PolicyClaimGuardPresence measures guard placement against it, so the
// two cannot disagree about which sites converge — a disagreement would
// let a site be required to carry a guard by one policy and be invisible
// to the other.
func noOpResultPositions(fn *ast.FuncDecl) []token.Pos {
	var out []token.Pos
	ast.Inspect(fn.Body, func(n ast.Node) bool {
		lit, ok := n.(*ast.CompositeLit)
		if !ok {
			return true
		}
		if name, ok := lit.Type.(*ast.Ident); !ok || name.Name != "Result" {
			return true
		}
		for _, elt := range lit.Elts {
			kv, ok := elt.(*ast.KeyValueExpr)
			if !ok {
				continue
			}
			key, ok := kv.Key.(*ast.Ident)
			if !ok || key.Name != "NoOp" {
				continue
			}
			if v, ok := kv.Value.(*ast.Ident); ok && v.Name == "true" {
				out = append(out, lit.Pos())
			}
		}
		return true
	})
	return out
}

// validateClaimScopes checks the ledger's own shape and returns the
// recorded scope per function. Split out so the malformed-entry arms are
// testable without a filesystem: the real ledger is well-formed, and a
// policy whose validation only runs against well-formed input has never
// been exercised.
func validateClaimScopes(ledger []claimScope) (map[string]string, []Violation) {
	recorded := make(map[string]string, len(ledger))
	var out []Violation
	for _, entry := range ledger {
		switch entry.Scope {
		case claimScopeTargetEntity, claimScopeConfigFile, claimScopeSweepDeciders, claimScopeNone:
		default:
			out = append(out, Violation{
				Policy: "noop-claim-scope",
				File:   "internal/policies/noop_claim_scope.go",
				Detail: fmt.Sprintf("%s records scope %q, which is outside the closed set — add the scope deliberately or correct the entry",
					entry.Func, entry.Scope),
			})
		}
		if strings.TrimSpace(entry.Reason) == "" {
			out = append(out, Violation{
				Policy: "noop-claim-scope",
				File:   "internal/policies/noop_claim_scope.go",
				Detail: fmt.Sprintf("%s records a claim scope with no reason; the reason is what separates a decision from a default",
					entry.Func),
			})
		}
		recorded[entry.Func] = entry.Scope
	}
	return recorded, out
}

// PolicyNoOpClaimScope asserts that every internal/verb function
// constructing a same-state NoOp has a recorded claim scope, and that no
// recorded scope names a function that no longer converges.
//
// Pins M-0284/AC-2's mechanical half. Its reach is bounded: it proves a
// scope is *recorded*, never that the scope is the right one for that
// claim, and never that the guard is wired to it. A verb could record
// `target-entity` and pass no paths at all — the ledger records names,
// and the paths a guard receives are an expression the scan does not
// evaluate. That judgement is the wrap review's.
func PolicyNoOpClaimScope(root string) ([]Violation, error) {
	files, err := WalkGoFiles(root, true)
	if err != nil { //coverage:ignore defensive: WalkGoFiles errors only when the scan root is unreadable, which every other policy in this package would fail on first
		return nil, err
	}

	recorded, out := validateClaimScopes(noOpClaimScopes)

	type site struct {
		fn   string
		file string
		line int
	}

	fset := token.NewFileSet()
	var sites []site
	converging := map[string]bool{}
	for _, f := range files {
		if !strings.HasPrefix(f.Path, "internal/verb/") || strings.HasSuffix(f.Path, "_test.go") {
			continue
		}
		astFile, perr := parser.ParseFile(fset, f.AbsPath, f.Contents, parser.AllErrors)
		if perr != nil { //coverage:ignore defensive: the tree under scan compiles, so a parse failure needs a file edited mid-run
			continue
		}
		for _, decl := range astFile.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			// Methods are in scope here and out of scope in
			// PolicyClaimGuardPresence, and the asymmetry is load-bearing
			// rather than an oversight: a converging method gets a
			// required ledger row from this scan, and that row is what
			// the sibling policy reports as unvouchable. Filtering
			// fn.Recv here for consistency with the neighbouring scans
			// would delete the interlock and leave both policies green
			// over an unguarded method. Pinned by
			// TestPolicyNoOpClaimScope_ConvergingMethodDemandsARow.
			if !ok || fn.Body == nil {
				continue
			}
			for _, pos := range noOpResultPositions(fn) {
				sites = append(sites, site{fn: fn.Name.Name, file: f.Path, line: fset.Position(pos).Line})
				converging[fn.Name.Name] = true
			}
		}
	}

	// Fail closed. No sites means the scan found no verb package, not
	// that every site is recorded — the difference matters, because the
	// second reads as a pass.
	if len(sites) == 0 {
		return append(out, Violation{
			Policy: "noop-claim-scope",
			File:   "internal/verb",
			Detail: "no same-state NoOp construction found under internal/verb — the policy is scanning nothing, which is not the same as everything being recorded",
		}), nil
	}

	for _, s := range sites {
		if _, ok := recorded[s.fn]; !ok {
			out = append(out, Violation{
				Policy: "noop-claim-scope",
				File:   s.file,
				Line:   s.line,
				Detail: fmt.Sprintf("%s converges with no recorded claim scope — add it to noOpClaimScopes naming what the claim reads (%s, %s, %s) or why nothing can contradict it (%s)",
					s.fn, claimScopeTargetEntity, claimScopeConfigFile, claimScopeSweepDeciders, claimScopeNone),
			})
		}
	}

	var stale []string
	for fn := range recorded {
		if !converging[fn] {
			stale = append(stale, fn)
		}
	}
	sort.Strings(stale)
	for _, fn := range stale {
		out = append(out, Violation{
			Policy: "noop-claim-scope",
			File:   "internal/policies/noop_claim_scope.go",
			Detail: fmt.Sprintf("%s is recorded in noOpClaimScopes but no longer converges to a NoOp — drop the entry, or restore the convergence it describes", fn),
		})
	}
	return out, nil
}
