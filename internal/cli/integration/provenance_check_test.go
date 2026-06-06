package integration

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/check"
	"github.com/23min/aiwf/internal/cli/cliutil/testutil"
)

// TestProvenanceCheck_CleanRepoSilent: after `aiwf init` + a couple
// of normal verb invocations, `aiwf check` produces no provenance
// findings. The trailer set is well-formed at every step, so all
// eleven standing rules stay quiet, and the upstream-aware audit
// pass scans an empty unpushed range without firing the
// scope-undefined advisory.
func TestProvenanceCheck_CleanRepoSilent(t *testing.T) {
	t.Parallel()
	bin := testutil.AiwfBinary(t)
	binDir := filepath.Dir(bin)
	root := testutil.SetupGitRepoWithUpstream(t, "peter@example.com")
	if out, err := testutil.RunBin(t, root, binDir, nil, "init"); err != nil {
		t.Fatalf("aiwf init: %v\n%s", err, out)
	}
	if out, err := testutil.RunBin(t, root, binDir, nil, "add", "epic", "--title", "Engine refactor"); err != nil {
		t.Fatalf("aiwf add epic: %v\n%s", err, out)
	}
	out, err := testutil.RunBin(t, root, binDir, nil, "check")
	if err != nil {
		t.Fatalf("aiwf check: %v\n%s", err, out)
	}
	if strings.Contains(out, "provenance-") {
		t.Errorf("clean repo produced provenance findings:\n%s", out)
	}
}

// TestProvenanceCheck_HandEditedAgentCommit: simulates an external
// agent commit (no scope, ai/... actor, no on-behalf-of) hand-crafted
// onto the repo. `aiwf check` fires
// provenance-no-active-scope.
func TestProvenanceCheck_HandEditedAgentCommit(t *testing.T) {
	t.Parallel()
	bin := testutil.AiwfBinary(t)
	binDir := filepath.Dir(bin)
	root := testutil.SetupGitRepoWithUpstream(t, "peter@example.com")
	if out, err := testutil.RunBin(t, root, binDir, nil, "init"); err != nil {
		t.Fatalf("aiwf init: %v\n%s", err, out)
	}
	if out, err := testutil.RunBin(t, root, binDir, nil, "add", "epic", "--title", "Engine"); err != nil {
		t.Fatalf("aiwf add: %v\n%s", err, out)
	}
	// Hand-craft a malformed commit: ai actor, no principal, no
	// on-behalf-of. The trailer-coherence rule (non-human actor
	// without principal) plus the no-active-scope rule both fire.
	msg := "chore: hand-crafted ai commit\n\n" +
		"aiwf-verb: promote\n" +
		"aiwf-entity: E-001\n" +
		"aiwf-actor: ai/claude\n"
	if out, err := testutil.RunGit(root, "commit", "--allow-empty", "-m", msg); err != nil {
		t.Fatalf("git commit: %v\n%s", err, out)
	}
	out, _ := testutil.RunBin(t, root, binDir, nil, "check")
	// `aiwf check` exits 1 on findings, so we ignore the Go error and
	// inspect stdout.
	if !strings.Contains(out, check.CodeProvenanceNoActiveScope) {
		t.Errorf("expected provenance-no-active-scope finding; got:\n%s", out)
	}
	if !strings.Contains(out, "provenance-trailer-incoherent") {
		t.Errorf("expected provenance-trailer-incoherent (missing principal); got:\n%s", out)
	}
}

// TestProvenanceCheck_UntrailedEntityCommit covers step 7b: a manual
// `git commit` lands on an entity file without an aiwf-verb: trailer.
// `aiwf check` fires the warning and points at `--audit-only`.
func TestProvenanceCheck_UntrailedEntityCommit(t *testing.T) {
	t.Parallel()
	bin := testutil.AiwfBinary(t)
	binDir := filepath.Dir(bin)
	root := testutil.SetupGitRepoWithUpstream(t, "peter@example.com")
	if out, err := testutil.RunBin(t, root, binDir, nil, "init"); err != nil {
		t.Fatalf("aiwf init: %v\n%s", err, out)
	}
	if out, err := testutil.RunBin(t, root, binDir, nil, "add", "gap", "--title", "Validators leak temp files"); err != nil {
		t.Fatalf("aiwf add gap: %v\n%s", err, out)
	}

	// Manually edit the gap file and commit without aiwf trailers —
	// the audit-trail hole G24 cares about.
	gapRel := mustFindFile(t, root, "G-0001-")
	manualFlipStatus(t, filepath.Join(root, gapRel), "open", "wontfix")
	if out, err := testutil.RunGit(root, "add", gapRel); err != nil {
		t.Fatalf("git add: %v\n%s", err, out)
	}
	if out, err := testutil.RunGit(root, "commit", "-m", "manually mark G-001 wontfix"); err != nil {
		t.Fatalf("manual commit: %v\n%s", err, out)
	}

	out, _ := testutil.RunBin(t, root, binDir, nil, "check")
	if !strings.Contains(out, "provenance-untrailered-entity-commit") {
		t.Fatalf("expected provenance-untrailered-entity-commit; got:\n%s", out)
	}
	// Severity is error since G-0231 item 3 (was warning before the
	// merge-commit carveout + audit-only backfill of the 5 direct-edit
	// findings cleared the historical noise). Errors render
	// per-instance with the severity as a prefix word:
	// `error <code>: <detail>` — distinct from the collapsed warning
	// shape `<code> (warning) × N — <detail>`. Assert the prefix form
	// so the check is structural, not a substring that could match
	// either shape.
	if !strings.Contains(out, "error provenance-untrailered-entity-commit:") {
		t.Errorf("expected error severity; got:\n%s", out)
	}
}

