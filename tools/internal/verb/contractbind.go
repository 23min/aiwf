package verb

import (
	"fmt"

	"github.com/23min/ai-workflow-v2/tools/internal/aiwfyaml"
	"github.com/23min/ai-workflow-v2/tools/internal/config"
	"github.com/23min/ai-workflow-v2/tools/internal/entity"
	"github.com/23min/ai-workflow-v2/tools/internal/gitops"
	"github.com/23min/ai-workflow-v2/tools/internal/tree"
)

// ContractBindOptions carries the bind-time arguments. All three
// path/name fields are required; Force is the escape hatch for the
// "binding already exists with different values" guard.
type ContractBindOptions struct {
	Validator string
	Schema    string
	Fixtures  string
	Force     bool
}

// ContractBind creates or replaces the binding for a contract entity
// in aiwf.yaml.contracts.entries[].
//
// The verb is idempotent against an exact match (returns a NoOp
// result), errors out when the existing binding differs unless
// opts.Force is set, and validates that:
//
//   - the contract entity exists in the tree;
//   - the validator name is declared in aiwf.yaml.contracts.validators
//     (unless current is nil — then the verb refuses, since there is
//     no validator universe to choose from yet).
//
// On success, the returned Plan carries one OpWrite for aiwf.yaml
// with the spliced contracts: block; the orchestrator commits it
// with the bind trailers.
func ContractBind(t *tree.Tree, doc *aiwfyaml.Doc, current *aiwfyaml.Contracts, id, actor string, opts ContractBindOptions) (*Result, error) {
	if doc == nil {
		return nil, fmt.Errorf("aiwf.yaml not found; run 'aiwf init' first")
	}
	if opts.Validator == "" || opts.Schema == "" || opts.Fixtures == "" {
		return nil, fmt.Errorf("contract bind requires --validator, --schema, and --fixtures")
	}
	e := t.ByID(id)
	if e == nil {
		return nil, fmt.Errorf("no contract entity %s found; create it first via 'aiwf add contract'", id)
	}
	if e.Kind != entity.KindContract {
		return nil, fmt.Errorf("%s is not a contract (it's a %s)", id, e.Kind)
	}

	next := cloneContracts(current)
	if _, ok := next.Validators[opts.Validator]; !ok {
		return nil, fmt.Errorf("validator %q not declared; install via 'aiwf contract recipe install %s' or 'aiwf contract recipe install --from <path>'", opts.Validator, opts.Validator)
	}

	desired := aiwfyaml.Entry{
		ID:        id,
		Validator: opts.Validator,
		Schema:    opts.Schema,
		Fixtures:  opts.Fixtures,
	}

	existingIdx := -1
	for i, en := range next.Entries {
		if en.ID == id {
			existingIdx = i
			break
		}
	}

	switch {
	case existingIdx >= 0 && next.Entries[existingIdx] == desired:
		return &Result{NoOp: true, NoOpMessage: fmt.Sprintf("binding for %s unchanged", id)}, nil
	case existingIdx >= 0 && !opts.Force:
		return nil, fmt.Errorf("binding for %s already exists with different values; pass --force to replace", id)
	case existingIdx >= 0:
		next.Entries[existingIdx] = desired
	default:
		next.Entries = append(next.Entries, desired)
	}

	if err := doc.SetContracts(next); err != nil {
		return nil, fmt.Errorf("updating aiwf.yaml: %w", err)
	}

	return plan(&Plan{
		Subject: fmt.Sprintf("aiwf contract bind %s", id),
		Trailers: []gitops.Trailer{
			{Key: "aiwf-verb", Value: "bind"},
			{Key: "aiwf-entity", Value: id},
			{Key: "aiwf-actor", Value: actor},
		},
		Ops: []FileOp{{Type: OpWrite, Path: config.FileName, Content: doc.Bytes()}},
	}), nil
}

// ContractUnbind removes the binding for a contract from
// aiwf.yaml.contracts.entries[]. The contract entity is left
// untouched; its status governs whether pre-push verification still
// runs (it doesn't, once unbound). Errors when no binding exists.
func ContractUnbind(doc *aiwfyaml.Doc, current *aiwfyaml.Contracts, id, actor string) (*Result, error) {
	if doc == nil {
		return nil, fmt.Errorf("aiwf.yaml not found; run 'aiwf init' first")
	}
	if current == nil {
		return nil, fmt.Errorf("no binding for %s in aiwf.yaml.contracts.entries", id)
	}

	next := cloneContracts(current)
	out := next.Entries[:0]
	found := false
	for _, en := range next.Entries {
		if en.ID == id {
			found = true
			continue
		}
		out = append(out, en)
	}
	if !found {
		return nil, fmt.Errorf("no binding for %s in aiwf.yaml.contracts.entries", id)
	}
	next.Entries = out

	if err := doc.SetContracts(next); err != nil {
		return nil, fmt.Errorf("updating aiwf.yaml: %w", err)
	}

	return plan(&Plan{
		Subject: fmt.Sprintf("aiwf contract unbind %s", id),
		Trailers: []gitops.Trailer{
			{Key: "aiwf-verb", Value: "unbind"},
			{Key: "aiwf-entity", Value: id},
			{Key: "aiwf-actor", Value: actor},
		},
		Ops: []FileOp{{Type: OpWrite, Path: config.FileName, Content: doc.Bytes()}},
	}), nil
}

// cloneContracts returns a deep-enough copy of c that callers can
// mutate the result without disturbing the input. A nil input
// produces a fresh empty Contracts so verbs don't have to nil-check
// the slices they're about to extend.
func cloneContracts(c *aiwfyaml.Contracts) *aiwfyaml.Contracts {
	out := &aiwfyaml.Contracts{
		Validators: make(map[string]aiwfyaml.Validator),
	}
	if c == nil {
		return out
	}
	for k, v := range c.Validators {
		out.Validators[k] = aiwfyaml.Validator{
			Command: v.Command,
			Args:    append([]string(nil), v.Args...),
		}
	}
	out.Entries = append([]aiwfyaml.Entry(nil), c.Entries...)
	return out
}
