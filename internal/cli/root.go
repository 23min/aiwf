// Package cli is the aiwf binary's root-command assembly. cmd/aiwf/main.go
// is entry-only (per G-0107): main() calls Execute(os.Args[1:]) and exits
// with the returned code. Every verb is a Cobra subcommand registered
// from NewRootCmd; the verb bodies live under internal/cli/<verb>/.
//
// Why this package exists separately from cmd/aiwf/: the cmd-package
// can only host the binary entry point. Other packages (notably
// internal/cli/doctor for --self-check) must be able to drive the
// dispatcher in-process, which is impossible across the cmd/aiwf
// boundary (no one can import a main package). The doctor.Dispatcher
// seam (wired by this package's init) is the bridge.
package cli

import (
	"errors"
	"fmt"
	"os"
	"runtime"
	"strings"

	"github.com/spf13/cobra"

	"github.com/23min/aiwf/internal/cli/acknowledge"
	"github.com/23min/aiwf/internal/cli/add"
	"github.com/23min/aiwf/internal/cli/archive"
	"github.com/23min/aiwf/internal/cli/authorize"
	"github.com/23min/aiwf/internal/cli/cancel"
	"github.com/23min/aiwf/internal/cli/check"
	"github.com/23min/aiwf/internal/cli/cliutil"
	"github.com/23min/aiwf/internal/cli/contract"
	"github.com/23min/aiwf/internal/cli/doctor"
	"github.com/23min/aiwf/internal/cli/editbody"
	"github.com/23min/aiwf/internal/cli/history"
	"github.com/23min/aiwf/internal/cli/importcmd"
	"github.com/23min/aiwf/internal/cli/initcmd"
	"github.com/23min/aiwf/internal/cli/list"
	"github.com/23min/aiwf/internal/cli/milestone"
	"github.com/23min/aiwf/internal/cli/move"
	"github.com/23min/aiwf/internal/cli/promote"
	"github.com/23min/aiwf/internal/cli/reallocate"
	"github.com/23min/aiwf/internal/cli/rename"
	"github.com/23min/aiwf/internal/cli/renamearea"
	"github.com/23min/aiwf/internal/cli/render"
	"github.com/23min/aiwf/internal/cli/retitle"
	"github.com/23min/aiwf/internal/cli/rewidth"
	"github.com/23min/aiwf/internal/cli/schema"
	"github.com/23min/aiwf/internal/cli/setarea"
	"github.com/23min/aiwf/internal/cli/show"
	"github.com/23min/aiwf/internal/cli/status"
	"github.com/23min/aiwf/internal/cli/template"
	"github.com/23min/aiwf/internal/cli/update"
	"github.com/23min/aiwf/internal/cli/upgrade"
	"github.com/23min/aiwf/internal/cli/whoami"
	"github.com/23min/aiwf/internal/version"
)

// init wires the doctor package's in-process Dispatcher seam to
// Execute. The doctor package's --self-check mode drives every aiwf
// verb against a throwaway repo and cannot import this package
// directly (it would cycle: cli → doctor → cli), so the wiring is a
// package-level variable assignment.
func init() {
	doctor.Dispatcher = Execute
}

// Execute is the in-process dispatcher: builds the Cobra root command
// tree, executes against the supplied args, returns the exit code.
// cmd/aiwf/main.go's main() is a thin shim that calls Execute(os.Args[1:])
// and os.Exits with the result.
//
// The doctor package's --self-check mode also reaches Execute via the
// doctor.Dispatcher seam (wired in init), so the dispatch path stays
// single-sourced.
//
// Args here are everything after the binary name. AssertSupportedOS
// preflight runs here (and not in main) so the same guard fires for
// the doctor self-check path too. The command tree is built fresh per
// call so tests can drive Execute in parallel without shared mutable
// state.
func Execute(args []string) int {
	if err := cliutil.AssertSupportedOS(runtime.GOOS); err != nil {
		fmt.Fprintln(os.Stderr, "aiwf:", err)
		return cliutil.ExitUsage
	}
	rootCmd := NewRootCmd()
	rootCmd.SetArgs(args)

	err := rootCmd.Execute()
	if err == nil {
		return cliutil.ExitOK
	}
	var ee *cliutil.ExitError
	if errors.As(err, &ee) {
		return ee.Code
	}
	// Non-cliutil.ExitError means Cobra surfaced a usage problem
	// (unknown verb, bad flag, missing required arg). With
	// SilenceErrors:true on the root, Cobra didn't print; we print
	// here in the existing house style.
	fmt.Fprintf(os.Stderr, "aiwf: %v\n", err)
	return cliutil.ExitUsage
}

