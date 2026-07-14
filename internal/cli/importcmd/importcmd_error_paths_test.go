package importcmd_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/23min/aiwf/internal/cli/cliutil"
	"github.com/23min/aiwf/internal/cli/cliutil/testutil"
	"github.com/23min/aiwf/internal/cli/importcmd"
)

// M-0256/AC-1 backfill: Run's ResolveRoot guard is `//coverage:ignore`d
// in importcmd.go itself. Every other flagged branch below is
// genuinely triggerable.

const singleEntityManifest = `version: 1
entities:
  - kind: epic
    id: E-0001
    frontmatter: {title: "Cake", status: active}
`

// emptyEntitiesManifest declares per-entity commit mode: in the
// default batched mode, buildImportPlans always returns exactly one
// Plan (even with zero entities), so len(res.Plans) == 0 is only
// reachable in per-entity mode, where each manifest entry maps 1:1 to
// a Plan.
const emptyEntitiesManifest = `version: 1
commit:
  mode: per-entity
entities: []
`

const duplicateIDManifest = `version: 1
entities:
  - kind: epic
    id: E-0001
    frontmatter: {title: "Cake", status: active}
  - kind: epic
    id: E-0001
    frontmatter: {title: "Cake2", status: active}
`

func writeManifest(t *testing.T, dir, body string) string {
	t.Helper()
	path := filepath.Join(dir, "manifest.yaml")
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	return path
}

func writeAiwfYAML(t *testing.T, root, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(root, "aiwf.yaml"), []byte(content), 0o644); err != nil {
		t.Fatalf("write aiwf.yaml: %v", err)
	}
}

func mustGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}

// TestRun_ResolveActorFailure covers Run's fallback cliutil.ResolveActor
// guard (both --actor and the manifest's own actor: field empty), using
// M-0252's BrokenGitIdentity fixture.
//
// Serial: BrokenGitIdentity uses t.Setenv, which panics under t.Parallel.
func TestRun_ResolveActorFailure(t *testing.T) {
	testutil.BrokenGitIdentity(t)
	root := t.TempDir()
	manifest := writeManifest(t, root, singleEntityManifest)
	rc := importcmd.Run(manifest, root, "", "", "", true, cliutil.OutputFormat{})
	if rc != cliutil.ExitUsage {
		t.Errorf("rc = %d, want ExitUsage", rc)
	}
}

// TestRun_LoadTreeWithTrunkFailure covers Run's bare
// cliutil.LoadTreeWithTrunk guard: a syntactically broken aiwf.yaml
// makes config.Load (called inside LoadTreeWithTrunk) fail with a
// non-ErrNotFound error. actor is supplied explicitly so ResolveActor
// is never reached; --dry-run skips the repo lock.
func TestRun_LoadTreeWithTrunkFailure(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	writeAiwfYAML(t, root, "tdd: [unterminated\n")
	manifest := writeManifest(t, root, singleEntityManifest)
	rc := importcmd.Run(manifest, root, "human/test", "", "", true, cliutil.OutputFormat{})
	if rc != cliutil.ExitInternal {
		t.Errorf("rc = %d, want ExitInternal", rc)
	}
}

// TestRun_PrincipalRequiredForNonHumanActor covers the provenance-
// coherence guard: a non-human --actor requires --principal.
func TestRun_PrincipalRequiredForNonHumanActor(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	manifest := writeManifest(t, root, singleEntityManifest)
	rc := importcmd.Run(manifest, root, "ai/claude", "", "", true, cliutil.OutputFormat{})
	if rc != cliutil.ExitUsage {
		t.Errorf("rc = %d, want ExitUsage", rc)
	}
}

// TestRun_PrincipalForbiddenForHumanActor covers the inverse guard: a
// human --actor forbids --principal (humans act directly).
func TestRun_PrincipalForbiddenForHumanActor(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	manifest := writeManifest(t, root, singleEntityManifest)
	rc := importcmd.Run(manifest, root, "human/test", "human/other", "", true, cliutil.OutputFormat{})
	if rc != cliutil.ExitUsage {
		t.Errorf("rc = %d, want ExitUsage", rc)
	}
}

