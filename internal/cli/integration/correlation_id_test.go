package integration

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/cli"
	"github.com/23min/aiwf/internal/cli/cancel"
	"github.com/23min/aiwf/internal/cli/cliutil"
	"github.com/23min/aiwf/internal/cli/cliutil/testutil"
	"github.com/23min/aiwf/internal/cli/move"
)

// TestCorrelationID_MatchesLogRunID pins M-0239/AC-1: the id NewRootCmd
// mints for an invocation shows up as both the JSON envelope's
// metadata.correlation_id and the diagnostic log line's run_id — one
// grep on either value finds the other.
func TestCorrelationID_MatchesLogRunID(t *testing.T) {
	root := setupCLITestRepo(t)
	mustRun(t, "init", "--root", root, "--actor", "human/test", "--skip-hook")
	mustRun(t, "add", "gap", "--body", "## What's missing\n\nFixture prose for test setup; not the subject under test.\n\n## Why it matters\n\nFixture prose for test setup; not the subject under test.\n", "--title", "Stale probe", "--actor", "human/test", "--root", root)

	logPath := filepath.Join(t.TempDir(), "diag.log")
	t.Setenv("AIWF_LOG", "info")
	t.Setenv("AIWF_LOG_FORMAT", "json")
	t.Setenv("AIWF_LOG_FILE", logPath)

	rc, stdout, stderr := testutil.CaptureRun(t, func() int {
		return cli.Execute([]string{"cancel", "G-0001", "--reason", "no longer needed", "--actor", "human/test", "--root", root, "--format=json"})
	})
	if rc != cliutil.ExitOK {
		t.Fatalf("aiwf cancel: rc=%d stderr=%s", rc, stderr)
	}

	var env struct {
		Metadata map[string]any `json:"metadata"`
	}
	if err := json.Unmarshal([]byte(stdout), &env); err != nil {
		t.Fatalf("stdout not JSON: %v\n%s", err, stdout)
	}
	correlationID, _ := env.Metadata["correlation_id"].(string)
	if correlationID == "" {
		t.Fatal("envelope metadata.correlation_id missing or empty")
	}

	raw, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("reading diagnostic log: %v", err)
	}
	var rec struct {
		RunID string `json:"run_id"`
	}
	if err := json.Unmarshal(raw, &rec); err != nil {
		t.Fatalf("diagnostic log %q not JSON: %v", raw, err)
	}
	if rec.RunID == "" {
		t.Fatal("run_id missing or empty from the diagnostic record")
	}
	if rec.RunID != correlationID {
		t.Errorf("run_id (%q) != envelope metadata.correlation_id (%q); the two surfaces must share one id", rec.RunID, correlationID)
	}
}

// TestCorrelationID_MoveMatchesLogRunID is
// TestCorrelationID_MatchesLogRunID's move.Run counterpart: cancel and
// move are the two verbs with both a WithVerb log call site and a JSON
// envelope today, and each reuses the invocation's id independently —
// a regression in either one's reuse needs its own test to catch it.
func TestCorrelationID_MoveMatchesLogRunID(t *testing.T) {
	root := setupCLITestRepo(t)
	mustRun(t, "init", "--root", root, "--actor", "human/test", "--skip-hook")
	mustRun(t, "add", "epic", "--title", "Source epic", "--actor", "human/test", "--root", root)
	mustRun(t, "add", "epic", "--title", "Target epic", "--actor", "human/test", "--root", root)
	mustRun(t, "add", "milestone", "--epic", "E-0001", "--tdd", "none", "--title", "Child", "--actor", "human/test", "--root", root)

	logPath := filepath.Join(t.TempDir(), "diag.log")
	t.Setenv("AIWF_LOG", "info")
	t.Setenv("AIWF_LOG_FORMAT", "json")
	t.Setenv("AIWF_LOG_FILE", logPath)

	rc, stdout, stderr := testutil.CaptureRun(t, func() int {
		return cli.Execute([]string{"move", "M-0001", "--epic", "E-0002", "--actor", "human/test", "--root", root, "--format=json"})
	})
	if rc != cliutil.ExitOK {
		t.Fatalf("aiwf move: rc=%d stderr=%s", rc, stderr)
	}

	var env struct {
		Metadata map[string]any `json:"metadata"`
	}
	if err := json.Unmarshal([]byte(stdout), &env); err != nil {
		t.Fatalf("stdout not JSON: %v\n%s", err, stdout)
	}
	correlationID, _ := env.Metadata["correlation_id"].(string)
	if correlationID == "" {
		t.Fatal("envelope metadata.correlation_id missing or empty")
	}

	raw, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("reading diagnostic log: %v", err)
	}
	var rec struct {
		RunID string `json:"run_id"`
	}
	if err := json.Unmarshal(raw, &rec); err != nil {
		t.Fatalf("diagnostic log %q not JSON: %v", raw, err)
	}
	if rec.RunID == "" {
		t.Fatal("run_id missing or empty from the diagnostic record")
	}
	if rec.RunID != correlationID {
		t.Errorf("run_id (%q) != envelope metadata.correlation_id (%q); the two surfaces must share one id", rec.RunID, correlationID)
	}
}

