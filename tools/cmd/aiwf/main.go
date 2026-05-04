// Command aiwf is the ai-workflow framework's single binary.
//
// Verbs: check, add, promote, cancel, rename, reallocate, init, update,
// upgrade, history, doctor, render, import, schema, template, plus help/version.
// See docs/pocv3/plans/poc-plan.md for the session breakdown that produced this
// surface.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"

	"github.com/23min/ai-workflow-v2/tools/internal/check"
	"github.com/23min/ai-workflow-v2/tools/internal/config"
	"github.com/23min/ai-workflow-v2/tools/internal/render"
	"github.com/23min/ai-workflow-v2/tools/internal/version"
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

// Exit codes per docs/pocv3/plans/poc-plan.md and tools/CLAUDE.md.
const (
	exitOK       = 0 // no error-severity findings (warnings allowed)
	exitFindings = 1 // at least one error-severity finding
	exitUsage    = 2
	exitInternal = 3
)

func main() {
	if err := assertSupportedOS(runtime.GOOS); err != nil {
		fmt.Fprintln(os.Stderr, "aiwf:", err)
		os.Exit(exitUsage)
	}
	os.Exit(run(os.Args[1:]))
}

func run(args []string) int {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "aiwf: missing verb. Try 'aiwf help'.")
		return exitUsage
	}
	switch args[0] {
	case "--help", "-h", "help":
		printHelp()
		return exitOK
	case "--version", "-v", "version":
		fmt.Println(resolvedVersion())
		return exitOK
	case "check":
		return runCheck(args[1:])
	case "add":
		return runAdd(args[1:])
	case "promote":
		return runPromote(args[1:])
	case "cancel":
		return runCancel(args[1:])
	case "rename":
		return runRename(args[1:])
	case "move":
		return runMove(args[1:])
	case "reallocate":
		return runReallocate(args[1:])
	case "init":
		return runInit(args[1:])
	case "update":
		return runUpdate(args[1:])
	case "upgrade":
		return runUpgrade(args[1:])
	case "history":
		return runHistory(args[1:])
	case "doctor":
		return runDoctor(args[1:])
	case "render":
		return runRender(args[1:])
	case "import":
		return runImport(args[1:])
	case "whoami":
		return runWhoami(args[1:])
	case "status":
		return runStatus(args[1:])
	case "schema":
		return runSchema(args[1:])
	case "show":
		return runShow(args[1:])
	case "template":
		return runTemplate(args[1:])
	case "contract":
		return runContract(args[1:])
	case "authorize":
		return runAuthorize(args[1:])
	default:
		fmt.Fprintf(os.Stderr, "aiwf: unknown verb %q. Try 'aiwf help'.\n", args[0])
		return exitUsage
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

Docs: docs/pocv3/plans/poc-plan.md and docs/pocv3/design/design-decisions.md.`)
}

func runCheck(args []string) int {
	flags := flag.NewFlagSet("check", flag.ContinueOnError)
	root := flags.String("root", "", "consumer repo root (default: discover via aiwf.yaml)")
	format := flags.String("format", "text", "output format: text or json")
	pretty := flags.Bool("pretty", false, "indent JSON output (only with --format=json)")
	since := flags.String("since", "", "explicit base ref for the provenance untrailered-entity audit (default: @{u} when set, else skipped)")
	flags.SetOutput(os.Stderr)
	if err := flags.Parse(args); err != nil {
		return exitUsage
	}

	if *format != "text" && *format != "json" {
		fmt.Fprintf(os.Stderr, "aiwf check: --format must be 'text' or 'json', got %q\n", *format)
		return exitUsage
	}
	if *pretty && *format != "json" {
		fmt.Fprintln(os.Stderr, "aiwf check: --pretty has no effect without --format=json")
	}

	resolved, err := resolveRoot(*root)
	if err != nil {
		fmt.Fprintf(os.Stderr, "aiwf check: %v\n", err)
		return exitUsage
	}

	ctx := context.Background()
	tr, loadErrs, err := loadTreeWithTrunk(ctx, resolved)
	if err != nil {
		fmt.Fprintf(os.Stderr, "aiwf check: loading tree: %v\n", err)
		return exitInternal
	}

	findings := check.Run(tr, loadErrs)

	contracts, contractErr := loadContractsBlock(resolved)
	if contractErr != nil {
		fmt.Fprintf(os.Stderr, "aiwf check: %v\n", contractErr)
		return exitInternal
	}
	contractFindings := runContractValidation(ctx, tr, resolved, contracts)
	findings = append(findings, contractFindings...)

	provenanceFindings, pErr := runProvenanceCheck(ctx, resolved, tr, *since)
	if pErr != nil {
		fmt.Fprintf(os.Stderr, "aiwf check: %v\n", pErr)
		return exitInternal
	}
	findings = append(findings, provenanceFindings...)

	requireMetrics := false
	if cfg, cfgErr := config.Load(resolved); cfgErr == nil && cfg != nil {
		requireMetrics = cfg.TDD.RequireTestMetrics
	}
	metricsFindings, mErr := runTestsMetricsCheck(ctx, resolved, tr, requireMetrics)
	if mErr != nil {
		fmt.Fprintf(os.Stderr, "aiwf check: %v\n", mErr)
		return exitInternal
	}
	findings = append(findings, metricsFindings...)

	applyHintsLikeRun(findings)
	check.SortFindings(findings)

	switch *format {
	case "text":
		if err := render.Text(os.Stdout, findings); err != nil {
			fmt.Fprintf(os.Stderr, "aiwf check: writing output: %v\n", err)
			return exitInternal
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
		if err := render.JSON(os.Stdout, env, *pretty); err != nil {
			fmt.Fprintf(os.Stderr, "aiwf check: writing output: %v\n", err)
			return exitInternal
		}
	}

	if check.HasErrors(findings) {
		return exitFindings
	}
	return exitOK
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
