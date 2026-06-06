package policies

import (
	"go/ast"
	"go/parser"
	"go/token"
	"regexp"
	"strconv"
	"strings"
)

// PolicyTrailerKeysViaConstants asserts that no production Go file
// outside the gitops package contains a string literal whose value
// exactly matches a defined gitops.Trailer* constant. The gitops
// package is the canonical home for trailer name constants
// (TrailerVerb, TrailerEntity, …); every other package must
// reference those constants by symbol so a future rename is a
// single edit, not a grep-and-pray.
//
// Why exact-match (not a regex over `aiwf-*`)? Skill directory
// names (`aiwf-add`, `aiwf-history`) and other framework markers
// share the prefix shape but are not trailers — flagging them
// would be noise. The set of "names that are trailers" is
// authoritatively the gitops constants; we read it at scan time.
//
// Why an AST walk (not a regex over file bytes)? A regex over text
// can't track lexical state — it pairs the closing `"` of one
// string with the opening `"` of the next, silently skipping the
// real literals in between. Pre-G-0231 the regex `"([^"\\]*)"`
// produced zero violations on a file that literally contained
// `"aiwf-verb"` and `"aiwf-actor"`. Parsing with go/parser and
// filtering on `*ast.BasicLit` of kind STRING is the correct shape
// and is what every other string-literal-scanning policy in this
// package does (enum_literal_adoption, finding_code_adoption, …).
// The positive-control test
// `TestPolicy_TrailerKeysViaConstants_PositiveControl` synthesizes
// a known violation and asserts the policy reports it, so the next
// regression is caught immediately.
//
// Test files are exempt: tests legitimately synthesize commit
// messages from literal trailer names.
func PolicyTrailerKeysViaConstants(root string) ([]Violation, error) {
	files, err := WalkGoFiles(root, true)
	if err != nil {
		return nil, err
	}
	trailerNames := loadGitopsTrailerNames(files)
	if len(trailerNames) == 0 {
		// No constants found — likely the gitops file moved. Surface
		// as a self-policy violation so the policy stays in sync.
		return []Violation{{
			Policy: "trailer-keys-via-constants",
			File:   "internal/gitops/trailers.go",
			Detail: "no Trailer* string constants found; policy cannot scan for literal misuses",
		}}, nil
	}
	var out []Violation
	fset := token.NewFileSet()
	for _, f := range files {
		if strings.HasPrefix(f.Path, "internal/gitops/") {
			continue
		}
		astFile, perr := parser.ParseFile(fset, f.Path, f.Contents, parser.SkipObjectResolution)
		if perr != nil {
			// Unparseable source: skip rather than fail. Build CI
			// catches compile errors; this policy is correctness for
			// well-formed sources only.
			continue
		}
		ast.Inspect(astFile, func(n ast.Node) bool {
			lit, ok := n.(*ast.BasicLit)
			if !ok || lit.Kind != token.STRING {
				return true
			}
			val, uerr := strconv.Unquote(lit.Value)
			if uerr != nil {
				return true
			}
			if _, hit := trailerNames[val]; !hit {
				return true
			}
			out = append(out, Violation{
				Policy: "trailer-keys-via-constants",
				File:   f.Path,
				Line:   fset.Position(lit.Pos()).Line,
				Detail: "literal \"" + val +
					"\" — reference the gitops.Trailer* constant instead",
			})
			return true
		})
	}
	return out, nil
}

// trailerConstPattern matches `Trailer<Name> = "<value>"` lines in
// the gitops package's const blocks.
var trailerConstPattern = regexp.MustCompile(`(?m)^\s*Trailer[A-Za-z]+\s*=\s*"([^"]+)"`)

// loadGitopsTrailerNames reads the gitops trailers.go file and
// returns the set of string-valued Trailer* constants. Returns an
// empty map when the file isn't where we expect — caller surfaces
// the structural drift.
func loadGitopsTrailerNames(files []FileEntry) map[string]struct{} {
	out := map[string]struct{}{}
	for _, f := range files {
		if !strings.HasPrefix(f.Path, "internal/gitops/") {
			continue
		}
		matches := trailerConstPattern.FindAllSubmatch(f.Contents, -1)
		for _, m := range matches {
			if len(m) < 2 {
				continue
			}
			val, err := strconv.Unquote(`"` + string(m[1]) + `"`)
			if err != nil {
				val = string(m[1])
			}
			out[val] = struct{}{}
		}
	}
	return out
}
