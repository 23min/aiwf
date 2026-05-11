package policies

import (
	"fmt"
	"io/fs"
	"path"
	"regexp"
	"strings"
)

// markdownLinkRegex captures the path-target inside a markdown
// link `[text](path)`. The capture stops at whitespace or `)`, which
// also handles the `[text](path "title")` form (title separated by
// space).
var markdownLinkRegex = regexp.MustCompile(`\[[^\]]*\]\(([^)\s]+)\)`)

// entityFilenameRegex matches the last path segment of an entity
// file — `<kind>-<digits>-<slug>.md`. Kinds per the kernel's six
// (ADR, E, M, G, D, C); digit width is loose per ADR-0008's
// "parsers tolerate narrower legacy widths on input" so refs that
// embed the narrow 3-digit form still match (and still fail
// resolution if the file is at the canonical 4-digit slug).
var entityFilenameRegex = regexp.MustCompile(`^(ADR|E|M|G|D|C)-\d{1,4}-[^/]*\.md$`)

// auditDanglingEntityRefs scans the named narrative docs under fsys
// for markdown links pointing at entity files, and reports each ref
// whose target path does not resolve. Closes G-0091's chokepoint
// gap: catches archive/rename/width drift at PR time instead of
// after the post-hoc lychee link-check fires.
//
// Each finding is `<doc-path>:<line>: <link-as-written> -> <resolved-target> (not found)`.
// The resolved target uses repo-root-relative form (the fsys is
// expected to be `os.DirFS(repoRoot)` in the seam test); paths that
// walk outside the fsys root (`../../...`) are flagged as dangling
// because `fs.Stat` cannot resolve them.
//
// Scope is narrow by design: only entity-file shapes
// (`(ADR|E|M|G|D|C)-\d+-*.md`). General markdown-link integrity
// (including refs to `docs/pocv3/...` and deleted `critical-path.md`)
// is the lychee workflow's concern. G-0091 covers the slug-rename /
// id-width / archive-sweep drift class specifically.
func auditDanglingEntityRefs(fsys fs.FS, paths []string) []string {
	var findings []string
	for _, docPath := range paths {
		data, err := fs.ReadFile(fsys, docPath)
		if err != nil {
			// The audit is best-effort over the named paths; a
			// missing source file is a no-op (the operator decides
			// which paths to probe via the caller's list).
			continue
		}
		docDir := path.Dir(docPath)
		for lineIdx, line := range strings.Split(string(data), "\n") {
			for _, m := range markdownLinkRegex.FindAllStringSubmatch(line, -1) {
				ref := m[1]
				wholeLink := m[0]
				// Strip URL fragment / anchor.
				if hashIdx := strings.Index(ref, "#"); hashIdx >= 0 {
					ref = ref[:hashIdx]
				}
				// Filter to entity-file shape. Last segment of the
				// path is what we test against — the entity filename
				// is what gets renamed, not the parent directories.
				base := path.Base(ref)
				if !entityFilenameRegex.MatchString(base) {
					continue
				}
				// Resolve relative to the docPath's directory. Use
				// `path.Clean` (not `filepath.Clean`) so the slash-
				// based form fs.FS expects is preserved.
				target := path.Clean(path.Join(docDir, ref))
				if _, statErr := fs.Stat(fsys, target); statErr == nil {
					continue
				}
				findings = append(findings, fmt.Sprintf("%s:%d: %s -> %s (not found)", docPath, lineIdx+1, wholeLink, target))
			}
		}
	}
	return findings
}
