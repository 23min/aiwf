package verb

import (
	"context"
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/aiwfyaml"
	"github.com/23min/aiwf/internal/tree"
)

const recipeBaseYAML = `aiwf_version: 0.1.0
actor: human/test
contracts:
  validators: {}
  entries: []
`

func TestRecipeInstall_NewValidator(t *testing.T) {
	t.Parallel()
	d, c := mustReadDoc(t, recipeBaseYAML)
	res, err := RecipeInstall(context.Background(), &tree.Tree{}, d, c, "cue", aiwfyaml.Validator{
		Command: "cue", Args: []string{"vet", "{{schema}}", "{{fixture}}"},
	}, "human/test", t.TempDir(), RecipeInstallOptions{})
	if err != nil {
		t.Fatalf("RecipeInstall: %v", err)
	}
	if res.Plan == nil {
		t.Fatal("expected Plan")
	}
	got := string(res.Plan.Ops[0].Content)
	if !strings.Contains(got, "cue") {
		t.Errorf("aiwf.yaml missing the new validator:\n%s", got)
	}
	mustHaveTrailerInPlan(t, res.Plan, "aiwf-verb", "contract-recipe-install")
}

func TestRecipeInstall_IdempotentExactMatch(t *testing.T) {
	t.Parallel()
	src := strings.Replace(recipeBaseYAML, "  validators: {}", `  validators:
    cue:
      command: cue
      args:
        - vet
        - "{{schema}}"
        - "{{fixture}}"`, 1)
	d, c := mustReadDoc(t, src)
	res, err := RecipeInstall(context.Background(), &tree.Tree{}, d, c, "cue", aiwfyaml.Validator{
		Command: "cue", Args: []string{"vet", "{{schema}}", "{{fixture}}"},
	}, "human/test", t.TempDir(), RecipeInstallOptions{})
	if err != nil {
		t.Fatalf("RecipeInstall: %v", err)
	}
	if !res.NoOp {
		t.Errorf("expected NoOp for exact-match install; got %+v", res)
	}
}

func TestRecipeInstall_DifferentRequiresForce(t *testing.T) {
	t.Parallel()
	src := strings.Replace(recipeBaseYAML, "  validators: {}", `  validators:
    cue:
      command: cue
      args:
        - eval`, 1)
	d, c := mustReadDoc(t, src)
	_, err := RecipeInstall(context.Background(), &tree.Tree{}, d, c, "cue", aiwfyaml.Validator{
		Command: "cue", Args: []string{"vet"},
	}, "human/test", t.TempDir(), RecipeInstallOptions{})
	if err == nil || !strings.Contains(err.Error(), "force") {
		t.Errorf("expected force-required error; got %v", err)
	}
}

func TestRecipeInstall_TrailersIncludeReferencingBindings(t *testing.T) {
	t.Parallel()
	src := `aiwf_version: 0.1.0
actor: human/test
contracts:
  validators:
    cue:
      command: cue
      args:
        - vet
    jsonschema:
      command: ajv
      args:
        - validate
  entries:
    - id: C-001
      validator: cue
      schema: a.cue
      fixtures: fa
    - id: C-002
      validator: cue
      schema: b.cue
      fixtures: fb
    - id: C-003
      validator: jsonschema
      schema: c.json
      fixtures: fc
`
	d, c := mustReadDoc(t, src)
	res, err := RecipeInstall(context.Background(), &tree.Tree{}, d, c, "cue", aiwfyaml.Validator{
		Command: "cue", Args: []string{"vet", "--all"},
	}, "human/test", t.TempDir(), RecipeInstallOptions{Force: true})
	if err != nil {
		t.Fatalf("RecipeInstall --force: %v", err)
	}
	wantIDs := map[string]bool{"C-0001": true, "C-0002": true}
	gotIDs := map[string]bool{}
	for _, tr := range res.Plan.Trailers {
		if tr.Key == "aiwf-entity" {
			gotIDs[tr.Value] = true
		}
	}
	if len(gotIDs) != len(wantIDs) {
		t.Errorf("trailer entity ids: got %v, want %v", gotIDs, wantIDs)
	}
	for id := range wantIDs {
		if !gotIDs[id] {
			t.Errorf("trailer aiwf-entity:%s missing", id)
		}
	}
	if gotIDs["C-0003"] {
		t.Error("C-003 should not have a trailer (different validator)")
	}
}

