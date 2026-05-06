package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/23min/ai-workflow-v2/internal/check"
	"github.com/23min/ai-workflow-v2/internal/entity"
	"github.com/23min/ai-workflow-v2/internal/gitops"
	"github.com/23min/ai-workflow-v2/internal/render"
	"github.com/23min/ai-workflow-v2/internal/tree"
	"github.com/23min/ai-workflow-v2/internal/verb"
)

// runAdd handles `aiwf add <kind> --title "..." [kind-specific flags]`.
// The "ac" sub-target dispatches to runAddAC: ACs are sub-elements of
// a milestone, not a kind, and the verb shape (parent id positional,
// then --title) differs.
func runAdd(args []string) int {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "aiwf add: missing kind. Usage: aiwf add <epic|milestone|adr|gap|decision|contract|ac> [...]")
		return exitUsage
	}
	kindArg := args[0]
	if kindArg == "ac" {
		return runAddAC(args[1:])
	}
	k, ok := parseKind(kindArg)
	if !ok {
		fmt.Fprintf(os.Stderr, "aiwf add: unknown kind %q\n", kindArg)
		return exitUsage
	}

	fs := flag.NewFlagSet("add "+kindArg, flag.ContinueOnError)
	title := fs.String("title", "", "entity title (required)")
	actor := fs.String("actor", "", "actor for the commit trailer")
	principal := fs.String("principal", "", "the human/<id> the actor is acting on behalf of (required when --actor is non-human; gates the verb through the I2.5 allow-rule)")
	root := fs.String("root", "", "consumer repo root")

	epicID := fs.String("epic", "", "parent epic id (milestone only)")
	discoveredIn := fs.String("discovered-in", "", "id of milestone or epic where the gap was discovered (gap only)")
	relatesTo := fs.String("relates-to", "", "comma-separated ids the decision relates to (decision only)")
	linkedADRs := fs.String("linked-adr", "", "comma-separated ADR ids motivating the contract (contract only)")
	bindValidator := fs.String("validator", "", "validator name (contract only; if set, --schema and --fixtures are also required and the binding is added atomically)")
	bindSchema := fs.String("schema", "", "repo-relative path to the schema (contract only; pairs with --validator and --fixtures)")
	bindFixtures := fs.String("fixtures", "", "repo-relative path to the fixtures-tree root (contract only; pairs with --validator and --schema)")
	bodyFile := fs.String("body-file", "", `path to a file whose content becomes the entity body, in the same atomic commit as the frontmatter (use "-" to read from stdin); replaces the per-kind default template; the file must contain body content only — leading "---" is refused`)

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
	tr, _, err := loadTreeWithTrunk(ctx, rootDir)
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

	if *bodyFile != "" {
		body, readErr := readBodyFile(*bodyFile)
		if readErr != nil {
			fmt.Fprintf(os.Stderr, "aiwf add: %v\n", readErr)
			return exitUsage
		}
		opts.BodyOverride = body
	}

	if k == entity.KindContract && *bindValidator != "" {
		doc, contracts, loadErr := loadContractsDoc(rootDir)
		if loadErr != nil {
			fmt.Fprintf(os.Stderr, "aiwf add: %v\n", loadErr)
			return exitUsage
		}
		opts.AiwfDoc = doc
		opts.AiwfContracts = contracts
		opts.RepoRoot = rootDir
	}

	result, err := verb.Add(ctx, tr, k, *title, actorStr, opts)
	pctx := provenanceContext{
		Actor:        actorStr,
		Principal:    strings.TrimSpace(*principal),
		VerbKind:     verb.VerbCreate,
		CreationRefs: addCreationRefs(k, opts),
	}
	return decorateAndFinish(ctx, rootDir, "aiwf add", tr, result, err, pctx)
}

// addCreationRefs returns the new entity's outbound references for
// the I2.5 allow-rule's VerbCreate reachability check. Each kind
// names a different set of ref-bearing fields; the helper centralizes
// the mapping so the cmd dispatcher doesn't duplicate the schema.
//
// An epic has no outbound references (root of the tree); an agent
// authorizing the addition of a fresh epic must be scoped to that
// epic's id, which doesn't yet exist — meaning agents cannot create
// top-level epics under any active scope (intentional; new top-level
// work is a human decision per the design).
func addCreationRefs(k entity.Kind, opts verb.AddOptions) []string {
	var refs []string
	if opts.EpicID != "" {
		refs = append(refs, opts.EpicID)
	}
	if opts.DiscoveredIn != "" {
		refs = append(refs, opts.DiscoveredIn)
	}
	refs = append(refs, opts.RelatesTo...)
	refs = append(refs, opts.LinkedADRs...)
	_ = k // reserved for future kind-specific ref derivation
	return refs
}