// TestCorrelationID_DiffersAcrossInvocations pins the "one id per
// invocation" half of AC-1: two separate cancel runs must not reuse the
// same correlation id.
func TestCorrelationID_DiffersAcrossInvocations(t *testing.T) {
	root := setupCLITestRepo(t)
	mustRun(t, "init", "--root", root, "--actor", "human/test", "--skip-hook")
	mustRun(t, "add", "gap", "--body", "## What's missing\n\nFixture prose for test setup; not the subject under test.\n\n## Why it matters\n\nFixture prose for test setup; not the subject under test.\n", "--title", "Stale probe one", "--actor", "human/test", "--root", root)
	mustRun(t, "add", "gap", "--body", "## What's missing\n\nFixture prose for test setup; not the subject under test.\n\n## Why it matters\n\nFixture prose for test setup; not the subject under test.\n", "--title", "Stale probe two", "--actor", "human/test", "--root", root)

	correlationIDOf := func(entityID string) string {
		t.Helper()
		rc, stdout, stderr := testutil.CaptureRun(t, func() int {
			return cli.Execute([]string{"cancel", entityID, "--reason", "no longer needed", "--actor", "human/test", "--root", root, "--format=json"})
		})
		if rc != cliutil.ExitOK {
			t.Fatalf("aiwf cancel %s: rc=%d stderr=%s", entityID, rc, stderr)
		}
		var env struct {
			Metadata map[string]any `json:"metadata"`
		}
		if err := json.Unmarshal([]byte(stdout), &env); err != nil {
			t.Fatalf("stdout not JSON: %v\n%s", err, stdout)
		}
		id, _ := env.Metadata["correlation_id"].(string)
		if id == "" {
			t.Fatalf("envelope metadata.correlation_id missing or empty for %s", entityID)
		}
		return id
	}

	first := correlationIDOf("G-0001")
	second := correlationIDOf("G-0002")
	if first == second {
		t.Errorf("two separate invocations minted the same correlation id (%q); expected a fresh one per invocation", first)
	}
}

// TestCorrelationID_PresentOnUnloggedMutatingVerb pins the universal
// half of AC-1's envelope wiring: a mutating verb that never calls
// logger.WithVerb at all (promote has no diagnostic-logging call site
// as of this milestone) still carries metadata.correlation_id — the id
// is threaded to every mutating verb's OutputFormat, not just the
// handful already logger-instrumented.
func TestCorrelationID_PresentOnUnloggedMutatingVerb(t *testing.T) {
	root := setupCLITestRepo(t)
	mustRun(t, "init", "--root", root, "--actor", "human/test", "--skip-hook")
	mustRun(t, "add", "gap", "--body", "## What's missing\n\nFixture prose for test setup; not the subject under test.\n\n## Why it matters\n\nFixture prose for test setup; not the subject under test.\n", "--title", "Stale probe", "--actor", "human/test", "--root", root)

	rc, stdout, stderr := testutil.CaptureRun(t, func() int {
		return cli.Execute([]string{"promote", "G-0001", "wontfix", "--actor", "human/test", "--root", root, "--format=json"})
	})
	if rc != cliutil.ExitOK {
		t.Fatalf("aiwf promote: rc=%d stderr=%s", rc, stderr)
	}
	var env struct {
		Metadata map[string]any `json:"metadata"`
	}
	if err := json.Unmarshal([]byte(stdout), &env); err != nil {
		t.Fatalf("stdout not JSON: %v\n%s", err, stdout)
	}
	if id, _ := env.Metadata["correlation_id"].(string); id == "" {
		t.Error("envelope metadata.correlation_id missing or empty on a mutating verb with no WithVerb call site")
	}
}

