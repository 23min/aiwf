package cliutil

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/23min/aiwf/internal/check"
	"github.com/23min/aiwf/internal/entity"
	"github.com/23min/aiwf/internal/gitops"
	"github.com/23min/aiwf/internal/verb"
)

// apply_outcome_test.go pins FinishVerbOutcome (M-0271/AC-1) — the
// dry-run- and multi-Plan-capable generalization of FinishVerb that
// archive, rewidth, and import (M-0271/AC-2) migrate onto. FinishVerb
// itself delegates to FinishVerbOutcome (see apply.go); its own
// existing exit-code assertions (apply_test.go) plus the byte-level
// emit-helper pinning (outputformat_test.go) together prove the
// delegation didn't move any existing consumer's envelope bytes —
// these tests cover only the two genuinely new capabilities.

func seedRepo(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	runGit(t, root, "init", "-q")
	runGit(t, root, "config", "user.email", "test@example.com")
	runGit(t, root, "config", "user.name", "aiwf-test")
	if err := os.WriteFile(filepath.Join(root, "seed.txt"), []byte("seed\n"), 0o644); err != nil {
		t.Fatalf("write seed file: %v", err)
	}
	runGit(t, root, "add", "seed.txt")
	runGit(t, root, "commit", "-q", "-m", "seed")
	return root
}

// TestFinishVerbOutcome_ErrBranches mirrors TestFinishVerb_ErrBranches
// on the new entry point: same err-in, same code-out contract.
func TestFinishVerbOutcome_ErrBranches(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	out := OutputFormat{Format: "json"}

	t.Run("coded error -> ExitFindings", func(t *testing.T) {
		t.Parallel()
		codedErr := &entity.FSMTransitionError{Kind: entity.KindGap, From: entity.StatusOpen, To: "bogus"}
		code, sha := FinishVerbOutcome(context.Background(), root, "aiwf test", nil, codedErr, out)
		if code != ExitFindings || sha != "" {
			t.Errorf("code=%d sha=%q, want ExitFindings/empty", code, sha)
		}
	})

	t.Run("plain error -> ExitUsage", func(t *testing.T) {
		t.Parallel()
		code, sha := FinishVerbOutcome(context.Background(), root, "aiwf test", nil, errors.New("boom"), out)
		if code != ExitUsage || sha != "" {
			t.Errorf("code=%d sha=%q, want ExitUsage/empty", code, sha)
		}
	})
}

func TestFinishVerbOutcome_NilOutcome(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	code, sha := FinishVerbOutcome(context.Background(), root, "aiwf test", nil, nil, OutputFormat{Format: "json"})
	if code != ExitInternal || sha != "" {
		t.Errorf("code=%d sha=%q, want ExitInternal/empty", code, sha)
	}
}

func TestFinishVerbOutcome_ErrorFindings(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	outcome := &Outcome{
		Findings: []check.Finding{
			{Code: check.CodeStatusValid, Severity: check.SeverityError, Message: "bad status", EntityID: "E-0001"},
		},
		Plans: []*verb.Plan{{Subject: "unused"}},
	}
	code, sha := FinishVerbOutcome(context.Background(), root, "aiwf test", outcome, nil, OutputFormat{Format: "json"})
	if code != ExitFindings || sha != "" {
		t.Errorf("code=%d sha=%q, want ExitFindings/empty", code, sha)
	}
}

func TestFinishVerbOutcome_NoOp(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	outcome := &Outcome{NoOp: true, NoOpMessage: "already at target state"}
	code, sha := FinishVerbOutcome(context.Background(), root, "aiwf test", outcome, nil, OutputFormat{Format: "json"})
	if code != ExitOK || sha != "" {
		t.Errorf("code=%d sha=%q, want ExitOK/empty", code, sha)
	}
}

// TestFinishVerbOutcome_NoPlans covers the "validation passed but no
// plan produced" guard when Plans is empty and NoOp is false — the
// multi-Plan analogue of FinishVerb's result.Plan == nil branch.
func TestFinishVerbOutcome_NoPlans(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	code, sha := FinishVerbOutcome(context.Background(), root, "aiwf test", &Outcome{}, nil, OutputFormat{Format: "json"})
	if code != ExitInternal || sha != "" {
		t.Errorf("code=%d sha=%q, want ExitInternal/empty", code, sha)
	}
}

