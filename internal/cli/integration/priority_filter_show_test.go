package integration

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/cli"
	"github.com/23min/aiwf/internal/cli/cliutil"
	"github.com/23min/aiwf/internal/cli/cliutil/testutil"
	"github.com/23min/aiwf/internal/cli/show"
	"github.com/23min/aiwf/internal/tree"
)

// TestRunShow_PriorityEnvelope pins M-0263/AC-3: aiwf show surfaces a
// gap's or decision's priority on both the JSON envelope's entity payload
// and the text rendering.
func TestRunShow_PriorityEnvelope(t *testing.T) {
	root := setupCLITestRepo(t)
	mustRun(t, "init", "--root", root, "--actor", "human/test", "--skip-hook")
	mustRun(t, "add", "gap", "--body", fixtureGapBody, "--title", "Urgent gap", "--priority", "urgent", "--actor", "human/test", "--root", root)
	mustRun(t, "add", "gap", "--body", fixtureGapBody, "--title", "Unprioritized gap", "--actor", "human/test", "--root", root)

	t.Run("json envelope carries priority", func(t *testing.T) {
		rc, stdout, _ := testutil.CaptureRun(t, func() int {
			return cli.Execute([]string{"show", "G-0001", "--format", "json", "--root", root})
		})
		if rc != cliutil.ExitOK {
			t.Fatalf("rc=%d, want ExitOK", rc)
		}
		var env struct {
			Result show.ShowView `json:"result"`
		}
		if err := json.Unmarshal([]byte(stdout), &env); err != nil {
			t.Fatalf("unmarshal: %v\n%s", err, stdout)
		}
		if env.Result.Priority != "urgent" {
			t.Errorf("Result.Priority = %q, want urgent", env.Result.Priority)
		}
	})

	t.Run("text rendering surfaces priority", func(t *testing.T) {
		rc, stdout, _ := testutil.CaptureRun(t, func() int {
			return cli.Execute([]string{"show", "G-0001", "--root", root})
		})
		if rc != cliutil.ExitOK {
			t.Fatalf("rc=%d, want ExitOK", rc)
		}
		if !strings.Contains(stdout, "priority: urgent") {
			t.Errorf("text output should surface the priority:\n%s", stdout)
		}
	})

	t.Run("absent priority omits the field", func(t *testing.T) {
		rc, stdout, _ := testutil.CaptureRun(t, func() int {
			return cli.Execute([]string{"show", "G-0002", "--format", "json", "--root", root})
		})
		if rc != cliutil.ExitOK {
			t.Fatalf("rc=%d, want ExitOK", rc)
		}
		var env struct {
			Result json.RawMessage `json:"result"`
		}
		if err := json.Unmarshal([]byte(stdout), &env); err != nil {
			t.Fatalf("unmarshal: %v\n%s", err, stdout)
		}
		if strings.Contains(string(env.Result), `"priority"`) {
			t.Errorf("unprioritized gap should omit the priority field:\n%s", env.Result)
		}
	})

	t.Run("absent priority omits the text segment", func(t *testing.T) {
		rc, stdout, _ := testutil.CaptureRun(t, func() int {
			return cli.Execute([]string{"show", "G-0002", "--root", root})
		})
		if rc != cliutil.ExitOK {
			t.Fatalf("rc=%d, want ExitOK", rc)
		}
		if strings.Contains(stdout, "priority:") {
			t.Errorf("unprioritized gap's text header should omit the priority segment:\n%s", stdout)
		}
	})

	t.Run("a kind that never carries priority also omits the field", func(t *testing.T) {
		mustRun(t, "add", "epic", "--title", "Some epic", "--actor", "human/test", "--root", root)
		rc, stdout, _ := testutil.CaptureRun(t, func() int {
			return cli.Execute([]string{"show", "E-0001", "--format", "json", "--root", root})
		})
		if rc != cliutil.ExitOK {
			t.Fatalf("rc=%d, want ExitOK", rc)
		}
		if strings.Contains(stdout, `"priority"`) {
			t.Errorf("an epic should never carry a priority field:\n%s", stdout)
		}
	})
}

// TestBuildShowView_CrossBranchResolved_SurfacesPriority pins the
// cross-branch-resolved half of AC-3 with a real value assertion (not
// just statement coverage): a gap minted on a sibling branch, absent
// from the checked-out branch, resolves live via BlobReader and its
// Priority rides along with the rest of the resolved content —
// mirroring TestBuildShowView_CrossBranchResolvesAndLabelsContent_M0260AC1AC2's
// shape with a prioritized gap instead of a milestone.
func TestBuildShowView_CrossBranchResolved_SurfacesPriority(t *testing.T) {
	root := setupCLITestRepo(t)
	writeAndCommit(t, root, "README.md", "# seed\n", "seed")

	if err := osExec(t, root, "git", "checkout", "-q", "-b", "sibling"); err != nil {
		t.Fatalf("checkout sibling: %v", err)
	}
	gBody := "---\nid: G-0100\ntitle: Sibling Gap\nstatus: open\npriority: urgent\n---\n\n## Problem\n\ndescribed.\n"
	writeAndCommit(t, root, "work/gaps/G-0100-sibling.md", gBody, "sibling: mint G-0100")
	if err := osExec(t, root, "git", "checkout", "-q", "main"); err != nil {
		t.Fatalf("checkout main: %v", err)
	}

	ctx := context.Background()
	tr, _, err := tree.Load(ctx, root)
	if err != nil {
		t.Fatalf("tree.Load: %v", err)
	}
	if tr.ByID("G-0100") != nil {
		t.Fatal("G-0100 must be absent from the local (main) tree for this fixture")
	}

	view, ok, err := show.BuildShowView(ctx, root, tr, nil, "G-0100", 5)
	if err != nil {
		t.Fatalf("BuildShowView: %v", err)
	}
	if !ok {
		t.Fatal("BuildShowView: not found, want cross-branch resolution")
	}
	if view.CrossBranch == nil || view.CrossBranch.Collision {
		t.Fatalf("CrossBranch = %+v, want a resolved (non-collision) cross-branch view", view.CrossBranch)
	}
	if view.Priority != "urgent" {
		t.Errorf("Priority = %q, want urgent (resolved from the sibling branch's content)", view.Priority)
	}
}