// runAddAC handles `aiwf add ac <milestone-id> --title "..."`. ACs
// are sub-elements of a milestone (composite id M-NNN/AC-N), not a
// kind in the schema sense, so they have their own verb shape.
func runAddAC(args []string) int {
	fs := flag.NewFlagSet("add ac", flag.ContinueOnError)
	var titles repeatedString
	fs.Var(&titles, "title", "AC title (required; repeat to create multiple ACs in one atomic commit — M-057)")
	actor := fs.String("actor", "", "actor for the commit trailer")
	principal := fs.String("principal", "", "the human/<id> the actor is acting on behalf of (required when --actor is non-human; gates the verb through the I2.5 allow-rule)")
	root := fs.String("root", "", "consumer repo root")
	tests := fs.String("tests", "", `optional test metrics for the seeded red phase (only valid when parent milestone is tdd: required and a single AC is being added); format: "pass=N fail=N skip=N total=N" — keys must be one of pass/fail/skip/total, integers non-negative`)
	fs.SetOutput(os.Stderr)
	if err := fs.Parse(reorderFlagsFirst(args, []string{"actor", "principal", "root", "title", "tests"}, nil)); err != nil {
		return exitUsage
	}
	rest := fs.Args()
	if len(rest) != 1 {
		fmt.Fprintln(os.Stderr, "aiwf add ac: usage: aiwf add ac <milestone-id> --title \"...\" [--title \"...\" ...]")
		return exitUsage
	}
	parentID := rest[0]

	if len(titles) == 0 {
		fmt.Fprintln(os.Stderr, "aiwf add ac: --title \"...\" is required (pass --title once per AC; repeat for batch)")
		return exitUsage
	}

	metrics, err := parseTestsFlag(*tests, "aiwf add ac")
	if err != nil {
		return exitUsage
	}

	rootDir, err := resolveRoot(*root)
	if err != nil {
		fmt.Fprintf(os.Stderr, "aiwf add ac: %v\n", err)
		return exitUsage
	}
	actorStr, err := resolveActor(*actor, rootDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "aiwf add ac: %v\n", err)
		return exitUsage
	}

	release, rc := acquireRepoLock(rootDir, "aiwf add ac")
	if release == nil {
		return rc
	}
	defer release()

	ctx := context.Background()
	tr, _, err := tree.Load(ctx, rootDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "aiwf add ac: loading tree: %v\n", err)
		return exitInternal
	}
	result, err := verb.AddACBatch(ctx, tr, parentID, []string(titles), actorStr, metrics)
	// An AC is a sub-element of its parent milestone — its sole
	// "outbound reference" for scope reachability is the parent id.
	pctx := provenanceContext{
		Actor:        actorStr,
		Principal:    strings.TrimSpace(*principal),
		VerbKind:     verb.VerbCreate,
		CreationRefs: []string{parentID},
	}
	return decorateAndFinish(ctx, rootDir, "aiwf add ac", tr, result, err, pctx)
}

// parseTestsFlag parses a `--tests` value at the verb dispatcher
// boundary. Empty input returns (nil, nil) — the flag wasn't set.
// Otherwise applies the strict on-wire grammar (pass/fail/skip/total
// keys, non-negative integers) and returns a *gitops.TestMetrics; on
// any malformed input writes a one-line error to stderr (prefixed
// with verbLabel) and returns the parse error so the dispatcher exits
// with exitUsage.
func parseTestsFlag(raw, verbLabel string) (*gitops.TestMetrics, error) {
	if strings.TrimSpace(raw) == "" {
		return nil, nil
	}
	m, err := gitops.ParseStrictTestMetrics(raw)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", verbLabel, err)
		return nil, err
	}
	if m == (gitops.TestMetrics{}) {
		// Empty after parse — defensive; ParseStrict returns zero
		// metrics for empty input but here the trimmed input was
		// non-empty so this shouldn't fire. If it does, treat the
		// flag as not set to avoid emitting a meaningless trailer.
		return nil, nil
	}
	return &m, nil
}

