package policies

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// m0127Pocv3ScanRoots are the surfaces M-0127's AC-3 commits to keeping
// free of the literal "docs/pocv3" substring: Go source, embedded skill
// markdown, docs/, and the planning tree. Any directory named "archive"
// is skipped during the walk (work/*/archive/, docs/archive/pocv3/) —
// those subtrees are the frozen, forget-by-default historical record
// (per ADR-0004), and their content is historical prose, not live
// pointers.
var m0127Pocv3ScanRoots = []string{
	"cmd",
	"internal",
	"docs",
	"work",
}

// m0127Pocv3TopLevelFiles are scanned in addition to the roots above.
var m0127Pocv3TopLevelFiles = []string{
	"README.md",
	"CONTRIBUTING.md",
	"CLAUDE.md",
	"CHANGELOG.md",
	"aiwf.yaml",
}

// m0127Pocv3AllowlistPaths are exact repo-relative paths where a
// "docs/pocv3" mention is deliberate, historical prose rather than a
// dangling live reference. Each carries a one-line rationale.
var m0127Pocv3AllowlistPaths = map[string]string{
	// This epic's own record of the migration — TRIAGE.md is the
	// executed contract naming the old paths verbatim, and M-0126/
	// M-0127/M-0129's specs narrate or scope around the retirement
	// by name.
	"work/epics/E-0034-retire-docs-pocv3-and-declare-doc-authority-hierarchy/TRIAGE.md":                                                         "the executed disposition contract; historical record of the retired tree",
	"work/epics/E-0034-retire-docs-pocv3-and-declare-doc-authority-hierarchy/epic.md":                                                           "epic spec narrating the retirement by name",
	"work/epics/E-0034-retire-docs-pocv3-and-declare-doc-authority-hierarchy/M-0126-triage-docs-pocv3-into-per-file-disposition-table.md":       "the Triage milestone's own spec, about docs/pocv3/ by definition",
	"work/epics/E-0034-retire-docs-pocv3-and-declare-doc-authority-hierarchy/M-0127-relocate-docs-pocv3-contents-and-sweep-cross-references.md": "this milestone's own spec, about docs/pocv3/ by definition",
	"work/epics/E-0034-retire-docs-pocv3-and-declare-doc-authority-hierarchy/M-0129-drift-chokepoint-forbid-docs-pocv3-literals-in-go-code.md":  "the drift-chokepoint milestone's own spec, about docs/pocv3/ literals by definition",
	"work/epics/E-0034-retire-docs-pocv3-and-declare-doc-authority-hierarchy/wrap.md":                                                           "the epic's own wrap artefact, narrating the docs/pocv3/ retirement it closes out",

	// Design-corpus policy comments explaining the migration itself
	// (why the scan root changed) — historical rationale, not a
	// dangling pointer.
	"internal/policies/design_doc_anchors.go":  "doc comment explaining the M-0127 scope migration",
	"internal/policies/m083_doc_sweep_test.go": "doc comment explaining the M-0127 scope migration",

	// aiwf.yaml's tree.allow_paths comment describes what TRIAGE.md
	// is *about* (a disposition table for the now-retired tree), not
	// a live path.
	"aiwf.yaml": "descriptive comment naming TRIAGE.md's historical subject",

	// A synthetic fixture string quoting E-0034's own (accurate,
	// unchanged) title for display-rendering tests — not a path.
	"internal/cli/status/worktrees_test.go": "fixture quotes E-0034's own title verbatim, not a path reference",

	// Genuinely historical narrative: a cancelled sibling epic's
	// prose, a foreign PoC-branch citation, and a dead, never-built
	// proposal path — none point at a file this milestone moved.
	"docs/adr/ADR-0011-legal-workflow-spec-methodology.md":                                "describes a cancelled prior epic's (E-0031) abandoned artifact, historical as of cancellation",
	"docs/explorations/01-policies-design-space.md":                                       "cites gap ids on a foreign PoC branch, not this repo's current tree",
	"work/gaps/G-0121-legal-workflows-and-verb-composition-aren-t-pinned-mechanically.md": "cites a proposed artifact path that was never built (gap's own text confirms this)",

	// Gaps whose entire subject is docs/pocv3's own retirement,
	// explicitly slated for supersession by this epic (see M-0126's
	// References section) rather than living cross-references to
	// edit around.
	"work/gaps/G-0074-docs-pocv3-body-prose-still-uses-poc-framing-needs-sweep.md":       "gap's subject is docs/pocv3 itself; superseded by E-0034, not yet promoted",
	"work/gaps/G-0075-docs-pocv3-directory-naming-is-now-historical-rename-or-accept.md": "gap's subject is docs/pocv3 itself; superseded by E-0034, not yet promoted",
	"work/gaps/G-0092-no-documented-doc-authority-hierarchy.md":                          "gap's subject is docs/pocv3's evidentiary role; superseded by E-0034 (M-0128), not yet promoted",
	"work/gaps/G-0077-post-promotion-working-paper-aiwf-s-thesis-not-yet-written.md":     "hypothetical example mention in an open gap's prose, not a path reference",

	// G-0434 quotes a real historical commit message verbatim
	// ("Triage docs/pocv3/...") as part of its root-cause narrative.
	"work/gaps/G-0434-resolveviapriorids-prefers-a-stale-prior-ids-match-over-a-reused-live-id.md": "quotes an actual git commit message verbatim",

	// This test file itself necessarily names the literal substring
	// in its own allowlist keys, docstrings, and error message.
	"internal/policies/m0127_no_dangling_pocv3_refs_test.go": "the check itself; the string appears in its allowlist keys and docs",

	// M-0126's still-live AC-2/4/5 tests parse TRIAGE.md's own row
	// format, whose File column literally starts with "docs/pocv3/"
	// for every row (the historical record of the executed contract)
	// — both in the parser's string-prefix match and its doc comment.
	"internal/policies/m0126_triage_table_test.go": "parses TRIAGE.md's row format, which cites the historical docs/pocv3/ paths verbatim",

	// CHANGELOG.md is append-only release history describing state as
	// of each release — the same historical carve-out this repo's own
	// doc-lint convention and m083_doc_sweep_test.go's allowlist give
	// it elsewhere.
	"CHANGELOG.md": "append-only release notes describing historical state at release time",

	// A follow-up gap filed during E-0034's own wrap doc-lint sweep;
	// its Why-it-matters section narrates docs/design/id-allocation.md's
	// relocation from docs/pocv3/design/ to explain why one of its two
	// stale cmd/aiwf/ citations pre-dates that move.
	"work/gaps/G-0436-claude-md-and-id-allocation-md-cite-stale-cmd-aiwf-paths-for-relocated-verbs.md": "narrates a relocated file's docs/pocv3/design/ origin to explain a pre-existing citation's staleness",
}

