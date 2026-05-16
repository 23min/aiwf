// Command aiwf is the ai-workflow framework's single binary.
//
// Dispatch is built on github.com/spf13/cobra: every verb, subverb,
// flag, and closed-set value is exposed to shell tab-completion. The
// command tree is assembled by newRootCmd; the drift test in
// completion_drift_test.go is the chokepoint that fails CI when a
// flag lands without completion wiring or an opt-out entry.
package main

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"

	"github.com/spf13/cobra"

	"github.com/23min/aiwf/internal/check"
	"github.com/23min/aiwf/internal/cli/cliutil"
	"github.com/23min/aiwf/internal/config"
	"github.com/23min/aiwf/internal/entity"
	"github.com/23min/aiwf/internal/render"
	"github.com/23min/aiwf/internal/tree"
	"github.com/23min/aiwf/internal/version"
)

// Version is the binary's reported version. The ldflags-stamped value
// takes precedence (used by `make install` for branch+SHA stamping
// during local development); when left at the default `"dev"`, the
// `aiwf version` verb falls back to runtime/debug.ReadBuildInfo via
// version.Current() so a `go install …@v0.1.0` binary correctly
// reports its tag.
var Version = "dev"

// resolvedVersion returns the version to display in user output.
// Prefers the ldflags-stamped Version global when set to anything
// other than the default sentinel, otherwise defers to buildinfo via
// version.Current. The two paths surface different conventions for
// "no version known" (Version="dev" vs DevelVersion="(devel)"); we
// normalize by always returning the buildinfo-style value when no
// ldflags stamp is present, so `aiwf version` and `aiwf doctor`'s
// binary: row stay byte-coherent for the same binary.
func resolvedVersion() string {
	if Version != "dev" && Version != "" {
		return Version
	}
	return version.Current().Version
}

// registerFormatCompletion wires `--format=` shell completion to the
// closed set {text, json}. Called by every read-only verb that
// accepts --format so the shell-completion experience is uniform
// across the surface (E-14's auto-completion-friendliness rule).
func registerFormatCompletion(cmd *cobra.Command) {
	_ = cmd.RegisterFlagCompletionFunc("format", cobra.FixedCompletions(
		[]string{"text", "json"},
		cobra.ShellCompDirectiveNoFileComp,
	))
}

// allKindNames returns the entity-kind names as strings, in the
// canonical iteration order from entity.AllKinds(). Used by the
// `aiwf add` and `aiwf schema` / `aiwf template` completion functions.
func allKindNames() []string {
	all := entity.AllKinds()
	names := make([]string, len(all))
	for i, k := range all {
		names[i] = string(k)
	}
	return names
}

// statusesForID returns the closed set of statuses that an entity's
// kind allows, derived from the id's prefix without loading the
// repo's tree. Used as the static-completion source for `aiwf promote
// <id> <new-status>`. Returns nil for ids whose kind isn't recognized
// (composite ids, malformed input) — the completion source then falls
// back to file completion at the shell level.
func statusesForID(id string) []string {
	if id == "" || entity.IsCompositeID(id) {
		return nil
	}
	k, ok := entity.KindFromID(id)
	if !ok {
		return nil
	}
	return entity.AllowedStatuses(k)
}

// completeEntityIDs returns the live ids in the consumer repo's
// planning tree, optionally filtered to a single kind. Designed for
// use as a Cobra ValidArgsFunction or RegisterFlagCompletionFunc body:
// failures (no aiwf.yaml, malformed tree, unreadable disk) collapse
// to an empty list rather than spamming the user's shell with errors,
// satisfying M-054 AC-2's graceful-no-op rule.
func completeEntityIDs(filter entity.Kind) ([]string, cobra.ShellCompDirective) {
	rootDir, err := resolveRoot("")
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	tr, _, err := tree.Load(context.Background(), rootDir)
	if err != nil || tr == nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	ids := make([]string, 0, len(tr.Entities))
	for _, e := range tr.Entities {
		if filter != "" && e.Kind != filter {
			continue
		}
		// Emit canonical ids so completion always offers the canonical
		// width, regardless of on-disk filename width (AC-3 in M-081).
		// Inputs at narrow width are still accepted everywhere
		// downstream via tree.ByID's lookup-side canonicalization.
		ids = append(ids, entity.Canonicalize(e.ID))
	}
	return ids, cobra.ShellCompDirectiveNoFileComp
}

