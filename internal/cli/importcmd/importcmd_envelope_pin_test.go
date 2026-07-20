package importcmd_test

import (
	"encoding/json"
	"os"
	"os/exec"
	"testing"

	"github.com/23min/aiwf/internal/cli/cliutil"
	"github.com/23min/aiwf/internal/cli/cliutil/testutil"
	"github.com/23min/aiwf/internal/cli/importcmd"
)

// seedGitRoot git-inits t.TempDir() so a real `--apply` invocation
// (which shells out to git add/commit) has a repository to write into.
func seedGitRoot(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	for _, args := range [][]string{
		{"init", "-q"},
		{"config", "user.email", "test@example.com"},
		{"config", "user.name", "aiwf-test"},
	} {
		cmd := exec.Command("git", args...)
		cmd.Dir = root
		cmd.Env = append(os.Environ(),
			"GIT_AUTHOR_NAME=aiwf-test", "GIT_AUTHOR_EMAIL=test@example.com",
			"GIT_COMMITTER_NAME=aiwf-test", "GIT_COMMITTER_EMAIL=test@example.com")
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	return root
}

// importcmd_envelope_pin_test.go pins importcmd.Run's envelope bytes
// (M-0271/AC-2) before its failImport/emitImportEnvelope/withCommitSHA
// triad is deleted in favor of cliutil.FinishVerbOutcome — the
// multi-Plan-specific shapes no existing test asserted byte-for-byte:
// the text-mode per-plan subject-line loop (batched single-plan and
// per-entity multi-plan), the applied JSON envelope's aggregate
// "N entities created" subject + commit_sha (the last plan's), and the
// dry-run text preview's exact multi-line shape.

const twoEntityPerEntityManifest = `version: 1
commit:
  mode: per-entity
entities:
  - kind: epic
    id: E-0001
    frontmatter: {title: "Cake", status: active}
  - kind: gap
    id: G-0001
    frontmatter: {title: "Icing", status: open}
`

// TestImport_TextApply_BatchedSinglePlan_OneSubjectLine pins the
// default (batched) commit mode's text-apply output: one line, the
// single Plan's own subject (not any aggregate summary).
func TestImport_TextApply_BatchedSinglePlan_OneSubjectLine(t *testing.T) {
	root := seedGitRoot(t)
	manifest := writeManifest(t, root, singleEntityManifest)
	out := testutil.CaptureStdout(t, func() {
		rc := importcmd.Run(manifest, root, "human/test", "", "", false, cliutil.OutputFormat{Format: "text"})
		if rc != cliutil.ExitOK {
			t.Errorf("rc = %d, want ExitOK", rc)
		}
	})
	want := "aiwf import 1 entities\n"
	if string(out) != want {
		t.Errorf("stdout = %q, want %q", out, want)
	}
}

// TestImport_TextApply_PerEntityMultiPlan_OneLinePerPlan pins the
// per-entity commit mode's text-apply output: one line per applied
// Plan, each Plan's own subject — the multi-Plan text loop no
// existing test asserted byte-for-byte.
func TestImport_TextApply_PerEntityMultiPlan_OneLinePerPlan(t *testing.T) {
	root := seedGitRoot(t)
	manifest := writeManifest(t, root, twoEntityPerEntityManifest)
	out := testutil.CaptureStdout(t, func() {
		rc := importcmd.Run(manifest, root, "human/test", "", "", false, cliutil.OutputFormat{Format: "text"})
		if rc != cliutil.ExitOK {
			t.Errorf("rc = %d, want ExitOK", rc)
		}
	})
	want := "aiwf import epic E-0001 \"Cake\"\naiwf import gap G-0001 \"Icing\"\n"
	if string(out) != want {
		t.Errorf("stdout = %q, want %q", out, want)
	}
}

// TestImport_JSONApply_AggregateSubjectAndLastPlanSHA pins the applied
// JSON envelope: the aggregate "N entities created" subject (not any
// individual Plan's subject) and metadata.commit_sha carrying the
// batch's last commit.
func TestImport_JSONApply_AggregateSubjectAndLastPlanSHA(t *testing.T) {
	root := seedGitRoot(t)
	manifest := writeManifest(t, root, twoEntityPerEntityManifest)
	out := testutil.CaptureStdout(t, func() {
		rc := importcmd.Run(manifest, root, "human/test", "", "", false, cliutil.OutputFormat{Format: "json"})
		if rc != cliutil.ExitOK {
			t.Errorf("rc = %d, want ExitOK", rc)
		}
	})
	var env struct {
		Status string `json:"status"`
		Result struct {
			Subject string `json:"subject"`
		} `json:"result"`
		Metadata map[string]any `json:"metadata"`
	}
	if err := json.Unmarshal(out, &env); err != nil {
		t.Fatalf("stdout not JSON: %v\n%s", err, out)
	}
	if env.Status != "ok" {
		t.Errorf("status = %q, want ok", env.Status)
	}
	if env.Result.Subject != "aiwf import: 2 entities created" {
		t.Errorf("result.subject = %q", env.Result.Subject)
	}
	sha, _ := env.Metadata["commit_sha"].(string)
	if sha == "" {
		t.Error("metadata.commit_sha is empty, want the batch's last commit sha")
	}
	if env.Metadata["imported_count"] != float64(2) {
		t.Errorf("metadata.imported_count = %v, want 2", env.Metadata["imported_count"])
	}
	ids, _ := env.Metadata["entity_ids"].([]any)
	if len(ids) != 2 {
		t.Errorf("metadata.entity_ids = %v, want 2 entries", env.Metadata["entity_ids"])
	}
}

// TestImport_TextDryRun_ExactPreview pins the dry-run text preview
// verbatim: the per-plan subject and write-op lines, plus the trailing
// completion hint.
func TestImport_TextDryRun_ExactPreview(t *testing.T) {
	root := t.TempDir()
	manifest := writeManifest(t, root, singleEntityManifest)
	out := testutil.CaptureStdout(t, func() {
		rc := importcmd.Run(manifest, root, "human/test", "", "", true, cliutil.OutputFormat{Format: "text"})
		if rc != cliutil.ExitOK {
			t.Errorf("rc = %d, want ExitOK", rc)
		}
	})
	want := "aiwf import: dry-run — 1 plan(s) would land:\n" +
		"  aiwf import 1 entities\n" +
		"    write work/epics/E-0001-cake/epic.md (47 bytes)\n" +
		"\naiwf import: dry-run complete. Re-run without --dry-run to apply.\n"
	if string(out) != want {
		t.Errorf("stdout =\n%q\nwant\n%q", out, want)
	}
}