// TestM0127_AC3_NoDanglingDocsPocv3References is the mechanical
// evidence for M-0127's AC-3: after the relocate sweep, the literal
// substring "docs/pocv3" appears nowhere under cmd/, internal/, docs/,
// or work/ (excluding any "archive" subtree, the frozen historical
// snapshot per ADR-0004) or in the top-level narrative files, except
// at the explicitly allowlisted paths above.
func TestM0127_AC3_NoDanglingDocsPocv3References(t *testing.T) {
	t.Parallel()
	root := repoRoot(t)

	var hits []string
	scanFile := func(relPath string) {
		if _, ok := m0127Pocv3AllowlistPaths[relPath]; ok {
			return
		}
		data, err := os.ReadFile(filepath.Join(root, relPath))
		if err != nil {
			t.Fatalf("reading %s: %v", relPath, err)
		}
		if strings.Contains(string(data), "docs/pocv3") {
			hits = append(hits, relPath)
		}
	}

	for _, base := range m0127Pocv3ScanRoots {
		baseAbs := filepath.Join(root, base)
		err := filepath.Walk(baseAbs, func(p string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if info.IsDir() {
				if info.Name() == "archive" {
					return filepath.SkipDir
				}
				return nil
			}
			if !strings.HasSuffix(p, ".go") && !strings.HasSuffix(p, ".md") {
				return nil
			}
			rel, relErr := filepath.Rel(root, p)
			if relErr != nil {
				return relErr
			}
			scanFile(filepath.ToSlash(rel))
			return nil
		})
		if err != nil {
			t.Fatalf("walking %s: %v", base, err)
		}
	}

	for _, f := range m0127Pocv3TopLevelFiles {
		scanFile(f)
	}

	if len(hits) > 0 {
		sort.Strings(hits)
		t.Errorf("AC-3: %d file(s) contain a dangling \"docs/pocv3\" reference:\n  %s", len(hits), strings.Join(hits, "\n  "))
	}
}