// completeEntityIDFlag is the standard Cobra flag-completion adapter
// over completeEntityIDs. Callers wire it via
// `cmd.RegisterFlagCompletionFunc(name, completeEntityIDFlag(kind))`
// where kind is either "" for all kinds or a specific entity.Kind.
func completeEntityIDFlag(filter entity.Kind) func(*cobra.Command, []string, string) ([]string, cobra.ShellCompDirective) {
	return func(_ *cobra.Command, _ []string, _ string) ([]string, cobra.ShellCompDirective) {
		return completeEntityIDs(filter)
	}
}

// completeEntityIDArg is the standard Cobra positional-arg completion
// adapter over completeEntityIDs. Callers assign it as a command's
// ValidArgsFunction. Unlike the flag adapter, this version respects
// the args slice — if the positional in question isn't the first one,
// it returns no suggestions (so e.g. `aiwf promote E-01 <TAB>` doesn't
// re-suggest entity ids when the second positional is the new-status).
func completeEntityIDArg(filter entity.Kind, position int) func(*cobra.Command, []string, string) ([]string, cobra.ShellCompDirective) {
	return func(_ *cobra.Command, args []string, _ string) ([]string, cobra.ShellCompDirective) {
		if len(args) != position {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		return completeEntityIDs(filter)
	}
}

func main() {
	if err := cliutil.AssertSupportedOS(runtime.GOOS); err != nil {
		fmt.Fprintln(os.Stderr, "aiwf:", err)
		os.Exit(cliutil.ExitUsage)
	}
	os.Exit(run(os.Args[1:]))
}

// run dispatches one CLI invocation through the Cobra root command.
// Args here are the args after the binary name (i.e., os.Args[1:]).
// The command tree is built fresh per call so tests can drive run() in
// parallel without any shared mutable state.
func run(args []string) int {
	rootCmd := newRootCmd()
	rootCmd.SetArgs(args)

	err := rootCmd.Execute()
	if err == nil {
		return cliutil.ExitOK
	}
	var ee *cliutil.ExitError
	if errors.As(err, &ee) {
		return ee.Code
	}
	// Non-cliutil.ExitError means Cobra surfaced a usage problem (unknown verb,
	// bad flag, missing required arg). With SilenceErrors:true on the
	// root, Cobra didn't print; we print here in the existing house style.
	fmt.Fprintf(os.Stderr, "aiwf: %v\n", err)
	return cliutil.ExitUsage
}

// newRootCmd assembles the Cobra command tree. Every verb is a
// native Cobra command (E-14 left no passthrough adapters); each
// verb's flags are tab-completable per the drift-prevention rule
// in completion_drift_test.go.
func newRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:           "aiwf",
		Short:         "ai-workflow framework CLI",
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(c *cobra.Command, args []string) error {
			if v, _ := c.Flags().GetBool("version"); v {
				fmt.Println(resolvedVersion())
				return nil
			}
			fmt.Fprintln(os.Stderr, "aiwf: missing verb. Try 'aiwf help'.")
			return cliutil.WrapExitCode(cliutil.ExitUsage)
		},
	}
	// Manual --version/-v registration (rather than cmd.Version) lets
	// us bind both the long form and the -v shorthand without relying
	// on Cobra's auto-add timing — the auto-flag is added during
	// Execute, after construction, so its Shorthand can't be set here.
	cmd.Flags().BoolP("version", "v", false, "print version and exit")

	// Until subsequent milestones populate per-verb metadata, the
	// hand-curated printHelp() text continues to be authoritative for
	// `aiwf`, `aiwf help`, `aiwf -h`, `aiwf --help`. Subverb help still
	// flows through the legacy handler via the passthrough adapter
	// (DisableFlagParsing leaves --help in args).
	cmd.SetHelpFunc(func(c *cobra.Command, _ []string) {
		if c == cmd {
			printHelp()
			return
		}
		// Non-root descendants render Cobra's standard usage block.
		// We can't call c.Help() here because SetHelpFunc on root is
		// inherited by every descendant — c.Help() would re-enter
		// this function and recurse until stack overflow.
		out := c.OutOrStderr()
		switch {
		case c.Long != "":
			_, _ = fmt.Fprintln(out, c.Long)
			_, _ = fmt.Fprintln(out)
		case c.Short != "":
			_, _ = fmt.Fprintln(out, c.Short)
			_, _ = fmt.Fprintln(out)
		}
		_, _ = fmt.Fprint(out, c.UsageString())
	})

	cmd.AddCommand(newVersionCmd())

	cmd.AddCommand(newCheckCmd())
	cmd.AddCommand(newAddCmd())
	cmd.AddCommand(newPromoteCmd())
	cmd.AddCommand(newCancelCmd())
	cmd.AddCommand(newRenameCmd())
	cmd.AddCommand(newRetitleCmd())
	cmd.AddCommand(newEditBodyCmd())
	cmd.AddCommand(newMoveCmd())
	cmd.AddCommand(newReallocateCmd())
	cmd.AddCommand(newRewidthCmd())
	cmd.AddCommand(newArchiveCmd())
	cmd.AddCommand(newInitCmd())
	cmd.AddCommand(newUpdateCmd())
	cmd.AddCommand(newUpgradeCmd())
	cmd.AddCommand(newHistoryCmd())
	cmd.AddCommand(newDoctorCmd())
	cmd.AddCommand(newRenderCmd())
	cmd.AddCommand(newImportCmd())
	cmd.AddCommand(newWhoamiCmd())
	cmd.AddCommand(newStatusCmd())
	cmd.AddCommand(newListCmd())
	cmd.AddCommand(newSchemaCmd())
	cmd.AddCommand(newShowCmd())
	cmd.AddCommand(newTemplateCmd())
	cmd.AddCommand(newContractCmd())
	cmd.AddCommand(newMilestoneCmd())
	cmd.AddCommand(newAuthorizeCmd())

	return cmd
}