// TestFinishVerbOutcome_DryRun_JSON pins the new dry-run JSON shape: no
// verb.Apply call, result.subject is the caller-resolved Subject
// (falling back to the last Plan's Subject when unset), metadata
// carries no commit_sha, and the returned sha is empty.
//
// SERIAL (do not add t.Parallel): uses captureStdStreams, which
// redirects the process-global os.Stdout/os.Stderr (see
// setup_test.go's serial-skip-list comment).
func TestFinishVerbOutcome_DryRun_JSON(t *testing.T) {
	root := t.TempDir() // never touched — dry-run must not call verb.Apply

	t.Run("explicit Subject override", func(t *testing.T) {
		outcome := &Outcome{
			DryRun:   true,
			Plans:    []*verb.Plan{{Subject: "sweep 3 entities"}},
			Subject:  "sweep 3 entities (dry-run; re-run with --apply to commit)",
			Metadata: map[string]any{"swept_count": 3},
		}
		out, errOut := captureStdStreams(t, func() {
			code, sha := FinishVerbOutcome(context.Background(), root, "aiwf archive", outcome, nil, OutputFormat{Format: "json"})
			if code != ExitOK || sha != "" {
				t.Errorf("code=%d sha=%q, want ExitOK/empty", code, sha)
			}
		})
		if errOut != "" {
			t.Errorf("stderr must be empty; got %q", errOut)
		}
		var env struct {
			Status   string                   `json:"status"`
			Result   struct{ Subject string } `json:"result"`
			Metadata map[string]any           `json:"metadata"`
		}
		if err := json.Unmarshal([]byte(out), &env); err != nil {
			t.Fatalf("stdout not JSON: %v\n%s", err, out)
		}
		if env.Status != "ok" {
			t.Errorf("status = %q, want ok", env.Status)
		}
		if env.Result.Subject != outcome.Subject {
			t.Errorf("result.subject = %q, want %q", env.Result.Subject, outcome.Subject)
		}
		if _, ok := env.Metadata["commit_sha"]; ok {
			t.Errorf("metadata carries commit_sha on a dry-run: %+v", env.Metadata)
		}
		if env.Metadata["swept_count"] != float64(3) {
			t.Errorf("metadata.swept_count = %v, want 3", env.Metadata["swept_count"])
		}
	})

	t.Run("Subject falls back to the last Plan's Subject", func(t *testing.T) {
		outcome := &Outcome{
			DryRun: true,
			Plans:  []*verb.Plan{{Subject: "first"}, {Subject: "last"}},
		}
		out, _ := captureStdStreams(t, func() {
			FinishVerbOutcome(context.Background(), root, "aiwf test", outcome, nil, OutputFormat{Format: "json"})
		})
		var env struct {
			Result struct{ Subject string } `json:"result"`
		}
		if err := json.Unmarshal([]byte(out), &env); err != nil {
			t.Fatalf("stdout not JSON: %v\n%s", err, out)
		}
		if env.Result.Subject != "last" {
			t.Errorf("result.subject = %q, want %q (last Plan's Subject)", env.Result.Subject, "last")
		}
	})
}

// TestFinishVerbOutcome_DryRun_Text pins the two text-mode dry-run
// shapes: a caller-supplied TextDetail callback owns the entire
// preview (archive/rewidth/import's verb-specific move/write
// listings), and a bare subject line is the fallback when no callback
// is given.
//
// SERIAL (do not add t.Parallel): uses captureStdStreams.
func TestFinishVerbOutcome_DryRun_Text(t *testing.T) {
	root := t.TempDir()

	t.Run("TextDetail callback owns the preview", func(t *testing.T) {
		called := false
		outcome := &Outcome{
			DryRun:  true,
			Plans:   []*verb.Plan{{Subject: "sweep 3 entities"}},
			Subject: "sweep 3 entities (dry-run; re-run with --apply to commit)",
			TextDetail: func() {
				called = true
				Println("sweep 3 entities (dry-run; re-run with --apply to commit)")
				Println("Moves (3):")
			},
		}
		out, errOut := captureStdStreams(t, func() {
			FinishVerbOutcome(context.Background(), root, "aiwf archive", outcome, nil, OutputFormat{Format: "text"})
		})
		if !called {
			t.Fatal("TextDetail was never invoked")
		}
		if errOut != "" {
			t.Errorf("stderr must be empty; got %q", errOut)
		}
		want := "sweep 3 entities (dry-run; re-run with --apply to commit)\nMoves (3):\n"
		if out != want {
			t.Errorf("stdout = %q, want %q", out, want)
		}
	})

	t.Run("no TextDetail falls back to printing the resolved subject", func(t *testing.T) {
		outcome := &Outcome{
			DryRun:  true,
			Plans:   []*verb.Plan{{Subject: "unused"}},
			Subject: "sweep 3 entities (dry-run; re-run with --apply to commit)",
		}
		out, _ := captureStdStreams(t, func() {
			FinishVerbOutcome(context.Background(), root, "aiwf archive", outcome, nil, OutputFormat{Format: "text"})
		})
		want := "sweep 3 entities (dry-run; re-run with --apply to commit)\n"
		if out != want {
			t.Errorf("stdout = %q, want %q", out, want)
		}
	})
}

