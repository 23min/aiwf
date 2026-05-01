package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/23min/ai-workflow-v2/tools/internal/check"
	"github.com/23min/ai-workflow-v2/tools/internal/entity"
	"github.com/23min/ai-workflow-v2/tools/internal/render"
	"github.com/23min/ai-workflow-v2/tools/internal/tree"
	"github.com/23min/ai-workflow-v2/tools/internal/verb"
)

// runAdd handles `aiwf add <kind> --title "..." [kind-specific flags]`.
func runAdd(args []string) int {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "aiwf add: missing kind. Usage: aiwf add <epic|milestone|adr|gap|decision|contract> --title \"...\"")
		return exitUsage
	}
	kindArg := args[0]
	k, ok := parseKind(kindArg)
	if !ok {
		fmt.Fprintf(os.Stderr, "aiwf add: unknown kind %q\n", kindArg)
		return exitUsage
	}

	fs := flag.NewFlagSet("add "+kindArg, flag.ContinueOnError)
	title := fs.String("title", "", "entity title (required)")
	actor := fs.String("actor", "", "actor for the commit trailer")
	root := fs.String("root", "", "consumer repo root")

	epicID := fs.String("epic", "", "parent epic id (milestone only)")
	discoveredIn := fs.String("discovered-in", "", "id of milestone or epic where the gap was discovered (gap only)")
	relatesTo := fs.String("relates-to", "", "comma-separated ids the decision relates to (decision only)")
	linkedADRs := fs.String("linked-adr", "", "comma-separated ADR ids motivating the contract (contract only)")
	bindValidator := fs.String("validator", "", "validator name (contract only; if set, --schema and --fixtures are also required and the binding is added atomically)")
	bindSchema := fs.String("schema", "", "repo-relative path to the schema (contract only; pairs with --validator and --fixtures)")
	bindFixtures := fs.String("fixtures", "", "repo-relative path to the fixtures-tree root (contract only; pairs with --validator and --schema)")

	fs.SetOutput(os.Stderr)
	if err := fs.Parse(args[1:]); err != nil {
		return exitUsage
	}

	rootDir, err := resolveRoot(*root)
	if err != nil {
		fmt.Fprintf(os.Stderr, "aiwf add: %v\n", err)
		return exitUsage
	}
	actorStr, err := resolveActor(*actor, rootDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "aiwf add: %v\n", err)
		return exitUsage
	}

	release, rc := acquireRepoLock(rootDir, "aiwf add")
	if release == nil {
		return rc
	}
	defer release()

	ctx := context.Background()
	tr, _, err := tree.Load(ctx, rootDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "aiwf add: loading tree: %v\n", err)
		return exitInternal
	}

	opts := verb.AddOptions{
		EpicID:        *epicID,
		DiscoveredIn:  *discoveredIn,
		BindValidator: *bindValidator,
		BindSchema:    *bindSchema,
		BindFixtures:  *bindFixtures,
	}
	opts.RelatesTo = splitCommaList(*relatesTo)
	opts.LinkedADRs = splitCommaList(*linkedADRs)

	if k == entity.KindContract && *bindValidator != "" {
		doc, contracts, loadErr := loadContractsDoc(rootDir)
		if loadErr != nil {
			fmt.Fprintf(os.Stderr, "aiwf add: %v\n", loadErr)
			return exitUsage
		}
		opts.AiwfDoc = doc
		opts.AiwfContracts = contracts
	}

	result, err := verb.Add(tr, k, *title, actorStr, opts)
	return finishVerb(ctx, rootDir, "aiwf add", result, err)
}

// splitCommaList parses comma-separated CLI values into a clean slice
// (trimmed, empty entries dropped). Shared between --relates-to and
// --linked-adr.
func splitCommaList(s string) []string {
	if s == "" {
		return nil
	}
	var out []string
	for _, item := range strings.Split(s, ",") {
		if item = strings.TrimSpace(item); item != "" {
			out = append(out, item)
		}
	}
	return out
}

