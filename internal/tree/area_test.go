package tree

import (
	"context"
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/entity"
)

// TestResolvedArea pins AC-3 of M-0171: root kinds carry their own area, a
// milestone derives its area from its parent epic, and an untagged or
// unresolvable parent yields "". Every branch of ResolvedArea is exercised:
// nil entity, milestone with a tagged parent, milestone with an untagged
// parent, milestone with an unresolvable (orphan) parent, and a root kind.
func TestResolvedArea(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	writeFile(t, root, "work/epics/E-0001-platform/epic.md", "---\nid: E-0001\ntitle: Platform\nstatus: active\narea: platform\n---\n")
	writeFile(t, root, "work/epics/E-0001-platform/M-0001-cache.md", "---\nid: M-0001\ntitle: Cache\nstatus: in_progress\nparent: E-0001\n---\n")
	writeFile(t, root, "work/epics/E-0001-platform/M-0003-orphan.md", "---\nid: M-0003\ntitle: Orphan\nstatus: draft\nparent: E-9999\n---\n")
	writeFile(t, root, "work/epics/E-0002-billing/epic.md", "---\nid: E-0002\ntitle: Billing\nstatus: active\n---\n")
	writeFile(t, root, "work/epics/E-0002-billing/M-0002-invoice.md", "---\nid: M-0002\ntitle: Invoice\nstatus: draft\nparent: E-0002\n---\n")
	writeFile(t, root, "work/gaps/G-0001-thrash.md", "---\nid: G-0001\ntitle: Thrash\nstatus: open\narea: tooling\n---\n")

	tr, _, err := Load(context.Background(), root)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if got := tr.ResolvedArea(nil); got != "" {
		t.Errorf("ResolvedArea(nil) = %q, want empty", got)
	}

	cases := []struct {
		id   string
		want string
	}{
		{"E-0001", "platform"}, // epic carries its own area (root kind)
		{"M-0001", "platform"}, // milestone derives from parent epic
		{"E-0002", ""},         // untagged epic
		{"M-0002", ""},         // milestone under an untagged epic
		{"M-0003", ""},         // milestone whose parent epic does not resolve
		{"G-0001", "tooling"},  // root kind carries its own area
	}
	for _, tc := range cases {
		t.Run(tc.id, func(t *testing.T) {
			t.Parallel()
			e := tr.ByID(tc.id)
			if e == nil {
				t.Fatalf("entity %s not found", tc.id)
			}
			if got := tr.ResolvedArea(e); got != tc.want {
				t.Errorf("ResolvedArea(%s) = %q, want %q", tc.id, got, tc.want)
			}
		})
	}
}

// TestLoad_MilestoneStoredAreaCleared pins Option-2 behavior: a milestone that
// carries its own `area:` on disk has it blanked at load, so the in-memory
// model never shows a milestone with its own area, and ResolvedArea returns the
// parent epic's value (not the stored one). The round-trip cleanup — serialize
// drops the blanked key on the next write-verb — is pinned too, so the
// auto-strip of the invalid value is intentional, not incidental.
func TestLoad_MilestoneStoredAreaCleared(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	writeFile(t, root, "work/epics/E-0001-platform/epic.md", "---\nid: E-0001\ntitle: Platform\nstatus: active\narea: platform\n---\n")
	writeFile(t, root, "work/epics/E-0001-platform/M-0001-cache.md", "---\nid: M-0001\ntitle: Cache\nstatus: in_progress\nparent: E-0001\narea: rogue\n---\n")

	tr, _, err := Load(context.Background(), root)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	m := tr.ByID("M-0001")
	if m == nil {
		t.Fatal("M-0001 not found")
	}
	if m.Area != "" {
		t.Errorf("milestone stored area = %q, want cleared", m.Area)
	}
	if got := tr.ResolvedArea(m); got != "platform" {
		t.Errorf("ResolvedArea(M-0001) = %q, want %q (parent epic, not stored 'rogue')", got, "platform")
	}
	out, err := entity.Serialize(m, []byte("body\n"))
	if err != nil {
		t.Fatalf("Serialize: %v", err)
	}
	if strings.Contains(string(out), "area:") {
		t.Errorf("serialized milestone still carries an area key:\n%s", out)
	}
}

// TestResolvedAreaByID pins the AC-3 AC-derivation seam (B1): the effective
// area for any id — including a composite acceptance-criterion id — resolves
// through one entry point, so downstream filter/grouping never re-derive the
// AC rollup. A composite AC id rolls up to its parent milestone's resolved
// area; a bare id passes through; an unresolvable id yields "".
func TestResolvedAreaByID(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	writeFile(t, root, "work/epics/E-0001-platform/epic.md", "---\nid: E-0001\ntitle: Platform\nstatus: active\narea: platform\n---\n")
	writeFile(t, root, "work/epics/E-0001-platform/M-0001-cache.md", "---\nid: M-0001\ntitle: Cache\nstatus: in_progress\nparent: E-0001\nacs:\n  - id: AC-1\n    title: x\n    status: open\n---\n")
	writeFile(t, root, "work/epics/E-0002-billing/epic.md", "---\nid: E-0002\ntitle: Billing\nstatus: active\n---\n")
	writeFile(t, root, "work/epics/E-0002-billing/M-0002-invoice.md", "---\nid: M-0002\ntitle: Invoice\nstatus: draft\nparent: E-0002\n---\n")
	writeFile(t, root, "work/gaps/G-0001-thrash.md", "---\nid: G-0001\ntitle: Thrash\nstatus: open\narea: tooling\n---\n")

	tr, _, err := Load(context.Background(), root)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	cases := []struct {
		id   string
		want string
	}{
		{"M-0001/AC-1", "platform"}, // AC derives via parent milestone → parent epic
		{"M-0002/AC-1", ""},         // AC under an untagged epic
		{"M-0001", "platform"},      // bare milestone id passes through to ResolvedArea
		{"E-0001", "platform"},      // bare epic id
		{"G-0001", "tooling"},       // bare root-kind id
		{"M-9999/AC-1", ""},         // composite id whose milestone does not resolve
		{"Z-0404", ""},              // bare id that does not resolve
	}
	for _, tc := range cases {
		t.Run(tc.id, func(t *testing.T) {
			t.Parallel()
			if got := tr.ResolvedAreaByID(tc.id); got != tc.want {
				t.Errorf("ResolvedAreaByID(%q) = %q, want %q", tc.id, got, tc.want)
			}
		})
	}
}
