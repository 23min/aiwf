package rewidth_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/23min/aiwf/internal/cli/cliutil"
	"github.com/23min/aiwf/internal/cli/cliutil/testutil"
	"github.com/23min/aiwf/internal/cli/rewidth"
)

// M-0253/AC-1 backfill: rewidth.Run's two flagged branches (base = the
// commit before M-0238/AC-3's rename) are both `out.JSON()`
// output-format checks — the NoOp-result envelope and the dry-run
// envelope — not error-handling guards. Every existing rewidth test
// (this package's own rewidth_test.go and
// internal/cli/integration/rewidth_cmd_test.go) drives text output
// only; `aiwf rewidth` is also explicitly excluded from
// format_coverage_test.go's generic `--format json` sweep ("bespoke
// multi-commit output; not via FinishVerb — G-0169"), so no other
// test reaches either branch. Both are trivially triggerable with
// --format json, so both get real tests below rather than a
// `//coverage:ignore`.
//
// Both tests capture os.Stdout (a process global), so neither calls
// t.Parallel — see setup_test.go's serial-test ledger.

// jsonEnvelope is the subset of render.Envelope these tests assert
// against.
type jsonEnvelope struct {
	Status string         `json:"status"`
	Result map[string]any `json:"result"`
}

// TestRun_NoOpJSON covers the NoOp-result branch's out.JSON() arm: an
// empty tree (no work/ dirs at all) needs no rewidth, and with
// --format json the result is an "ok" envelope carrying the no-op
// message as its result.subject, not a plain-text Println.
func TestRun_NoOpJSON(t *testing.T) {
	root := t.TempDir()
	out := cliutil.OutputFormat{Format: "json"}

	stdout := testutil.CaptureStdout(t, func() {
		rc := rewidth.Run("human/test", "", root, false, false, out)
		if rc != cliutil.ExitOK {
			t.Errorf("rc = %d, want ExitOK", rc)
		}
	})

	var env jsonEnvelope
	if err := json.Unmarshal(stdout, &env); err != nil {
		t.Fatalf("unmarshal envelope: %v\nstdout: %s", err, stdout)
	}
	if env.Status != "ok" {
		t.Errorf("status = %q, want %q", env.Status, "ok")
	}
	if subject, _ := env.Result["subject"].(string); subject == "" {
		t.Error("result.subject empty, want the no-op message")
	}
}

// TestRun_DryRunJSON covers the dry-run branch's out.JSON() arm: a
// tree with one narrow-width entity produces a real (non-NoOp) plan,
// and dry-run mode (no --apply) with --format json emits the plan's
// subject as an "ok" envelope instead of rewidth's human-readable
// printRewidthDryRun summary.
func TestRun_DryRunJSON(t *testing.T) {
	root := t.TempDir()
	gapDir := filepath.Join(root, "work", "gaps")
	if err := os.MkdirAll(gapDir, 0o755); err != nil {
		t.Fatal(err)
	}
	body := "---\nid: G-099\ntitle: Some gap\nstatus: open\n---\n## What's missing\n\nNo refs.\n"
	if err := os.WriteFile(filepath.Join(gapDir, "G-099-some-gap.md"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	out := cliutil.OutputFormat{Format: "json"}

	stdout := testutil.CaptureStdout(t, func() {
		rc := rewidth.Run("human/test", "", root, false, false, out)
		if rc != cliutil.ExitOK {
			t.Errorf("rc = %d, want ExitOK", rc)
		}
	})

	var env jsonEnvelope
	if err := json.Unmarshal(stdout, &env); err != nil {
		t.Fatalf("unmarshal envelope: %v\nstdout: %s", err, stdout)
	}
	if env.Status != "ok" {
		t.Errorf("status = %q, want %q", env.Status, "ok")
	}
	if subject, _ := env.Result["subject"].(string); subject == "" {
		t.Error("result.subject empty, want the dry-run plan subject")
	}
}
