package policies

import (
	"go/ast"
	"go/parser"
	"go/token"
	"strings"
)

// lockDispatcherPrefix scopes the repo-lock scan to the per-verb
// dispatchers. The lock is taken at this layer and at no other —
// internal/verb never acquires it — so this prefix holds the whole
// population of dispatchers the property applies to.
const lockDispatcherPrefix = "internal/cli/"

// lockHelperPrefix is the shared helper layer beneath the dispatchers.
// Its functions run inside a dispatcher that has already taken the
// lock, so requiring them to re-take it would assert the opposite of
// what the lock is for.
const lockHelperPrefix = "internal/cli/cliutil/"

// applyDispatcher is one dispatcher that reaches verb.Apply, with
// whether it takes the repo lock first.
type applyDispatcher struct {
	File   string
	Line   int
	Func   string
	Locked bool
}

// applyReachingDispatchers returns every function under
// lockDispatcherPrefix, helper layer excluded, that reaches verb.Apply
// directly or through cliutil's finish helpers.
//
// Every function under the prefix is a candidate. Selecting them by
// name does not work: a verb's entry point is `Run` and a subverb's is
// `run<Sub>`, so any single prefix silently covers one group and drops
// the other. Membership is the file's package.
//
// Returned separately from the policy so a test can assert the prefix
// still holds dispatchers: a scan over an empty population reports
// success, which is indistinguishable from a scan that found nothing
// wrong.
func applyReachingDispatchers(root string) ([]applyDispatcher, error) {
	files, err := WalkGoFiles(root, true)
	if err != nil {
		return nil, err
	}
	var out []applyDispatcher
	fset := token.NewFileSet()
	for _, f := range files {
		if !strings.HasPrefix(f.Path, lockDispatcherPrefix) ||
			strings.HasPrefix(f.Path, lockHelperPrefix) {
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
			callsApply := strings.Contains(body, "verb.Apply(") ||
				strings.Contains(body, "cliutil.FinishVerb") ||
				strings.Contains(body, "cliutil.DecorateAndFinish")
			if !callsApply {
				continue
			}
			out = append(out, applyDispatcher{
				File:   f.Path,
				Line:   fset.Position(fn.Pos()).Line,
				Func:   fn.Name.Name,
				Locked: strings.Contains(body, "cliutil.AcquireRepoLock"),
			})
		}
	}
	return out, nil
}

// PolicyApplyCallersAcquireLock asserts that every per-verb dispatcher
// reaching `verb.Apply` — directly, or through `cliutil.FinishVerb` /
// `cliutil.DecorateAndFinish` — also calls `cliutil.AcquireRepoLock`.
// Apply is the only path that writes to disk; without the repo lock,
// two concurrent verb invocations could corrupt each other's state.
//
// TestPolicyApplyCallersAcquireLock_ScopeIsNotOrphaned asserts the
// prefix still holds dispatchers of both spellings, and
// TestPolicyApplyCallersAcquireLock_HelperLayerExempt pins the helper
// exclusion.
func PolicyApplyCallersAcquireLock(root string) ([]Violation, error) {
	dispatchers, err := applyReachingDispatchers(root)
	if err != nil {
		return nil, err
	}
	var out []Violation
	for _, d := range dispatchers {
		if d.Locked {
			continue
		}
		out = append(out, Violation{
			Policy: "apply-callers-acquire-lock",
			File:   d.File,
			Line:   d.Line,
			Detail: d.Func +
				" calls verb.Apply (or cliutil.FinishVerb / cliutil.DecorateAndFinish) without cliutil.AcquireRepoLock;" +
				" concurrent invocations could corrupt repo state. If this is a shared helper running beneath a" +
				" dispatcher that already holds the lock, hoist the call or exempt the helper layer — a second" +
				" acquire in the same process is a distinct file description and blocks until the lock timeout",
		})
	}
	return out, nil
}
