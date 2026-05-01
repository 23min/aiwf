// Command aiwf is the ai-workflow framework's single binary.
//
// Verbs: check, add, promote, cancel, rename, reallocate, init, update,
// history, doctor, render, import, schema, template, plus help/version.
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
	"github.com/23min/ai-workflow-v2/tools/internal/render"
	"github.com/23min/ai-workflow-v2/tools/internal/tree"
)

// Version is the binary's reported version. Set via -ldflags at build
// time once releases start shipping; defaults to "dev" otherwise.
var Version = "dev"

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
		fmt.Println(Version)
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
	default:
		fmt.Fprintf(os.Stderr, "aiwf: unknown verb %q. Try 'aiwf help'.\n", args[0])
		return exitUsage
	}
}

func printHelp() {
	fmt.Println(`aiwf — ai-workflow framework CLI

Usage: aiwf <verb> [args]

Verbs:
  check                          validate the consumer repo's planning state
  add <kind> --title "..."       create a new entity of the given kind
  promote <id> <new-status>      advance an entity's status (optional --reason "..."; --force --reason "..." to skip the FSM); composite ids (M-NNN/AC-N) accepted; --phase <p> for AC tdd_phase (mutex with positional new-status)
  cancel <id>                    promote to the kind's terminal-cancel status (optional --reason "..."; --force --reason "..." records the cancellation as an audit event)
  rename <id> <new-slug>         rename the file/dir slug; id preserved
  move <M-id> --epic <E-id>      move a milestone to a different epic; id preserved
  reallocate <id-or-path>        renumber the entity; rewrite refs in others
  init                           one-time setup: aiwf.yaml, scaffolding, skills, pre-push hook
  update                         re-materialize embedded skills into .claude/skills/aiwf-*/
  history <id>                   show the entity's lifecycle from git log trailers
  doctor [--self-check]          drift / version / id-collision health check; --self-check drives every verb against a temp repo
  render roadmap [--write]       print ROADMAP.md (markdown of epics + milestones); --write commits it
  import <manifest>              bulk-create entities from a YAML/JSON manifest (one commit by default)
  whoami                         print the resolved actor and the source it came from
  status                         project snapshot: in-flight work, open decisions, gaps, recent activity
  show <id>                      aggregate view: frontmatter + recent history + active findings (composite ids accepted)
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

Flags for 'add':
  --epic <id>                    parent epic id (milestone)
  --discovered-in <id>           discovery context (gap)
  --relates-to <id,id,...>       related entities (decision)
  --linked-adr <id,id,...>       ADRs motivating the contract (contract)
  --validator <name>             validator name to bind (contract; with --schema, --fixtures: atomic add+bind)
  --schema <path>                schema path (contract; pairs with --validator and --fixtures)
  --fixtures <path>              fixtures-tree root (contract; pairs with --validator and --schema)

Flags for 'check', 'history', and 'contract verify':
  --format <fmt>                 output format: text (default) or json
  --pretty                       indent JSON output (only with --format=json)

Flags for 'import':
  --on-collision <mode>          fail (default) | skip | update — behavior when an explicit id already exists
  --dry-run                      validate the projection and print the would-be plans without writing

Exit codes: 0 = no errors, 1 = errors found, 2 = usage error, 3 = internal error.

Docs: docs/pocv3/plans/poc-plan.md and docs/pocv3/design/design-decisions.md.`)
}

func runCheck(args []string) int {
	flags := flag.NewFlagSet("check", flag.ContinueOnError)
	root := flags.String("root", "", "consumer repo root (default: discover via aiwf.yaml)")
	format := flags.String("format", "text", "output format: text or json")
	pretty := flags.Bool("pretty", false, "indent JSON output (only with --format=json)")
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
	tr, loadErrs, err := tree.Load(ctx, resolved)
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