// runPromote handles `aiwf promote <id> <new-status>`.
func runPromote(args []string) int {
	fs := flag.NewFlagSet("promote", flag.ContinueOnError)
	actor := fs.String("actor", "", "actor for the commit trailer")
	root := fs.String("root", "", "consumer repo root")
	reason := fs.String("reason", "", "free-form prose explaining why; lands in the commit body, surfaces in `aiwf history`")
	fs.SetOutput(os.Stderr)
	if err := fs.Parse(reorderFlagsFirst(args, []string{"actor", "root", "reason"})); err != nil {
		return exitUsage
	}
	rest := fs.Args()
	if len(rest) != 2 {
		fmt.Fprintln(os.Stderr, "aiwf promote: usage: aiwf promote <id> <new-status> [--reason \"...\"]")
		return exitUsage
	}
	id, newStatus := rest[0], rest[1]

	rootDir, err := resolveRoot(*root)
	if err != nil {
		fmt.Fprintf(os.Stderr, "aiwf promote: %v\n", err)
		return exitUsage
	}
	actorStr, err := resolveActor(*actor, rootDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "aiwf promote: %v\n", err)
		return exitUsage
	}

	release, rc := acquireRepoLock(rootDir, "aiwf promote")
	if release == nil {
		return rc
	}
	defer release()

	ctx := context.Background()
	tr, _, err := tree.Load(ctx, rootDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "aiwf promote: loading tree: %v\n", err)
		return exitInternal
	}

	result, err := verb.Promote(tr, id, newStatus, actorStr, *reason)
	return finishVerb(ctx, rootDir, "aiwf promote", result, err)
}

// runCancel handles `aiwf cancel <id> [--reason "..."]`.
func runCancel(args []string) int {
	fs := flag.NewFlagSet("cancel", flag.ContinueOnError)
	actor := fs.String("actor", "", "actor for the commit trailer")
	root := fs.String("root", "", "consumer repo root")
	reason := fs.String("reason", "", "free-form prose explaining why; lands in the commit body, surfaces in `aiwf history`")
	fs.SetOutput(os.Stderr)
	if err := fs.Parse(reorderFlagsFirst(args, []string{"actor", "root", "reason"})); err != nil {
		return exitUsage
	}
	rest := fs.Args()
	if len(rest) != 1 {
		fmt.Fprintln(os.Stderr, "aiwf cancel: usage: aiwf cancel <id> [--reason \"...\"]")
		return exitUsage
	}
	id := rest[0]

	rootDir, err := resolveRoot(*root)
	if err != nil {
		fmt.Fprintf(os.Stderr, "aiwf cancel: %v\n", err)
		return exitUsage
	}
	actorStr, err := resolveActor(*actor, rootDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "aiwf cancel: %v\n", err)
		return exitUsage
	}

	release, rc := acquireRepoLock(rootDir, "aiwf cancel")
	if release == nil {
		return rc
	}
	defer release()

	ctx := context.Background()
	tr, _, err := tree.Load(ctx, rootDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "aiwf cancel: loading tree: %v\n", err)
		return exitInternal
	}
	result, err := verb.Cancel(tr, id, actorStr, *reason)
	return finishVerb(ctx, rootDir, "aiwf cancel", result, err)
}

// runRename handles `aiwf rename <id> <new-slug>`.
func runRename(args []string) int {
	fs := flag.NewFlagSet("rename", flag.ContinueOnError)
	actor := fs.String("actor", "", "actor for the commit trailer")
	root := fs.String("root", "", "consumer repo root")
	fs.SetOutput(os.Stderr)
	if err := fs.Parse(args); err != nil {
		return exitUsage
	}
	rest := fs.Args()
	if len(rest) != 2 {
		fmt.Fprintln(os.Stderr, "aiwf rename: usage: aiwf rename <id> <new-slug>")
		return exitUsage
	}
	id, newSlug := rest[0], rest[1]

	rootDir, err := resolveRoot(*root)
	if err != nil {
		fmt.Fprintf(os.Stderr, "aiwf rename: %v\n", err)
		return exitUsage
	}
	actorStr, err := resolveActor(*actor, rootDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "aiwf rename: %v\n", err)
		return exitUsage
	}

	release, rc := acquireRepoLock(rootDir, "aiwf rename")
	if release == nil {
		return rc
	}
	defer release()

	ctx := context.Background()
	tr, _, err := tree.Load(ctx, rootDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "aiwf rename: loading tree: %v\n", err)
		return exitInternal
	}
	result, err := verb.Rename(tr, id, newSlug, actorStr)
	return finishVerb(ctx, rootDir, "aiwf rename", result, err)
}

