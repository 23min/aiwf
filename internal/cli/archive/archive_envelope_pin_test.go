package archive_test

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/23min/aiwf/internal/cli/archive"
	"github.com/23min/aiwf/internal/cli/cliutil"
	"github.com/23min/aiwf/internal/cli/cliutil/testutil"
)

// archive_envelope_pin_test.go pins archive.Run's envelope bytes
// (M-0271/AC-2) before its failArchive/emitArchiveEnvelope/
// withCommitSHA triad is deleted in favor of cliutil.FinishVerbOutcome
// — the text-mode dry-run move listing, the text-mode applied subject
// line, and the JSON applied envelope's commit_sha (a shape the
// existing internal/cli/integration coverage doesn't exercise; it
// pins JSON NoOp and JSON dry-run only).

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

// seedOneTerminalGap writes a single terminal-status gap (the
// smallest fixture that gives archive real work) and commits it.
func seedOneTerminalGap(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	runGit(t, root, "init", "-q")
	runGit(t, root, "config", "user.email", "test@example.com")
	runGit(t, root, "config", "user.name", "aiwf-test")
	full := filepath.Join(root, "work", "gaps", "G-0010-addressed-gap.md")
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	body := "---\nid: G-0010\ntitle: Addressed gap\nstatus: addressed\n---\n## What's missing\n\nFixed.\n"
	if err := os.WriteFile(full, []byte(body), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	runGit(t, root, "add", "-A")
	runGit(t, root, "commit", "-q", "-m", "seed")
	return root
}

// TestArchive_TextDryRun_ExactMoveListing pins the dry-run text
// preview verbatim: the subject-with-suffix line, then "Moves (N):"
// followed by one "  <from> -> <to>" line per move.
func TestArchive_TextDryRun_ExactMoveListing(t *testing.T) {
	root := seedOneTerminalGap(t)
	out := testutil.CaptureStdout(t, func() {
		rc := archive.Run("human/test", "", root, "", false, cliutil.OutputFormat{Format: "text"})
		if rc != cliutil.ExitOK {
			t.Errorf("rc = %d, want ExitOK", rc)
		}
	})
	want := "aiwf archive: sweep 1 entity into archive/ (1 gap) (dry-run; re-run with --apply to commit)\n\n" +
		"Per ADR-0004: sweep terminal-status entities into per-kind archive/.\n\n" +
		"Per-kind counts:\n  gap       1 entity\n\n" +
		"Affected ids:\n  G-0010\n\n" +
		"Moves (1):\n" +
		"  work/gaps/G-0010-addressed-gap.md -> work/gaps/archive/G-0010-addressed-gap.md\n"
	if string(out) != want {
		t.Errorf("stdout =\n%q\nwant\n%q", out, want)
	}
}

// TestArchive_TextApply_ExactSubjectLine pins the applied text output:
// exactly the plan's subject, no dry-run suffix.
func TestArchive_TextApply_ExactSubjectLine(t *testing.T) {
	root := seedOneTerminalGap(t)
	out := testutil.CaptureStdout(t, func() {
		rc := archive.Run("human/test", "", root, "", true, cliutil.OutputFormat{Format: "text"})
		if rc != cliutil.ExitOK {
			t.Errorf("rc = %d, want ExitOK", rc)
		}
	})
	want := "aiwf archive: sweep 1 entity into archive/ (1 gap)\n"
	if string(out) != want {
		t.Errorf("stdout = %q, want %q", out, want)
	}
}

// TestArchive_JSONApply_CarriesCommitSHA pins the applied JSON
// envelope: status ok, result.subject is the plan's (non-dry-run)
// subject, and metadata.commit_sha is the resulting commit's sha — a
// shape no existing test exercised (JSON coverage was NoOp and
// dry-run only).
func TestArchive_JSONApply_CarriesCommitSHA(t *testing.T) {
	root := seedOneTerminalGap(t)
	out := testutil.CaptureStdout(t, func() {
		rc := archive.Run("human/test", "", root, "", true, cliutil.OutputFormat{Format: "json"})
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
	if env.Result.Subject != "aiwf archive: sweep 1 entity into archive/ (1 gap)" {
		t.Errorf("result.subject = %q", env.Result.Subject)
	}
	sha, _ := env.Metadata["commit_sha"].(string)
	if sha == "" {
		t.Error("metadata.commit_sha is empty, want the resulting commit sha")
	}
}
