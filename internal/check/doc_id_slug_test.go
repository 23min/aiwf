package check

// M-0289 AC-4: an id written with a slug must carry the slug its entity
// actually has.
//
// This is the exact half of the fiction problem. A canonical-width id invented
// for a worked example is invisible to any width rule and collides with
// whatever real entity holds that number — measured in this repo's own
// workflows guide, where a fictional ADR borrowed a real id and spelled out a
// filename contradicting it. When the slug is written too, the contradiction
// stops being a matter of judgment: the loader knows the entity's real slug,
// so this is string equality, with no heuristic and no false-positive surface.

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/entity"
	"github.com/23min/aiwf/internal/tree"
)

// TestDocSlugIndex_SkipsPathsCarryingNoSlug covers the entity whose path does
// not encode its id — a stub built from an unparseable file, say. It has no
// slug to compare against, so it must be absent from the index rather than
// present with an empty one, which would make every citation of it "wrong".
func TestDocSlugIndex_SkipsPathsCarryingNoSlug(t *testing.T) {
	t.Parallel()
	idx := DocSlugIndex(&tree.Tree{Entities: []*entity.Entity{
		{ID: "M-0007", Path: "work/epics/E-0002-auth/notes.md"},
	}})
	if _, ok := idx["M-0007"]; ok {
		t.Errorf("index holds an entry for a path encoding no slug: %+v", idx)
	}
}

// TestDocIDSlugReference_ReadsConfiguredPaths covers the walker, and confirms
// it routes through the same root-containment guard the width rule uses.
func TestDocIDSlugReference_ReadsConfiguredPaths(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "README.md"),
		[]byte("# R\n\nSee docs/adr/ADR-0001-something-else-entirely.md.\n"), 0o644); err != nil {
		t.Fatalf("seeding doc: %v", err)
	}
	tr := &tree.Tree{Root: root, Entities: []*entity.Entity{
		{ID: "ADR-0001", Path: "docs/adr/ADR-0001-mint-entity-ids-at-trunk-integration.md"},
	}}
	got := DocIDSlugReference(tr, []string{"README.md"})
	if len(got) != 1 {
		t.Fatalf("want 1 finding, got %d: %+v", len(got), got)
	}
	if got[0].Path != "README.md" {
		t.Errorf("path = %q, want %q", got[0].Path, "README.md")
	}
	if len(DocIDSlugReference(tr, []string{"../escape.md"})) != 0 {
		t.Error("a path escaping the root must be skipped")
	}
}

// docSlugFixture is a tree holding entities whose ids and slugs are known, so
// a doc citing those ids can be checked against the truth.
func docSlugFixture() *tree.Tree {
	return &tree.Tree{Entities: []*entity.Entity{
		{ID: "ADR-0001", Path: "docs/adr/ADR-0001-mint-entity-ids-at-trunk-integration.md"},
		{ID: "M-0007", Path: "work/epics/E-0002-auth/M-0007-schema-migration.md"},
	}}
}

func TestScanDocIDSlug_MismatchFires(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		body string
		want string
	}{
		{
			// The exact shape found in this repo: an invented ADR borrowing a
			// real id, with a filename that contradicts the entity.
			name: "path form",
			body: "# Doc\n\n```bash\n# → docs/adr/ADR-0001-use-oauth-21-with-passkey-support.md\n```\n",
			want: "ADR-0001",
		},
		{
			name: "bare id-slug form",
			body: "# Doc\n\nSee M-0007-login-form-refactor for the example.\n",
			want: "M-0007",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := ScanDocIDSlug([]byte(tc.body), "docs/workflows.md", DocSlugIndex(docSlugFixture()))
			if len(got) != 1 {
				t.Fatalf("want 1 finding, got %d: %+v", len(got), got)
			}
			if got[0].Code != CodeDocIDSlug {
				t.Errorf("code = %q, want %q", got[0].Code, CodeDocIDSlug)
			}
			// Advisory at the source, like its sibling: only ApplyDocsStrict
			// raises it, so a repo that has not opted in is never blocked.
			if got[0].Severity != SeverityWarning {
				t.Errorf("severity = %q, want %q — the rule ships advisory and is escalated by config, never the reverse", got[0].Severity, SeverityWarning)
			}
			if !strings.Contains(got[0].Message, tc.want) {
				t.Errorf("message %q does not name the entity %q", got[0].Message, tc.want)
			}
		})
	}
}

