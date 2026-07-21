package verb

import (
	"context"
	"fmt"
	"sort"

	"github.com/23min/aiwf/internal/aiwfyaml"
	"github.com/23min/aiwf/internal/check"
	"github.com/23min/aiwf/internal/config"
	"github.com/23min/aiwf/internal/entity"
	"github.com/23min/aiwf/internal/gitops"
	"github.com/23min/aiwf/internal/tree"
)

// RecipeInstallOptions carries the recipe-install arguments shared
// between embedded-name and --from-path entry points. Force allows
// replacing a validator that already exists with a different shape.
type RecipeInstallOptions struct {
	Force bool
}

// RecipeInstall registers `validator` under `name` in
// aiwf.yaml.contracts.validators. The verb is idempotent on exact
// match (NoOp result), errors when an existing validator carries the
// same name with different fields unless opts.Force is set.
//
// The returned Plan trailers carry one `aiwf-entity:` per binding
// currently referencing `name` in aiwf.yaml.contracts.entries[] so
// `aiwf history` for those contracts surfaces the recipe change.
//
// t and repoRoot feed the shared diff-based gate (D-0041): installing
// a validator only touches contracts.validators, never entries[], so
// in practice the gate can never find an introduced finding here —
// it is wired in as a safety net, not because this mutation is
// expected to trip it.
func RecipeInstall(ctx context.Context, t *tree.Tree, doc *aiwfyaml.Doc, current *aiwfyaml.Contracts, name string, validator aiwfyaml.Validator, actor, repoRoot string, opts RecipeInstallOptions) (*Result, error) {
	_ = ctx
	if doc == nil {
		return nil, fmt.Errorf("aiwf.yaml not found; run 'aiwf init' first")
	}
	if name == "" {
		return nil, fmt.Errorf("validator name is required")
	}
	if validator.Command == "" {
		return nil, fmt.Errorf("validator command is required")
	}

	next := cloneContracts(current)
	if existing, ok := next.Validators[name]; ok {
		if validatorEqual(existing, validator) {
			return &Result{NoOp: true, NoOpMessage: fmt.Sprintf("validator %q unchanged", name)}, nil
		}
		if !opts.Force {
			return nil, fmt.Errorf("validator %q already declared with different definition; pass --force to replace", name)
		}
	}
	next.Validators[name] = aiwfyaml.Validator{
		Command: validator.Command,
		Args:    append([]string(nil), validator.Args...),
	}

	if introduced := contractMutationGate(t, current, next, repoRoot); check.HasErrors(introduced) {
		return findings(introduced), nil //coverage:ignore installing a validator never touches contracts.Entries, so contractMutationGate's diff is always empty here; this branch is a safety net for a scenario no real input can construct today, per the doc comment above.
	}

	if err := doc.SetContracts(next); err != nil {
		return nil, fmt.Errorf("updating aiwf.yaml: %w", err)
	}

	trailers := []gitops.Trailer{
		{Key: gitops.TrailerVerb, Value: "contract-recipe-install"},
		{Key: gitops.TrailerActor, Value: actor},
	}
	for _, id := range bindingsReferencing(current, name) {
		trailers = append(trailers, gitops.Trailer{Key: gitops.TrailerEntity, Value: id})
	}

	result := plan(&Plan{
		Subject:  fmt.Sprintf("aiwf contract recipe install %s", name),
		Trailers: trailers,
		Ops:      []FileOp{{Type: OpWrite, Path: config.FileName, Content: doc.Bytes()}},
	})
	result.Metadata = map[string]any{"validator": name}
	return result, nil
}

// RecipeRemove removes the named validator from
// aiwf.yaml.contracts.validators. Errors when one or more bindings
// in entries[] still reference the validator — the user must
// `unbind` or rebind those contracts first. That referential-
// integrity check runs before the shared gate and keeps its own
// precise error message (the milestone's constraint): the gate is an
// additional safety net on top, not a replacement for it.
func RecipeRemove(ctx context.Context, t *tree.Tree, doc *aiwfyaml.Doc, current *aiwfyaml.Contracts, name, actor, repoRoot string) (*Result, error) {
	_ = ctx
	if doc == nil {
		return nil, fmt.Errorf("aiwf.yaml not found; run 'aiwf init' first")
	}
	if current == nil {
		return nil, fmt.Errorf("validator %q not declared", name)
	}
	if _, ok := current.Validators[name]; !ok {
		return nil, fmt.Errorf("validator %q not declared", name)
	}
	if refs := bindingsReferencing(current, name); len(refs) > 0 {
		return nil, fmt.Errorf("validator %q is referenced by bindings: %s. Unbind or rebind those contracts first", name, joinIDs(refs))
	}

	next := cloneContracts(current)
	delete(next.Validators, name)

	if introduced := contractMutationGate(t, current, next, repoRoot); check.HasErrors(introduced) {
		return findings(introduced), nil //coverage:ignore removing a validator never touches contracts.Entries (and the referential-integrity check above already refuses when any entry still references it), so contractMutationGate's diff is always empty here; this branch is a safety net for a scenario no real input can construct today, per the doc comment above.
	}

	if err := doc.SetContracts(next); err != nil {
		return nil, fmt.Errorf("updating aiwf.yaml: %w", err)
	}
	result := plan(&Plan{
		Subject: fmt.Sprintf("aiwf contract recipe remove %s", name),
		Trailers: []gitops.Trailer{
			{Key: gitops.TrailerVerb, Value: "contract-recipe-remove"},
			{Key: gitops.TrailerActor, Value: actor},
		},
		Ops: []FileOp{{Type: OpWrite, Path: config.FileName, Content: doc.Bytes()}},
	})
	result.Metadata = map[string]any{"validator": name}
	return result, nil
}

// validatorEqual compares two validators field-for-field with
// args-slice equality. Used for idempotency on RecipeInstall.
func validatorEqual(a, b aiwfyaml.Validator) bool {
	if a.Command != b.Command {
		return false
	}
	if len(a.Args) != len(b.Args) {
		return false
	}
	for i := range a.Args {
		if a.Args[i] != b.Args[i] {
			return false
		}
	}
	return true
}

// bindingsReferencing returns the sorted list of contract ids in
// current.Entries that name `validator` as their validator. nil
// inputs return nil. Emitted ids are canonicalized per AC-3 in M-081
// so the trailers / error messages downstream are uniform width.
func bindingsReferencing(current *aiwfyaml.Contracts, validator string) []string {
	if current == nil {
		return nil
	}
	var out []string
	for _, e := range current.Entries {
		if e.Validator == validator {
			out = append(out, entity.Canonicalize(e.ID))
		}
	}
	sort.Strings(out)
	return out
}

// joinIDs renders a sorted comma-separated list for error messages.
func joinIDs(ids []string) string {
	switch len(ids) {
	case 0:
		return ""
	case 1:
		return ids[0]
	}
	out := ids[0]
	for _, id := range ids[1:] {
		out += ", " + id
	}
	return out
}