// newVersionCmd is the M-049 reference shape: a native Cobra command
// whose RunE writes a single-line version string to stdout. It must
// stay byte-coherent with `aiwf -v` / `aiwf --version` (both backed by
// resolvedVersion via the root RunE).
func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print the binary version",
		Example: `  # Print the installed binary's version
  aiwf version`,
		Args:          cobra.NoArgs,
		SilenceErrors: true,
		SilenceUsage:  true,
		RunE: func(c *cobra.Command, args []string) error {
			fmt.Println(resolvedVersion())
			return nil
		},
	}
}

func printHelp() {
	fmt.Println(`aiwf — ai-workflow framework CLI

Usage: aiwf <verb> [args]

Verbs:
  check                          validate the consumer repo's planning state; with aiwf.yaml tdd.require_test_metrics=true (default false), warns on ACs at tdd_phase=done whose history carries no aiwf-tests trailer
  add <kind> --title "..."       create a new entity of the given kind
  promote <id> <new-status>      advance an entity's status (optional --reason "..."; --force --reason "..." to skip the FSM); composite ids (M-NNN/AC-N) accepted; --phase <p> for AC tdd_phase (mutex with positional new-status); --tests "pass=N fail=N skip=N [total=N]" attaches an aiwf-tests trailer in phase mode (recognized keys only; non-negative integers)
  cancel <id>                    promote to the kind's terminal-cancel status (optional --reason "..."; --force --reason "..." records the cancellation as an audit event)
  rename <id> <new-slug>         rename the file/dir slug; id preserved
  edit-body <id> [--body-file <p>] replace the entity's markdown body (frontmatter untouched); omit --body-file to bless current working-copy edits, or use --body-file - for stdin; --reason "..." optional
  move <M-id> --epic <E-id>      move a milestone to a different epic; id preserved
  reallocate <id-or-path>        renumber the entity; rewrite refs in others
  authorize <id> --to <agent>    open an autonomous-work scope on <id> for <agent>; --pause "<reason>" / --resume "<reason>" cycle the scope; human-only verb
  init                           one-time setup: aiwf.yaml, scaffolding, skills, pre-push hook
  update                         re-materialize embedded skills into .claude/skills/aiwf-*/
  upgrade [--version vX.Y.Z]     fetch a newer (or specified) aiwf binary via 'go install' and re-exec into 'aiwf update' (default: latest)
  history <id>                   show the entity's lifecycle from git log trailers
  doctor [--self-check] [--check-latest]  drift / version / id-collision health check; --self-check drives every verb against a temp repo; --check-latest hits the Go module proxy for the latest published aiwf version (advisory)
  render roadmap [--write]       print ROADMAP.md (markdown of epics + milestones); --write commits it
  render --format=html [--out <dir>] [--pretty]  render the static-site governance page (index + per-epic + per-milestone HTML) under aiwf.yaml.html.out_dir (default 'site') or --out; emits a JSON envelope on stdout with out_dir/files_written/elapsed_ms
  import <manifest>              bulk-create entities from a YAML/JSON manifest (one commit by default)
  whoami                         print the resolved actor and the source it came from
  status                         project snapshot: in-flight work, open decisions, gaps, recent activity
  show <id>                      aggregate view: frontmatter + acs + recent history + active findings + referenced_by (the ids of entities that name this one as a reference target); JSON also carries body (map of section-heading slug to prose: epic goal/scope/out_of_scope; milestone goal/acceptance_criteria; adr context/decision/consequences; gap what_s_missing/why_it_matters; decision question/decision/reasoning; contract purpose/stability) and per-AC description (the AC-N body section) on milestones; history events carry tests {pass,fail,skip,total} when the commit had an aiwf-tests trailer; composite ids (M-NNN/AC-N) accepted
  schema [kind]                  print the frontmatter contract for one kind (or all six); read-only
  template [kind]                print the body-section template 'aiwf add' would scaffold for the kind; read-only
  contract verify                run the verify and evolve passes for every contract binding in aiwf.yaml
  contract bind <C-id>           add or replace a binding in aiwf.yaml (--validator, --schema, --fixtures; --force to replace)
  contract unbind <C-id>         remove a binding from aiwf.yaml (entity status untouched)
  contract recipes               list embedded validator recipes and currently declared validators
  contract recipe show <name>    print an embedded recipe's markdown
  contract recipe install <name|--from <path>> [--force]  install a validator from the embedded set or from a YAML file
  contract recipe remove <name>  remove a declared validator (errors when bindings still reference it)
  completion <bash|zsh|fish|powershell>  emit a sourceable shell-completion script (kubectl/gh idiom)
  help, --help                   show this message
  version, --version             print the binary version

Common flags:
  --root <path>                  consumer repo root (default: walk up looking for aiwf.yaml, else cwd)
  --actor <role>/<identifier>    actor for the commit trailer (default: derived from git config user.email)
  --principal human/<id>         the human accountable for the act; required when --actor is non-human (ai/..., bot/...), forbidden when --actor is human/...

Provenance:
  When the operator is non-human, --principal must be supplied; the kernel stamps aiwf-principal: on the commit. To delegate autonomous work, run 'aiwf authorize <id> --to <agent>' first; subsequent agent verbs match the active scope and the kernel adds aiwf-on-behalf-of: + aiwf-authorized-by: trailers automatically. See the aiwf-authorize skill or docs/pocv3/design/provenance-model.md.

Flags for 'add':
  --epic <id>                    parent epic id (milestone)
  --depends-on <id,id,...>       milestones the new milestone depends on (milestone)
  --discovered-in <id>           discovery context (gap)
  --relates-to <id,id,...>       related entities (decision)
  --linked-adr <id,id,...>       ADRs motivating the contract (contract)
  --validator <name>             validator name to bind (contract; with --schema, --fixtures: atomic add+bind)
  --schema <path>                schema path (contract; pairs with --validator and --fixtures)
  --fixtures <path>              fixtures-tree root (contract; pairs with --validator and --schema)
  --tests "pass=N fail=N ..."    test metrics for the seeded red phase (ac; only when parent milestone is tdd: required); recognized keys: pass, fail, skip, total; non-negative integers

Flags for 'check', 'history', and 'contract verify':
  --format <fmt>                 output format: text (default) or json
  --pretty                       indent JSON output (only with --format=json)

Flags for 'history':
  --show-authorization           include the full aiwf-authorized-by SHA on scope-authorized rows (text format)

Flags for 'promote' and 'cancel':
  --audit-only --reason "..."    backfill an audit trail when state was reached via a manual commit; verb writes an empty-diff commit carrying aiwf-audit-only:; entity must already be at the target state (no FSM transition); mutually exclusive with --force; human-only

Flags for 'authorize':
  --to <agent>                   open scope (e.g. ai/claude); refused on terminal scope-entity unless --force --reason
  --pause "<reason>"             pause the most-recently-opened active scope on <id>
  --resume "<reason>"            resume the most-recently-paused scope on <id>

Flags for 'import':
  --on-collision <mode>          fail (default) | skip | update — behavior when an explicit id already exists
  --dry-run                      validate the projection and print the would-be plans without writing

Flags for 'upgrade':
  --version <semver|latest>      version to install (default: latest); a 'v'-prefixed semver tag pins to a specific release
  --check                        print the current/target comparison and exit; does not invoke 'go install'
  --root <path>                  consumer repo root for the post-install 'aiwf update' step (default: cwd)

Exit codes: 0 = no errors, 1 = errors found, 2 = usage error, 3 = internal error.

Docs: docs/pocv3/archive/poc-plan-pre-migration.md and docs/pocv3/design/design-decisions.md.`)
}

