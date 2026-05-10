package policies

import (
	"context"
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/check"
	"github.com/23min/aiwf/internal/tree"
)

// TestPolicy_ThisRepoTreeIsClean is the AC-6 chokepoint for M-081:
// `aiwf check` on this repo's still-narrow-on-disk tree must produce
// zero findings related to id widths. The load-bearing assertion: the
// parser-tolerance change in AC-2 is genuinely pure-additive — every
// existing entity continues to load, every reference resolves, every
// frontmatter shape validates.
//
// Findings unrelated to id widths (e.g. provenance scope-undefined
// when no upstream is configured) are tolerated; only id-width-shaped
// codes (refs-resolve, ids-unique, id-path-consistent,
// frontmatter-shape, status-valid) trigger a failure.
//
// Per CLAUDE.md "framework correctness must not depend on the LLM's
// behavior", AC-6's discipline lives here, not in reviewer recall.
func TestPolicy_ThisRepoTreeIsClean(t *testing.T) {
	root, err := repoRootFromTest(t)
	if err != nil {
		t.Fatalf("locate repo root: %v", err)
	}
	tr, loadErrs, err := tree.Load(context.Background(), root)
	if err != nil {
		t.Fatalf("tree.Load: %v", err)
	}
	if len(loadErrs) != 0 {
		t.Fatalf("loadErrs: %v", loadErrs)
	}

	findings := check.Run(tr, loadErrs)
	// AC-6 cares about id-width-shaped codes specifically. The wider
	// codes here are the ones an id-width regression would surface
	// under: a parser that fails to tolerate a narrow id would fire
	// refs-resolve/unresolved or frontmatter-shape; an allocator
	// emitting the wrong width would fire id-path-consistent.
	idWidthShaped := map[string]bool{
		"refs-resolve":       true,
		"ids-unique":         true,
		"id-path-consistent": true,
		"frontmatter-shape":  true,
	}
	var unwanted []check.Finding
	for _, f := range findings {
		if f.Severity != check.SeverityError {
			continue
		}
		if !idWidthShaped[f.Code] {
			continue
		}
		unwanted = append(unwanted, f)
	}
	if len(unwanted) > 0 {
		var lines []string
		for _, f := range unwanted {
			lines = append(lines, "  "+f.Code+": "+f.Message+" ("+f.EntityID+" at "+f.Path+")")
		}
		t.Errorf("AC-6: %d id-width-shaped findings on this repo's tree (parser-tolerance regression):\n%s",
			len(unwanted), strings.Join(lines, "\n"))
	}
}
