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

	"github.com/23min/aiwf/internal/check"
)

// planningLabelPattern matches an unhyphenated gap label such as `G24`, a
// spelling no allocator emits and a consumer's tree cannot resolve. Only the
// gap prefix is matched: other letters run into digits (`D5`, `M1`) are too
// often something else to read as a label.
var planningLabelPattern = regexp.MustCompile(`\bG\d+\b`)

// sourcePathPattern matches a relative path starting in one of aiwf's
// top-level source directories, capturing that directory and the segment
// under it. It starts at a token boundary so a module path such as
// `github.com/23min/aiwf/cmd/aiwf` — which a consumer can install from — is
// not read as one.
var sourcePathPattern = regexp.MustCompile(`(?:^|[^\w./-])(?:\.{1,2}/)*(internal|cmd|docs)/([\w.-]*)`)

// PolicyCLITextCarriesNoInternalIDs asserts that no string literal in the
// packages inOperatorTextScope names tells an operator about aiwf's own
// planning tree or source tree. That text reaches a consumer through command
// help, shell completion, doctor output, error messages, the commit messages
// a verb writes into the consumer's history, and the comments aiwf init
// writes into the consumer's aiwf.yaml. In a consumer's repo aiwf's ids name
// nothing — or name the consumer's own, unrelated entity with the same
// number — while aiwf's source paths do not exist.
//
// Three shapes offend:
//   - an id-shaped token, judged by the classification skill-body-id applies
//     to the embedded skills: any digit-bearing id at any width, and any
//     placeholder other than the canonical letter-N form (`E-NNNN`,
//     `M-NNNN/AC-N`), which is the one shape an illustration may use;
//   - an unhyphenated gap label (`G24`);
//   - a path into aiwf's source tree: one under `internal/`, `cmd/` or
//     `docs/` whose first two segments exist in the tree being checked.
//     Existence is what separates aiwf's `internal/cli` from a consumer's
//     `internal/billing` in an areas example; `docs/adr/` is exempt because
//     aiwf writes a consumer's own ADRs there, and a bare top-level
//     directory names nothing in particular.
//
// Only a literal containing whitespace is judged: text an operator reads is
// prose, and a literal without spaces is almost always data — an argument
// passed to a verb, a fixture id, a path the code matches against. The check
// therefore reads what a literal says, not what a sentence becomes: an id
// held in a constant and joined into a message by concatenation or a format
// verb is not seen. Comments are not literals and are not read.
func PolicyCLITextCarriesNoInternalIDs(root string) ([]Violation, error) {
	files, err := WalkGoFiles(root, true)
	if err != nil {
		return nil, err
	}
	var out []Violation
	fset := token.NewFileSet()
	for _, f := range files {
		if !inOperatorTextScope(f.Path) {
			continue
		}
		astFile, perr := parser.ParseFile(fset, f.AbsPath, f.Contents, 0)
		if perr != nil {
			continue
		}
		ast.Inspect(astFile, func(n ast.Node) bool {
			lit, ok := n.(*ast.BasicLit)
			if !ok || lit.Kind != token.STRING {
				return true
			}
			text, uerr := strconv.Unquote(lit.Value)
			if uerr != nil || !strings.ContainsAny(text, " \t\n") {
				return true
			}
			line := fset.Position(lit.Pos()).Line
			// A raw literal spans source lines one for one; an
			// interpreted one keeps its escaped newlines on a single
			// source line, so every token in it reports that line.
			raw := strings.HasPrefix(lit.Value, "`")
			lineAt := func(offset int) int {
				if !raw {
					return line
				}
				return line + strings.Count(text[:offset], "\n")
			}
			findings := check.ScanPlainTextIDs(text, f.Path)
			for i := range findings {
				at := line
				if raw {
					at = line + findings[i].Line - 1
				}
				out = append(out, Violation{
					Policy: "cli-text-internal-id",
					File:   f.Path,
					Line:   at,
					Detail: "operator-facing string: " + findings[i].Message,
				})
			}
			for _, m := range planningLabelPattern.FindAllStringIndex(text, -1) {
				out = append(out, Violation{
					Policy: "cli-text-internal-id",
					File:   f.Path,
					Line:   lineAt(m[0]),
					Detail: "operator-facing string cites planning label " + strconv.Quote(text[m[0]:m[1]]) +
						", which names nothing in a consumer's repo; state the reason in words",
				})
			}
			for _, m := range sourcePathPattern.FindAllStringSubmatchIndex(text, -1) {
				dir, sub := text[m[2]:m[3]], strings.TrimRight(text[m[4]:m[5]], ".")
				if sub == "" || (dir == "docs" && sub == "adr") {
					continue
				}
				if _, serr := os.Stat(filepath.Join(root, dir, sub)); serr != nil {
					continue
				}
				out = append(out, Violation{
					Policy: "cli-text-internal-id",
					File:   f.Path,
					Line:   lineAt(m[2]),
					Detail: "operator-facing string names " + strconv.Quote(dir+"/"+sub) +
						", a path in aiwf's own source tree that a consumer's repo does not contain; describe the thing instead",
				})
			}
			return true
		})
	}
	return out, nil
}

// inOperatorTextScope reports whether a repo-relative Go file holds text an
// operator reads. Command help, doctor output and the CLI's own messages live
// under internal/cli/; the refusals a verb returns and the commit messages it
// writes live under internal/verb/; the config schema's field descriptions,
// which aiwf init writes into aiwf.yaml as comments, and the errors config,
// aiwf.yaml and trailer parsing return live under internal/config/,
// internal/aiwfyaml/ and internal/gitops/. Test support is excluded, since
// only tests import it.
func inOperatorTextScope(path string) bool {
	if strings.Contains(path, "/testutil/") {
		return false
	}
	scopes := [...]string{"internal/cli/", "internal/verb/", "internal/config/", "internal/aiwfyaml/", "internal/gitops/"}
	for _, scope := range scopes {
		if strings.HasPrefix(path, scope) {
			return true
		}
	}
	return false
}
