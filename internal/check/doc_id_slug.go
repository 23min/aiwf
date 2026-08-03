package check

// M-0289: doc-id-slug rule.
//
// The companion to doc-id-width, closing the case width cannot see. A worked
// example that invents an id at canonical width is indistinguishable from a
// citation — until the slug is written alongside it, at which point the doc
// makes a claim the tree can settle. `ADR-0001-use-oauth-21-with-passkey-
// support.md` is checkable against whatever ADR-0001 actually is; a bare
// `ADR-0001` is not.
//
// The rule is therefore deliberately narrow, and its silences are the design
// rather than gaps in it:
//
//   - A bare id is silent. Whether it cites or invents is the judgment this
//     rule declines to make, because no mechanical signature separates them.
//   - An id naming no entity is silent. Whether documentation should be
//     reference-checked the way entity bodies are is a much larger change,
//     carrying the cross-branch and archived-id questions body-prose-id had to
//     settle; it is tracked on its own rather than smuggled in here.
//
// What remains is exact: the entity's slug is known, so a written slug either
// equals it or does not. No heuristic, and nothing to tune.

import (
	"fmt"
	"path"
	"regexp"
	"strings"

	"github.com/23min/aiwf/internal/entity"
	"github.com/23min/aiwf/internal/tree"
)

// idSlugTokenPattern matches an id followed by a slug: a kind prefix, a
// digit-bearing number, then a hyphen and at least one slug word.
//
// Restricting the number to digits keeps letter-N placeholders out of the scan
// entirely. That is a narrowing, not the guarantee: a placeholder would be
// silent anyway, because it resolves to no index entry and the comparison is
// skipped. Both paths lead to silence, so no test can tell them apart — the
// index lookup is what a reader should trust here.
//
// The slug run stops at any character a slug cannot contain, so a trailing
// `.md`, a closing parenthesis, or sentence punctuation is not swallowed.
var idSlugTokenPattern = regexp.MustCompile(`(?:E|M|G|D|C|ADR)-\d+-[a-z0-9]+(?:-[a-z0-9]+)*`)

// DocSlugIndex maps canonical entity id to the slug the entity's own path
// carries. Built once per check run and handed to the scanner, mirroring
// BodyProseIDIndex's shape.
func DocSlugIndex(t *tree.Tree) map[string]string {
	idx := make(map[string]string, len(t.Entities))
	for _, e := range t.Entities {
		if slug, ok := slugFromEntityPath(e.Path, e.ID); ok {
			idx[entity.Canonicalize(e.ID)] = slug
		}
	}
	return idx
}

// slugFromEntityPath extracts the slug an entity's path encodes — the segment
// after `<id>-`, with any file extension removed. An epic stores its body at
// `<id>-<slug>/epic.md`, so the id-bearing segment is the directory rather
// than the file; scanning every segment finds it either way.
func slugFromEntityPath(relPath, id string) (string, bool) {
	prefix := id + "-"
	for _, seg := range strings.Split(path.Clean(relPath), "/") {
		if !strings.HasPrefix(seg, prefix) {
			continue
		}
		return strings.TrimSuffix(seg[len(prefix):], path.Ext(seg)), true
	}
	return "", false
}

// ScanDocIDSlug returns one finding per unique id-with-slug token in content
// whose slug contradicts the slug its entity carries, deduped within content.
//
// Findings are warnings, escalated with the rest of the doc corpus by
// ApplyDocIDWidthStrict — the two rules share a corpus and a remediation
// posture, so splitting their severity would leave a repo half-blocked.
func ScanDocIDSlug(content []byte, docPath string, idx map[string]string) []Finding {
	masked := proseAndCodeMask(content)
	var findings []Finding
	seen := map[string]bool{}
	for _, m := range idSlugTokenPattern.FindAllStringIndex(masked, -1) {
		tok := masked[m[0]:m[1]]
		id, slug, ok := splitIDSlug(tok)
		if !ok {
			//coverage:ignore unreachable: idSlugTokenPattern only matches tokens carrying both halves.
			continue
		}
		want, known := idx[entity.Canonicalize(id)]
		if !known || want == slug || seen[tok] {
			continue
		}
		seen[tok] = true
		findings = append(findings, Finding{
			Code:     CodeDocIDSlug,
			Severity: SeverityWarning,
			Message: fmt.Sprintf("doc writes %q, but %s is %q — cite the entity by its real slug, or use the canonical placeholder if the example is invented",
				tok, id, id+"-"+want),
			Path:  docPath,
			Line:  1 + strings.Count(masked[:m[0]], "\n"),
			Field: "body",
		})
	}
	return findings
}

// splitIDSlug divides an id-with-slug token at the hyphen ending the number.
func splitIDSlug(tok string) (id, slug string, ok bool) {
	dash := strings.Index(tok, "-")
	if dash < 0 {
		//coverage:ignore unreachable: idSlugTokenPattern requires a prefix hyphen.
		return "", "", false
	}
	rest := tok[dash+1:]
	end := strings.Index(rest, "-")
	if end < 0 {
		//coverage:ignore unreachable: idSlugTokenPattern requires a hyphen after the number.
		return "", "", false
	}
	return tok[:dash+1+end], rest[end+1:], true
}

// DocIDSlugReference scans the configured documentation paths for id-with-slug
// citations that contradict the tree. Shares the corpus resolution and the
// root-containment guard with DocIDWidthReference by reading the same files.
func DocIDSlugReference(t *tree.Tree, paths []string) []Finding {
	idx := DocSlugIndex(t)
	var findings []Finding
	for _, rel := range paths {
		raw, ok := readDocUnderRoot(t.Root, rel)
		if !ok {
			continue
		}
		findings = append(findings, ScanDocIDSlug(raw, rel, idx)...)
	}
	return findings
}
