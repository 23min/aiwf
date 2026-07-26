package status

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/23min/aiwf/internal/entity"
	"github.com/23min/aiwf/internal/render"
)

// TestRenderWorktreeSection_DirtyMarker pins the G-0122 "dirty" branch
// in the verbose per-worktree section: a worktree with uncommitted
// changes gets a colored "dirty" badge appended to the branch/last-
// commit line, styled as in-progress (the same bucket as the
// entity.StatusInProgress driver-status color) regardless of the
// driver's own status. The wantMarker value is computed via the same
// render.StatusColor call the production line makes — this test pins
// the wiring (Dirty -> this specific call with these arguments), not
// StatusColor's own color-mapping correctness (covered directly by
// render.TestStatusColor).
func TestRenderWorktreeSection_DirtyMarker(t *testing.T) {
	t.Parallel()
	v := &WorktreeView{Path: "/repo/wt-dirty", Branch: "milestone/M-9080-x", Dirty: true}
	var buf bytes.Buffer
	if err := renderWorktreeSection(&buf, v, true); err != nil {
		t.Fatalf("renderWorktreeSection: %v", err)
	}
	wantMarker := render.StatusColor("dirty", string(entity.StatusInProgress), true)
	got := buf.String()
	if !strings.Contains(got, wantMarker) {
		t.Errorf("renderWorktreeSection output missing dirty marker %q:\n%s", wantMarker, got)
	}
}

// TestRenderWorktreeSection_NotDirtyOmitsMarker is the negative
// counterpart: with Dirty false, no dirty marker appears at all.
func TestRenderWorktreeSection_NotDirtyOmitsMarker(t *testing.T) {
	t.Parallel()
	v := &WorktreeView{Path: "/repo/wt-clean", Branch: "milestone/M-9081-x"}
	var buf bytes.Buffer
	if err := renderWorktreeSection(&buf, v, true); err != nil {
		t.Fatalf("renderWorktreeSection: %v", err)
	}
	if strings.Contains(buf.String(), "dirty") {
		t.Errorf("renderWorktreeSection output should not mention dirty when Dirty=false:\n%s", buf.String())
	}
}

// TestRenderWorktreeShortDriver_DirtyMarker pins the same dirty-marker
// branch in the compact one-line-per-driver rendering used by the
// short status view.
func TestRenderWorktreeShortDriver_DirtyMarker(t *testing.T) {
	t.Parallel()
	v := &WorktreeView{
		DriverEntityID: "M-9082", DriverKind: "milestone", DriverStatus: "in_progress",
		DriverTitle: "Detail work", Dirty: true,
	}
	var b strings.Builder
	renderWorktreeShortDriver(&b, v, time.Now(), 120, true)
	wantMarker := render.StatusColor("dirty", string(entity.StatusInProgress), true)
	got := b.String()
	if !strings.Contains(got, wantMarker) {
		t.Errorf("renderWorktreeShortDriver output missing dirty marker %q:\n%s", wantMarker, got)
	}
}

// TestRenderWorktreeShortDriver_NotDirtyOmitsMarker is the negative
// counterpart for the compact driver line.
func TestRenderWorktreeShortDriver_NotDirtyOmitsMarker(t *testing.T) {
	t.Parallel()
	v := &WorktreeView{
		DriverEntityID: "M-9083", DriverKind: "milestone", DriverStatus: "in_progress",
		DriverTitle: "Clean work",
	}
	var b strings.Builder
	renderWorktreeShortDriver(&b, v, time.Now(), 120, true)
	if strings.Contains(b.String(), "dirty") {
		t.Errorf("renderWorktreeShortDriver output should not mention dirty when Dirty=false:\n%s", b.String())
	}
}

// TestRenderWorktreeShortTrunk_DirtyMarker pins the dirty-marker branch
// in the compact trunk (no-driver) line.
func TestRenderWorktreeShortTrunk_DirtyMarker(t *testing.T) {
	t.Parallel()
	v := &WorktreeView{Path: "/repo", Branch: "main", Dirty: true}
	var b strings.Builder
	renderWorktreeShortTrunk(&b, v, time.Now(), true)
	wantMarker := render.StatusColor("dirty", string(entity.StatusInProgress), true)
	got := b.String()
	if !strings.Contains(got, wantMarker) {
		t.Errorf("renderWorktreeShortTrunk output missing dirty marker %q:\n%s", wantMarker, got)
	}
}

// TestRenderWorktreeShortTrunk_NotDirtyOmitsMarker is the negative
// counterpart for the compact trunk line.
func TestRenderWorktreeShortTrunk_NotDirtyOmitsMarker(t *testing.T) {
	t.Parallel()
	v := &WorktreeView{Path: "/repo", Branch: "main"}
	var b strings.Builder
	renderWorktreeShortTrunk(&b, v, time.Now(), true)
	if strings.Contains(b.String(), "dirty") {
		t.Errorf("renderWorktreeShortTrunk output should not mention dirty when Dirty=false:\n%s", b.String())
	}
}
