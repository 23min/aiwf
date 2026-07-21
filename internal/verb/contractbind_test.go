package verb

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/23min/aiwf/internal/aiwfyaml"
	"github.com/23min/aiwf/internal/check"
	"github.com/23min/aiwf/internal/entity"
	"github.com/23min/aiwf/internal/tree"
)

// bindRepo returns a tmpdir containing the schema/fixtures pairs
// every test in this file references (schema.cue + fixtures/, plus
// new.cue + new/ for the force-replace test). Tests that exercise
// the happy path use this so G18's verb-side contractcheck-on-
// projection finds real files. Tests that fail before reaching the
// projection check could pass "" instead; using bindRepo
// unconditionally keeps the diff simple and the helper one-liner.
func bindRepo(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	for _, file := range []string{"schema.cue", "new.cue"} {
		if err := os.WriteFile(filepath.Join(root, file), []byte(""), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	for _, dir := range []string{"fixtures", "new"} {
		if err := os.Mkdir(filepath.Join(root, dir), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

// contractFixtureBody returns minimal real prose satisfying the
// G-0326 empty-body gate for kind=contract, for tests in this file
// whose subject is the atomic add+bind wiring, not body content.
func contractFixtureBody() []byte {
	return []byte("## Purpose\n\nFixture prose for test setup; not the subject under test.\n\n" +
		"## Stability\n\nFixture prose for test setup; not the subject under test.\n")
}

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
	t.Parallel()
	tr := contractTree("C-0001", "proposed")
	d, c := mustReadDoc(t, baseAiwfYAML)

	res, err := ContractBind(context.Background(), tr, d, c, "C-0001", "human/test", bindRepo(t), ContractBindOptions{
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
	if !strings.Contains(string(res.Plan.Ops[0].Content), "C-0001") {
		t.Errorf("aiwf.yaml content missing the new entry id:\n%s", res.Plan.Ops[0].Content)
	}
	mustHaveTrailerInPlan(t, res.Plan, "aiwf-verb", "contract-bind")
	mustHaveTrailerInPlan(t, res.Plan, "aiwf-entity", "C-0001")
}

func TestContractBind_IdempotentExactMatch(t *testing.T) {
	t.Parallel()
	src := strings.Replace(baseAiwfYAML, "  entries: []", `  entries:
    - id: C-001
      validator: cue
      schema: schema.cue
      fixtures: fixtures`, 1)
	tr := contractTree("C-0001", "proposed")
	d, c := mustReadDoc(t, src)

	res, err := ContractBind(context.Background(), tr, d, c, "C-0001", "human/test", bindRepo(t), ContractBindOptions{
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
	t.Parallel()
	src := strings.Replace(baseAiwfYAML, "  entries: []", `  entries:
    - id: C-001
      validator: cue
      schema: old.cue
      fixtures: old`, 1)
	tr := contractTree("C-0001", "proposed")
	d, c := mustReadDoc(t, src)

	_, err := ContractBind(context.Background(), tr, d, c, "C-0001", "human/test", bindRepo(t), ContractBindOptions{
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
	t.Parallel()
	src := strings.Replace(baseAiwfYAML, "  entries: []", `  entries:
    - id: C-001
      validator: cue
      schema: old.cue
      fixtures: old`, 1)
	tr := contractTree("C-0001", "proposed")
	d, c := mustReadDoc(t, src)

	res, err := ContractBind(context.Background(), tr, d, c, "C-0001", "human/test", bindRepo(t), ContractBindOptions{
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
	t.Parallel()
	tr := &tree.Tree{}
	d, c := mustReadDoc(t, baseAiwfYAML)

	_, err := ContractBind(context.Background(), tr, d, c, "C-0001", "human/test", bindRepo(t), ContractBindOptions{
		Validator: "cue", Schema: "schema.cue", Fixtures: "fixtures",
	})
	if err == nil || !strings.Contains(err.Error(), "no contract entity") {
		t.Errorf("expected missing-entity error; got %v", err)
	}
}

func TestContractBind_RejectsUndeclaredValidator(t *testing.T) {
	t.Parallel()
	tr := contractTree("C-0001", "proposed")
	d, c := mustReadDoc(t, baseAiwfYAML)

	_, err := ContractBind(context.Background(), tr, d, c, "C-0001", "human/test", bindRepo(t), ContractBindOptions{
		Validator: "ghost", Schema: "schema.cue", Fixtures: "fixtures",
	})
	if err == nil || !strings.Contains(err.Error(), "ghost") {
		t.Errorf("expected error mentioning the undeclared validator; got %v", err)
	}
}

func TestContractBind_RejectsMissingFlags(t *testing.T) {
	t.Parallel()
	tr := contractTree("C-0001", "proposed")
	d, c := mustReadDoc(t, baseAiwfYAML)

	_, err := ContractBind(context.Background(), tr, d, c, "C-0001", "human/test", bindRepo(t), ContractBindOptions{
		Validator: "cue", // schema and fixtures missing
	})
	if err == nil {
		t.Fatal("expected error for missing required flags")
	}
}

func TestContractUnbind_Removes(t *testing.T) {
	t.Parallel()
	src := strings.Replace(baseAiwfYAML, "  entries: []", `  entries:
    - id: C-001
      validator: cue
      schema: s.cue
      fixtures: f`, 1)
	d, c := mustReadDoc(t, src)

	res, err := ContractUnbind(context.Background(), &tree.Tree{}, d, c, "C-0001", "human/test", t.TempDir())
	if err != nil {
		t.Fatalf("ContractUnbind: %v", err)
	}
	if res.Plan == nil {
		t.Fatal("expected Plan")
	}
	got := string(res.Plan.Ops[0].Content)
	if strings.Contains(got, "C-0001") {
		t.Errorf("entry not removed from aiwf.yaml:\n%s", got)
	}
	mustHaveTrailerInPlan(t, res.Plan, "aiwf-verb", "contract-unbind")
	mustHaveTrailerInPlan(t, res.Plan, "aiwf-entity", "C-0001")
}

// TestContractUnbind_IntroducesNoBindingWarningButDoesNotBlock: proves
// the shared gate wired into ContractUnbind (D-0041's safety net) is
// live, not dead plumbing. Unbinding a contract entity that still
// exists in the tree genuinely changes contractcheck.Run's output —
// the entity gains a fresh "no-binding" warning it didn't carry while
// bound. Confirmed two ways: the gate itself (called directly, with
// the same inputs ContractUnbind uses) reports exactly that warning
// as introduced; and ContractUnbind still succeeds (warnings don't
// block — only check.HasErrors gates the write, and unbind can never
// structurally introduce an error-severity finding).
func TestContractUnbind_IntroducesNoBindingWarningButDoesNotBlock(t *testing.T) {
	t.Parallel()
	tr := contractTree("C-0001", "proposed")
	root := bindRepo(t)
	src := `aiwf_version: 0.1.0
actor: human/test
contracts:
  validators:
    cue:
      command: cue
      args: [vet]
  entries:
    - id: C-0001
      validator: cue
      schema: schema.cue
      fixtures: fixtures
`
	d, c := mustReadDoc(t, src)

	next := cloneContracts(c)
	next.Entries = nil
	gated := contractMutationGate(tr, c, next, root)
	if len(gated) != 1 || gated[0].Subcode != "no-binding" || gated[0].EntityID != "C-0001" {
		t.Fatalf("expected exactly one introduced no-binding warning for C-0001; got %+v", gated)
	}
	if gated[0].Severity != check.SeverityWarning {
		t.Errorf("expected the no-binding finding to be a warning, not an error; got %+v", gated[0])
	}

	res, err := ContractUnbind(context.Background(), tr, d, c, "C-0001", "human/test", root)
	if err != nil {
		t.Fatalf("ContractUnbind: %v", err)
	}
	if res.Plan == nil {
		t.Fatalf("expected Plan — a warning-only introduced finding must not block unbind; got Findings: %+v", res.Findings)
	}
}

// TestContractUnbind_ConsultsTheTreeViaTheSharedGate: unbind can never
// structurally trigger check.HasErrors (only ever a warning, per the
// test above), so no behavioral assertion on ContractUnbind's return
// value can distinguish "the gate call runs" from "it was silently
// deleted" — both leave res.Plan non-nil. Passing a nil tree makes the
// distinction observable instead: contractMutationGate always reaches
// contractcheck.Run, which calls t.ByKind on the tree — a nil
// *tree.Tree panics there. If ContractUnbind stops calling the gate,
// t is never dereferenced and nothing panics.
func TestContractUnbind_ConsultsTheTreeViaTheSharedGate(t *testing.T) {
	t.Parallel()
	src := strings.Replace(baseAiwfYAML, "  entries: []", `  entries:
    - id: C-001
      validator: cue
      schema: s.cue
      fixtures: f`, 1)
	d, c := mustReadDoc(t, src)

	defer func() {
		if recover() == nil {
			t.Fatal("expected ContractUnbind to consult the tree via the shared gate (contractMutationGate) and panic on a nil tree; it didn't — the gate call may have been removed")
		}
	}()
	_, _ = ContractUnbind(context.Background(), nil, d, c, "C-0001", "human/test", t.TempDir())
}

func TestContractUnbind_RejectsMissingEntry(t *testing.T) {
	t.Parallel()
	d, c := mustReadDoc(t, baseAiwfYAML)
	_, err := ContractUnbind(context.Background(), &tree.Tree{}, d, c, "C-0001", "human/test", t.TempDir())
	if err == nil {
		t.Fatal("expected error for missing entry")
	}
}

func TestContractUnbind_RejectsNoContractsBlock(t *testing.T) {
	t.Parallel()
	src := `aiwf_version: 0.1.0
actor: human/test
`
	d, c := mustReadDoc(t, src)
	_, err := ContractUnbind(context.Background(), &tree.Tree{}, d, c, "C-0001", "human/test", t.TempDir())
	if err == nil {
		t.Fatal("expected error when no contracts: block exists")
	}
}

// cloneContracts is a private helper but its behavior matters for
// every mutating verb that touches contracts: we exercise it
// directly so a regression here surfaces immediately.
func TestCloneContracts_DeepCopy(t *testing.T) {
	t.Parallel()
	src := &aiwfyaml.Contracts{
		Validators: map[string]aiwfyaml.Validator{
			"cue": {Command: "cue", Args: []string{"vet"}},
		},
		Entries: []aiwfyaml.Entry{{ID: "C-0001", Validator: "cue", Schema: "s", Fixtures: "f"}},
	}
	dst := cloneContracts(src)
	dst.Validators["cue"] = aiwfyaml.Validator{Command: "tampered"}
	dst.Entries[0].Schema = "tampered"
	dst.Entries = append(dst.Entries, aiwfyaml.Entry{ID: "C-0002"})

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
	t.Parallel()
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
		RepoRoot:      bindRepo(t),
		BodyOverride:  contractFixtureBody(),
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
	t.Parallel()
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
	t.Parallel()
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
	t.Parallel()
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
	t.Parallel()
	tr := &tree.Tree{}
	d, c := mustReadDoc(t, baseAiwfYAML)
	_, err := Add(context.Background(), tr, entity.KindContract, "API", "human/test", AddOptions{
		BindValidator: "ghost",
		BindSchema:    "schema.cue",
		BindFixtures:  "fixtures",
		AiwfDoc:       d,
		AiwfContracts: c,
		BodyOverride:  contractFixtureBody(),
	})
	if err == nil || !strings.Contains(err.Error(), "ghost") {
		t.Errorf("expected error naming the undeclared validator; got %v", err)
	}
}

// TestContractBind_RejectsEmptyID: bind needs a non-empty C-id; the
// CLI dispatcher errors at parse time on an empty positional, but
// the verb itself should also refuse a programmatic empty id (defensive).
func TestContractBind_RejectsEmptyID(t *testing.T) {
	t.Parallel()
	tr := &tree.Tree{}
	d, c := mustReadDoc(t, baseAiwfYAML)
	_, err := ContractBind(context.Background(), tr, d, c, "", "human/test", bindRepo(t), ContractBindOptions{
		Validator: "cue", Schema: "s", Fixtures: "f",
	})
	if err == nil {
		t.Error("expected error for empty id")
	}
}

// TestContractBind_RejectsNonContractEntity: an id that exists but
// resolves to (e.g.) an epic must be refused with a clear message.
func TestContractBind_RejectsNonContractEntity(t *testing.T) {
	t.Parallel()
	tr := &tree.Tree{
		Entities: []*entity.Entity{{
			ID: "E-0001", Kind: entity.KindEpic, Title: "Foo", Status: "active",
			Path: "work/epics/E-01-foo/epic.md",
		}},
	}
	d, c := mustReadDoc(t, baseAiwfYAML)
	_, err := ContractBind(context.Background(), tr, d, c, "E-0001", "human/test", bindRepo(t), ContractBindOptions{
		Validator: "cue", Schema: "s", Fixtures: "f",
	})
	if err == nil || !strings.Contains(err.Error(), "epic") {
		t.Errorf("expected error mentioning the kind mismatch; got %v", err)
	}
}

// TestContractUnbind_OnlyRemovesNamedID: unbind on one of several
// bindings must keep the rest untouched.
func TestContractUnbind_OnlyRemovesNamedID(t *testing.T) {
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
	res, err := ContractUnbind(context.Background(), &tree.Tree{}, d, c, "C-0002", "human/test", t.TempDir())
	if err != nil {
		t.Fatalf("ContractUnbind: %v", err)
	}
	got := string(res.Plan.Ops[0].Content)
	// On-disk yaml entries preserve their authored width — width
	// canonicalization of body content is M-082's `aiwf rewidth` job.
	// Both narrow and canonical absence-checks confirm removal.
	if strings.Contains(got, "C-0002") || strings.Contains(got, "C-002") {
		t.Errorf("C-002 not removed:\n%s", got)
	}
	for _, keep := range []string{"C-001", "C-003"} {
		if !strings.Contains(got, keep) {
			t.Errorf("expected %s to remain:\n%s", keep, got)
		}
	}
}

// TestContractBind_G18_MissingSchemaCaughtAtVerb: G18 — when
// the bound schema path doesn't exist on disk, the verb returns the
// contractcheck/missing-schema finding instead of writing the bad
// binding and deferring detection to the pre-push hook.
func TestContractBind_G18_MissingSchemaCaughtAtVerb(t *testing.T) {
	t.Parallel()
	tr := contractTree("C-0001", "proposed")
	d, c := mustReadDoc(t, baseAiwfYAML)
	root := t.TempDir() // no schema.cue, no fixtures/

	res, err := ContractBind(context.Background(), tr, d, c, "C-0001", "human/test", root, ContractBindOptions{
		Validator: "cue", Schema: "schema.cue", Fixtures: "fixtures",
	})
	if err != nil {
		t.Fatalf("unexpected Go error: %v", err)
	}
	if res.Plan != nil {
		t.Fatal("expected Findings, got Plan — G18 check did not block the bind")
	}
	if len(res.Findings) == 0 {
		t.Fatal("expected at least one finding")
	}
	var sawMissingSchema, sawMissingFixtures bool
	for _, f := range res.Findings {
		if f.Code == "contract-config" && f.Subcode == "missing-schema" {
			sawMissingSchema = true
		}
		if f.Code == "contract-config" && f.Subcode == "missing-fixtures" {
			sawMissingFixtures = true
		}
		if f.EntityID != "C-0001" {
			t.Errorf("finding for unrelated entity %q surfaced from this verb: %+v", f.EntityID, f)
		}
	}
	if !sawMissingSchema {
		t.Error("expected contract-config/missing-schema finding")
	}
	if !sawMissingFixtures {
		t.Error("expected contract-config/missing-fixtures finding")
	}
}

// TestContractBind_G18_OnlyTouchesBoundEntity: the verb's verification
// must filter to findings on its own bound id; pre-existing findings
// on other entries (a stale binding pointing at a deleted schema) are
// not the verb's responsibility and should not block it.
func TestContractBind_G18_OnlyTouchesBoundEntity(t *testing.T) {
	t.Parallel()
	// aiwf.yaml has a pre-existing binding for C-002 pointing at
	// missing files, plus a contract entity for C-002 so the only
	// per-entry finding is missing-schema/missing-fixtures.
	src := strings.Replace(baseAiwfYAML, "  entries: []", `  entries:
    - id: C-002
      validator: cue
      schema: gone.cue
      fixtures: gone`, 1)
	tr := &tree.Tree{
		Entities: []*entity.Entity{
			{ID: "C-0001", Kind: entity.KindContract, Title: "new", Status: "proposed", Path: "work/contracts/C-001-new/contract.md"},
			{ID: "C-0002", Kind: entity.KindContract, Title: "stale", Status: "proposed", Path: "work/contracts/C-002-stale/contract.md"},
		},
	}
	d, c := mustReadDoc(t, src)

	res, err := ContractBind(context.Background(), tr, d, c, "C-0001", "human/test", bindRepo(t), ContractBindOptions{
		Validator: "cue", Schema: "schema.cue", Fixtures: "fixtures",
	})
	if err != nil {
		t.Fatalf("ContractBind: %v", err)
	}
	if res.Plan == nil {
		t.Fatalf("expected Plan; got Findings (the verb should ignore pre-existing C-002 errors): %+v", res.Findings)
	}
}

// TestAdd_ContractWithBindBadPathsCaughtAtVerb: G18 — same
// enforcement on the atomic add+bind path through verb.Add.
func TestAdd_ContractWithBindBadPathsCaughtAtVerb(t *testing.T) {
	t.Parallel()
	tr := &tree.Tree{}
	d, c := mustReadDoc(t, baseAiwfYAML)

	res, err := Add(context.Background(), tr, entity.KindContract, "API", "human/test", AddOptions{
		BindValidator: "cue",
		BindSchema:    "schema.cue", // doesn't exist under root
		BindFixtures:  "fixtures",
		AiwfDoc:       d,
		AiwfContracts: c,
		RepoRoot:      t.TempDir(), // empty tmpdir; nothing on disk
		BodyOverride:  contractFixtureBody(),
	})
	if err != nil {
		t.Fatalf("unexpected Go error: %v", err)
	}
	if res.Plan != nil {
		t.Fatal("expected Findings, got Plan — G18 check did not block atomic add+bind")
	}
	if len(res.Findings) == 0 {
		t.Fatal("expected at least one finding")
	}
	var sawIt bool
	for _, f := range res.Findings {
		if f.Code == "contract-config" && (f.Subcode == "missing-schema" || f.Subcode == "missing-fixtures") {
			sawIt = true
			break
		}
	}
	if !sawIt {
		t.Errorf("expected contract-config/missing-{schema,fixtures} finding; got %+v", res.Findings)
	}
}

// TestContractBind_PartialBindOptionsRejected: missing fixtures on
// the verb level (not just CLI level) errors.
func TestContractBind_PartialBindOptionsRejected(t *testing.T) {
	t.Parallel()
	tr := contractTree("C-0001", "proposed")
	d, c := mustReadDoc(t, baseAiwfYAML)
	_, err := ContractBind(context.Background(), tr, d, c, "C-0001", "human/test", bindRepo(t), ContractBindOptions{
		Validator: "cue",
		Schema:    "s",
		// Fixtures missing
	})
	if err == nil {
		t.Error("expected error for missing --fixtures")
	}
}
