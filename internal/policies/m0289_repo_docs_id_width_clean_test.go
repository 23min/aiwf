package policies

// M-0289 AC-2: this repo's own declared doc corpus carries no
// below-canonical-width id shape.
//
// The rule itself ships and is unit-tested in internal/check. What lives here
// is the claim that is only about THIS repo — that the sweep actually landed
// and stays landed. A consumer's docs are their business; ours are a repo
// invariant, which is what puts the assertion in this package rather than in
// the shipped rule's own tests.
//
// The corpus is read from aiwf.yaml rather than hardcoded, so the assertion
// tracks whatever this repo declares. That alone would pass vacuously if the
// declaration were emptied or narrowed, so the declared set is itself pinned
// below: dropping a doc from the config is then a visible act rather than a
// silent way to make this test green.

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/23min/aiwf/internal/check"
	"github.com/23min/aiwf/internal/config"
)

// m0289RequiredDocs are the documents whose narrow ids M-0289 swept. They
// teach the workflow, so a stale width in them is what an assistant learns.
var m0289RequiredDocs = []string{"README.md", "docs/workflows.md"}

func TestM0289_AC2_RepoDocsCarryNoNarrowIDWidth(t *testing.T) {
	t.Parallel()
	root, tr := sharedRepoTree(t)

	cfg, err := config.Load(root)
	if err != nil {
		t.Fatalf("loading aiwf.yaml: %v", err)
	}
	paths := cfg.DocsPaths()

	declared := map[string]bool{}
	for _, p := range paths {
		declared[p] = true
	}
	for _, want := range m0289RequiredDocs {
		if !declared[want] {
			t.Fatalf("aiwf.yaml docs.paths no longer declares %q — "+
				"the sweep's guarantee is only as wide as the declared corpus, so "+
				"removing a doc silently retires this assertion rather than satisfying it", want)
		}
	}

	// A declared path that no longer resolves is skipped by the scanner, so
	// without this the document silently leaves the corpus forever: rename it
	// or move it under a new directory and both this test and `aiwf check`
	// stay green while guarding nothing.
	for _, rel := range paths {
		if _, err := os.Stat(filepath.Join(root, rel)); err != nil {
			t.Errorf("aiwf.yaml declares %q but it does not resolve (%v) — the doc "+
				"has left the corpus and nothing else would say so", rel, err)
		}
	}

	for _, f := range check.DocIDWidthReference(tr, paths) {
		t.Errorf("%s:%d: %s", f.Path, f.Line, f.Message)
	}
	// The slug rule shares this corpus; running only its sibling would leave a
	// drift after `aiwf retitle` with no CI backstop.
	for _, f := range check.DocIDSlugReference(tr, paths) {
		t.Errorf("%s:%d: %s", f.Path, f.Line, f.Message)
	}
}

// TestM0289_AC2_RepoDocsIDWidthIsBlocking pins the severity this repo holds
// itself to. The shipped default is advisory (internal/check/doc_id_width.go
// says why); this repo has swept its docs, so it takes the blocking bar.
//
// Without this, `docs.strict` could quietly revert to the shipped default and
// the sweep above would keep passing while nothing enforced it at the push.
func TestM0289_AC2_RepoDocsIDWidthIsBlocking(t *testing.T) {
	t.Parallel()
	root, _ := sharedRepoTree(t)
	cfg, err := config.Load(root)
	if err != nil {
		t.Fatalf("loading aiwf.yaml: %v", err)
	}
	if !cfg.DocsStrict() {
		t.Error("aiwf.yaml docs.strict is off — this repo's swept docs " +
			"should block the push on a regression, not merely warn")
	}
}