// TestCorrelationID_PresentAcrossMutatingVerbs spot-checks the
// remaining mechanically-edited mutating verbs beyond cancel/move/
// promote: every one of these gained an identical `out.CorrelationID
// = correlationID` line in its own NewCmd, added by the same
// mechanical pass. rename and add exercise the DecorateAndFinish-
// mediated path (same shape as promote); authorize exercises
// FinishVerb directly (DecorateAndFinish only wraps FinishVerb when a
// plan exists to gate, so this is a structurally distinct entry
// point worth checking on its own).
func TestCorrelationID_PresentAcrossMutatingVerbs(t *testing.T) {
	envelopeCorrelationID := func(t *testing.T, args ...string) string {
		t.Helper()
		rc, stdout, stderr := testutil.CaptureRun(t, func() int {
			return cli.Execute(args)
		})
		if rc != cliutil.ExitOK {
			t.Fatalf("aiwf %v: rc=%d stderr=%s", args, rc, stderr)
		}
		var env struct {
			Metadata map[string]any `json:"metadata"`
		}
		if err := json.Unmarshal([]byte(stdout), &env); err != nil {
			t.Fatalf("stdout not JSON: %v\n%s", err, stdout)
		}
		id, _ := env.Metadata["correlation_id"].(string)
		if id == "" {
			t.Fatalf("aiwf %v: envelope metadata.correlation_id missing or empty", args)
		}
		return id
	}

	t.Run("rename", func(t *testing.T) {
		root := setupCLITestRepo(t)
		mustRun(t, "init", "--root", root, "--actor", "human/test", "--skip-hook")
		mustRun(t, "add", "epic", "--title", "Home", "--actor", "human/test", "--root", root)
		mustRun(t, "add", "milestone", "--tdd", "none", "--epic", "E-0001", "--title", "Original", "--actor", "human/test", "--root", root)
		envelopeCorrelationID(t, "rename", "M-0001", "renamed-slug", "--actor", "human/test", "--root", root, "--format=json")
	})

	t.Run("add", func(t *testing.T) {
		root := setupCLITestRepo(t)
		mustRun(t, "init", "--root", root, "--actor", "human/test", "--skip-hook")
		envelopeCorrelationID(t, "add", "gap", "--title", "Spot-check", "--body", "## What's missing\n\nFixture prose.\n\n## Why it matters\n\nFixture prose.\n", "--actor", "human/test", "--root", root, "--format=json")
	})

	t.Run("authorize", func(t *testing.T) {
		root := setupCLITestRepo(t)
		mustRun(t, "init", "--root", root, "--actor", "human/test", "--skip-hook")
		mustRun(t, "add", "epic", "--title", "Adoption", "--actor", "human/test", "--root", root)
		mustRun(t, "add", "milestone", "--tdd", "none", "--epic", "E-0001", "--title", "Schema parser", "--actor", "human/test", "--root", root)
		mustRun(t, "promote", "--root", root, "--actor", "human/test", "M-0001", "in_progress")
		// The authorize verb's AI-target preflight requires a ritual-
		// shape branch checkout (M-0103) — mirrors
		// TestRender_AllPagesAreWellFormed's identical setup step.
		if out, err := testutil.RunGit(root, "checkout", "-b", "epic/E-0001-adoption"); err != nil {
			t.Fatalf("git checkout -b: %v\n%s", err, out)
		}
		envelopeCorrelationID(t, "authorize", "--root", root, "--actor", "human/test", "M-0001", "--to", "ai/claude", "--format=json")
	})

	// add-ac and acknowledge-illegal both needed a manual fix during
	// this milestone's mechanical NewCmd(correlationID) rollout — each
	// is a parent command delegating to an unexported nested-
	// subcommand constructor (newACCmd / newIllegalCmd), where the
	// mechanical sed pass initially missed threading correlationID
	// into the nested call. Both spot-checked directly rather than
	// trusted by pattern, since that's exactly the bug class this
	// shape produced once already.

	t.Run("add ac", func(t *testing.T) {
		root := setupCLITestRepo(t)
		mustRun(t, "init", "--root", root, "--actor", "human/test", "--skip-hook")
		mustRun(t, "add", "epic", "--title", "Home", "--actor", "human/test", "--root", root)
		mustRun(t, "add", "milestone", "--tdd", "none", "--epic", "E-0001", "--title", "Parent", "--actor", "human/test", "--root", root)
		envelopeCorrelationID(t, "add", "ac", "M-0001", "--title", "Spot-check criterion", "--actor", "human/test", "--root", root, "--format=json")
	})

	t.Run("acknowledge illegal", func(t *testing.T) {
		root := setupCLITestRepo(t)
		mustRun(t, "init", "--root", root, "--actor", "human/test", "--skip-hook")
		mustRun(t, "add", "gap", "--title", "Spot-check", "--body", "## What's missing\n\nFixture prose.\n\n## Why it matters\n\nFixture prose.\n", "--actor", "human/test", "--root", root)
		sha, err := testutil.RunGit(root, "rev-parse", "HEAD")
		if err != nil {
			t.Fatalf("git rev-parse HEAD: %v\n%s", err, sha)
		}
		sha = strings.TrimSpace(sha)
		envelopeCorrelationID(t, "acknowledge", "illegal", sha, "--reason", "spot-check", "--actor", "human/test", "--root", root, "--format=json")
	})
}