func TestRecipeRemove_Success(t *testing.T) {
	t.Parallel()
	src := strings.Replace(recipeBaseYAML, "  validators: {}", `  validators:
    cue:
      command: cue
      args:
        - vet`, 1)
	d, c := mustReadDoc(t, src)
	res, err := RecipeRemove(context.Background(), &tree.Tree{}, d, c, "cue", "human/test", t.TempDir())
	if err != nil {
		t.Fatalf("RecipeRemove: %v", err)
	}
	got := string(res.Plan.Ops[0].Content)
	if strings.Contains(got, "cue:") {
		t.Errorf("validator not removed:\n%s", got)
	}
}

// TestRecipeInstall_ConsultsTheTreeViaTheSharedGate: RecipeInstall's
// mutation only touches contracts.validators, so contractMutationGate
// can never find an introduced finding here — no behavioral assertion
// on RecipeInstall's return value can distinguish "the gate call
// runs" from "it was silently deleted." Passing a nil tree makes the
// distinction observable instead: contractMutationGate always reaches
// contractcheck.Run, which calls t.ByKind on the tree — a nil
// *tree.Tree panics there. If RecipeInstall stops calling the gate, t
// is never dereferenced and nothing panics.
func TestRecipeInstall_ConsultsTheTreeViaTheSharedGate(t *testing.T) {
	t.Parallel()
	d, c := mustReadDoc(t, recipeBaseYAML)

	defer func() {
		if recover() == nil {
			t.Fatal("expected RecipeInstall to consult the tree via the shared gate (contractMutationGate) and panic on a nil tree; it didn't — the gate call may have been removed")
		}
	}()
	_, _ = RecipeInstall(context.Background(), nil, d, c, "cue", aiwfyaml.Validator{
		Command: "cue", Args: []string{"vet"},
	}, "human/test", t.TempDir(), RecipeInstallOptions{})
}

// TestRecipeRemove_ConsultsTheTreeViaTheSharedGate: same rationale as
// TestRecipeInstall_ConsultsTheTreeViaTheSharedGate above — removing a
// validator only touches contracts.validators, so the gate can never
// block, and only a nil-tree panic distinguishes "wired" from
// "removed."
func TestRecipeRemove_ConsultsTheTreeViaTheSharedGate(t *testing.T) {
	t.Parallel()
	src := strings.Replace(recipeBaseYAML, "  validators: {}", `  validators:
    cue:
      command: cue
      args:
        - vet`, 1)
	d, c := mustReadDoc(t, src)

	defer func() {
		if recover() == nil {
			t.Fatal("expected RecipeRemove to consult the tree via the shared gate (contractMutationGate) and panic on a nil tree; it didn't — the gate call may have been removed")
		}
	}()
	_, _ = RecipeRemove(context.Background(), nil, d, c, "cue", "human/test", t.TempDir())
}

func TestRecipeRemove_RejectsReferencedValidator(t *testing.T) {
	t.Parallel()
	src := `aiwf_version: 0.1.0
actor: human/test
contracts:
  validators:
    cue:
      command: cue
      args: [vet]
  entries:
    - id: C-001
      validator: cue
      schema: s.cue
      fixtures: f
`
	d, c := mustReadDoc(t, src)
	_, err := RecipeRemove(context.Background(), &tree.Tree{}, d, c, "cue", "human/test", t.TempDir())
	if err == nil || !strings.Contains(err.Error(), "C-0001") {
		t.Errorf("expected error naming C-001; got %v", err)
	}
}

func TestRecipeRemove_RejectsMissingValidator(t *testing.T) {
	t.Parallel()
	d, c := mustReadDoc(t, recipeBaseYAML)
	if _, err := RecipeRemove(context.Background(), &tree.Tree{}, d, c, "ghost", "human/test", t.TempDir()); err == nil {
		t.Error("expected error for missing validator")
	}
}

// --- Edge case coverage (added during the I1 hardening pass) ---

func TestRecipeInstall_RejectsEmptyName(t *testing.T) {
	t.Parallel()
	d, c := mustReadDoc(t, recipeBaseYAML)
	_, err := RecipeInstall(context.Background(), &tree.Tree{}, d, c, "", aiwfyaml.Validator{Command: "x"}, "human/test", t.TempDir(), RecipeInstallOptions{})
	if err == nil {
		t.Error("expected error for empty name")
	}
}

