package policies

import (
	"go/ast"
	"go/parser"
	"go/token"
	"strings"
)

// sovereignDispatcherPrefix scopes the sovereign-dispatcher scan to the
// layer where --actor is parsed and validated. Per-verb dispatchers live
// at internal/cli/<verb>/; the prefix is the whole of internal/cli/, a
// superset that also sweeps the root command and shared helpers, since
// nothing turns on excluding them.
const sovereignDispatcherPrefix = "internal/cli/"

// authorizeDispatcherPrefix names the authorize verb's package. The
// verb is sovereign by what it does rather than by pairing --force with
// --reason, so it is identified rather than pattern-matched: the prefix
// selects the package, and --to selects the dispatcher within it.
const authorizeDispatcherPrefix = "internal/cli/authorize/"

// sovereignForceGuardCall names the shared dispatcher-layer guard. It
// delegates to the same rule function verb.Apply consults, so requiring
// this call by name — rather than any actor comparison — is what keeps
// one implementation of "who may wield --force" instead of a per-verb
// opinion that can drift from the seam's.
const sovereignForceGuardCall = "RefuseNonHumanSovereignForce"

// guardedElsewhere is a dispatcher package that correctly does not
// carry the shared guard, with the measured reason and the test holding
// that reason true. An exemption whose premise nothing pins is
// indistinguishable from a dispatcher someone forgot.
type guardedElsewhere struct {
	Prefix   string
	Why      string
	PinnedBy string
}

// dispatchersGuardedElsewhere are the dispatchers for which a
// flag-keyed pre-check at this layer would be wrong rather than merely
// redundant. Both reasons were measured against the shipped code while
// re-aiming this policy; neither is a standing excuse.
var dispatchersGuardedElsewhere = []guardedElsewhere{
	{
		Prefix: "internal/cli/add/",
		Why: "--force is conditional here: it bypasses the born-complete body gate, and the " +
			"verb stamps the force trailer only when the flag actually bypassed something. " +
			"The flag is inert on epic and milestone, which have no such gate, and on a " +
			"born-complete kind whose body was already non-empty. A pre-check keyed on the " +
			"flag would refuse those invocations, which emit no force trailer and are " +
			"accepted today from any actor. The apply seam guards this verb, judging the " +
			"trailer's presence rather than the flag's",
		PinnedBy: "TestAddForceIsInertWithoutAGateToBypass (internal/verb)",
	},
	{
		Prefix: "internal/cli/authorize/",
		Why: "the verb refuses every non-human actor, forced or not — strictly stronger than " +
			"this guard, which fires only on --force. A pre-check here preempts that refusal " +
			"with a weaker message and a different exit class, so the operator learns their " +
			"force trailer was rejected rather than that they may not authorize at all",
		PinnedBy: "TestRunAuthorize_RefusesNonHumanActor (internal/cli/integration)",
	},
}

// sovereignDispatcher is one CLI dispatcher that parses a sovereign
// act, with the trigger that identified it and whether its package
// calls the human-actor guard.
type sovereignDispatcher struct {
	File    string
	Line    int
	Func    string
	Trigger string
	Guarded bool
	// Exempt marks a dispatcher whose --force is conditional, so the
	// guard does not belong at this layer for it. Carried on the record
	// rather than filtered out of the scan, so the dispatcher stays
	// visible to callers auditing what the scan reaches.
	Exempt bool
}

