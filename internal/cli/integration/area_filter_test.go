package integration

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/cli"
	"github.com/23min/aiwf/internal/cli/cliutil"
	"github.com/23min/aiwf/internal/cli/cliutil/testutil"
	"github.com/23min/aiwf/internal/cli/list"
	"github.com/23min/aiwf/internal/tree"
)

// setupAreaTree writes a multi-workstream planning tree into a fresh
// tempdir and loads it (no git — tree.Load reads files directly, which
// keeps the area-filter unit tests fast). The shape, by effective area:
//
//	platform: E-0001 (+M-0001 derived), E-0004 (proposed), G-0001, ADR-0001
//	billing:  E-0002 (+M-0002 derived), D-0001
//	untagged: E-0003 (+M-0003 derived), G-0002
//
// aiwf.yaml declares {platform, billing}. Shared by the list, status,
// and show area-filter unit tests (E-0043, M-0174).
func setupAreaTree(t *testing.T) (string, *tree.Tree) {
	t.Helper()
	root := t.TempDir()
	w := func(rel, content string) { mustWriteFile(t, filepath.Join(root, rel), content) }
	w("aiwf.yaml", "areas:\n  members:\n    - platform\n    - billing\n")
	w("work/epics/E-0001-platform/epic.md", "---\nid: E-0001\ntitle: Platform epic\nstatus: active\narea: platform\n---\n")
	w("work/epics/E-0001-platform/M-0001-cache.md", "---\nid: M-0001\ntitle: Platform milestone\nstatus: in_progress\nparent: E-0001\nacs:\n  - id: AC-1\n    title: derives platform from parent epic\n    status: open\n---\n")
	w("work/epics/E-0002-billing/epic.md", "---\nid: E-0002\ntitle: Billing epic\nstatus: active\narea: billing\n---\n")
	w("work/epics/E-0002-billing/M-0002-invoice.md", "---\nid: M-0002\ntitle: Billing milestone\nstatus: in_progress\nparent: E-0002\n---\n")
	w("work/epics/E-0003-untagged/epic.md", "---\nid: E-0003\ntitle: Untagged epic\nstatus: active\n---\n")
	w("work/epics/E-0003-untagged/M-0003-misc.md", "---\nid: M-0003\ntitle: Untagged milestone\nstatus: in_progress\nparent: E-0003\n---\n")
	w("work/epics/E-0004-planned/epic.md", "---\nid: E-0004\ntitle: Planned platform epic\nstatus: proposed\narea: platform\n---\n")
	w("work/gaps/G-0001-leak.md", "---\nid: G-0001\ntitle: Platform gap\nstatus: open\narea: platform\n---\n")
	w("work/gaps/G-0002-misc.md", "---\nid: G-0002\ntitle: Untagged gap\nstatus: open\n---\n")
	w("work/decisions/D-0001-choice.md", "---\nid: D-0001\ntitle: Billing decision\nstatus: proposed\narea: billing\n---\n")
	w("docs/adr/ADR-0001-shape.md", "---\nid: ADR-0001\ntitle: Platform ADR\nstatus: proposed\narea: platform\n---\n")

	tr, _, err := tree.Load(context.Background(), root)
	if err != nil {
		t.Fatalf("tree.Load: %v", err)
	}
	return root, tr
}

// listIDsByArea returns the set of ids `aiwf list --area area` yields,
// with archived=true so terminality never interferes — the test isolates
// the area axis.
func listIDsByArea(tr *tree.Tree, area string) map[string]bool {
	got := map[string]bool{}
	for _, r := range list.BuildListRows(tr, "", "", "", area, true) {
		got[r.ID] = true
	}
	return got
}