// TestRun_OnCollisionInvalidValue covers verb.Import's own
// --on-collision validation guard. Cobra's FixedCompletions only hints
// the shell completion set — it does not restrict the flag's actual
// value — so an unrecognized value reaches verb.Import and is refused
// there.
func TestRun_OnCollisionInvalidValue(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	manifest := writeManifest(t, root, singleEntityManifest)
	rc := importcmd.Run(manifest, root, "human/test", "", "bogus", true, cliutil.OutputFormat{})
	if rc != cliutil.ExitUsage {
		t.Errorf("rc = %d, want ExitUsage", rc)
	}
}

// TestRun_FindingsTextAndJSON covers the check.HasErrors(res.Findings)
// branch in both output formats: a manifest declaring the same id
// twice trips verb.Import's own import-duplicate-id finding, with no
// pre-existing tree state needed.
func TestRun_FindingsTextAndJSON(t *testing.T) {
	t.Parallel()
	for _, jsonOut := range []bool{false, true} {
		root := t.TempDir()
		manifest := writeManifest(t, root, duplicateIDManifest)
		out := cliutil.OutputFormat{}
		if jsonOut {
			out.Format = "json"
		}
		rc := importcmd.Run(manifest, root, "human/test", "", "", true, out)
		if rc != cliutil.ExitFindings {
			t.Errorf("jsonOut=%v: rc = %d, want ExitFindings", jsonOut, rc)
		}
	}
}

// TestRun_EmptyManifestTextAndJSON covers the "manifest had no
// entities to import" branch in both output formats.
func TestRun_EmptyManifestTextAndJSON(t *testing.T) {
	t.Parallel()
	for _, jsonOut := range []bool{false, true} {
		root := t.TempDir()
		manifest := writeManifest(t, root, emptyEntitiesManifest)
		out := cliutil.OutputFormat{}
		if jsonOut {
			out.Format = "json"
		}
		rc := importcmd.Run(manifest, root, "human/test", "", "", true, out)
		if rc != cliutil.ExitOK {
			t.Errorf("jsonOut=%v: rc = %d, want ExitOK", jsonOut, rc)
		}
	}
}

// TestRun_DryRunJSON covers the --dry-run JSON-envelope branch (the
// text --dry-run variant is already covered by
// internal/cli/integration/import_cmd_test.go's TestRun_ImportDryRun).
func TestRun_DryRunJSON(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	manifest := writeManifest(t, root, singleEntityManifest)
	rc := importcmd.Run(manifest, root, "human/test", "", "", true, cliutil.OutputFormat{Format: "json"})
	if rc != cliutil.ExitOK {
		t.Errorf("rc = %d, want ExitOK", rc)
	}
}

// TestRun_ApplyGitOperationInProgress covers the verb.Apply guard:
// Apply refuses to write while a merge/cherry-pick/revert/rebase is
// mid-flight in the target repo, rather than layering a new commit on
// top of an unresolved operation. Requires a real git repo since
// gitops.GitDir shells out; the marker file is the same one a real
// `git merge` in progress leaves behind.
func TestRun_ApplyGitOperationInProgress(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	mustGit(t, root, "init", "-q")
	mustGit(t, root, "config", "user.email", "test@example.com")
	mustGit(t, root, "config", "user.name", "Tester")
	if err := os.WriteFile(filepath.Join(root, ".git", "MERGE_HEAD"), []byte("deadbeef\n"), 0o644); err != nil {
		t.Fatalf("write MERGE_HEAD: %v", err)
	}

	manifest := writeManifest(t, root, singleEntityManifest)
	rc := importcmd.Run(manifest, root, "human/test", "", "", false, cliutil.OutputFormat{})
	if rc != cliutil.ExitInternal {
		t.Errorf("rc = %d, want ExitInternal", rc)
	}
}
