package verb

import (
	"context"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/23min/ai-workflow-v2/tools/internal/aiwfyaml"
	"github.com/23min/ai-workflow-v2/tools/internal/entity"
	"github.com/23min/ai-workflow-v2/tools/internal/tree"
)

const baseAiwfYAML = `aiwf_version: 0.1.0
actor: human/test
contracts:
  validators:
    cue:
      command: cue
      args: [vet, "{{schema}}", "{{fixture}}"]
  entries: []
`

func contractTree(id, status string) *tree.Tree {
	return &tree.Tree{
		Entities: []*entity.Entity{{
			ID:    id,
			Kind:  entity.KindContract,
			Title: "Test contract",
			Status: func() string {
				if status == "" {
					return "proposed"
				}
				return status
			}(),
			Path: "work/contracts/" + id + "-test/contract.md",
		}},
	}
}

func mustReadDoc(t *testing.T, src string) (*aiwfyaml.Doc, *aiwfyaml.Contracts) {
	t.Helper()
	d, c, err := aiwfyaml.ReadBytes([]byte(src))
	if err != nil {
		t.Fatalf("ReadBytes: %v", err)
	}
	return d, c
}

func TestContractBind_NewBinding(t *testing.T) {
	tr := contractTree("C-001", "proposed")
	d, c := mustReadDoc(t, baseAiwfYAML)

	res, err := ContractBind(context.Background(), tr, d, c, "C-001", "human/test", ContractBindOptions{
		Validator: "cue", Schema: "schema.cue", Fixtures: "fixtures",
	})
	if err != nil {
		t.Fatalf("ContractBind: %v", err)
	}
	if res.Plan == nil {
		t.Fatal("expected a Plan; got NoOp or nil")
	}
	if len(res.Plan.Ops) != 1 || res.Plan.Ops[0].Path != "aiwf.yaml" {
		t.Errorf("expected single OpWrite for aiwf.yaml; got %+v", res.Plan.Ops)
	}
	if !strings.Contains(string(res.Plan.Ops[0].Content), "C-001") {
		t.Errorf("aiwf.yaml content missing the new entry id:\n%s", res.Plan.Ops[0].Content)
	}
	mustHaveTrailerInPlan(t, res.Plan, "aiwf-verb", "bind")
	mustHaveTrailerInPlan(t, res.Plan, "aiwf-entity", "C-001")
}

func TestContractBind_IdempotentExactMatch(t *testing.T) {
	src := strings.Replace(baseAiwfYAML, "  entries: []", `  entries:
    - id: C-001
      validator: cue
      schema: schema.cue
      fixtures: fixtures`, 1)
	tr := contractTree("C-001", "proposed")
	d, c := mustReadDoc(t, src)

	res, err := ContractBind(context.Background(), tr, d, c, "C-001", "human/test", ContractBindOptions{
		Validator: "cue", Schema: "schema.cue", Fixtures: "fixtures",
	})
	if err != nil {
		t.Fatalf("ContractBind: %v", err)
	}
	if !res.NoOp {
		t.Errorf("expected NoOp result; got %+v", res)
	}
	if !strings.Contains(res.NoOpMessage, "unchanged") {
		t.Errorf("NoOpMessage = %q, want a 'unchanged' message", res.NoOpMessage)
	}
}

func TestContractBind_DifferentValuesRequiresForce(t *testing.T) {
	src := strings.Replace(baseAiwfYAML, "  entries: []", `  entries:
    - id: C-001
      validator: cue
      schema: old.cue
      fixtures: old`, 1)
	tr := contractTree("C-001", "proposed")
	d, c := mustReadDoc(t, src)

	_, err := ContractBind(context.Background(), tr, d, c, "C-001", "human/test", ContractBindOptions{
		Validator: "cue", Schema: "new.cue", Fixtures: "new",
	})
	if err == nil {
		t.Fatal("expected error without --force")
	}
	if !strings.Contains(err.Error(), "force") {
		t.Errorf("error %q does not mention --force", err)
	}
}

