package policies

import (
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
			File:   "tools/internal/gitops/trailers.go",
			Detail: "no Trailer* string constants found; policy cannot scan for literal misuses",
		}}, nil
	}
	var out []Violation
	for _, f := range files {
		if strings.HasPrefix(f.Path, "tools/internal/gitops/") {
			continue
		}
		// Walk every string literal in the file; flag those whose
		// content is in trailerNames.
		matches := stringLiteralPattern.FindAllSubmatchIndex(f.Contents, -1)
		for _, m := range matches {
			lit := string(f.Contents[m[2]:m[3]])
			if _, ok := trailerNames[lit]; !ok {
				continue
			}
			out = append(out, Violation{
				Policy: "trailer-keys-via-constants",
				File:   f.Path,
				Line:   LineOf(f.Contents, m[0]),
				Detail: "literal \"" + lit +
					"\" — reference the gitops.Trailer* constant instead",
			})
		}
	}
	return out, nil
}

// stringLiteralPattern matches a Go double-quoted string literal
// without trying to handle escape sequences perfectly — good
// enough for the kebab-case trailer names we care about.
var stringLiteralPattern = regexp.MustCompile(`"([^"\\]*)"`)

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
		if !strings.HasPrefix(f.Path, "tools/internal/gitops/") {
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