// TestBuildListRows_AreaFilter pins M-0174/AC-1: `aiwf list --area`
// returns exactly the entities whose effective area matches — root kinds
// by their own field (epic, gap, decision, ADR), milestones by parent-
// epic derivation (M-0001 → E-0001's platform). An empty area applies no
// filter.
func TestBuildListRows_AreaFilter(t *testing.T) {
	t.Parallel()
	_, tr := setupAreaTree(t)

	plat := listIDsByArea(tr, "platform")
	wantPlat := []string{"E-0001", "M-0001", "E-0004", "G-0001", "ADR-0001"}
	assertExactIDSet(t, "platform", plat, wantPlat)

	bill := listIDsByArea(tr, "billing")
	wantBill := []string{"E-0002", "M-0002", "D-0001"}
	assertExactIDSet(t, "billing", bill, wantBill)

	// Empty area applies no filter: untagged entities are present.
	all := listIDsByArea(tr, "")
	for _, id := range []string{"E-0003", "M-0003", "G-0002"} {
		if !all[id] {
			t.Errorf("empty --area should not filter; %s missing from %v", id, all)
		}
	}
}

// TestBuildListRows_ExcludesUntagged pins M-0174/AC-6 on the list
// surface: untagged entities (effective area "") never match a specific
// `--area`, so they are excluded from every named-area filter and surface
// only under the no-filter view. (Grouping the untagged complement under
// the default label is M-0175, not this milestone.)
func TestBuildListRows_ExcludesUntagged(t *testing.T) {
	t.Parallel()
	_, tr := setupAreaTree(t)

	untagged := []string{"E-0003", "M-0003", "G-0002"}
	for _, area := range []string{"platform", "billing"} {
		got := listIDsByArea(tr, area)
		for _, id := range untagged {
			if got[id] {
				t.Errorf("--area %s should exclude untagged %s; got %v", area, id, got)
			}
		}
	}
}

// TestRunList_AreaViaDispatcher pins the list dispatcher seam for AC-1
// (the --area flag flows through cli.Execute and filters) and AC-5 (an
// undeclared value notes to stderr, yields an empty result, exits 0).
func TestRunList_AreaViaDispatcher(t *testing.T) {
	root := setupAreaRepo(t) // declares {platform, billing}, git-backed
	mustRun(t, "add", "epic", "--title", "Platform work", "--area", "platform", "--actor", "human/test", "--root", root)
	mustRun(t, "add", "epic", "--title", "Untagged work", "--actor", "human/test", "--root", root)

	t.Run("declared area filters through the dispatcher", func(t *testing.T) {
		rc, stdout, _ := testutil.CaptureRun(t, func() int {
			return cli.Execute([]string{"list", "--area", "platform", "--format", "json", "--root", root})
		})
		if rc != cliutil.ExitOK {
			t.Fatalf("rc=%d, want ExitOK", rc)
		}
		if !strings.Contains(stdout, "E-0001") {
			t.Errorf("platform filter should include E-0001:\n%s", stdout)
		}
		if strings.Contains(stdout, "E-0002") {
			t.Errorf("platform filter should exclude untagged E-0002:\n%s", stdout)
		}
	})

	t.Run("undeclared value: note + empty result + exit 0", func(t *testing.T) {
		rc, stdout, stderr := testutil.CaptureRun(t, func() int {
			return cli.Execute([]string{"list", "--area", "nonsense", "--format", "json", "--root", root})
		})
		if rc != cliutil.ExitOK {
			t.Errorf("undeclared --area rc=%d, want ExitOK (reads are non-destructive)", rc)
		}
		if !strings.Contains(stderr, "nonsense") {
			t.Errorf("stderr should carry the undeclared-area note:\n%s", stderr)
		}
		if strings.Contains(stdout, "E-0001") || strings.Contains(stdout, "E-0002") {
			t.Errorf("undeclared area should match nothing:\n%s", stdout)
		}
	})
}

// assertExactIDSet fails unless got contains exactly the want ids.
func assertExactIDSet(t *testing.T, label string, got map[string]bool, want []string) {
	t.Helper()
	for _, id := range want {
		if !got[id] {
			t.Errorf("%s: missing %s (got %v)", label, id, got)
		}
	}
	if len(got) != len(want) {
		t.Errorf("%s: got %d ids %v, want exactly %d %v", label, len(got), got, len(want), want)
	}
}
