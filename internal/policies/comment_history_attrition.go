package policies

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// PolicyCommentHistoryAttrition is the diff-scoped guard against history
// attrition in Go comments: a past-tense clause appended to a guard that
// narrates a defect no reader can encounter. It is the mechanical half of
// the shipped guidance rule "state the conclusion, not the drafting
// history", which holds comments to the same bar as entity bodies and docs.
//
// The discriminator the rule turns on: a past state a reader can still
// encounter — a legacy on-disk format, a supported older release, an
// external contract — is current truth about the input space and belongs in
// the comment, written in the present tense. Everything else is owned by
// git blame and the tracked entity, where nothing lets it go stale.
//
// It is a Go policy (CI tier), not an `aiwf check` finding, because it
// polices this repo's Go source; `aiwf check` validates the planning tree in
// consumer repos, whose source language the kernel knows nothing about.
//
// Like PolicyBranchCoverageAudit and PolicySkillEditStructuralTestBackstop
// it is diff-scoped and reads its base ref from the environment, keeping the
// uniform func(root) ([]Violation, error) shape the runPolicy harness drives:
//
//   - AIWF_COVERAGE_BASE — the git ref to diff HEAD against. An empty or
//     all-zero value (the default in the broad `go test ./...` job, and a
//     brand-new branch's github.event.before) means "no comparison point"
//     and the audit no-ops. The authoritative invocations are the dedicated
//     CI coverage-gate step and `make coverage-gate`.
//
// Only lines the diff adds or modifies are scanned, so untouched code never
// blocks a change — which is what makes the rule adoptable on a tree that
// already carries the pattern in a few hundred places.
//
// Both authoritative invocations compare committed HEAD to the merge-base,
// which is why the scan parses the working-tree file: at those call sites it
// is HEAD. This matches PolicyBranchCoverageAudit's readSourceLines.
func PolicyCommentHistoryAttrition(root string) ([]Violation, error) {
	return commentHistoryViolations(root, strings.TrimSpace(os.Getenv("AIWF_COVERAGE_BASE")))
}

// historyOKMarker is the deliberate-exception escape, mirroring
// //coverage:ignore. It is directive-shaped (no space after the slashes) so
// gofmt leaves it alone, and it exempts the whole comment group it appears
// in — a legacy-format note is usually a paragraph, not one line. A bare
// marker is not an escape: the reason is the point.
const historyOKMarker = "history:ok"

// historyAttritionPhrases are the lower-cased trigger substrings, calibrated
// against this tree for precision: every phrasing here matches only comments
// that genuinely narrate a superseded state, and none of the comments it
// matches today are legitimate.
//
// The set is therefore deliberately under-inclusive. Four classes are held
// out because their historical and legitimate senses are indistinguishable
// without reading the sentence:
//
//   - the bare employed-to form — its purpose sense ("a stub ... to exercise
//     X") is roughly half its occurrences here;
//   - the not-any-more form — as often describes an input a reader still
//     meets on disk, which is current truth about the input space;
//   - the drift form — reads as a present-tense assertion about what an
//     invariant currently guarantees;
//   - naming the defect a guard exists for — legitimate when it names a
//     failure *mode* (the "why" a comment should carry), attrition only when
//     it names a specific past incident.
//
// A gate that never cries wolf and catches the clear cases beats a broad one
// every author learns to suppress; the remainder stays a review-time
// judgment call, the same split the skill-body-id rule already accepts for
// shipped surfaces. Broadening is cheap once the tree is clean, and the
// escape hatch is what makes broadening safe.
var historyAttritionPhrases = []string{
	"used to be",
	"before this fix",
	"earlier version",
	"originally proposed",
	"at one point",
	"we once",
	"prior to this change",
}

// commentLine is one source line of a comment group, carrying the raw text
// (comment markers included) and whether its group is exempted.
type commentLine struct {
	line   int
	text   string
	exempt bool
}