func TestContractBind_ForceReplaces(t *testing.T) {
	src := strings.Replace(baseAiwfYAML, "  entries: []", `  entries:
    - id: C-001
      validator: cue
      schema: old.cue
      fixtures: old`, 1)
	tr := contractTree("C-001", "proposed")
	d, c := mustReadDoc(t, src)

	res, err := ContractBind(context.Background(), tr, d, c, "C-001", "human/test", ContractBindOptions{
		Validator: "cue", Schema: "new.cue", Fixtures: "new", Force: true,
	})
	if err != nil {
		t.Fatalf("ContractBind --force: %v", err)
	}
	if res.Plan == nil {
		t.Fatal("expected a Plan")
	}
	got := string(res.Plan.Ops[0].Content)
	if !strings.Contains(got, "new.cue") || strings.Contains(got, "old.cue") {
		t.Errorf("aiwf.yaml content not updated:\n%s", got)
	}
}

func TestContractBind_RejectsMissingEntity(t *testing.T) {
	tr := &tree.Tree{}
	d, c := mustReadDoc(t, baseAiwfYAML)

	_, err := ContractBind(context.Background(), tr, d, c, "C-001", "human/test", ContractBindOptions{
		Validator: "cue", Schema: "schema.cue", Fixtures: "fixtures",
	})
	if err == nil || !strings.Contains(err.Error(), "no contract entity") {
		t.Errorf("expected missing-entity error; got %v", err)
	}
}

func TestContractBind_RejectsUndeclaredValidator(t *testing.T) {
	tr := contractTree("C-001", "proposed")
	d, c := mustReadDoc(t, baseAiwfYAML)

	_, err := ContractBind(context.Background(), tr, d, c, "C-001", "human/test", ContractBindOptions{
		Validator: "ghost", Schema: "schema.cue", Fixtures: "fixtures",
	})
	if err == nil || !strings.Contains(err.Error(), "ghost") {
		t.Errorf("expected error mentioning the undeclared validator; got %v", err)
	}
}

func TestContractBind_RejectsMissingFlags(t *testing.T) {
	tr := contractTree("C-001", "proposed")
	d, c := mustReadDoc(t, baseAiwfYAML)

	_, err := ContractBind(context.Background(), tr, d, c, "C-001", "human/test", ContractBindOptions{
		Validator: "cue", // schema and fixtures missing
	})
	if err == nil {
		t.Fatal("expected error for missing required flags")
	}
}

func TestContractUnbind_Removes(t *testing.T) {
	src := strings.Replace(baseAiwfYAML, "  entries: []", `  entries:
    - id: C-001
      validator: cue
      schema: s.cue
      fixtures: f`, 1)
	d, c := mustReadDoc(t, src)

	res, err := ContractUnbind(context.Background(), d, c, "C-001", "human/test")
	if err != nil {
		t.Fatalf("ContractUnbind: %v", err)
	}
	if res.Plan == nil {
		t.Fatal("expected Plan")
	}
	got := string(res.Plan.Ops[0].Content)
	if strings.Contains(got, "C-001") {
		t.Errorf("entry not removed from aiwf.yaml:\n%s", got)
	}
	mustHaveTrailerInPlan(t, res.Plan, "aiwf-verb", "unbind")
	mustHaveTrailerInPlan(t, res.Plan, "aiwf-entity", "C-001")
}

func TestContractUnbind_RejectsMissingEntry(t *testing.T) {
	d, c := mustReadDoc(t, baseAiwfYAML)
	_, err := ContractUnbind(context.Background(), d, c, "C-001", "human/test")
	if err == nil {
		t.Fatal("expected error for missing entry")
	}
}

func TestContractUnbind_RejectsNoContractsBlock(t *testing.T) {
	src := `aiwf_version: 0.1.0
actor: human/test
`
	d, c := mustReadDoc(t, src)
	_, err := ContractUnbind(context.Background(), d, c, "C-001", "human/test")
	if err == nil {
		t.Fatal("expected error when no contracts: block exists")
	}
}