// readBodyFile loads body content for `aiwf add --body-file`. A path
// of "-" reads stdin (so callers can pipe body text without a temp
// file). Any other value is read as a regular file. Returns the raw
// bytes; the verb-side resolveAddBody is the rule-checking layer
// (it refuses content that begins with a frontmatter delimiter so
// the create commit can't accidentally produce a double-frontmatter
// file).
func readBodyFile(path string) ([]byte, error) {
	if path == "-" {
		return io.ReadAll(os.Stdin)
	}
	return os.ReadFile(path)
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

// runPromote handles `aiwf promote <id> <new-status>` and the I2
// composite/--phase variants:
//
//	aiwf promote E-01 active                       (top-level entity)
//	aiwf promote M-007/AC-1 met                    (composite, status mode)
//	aiwf promote M-007/AC-1 --phase green          (composite, phase mode)
//
// --phase is mutex with the positional new-status: pass one or the
// other, never both. --phase is only valid for composite ids; using
// it on a top-level entity is a usage error.
func runPromote(args []string) int {
	fs := flag.NewFlagSet("promote", flag.ContinueOnError)
	actor := fs.String("actor", "", "actor for the commit trailer")
	principal := fs.String("principal", "", "the human/<id> the actor is acting on behalf of (required when --actor is non-human; gates the verb through the I2.5 allow-rule)")
	root := fs.String("root", "", "consumer repo root")
	reason := fs.String("reason", "", "free-form prose explaining why; lands in the commit body, surfaces in `aiwf history`")
	phase := fs.String("phase", "", "advance an AC's tdd_phase (composite ids only; mutex with positional new-status)")
	tests := fs.String("tests", "", `optional test metrics for a phase promotion (composite + --phase only); format: "pass=N fail=N skip=N total=N" — keys must be one of pass/fail/skip/total, integers non-negative`)
	by := fs.String("by", "", "comma-separated entity ids to write into addressed_by (gap → addressed only); satisfies gap-resolved-has-resolver atomically with the status change")
	byCommit := fs.String("by-commit", "", "comma-separated commit SHAs to write into addressed_by_commit (gap → addressed only); use when the gap was closed by a specific commit rather than a milestone")
	supersededBy := fs.String("superseded-by", "", "ADR id to write into superseded_by (adr → superseded only); satisfies adr-supersession-mutual atomically with the status change")
	force := fs.Bool("force", false, "skip the FSM transition rule (requires --reason); coherence checks still run")
	auditOnly := fs.Bool("audit-only", false, "record an audit-trail commit without mutating files; entity must already be at <new-status> (requires --reason; mutex with --force; G24 recovery path)")
	fs.SetOutput(os.Stderr)
	if err := fs.Parse(reorderFlagsFirst(args, []string{"actor", "principal", "root", "reason", "phase", "tests", "by", "by-commit", "superseded-by"}, []string{"force", "audit-only"})); err != nil {
		return exitUsage
	}
	rest := fs.Args()
	if len(rest) < 1 || len(rest) > 2 {
		fmt.Fprintln(os.Stderr, "aiwf promote: usage: aiwf promote <id> <new-status>  |  aiwf promote <composite-id> --phase <p>  [--reason \"...\"] [--force --reason \"...\"] [--audit-only --reason \"...\"]")
		return exitUsage
	}
	id := rest[0]

	// Mode resolution: phase mode (composite + --phase, no positional state)
	// vs. status mode (positional new-status).
	phaseMode := *phase != ""
	switch {
	case phaseMode && len(rest) == 2:
		fmt.Fprintln(os.Stderr, "aiwf promote: --phase is mutex with the positional new-status; pass one or the other")
		return exitUsage
	case phaseMode && !entity.IsCompositeID(id):
		fmt.Fprintf(os.Stderr, "aiwf promote: --phase is only valid for composite ids (M-NNN/AC-N); got %q\n", id)
		return exitUsage
	case !phaseMode && len(rest) != 2:
		fmt.Fprintln(os.Stderr, "aiwf promote: missing new-status. Usage: aiwf promote <id> <new-status>")
		return exitUsage
	}

	if *force && *auditOnly {
		fmt.Fprintln(os.Stderr, "aiwf promote: --force and --audit-only cannot coexist (force makes a transition; audit-only records one that already happened)")
		return exitUsage
	}
	if (*force || *auditOnly) && strings.TrimSpace(*reason) == "" {
		gateFlag := "--force"
		if *auditOnly {
			gateFlag = "--audit-only"
		}
		fmt.Fprintf(os.Stderr, "aiwf promote: --reason \"...\" is required when %s is set (non-empty after trim)\n", gateFlag)
		return exitUsage
	}

	resolverOpts := verb.PromoteOptions{
		AddressedBy:       splitCommaList(*by),
		AddressedByCommit: splitCommaList(*byCommit),
		SupersededBy:      strings.TrimSpace(*supersededBy),
	}
	resolverSet := len(resolverOpts.AddressedBy) > 0 || len(resolverOpts.AddressedByCommit) > 0 || resolverOpts.SupersededBy != ""
	if resolverSet && *auditOnly {
		fmt.Fprintln(os.Stderr, "aiwf promote: --by/--by-commit/--superseded-by are not allowed with --audit-only (audit-only records an existing transition; resolver-flag values would imply a mutation)")
		return exitUsage
	}
	if resolverSet && *phase != "" {
		fmt.Fprintln(os.Stderr, "aiwf promote: --by/--by-commit/--superseded-by are not valid in phase mode (resolver fields apply to entity status, not AC phase)")
		return exitUsage
	}

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

	pctx := provenanceContext{
		Actor:     actorStr,
		Principal: strings.TrimSpace(*principal),
		VerbKind:  verb.VerbAct,
		TargetID:  id,
	}

	if phaseMode {
		metrics, mErr := parseTestsFlag(*tests, "aiwf promote")
		if mErr != nil {
			return exitUsage
		}
		var result *verb.Result
		var vErr error
		if *auditOnly {
			if metrics != nil {
				fmt.Fprintln(os.Stderr, "aiwf promote: --tests is not allowed with --audit-only (audit-only records an existing transition; no test cycle ran)")
				return exitUsage
			}
			result, vErr = verb.PromoteACPhaseAuditOnly(ctx, tr, id, *phase, actorStr, *reason)
		} else {
			result, vErr = verb.PromoteACPhase(ctx, tr, id, *phase, actorStr, *reason, *force, metrics)
		}
		return decorateAndFinish(ctx, rootDir, "aiwf promote", tr, result, vErr, pctx)
	}
	if strings.TrimSpace(*tests) != "" {
		fmt.Fprintln(os.Stderr, "aiwf promote: --tests is only valid in phase mode (composite id with --phase <p>)")
		return exitUsage
	}
	newStatus := rest[1]
	if !entity.IsCompositeID(id) {
		if e := tr.ByID(id); e != nil {
			pctx.IsTerminalPromote = isTerminalPromote(e.Kind, newStatus)
		}
	}
	if *auditOnly {
		result, vErr := verb.PromoteAuditOnly(ctx, tr, id, newStatus, actorStr, *reason)
		return decorateAndFinish(ctx, rootDir, "aiwf promote", tr, result, vErr, pctx)
	}
	result, vErr := verb.Promote(ctx, tr, id, newStatus, actorStr, *reason, *force, resolverOpts)
	return decorateAndFinish(ctx, rootDir, "aiwf promote", tr, result, vErr, pctx)
}

// runCancel handles `aiwf cancel <id> [--reason "..."]`.
func runCancel(args []string) int {
	fs := flag.NewFlagSet("cancel", flag.ContinueOnError)
	actor := fs.String("actor", "", "actor for the commit trailer")
	principal := fs.String("principal", "", "the human/<id> the actor is acting on behalf of (required when --actor is non-human; gates the verb through the I2.5 allow-rule)")
	root := fs.String("root", "", "consumer repo root")
	reason := fs.String("reason", "", "free-form prose explaining why; lands in the commit body, surfaces in `aiwf history`")
	force := fs.Bool("force", false, "record an audit trailer even when the verb's existing checks would normally allow it (requires --reason)")
	auditOnly := fs.Bool("audit-only", false, "record an audit-trail commit without mutating files; entity must already be at the kind's terminal-cancel target (requires --reason; mutex with --force; G24 recovery path)")
	fs.SetOutput(os.Stderr)
	if err := fs.Parse(reorderFlagsFirst(args, []string{"actor", "principal", "root", "reason"}, []string{"force", "audit-only"})); err != nil {
		return exitUsage
	}
	rest := fs.Args()
	if len(rest) != 1 {
		fmt.Fprintln(os.Stderr, "aiwf cancel: usage: aiwf cancel <id> [--reason \"...\"] [--force --reason \"...\"] [--audit-only --reason \"...\"]")
		return exitUsage
	}
	id := rest[0]

	if *force && *auditOnly {
		fmt.Fprintln(os.Stderr, "aiwf cancel: --force and --audit-only cannot coexist (force makes a transition; audit-only records one that already happened)")
		return exitUsage
	}
	if (*force || *auditOnly) && strings.TrimSpace(*reason) == "" {
		gateFlag := "--force"
		if *auditOnly {
			gateFlag = "--audit-only"
		}
		fmt.Fprintf(os.Stderr, "aiwf cancel: --reason \"...\" is required when %s is set (non-empty after trim)\n", gateFlag)
		return exitUsage
	}

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
	pctx := provenanceContext{
		Actor:             actorStr,
		Principal:         strings.TrimSpace(*principal),
		VerbKind:          verb.VerbAct,
		TargetID:          id,
		IsTerminalPromote: !entity.IsCompositeID(id), // cancel always lands on a kind's terminal-cancel target
	}
	if *auditOnly {
		result, vErr := verb.CancelAuditOnly(ctx, tr, id, actorStr, *reason)
		return decorateAndFinish(ctx, rootDir, "aiwf cancel", tr, result, vErr, pctx)
	}
	result, vErr := verb.Cancel(ctx, tr, id, actorStr, *reason, *force)
	return decorateAndFinish(ctx, rootDir, "aiwf cancel", tr, result, vErr, pctx)
}

// runRename handles `aiwf rename <id> <new-slug>`.
func runRename(args []string) int {
	fs := flag.NewFlagSet("rename", flag.ContinueOnError)
	actor := fs.String("actor", "", "actor for the commit trailer")
	principal := fs.String("principal", "", "the human/<id> the actor is acting on behalf of (required when --actor is non-human; gates the verb through the I2.5 allow-rule)")
	root := fs.String("root", "", "consumer repo root")
	fs.SetOutput(os.Stderr)
	if err := fs.Parse(reorderFlagsFirst(args, []string{"actor", "principal", "root"}, nil)); err != nil {
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
	result, err := verb.Rename(ctx, tr, id, newSlug, actorStr)
	pctx := provenanceContext{
		Actor:     actorStr,
		Principal: strings.TrimSpace(*principal),
		VerbKind:  verb.VerbAct,
		TargetID:  id,
	}
	return decorateAndFinish(ctx, rootDir, "aiwf rename", tr, result, err, pctx)
}

// runMove handles `aiwf move <M-id> --epic <E-id>`: relocates a
// milestone to a different epic in one commit.
func runMove(args []string) int {
	fs := flag.NewFlagSet("move", flag.ContinueOnError)
	actor := fs.String("actor", "", "actor for the commit trailer")
	principal := fs.String("principal", "", "the human/<id> the actor is acting on behalf of (required when --actor is non-human; gates the verb through the I2.5 allow-rule)")
	root := fs.String("root", "", "consumer repo root")
	epic := fs.String("epic", "", "target epic id (e.g., E-04)")
	fs.SetOutput(os.Stderr)
	if err := fs.Parse(reorderFlagsFirst(args, []string{"actor", "principal", "root", "epic"}, nil)); err != nil {
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
	// Move endpoints for the allow-rule are the source epic (the
	// milestone's current parent) and the destination epic (--epic).
	// Both must reach the scope-entity per the strict-move rule.
	var moveSource string
	if e := tr.ByID(id); e != nil {
		moveSource = e.Parent
	}
	result, err := verb.Move(ctx, tr, id, *epic, actorStr)
	pctx := provenanceContext{
		Actor:      actorStr,
		Principal:  strings.TrimSpace(*principal),
		VerbKind:   verb.VerbMove,
		TargetID:   *epic,
		MoveSource: moveSource,
	}
	return decorateAndFinish(ctx, rootDir, "aiwf move", tr, result, err, pctx)
}

// runReallocate handles `aiwf reallocate <id-or-path>`.
func runReallocate(args []string) int {
	fs := flag.NewFlagSet("reallocate", flag.ContinueOnError)
	actor := fs.String("actor", "", "actor for the commit trailer")
	principal := fs.String("principal", "", "the human/<id> the actor is acting on behalf of (required when --actor is non-human; gates the verb through the I2.5 allow-rule)")
	root := fs.String("root", "", "consumer repo root")
	fs.SetOutput(os.Stderr)
	if err := fs.Parse(reorderFlagsFirst(args, []string{"actor", "principal", "root"}, nil)); err != nil {
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
	tr, _, err := loadTreeWithTrunk(ctx, rootDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "aiwf reallocate: loading tree: %v\n", err)
		return exitInternal
	}
	result, err := verb.Reallocate(ctx, tr, target, actorStr)
	pctx := provenanceContext{
		Actor:     actorStr,
		Principal: strings.TrimSpace(*principal),
		VerbKind:  verb.VerbAct,
		TargetID:  target,
	}
	return decorateAndFinish(ctx, rootDir, "aiwf reallocate", tr, result, err, pctx)
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