// commentHistoryViolations is the testable IO core: it resolves the changed
// Go lines between baseRef and HEAD, parses each touched file for the
// comment text on those lines, and delegates the decision to the pure
// detector.
func commentHistoryViolations(root, baseRef string) ([]Violation, error) {
	baseRef = strings.TrimSpace(baseRef)
	if baseRef == "" || baseRef == zeroSHA {
		return nil, nil
	}
	changed, err := changedLines(root, baseRef)
	if err != nil {
		return nil, err
	}
	rels := make([]string, 0, len(changed))
	for rel := range changed {
		rels = append(rels, rel)
	}
	sort.Strings(rels)

	var out []Violation
	for _, rel := range rels {
		abs := filepath.Join(root, filepath.FromSlash(rel))
		lines, perr := addedCommentLines(abs, changed[rel])
		if perr != nil {
			// Gone from the working tree, or no longer parseable. Neither is
			// this gate's business — the compiler and the build own that.
			continue
		}
		out = append(out, detectHistoryAttrition(rel, lines)...)
	}
	return out, nil
}

// detectHistoryAttrition is the pure core: given the comment lines a diff
// touched, report the ones narrating history. Keeping the Violation
// construction here (rather than in the IO shell) is what lets a plain table
// test discharge the firing-fixture meta-gate without a git fixture.
func detectHistoryAttrition(rel string, lines []commentLine) []Violation {
	var out []Violation
	for _, cl := range lines {
		if cl.exempt {
			continue
		}
		low := strings.ToLower(cl.text)
		for _, phrase := range historyAttritionPhrases {
			if !strings.Contains(low, phrase) {
				continue
			}
			out = append(out, Violation{
				Policy: "comment-history-attrition",
				File:   rel,
				Line:   cl.line,
				Detail: fmt.Sprintf("comment narrates history (%q): a guard is documented by what it guarantees, not by the defect that motivated it — git blame and the tracked entity already own that, and the comment is the copy nothing checks. If the past state is one a reader can still encounter (a legacy on-disk format, a supported older release, an external contract), state it in the present tense as current truth about the input space; otherwise drop the clause. Deliberate exception: add `//%s <reason>` to the comment.", phrase, historyOKMarker),
			})
			break
		}
	}
	return out
}

// addedCommentLines parses path and returns the comment lines whose line
// number is in changed. Parsing rather than grepping is what keeps string
// literals containing comment markers out of the scan and gets /* */ spans
// right.
func addedCommentLines(path string, changed map[int]bool) ([]commentLine, error) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, path, nil, parser.ParseComments|parser.SkipObjectResolution)
	if err != nil {
		return nil, fmt.Errorf("parsing %s: %w", path, err)
	}
	var out []commentLine
	for _, grp := range f.Comments {
		lines := commentGroupLines(fset, grp)
		exempt := false
		for _, cl := range lines {
			if hasHistoryOK(cl.text) {
				exempt = true
				break
			}
		}
		for _, cl := range lines {
			if !changed[cl.line] {
				continue
			}
			cl.exempt = exempt
			out = append(out, cl)
		}
	}
	return out, nil
}

// commentGroupLines flattens a comment group into one commentLine per source
// line. A //-comment contributes one line; a /* */ comment contributes as
// many as it spans. The raw text is kept (ast.CommentGroup.Text would strip
// the markers and, being directive-shaped, the escape along with them).
func commentGroupLines(fset *token.FileSet, grp *ast.CommentGroup) []commentLine {
	var out []commentLine
	for _, c := range grp.List {
		start := fset.Position(c.Pos()).Line
		for i, raw := range strings.Split(c.Text, "\n") {
			out = append(out, commentLine{line: start + i, text: raw})
		}
	}
	return out
}

// hasHistoryOK reports whether a raw comment line carries the escape marker
// with a non-empty reason after it.
func hasHistoryOK(raw string) bool {
	_, reason, found := strings.Cut(raw, historyOKMarker)
	if !found {
		return false
	}
	return strings.TrimSpace(reason) != ""
}
