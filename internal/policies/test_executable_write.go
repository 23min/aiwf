package policies

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

// PolicyTestExecutableWrite is the diff-scoped guard keeping test
// fixtures off a bare os.WriteFile for an executable stand-in.
//
// The hazard is ETXTBSY: a plain write holds a writable descriptor on
// the new file, a concurrent fork elsewhere in the process inherits it,
// and execve on a file any descriptor holds open for writing fails. In a
// package whose tests spawn subprocesses in parallel the colliding forks
// are the suite's own, so the fixture step fails and the property under
// test is never evaluated — a fixture defect reported as a product
// defect (G-0491). testsupport.WriteExecutable closes the window by
// holding syscall.ForkLock across the write.
//
// It is a Go policy (CI tier), not an `aiwf check` finding, because it
// polices this repo's Go test sources; `aiwf check` validates the
// planning tree in consumer repos, whose source language the kernel
// knows nothing about.
//
// Like PolicyBranchCoverageAudit and PolicySkillEditStructuralTestBackstop
// it is diff-scoped and reads its base ref from the environment, keeping
// the uniform func(root) ([]Violation, error) shape the runPolicy harness
// drives:
//
//   - AIWF_COVERAGE_BASE — the git ref to diff the working tree against.
//     An empty or all-zero value (the default in the broad `go test ./...`
//     job, and a brand-new branch's github.event.before) means "no
//     comparison point" and the audit no-ops. The authoritative
//     invocations are the dedicated CI coverage-gate step and
//     `make coverage-gate`.
//
// It does not also run pre-push, where its sibling comment-history scan
// does. The hook's diff-scoped block is comment-history-specific down to
// its failure message, so sharing it would report an exec-race finding
// under comment-hygiene guidance; a second block wants its own operator
// message and hook-test coverage, which is a change of its own.
//
// Only calls the diff adds or modifies are flagged. The tree carries the
// bare pattern in roughly a hundred places outside internal/stresstest,
// none of them measured as flaking; diff-scoping stops the shape
// spreading by copy without demanding a sweep of all of them first.
func PolicyTestExecutableWrite(root string) ([]Violation, error) {
	return testExecutableWriteViolations(root, strings.TrimSpace(os.Getenv("AIWF_COVERAGE_BASE")))
}

// execOKMarker is the deliberate-exception escape, mirroring
// //coverage:ignore and //history:ok. It is directive-shaped (no space
// after the slashes) so gofmt leaves it alone, and a bare marker is not
// an escape — the reason is the point. The case it exists for is a test
// whose subject *is* the mode, which WriteExecutable's fixed 0755 cannot
// express.
const execOKMarker = "exec:ok"

// executableBits is the mask distinguishing a stand-in meant to be run
// from an ordinary fixture file. Keying on the mask rather than the
// literal 0o755 is what catches the 0o700 spelling, which is the same
// hazard and was already present in the tree.
const executableBits = 0o111

// testExecutableWriteViolations is the testable IO core: it resolves the
// changed Go lines between baseRef and the working tree, then flags each
// os.WriteFile call in a changed _test.go region whose mode argument
// carries an executable bit.
func testExecutableWriteViolations(root, baseRef string) ([]Violation, error) {
	baseRef = strings.TrimSpace(baseRef)
	if baseRef == "" || baseRef == zeroSHA {
		return nil, nil
	}
	changed, err := changedLines(root, baseRef)
	if err != nil {
		return nil, err
	}
	// Untracked test files are in scope for the same reason the coverage
	// audit adds them: a fixture written but not yet committed is exactly
	// the change this is meant to catch, and catching it only after the
	// commit would defeat a gate that runs before one.
	if err := addUntrackedGoLines(root, changed); err != nil { //coverage:ignore unreachable from here: changedLines already ran git against this same repo and returned, so a listing that fails now means git stopped resolving the repo mid-call. addUntrackedGoLines' own error path is unit-tested directly.
		return nil, err
	}

	paths := make([]string, 0, len(changed))
	for rel := range changed {
		if strings.HasSuffix(rel, "_test.go") {
			paths = append(paths, rel)
		}
	}
	sort.Strings(paths)

	var violations []Violation
	for _, rel := range paths {
		violations = append(violations, detectBareExecutableWrites(root, rel, changed[rel])...)
	}
	return violations, nil
}