// runMove handles `aiwf move <M-id> --epic <E-id>`: relocates a
// milestone to a different epic in one commit.
func runMove(args []string) int {
	fs := flag.NewFlagSet("move", flag.ContinueOnError)
	actor := fs.String("actor", "", "actor for the commit trailer")
	root := fs.String("root", "", "consumer repo root")
	epic := fs.String("epic", "", "target epic id (e.g., E-04)")
	fs.SetOutput(os.Stderr)
	if err := fs.Parse(args); err != nil {
		return exitUsage
	}
	rest := fs.Args()
	if len(rest) != 1 {
		fmt.Fprintln(os.Stderr, "aiwf move: usage: aiwf move <M-id> --epic <E-id>")
		return exitUsage
	}
	id := rest[0]
	if *epic == "" {
		fmt.Fprintln(os.Stderr, "aiwf move: --epic <E-id> is required")
		return exitUsage
	}

	rootDir, err := resolveRoot(*root)
	if err != nil {
		fmt.Fprintf(os.Stderr, "aiwf move: %v\n", err)
		return exitUsage
	}
	actorStr, err := resolveActor(*actor, rootDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "aiwf move: %v\n", err)
		return exitUsage
	}

	release, rc := acquireRepoLock(rootDir, "aiwf move")
	if release == nil {
		return rc
	}
	defer release()

	ctx := context.Background()
	tr, _, err := tree.Load(ctx, rootDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "aiwf move: loading tree: %v\n", err)
		return exitInternal
	}
	result, err := verb.Move(tr, id, *epic, actorStr)
	return finishVerb(ctx, rootDir, "aiwf move", result, err)
}

// runReallocate handles `aiwf reallocate <id-or-path>`.
func runReallocate(args []string) int {
	fs := flag.NewFlagSet("reallocate", flag.ContinueOnError)
	actor := fs.String("actor", "", "actor for the commit trailer")
	root := fs.String("root", "", "consumer repo root")
	fs.SetOutput(os.Stderr)
	if err := fs.Parse(args); err != nil {
		return exitUsage
	}
	rest := fs.Args()
	if len(rest) != 1 {
		fmt.Fprintln(os.Stderr, "aiwf reallocate: usage: aiwf reallocate <id-or-path>")
		return exitUsage
	}
	target := rest[0]

	rootDir, err := resolveRoot(*root)
	if err != nil {
		fmt.Fprintf(os.Stderr, "aiwf reallocate: %v\n", err)
		return exitUsage
	}
	actorStr, err := resolveActor(*actor, rootDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "aiwf reallocate: %v\n", err)
		return exitUsage
	}

	release, rc := acquireRepoLock(rootDir, "aiwf reallocate")
	if release == nil {
		return rc
	}
	defer release()

	ctx := context.Background()
	tr, _, err := tree.Load(ctx, rootDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "aiwf reallocate: loading tree: %v\n", err)
		return exitInternal
	}
	result, err := verb.Reallocate(tr, target, actorStr)
	return finishVerb(ctx, rootDir, "aiwf reallocate", result, err)
}

// finishVerb is the post-verb handler shared by every mutating
// subcommand: it surfaces a Go error as a usage error, renders any
// findings, applies the plan when present, and prints a one-line
// summary on success. NoOp results bypass the apply path entirely
// and print NoOpMessage on stdout.
func finishVerb(ctx context.Context, root, label string, result *verb.Result, err error) int {
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", label, err)
		return exitUsage
	}
	if result == nil {
		fmt.Fprintf(os.Stderr, "%s: no result returned\n", label)
		return exitInternal
	}
	if check.HasErrors(result.Findings) {
		_ = render.Text(os.Stderr, result.Findings)
		return exitFindings
	}
	if result.NoOp {
		fmt.Println(result.NoOpMessage)
		return exitOK
	}
	if result.Plan == nil {
		fmt.Fprintf(os.Stderr, "%s: validation passed but no plan produced\n", label)
		return exitInternal
	}
	if applyErr := verb.Apply(ctx, root, result.Plan); applyErr != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", label, applyErr)
		return exitInternal
	}
	if len(result.Findings) > 0 {
		// Warning-level findings travel with a successful plan
		// (e.g., reallocate body-prose mentions). Surface them but
		// keep the exit code clean.
		_ = render.Text(os.Stderr, result.Findings)
	}
	fmt.Println(result.Plan.Subject)
	return exitOK
}

// parseKind parses a CLI kind argument (lowercase string) into the
// entity.Kind constant.
func parseKind(s string) (entity.Kind, bool) {
	for _, k := range entity.AllKinds() {
		if string(k) == s {
			return k, true
		}
	}
	return "", false
}
