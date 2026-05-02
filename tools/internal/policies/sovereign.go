package policies

import (
	"go/ast"
	"go/parser"
	"go/token"
	"strings"
)

// PolicySovereignDispatchersGuardHumanActor asserts that every cmd
// dispatcher (in tools/cmd/aiwf/) which parses a sovereign-act
// flag pair also references "human/" — the actor-shape prefix the
// kernel gates these acts on.
//
// "Sovereign" is identified by a structural pattern: the dispatcher
// declares both `--force` AND `--reason` flags (the FSM-bypass
// override), OR it declares `--audit-only` (G24 backfill), OR it
// declares `--to` and is named runAuthorize (the authorize verb).
//
// We deliberately don't fire on dispatchers that only have
// `--force` without `--reason` (e.g. `aiwf contract bind --force`
// for force-replace) — that's a different concept of "force." The
// pairing with `--reason` is the kernel's signal that this is the
// sovereign-FSM-bypass meaning.
//
// Scope is the cmd dispatcher level rather than the verb function
// level because the actor is parsed from --actor at that layer and
// validated there.
func PolicySovereignDispatchersGuardHumanActor(root string) ([]Violation, error) {
	files, err := WalkGoFiles(root, true)
	if err != nil {
		return nil, err
	}
	var out []Violation
	fset := token.NewFileSet()
	for _, f := range files {
		if !strings.HasPrefix(f.Path, "tools/cmd/aiwf/") {
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
			hasForce := strings.Contains(body, `"force"`)
			hasReason := strings.Contains(body, `"reason"`)
			triggerForceReason := hasForce && hasReason

			// Sovereign trigger B: --audit-only flag declared.
			triggerAuditOnly := strings.Contains(body, `"audit-only"`)

			// Sovereign trigger C: this is runAuthorize specifically.
			triggerAuthorize := fn.Name.Name == "runAuthorize"

			if !triggerForceReason && !triggerAuditOnly && !triggerAuthorize {
				continue
			}

			hasGuard := strings.Contains(body, "human/") ||
				strings.Contains(body, "actorIsNonHuman") ||
				strings.Contains(body, "HasPrefix(actor")
			if hasGuard {
				continue
			}

			var triggerDesc string
			switch {
			case triggerAuthorize:
				triggerDesc = "is runAuthorize"
			case triggerAuditOnly:
				triggerDesc = "declares --audit-only"
			default:
				triggerDesc = "declares --force + --reason"
			}
			out = append(out, Violation{
				Policy: "sovereign-dispatchers-guard-human-actor",
				File:   f.Path,
				Line:   fset.Position(fn.Pos()).Line,
				Detail: fn.Name.Name + " " + triggerDesc +
					" but does not reference \"human/\" or actorIsNonHuman; add the actor refusal guard",
			})
		}
	}
	return out, nil
}