// newCheckCmd builds `aiwf check`: validate the consumer repo's
// planning state. Read-only; produces no commit. The pre-push git hook
// runs this verb — its findings + exit code are the framework's
// authoritative correctness gate.
func newCheckCmd() *cobra.Command {
	var (
		root      string
		format    string
		pretty    bool
		since     string
		shapeOnly bool
		verbose   bool
	)
	cmd := &cobra.Command{
		Use:   "check",
		Short: "Validate the consumer repo's planning state",
		Example: `  # Default: errors per-instance, warnings collapsed to a per-code summary
  aiwf check

  # Restore the full per-instance shape (one line per finding) for warnings too
  aiwf check --verbose

  # Emit a JSON envelope for CI scripts (always per-instance regardless of --verbose)
  aiwf check --format=json --pretty`,
		Args:          cobra.NoArgs,
		SilenceErrors: true,
		SilenceUsage:  true,
		RunE: func(c *cobra.Command, args []string) error {
			return cliutil.WrapExitCode(runCheckCmd(root, format, pretty, since, shapeOnly, verbose))
		},
	}
	cmd.Flags().StringVar(&root, "root", "", "consumer repo root (default: discover via aiwf.yaml)")
	cmd.Flags().StringVar(&format, "format", "text", "output format: text or json")
	cmd.Flags().BoolVar(&pretty, "pretty", false, "indent JSON output (only with --format=json)")
	cmd.Flags().StringVar(&since, "since", "", "explicit base ref for the provenance untrailered-entity audit (default: @{u} when set, else skipped)")
	cmd.Flags().BoolVar(&shapeOnly, "shape-only", false, "run only the tree-discipline rule (skips trunk read, provenance audit, contract validation); used by the pre-commit hook for a fast LLM-loop check")
	cmd.Flags().BoolVar(&verbose, "verbose", false, "print one line per warning instance instead of the per-code summary; errors are always per-instance regardless")
	registerFormatCompletion(cmd)
	return cmd
}

