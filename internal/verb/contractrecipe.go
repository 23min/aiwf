package verb

import (
	"context"
	"fmt"
	"sort"

	"github.com/23min/ai-workflow-v2/internal/aiwfyaml"
	"github.com/23min/ai-workflow-v2/internal/config"
	"github.com/23min/ai-workflow-v2/internal/entity"
	"github.com/23min/ai-workflow-v2/internal/gitops"
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
func RecipeInstall(ctx context.Context, doc *aiwfyaml.Doc, current *aiwfyaml.Contracts, name string, validator aiwfyaml.Validator, actor string, opts RecipeInstallOptions) (*Result, error) {
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
	if err := doc.SetContracts(next); err != nil {
		return nil, fmt.Errorf("updating aiwf.yaml: %w", err)
	}

	trailers := []gitops.Trailer{
		{Key: gitops.TrailerVerb, Value: "recipe-install"},
		{Key: gitops.TrailerActor, Value: actor},
	}
	for _, id := range bindingsReferencing(current, name) {
		trailers = append(trailers, gitops.Trailer{Key: gitops.TrailerEntity, Value: id})
	}

	return plan(&Plan{
		Subject:  fmt.Sprintf("aiwf contract recipe install %s", name),
		Trailers: trailers,
		Ops:      []FileOp{{Type: OpWrite, Path: config.FileName, Content: doc.Bytes()}},
	}), nil
}

// RecipeRemove removes the named validator from
// aiwf.yaml.contracts.validators. Errors when one or more bindings
// in entries[] still reference the validator — the user must
// `unbind` or rebind those contracts first.
func RecipeRemove(ctx context.Context, doc *aiwfyaml.Doc, current *aiwfyaml.Contracts, name, actor string) (*Result, error) {
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
	if err := doc.SetContracts(next); err != nil {
		return nil, fmt.Errorf("updating aiwf.yaml: %w", err)
	}
	return plan(&Plan{
		Subject: fmt.Sprintf("aiwf contract recipe remove %s", name),
		Trailers: []gitops.Trailer{
			{Key: gitops.TrailerVerb, Value: "recipe-remove"},
			{Key: gitops.TrailerActor, Value: actor},
		},
		Ops: []FileOp{{Type: OpWrite, Path: config.FileName, Content: doc.Bytes()}},
	}), nil
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
