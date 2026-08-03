package check

// M-0289: doc-id-width rule.
//
// The third member of the id-shape family, and the only one whose subject is
// width alone. body-prose-id walks ENTITY bodies, where a real id is required
// and a placeholder is the defect. skill-body-id walks SHIPPED surfaces, where
// a real id is the defect and a canonical placeholder is correct. This rule
// walks a repo's own documentation, where BOTH are correct — a doc is read in
// the repo that owns the ids, so citing one is legitimate, and teaching with a
// placeholder is legitimate too. What is never correct is either shape written
// narrower than any allocator emits.
//
// Code constructs are in scope. Measured over the corpus this rule was built
// for, the debris concentrates in command examples and fenced blocks rather
// than in sentences, so a mask that exempted code would see almost none of it.
// A reader copies a command example at least as readily as a sentence, which
// is why backticks are not an opt-out here — the same call skill-body-id makes,
// and the opposite of body-prose-id, where backticks are the sanctioned way to
// discuss id syntax without citing an entity.
//
// A markdown link's destination is masked along with the rest of the
// non-prose link carriers, so an id there is silent. That carve-out is
// inherited from the shared mask rather than chosen here; the corpus
// carries no id-bearing destinations today.
//
// Severity is advisory by default and escalated by config, never the reverse.
// A repo whose entities were migrated to canonical width still carries narrow
// ids throughout its prose — the migration verb never touched docs — so an
// error-by-default rule would block pushes on upgrade, over files the operator
// never edited, with neither a fixer nor a suppression mechanism to reach for.
// The blocking behavior is what a repo opts into once its own sweep is done.

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/23min/aiwf/internal/entity"
	"github.com/23min/aiwf/internal/pathutil"
	"github.com/23min/aiwf/internal/tree"
)

// DocIDWidthReference scans each configured documentation path under the tree
// root and returns its id-width findings. Paths are repo-relative and come
// from `docs.paths`, defaulting to README.md.
//
// A path that does not resolve to a readable file is skipped rather than
// reported. The default names README.md, which not every repo carries, and a
// missing document is not an id defect — reporting it would turn the default
// into a nag for repos the rule has nothing to say about.
//
// A path escaping the repo root is skipped too: config is operator-supplied
// text, and a rule that reads through `../` would let a checked-in aiwf.yaml
// direct the scan at arbitrary files on the machine running it.
func DocIDWidthReference(t *tree.Tree, paths []string) []Finding {
	var findings []Finding
	for _, rel := range paths {
		raw, ok := readDocUnderRoot(t.Root, rel)
		if !ok {
			continue
		}
		findings = append(findings, ScanDocIDWidth(raw, rel)...)
	}
	return findings
}

// readDocUnderRoot reads one repo-relative documentation path, reporting false
// when it does not resolve to a readable file inside root. Both doc rules read
// their corpus through here so the containment guard cannot be honored by one
// and forgotten by the other.
func readDocUnderRoot(root, rel string) ([]byte, bool) {
	// Resolve before Inside: the check must hold against the path the OS will
	// actually open, not its lexical form. An actor who can commit aiwf.yaml
	// can commit a symlink too, so a purely lexical guard does not constrain
	// the actor it names — the scan would follow the link out of the repo and
	// quote what it found back in the finding. Resolve returns the lexical
	// form for a path that does not exist, so the missing-file skip below is
	// unaffected. Same order as internal/contractconfig.
	full, err := pathutil.Resolve(filepath.Join(root, filepath.FromSlash(rel)))
	if err != nil || !pathutil.Inside(root, full) {
		return nil, false
	}
	raw, err := os.ReadFile(full)
	if err != nil {
		return nil, false
	}
	return raw, true
}