// cloneContracts is a private helper but its behavior matters for
// every mutating verb that touches contracts: we exercise it
// directly so a regression here surfaces immediately.
func TestCloneContracts_DeepCopy(t *testing.T) {
	src := &aiwfyaml.Contracts{
		Validators: map[string]aiwfyaml.Validator{
			"cue": {Command: "cue", Args: []string{"vet"}},
		},
		Entries: []aiwfyaml.Entry{{ID: "C-001", Validator: "cue", Schema: "s", Fixtures: "f"}},
	}
	dst := cloneContracts(src)
	dst.Validators["cue"] = aiwfyaml.Validator{Command: "tampered"}
	dst.Entries[0].Schema = "tampered"
	dst.Entries = append(dst.Entries, aiwfyaml.Entry{ID: "C-002"})

	if src.Validators["cue"].Command != "cue" {
		t.Errorf("source validators map mutated: %+v", src.Validators)
	}
	if src.Entries[0].Schema != "s" {
		t.Errorf("source entries mutated: %+v", src.Entries)
	}
	if len(src.Entries) != 1 {
		t.Errorf("source entries length changed: %d", len(src.Entries))
	}
	// Sanity: clone really copied the original values across.
	if diff := cmp.Diff([]string{"vet"}, src.Validators["cue"].Args); diff != "" {
		t.Errorf("args mismatch: %s", diff)
	}
}

func mustHaveTrailerInPlan(t *testing.T, p *Plan, key, value string) {
	t.Helper()
	for _, tr := range p.Trailers {
		if tr.Key == key && tr.Value == value {
			return
		}
	}
	t.Errorf("trailer %s=%q missing from plan: %+v", key, value, p.Trailers)
}

// TestAdd_ContractWithBindingProducesTwoOps: when --validator/--schema/
// --fixtures are supplied to `aiwf add contract`, the resulting Plan
// must carry two OpWrites — one for the entity file, one for the
// updated aiwf.yaml — so the atomic bind lands in a single commit.
func TestAdd_ContractWithBindingProducesTwoOps(t *testing.T) {
	tr := &tree.Tree{
		Entities: []*entity.Entity{{
			ID: "ADR-0001", Kind: entity.KindADR, Title: "Adopt Cue", Status: "accepted",
			Path: "docs/adr/ADR-0001-adopt-cue.md",
		}},
	}
	d, c := mustReadDoc(t, baseAiwfYAML)

	res, err := Add(context.Background(), tr, entity.KindContract, "Public API", "human/test", AddOptions{
		LinkedADRs:    []string{"ADR-0001"},
		BindValidator: "cue",
		BindSchema:    "schema.cue",
		BindFixtures:  "fixtures",
		AiwfDoc:       d,
		AiwfContracts: c,
	})
	if err != nil {
		t.Fatalf("Add: %v", err)
	}
	if res.Plan == nil {
		t.Fatal("expected Plan")
	}
	if len(res.Plan.Ops) != 2 {
		t.Fatalf("expected 2 ops (entity + aiwf.yaml), got %d: %+v", len(res.Plan.Ops), res.Plan.Ops)
	}
	pathByOp := map[string]bool{}
	for _, op := range res.Plan.Ops {
		pathByOp[op.Path] = true
	}
	if !pathByOp["aiwf.yaml"] {
		t.Errorf("aiwf.yaml OpWrite missing from plan: %+v", res.Plan.Ops)
	}
	// And the entity content carries the linked-adr.
	for _, op := range res.Plan.Ops {
		if op.Path == "aiwf.yaml" {
			continue
		}
		if !strings.Contains(string(op.Content), "ADR-0001") {
			t.Errorf("entity file missing linked_adrs:\n%s", op.Content)
		}
	}
}

func TestAdd_ContractRejectsPartialBindTriplet(t *testing.T) {
	tr := &tree.Tree{}
	d, c := mustReadDoc(t, baseAiwfYAML)
	_, err := Add(context.Background(), tr, entity.KindContract, "Public API", "human/test", AddOptions{
		BindValidator: "cue",
		// schema and fixtures missing
		AiwfDoc:       d,
		AiwfContracts: c,
	})
	if err == nil || !strings.Contains(err.Error(), "all of") {
		t.Errorf("expected partial-triplet error; got %v", err)
	}
}

func TestAdd_NonContractRejectsContractFlags(t *testing.T) {
	tr := &tree.Tree{}
	_, err := Add(context.Background(), tr, entity.KindEpic, "Epic", "human/test", AddOptions{
		LinkedADRs: []string{"ADR-0001"},
	})
	if err == nil {
		t.Fatal("expected error for --linked-adr on non-contract kind")
	}
}

// --- Edge case coverage (added during the I1 hardening pass) ---

