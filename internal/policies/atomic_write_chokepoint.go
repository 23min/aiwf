package policies

import (
	"go/ast"
	"go/parser"
	"go/token"
)

// PolicyAtomicWriteChokepoint asserts that production code does not
// write persistent files via os.WriteFile, os.Create, or a
// write-mode os.OpenFile. The kernel pattern (G-0221) is disk-level
// write atomicity: every write to a file that outlives the process
// routes through pathutil.AtomicWriteFile (temp + fsync + rename),
// so an OS crash or hard kill mid-write never leaves a half-written
// file behind.
//
// os.OpenFile counts as a write site only when its flag argument
// names a write-intent flag (O_WRONLY, O_RDWR, O_APPEND, O_CREATE,
// O_TRUNC); a read-only open is fine.
//
// Scope is every non-test Go file the repo walk returns (internal/
// and cmd/; the policies package itself and test files are excluded
// by WalkGoFiles). A new raw write call surfaces here with a finding
// pointing at the offending line. Legitimate exceptions — writes
// whose targets are confined to a process-lifetime temp dir, or
// opens that never write content — are allowlisted by file path with
// a one-line rationale; this is the load-bearing piece that keeps
// the discipline architectural rather than one-of.
//
// Known blind spots, consistent with the repo's other AST policies
// (e.g. verbs_validate_then_write.go): an aliased os import
// (`o "os"`), method-value indirection (`w := os.WriteFile`), and an
// OpenFile flag argument that is a variable rather than an
// expression naming os.O_* constants inline are not matched.
func PolicyAtomicWriteChokepoint(root string) ([]Violation, error) {
	// File-path allowlist. Key is the repo-relative forward-slash
	// path; value is the rationale (kept here so the exemption and
	// its justification travel together).
	allow := map[string]string{
		// Self-check writes its fake home, synthesized repo edits, and
		// aiwf.yaml flips inside os.MkdirTemp sandboxes that are
		// removed on exit — nothing it writes outlives the run.
		"internal/cli/doctor/selfcheck.go": "writes confined to the self-check temp sandbox",
		// The lockfile fd is opened O_RDWR|O_CREATE solely to carry
		// the flock; no file content is ever written through it.
		"internal/repolock/repolock_unix.go": "lockfile fd carries flock only; content is never written",
	}
	files, err := WalkGoFiles(root, true)
	if err != nil {
		return nil, err
	}
	var out []Violation
	fset := token.NewFileSet()
	for _, f := range files {
		if _, ok := allow[f.Path]; ok {
			continue
		}
		astFile, perr := parser.ParseFile(fset, f.AbsPath, f.Contents, parser.AllErrors)
		if perr != nil {
			continue
		}
		ast.Inspect(astFile, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			sel, ok := call.Fun.(*ast.SelectorExpr)
			if !ok {
				return true
			}
			pkg, ok := sel.X.(*ast.Ident)
			if !ok || pkg.Name != "os" {
				return true
			}
			switch sel.Sel.Name {
			case "WriteFile", "Create":
				// Always a write site.
			case "OpenFile":
				if len(call.Args) < 2 || !hasOSWriteFlag(call.Args[1]) {
					return true
				}
			default:
				return true
			}
			out = append(out, Violation{
				Policy: "atomic-write-chokepoint",
				File:   f.Path,
				Line:   fset.Position(call.Pos()).Line,
				Detail: "os." + sel.Sel.Name + " writes a persistent file non-atomically; " +
					"route the write through pathutil.AtomicWriteFile (G-0221) " +
					"or allowlist the file with a rationale",
			})
			return true
		})
	}
	return out, nil
}

// hasOSWriteFlag reports whether the expression (an os.OpenFile flag
// argument, typically a |-chain) references one of the os.O_* flags
// that signal write intent.
func hasOSWriteFlag(e ast.Expr) bool {
	found := false
	ast.Inspect(e, func(n ast.Node) bool {
		sel, ok := n.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		pkg, ok := sel.X.(*ast.Ident)
		if !ok || pkg.Name != "os" {
			return true
		}
		switch sel.Sel.Name {
		case "O_WRONLY", "O_RDWR", "O_APPEND", "O_CREATE", "O_TRUNC":
			found = true
		}
		return true
	})
	return found
}
