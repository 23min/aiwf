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

// sovereignDispatcher is one CLI dispatcher that parses a sovereign
// act, with the trigger that identified it and whether it references
// the human-actor guard.
type sovereignDispatcher struct {
	File    string
	Line    int
	Func    string
	Trigger string
	Guarded bool
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
// Returned separately from the policy so a test can assert the scanned
// prefix still holds dispatchers: a scope emptied by a relocation makes
// the policy vacuous, and a vacuous policy reports success.
func sovereignDispatchers(root string) ([]sovereignDispatcher, error) {
	files, err := WalkGoFiles(root, true)
	if err != nil {
		return nil, err
	}
	var out []sovereignDispatcher
	fset := token.NewFileSet()
	for _, f := range files {
		if !strings.HasPrefix(f.Path, sovereignDispatcherPrefix) {
			continue
		}
		astFile, perr := parser.ParseFile(fset, f.AbsPath, f.Contents, parser.AllErrors)
		if perr != nil {
			continue
		}
		for _, decl := range astFile.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Body == nil {
				continue
			}
			start := fset.Position(fn.Body.Lbrace).Offset
			end := fset.Position(fn.Body.Rbrace).Offset
			if start < 0 || end <= start || end > len(f.Contents) {
				continue
			}
			body := string(f.Contents[start:end])

			// Sovereign trigger A: --force paired with --reason.
			triggerForceReason := strings.Contains(body, `"force"`) &&
				strings.Contains(body, `"reason"`)

			// Sovereign trigger B: --audit-only flag declared.
			triggerAuditOnly := strings.Contains(body, `"audit-only"`)

			// Sovereign trigger C: the authorize verb's dispatcher. The
			// package path names the verb and --to picks the dispatcher
			// out of the package, since the Cobra constructor's name is
			// shared with every other verb and identifies nothing.
			triggerAuthorize := strings.HasPrefix(f.Path, authorizeDispatcherPrefix) &&
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
				File:    f.Path,
				Line:    fset.Position(fn.Pos()).Line,
				Func:    fn.Name.Name,
				Trigger: trigger,
				Guarded: strings.Contains(body, "human/") ||
					strings.Contains(body, "actorIsNonHuman") ||
					strings.Contains(body, "HasPrefix(actor"),
			})
		}
	}
	return out, nil
}

// PolicySovereignDispatchersGuardHumanActor asserts that every CLI
// dispatcher which parses a sovereign act also references "human/" —
// the actor-shape prefix the kernel gates these acts on.
//
// Scope is the dispatcher level rather than the verb function level
// because the actor is parsed from --actor at that layer and validated
// there.
//
// The guard predicate is satisfied by any occurrence of the actor-shape
// prefix in the function body, including one inside a flag-help string,
// so it detects a dispatcher that never mentions the guard at all
// rather than proving one is enforced in code. Narrowing it to code
// references is tracked as G-0534, which turns on an unsettled question
// about what this layer should assert that internal/verb does not.
func PolicySovereignDispatchersGuardHumanActor(root string) ([]Violation, error) {
	dispatchers, err := sovereignDispatchers(root)
	if err != nil {
		return nil, err
	}
	var out []Violation
	for _, d := range dispatchers {
		if d.Guarded {
			continue
		}
		out = append(out, Violation{
			Policy: "sovereign-dispatchers-guard-human-actor",
			File:   d.File,
			Line:   d.Line,
			Detail: d.Func + " " + d.Trigger +
				" but does not reference \"human/\" or actorIsNonHuman; add the actor refusal guard",
		})
	}
	return out, nil
}