// TestFinishVerbOutcome_MultiPlan_Apply pins the multi-Plan apply
// path: each Plan applies in order, text mode prints one line per
// Plan's own Subject (reproducing import's existing per-plan text
// loop), and JSON mode carries the resolved aggregate Subject plus the
// last Plan's commit sha.
//
// SERIAL (do not add t.Parallel): uses captureStdStreams.
func TestFinishVerbOutcome_MultiPlan_Apply(t *testing.T) {
	t.Run("text mode prints one line per Plan", func(t *testing.T) {
		root := seedRepo(t)
		outcome := &Outcome{
			Plans: []*verb.Plan{
				{Subject: "create G-0001", Trailers: []gitops.Trailer{{Key: "aiwf-verb", Value: "test"}}, Ops: []verb.FileOp{{Type: verb.OpWrite, Path: "a.md", Content: []byte("a\n")}}},
				{Subject: "create G-0002", Trailers: []gitops.Trailer{{Key: "aiwf-verb", Value: "test"}}, Ops: []verb.FileOp{{Type: verb.OpWrite, Path: "b.md", Content: []byte("b\n")}}},
			},
			Subject: "aiwf import: 2 entities created",
		}
		out, errOut := captureStdStreams(t, func() {
			code, sha := FinishVerbOutcome(context.Background(), root, "aiwf import", outcome, nil, OutputFormat{Format: "text"})
			if code != ExitOK {
				t.Errorf("code = %d, want ExitOK", code)
			}
			if sha == "" {
				t.Error("sha is empty, want the last Plan's commit sha")
			}
		})
		if errOut != "" {
			t.Errorf("stderr must be empty; got %q", errOut)
		}
		want := "create G-0001\ncreate G-0002\n"
		if out != want {
			t.Errorf("stdout = %q, want %q (per-plan subject lines, not the aggregate Subject)", out, want)
		}
	})

	t.Run("json mode carries the aggregate Subject and the last plan's sha", func(t *testing.T) {
		root := seedRepo(t)
		outcome := &Outcome{
			Plans: []*verb.Plan{
				{Subject: "create G-0001", Trailers: []gitops.Trailer{{Key: "aiwf-verb", Value: "test"}}, Ops: []verb.FileOp{{Type: verb.OpWrite, Path: "a.md", Content: []byte("a\n")}}},
				{Subject: "create G-0002", Trailers: []gitops.Trailer{{Key: "aiwf-verb", Value: "test"}}, Ops: []verb.FileOp{{Type: verb.OpWrite, Path: "b.md", Content: []byte("b\n")}}},
			},
			Subject:  "aiwf import: 2 entities created",
			Metadata: map[string]any{"imported_count": 2},
		}
		var gotSHA string
		out, _ := captureStdStreams(t, func() {
			var code int
			code, gotSHA = FinishVerbOutcome(context.Background(), root, "aiwf import", outcome, nil, OutputFormat{Format: "json"})
			if code != ExitOK {
				t.Errorf("code = %d, want ExitOK", code)
			}
		})
		var env struct {
			Result   struct{ Subject string } `json:"result"`
			Metadata map[string]any           `json:"metadata"`
		}
		if err := json.Unmarshal([]byte(out), &env); err != nil {
			t.Fatalf("stdout not JSON: %v\n%s", err, out)
		}
		if env.Result.Subject != outcome.Subject {
			t.Errorf("result.subject = %q, want %q", env.Result.Subject, outcome.Subject)
		}
		if env.Metadata["commit_sha"] != gotSHA {
			t.Errorf("metadata.commit_sha = %v, want %q (the last plan's sha)", env.Metadata["commit_sha"], gotSHA)
		}
		if env.Metadata["imported_count"] != float64(2) {
			t.Errorf("metadata.imported_count = %v, want 2", env.Metadata["imported_count"])
		}
	})
}