// detectBareExecutableWrites parses one test file and reports each
// os.WriteFile call carrying an executable mode whose source range
// intersects changedInFile.
//
// Gone from the working tree, or no longer parseable: neither is this
// gate's business. A file that vanished between the diff and this read
// has nothing to audit, and a syntax error is the compiler's finding —
// reporting either here would turn an unrelated breakage into a
// confusing policy failure. Same disposition as the sibling scan in
// comment_history_attrition.go.
func detectBareExecutableWrites(root, rel string, changedInFile map[int]bool) []Violation {
	src, err := os.ReadFile(filepath.Join(root, rel))
	if err != nil {
		return nil
	}
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, rel, src, parser.ParseComments)
	if err != nil {
		return nil
	}
	exempt := exemptLines(fset, file, src)

	var violations []Violation
	ast.Inspect(file, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok || !isOSWriteFile(call) {
			return true
		}
		if len(call.Args) != 3 || !isExecutableMode(call.Args[2]) {
			return true
		}
		start := fset.Position(call.Pos()).Line
		end := fset.Position(call.End()).Line
		if !anyLine(changedInFile, start, end) || anyLine(exempt, start, end) {
			return true
		}
		violations = append(violations, Violation{
			Policy: "test-executable-write",
			File:   rel,
			Line:   start,
			Detail: "os.WriteFile with an executable mode in a test: a concurrent fork can inherit the write descriptor and make the exec fail with ETXTBSY (G-0491). Use testsupport.WriteExecutable. When the mode itself is the subject, annotate with //" + execOKMarker + " <reason>, placed either on its own line directly above the call or trailing the call's opening line.",
		})
		return true
	})
	return violations
}

// isOSWriteFile reports whether the call is a selector call on the `os`
// package named WriteFile. Matching the package identifier rather than
// the bare method name keeps a local helper called WriteFile out of
// scope.
func isOSWriteFile(call *ast.CallExpr) bool {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok || sel.Sel.Name != "WriteFile" {
		return false
	}
	pkg, ok := sel.X.(*ast.Ident)
	return ok && pkg.Name == "os"
}

// isExecutableMode reports whether an argument is an integer literal
// with any executable bit set. A non-literal mode — a named constant or
// a variable — is out of scope: resolving it needs type information this
// AST-only pass does not have, and every site in the tree spells the
// mode inline.
func isExecutableMode(arg ast.Expr) bool {
	lit, ok := arg.(*ast.BasicLit)
	if !ok || lit.Kind != token.INT {
		return false
	}
	mode, err := strconv.ParseInt(strings.ReplaceAll(lit.Value, "_", ""), 0, 32)
	if err != nil {
		return false
	}
	return mode&executableBits != 0
}

// exemptLines returns the line numbers covered by an //exec:ok comment
// carrying a reason.
//
// A comment standing on its own line annotates the line below it, which
// is where the call it introduces begins. A trailing comment annotates
// only the line it sits on — extending it downward would silently exempt
// the next, unannotated call.
func exemptLines(fset *token.FileSet, file *ast.File, src []byte) map[int]bool {
	lines := strings.Split(string(src), "\n")
	exempt := map[int]bool{}
	for _, group := range file.Comments {
		for _, c := range group.List {
			if !hasExecOK(c.Text) {
				continue
			}
			pos := fset.Position(c.Pos())
			exempt[pos.Line] = true
			if ownLine(lines, pos.Line, pos.Column) {
				exempt[pos.Line+1] = true
			}
		}
	}
	return exempt
}

// ownLine reports whether the comment at (line, column) is the first
// thing on its source line, as opposed to trailing some code.
func ownLine(lines []string, line, column int) bool {
	if line < 1 || line > len(lines) {
		return false
	}
	text := lines[line-1]
	if column-1 > len(text) {
		return false
	}
	return strings.TrimSpace(text[:column-1]) == ""
}

// hasExecOK reports whether a raw comment is the escape directive with a
// non-empty reason after it.
//
// The marker must open the comment, directive-style (`//exec:ok why`).
// Matching it anywhere in the text would let prose that merely mentions
// the escape — including this file's own doc comments — silently exempt
// a neighbouring call.
//
// Whitespace must separate the marker from the reason, so that a longer
// word opening with the marker's letters (`//exec:okay`) reads as a
// different comment rather than as the directive with "ay" for a reason.
func hasExecOK(raw string) bool {
	rest, found := strings.CutPrefix(raw, "//"+execOKMarker)
	if !found {
		return false
	}
	reason := strings.TrimLeft(rest, " \t")
	if len(reason) == len(rest) {
		// Nothing separates the marker from what follows: either the
		// comment is the bare marker, or the marker is a prefix of a
		// longer word. Neither is an escape.
		return false
	}
	return strings.TrimSpace(reason) != ""
}

// anyLine reports whether any line in [start,end] is in the set.
func anyLine(set map[int]bool, start, end int) bool {
	for ln := start; ln <= end; ln++ {
		if set[ln] {
			return true
		}
	}
	return false
}