func runCheckCmd(root, format string, pretty bool, since string, shapeOnly, verbose bool) int {
	if format != "text" && format != "json" {
		fmt.Fprintf(os.Stderr, "aiwf check: --format must be 'text' or 'json', got %q\n", format)
		return cliutil.ExitUsage
	}
	if pretty && format != "json" {
		fmt.Fprintln(os.Stderr, "aiwf check: --pretty has no effect without --format=json")
	}

	resolved, err := resolveRoot(root)
	if err != nil {
		fmt.Fprintf(os.Stderr, "aiwf check: %v\n", err)
		return cliutil.ExitUsage
	}

	ctx := context.Background()
	if shapeOnly {
		return runCheckShapeOnly(ctx, resolved, format, pretty)
	}

	tr, loadErrs, err := cliutil.LoadTreeWithTrunk(ctx, resolved)
	if err != nil {
		fmt.Fprintf(os.Stderr, "aiwf check: loading tree: %v\n", err)
		return cliutil.ExitInternal
	}

	findings := check.Run(tr, loadErrs)

	contracts, contractErr := loadContractsBlock(resolved)
	if contractErr != nil {
		fmt.Fprintf(os.Stderr, "aiwf check: %v\n", contractErr)
		return cliutil.ExitInternal
	}
	contractFindings := runContractValidation(ctx, tr, resolved, contracts)
	findings = append(findings, contractFindings...)

	provenanceFindings, pErr := runProvenanceCheck(ctx, resolved, tr, since)
	if pErr != nil {
		fmt.Fprintf(os.Stderr, "aiwf check: %v\n", pErr)
		return cliutil.ExitInternal
	}
	findings = append(findings, provenanceFindings...)

	requireMetrics := false
	var treeAllow []string
	treeStrict := false
	tddStrict := false
	archiveThreshold := 0
	archiveThresholdSet := false
	if cfg, cfgErr := config.Load(resolved); cfgErr == nil && cfg != nil {
		requireMetrics = cfg.TDD.RequireTestMetrics
		treeAllow = cfg.Tree.AllowPaths
		treeStrict = cfg.Tree.Strict
		tddStrict = cfg.TDD.Strict
		archiveThreshold, archiveThresholdSet = cfg.ArchiveSweepThreshold()
	}
	metricsFindings, mErr := runTestsMetricsCheck(ctx, resolved, tr, requireMetrics)
	if mErr != nil {
		fmt.Fprintf(os.Stderr, "aiwf check: %v\n", mErr)
		return cliutil.ExitInternal
	}
	findings = append(findings, metricsFindings...)

	findings = append(findings, check.TreeDiscipline(tr, treeAllow, treeStrict)...)

	// M-066/AC-2: aiwf.yaml: tdd.strict bumps entity-body-empty
	// (and any future TDD-strict-covered finding) from warning to
	// error so the pre-push hook blocks the push.
	check.ApplyTDDStrict(findings, tddStrict)

	// M-0088/AC-2: aiwf.yaml: archive.sweep_threshold bumps the
	// aggregate `archive-sweep-pending` finding from warning to
	// error when the pending-sweep count exceeds the consumer's
	// declared ceiling. The count is the same value the rule's
	// Message already names — computed once via CountPendingSweep
	// so the bumper does not re-iterate the tree.
	check.ApplyArchiveSweepThreshold(findings, archiveThreshold, archiveThresholdSet, check.CountPendingSweep(tr))

	applyHintsLikeRun(findings)
	check.SortFindings(findings)

	switch format {
	case "text":
		// M-0089 AC-1/AC-2/AC-3: default text mode collapses warnings
		// into a per-code summary while keeping errors per-instance;
		// --verbose restores the full per-instance shape (byte-for-byte
		// identical to the pre-M-0089 output). JSON is never affected
		// (AC-4).
		writeText := render.TextSummary
		if verbose {
			writeText = render.Text
		}
		if err := writeText(os.Stdout, findings); err != nil {
			fmt.Fprintf(os.Stderr, "aiwf check: writing output: %v\n", err)
			return cliutil.ExitInternal
		}
	case "json":
		env := render.Envelope{
			Tool:     "aiwf",
			Version:  Version,
			Status:   render.StatusFor(findings),
			Findings: findings,
			Metadata: map[string]any{
				"root":     resolved,
				"entities": len(tr.Entities),
				"bindings": bindingCount(contracts),
				"findings": len(findings),
			},
		}
		if err := render.JSON(os.Stdout, env, pretty); err != nil {
			fmt.Fprintf(os.Stderr, "aiwf check: writing output: %v\n", err)
			return cliutil.ExitInternal
		}
	}

	if check.HasErrors(findings) {
		return cliutil.ExitFindings
	}
	return cliutil.ExitOK
}