// NewRootCmd assembles the Cobra command tree. Every verb is a native
// Cobra command (E-14 left no passthrough adapters); each verb's flags
// are tab-completable per the drift-prevention rule in
// completion_drift_test.go.
//
// Exported (rather than newRootCmd) because policy tests under
// internal/policies/ resolve top-level verb names by walking the AST
// of newRootCmd's AddCommand calls. The walker recognizes both
// `pkgIdent.NewCmd()` (subpackage) and `newXCmd()` (legacy) forms;
// keeping the entry point exported lets test callers in cmd/aiwf/ and
// any future cross-package consumer (binary integration tests, e.g.)
// build the tree the same way.
func NewRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:           "aiwf",
		Short:         "ai-workflow framework CLI",
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(c *cobra.Command, args []string) error {
			if v, _ := c.Flags().GetBool("version"); v {
				fmt.Println(version.Current().Version)
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

	cmd.AddCommand(check.NewCmd())
	cmd.AddCommand(add.NewCmd())
	cmd.AddCommand(promote.NewCmd())
	cmd.AddCommand(cancel.NewCmd())
	cmd.AddCommand(rename.NewCmd())
	cmd.AddCommand(renamearea.NewCmd())
	cmd.AddCommand(setarea.NewCmd())
	cmd.AddCommand(retitle.NewCmd())
	cmd.AddCommand(editbody.NewCmd())
	cmd.AddCommand(move.NewCmd())
	cmd.AddCommand(reallocate.NewCmd())
	cmd.AddCommand(rewidth.NewCmd())
	cmd.AddCommand(archive.NewCmd())
	cmd.AddCommand(initcmd.NewCmd())
	cmd.AddCommand(update.NewCmd())
	cmd.AddCommand(upgrade.NewCmd())
	cmd.AddCommand(history.NewCmd())
	cmd.AddCommand(doctor.NewCmd())
	cmd.AddCommand(render.NewCmd())
	cmd.AddCommand(importcmd.NewCmd())
	cmd.AddCommand(whoami.NewCmd())
	cmd.AddCommand(status.NewCmd())
	cmd.AddCommand(list.NewCmd())
	cmd.AddCommand(schema.NewCmd())
	cmd.AddCommand(show.NewCmd())
	cmd.AddCommand(template.NewCmd())
	cmd.AddCommand(contract.NewCmd())
	cmd.AddCommand(milestone.NewCmd())
	cmd.AddCommand(authorize.NewCmd())
	cmd.AddCommand(acknowledge.NewCmd())

	// G-0150: snapshot the explicit verb set into Annotations BEFORE
	// any caller calls Execute (which is when Cobra's `help` and
	// `completion` auto-adds enter the tree). The trailer-verb-unknown
	// rule reads this annotation at RunE time to filter Cobra's
	// auto-adds out of the closed set without hardcoding their names.
	if cmd.Annotations == nil {
		cmd.Annotations = map[string]string{}
	}
	explicit := make([]string, 0, len(cmd.Commands()))
	for _, c := range cmd.Commands() {
		explicit = append(explicit, c.Name())
	}
	cmd.Annotations[cliutil.AnnotationRegisteredVerbs] = strings.Join(explicit, "\n")
	return cmd
}

// newVersionCmd is the M-049 reference shape: a native Cobra command
// whose RunE writes a single-line version string to stdout. It must
// stay byte-coherent with `aiwf -v` / `aiwf --version` (both backed by
// version.Current via the root RunE).
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
			fmt.Println(version.Current().Version)
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
  rename-area <old> <new>        rename a declared area (aiwf.yaml areas.members) and rewrite every entity tagged with it, in one commit
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
