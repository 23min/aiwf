package integration

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/cli"
	"github.com/23min/aiwf/internal/cli/cliutil"
	"github.com/23min/aiwf/internal/cli/cliutil/testutil"
	"github.com/23min/aiwf/internal/cli/show"
)

// TestAreaMissLine pins the show area-miss message shapes (M-0174/AC-3):
// a differently-tagged entity names its actual area; an untagged entity
// gets distinct wording. Pure string function — no fixture.
func TestAreaMissLine(t *testing.T) {
	t.Parallel()
	if got := show.AreaMissLine("E-0001", "platform", "billing"); !strings.Contains(got, "platform") || !strings.Contains(got, "billing") {
		t.Errorf("tagged miss = %q, want it to name both areas", got)
	}
	got := show.AreaMissLine("E-0003", "", "platform")
	if !strings.Contains(got, "untagged") || !strings.Contains(got, "platform") {
		t.Errorf("untagged miss = %q, want 'untagged' + requested area", got)
	}
}

// setupAreaShowRepo builds a git-backed repo (areas {platform, billing})
// with E-0001 platform (+ milestone M-0001 + AC-1), E-0002 billing,
// E-0003 untagged. Returns the root.
func setupAreaShowRepo(t *testing.T) string {
	t.Helper()
	root := setupAreaRepo(t)
	mustRun(t, "add", "epic", "--title", "Platform", "--area", "platform", "--actor", "human/test", "--root", root)
	mustRun(t, "add", "epic", "--title", "Billing", "--area", "billing", "--actor", "human/test", "--root", root)
	mustRun(t, "add", "epic", "--title", "Untagged", "--actor", "human/test", "--root", root)
	mustRun(t, "add", "milestone", "--epic", "E-0001", "--tdd", "none", "--title", "Cache", "--actor", "human/test", "--root", root)
	mustRun(t, "add", "ac", "M-0001", "--title", "first criterion", "--actor", "human/test", "--root", root)
	return root
}

// TestRunShow_AreaPredicate pins M-0174/AC-3 through the dispatcher: with
// --area, `aiwf show` is a predicate over the single named entity — shown
// when its effective area matches, hidden with a one-line note (exit 0)
// when it differs, including the untagged and composite-AC cases. AC-5's
// undeclared-value note is exercised on the show surface too.
func TestRunShow_AreaPredicate(t *testing.T) {
	root := setupAreaShowRepo(t)

	t.Run("match renders the full view", func(t *testing.T) {
		rc, stdout, _ := testutil.CaptureRun(t, func() int {
			return cli.Execute([]string{"show", "E-0001", "--area", "platform", "--root", root})
		})
		if rc != cliutil.ExitOK {
			t.Fatalf("rc=%d", rc)
		}
		if !strings.Contains(stdout, "Findings") {
			t.Errorf("a match should render the full view (with a Findings block):\n%s", stdout)
		}
		if strings.Contains(stdout, "not \"") {
			t.Errorf("a match should not print an area-miss line:\n%s", stdout)
		}
	})

	t.Run("different area is hidden with a note, exit 0", func(t *testing.T) {
		rc, stdout, _ := testutil.CaptureRun(t, func() int {
			return cli.Execute([]string{"show", "E-0001", "--area", "billing", "--root", root})
		})
		if rc != cliutil.ExitOK {
			t.Errorf("predicate miss rc=%d, want ExitOK (entity hidden, like an empty list)", rc)
		}
		if !strings.Contains(stdout, "platform") || !strings.Contains(stdout, "billing") {
			t.Errorf("miss line should name actual + requested area:\n%s", stdout)
		}
		if strings.Contains(stdout, "Findings") {
			t.Errorf("a miss should hide the full view (no Findings block):\n%s", stdout)
		}
	})

	t.Run("untagged entity is hidden under a named area", func(t *testing.T) {
		_, stdout, _ := testutil.CaptureRun(t, func() int {
			return cli.Execute([]string{"show", "E-0003", "--area", "platform", "--root", root})
		})
		if !strings.Contains(stdout, "untagged") {
			t.Errorf("untagged entity miss should say 'untagged':\n%s", stdout)
		}
	})

	t.Run("composite AC id matches via parent epic", func(t *testing.T) {
		rc, stdout, _ := testutil.CaptureRun(t, func() int {
			return cli.Execute([]string{"show", "M-0001/AC-1", "--area", "platform", "--root", root})
		})
		if rc != cliutil.ExitOK {
			t.Fatalf("rc=%d", rc)
		}
		if !strings.Contains(stdout, "parent: M-0001") {
			t.Errorf("composite match should render the AC view:\n%s", stdout)
		}
	})

	t.Run("composite AC id misses under the wrong area", func(t *testing.T) {
		_, stdout, _ := testutil.CaptureRun(t, func() int {
			return cli.Execute([]string{"show", "M-0001/AC-1", "--area", "billing", "--root", root})
		})
		if !strings.Contains(stdout, "M-0001/AC-1") || !strings.Contains(stdout, "platform") {
			t.Errorf("composite miss should name the AC id + its derived area:\n%s", stdout)
		}
	})

	t.Run("undeclared value notes to stderr and exits 0", func(t *testing.T) {
		rc, _, stderr := testutil.CaptureRun(t, func() int {
			return cli.Execute([]string{"show", "E-0001", "--area", "nonsense", "--root", root})
		})
		if rc != cliutil.ExitOK {
			t.Errorf("undeclared --area rc=%d, want ExitOK", rc)
		}
		if !strings.Contains(stderr, "nonsense") {
			t.Errorf("stderr should carry the undeclared-area note:\n%s", stderr)
		}
	})

	t.Run("json miss emits a null result with filtered_out metadata", func(t *testing.T) {
		_, stdout, _ := testutil.CaptureRun(t, func() int {
			return cli.Execute([]string{"show", "E-0001", "--area", "billing", "--format", "json", "--root", root})
		})
		var env struct {
			Status   string                 `json:"status"`
			Result   json.RawMessage        `json:"result"`
			Metadata map[string]interface{} `json:"metadata"`
		}
		if err := json.Unmarshal([]byte(stdout), &env); err != nil {
			t.Fatalf("unmarshal: %v\n%s", err, stdout)
		}
		if env.Status != "ok" {
			t.Errorf("status = %q, want ok", env.Status)
		}
		if s := strings.TrimSpace(string(env.Result)); s != "null" && s != "" {
			t.Errorf("result = %s, want null (entity hidden)", s)
		}
		if env.Metadata["filtered_out"] != true {
			t.Errorf("metadata.filtered_out = %v, want true", env.Metadata["filtered_out"])
		}
	})
}
