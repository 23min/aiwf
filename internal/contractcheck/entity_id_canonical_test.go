package contractcheck

import (
	"testing"

	"github.com/23min/aiwf/internal/aiwfyaml"
	"github.com/23min/aiwf/internal/entity"
	"github.com/23min/aiwf/internal/tree"
)

// TestRun_FindingsNameBindingsAtCanonicalWidth: a binding stored at a
// legacy narrow width yields findings whose entity_id is canonical, on
// every finding the configuration check builds for an entry — including
// the path-escape finding contractconfig contributes.
func TestRun_FindingsNameBindingsAtCanonicalWidth(t *testing.T) {
	t.Parallel()
	repo := t.TempDir()
	tr := &tree.Tree{Root: repo}
	contracts := &aiwfyaml.Contracts{
		Validators: map[string]aiwfyaml.Validator{"cue": {Command: "cue"}},
		Entries: []aiwfyaml.Entry{
			// No entity, no schema, no fixtures: missing-entity,
			// missing-schema and missing-fixtures.
			{ID: "C-001", Validator: "cue", Schema: "absent.cue", Fixtures: "absent"},
			// Resolves outside the repo root: path-escape.
			{ID: "C-002", Validator: "cue", Schema: "../outside.cue", Fixtures: "absent"},
		},
	}
	got := Run(tr, contracts, repo)
	seen := map[string]bool{}
	for _, f := range got {
		seen[f.Subcode] = true
		if canon := entity.Canonicalize(f.EntityID); f.EntityID != canon {
			t.Errorf("%s/%s finding names %q, want canonical %q", f.Code, f.Subcode, f.EntityID, canon)
		}
	}
	for _, sub := range []string{"missing-entity", "missing-schema", "missing-fixtures", "path-escape"} {
		if !seen[sub] {
			t.Errorf("no %s finding; the fixture did not reach that site (got %+v)", sub, got)
		}
	}
}