// TestCorrelationID_FallsBackWhenOutputFormatCarriesNone pins the
// defensive fallback in cancel.Run: a caller that builds
// cliutil.OutputFormat directly (bypassing NewCmd/Execute, so
// CorrelationID is the empty zero value) still gets a non-empty
// run_id in its diagnostic log line, rather than an empty string.
// cli.Execute always threads a real id (NewRootCmd mints one
// unconditionally), so this path is unreachable through the CLI
// surface — only a direct cancel.Run caller can exercise it.
func TestCorrelationID_FallsBackWhenOutputFormatCarriesNone(t *testing.T) {
	root := setupCLITestRepo(t)
	mustRun(t, "init", "--root", root, "--actor", "human/test", "--skip-hook")
	mustRun(t, "add", "gap", "--body", "## What's missing\n\nFixture prose for test setup; not the subject under test.\n\n## Why it matters\n\nFixture prose for test setup; not the subject under test.\n", "--title", "Stale probe", "--actor", "human/test", "--root", root)

	logPath := filepath.Join(t.TempDir(), "diag.log")
	t.Setenv("AIWF_LOG", "info")
	t.Setenv("AIWF_LOG_FORMAT", "json")
	t.Setenv("AIWF_LOG_FILE", logPath)

	rc := cancel.Run("G-0001", "human/test", "", root, "no longer needed", false, false, cliutil.OutputFormat{})
	if rc != cliutil.ExitOK {
		t.Fatalf("cancel.Run: rc=%d", rc)
	}

	raw, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("reading diagnostic log: %v", err)
	}
	var rec struct {
		RunID string `json:"run_id"`
	}
	if err := json.Unmarshal(raw, &rec); err != nil {
		t.Fatalf("diagnostic log %q not JSON: %v", raw, err)
	}
	if rec.RunID == "" {
		t.Error("run_id empty even though OutputFormat carried no CorrelationID; the fallback mint did not fire")
	}
}

// TestCorrelationID_MoveFallsBackWhenOutputFormatCarriesNone is
// TestCorrelationID_FallsBackWhenOutputFormatCarriesNone's move.Run
// counterpart — the identical fallback branch lives in a separate
// file and needs its own exercising test.
func TestCorrelationID_MoveFallsBackWhenOutputFormatCarriesNone(t *testing.T) {
	root := setupCLITestRepo(t)
	mustRun(t, "init", "--root", root, "--actor", "human/test", "--skip-hook")
	mustRun(t, "add", "epic", "--title", "Source epic", "--actor", "human/test", "--root", root)
	mustRun(t, "add", "epic", "--title", "Target epic", "--actor", "human/test", "--root", root)
	mustRun(t, "add", "milestone", "--epic", "E-0001", "--tdd", "none", "--title", "Child", "--actor", "human/test", "--root", root)

	logPath := filepath.Join(t.TempDir(), "diag.log")
	t.Setenv("AIWF_LOG", "info")
	t.Setenv("AIWF_LOG_FORMAT", "json")
	t.Setenv("AIWF_LOG_FILE", logPath)

	rc := move.Run("M-0001", "E-0002", "human/test", "", root, cliutil.OutputFormat{})
	if rc != cliutil.ExitOK {
		t.Fatalf("move.Run: rc=%d", rc)
	}

	raw, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("reading diagnostic log: %v", err)
	}
	var rec struct {
		RunID string `json:"run_id"`
	}
	if err := json.Unmarshal(raw, &rec); err != nil {
		t.Fatalf("diagnostic log %q not JSON: %v", raw, err)
	}
	if rec.RunID == "" {
		t.Error("run_id empty even though OutputFormat carried no CorrelationID; the fallback mint did not fire")
	}
}
