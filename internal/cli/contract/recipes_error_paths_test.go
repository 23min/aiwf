package contract

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/23min/aiwf/internal/cli/cliutil"
	"github.com/23min/aiwf/internal/cli/cliutil/testutil"
)

// M-0254/AC-1 backfill: this package's recipes.go carries the single
// largest concentration of flagged sites of any file in the epic's
// scope. The ResolveRoot guards (runRecipes, runRecipeInstall,
// runRecipeRemove) and recipe.List's error guard are
// `//coverage:ignore`d in recipes.go itself; so is runRecipeShow's
// os.Stdout write guard. Everything else below gets a real test.
// White-box (package contract) so this file can call the unexported
// runRecipes / runRecipeShow / runRecipeInstall / runRecipeRemove
// directly.

const contractsMalformedYAML = "contracts:\n  bindings:\n    - not a valid binding\n"

// writeAiwfYAML writes root/aiwf.yaml with the given content.
func writeAiwfYAML(t *testing.T, root, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(root, "aiwf.yaml"), []byte(content), 0o644); err != nil {
		t.Fatalf("write aiwf.yaml: %v", err)
	}
}

// TestRunRecipes_NoDeclaredValidators drives runRecipes end to end
// against a root with no contracts: block at all, covering the
// "(none)" arm of the declared-validators listing.
func TestRunRecipes_NoDeclaredValidators(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	writeAiwfYAML(t, root, "")
	if rc := runRecipes(root); rc != cliutil.ExitOK {
		t.Errorf("rc = %d, want ExitOK", rc)
	}
}

// TestRunRecipes_WithDeclaredValidator drives runRecipes against a
// root with one declared validator, covering the populated arm of the
// declared-validators listing (the sorted-names print loop).
func TestRunRecipes_WithDeclaredValidator(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	writeAiwfYAML(t, root, "contracts:\n  validators:\n    cue:\n      command: cue\n")
	if rc := runRecipes(root); rc != cliutil.ExitOK {
		t.Errorf("rc = %d, want ExitOK", rc)
	}
}

// TestRunRecipes_LoadContractsBlockFailure covers runRecipes' own
// cliutil.LoadContractsBlock guard, reusing the malformed-contracts-
// block trigger already proven at
// internal/cli/add/add_error_paths_test.go. Distinct from the two
// happy-path tests above, which only exercise the success arm of this
// same call.
func TestRunRecipes_LoadContractsBlockFailure(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	writeAiwfYAML(t, root, contractsMalformedYAML)
	if rc := runRecipes(root); rc != cliutil.ExitInternal {
		t.Errorf("rc = %d, want ExitInternal", rc)
	}
}

// TestRunRecipeShow_UnknownName covers runRecipeShow's recipe.Get
// guard: a name absent from the embedded set.
func TestRunRecipeShow_UnknownName(t *testing.T) {
	t.Parallel()
	if rc := runRecipeShow("not-a-real-recipe"); rc != cliutil.ExitUsage {
		t.Errorf("rc = %d, want ExitUsage", rc)
	}
}

// TestRunRecipeInstall_MutexBothFromAndArg covers the --from/positional
// mutex: both supplied at once is a usage error, checked before any
// root/tree work.
func TestRunRecipeInstall_MutexBothFromAndArg(t *testing.T) {
	t.Parallel()
	rc := runRecipeInstall([]string{"render"}, "", "", "custom.yaml", false, cliutil.OutputFormat{})
	if rc != cliutil.ExitUsage {
		t.Errorf("rc = %d, want ExitUsage", rc)
	}
}

// TestRunRecipeInstall_NeitherFromNorArg covers the default usage-error
// case: neither a positional name nor --from.
func TestRunRecipeInstall_NeitherFromNorArg(t *testing.T) {
	t.Parallel()
	rc := runRecipeInstall(nil, "", "", "", false, cliutil.OutputFormat{})
	if rc != cliutil.ExitUsage {
		t.Errorf("rc = %d, want ExitUsage", rc)
	}
}

// TestRunRecipeInstall_UnknownRecipeName covers the loadErr guard via
// recipe.Get failing on a name absent from the embedded set.
func TestRunRecipeInstall_UnknownRecipeName(t *testing.T) {
	t.Parallel()
	rc := runRecipeInstall([]string{"not-a-real-recipe"}, "", "", "", false, cliutil.OutputFormat{})
	if rc != cliutil.ExitUsage {
		t.Errorf("rc = %d, want ExitUsage", rc)
	}
}

// TestRunRecipeInstall_ResolveActorFailure covers runRecipeInstall's
// cliutil.ResolveActor guard using M-0252's BrokenGitIdentity fixture.
// Serial: BrokenGitIdentity uses t.Setenv, which panics under
// t.Parallel.
func TestRunRecipeInstall_ResolveActorFailure(t *testing.T) {
	testutil.BrokenGitIdentity(t)
	root := t.TempDir()
	rc := runRecipeInstall([]string{"cue"}, root, "", "", false, cliutil.OutputFormat{})
	if rc != cliutil.ExitUsage {
		t.Errorf("rc = %d, want ExitUsage", rc)
	}
}

// TestRunRecipeInstall_LoadContractsDocFailure covers runRecipeInstall's
// cliutil.LoadContractsDoc guard, reusing the malformed-contracts-block
// trigger already proven at
// internal/cli/add/add_error_paths_test.go. Uses "cue" (a real embedded
// recipe) so recipe.Get succeeds and the run actually reaches
// LoadContractsDoc, rather than failing earlier at the loadErr guard.
func TestRunRecipeInstall_LoadContractsDocFailure(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	writeAiwfYAML(t, root, contractsMalformedYAML)
	rc := runRecipeInstall([]string{"cue"}, root, "human/test", "", false, cliutil.OutputFormat{})
	if rc != cliutil.ExitUsage {
		t.Errorf("rc = %d, want ExitUsage", rc)
	}
}

// TestRunRecipeRemove_ResolveActorFailure covers runRecipeRemove's
// cliutil.ResolveActor guard using M-0252's BrokenGitIdentity fixture.
// Serial: BrokenGitIdentity uses t.Setenv, which panics under
// t.Parallel.
func TestRunRecipeRemove_ResolveActorFailure(t *testing.T) {
	testutil.BrokenGitIdentity(t)
	root := t.TempDir()
	rc := runRecipeRemove("cue", root, "", cliutil.OutputFormat{})
	if rc != cliutil.ExitUsage {
		t.Errorf("rc = %d, want ExitUsage", rc)
	}
}

// TestRunRecipeRemove_LoadContractsDocFailure covers runRecipeRemove's
// cliutil.LoadContractsDoc guard, reusing the same malformed-contracts-
// block trigger.
func TestRunRecipeRemove_LoadContractsDocFailure(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	writeAiwfYAML(t, root, contractsMalformedYAML)
	rc := runRecipeRemove("cue", root, "human/test", cliutil.OutputFormat{})
	if rc != cliutil.ExitUsage {
		t.Errorf("rc = %d, want ExitUsage", rc)
	}
}
