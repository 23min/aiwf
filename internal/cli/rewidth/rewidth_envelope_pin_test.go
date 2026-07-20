package rewidth_test

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/23min/aiwf/internal/cli/cliutil"
	"github.com/23min/aiwf/internal/cli/cliutil/testutil"
	"github.com/23min/aiwf/internal/cli/rewidth"
)

// rewidth_envelope_pin_test.go pins rewidth.Run's envelope bytes
// (M-0271/AC-2) before its failRewidth/emitRewidthEnvelope/
// withCommitSHA triad is deleted in favor of cliutil.FinishVerbOutcome
// — the text-mode dry-run operations listing, the text-mode applied
// subject line, and the JSON applied envelope's commit_sha (no
// existing test in internal/cli/integration asserts JSON envelope
// bytes for rewidth at all).

func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(),
		"GIT_AUTHOR_NAME=aiwf-test", "GIT_AUTHOR_EMAIL=test@example.com",
		"GIT_COMMITTER_NAME=aiwf-test", "GIT_COMMITTER_EMAIL=test@example.com")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}

// seedOneNarrowGap writes a single narrow-width gap (the smallest
// fixture that gives rewidth real work) and commits it.
func seedOneNarrowGap(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	runGit(t, root, "init", "-q")
	runGit(t, root, "config", "user.email", "test@example.com")
	runGit(t, root, "config", "user.name", "aiwf-test")
	full := filepath.Join(root, "work", "gaps", "G-099-some-gap.md")
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	body := "---\nid: G-099\ntitle: Some gap\nstatus: open\n---\n## What's missing\n\nNo refs.\n"
	if err := os.WriteFile(full, []byte(body), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	runGit(t, root, "add", "-A")
	runGit(t, root, "commit", "-q", "-m", "seed")
	return root
}

// TestRewidth_TextDryRun_ExactOperationsListing pins the dry-run text
// preview verbatim.
func TestRewidth_TextDryRun_ExactOperationsListing(t *testing.T) {
	root := seedOneNarrowGap(t)
	out := testutil.CaptureStdout(t, func() {
		rc := rewidth.Run("human/test", "", root, false, true, cliutil.OutputFormat{Format: "text"})
		if rc != cliutil.ExitOK {
			t.Errorf("rc = %d, want ExitOK", rc)
		}
	})
	want := "aiwf rewidth: 1 rename(s), 1 body rewrite(s) (dry-run; re-run with --apply to commit)\n\n" +
		"Per ADR-0008: canonicalize narrow-width entity ids to 4-digit form.\n\n" +
		"Renames:\n  G-  1 file(s)\n\n" +
		"Body rewrites: 1 file(s)\n\n" +
		"Operations:\n" +
		"  rename  work/gaps/G-099-some-gap.md -> work/gaps/G-0099-some-gap.md\n" +
		"  rewrite work/gaps/G-0099-some-gap.md (76 bytes)\n"
	if string(out) != want {
		t.Errorf("stdout =\n%q\nwant\n%q", out, want)
	}
}

// TestRewidth_TextApply_ExactSubjectLine pins the applied text output.
func TestRewidth_TextApply_ExactSubjectLine(t *testing.T) {
	root := seedOneNarrowGap(t)
	out := testutil.CaptureStdout(t, func() {
		rc := rewidth.Run("human/test", "", root, true, true, cliutil.OutputFormat{Format: "text"})
		if rc != cliutil.ExitOK {
			t.Errorf("rc = %d, want ExitOK", rc)
		}
	})
	want := "aiwf rewidth: 1 rename(s), 1 body rewrite(s)\n"
	if string(out) != want {
		t.Errorf("stdout = %q, want %q", out, want)
	}
}

// TestRewidth_JSONApply_CarriesCommitSHA pins the applied JSON
// envelope — a shape no existing test exercised.
func TestRewidth_JSONApply_CarriesCommitSHA(t *testing.T) {
	root := seedOneNarrowGap(t)
	out := testutil.CaptureStdout(t, func() {
		rc := rewidth.Run("human/test", "", root, true, true, cliutil.OutputFormat{Format: "json"})
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
	if env.Result.Subject != "aiwf rewidth: 1 rename(s), 1 body rewrite(s)" {
		t.Errorf("result.subject = %q", env.Result.Subject)
	}
	sha, _ := env.Metadata["commit_sha"].(string)
	if sha == "" {
		t.Error("metadata.commit_sha is empty, want the resulting commit sha")
	}
}

// TestRewidth_JSONDryRun_NoCommitSHA pins the dry-run JSON envelope: a
// subject with the dry-run suffix and no commit_sha key.
func TestRewidth_JSONDryRun_NoCommitSHA(t *testing.T) {
	root := seedOneNarrowGap(t)
	out := testutil.CaptureStdout(t, func() {
		rc := rewidth.Run("human/test", "", root, false, true, cliutil.OutputFormat{Format: "json"})
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
	want := "aiwf rewidth: 1 rename(s), 1 body rewrite(s) (dry-run; re-run with --apply to commit)"
	if env.Result.Subject != want {
		t.Errorf("result.subject = %q, want %q", env.Result.Subject, want)
	}
	if _, ok := env.Metadata["commit_sha"]; ok {
		t.Errorf("metadata carries commit_sha on a dry-run: %+v", env.Metadata)
	}
}