// sovereignDispatchers returns every dispatcher under
// sovereignDispatcherPrefix that parses a sovereign act.
//
// "Sovereign" is identified by a structural pattern: the dispatcher
// declares both `--force` AND `--reason` flags (the FSM-bypass
// override), OR it declares `--audit-only`, OR it is the authorize
// verb's dispatcher.
//
// Dispatchers that declare `--force` without `--reason` (e.g. `aiwf
// contract bind --force` for force-replace) are deliberately not
// sovereign — that's a different concept of "force". The pairing with
// `--reason` is the kernel's signal for the sovereign-FSM-bypass
// meaning.
//
// Guardedness is decided per package, not per function. The flags are
// declared in the Cobra constructor, where no actor exists yet; the
// guard runs in the verb's Run function, after the prelude resolves
// one. A per-function predicate could not be satisfied by a
// correctly-placed guard at all.
//
// Returned separately from the policy so a test can assert the scanned
// prefix still holds dispatchers: a scope emptied by a relocation makes
// the policy vacuous, and a vacuous policy reports success.
func sovereignDispatchers(root string) ([]sovereignDispatcher, error) {
	files, err := WalkGoFiles(root, true)
	if err != nil {
		return nil, err
	}
	fset := token.NewFileSet()

	// Parse once, then read the result twice. Guardedness is a property
	// of the package, so it cannot be decided while walking a single
	// file — but re-walking to learn it would parse every dispatcher
	// twice and duplicate this loop's skip conditions.
	type parsedFile struct {
		path string
		src  []byte
		ast  *ast.File
	}
	var parsed []parsedFile
	guardedPkgs := map[string]bool{}
	for _, f := range files {
		if !strings.HasPrefix(f.Path, sovereignDispatcherPrefix) {
			continue
		}
		astFile, perr := parser.ParseFile(fset, f.AbsPath, f.Contents, parser.AllErrors)
		if perr != nil {
			continue
		}
		parsed = append(parsed, parsedFile{path: f.Path, src: f.Contents, ast: astFile})
		if callsSovereignForceGuard(astFile) {
			guardedPkgs[goPackageDir(f.Path)] = true
		}
	}

	var out []sovereignDispatcher
	for _, f := range parsed {
		for _, decl := range f.ast.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Body == nil {
				continue
			}
			start := fset.Position(fn.Body.Lbrace).Offset
			end := fset.Position(fn.Body.Rbrace).Offset
			if start < 0 || end <= start || end > len(f.src) { //coverage:ignore defensive: the brace positions come from the same ParseFile call that was handed f.src, so they always resolve inside it
				continue
			}
			body := string(f.src[start:end])

			// Sovereign trigger A: --force paired with --reason.
			triggerForceReason := strings.Contains(body, `"force"`) &&
				strings.Contains(body, `"reason"`)

			// Sovereign trigger B: --audit-only flag declared.
			triggerAuditOnly := strings.Contains(body, `"audit-only"`)

			// Sovereign trigger C: the authorize verb's dispatcher. The
			// package path names the verb and --to picks the dispatcher
			// out of the package, since the Cobra constructor's name is
			// shared with every other verb and identifies nothing.
			triggerAuthorize := strings.HasPrefix(f.path, authorizeDispatcherPrefix) &&
				strings.Contains(body, `"to"`)

			var trigger string
			switch {
			case triggerAuthorize:
				trigger = "is the authorize dispatcher"
			case triggerAuditOnly:
				trigger = "declares --audit-only"
			case triggerForceReason:
				trigger = "declares --force + --reason"
			default:
				continue
			}

			out = append(out, sovereignDispatcher{
				File:    f.path,
				Line:    fset.Position(fn.Pos()).Line,
				Func:    fn.Name.Name,
				Trigger: trigger,
				Guarded: guardedPkgs[goPackageDir(f.path)],
				Exempt:  isGuardedElsewhere(f.path),
			})
		}
	}
	return out, nil
}

// callsSovereignForceGuard reports whether this file calls the shared
// guard.
//
// Answered from call expressions rather than from the file's text: the
// predicate this replaces was a substring search over function bodies,
// which every dispatcher satisfied through a flag-help string naming
// the actor shape (G-0534). A call expression is not something a
// string literal or a comment can be mistaken for.
func callsSovereignForceGuard(astFile *ast.File) bool {
	found := false
	ast.Inspect(astFile, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		switch fn := call.Fun.(type) {
		case *ast.SelectorExpr:
			if fn.Sel.Name == sovereignForceGuardCall {
				found = true
			}
		case *ast.Ident:
			if fn.Name == sovereignForceGuardCall {
				found = true
			}
		}
		return !found
	})
	return found
}

// goPackageDir returns the slash-terminated directory of a
// repo-relative Go file path, which is its package for this scan.
func goPackageDir(path string) string {
	idx := strings.LastIndex(path, "/")
	if idx < 0 {
		return ""
	}
	return path[:idx+1]
}

// isGuardedElsewhere reports whether path lives in a dispatcher package
// whose guard correctly sits at another layer.
func isGuardedElsewhere(path string) bool {
	for _, e := range dispatchersGuardedElsewhere {
		if strings.HasPrefix(path, e.Prefix) {
			return true
		}
	}
	return false
}

// PolicySovereignDispatchersGuardHumanActor asserts that every CLI
// dispatcher package which parses a sovereign act calls the shared
// dispatcher-layer guard, in code.
//
// Scope is the dispatcher layer because that is where --actor is parsed
// and where a refusal costs neither the repo lock nor a tree load. The
// guard it requires delegates to the rule verb.Apply enforces at the
// commit seam, so this layer adds a moment rather than an opinion: the
// seam remains authoritative and no second notion of who may wield the
// flag can drift into being.
//
// The predicate is a call-expression scan. Its predecessor searched
// function-body text for the actor-shape prefix and was therefore
// satisfied by a flag-help string, which every sovereign dispatcher
// carries — measured in G-0534, where all four passed on help text and
// no guard at all.
//
// A dispatcher whose guard correctly sits at another layer is exempt;
// see dispatchersGuardedElsewhere for which, why, and what pins each
// reason true.
func PolicySovereignDispatchersGuardHumanActor(root string) ([]Violation, error) {
	dispatchers, err := sovereignDispatchers(root)
	if err != nil {
		return nil, err
	}
	var out []Violation
	for _, d := range dispatchers {
		if d.Guarded || d.Exempt {
			continue
		}
		out = append(out, Violation{
			Policy: "sovereign-dispatchers-guard-human-actor",
			File:   d.File,
			Line:   d.Line,
			Detail: d.Func + " " + d.Trigger + " but its package never calls " +
				sovereignForceGuardCall + "; call it right after the prelude resolves the " +
				"actor, so a non-human --force is refused before the repo lock and the tree " +
				"load. If this verb's guard belongs at another layer instead — its --force " +
				"does not by itself produce a force trailer, or it already refuses non-human " +
				"actors outright — add the package to dispatchersGuardedElsewhere with the " +
				"reason and the test that pins it",
		})
	}
	return out, nil
}
