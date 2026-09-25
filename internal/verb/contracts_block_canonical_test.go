package verb

import (
	"context"
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/aiwfyaml"
	"github.com/23min/aiwf/internal/entity"
	"github.com/23min/aiwf/internal/tree"
)

// legacyContractsYAML carries two bindings at a legacy narrow width, as an
// aiwf.yaml written by an older release does.
const legacyContractsYAML = `aiwf_version: 0.1.0
actor: human/test
contracts:
  validators:
    cue:
      command: cue
      args: [vet, "{{schema}}", "{{fixture}}"]
    spare:
      command: spare
  entries:
    - id: C-002
      validator: cue
      schema: schema.cue
      fixtures: fixtures
    - id: C-003
      validator: cue
      schema: schema.cue
      fixtures: fixtures
`

// TestContractsBlockWrites_WriteEveryBindingIDCanonical: every verb that
// rewrites aiwf.yaml's contracts block writes each binding id at
// canonical width — an id it was handed narrow, and a legacy narrow entry
// it was not asked to touch.
func TestContractsBlockWrites_WriteEveryBindingIDCanonical(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	tr := &tree.Tree{}
	for _, id := range []string{"C-0001", "C-0002", "C-0003"} {
		tr.Entities = append(tr.Entities, contractTree(id, "").Entities...)
	}
	ops := func(t *testing.T, res *Result, err error) []FileOp {
		t.Helper()
		if err != nil {
			t.Fatalf("verb: %v", err)
		}
		if res.Plan == nil {
			t.Fatalf("verb returned no plan: %+v", res)
		}
		return res.Plan.Ops
	}
	tests := []struct {
		name  string
		write func(t *testing.T, d *aiwfyaml.Doc, c *aiwfyaml.Contracts, root string) []FileOp
	}{
		{"contract bind", func(t *testing.T, d *aiwfyaml.Doc, c *aiwfyaml.Contracts, root string) []FileOp {
			t.Helper()
			res, err := ContractBind(ctx, tr, d, c, "C-001", "human/test", root, ContractBindOptions{
				Validator: "cue", Schema: "schema.cue", Fixtures: "fixtures",
			})
			written := ops(t, res, err)
			if want := "aiwf contract bind C-0001"; res.Plan.Subject != want {
				t.Errorf("subject = %q, want %q", res.Plan.Subject, want)
			}
			return written
		}},
		{"contract unbind", func(t *testing.T, d *aiwfyaml.Doc, c *aiwfyaml.Contracts, root string) []FileOp {
			t.Helper()
			res, err := ContractUnbind(ctx, tr, d, c, "C-003", "human/test", root)
			written := ops(t, res, err)
			if want := "aiwf contract unbind C-0003"; res.Plan.Subject != want {
				t.Errorf("subject = %q, want %q", res.Plan.Subject, want)
			}
			return written
		}},
		{"recipe install", func(t *testing.T, d *aiwfyaml.Doc, c *aiwfyaml.Contracts, root string) []FileOp {
			t.Helper()
			res, err := RecipeInstall(ctx, tr, d, c, "extra", aiwfyaml.Validator{Command: "extra"}, "human/test", root, RecipeInstallOptions{})
			return ops(t, res, err)
		}},
		{"recipe remove", func(t *testing.T, d *aiwfyaml.Doc, c *aiwfyaml.Contracts, root string) []FileOp {
			t.Helper()
			res, err := RecipeRemove(ctx, tr, d, c, "spare", "human/test", root)
			return ops(t, res, err)
		}},
		{"add contract with an atomic bind", func(t *testing.T, d *aiwfyaml.Doc, c *aiwfyaml.Contracts, root string) []FileOp {
			t.Helper()
			fileOps, findings, err := atomicContractBind(tr, "C-0001", AddOptions{
				BindValidator: "cue", BindSchema: "schema.cue", BindFixtures: "fixtures",
				AiwfDoc: d, AiwfContracts: c, RepoRoot: root,
			})
			if err != nil || len(findings) != 0 {
				t.Fatalf("atomicContractBind: err=%v findings=%+v", err, findings)
			}
			return fileOps
		}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			d, c := mustReadDoc(t, legacyContractsYAML)
			written := tc.write(t, d, c, bindRepo(t))
			if len(written) != 1 {
				t.Fatalf("want one aiwf.yaml write, got %+v", written)
			}
			_, got, err := aiwfyaml.ReadBytes(written[0].Content)
			if err != nil {
				t.Fatalf("re-read written aiwf.yaml: %v", err)
			}
			if len(got.Entries) == 0 {
				t.Fatal("written aiwf.yaml holds no bindings; nothing was measured")
			}
			for _, e := range got.Entries {
				if canon := entity.Canonicalize(e.ID); e.ID != canon {
					t.Errorf("binding written as %q, want canonical %q", e.ID, canon)
				}
			}
		})
	}
}

// TestAtomicContractBind_RefusesAStaleBindingAtAnyWidth: when aiwf.yaml
// already binds the id a new contract was allocated — a stale binding left
// by a contract that no longer exists, stored at a legacy width — the add
// refuses rather than writing a second binding for the same id.
func TestAtomicContractBind_RefusesAStaleBindingAtAnyWidth(t *testing.T) {
	t.Parallel()
	d, c := mustReadDoc(t, legacyContractsYAML)
	_, _, err := atomicContractBind(&tree.Tree{}, "C-0002", AddOptions{
		BindValidator: "cue", BindSchema: "schema.cue", BindFixtures: "fixtures",
		AiwfDoc: d, AiwfContracts: c, RepoRoot: bindRepo(t),
	})
	if err == nil {
		t.Fatal("atomicContractBind wrote a second binding for C-0002")
	}
	if want := "aiwf contract unbind C-0002"; !strings.Contains(err.Error(), want) {
		t.Errorf("refusal %q does not name the fix %q", err, want)
	}
}
