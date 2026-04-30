package verb

import (
	"strings"
	"testing"

	"github.com/23min/ai-workflow-v2/tools/internal/aiwfyaml"
)

const recipeBaseYAML = `aiwf_version: 0.1.0
actor: human/test
contracts:
  validators: {}
  entries: []
`

func TestRecipeInstall_NewValidator(t *testing.T) {
	d, c := mustReadDoc(t, recipeBaseYAML)
	res, err := RecipeInstall(d, c, "cue", aiwfyaml.Validator{
		Command: "cue", Args: []string{"vet", "{{schema}}", "{{fixture}}"},
	}, "human/test", RecipeInstallOptions{})
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
	mustHaveTrailerInPlan(t, res.Plan, "aiwf-verb", "recipe-install")
}

func TestRecipeInstall_IdempotentExactMatch(t *testing.T) {
	src := strings.Replace(recipeBaseYAML, "  validators: {}", `  validators:
    cue:
      command: cue
      args:
        - vet
        - "{{schema}}"
        - "{{fixture}}"`, 1)
	d, c := mustReadDoc(t, src)
	res, err := RecipeInstall(d, c, "cue", aiwfyaml.Validator{
		Command: "cue", Args: []string{"vet", "{{schema}}", "{{fixture}}"},
	}, "human/test", RecipeInstallOptions{})
	if err != nil {
		t.Fatalf("RecipeInstall: %v", err)
	}
	if !res.NoOp {
		t.Errorf("expected NoOp for exact-match install; got %+v", res)
	}
}

func TestRecipeInstall_DifferentRequiresForce(t *testing.T) {
	src := strings.Replace(recipeBaseYAML, "  validators: {}", `  validators:
    cue:
      command: cue
      args:
        - eval`, 1)
	d, c := mustReadDoc(t, src)
	_, err := RecipeInstall(d, c, "cue", aiwfyaml.Validator{
		Command: "cue", Args: []string{"vet"},
	}, "human/test", RecipeInstallOptions{})
	if err == nil || !strings.Contains(err.Error(), "force") {
		t.Errorf("expected force-required error; got %v", err)
	}
}

func TestRecipeInstall_TrailersIncludeReferencingBindings(t *testing.T) {
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
	res, err := RecipeInstall(d, c, "cue", aiwfyaml.Validator{
		Command: "cue", Args: []string{"vet", "--all"},
	}, "human/test", RecipeInstallOptions{Force: true})
	if err != nil {
		t.Fatalf("RecipeInstall --force: %v", err)
	}
	wantIDs := map[string]bool{"C-001": true, "C-002": true}
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
	if gotIDs["C-003"] {
		t.Error("C-003 should not have a trailer (different validator)")
	}
}

func TestRecipeRemove_Success(t *testing.T) {
	src := strings.Replace(recipeBaseYAML, "  validators: {}", `  validators:
    cue:
      command: cue
      args:
        - vet`, 1)
	d, c := mustReadDoc(t, src)
	res, err := RecipeRemove(d, c, "cue", "human/test")
	if err != nil {
		t.Fatalf("RecipeRemove: %v", err)
	}
	got := string(res.Plan.Ops[0].Content)
	if strings.Contains(got, "cue:") {
		t.Errorf("validator not removed:\n%s", got)
	}
}

func TestRecipeRemove_RejectsReferencedValidator(t *testing.T) {
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
	_, err := RecipeRemove(d, c, "cue", "human/test")
	if err == nil || !strings.Contains(err.Error(), "C-001") {
		t.Errorf("expected error naming C-001; got %v", err)
	}
}

func TestRecipeRemove_RejectsMissingValidator(t *testing.T) {
	d, c := mustReadDoc(t, recipeBaseYAML)
	if _, err := RecipeRemove(d, c, "ghost", "human/test"); err == nil {
		t.Error("expected error for missing validator")
	}
}