// TestScanDocIDSlug_RequiresAWordBoundary guards against a longer word
// donating its tail. Without a leading \b, any word ending in a kind letter
// reads as an id — and the finding then quotes a token that appears nowhere in
// the file, so the operator has nothing to search for.
func TestScanDocIDSlug_RequiresAWordBoundary(t *testing.T) {
	t.Parallel()
	for _, body := range []string{
		"# Doc\n\nSee RFC-0001-some-other-thing.md for details.\n",
		"# Doc\n\nSee XM-0007-a-different-thing.md for details.\n",
	} {
		if got := ScanDocIDSlug([]byte(body), "docs/workflows.md", DocSlugIndex(docSlugFixture())); len(got) != 0 {
			t.Fatalf("a longer word must not donate its tail, got %d: %+v", len(got), got)
		}
	}
}

// TestScanDocIDSlug_NamesTheEntityCanonically pins the remediation against a
// path that can actually exist. Building it from the doc's spelling of the id
// yields one when the doc also wrote the id narrow — an operator who obeys
// writes a second wrong path and stays blocked.
func TestScanDocIDSlug_NamesTheEntityCanonically(t *testing.T) {
	t.Parallel()
	idx := DocSlugIndex(&tree.Tree{Entities: []*entity.Entity{
		{ID: "M-0007", Path: "work/epics/E-0002-auth/M-0007-schema-migration.md"},
	}})
	got := ScanDocIDSlug([]byte("# Doc\n\nSee M-007-wrong-slug.md.\n"), "README.md", idx)
	if len(got) != 1 {
		t.Fatalf("a narrow id with a wrong slug must fire, got %d: %+v", len(got), got)
	}
	if !strings.Contains(got[0].Message, "M-0007-schema-migration") {
		t.Errorf("message %q does not name the entity's real, canonical path form", got[0].Message)
	}
	if strings.Contains(got[0].Message, "M-007-schema-migration") {
		t.Errorf("message %q names a path built from the doc's spelling, which exists nowhere", got[0].Message)
	}
}

// TestScanDocIDSlug_TrueSlugSilent is the arm that keeps the corpus writable:
// citing a real entity by its real path is exactly what these docs are for.
func TestScanDocIDSlug_TrueSlugSilent(t *testing.T) {
	t.Parallel()
	for _, body := range []string{
		"# Doc\n\nSee docs/adr/ADR-0001-mint-entity-ids-at-trunk-integration.md.\n",
		"# Doc\n\nSee work/epics/E-0002-auth/M-0007-schema-migration.md.\n",
	} {
		if got := ScanDocIDSlug([]byte(body), "docs/workflows.md", DocSlugIndex(docSlugFixture())); len(got) != 0 {
			t.Fatalf("a correct citation must be silent, got %d: %+v", len(got), got)
		}
	}
}

// TestScanDocIDSlug_BareIDSilent bounds the rule to what it can prove. An id
// with no slug carries no contradiction to detect — whether it is a citation
// or fiction is exactly the judgment this rule declines to make.
func TestScanDocIDSlug_BareIDSilent(t *testing.T) {
	t.Parallel()
	body := "# Doc\n\nRun `aiwf promote ADR-0001 accepted` when ready.\n"
	if got := ScanDocIDSlug([]byte(body), "docs/workflows.md", DocSlugIndex(docSlugFixture())); len(got) != 0 {
		t.Fatalf("a bare id must be silent, got %d: %+v", len(got), got)
	}
}

// TestScanDocIDSlug_UnknownIDSilent keeps this rule off the resolution
// question. An id naming no entity has no slug to disagree with; whether docs
// should be reference-checked at all is a separate and much larger change.
func TestScanDocIDSlug_UnknownIDSilent(t *testing.T) {
	t.Parallel()
	body := "# Doc\n\nSee docs/adr/ADR-9999-some-invented-thing.md.\n"
	if got := ScanDocIDSlug([]byte(body), "docs/workflows.md", DocSlugIndex(docSlugFixture())); len(got) != 0 {
		t.Fatalf("an unknown id must be silent, got %d: %+v", len(got), got)
	}
}

// TestScanDocIDSlug_PlaceholderSilent guards the interaction with AC-1's rule.
// The canonical placeholder is the sanctioned way to write fiction, so it must
// pass here even when followed by slug-shaped words.
func TestScanDocIDSlug_PlaceholderSilent(t *testing.T) {
	t.Parallel()
	body := "# Doc\n\n```bash\n# → docs/adr/ADR-NNNN-use-oauth-21-with-passkey-support.md\n```\n"
	if got := ScanDocIDSlug([]byte(body), "docs/workflows.md", DocSlugIndex(docSlugFixture())); len(got) != 0 {
		t.Fatalf("the canonical placeholder must be silent, got %d: %+v", len(got), got)
	}
}
