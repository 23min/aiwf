package policies

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"os/exec"
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
//   - AIWF_COVERAGE_BASE — the git ref to diff the working tree against. An empty or
//     all-zero value (the default in the broad `go test ./...` job, and a
//     brand-new branch's github.event.before) means "no comparison point"
//     and the audit no-ops. The authoritative invocations are the dedicated
//     CI coverage-gate step and `make coverage-gate`.
//
// Only lines the diff adds or modifies are scanned, so untouched code never
// blocks a change — which is what makes the rule adoptable on a tree that
// already carries the pattern in a few hundred places.
//
// The diff is taken against the working tree, which is the tree the scan
// then parses, so a reported line number always points at the text it
// names. Where the tree is clean — CI, and the push boundary — that tree
// is HEAD. This matches PolicyBranchCoverageAudit's readSourceLines.
func PolicyCommentHistoryAttrition(root string) ([]Violation, error) {
	return commentHistoryViolations(root, strings.TrimSpace(os.Getenv("AIWF_COVERAGE_BASE")))
}

// PolicyCommentHistoryAttritionTree is the whole-tree gate: the same scan
// with the empty tree as its base, so `git diff` presents every tracked Go
// file as newly added and every line lands in the changed set. One matcher,
// one phrase set, one escape convention — the two entry points differ only
// in which base ref they pass.
//
// It takes no environment, so it runs in the ordinary policy suite
// (`make check-fast`, `make ci`, CI) and blocks any offending comment
// anywhere in the tree. The diff-scoped sibling stays because it answers a
// different question — "did *this change* introduce one" — cheaply enough
// for the push boundary.
func PolicyCommentHistoryAttritionTree(root string) ([]Violation, error) {
	base, err := emptyTreeOID(root)
	if err != nil { //coverage:ignore emptyTreeOID hashes without consulting a repository, so it fails only if git is absent from PATH — not deterministically reachable from a test.
		return nil, err
	}
	return commentHistoryViolations(root, base)
}

// emptyTreeOID returns the empty-tree object id. It is asked of git rather
// than hardcoded so the value is correct for the repository's object format —
// the well-known 4b825dc… constant is the SHA-1 spelling, and a SHA-256 repo
// has a different one. Hashing needs no repository, so this succeeds in any
// directory; a wrong-format id would fail loudly at the `git diff` in
// changedLines rather than silently scanning nothing.
func emptyTreeOID(root string) (string, error) {
	cmd := exec.Command("git", "hash-object", "-t", "tree", os.DevNull)
	cmd.Dir = root
	out, err := cmd.CombinedOutput()
	if err != nil { //coverage:ignore hashing a fixed input needs no repository and no network; reaching this requires git to be absent from PATH.
		return "", fmt.Errorf("resolving the empty tree in %s: %w\n%s", root, err, out)
	}
	return strings.TrimSpace(string(out)), nil
}

// historyOKMarker is the deliberate-exception escape, mirroring
// //coverage:ignore. It is directive-shaped (no space after the slashes) so
// gofmt leaves it alone, and hasDirectiveComment requires it to open the
// comment it escapes.
//
// It exempts the whole comment group it appears in — a legacy-format note is
// usually a paragraph, not one line. That is where it parts company with the
// sibling //exec:ok, which reaches no further than the line it annotates and,
// where it stands on its own line, the call below it: //exec:ok annotates a
// call that follows the comment, so extending it further would cover an
// unannotated second call, while //history:ok annotates the prose it sits in,
// whose unit is the paragraph.
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
// Go lines between baseRef and the working tree, parses each touched file
// for the comment text on those lines, and delegates the decision to the
// pure detector. Reading the same tree the diff is taken against is what
// keeps the reported line numbers pointing at the text they name.
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
				Detail: fmt.Sprintf("comment narrates history (%q): a guard is documented by what it guarantees, not by the defect that motivated it — git blame and the tracked entity already own that, and the comment is the copy nothing checks. If the past state is one a reader can still encounter (a legacy on-disk format, a supported older release, an external contract), state it in the present tense as current truth about the input space; otherwise drop the clause. Deliberate exception: `//%s <reason>`, which exempts the whole comment group it sits in. The marker must open the comment it escapes, so write it as its own line in the group, or open a trailing comment with it — appending it to the end of an existing comment does nothing.", phrase, historyOKMarker),
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
		// The escape is looked for on whole comments rather than on the
		// flattened source lines, so a marker on an interior line of a /* */
		// block is text rather than a directive. Scanning every comment in
		// the group, not just the changed ones, is what lets an untouched
		// directive keep covering a line the diff edits.
		exempt := false
		for _, c := range grp.List {
			if hasDirectiveComment(c.Text, historyOKMarker) {
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
