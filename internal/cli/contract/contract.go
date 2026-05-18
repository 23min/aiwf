// Package contract implements the `aiwf contract` verb and its
// subcommand graph (verify, bind, unbind, recipes, recipe show /
// install / remove). The parent verb is non-Runnable — `aiwf
// contract` with no subcommand prints help.
//
// RunValidation, ApplyHintsLikeRun, BindingCount, and ResultToFinding
// are exported because the `check` verb (in cmd/aiwf/main.go until
// M-0118 moves it) calls them to run contract validation as part of
// `aiwf check`.
package contract

import (
	"context"
	"sort"

	"github.com/spf13/cobra"

	"github.com/23min/aiwf/internal/aiwfyaml"
	"github.com/23min/aiwf/internal/check"
	"github.com/23min/aiwf/internal/cli/cliutil"
	"github.com/23min/aiwf/internal/contractcheck"
	"github.com/23min/aiwf/internal/contractverify"
	"github.com/23min/aiwf/internal/entity"
	"github.com/23min/aiwf/internal/recipe"
	"github.com/23min/aiwf/internal/tree"
)

// NewCmd builds the `aiwf contract` parent command. Five direct
// children (verify, bind, unbind, recipes, recipe) plus the recipe
// sub-tree (show, install, remove). The parent itself is non-Runnable
// — `aiwf contract` with no subcommand prints help.
func NewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:           "contract",
		Short:         "Manage contract entities and their validators",
		Args:          cobra.NoArgs,
		SilenceErrors: true,
		SilenceUsage:  true,
	}
	cmd.AddCommand(newVerifyCmd())
	cmd.AddCommand(newBindCmd())
	cmd.AddCommand(newUnbindCmd())
	cmd.AddCommand(newRecipesCmd())
	cmd.AddCommand(newRecipeCmd())
	return cmd
}

// RunValidation is the shared entry point for both the CLI
// `aiwf contract verify` and the pre-push integration in `aiwf check`.
// It runs contractcheck (config correspondence) plus contractverify
// (subprocess validators) and returns the combined findings slice
// (un-sorted, hints not applied — caller composes).
//
// A nil contracts argument is treated as "no contracts configured":
// the function returns nil. Terminal-state contract entities
// (rejected, retired) are excluded from verification.
func RunValidation(ctx context.Context, tr *tree.Tree, rootDir string, contracts *aiwfyaml.Contracts) []check.Finding {
	if contracts == nil {
		return nil
	}
	configFindings := contractcheck.Run(tr, contracts, rootDir)

	skip := make(map[string]bool)
	for _, e := range tr.ByKind(entity.KindContract) {
		if e.Status == entity.StatusRejected || e.Status == entity.StatusRetired {
			skip[entity.Canonicalize(e.ID)] = true
		}
	}
	verifyResults := contractverify.Run(ctx, contractverify.Options{
		RepoRoot:  rootDir,
		Contracts: contracts,
		SkipIDs:   skip,
	})

	out := append([]check.Finding(nil), configFindings...)
	for _, r := range verifyResults {
		out = append(out, ResultToFinding(r, contracts.StrictValidators))
	}
	return out
}

// ApplyHintsLikeRun fills the Hint field on every finding from the
// shared hint table. Mirrors the post-processing check.Run does for
// entity-level findings; we inline it here because we don't go
// through check.Run for contract verify.
func ApplyHintsLikeRun(findings []check.Finding) {
	for i := range findings {
		f := &findings[i]
		if f.Hint != "" {
			continue
		}
		f.Hint = check.HintFor(f.Code, f.Subcode)
	}
}

// BindingCount is a small helper for the JSON envelope's metadata.
func BindingCount(c *aiwfyaml.Contracts) int {
	if c == nil {
		return 0
	}
	return len(c.Entries)
}

// ResultToFinding converts a contractverify.Result into the Finding
// shape the render layer expects. Most codes are errors; the
// per-machine `validator-unavailable` code is a warning by default,
// upgraded to an error by strictValidators.
func ResultToFinding(r contractverify.Result, strictValidators bool) check.Finding {
	severity := check.SeverityError
	code := r.Code
	subcode := ""
	if r.Code == contractverify.CodeValidatorUnavailable {
		code = "contract-config"
		subcode = "validator-unavailable"
		if !strictValidators {
			severity = check.SeverityWarning
		}
	}
	return check.Finding{
		Code:     code,
		Severity: severity,
		Subcode:  subcode,
		Message:  r.Message,
		Path:     r.FixturePath,
		EntityID: r.EntityID,
	}
}

// completeEmbeddedRecipeNamesArg is a Cobra ValidArgsFunction that
// returns the names of every embedded recipe. Used by `recipe show`
// and `recipe install`. Failures collapse to the empty list per the
// M-054 graceful-no-op rule.
func completeEmbeddedRecipeNamesArg(_ *cobra.Command, args []string, _ string) ([]string, cobra.ShellCompDirective) {
	if len(args) > 0 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	rs, err := recipe.List()
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	names := make([]string, 0, len(rs))
	for _, r := range rs {
		names = append(names, r.Name)
	}
	return names, cobra.ShellCompDirectiveNoFileComp
}

// completeDeclaredValidatorsArg returns validator names currently
// declared under aiwf.yaml.contracts.validators. Used by `recipe
// remove`. Empty when aiwf.yaml is absent or the block is empty.
func completeDeclaredValidatorsArg(_ *cobra.Command, args []string, _ string) ([]string, cobra.ShellCompDirective) {
	if len(args) > 0 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	return declaredValidatorNames(), cobra.ShellCompDirectiveNoFileComp
}

// completeDeclaredValidators is the flag-side adapter for `contract
// bind --validator <TAB>` so the user picks from the already-declared
// set rather than free-typing.
func completeDeclaredValidators(_ *cobra.Command, _ []string, _ string) ([]string, cobra.ShellCompDirective) {
	return declaredValidatorNames(), cobra.ShellCompDirectiveNoFileComp
}

func declaredValidatorNames() []string {
	rootDir, err := cliutil.ResolveRoot("")
	if err != nil {
		return nil
	}
	contracts, err := cliutil.LoadContractsBlock(rootDir)
	if err != nil || contracts == nil || len(contracts.Validators) == 0 {
		return nil
	}
	names := make([]string, 0, len(contracts.Validators))
	for n := range contracts.Validators {
		names = append(names, n)
	}
	sort.Strings(names)
	return names
}