// runCheckShapeOnly runs the tree-discipline rule and nothing else.
// Used by the pre-commit hook to give the LLM a fast, in-loop signal
// when a stray file lands under work/ — the full check.Run pipeline
// (trunk read, provenance walk, contract validation) is too slow and
// too noisy to fire on every commit, but the tree-discipline rule is
// cheap and exact. Honors `aiwf.yaml: tree.{allow_paths,strict}` the
// same way the full check does.
//
// Exit codes match `aiwf check`'s contract: 0 ok, 1 findings (errors
// present — only fires when tree.strict: true), 3 internal.
func runCheckShapeOnly(ctx context.Context, root, format string, pretty bool) int {
	tr, _, err := tree.Load(ctx, root)
	if err != nil {
		fmt.Fprintf(os.Stderr, "aiwf check: loading tree: %v\n", err)
		return cliutil.ExitInternal
	}
	var allow []string
	strict := false
	if cfg, cfgErr := config.Load(root); cfgErr == nil && cfg != nil {
		allow = cfg.Tree.AllowPaths
		strict = cfg.Tree.Strict
	}
	findings := check.TreeDiscipline(tr, allow, strict)
	applyHintsLikeRun(findings)
	check.SortFindings(findings)

	switch format {
	case "text":
		if err := render.Text(os.Stdout, findings); err != nil {
			fmt.Fprintf(os.Stderr, "aiwf check: writing output: %v\n", err)
			return cliutil.ExitInternal
		}
	case "json":
		env := render.Envelope{
			Tool:     "aiwf",
			Version:  Version,
			Status:   render.StatusFor(findings),
			Findings: findings,
			Metadata: map[string]any{
				"root":       root,
				"entities":   len(tr.Entities),
				"shape_only": true,
				"findings":   len(findings),
			},
		}
		if err := render.JSON(os.Stdout, env, pretty); err != nil {
			fmt.Fprintf(os.Stderr, "aiwf check: writing output: %v\n", err)
			return cliutil.ExitInternal
		}
	}

	if check.HasErrors(findings) {
		return cliutil.ExitFindings
	}
	return cliutil.ExitOK
}

// resolveRoot picks the consumer repo root. If explicit is non-empty,
// it is used as-is (resolved to absolute). Otherwise, walks up from cwd
// looking for aiwf.yaml; if found, uses its parent. If not found, falls
// back to cwd (lenient pre-init behavior; tightens once `aiwf init` is
// part of the standard adoption path in Session 3).
func resolveRoot(explicit string) (string, error) {
	if explicit != "" {
		abs, err := filepath.Abs(explicit)
		if err != nil {
			return "", fmt.Errorf("resolving --root: %w", err)
		}
		return abs, nil
	}
	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("getting cwd: %w", err)
	}
	if found, ok := walkUpFor(cwd, "aiwf.yaml"); ok {
		return found, nil
	}
	return cwd, nil
}

// walkUpFor walks from start toward root looking for filename.
// Returns the directory containing filename (not the filename itself),
// and true if found.
func walkUpFor(start, filename string) (string, bool) {
	dir := start
	for {
		candidate := filepath.Join(dir, filename)
		info, err := os.Stat(candidate)
		if err == nil && !info.IsDir() {
			return dir, true
		}
		if err != nil && !errors.Is(err, fs.ErrNotExist) {
			return "", false
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", false
		}
		dir = parent
	}
}
