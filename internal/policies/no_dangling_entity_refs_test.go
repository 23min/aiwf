package policies

import (
	"os"
	"strings"
	"testing"
	"testing/fstest"
)

// TestPolicy_NoDanglingEntityRefsInNarrativeDocs pins G-0091's
// chokepoint claim: path-form markdown links to entity files in
// narrative docs (CLAUDE.md, ROADMAP.md) must resolve to an existing
// file. A reference whose path no longer points at the entity (slug
// renamed, id-width canonicalized, file archive-swept, or entity
// deleted) fires here, at PR time, instead of waiting for the
// post-hoc `link-check` workflow.
//
// Scope is intentionally narrow — only entity-file shapes (paths
// ending in `(ADR|E|M|G|D|C)-\d+-<slug>.md`). General markdown link
// integrity is the lychee workflow's concern; this rule is the
// archive/rename/width drift class specifically.
func TestPolicy_NoDanglingEntityRefsInNarrativeDocs(t *testing.T) {
	t.Parallel()
	root := repoRoot(t)
	fsys := os.DirFS(root)

	// CLAUDE.md and ROADMAP.md are the two hand-authored narrative
	// docs that historically embed path-form entity refs. If a new
	// narrative doc enters the same pattern, add its path here.
	paths := []string{"CLAUDE.md", "ROADMAP.md"}

	findings := auditDanglingEntityRefs(fsys, paths)
	for _, f := range findings {
		t.Errorf("dangling entity-file ref: %s — update the path to the entity's current location (or switch to bare-id form `G-NNNN` which the loader resolves)", f)
	}
}

// TestAuditDanglingEntityRefs_BranchCoverage exercises every
// reachable arm of the helper against synthetic in-memory inputs:
// clean (no entity refs), valid-ref (resolves), dangling-ref
// (doesn't resolve), mixed (one of each), anchor-stripped (link
// with #fragment), non-entity-shape (out of scope), and the
// missing-file fs error arm. Per CLAUDE.md §"Test untested code
// paths before declaring code paths done", each branch has a test
// that traverses it.
func TestAuditDanglingEntityRefs_BranchCoverage(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name         string
		fsys         fstest.MapFS
		paths        []string
		wantFindings int
		wantContains string
	}{
		{
			name: "clean-no-entity-refs",
			fsys: fstest.MapFS{
				"CLAUDE.md": {Data: []byte("# Heading\n\nSome prose without links.\n")},
			},
			paths:        []string{"CLAUDE.md"},
			wantFindings: 0,
		},
		{
			name: "valid-entity-ref-resolves",
			fsys: fstest.MapFS{
				"docs/adr/ADR-0007-foo.md": {Data: []byte("# ADR-0007\n")},
				"CLAUDE.md":                {Data: []byte("See [ADR-0007](docs/adr/ADR-0007-foo.md).\n")},
			},
			paths:        []string{"CLAUDE.md"},
			wantFindings: 0,
		},
		{
			name: "dangling-entity-ref-fires",
			fsys: fstest.MapFS{
				"CLAUDE.md": {Data: []byte("See [ADR-0007](docs/adr/ADR-0007-renamed.md).\n")},
			},
			paths:        []string{"CLAUDE.md"},
			wantFindings: 1,
			wantContains: "ADR-0007-renamed.md",
		},
		{
			name: "mixed-valid-and-dangling",
			fsys: fstest.MapFS{
				"docs/adr/ADR-0007-foo.md": {Data: []byte("")},
				"CLAUDE.md": {Data: []byte(
					"Valid: [ADR-0007](docs/adr/ADR-0007-foo.md).\n" +
						"Broken: [G-0099](work/gaps/G-0099-gone.md).\n",
				)},
			},
			paths:        []string{"CLAUDE.md"},
			wantFindings: 1,
			wantContains: "G-0099-gone.md",
		},
		{
			name: "anchor-fragment-stripped-before-resolve",
			fsys: fstest.MapFS{
				"docs/adr/ADR-0007-foo.md": {Data: []byte("")},
				"CLAUDE.md":                {Data: []byte("See [foo](docs/adr/ADR-0007-foo.md#section).\n")},
			},
			paths:        []string{"CLAUDE.md"},
			wantFindings: 0,
		},
		{
			name: "non-entity-shape-link-ignored",
			fsys: fstest.MapFS{
				// Path-form link to a non-entity .md file — out of
				// G-0091's scope. The lychee workflow catches these.
				"CLAUDE.md": {Data: []byte("See [docs](docs/some-non-entity-doc.md) and [readme](README.md).\n")},
			},
			paths:        []string{"CLAUDE.md"},
			wantFindings: 0,
		},
		{
			name: "relative-walk-outside-fsys-is-dangling",
			fsys: fstest.MapFS{
				// `../../gaps/G-0055-x.md` walks outside the fsys
				// root — must fire as a dangling ref.
				"ROADMAP.md": {Data: []byte("See [G-0055](../../gaps/G-0055-x.md).\n")},
			},
			paths:        []string{"ROADMAP.md"},
			wantFindings: 1,
			wantContains: "G-0055-x.md",
		},
		{
			name: "archive-path-resolves",
			fsys: fstest.MapFS{
				"work/gaps/archive/G-0055-archived.md": {Data: []byte("")},
				"ROADMAP.md":                           {Data: []byte("See [G-0055](work/gaps/archive/G-0055-archived.md).\n")},
			},
			paths:        []string{"ROADMAP.md"},
			wantFindings: 0,
		},
		{
			name: "narrow-id-width-link-still-checked",
			fsys: fstest.MapFS{
				// G-055 (3-digit narrow) is a valid id-shape per
				// ADR-0008 ("parsers tolerate narrower legacy widths"),
				// so the regex matches it; the file doesn't exist
				// → finding.
				"ROADMAP.md": {Data: []byte("See [G-055](work/gaps/G-055-x.md).\n")},
			},
			paths:        []string{"ROADMAP.md"},
			wantFindings: 1,
			wantContains: "G-055-x.md",
		},
		{
			name: "multiple-refs-one-line",
			fsys: fstest.MapFS{
				"ROADMAP.md": {Data: []byte("Bad [G-0055](work/gaps/G-0055-a.md) and [G-0058](work/gaps/G-0058-b.md).\n")},
			},
			paths:        []string{"ROADMAP.md"},
			wantFindings: 2,
		},
		{
			name: "missing-source-file-silently-skipped",
			fsys: fstest.MapFS{
				// CLAUDE.md is absent from fsys — the audit should
				// skip silently per the helper's defensive contract.
			},
			paths:        []string{"CLAUDE.md"},
			wantFindings: 0,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			findings := auditDanglingEntityRefs(tc.fsys, tc.paths)
			if len(findings) != tc.wantFindings {
				t.Fatalf("%s: expected %d findings, got %d: %v", tc.name, tc.wantFindings, len(findings), findings)
			}
			if tc.wantContains != "" && len(findings) > 0 {
				found := false
				for _, f := range findings {
					if strings.Contains(f, tc.wantContains) {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("%s: expected a finding containing %q, got %v", tc.name, tc.wantContains, findings)
				}
			}
		})
	}
}