// ScanDocIDWidth returns one finding per unique below-canonical-width id shape
// in content, deduped within it. Both populations fire: a real id at a legacy
// numeric width, and a letter-N placeholder narrower than the canonical form.
//
// Findings are warnings. ApplyDocsStrict is the only thing that raises them,
// so the rule itself stays config-agnostic and independently testable — the
// seam ApplyTDDStrict and ApplyAreaRequiredStrict already establish.
//
// Non-prose content is masked (not stripped) via proseAndCodeMask so byte
// offsets stay stable; Line is 1-based within content.
func ScanDocIDWidth(content []byte, path string) []Finding {
	masked := proseAndCodeMask(content)
	var findings []Finding
	seen := map[string]bool{}
	for _, m := range idTokenPattern.FindAllStringIndex(masked, -1) {
		tok := masked[m[0]:m[1]]
		msg := docIDWidthMessage(tok)
		if msg == "" || seen[tok] {
			continue
		}
		seen[tok] = true
		findings = append(findings, Finding{
			Code:     CodeDocIDWidth,
			Severity: SeverityWarning,
			Message:  msg,
			Path:     path,
			Line:     1 + strings.Count(masked[:m[0]], "\n"),
			Field:    "body",
		})
	}
	return findings
}

// docIDWidthMessage classifies one id-shaped token and returns the finding
// message, or "" when the token carries no width defect. Empty-means-clean
// mirrors the sibling classifiers.
//
// Only the parent segment of a composite carries a width claim: the AC segment
// is a single digit by grammar, so `M-0001/AC-1` is canonical despite its
// one-digit tail.
//
// Anything that is neither all-digits nor all-N is silent here, and nothing
// else catches it either: body-prose-id walks entities, and a document is not
// one. So a malformed shape in a doc goes unreported — a deliberate scope
// limit, not a hand-off, and worth stating as such so the gap stays visible
// rather than looking covered.
func docIDWidthMessage(tok string) string {
	parent, tail := tok, ""
	if i := strings.Index(tok, "/"); i >= 0 {
		parent, tail = tok[:i], tok[i:]
	}
	dash := strings.LastIndex(parent, "-")
	if dash < 0 {
		//coverage:ignore unreachable: idTokenPattern only matches tokens containing a '-'.
		return ""
	}
	prefix, suffix := parent[:dash+1], parent[dash+1:]
	if len(suffix) >= entity.CanonicalPad {
		// At or above canonical width. Wider is legitimate rather than
		// merely tolerated: CanonicalPad is a minimum, so a tree that
		// allocates past its four-digit range emits five digits correctly.
		return ""
	}
	pad := strings.Repeat("0", entity.CanonicalPad-len(suffix))
	switch {
	case isAllDigits(suffix):
		// Both remediations are offered because the defect does not say
		// which applies: a narrow id naming a real entity wants widening,
		// while one invented for a worked example wants the placeholder.
		// Widening an illustrative id is the worse error of the two — it
		// turns fiction into a citation of whatever entity now holds that
		// number — so the choice belongs to the author, not the rule.
		return fmt.Sprintf("doc cites id %q below canonical width — write %s%s%s if it names a real entity, or %sNNNN%s if it is illustrative",
			tok, prefix, pad+suffix, tail, prefix, tail)
	case isAllN(suffix):
		return fmt.Sprintf("doc uses narrow placeholder %q — write the canonical letter-N form %sNNNN%s, which is the id width a reader should learn",
			tok, prefix, tail)
	}
	return ""
}

// isAllDigits reports whether s is non-empty and entirely ASCII digits.
func isAllDigits(s string) bool {
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return s != ""
}

// isAllN reports whether s is non-empty and entirely uppercase 'N' — the
// letter-N placeholder alphabet.
func isAllN(s string) bool {
	for _, r := range s {
		if r != 'N' {
			return false
		}
	}
	return s != ""
}

// ApplyDocsStrict raises the doc rules' findings from warning to error when
// strict is true, mutating the slice in place. Scoped to the doc codes; every
// other finding passes through untouched.
//
// Both doc rules escalate together because they share a corpus: a repo that
// swept its docs swept them for both, and a per-rule knob would only let a
// tree be half-guarded.
//
// Composed at the CLI layer where `docs.strict` is in scope, mirroring
// ApplyTDDStrict and ApplyAreaRequiredStrict.
func ApplyDocsStrict(findings []Finding, strict bool) {
	if !strict {
		return
	}
	for i := range findings {
		switch findings[i].Code {
		case CodeDocIDWidth, CodeDocIDSlug:
			findings[i].Severity = SeverityError
		}
	}
}
