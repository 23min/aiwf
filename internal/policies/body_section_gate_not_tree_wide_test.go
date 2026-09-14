package policies

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/23min/aiwf/internal/check"
	"github.com/23min/aiwf/internal/entity"
	"github.com/23min/aiwf/internal/tree"
)

// body_section_gate_not_tree_wide_test.go — M-0331/AC-3. The push-seam
// membership gate rides a commit range, not the tree, so `aiwf check`'s
// tree-wide pass reports nothing new and no entity already carrying an omission
// gains a finding.
//
// The fixture is a tree holding exactly the entity that would gain one: a gap
// whose body omits a required section. It is built here rather than read from
// the live tree, so the test does not depend on that tree still carrying debt.

func TestPolicy_BodySectionGateIsNotTreeWide(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	const rel = "work/gaps/G-0001-omits-a-section.md"
	body := "---\nid: G-0001\ntitle: Omits a section\nstatus: open\n---\n## What's missing\n\nOnly this.\n"
	if err := os.MkdirAll(filepath.Join(root, filepath.Dir(rel)), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, rel), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	tr, loadErrs, err := tree.Load(context.Background(), root)
	if err != nil {
		t.Fatalf("tree.Load: %v", err)
	}
	e := tr.ByID("G-0001")
	if e == nil {
		t.Fatal("the fixture gap did not load, so the tree-wide pass has nothing to report on")
	}
	_, entityBody, _ := entity.Split([]byte(body))
	if absent := check.AbsentRequiredSections(e.Kind, entityBody); len(absent) == 0 {
		t.Fatal("the fixture gap omits no required section, so it could not gain a finding")
	}

	for _, f := range check.Run(tr, loadErrs) {
		if f.Code == check.CodeEntityBodySectionDropped.ID {
			t.Errorf("the tree-wide pass emitted %s on %s — the gate rides a commit range and must not join check.Run",
				f.Code, f.EntityID)
		}
	}
}