// TestProvenanceCheck_AuthorizationMissing: a hand-crafted commit
// references an authorize SHA that doesn't exist.
func TestProvenanceCheck_AuthorizationMissing(t *testing.T) {
	t.Parallel()
	bin := testutil.AiwfBinary(t)
	binDir := filepath.Dir(bin)
	root := testutil.SetupGitRepoWithUpstream(t, "peter@example.com")
	if out, err := testutil.RunBin(t, root, binDir, nil, "init"); err != nil {
		t.Fatalf("aiwf init: %v\n%s", err, out)
	}
	if out, err := testutil.RunBin(t, root, binDir, nil, "add", "epic", "--title", "Engine"); err != nil {
		t.Fatalf("aiwf add: %v\n%s", err, out)
	}
	msg := "chore: hand-crafted scoped commit\n\n" +
		"aiwf-verb: promote\n" +
		"aiwf-entity: E-001\n" +
		"aiwf-actor: ai/claude\n" +
		"aiwf-principal: human/peter\n" +
		"aiwf-on-behalf-of: human/peter\n" +
		"aiwf-authorized-by: 0000000000000000000000000000000000000000\n"
	if out, err := testutil.RunGit(root, "commit", "--allow-empty", "-m", msg); err != nil {
		t.Fatalf("git commit: %v\n%s", err, out)
	}
	out, _ := testutil.RunBin(t, root, binDir, nil, "check")
	if !strings.Contains(out, "provenance-authorization-missing") {
		t.Errorf("expected provenance-authorization-missing; got:\n%s", out)
	}
}

// TestProvenanceCheck_WrapBundleAfterPromoteIsTolerated covers G-0120
// at the seam: drives `aiwf check` end-to-end against a repo whose
// commit graph matches the pre-G-0119 wrap-ritual order — an agent
// authorize-opened scope, then a same-entity terminal-promote (which
// ends the scope), then a `wrap-epic` artefact commit landing AFTER
// the scope ended. Pre-fix `aiwf check` printed an
// `provenance-authorization-ended` error against the wrap commit;
// post-fix the rule recognises the wrap-bundle pattern and stays
// silent.
//
// The commits are hand-crafted (rather than driven through `aiwf
// authorize` / `aiwf promote`) so the failing case is reproducible
// without depending on the rituals plugin being installed in the
// test environment.
func TestProvenanceCheck_WrapBundleAfterPromoteIsTolerated(t *testing.T) {
	t.Parallel()
	bin := testutil.AiwfBinary(t)
	binDir := filepath.Dir(bin)
	root := testutil.SetupGitRepoWithUpstream(t, "peter@example.com")
	if out, err := testutil.RunBin(t, root, binDir, nil, "init"); err != nil {
		t.Fatalf("aiwf init: %v\n%s", err, out)
	}
	if out, err := testutil.RunBin(t, root, binDir, nil, "add", "epic", "--title", "Engine"); err != nil {
		t.Fatalf("aiwf add: %v\n%s", err, out)
	}
	// Authorize-opened commit — hand-crafted so we can pin its SHA via
	// rev-parse and reference it from the subsequent commits.
	authMsg := "aiwf authorize E-0001 --to ai/claude\n\n" +
		"aiwf-verb: authorize\n" +
		"aiwf-entity: E-0001\n" +
		"aiwf-actor: human/peter\n" +
		"aiwf-to: ai/claude\n" +
		"aiwf-scope: opened\n"
	if out, err := testutil.RunGit(root, "commit", "--allow-empty", "-m", authMsg); err != nil {
		t.Fatalf("authorize commit: %v\n%s", err, out)
	}
	authSHA, err := testutil.RunGit(root, "rev-parse", "HEAD")
	if err != nil {
		t.Fatalf("rev-parse authorize: %v\n%s", err, authSHA)
	}
	authSHA = strings.TrimSpace(authSHA)
	// Terminal promote that ends the scope — same entity, with
	// scope-ends trailer naming the auth SHA.
	promoteMsg := "aiwf promote E-0001 active -> done\n\n" +
		"aiwf-verb: promote\n" +
		"aiwf-entity: E-0001\n" +
		"aiwf-actor: ai/claude\n" +
		"aiwf-to: done\n" +
		"aiwf-principal: human/peter\n" +
		"aiwf-on-behalf-of: human/peter\n" +
		"aiwf-authorized-by: " + authSHA + "\n" +
		"aiwf-scope-ends: " + authSHA + "\n"
	if out, err := testutil.RunGit(root, "commit", "--allow-empty", "-m", promoteMsg); err != nil {
		t.Fatalf("promote commit: %v\n%s", err, out)
	}
	// Wrap-epic artefact commit, lands AFTER the promote that ended
	// the scope. Pre-G-0120 this fired authorization-ended; post-fix
	// the wrap-bundle exception suppresses it.
	wrapMsg := "chore(E-0001): wrap artefact\n\n" +
		"aiwf-verb: wrap-epic\n" +
		"aiwf-entity: E-0001\n" +
		"aiwf-actor: ai/claude\n" +
		"aiwf-principal: human/peter\n" +
		"aiwf-on-behalf-of: human/peter\n" +
		"aiwf-authorized-by: " + authSHA + "\n"
	if out, err := testutil.RunGit(root, "commit", "--allow-empty", "-m", wrapMsg); err != nil {
		t.Fatalf("wrap commit: %v\n%s", err, out)
	}
	out, _ := testutil.RunBin(t, root, binDir, nil, "check")
	if strings.Contains(out, "provenance-authorization-ended") {
		t.Errorf("wrap-bundle commit fired authorization-ended despite same-entity terminal promote:\n%s", out)
	}
}