func TestRecipeInstall_RejectsEmptyCommand(t *testing.T) {
	t.Parallel()
	d, c := mustReadDoc(t, recipeBaseYAML)
	_, err := RecipeInstall(context.Background(), &tree.Tree{}, d, c, "x", aiwfyaml.Validator{Command: ""}, "human/test", t.TempDir(), RecipeInstallOptions{})
	if err == nil {
		t.Error("expected error for empty command")
	}
}

func TestRecipeInstall_NoTrailersForUnreferencedValidator(t *testing.T) {
	t.Parallel()
	// Brand-new validator with no bindings yet — install should NOT
	// emit any aiwf-entity trailers.
	d, c := mustReadDoc(t, recipeBaseYAML)
	res, err := RecipeInstall(context.Background(), &tree.Tree{}, d, c, "fresh", aiwfyaml.Validator{
		Command: "fresh", Args: []string{"--check"},
	}, "human/test", t.TempDir(), RecipeInstallOptions{})
	if err != nil {
		t.Fatalf("RecipeInstall: %v", err)
	}
	for _, tr := range res.Plan.Trailers {
		if tr.Key == "aiwf-entity" {
			t.Errorf("unexpected aiwf-entity trailer for unreferenced validator: %+v", tr)
		}
	}
}

func TestRecipeInstall_ForceUpdatesArgsAndKeepsValidator(t *testing.T) {
	t.Parallel()
	src := strings.Replace(recipeBaseYAML, "  validators: {}", `  validators:
    cue:
      command: cue
      args:
        - vet`, 1)
	d, c := mustReadDoc(t, src)
	res, err := RecipeInstall(context.Background(), &tree.Tree{}, d, c, "cue", aiwfyaml.Validator{
		Command: "cue",
		Args:    []string{"vet", "--all"},
	}, "human/test", t.TempDir(), RecipeInstallOptions{Force: true})
	if err != nil {
		t.Fatalf("RecipeInstall force: %v", err)
	}
	got := string(res.Plan.Ops[0].Content)
	if !strings.Contains(got, "--all") {
		t.Errorf("force-replace did not update args:\n%s", got)
	}
}

func TestRecipeRemove_NamesMultipleReferencesInError(t *testing.T) {
	t.Parallel()
	src := `aiwf_version: 0.1.0
actor: human/test
contracts:
  validators:
    cue:
      command: cue
      args: [vet]
  entries:
    - id: C-001
      validator: cue
      schema: a.cue
      fixtures: fa
    - id: C-007
      validator: cue
      schema: b.cue
      fixtures: fb
`
	d, c := mustReadDoc(t, src)
	_, err := RecipeRemove(context.Background(), &tree.Tree{}, d, c, "cue", "human/test", t.TempDir())
	if err == nil {
		t.Fatal("expected error for referenced validator")
	}
	for _, want := range []string{"C-0001", "C-0007"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error %q missing reference %s", err, want)
		}
	}
}

func TestValidatorEqual_HandlesNilArgs(t *testing.T) {
	t.Parallel()
	a := aiwfyaml.Validator{Command: "x", Args: nil}
	b := aiwfyaml.Validator{Command: "x", Args: []string{}}
	if !validatorEqual(a, b) {
		t.Error("nil args and empty args slice should compare equal")
	}
}

func TestBindingsReferencing_IsNilSafeAndSorted(t *testing.T) {
	t.Parallel()
	if got := bindingsReferencing(nil, "x"); got != nil {
		t.Errorf("nil contracts should yield nil; got %+v", got)
	}
	c := &aiwfyaml.Contracts{
		Validators: map[string]aiwfyaml.Validator{"cue": {Command: "cue"}},
		Entries: []aiwfyaml.Entry{
			{ID: "C-0002", Validator: "cue", Schema: "s", Fixtures: "f"},
			{ID: "C-0001", Validator: "cue", Schema: "s", Fixtures: "f"},
		},
	}
	got := bindingsReferencing(c, "cue")
	if len(got) != 2 || got[0] != "C-0001" || got[1] != "C-0002" {
		t.Errorf("expected sorted [C-001 C-002]; got %v", got)
	}
}