// TestFinishVerbOutcome_ApplyError_MessageFormat pins the two
// apply-error message shapes that archive/rewidth (single-Plan) and
// import (multi-Plan) each shipped before the migration: a lone Plan's
// failure reports the bare error, while a batch of more than one Plan
// prefixes the failing index — exactly the two literal formats
// failArchive/failRewidth and failImport already produced.
//
// SERIAL (do not add t.Parallel): uses captureStdStreams.
func TestFinishVerbOutcome_ApplyError_MessageFormat(t *testing.T) {
	// A Plan with zero Ops and AllowEmpty unset is verb.Apply's own
	// "nothing to commit" refusal (see internal/verb/apply_test.go's
	// TestApply_RefusesNothingToCommit) — deterministic, no lock
	// contention or filesystem tricks needed.
	emptyPlan := func(subject string) *verb.Plan {
		return &verb.Plan{Subject: subject, Trailers: []gitops.Trailer{{Key: "aiwf-verb", Value: "test"}}}
	}

	t.Run("single Plan: bare error message", func(t *testing.T) {
		root := seedRepo(t)
		outcome := &Outcome{Plans: []*verb.Plan{emptyPlan("empty plan")}}
		_, errOut := captureStdStreams(t, func() {
			code, sha := FinishVerbOutcome(context.Background(), root, "aiwf test", outcome, nil, OutputFormat{Format: "text"})
			if code != ExitInternal || sha != "" {
				t.Errorf("code=%d sha=%q, want ExitInternal/empty", code, sha)
			}
		})
		const wantPrefix = "aiwf test: "
		if len(errOut) < len(wantPrefix) || errOut[:len(wantPrefix)] != wantPrefix {
			t.Fatalf("stderr = %q, want prefix %q", errOut, wantPrefix)
		}
		if got := errOut[len(wantPrefix):]; got != "" && got[:len("applying plan")] == "applying plan" {
			t.Errorf("stderr = %q, single-Plan apply errors must not carry the \"applying plan N:\" prefix", errOut)
		}
	})

	t.Run("multi-Plan batch: indexed error message", func(t *testing.T) {
		root := seedRepo(t)
		outcome := &Outcome{
			Plans: []*verb.Plan{
				{Subject: "ok", Trailers: []gitops.Trailer{{Key: "aiwf-verb", Value: "test"}}, Ops: []verb.FileOp{{Type: verb.OpWrite, Path: "a.md", Content: []byte("a\n")}}},
				emptyPlan("fails"),
			},
		}
		_, errOut := captureStdStreams(t, func() {
			code, sha := FinishVerbOutcome(context.Background(), root, "aiwf import", outcome, nil, OutputFormat{Format: "text"})
			if code != ExitInternal || sha != "" {
				t.Errorf("code=%d sha=%q, want ExitInternal/empty", code, sha)
			}
		})
		const wantPrefix = "aiwf import: applying plan 1: "
		if len(errOut) < len(wantPrefix) || errOut[:len(wantPrefix)] != wantPrefix {
			t.Fatalf("stderr = %q, want prefix %q", errOut, wantPrefix)
		}
	})
}

// TestFinishVerbOutcome_ApplySuccess_FindingsRenderInTextMode pins the
// warning-findings-alongside-a-successful-apply text-mode branch (e.g.
// reallocate's body-prose-mention warnings): findings render to
// stderr before the per-plan subject lines print to stdout.
//
// SERIAL (do not add t.Parallel): uses captureStdStreams.
func TestFinishVerbOutcome_ApplySuccess_FindingsRenderInTextMode(t *testing.T) {
	root := seedRepo(t)
	outcome := &Outcome{
		Findings: []check.Finding{
			{Code: check.CodeStatusValid, Severity: check.SeverityWarning, Message: "heads up", EntityID: "E-0001"},
		},
		Plans: []*verb.Plan{
			{Subject: "add a file", Trailers: []gitops.Trailer{{Key: "aiwf-verb", Value: "test"}}, Ops: []verb.FileOp{{Type: verb.OpWrite, Path: "new.md", Content: []byte("hi\n")}}},
		},
	}
	out, errOut := captureStdStreams(t, func() {
		code, sha := FinishVerbOutcome(context.Background(), root, "aiwf test", outcome, nil, OutputFormat{Format: "text"})
		if code != ExitOK {
			t.Errorf("code = %d, want ExitOK", code)
		}
		if sha == "" {
			t.Error("sha is empty, want the resulting commit sha")
		}
	})
	if errOut == "" {
		t.Error("stderr is empty, want the warning finding rendered")
	}
	if out != "add a file\n" {
		t.Errorf("stdout = %q, want %q", out, "add a file\n")
	}
}
