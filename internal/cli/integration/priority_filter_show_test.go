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