// TestAdd_ContractBindWithoutAiwfDocRejected: requesting --validator
// /etc on a kind=contract add without supplying the editable doc is
// a usage error. Without the doc we can't perform the atomic splice,
// so we refuse rather than write the entity in isolation.
func TestAdd_ContractBindWithoutAiwfDocRejected(t *testing.T) {
	tr := &tree.Tree{}
	_, err := Add(context.Background(), tr, entity.KindContract, "API", "human/test", AddOptions{
		BindValidator: "cue",
		BindSchema:    "schema.cue",
		BindFixtures:  "fixtures",
		// AiwfDoc intentionally nil
	})
	if err == nil {
		t.Fatal("expected error when bind flags are set but AiwfDoc is nil")
	}
}

// TestAdd_ContractBindWithUndeclaredValidatorRejected: the atomic
// add+bind variant must validate the validator name *before* writing
// any file ops. The verb is all-or-nothing across both files.
func TestAdd_ContractBindWithUndeclaredValidatorRejected(t *testing.T) {
	tr := &tree.Tree{}
	d, c := mustReadDoc(t, baseAiwfYAML)
	_, err := Add(context.Background(), tr, entity.KindContract, "API", "human/test", AddOptions{
		BindValidator: "ghost",
		BindSchema:    "schema.cue",
		BindFixtures:  "fixtures",
		AiwfDoc:       d,
		AiwfContracts: c,
	})
	if err == nil || !strings.Contains(err.Error(), "ghost") {
		t.Errorf("expected error naming the undeclared validator; got %v", err)
	}
}

// TestContractBind_RejectsEmptyID: bind needs a non-empty C-id; the
// CLI dispatcher errors at parse time on an empty positional, but
// the verb itself should also refuse a programmatic empty id (defensive).
func TestContractBind_RejectsEmptyID(t *testing.T) {
	tr := &tree.Tree{}
	d, c := mustReadDoc(t, baseAiwfYAML)
	_, err := ContractBind(context.Background(), tr, d, c, "", "human/test", ContractBindOptions{
		Validator: "cue", Schema: "s", Fixtures: "f",
	})
	if err == nil {
		t.Error("expected error for empty id")
	}
}

// TestContractBind_RejectsNonContractEntity: an id that exists but
// resolves to (e.g.) an epic must be refused with a clear message.
func TestContractBind_RejectsNonContractEntity(t *testing.T) {
	tr := &tree.Tree{
		Entities: []*entity.Entity{{
			ID: "E-01", Kind: entity.KindEpic, Title: "Foo", Status: "active",
			Path: "work/epics/E-01-foo/epic.md",
		}},
	}
	d, c := mustReadDoc(t, baseAiwfYAML)
	_, err := ContractBind(context.Background(), tr, d, c, "E-01", "human/test", ContractBindOptions{
		Validator: "cue", Schema: "s", Fixtures: "f",
	})
	if err == nil || !strings.Contains(err.Error(), "epic") {
		t.Errorf("expected error mentioning the kind mismatch; got %v", err)
	}
}

// TestContractUnbind_OnlyRemovesNamedID: unbind on one of several
// bindings must keep the rest untouched.
func TestContractUnbind_OnlyRemovesNamedID(t *testing.T) {
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
    - id: C-002
      validator: cue
      schema: b.cue
      fixtures: fb
    - id: C-003
      validator: cue
      schema: c.cue
      fixtures: fc
`
	d, c := mustReadDoc(t, src)
	res, err := ContractUnbind(context.Background(), d, c, "C-002", "human/test")
	if err != nil {
		t.Fatalf("ContractUnbind: %v", err)
	}
	got := string(res.Plan.Ops[0].Content)
	if strings.Contains(got, "C-002") {
		t.Errorf("C-002 not removed:\n%s", got)
	}
	for _, keep := range []string{"C-001", "C-003"} {
		if !strings.Contains(got, keep) {
			t.Errorf("expected %s to remain:\n%s", keep, got)
		}
	}
}

// TestContractBind_PartialBindOptionsRejected: missing fixtures on
// the verb level (not just CLI level) errors.
func TestContractBind_PartialBindOptionsRejected(t *testing.T) {
	tr := contractTree("C-001", "proposed")
	d, c := mustReadDoc(t, baseAiwfYAML)
	_, err := ContractBind(context.Background(), tr, d, c, "C-001", "human/test", ContractBindOptions{
		Validator: "cue",
		Schema:    "s",
		// Fixtures missing
	})
	if err == nil {
		t.Error("expected error for missing --fixtures")
	}
}
