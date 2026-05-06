package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/23min/ai-workflow-v2/internal/check"
	"github.com/23min/ai-workflow-v2/internal/entity"
	"github.com/23min/ai-workflow-v2/internal/gitops"
	"github.com/23min/ai-workflow-v2/internal/render"
	"github.com/23min/ai-workflow-v2/internal/tree"
	"github.com/23min/ai-workflow-v2/internal/verb"
)

// newAddCmd builds `aiwf add <kind> --title "..." [kind-specific flags]`
// and the `aiwf add ac <milestone-id> --title "..."` sub-shape. ACs are
// modeled as a Cobra subcommand of add (matching their composite-id
// status as sub-elements of a milestone, not a kind in the schema
// sense). For the six top-level kinds, args[0] is the kind and the
// runtime validates kind-vs-flag relevance — same shape as pre-Cobra.
func newAddCmd() *cobra.Command {
	var (
		titles        []string
		actor         string
		principal     string
		root          string
		epicID        string
		discoveredIn  string
		relatesTo     string
		linkedADRs    string
		bindValidator string
		bindSchema    string
		bindFixtures  string
		bodyFile      string
	)
	cmd := &cobra.Command{
		Use:           "add <kind> [...]",
		Short:         "Create a new entity of the given kind",
		Args:          cobra.MinimumNArgs(1),
		SilenceErrors: true,
		SilenceUsage:  true,
		RunE: func(c *cobra.Command, args []string) error {
			if len(args) > 1 {
				fmt.Fprintf(os.Stderr, "aiwf add: unexpected args after kind %q: %v\n", args[0], args[1:])
				return &exitError{code: exitUsage}
			}
			kindArg := args[0]
			k, ok := parseKind(kindArg)
			if !ok {
				fmt.Fprintf(os.Stderr, "aiwf add: unknown kind %q\n", kindArg)
				return &exitError{code: exitUsage}
			}
			if len(titles) > 1 {
				fmt.Fprintf(os.Stderr, "aiwf add: --title may not be repeated for kind %q (only `aiwf add ac` accepts a repeated --title for batched creation)\n", kindArg)
				return &exitError{code: exitUsage}
			}
			title := ""
			if len(titles) == 1 {
				title = titles[0]
			}
			return wrapExitCode(runAddCmd(k, title, actor, principal, root,
				epicID, discoveredIn, relatesTo, linkedADRs,
				bindValidator, bindSchema, bindFixtures, bodyFile))
		},
	}
	// PersistentFlags are inherited by the `add ac` child so the shared
	// `--title`, `--actor`, `--principal`, `--root` work uniformly on
	// both `aiwf add <kind>` and `aiwf add ac <milestone-id>`.
	cmd.PersistentFlags().StringArrayVar(&titles, "title", nil, "entity title (required; for `aiwf add ac` may repeat to create multiple ACs in one atomic commit — M-057)")
	cmd.PersistentFlags().StringVar(&actor, "actor", "", "actor for the commit trailer")
	cmd.PersistentFlags().StringVar(&principal, "principal", "", "the human/<id> the actor is acting on behalf of (required when --actor is non-human; gates the verb through the I2.5 allow-rule)")
	cmd.PersistentFlags().StringVar(&root, "root", "", "consumer repo root")
	cmd.Flags().StringVar(&epicID, "epic", "", "parent epic id (milestone only)")
	cmd.Flags().StringVar(&discoveredIn, "discovered-in", "", "id of milestone or epic where the gap was discovered (gap only)")
	cmd.Flags().StringVar(&relatesTo, "relates-to", "", "comma-separated ids the decision relates to (decision only)")
	cmd.Flags().StringVar(&linkedADRs, "linked-adr", "", "comma-separated ADR ids motivating the contract (contract only)")
	cmd.Flags().StringVar(&bindValidator, "validator", "", "validator name (contract only; if set, --schema and --fixtures are also required and the binding is added atomically)")
	cmd.Flags().StringVar(&bindSchema, "schema", "", "repo-relative path to the schema (contract only; pairs with --validator and --fixtures)")
	cmd.Flags().StringVar(&bindFixtures, "fixtures", "", "repo-relative path to the fixtures-tree root (contract only; pairs with --validator and --schema)")
	cmd.Flags().StringVar(&bodyFile, "body-file", "", `path to a file whose content becomes the entity body, in the same atomic commit as the frontmatter (use "-" to read from stdin); replaces the per-kind default template; the file must contain body content only — leading "---" is refused`)

	cmd.AddCommand(newAddACCmd(&titles, &actor, &principal, &root))
	return cmd
}

func runAddCmd(k entity.Kind, title, actor, principal, root,
	epicID, discoveredIn, relatesTo, linkedADRs,
	bindValidator, bindSchema, bindFixtures, bodyFile string,
) int {
	rootDir, err := resolveRoot(root)
	if err != nil {
		fmt.Fprintf(os.Stderr, "aiwf add: %v\n", err)
		return exitUsage
	}
	actorStr, err := resolveActor(actor, rootDir)
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
		EpicID:        epicID,
		DiscoveredIn:  discoveredIn,
		BindValidator: bindValidator,
		BindSchema:    bindSchema,
		BindFixtures:  bindFixtures,
	}
	opts.RelatesTo = splitCommaList(relatesTo)
	opts.LinkedADRs = splitCommaList(linkedADRs)

	if bodyFile != "" {
		body, readErr := readBodyFile(bodyFile)
		if readErr != nil {
			fmt.Fprintf(os.Stderr, "aiwf add: %v\n", readErr)
			return exitUsage
		}
		opts.BodyOverride = body
	}

	if k == entity.KindContract && bindValidator != "" {
		doc, contracts, loadErr := loadContractsDoc(rootDir)
		if loadErr != nil {
			fmt.Fprintf(os.Stderr, "aiwf add: %v\n", loadErr)
			return exitUsage
		}
		opts.AiwfDoc = doc
		opts.AiwfContracts = contracts
		opts.RepoRoot = rootDir
	}

	result, err := verb.Add(ctx, tr, k, title, actorStr, opts)
	pctx := provenanceContext{
		Actor:        actorStr,
		Principal:    strings.TrimSpace(principal),
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

// newAddACCmd builds `aiwf add ac <milestone-id> --title "..." [--title
// "..."]`. ACs are sub-elements (composite id M-NNN/AC-N), not a kind in
// the schema sense, so they're modeled as a child Cobra command. The
// pointers to the parent's flag variables let one --title slice be
// shared between kinds and ac (a typical pattern with cobra child cmds).
func newAddACCmd(titles *[]string, actor, principal, root *string) *cobra.Command {
	var tests string
	cmd := &cobra.Command{
		Use:           "ac <milestone-id>",
		Short:         "Add one or more acceptance criteria to a milestone",
		Args:          cobra.ExactArgs(1),
		SilenceErrors: true,
		SilenceUsage:  true,
		RunE: func(c *cobra.Command, args []string) error {
			return wrapExitCode(runAddACCmd(args[0], *titles, *actor, *principal, *root, tests))
		},
	}
	cmd.Flags().StringVar(&tests, "tests", "", `optional test metrics for the seeded red phase (only valid when parent milestone is tdd: required and a single AC is being added); format: "pass=N fail=N skip=N total=N" — keys must be one of pass/fail/skip/total, integers non-negative`)
	return cmd
}

func runAddACCmd(parentID string, titles []string, actor, principal, root, tests string) int {
	if len(titles) == 0 {
		fmt.Fprintln(os.Stderr, "aiwf add ac: --title \"...\" is required (pass --title once per AC; repeat for batch)")
		return exitUsage
	}
	metrics, err := parseTestsFlag(tests, "aiwf add ac")
	if err != nil {
		return exitUsage
	}

	rootDir, err := resolveRoot(root)
	if err != nil {
		fmt.Fprintf(os.Stderr, "aiwf add ac: %v\n", err)
		return exitUsage
	}
	actorStr, err := resolveActor(actor, rootDir)
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
	result, err := verb.AddACBatch(ctx, tr, parentID, titles, actorStr, metrics)
	// An AC is a sub-element of its parent milestone — its sole
	// "outbound reference" for scope reachability is the parent id.
	pctx := provenanceContext{
		Actor:        actorStr,
		Principal:    strings.TrimSpace(principal),
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

// newPromoteCmd builds `aiwf promote <id> <new-status>` and the I2
// composite/--phase variants:
//
//	aiwf promote E-01 active                       (top-level entity)
//	aiwf promote M-007/AC-1 met                    (composite, status mode)
//	aiwf promote M-007/AC-1 --phase green          (composite, phase mode)
//
// --phase is mutex with the positional new-status: pass one or the
// other, never both. --phase is only valid for composite ids; using
// it on a top-level entity is a usage error.
func newPromoteCmd() *cobra.Command {
	var (
		actor        string
		principal    string
		root         string
		reason       string
		phase        string
		tests        string
		by           string
		byCommit     string
		supersededBy string
		force        bool
		auditOnly    bool
	)
	cmd := &cobra.Command{
		Use:           "promote <id> [new-status]",
		Short:         "Advance an entity's status (or AC tdd_phase via --phase)",
		Args:          cobra.RangeArgs(1, 2),
		SilenceErrors: true,
		SilenceUsage:  true,
		RunE: func(c *cobra.Command, args []string) error {
			return wrapExitCode(runPromoteCmd(args, actor, principal, root, reason,
				phase, tests, by, byCommit, supersededBy, force, auditOnly))
		},
	}
	cmd.Flags().StringVar(&actor, "actor", "", "actor for the commit trailer")
	cmd.Flags().StringVar(&principal, "principal", "", "the human/<id> the actor is acting on behalf of (required when --actor is non-human; gates the verb through the I2.5 allow-rule)")
	cmd.Flags().StringVar(&root, "root", "", "consumer repo root")
	cmd.Flags().StringVar(&reason, "reason", "", "free-form prose explaining why; lands in the commit body, surfaces in `aiwf history`")
	cmd.Flags().StringVar(&phase, "phase", "", "advance an AC's tdd_phase (composite ids only; mutex with positional new-status)")
	cmd.Flags().StringVar(&tests, "tests", "", `optional test metrics for a phase promotion (composite + --phase only); format: "pass=N fail=N skip=N total=N" — keys must be one of pass/fail/skip/total, integers non-negative`)
	cmd.Flags().StringVar(&by, "by", "", "comma-separated entity ids to write into addressed_by (gap → addressed only); satisfies gap-resolved-has-resolver atomically with the status change")
	cmd.Flags().StringVar(&byCommit, "by-commit", "", "comma-separated commit SHAs to write into addressed_by_commit (gap → addressed only); use when the gap was closed by a specific commit rather than a milestone")
	cmd.Flags().StringVar(&supersededBy, "superseded-by", "", "ADR id to write into superseded_by (adr → superseded only); satisfies adr-supersession-mutual atomically with the status change")
	cmd.Flags().BoolVar(&force, "force", false, "skip the FSM transition rule (requires --reason); coherence checks still run")
	cmd.Flags().BoolVar(&auditOnly, "audit-only", false, "record an audit-trail commit without mutating files; entity must already be at <new-status> (requires --reason; mutex with --force; G24 recovery path)")
	return cmd
}

func runPromoteCmd(args []string, actor, principal, root, reason,
	phase, tests, by, byCommit, supersededBy string, force, auditOnly bool,
) int {
	id := args[0]

	phaseMode := phase != ""
	switch {
	case phaseMode && len(args) == 2:
		fmt.Fprintln(os.Stderr, "aiwf promote: --phase is mutex with the positional new-status; pass one or the other")
		return exitUsage
	case phaseMode && !entity.IsCompositeID(id):
		fmt.Fprintf(os.Stderr, "aiwf promote: --phase is only valid for composite ids (M-NNN/AC-N); got %q\n", id)
		return exitUsage
	case !phaseMode && len(args) != 2:
		fmt.Fprintln(os.Stderr, "aiwf promote: missing new-status. Usage: aiwf promote <id> <new-status>")
		return exitUsage
	}

	if force && auditOnly {
		fmt.Fprintln(os.Stderr, "aiwf promote: --force and --audit-only cannot coexist (force makes a transition; audit-only records one that already happened)")
		return exitUsage
	}
	if (force || auditOnly) && strings.TrimSpace(reason) == "" {
		gateFlag := "--force"
		if auditOnly {
			gateFlag = "--audit-only"
		}
		fmt.Fprintf(os.Stderr, "aiwf promote: --reason \"...\" is required when %s is set (non-empty after trim)\n", gateFlag)
		return exitUsage
	}

	resolverOpts := verb.PromoteOptions{
		AddressedBy:       splitCommaList(by),
		AddressedByCommit: splitCommaList(byCommit),
		SupersededBy:      strings.TrimSpace(supersededBy),
	}
	resolverSet := len(resolverOpts.AddressedBy) > 0 || len(resolverOpts.AddressedByCommit) > 0 || resolverOpts.SupersededBy != ""
	if resolverSet && auditOnly {
		fmt.Fprintln(os.Stderr, "aiwf promote: --by/--by-commit/--superseded-by are not allowed with --audit-only (audit-only records an existing transition; resolver-flag values would imply a mutation)")
		return exitUsage
	}
	if resolverSet && phase != "" {
		fmt.Fprintln(os.Stderr, "aiwf promote: --by/--by-commit/--superseded-by are not valid in phase mode (resolver fields apply to entity status, not AC phase)")
		return exitUsage
	}

	rootDir, err := resolveRoot(root)
	if err != nil {
		fmt.Fprintf(os.Stderr, "aiwf promote: %v\n", err)
		return exitUsage
	}
	actorStr, err := resolveActor(actor, rootDir)
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
		Principal: strings.TrimSpace(principal),
		VerbKind:  verb.VerbAct,
		TargetID:  id,
	}

	if phaseMode {
		metrics, mErr := parseTestsFlag(tests, "aiwf promote")
		if mErr != nil {
			return exitUsage
		}
		var result *verb.Result
		var vErr error
		if auditOnly {
			if metrics != nil {
				fmt.Fprintln(os.Stderr, "aiwf promote: --tests is not allowed with --audit-only (audit-only records an existing transition; no test cycle ran)")
				return exitUsage
			}
			result, vErr = verb.PromoteACPhaseAuditOnly(ctx, tr, id, phase, actorStr, reason)
		} else {
			result, vErr = verb.PromoteACPhase(ctx, tr, id, phase, actorStr, reason, force, metrics)
		}
		return decorateAndFinish(ctx, rootDir, "aiwf promote", tr, result, vErr, pctx)
	}
	if strings.TrimSpace(tests) != "" {
		fmt.Fprintln(os.Stderr, "aiwf promote: --tests is only valid in phase mode (composite id with --phase <p>)")
		return exitUsage
	}
	newStatus := args[1]
	if !entity.IsCompositeID(id) {
		if e := tr.ByID(id); e != nil {
			pctx.IsTerminalPromote = isTerminalPromote(e.Kind, newStatus)
		}
	}
	if auditOnly {
		result, vErr := verb.PromoteAuditOnly(ctx, tr, id, newStatus, actorStr, reason)
		return decorateAndFinish(ctx, rootDir, "aiwf promote", tr, result, vErr, pctx)
	}
	result, vErr := verb.Promote(ctx, tr, id, newStatus, actorStr, reason, force, resolverOpts)
	return decorateAndFinish(ctx, rootDir, "aiwf promote", tr, result, vErr, pctx)
}

// newEditBodyCmd builds `aiwf edit-body <id> --body-file <path>` (and
// `--body-file -` for stdin) — the post-creation body-edit verb that
// closes the plain-git carve-out from G-052 / M-058. Frontmatter is
// untouched; only the markdown body below the frontmatter delimiter
// is replaced. One commit per invocation, standard provenance.
func newEditBodyCmd() *cobra.Command {
	var (
		actor     string
		principal string
		root      string
		reason    string
		bodyFile  string
	)
	cmd := &cobra.Command{
		Use:           "edit-body <id>",
		Short:         "Replace the entity's markdown body (frontmatter untouched)",
		Args:          cobra.ExactArgs(1),
		SilenceErrors: true,
		SilenceUsage:  true,
		RunE: func(c *cobra.Command, args []string) error {
			return wrapExitCode(runEditBodyCmd(args[0], actor, principal, root, reason, bodyFile))
		},
	}
	cmd.Flags().StringVar(&actor, "actor", "", "actor for the commit trailer")
	cmd.Flags().StringVar(&principal, "principal", "", "the human/<id> the actor is acting on behalf of (required when --actor is non-human; gates the verb through the I2.5 allow-rule)")
	cmd.Flags().StringVar(&root, "root", "", "consumer repo root")
	cmd.Flags().StringVar(&reason, "reason", "", "free-form prose explaining why; lands in the commit body, surfaces in `aiwf history`")
	cmd.Flags().StringVar(&bodyFile, "body-file", "", `path to a file whose content becomes the entity's new body (use "-" to read from stdin); the file must contain body content only — leading "---" is refused. Omit to use bless mode: commit whatever the user edited in the working copy of the entity file`)
	return cmd
}

func runEditBodyCmd(id, actor, principal, root, reason, bodyFile string) int {
	// Bless mode (M-060): when --body-file is absent, pass nil bytes
	// so the verb reads working-copy and HEAD itself and commits the
	// diff. Explicit mode (M-058): when --body-file is set, read the
	// file (or stdin for "-") and pass the bytes through.
	var body []byte
	if bodyFile != "" {
		var readErr error
		body, readErr = readBodyFile(bodyFile)
		if readErr != nil {
			fmt.Fprintf(os.Stderr, "aiwf edit-body: %v\n", readErr)
			return exitUsage
		}
		if body == nil {
			body = []byte{}
		}
	}

	rootDir, err := resolveRoot(root)
	if err != nil {
		fmt.Fprintf(os.Stderr, "aiwf edit-body: %v\n", err)
		return exitUsage
	}
	actorStr, err := resolveActor(actor, rootDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "aiwf edit-body: %v\n", err)
		return exitUsage
	}

	release, rc := acquireRepoLock(rootDir, "aiwf edit-body")
	if release == nil {
		return rc
	}
	defer release()

	ctx := context.Background()
	tr, _, err := tree.Load(ctx, rootDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "aiwf edit-body: loading tree: %v\n", err)
		return exitInternal
	}

	pctx := provenanceContext{
		Actor:     actorStr,
		Principal: strings.TrimSpace(principal),
		VerbKind:  verb.VerbAct,
		TargetID:  id,
	}
	result, vErr := verb.EditBody(ctx, tr, id, body, actorStr, reason)
	return decorateAndFinish(ctx, rootDir, "aiwf edit-body", tr, result, vErr, pctx)
}

// newCancelCmd builds `aiwf cancel <id> [--reason "..."]`.
func newCancelCmd() *cobra.Command {
	var (
		actor     string
		principal string
		root      string
		reason    string
		force     bool
		auditOnly bool
	)
	cmd := &cobra.Command{
		Use:           "cancel <id>",
		Short:         "Promote to the kind's terminal-cancel status",
		Args:          cobra.ExactArgs(1),
		SilenceErrors: true,
		SilenceUsage:  true,
		RunE: func(c *cobra.Command, args []string) error {
			return wrapExitCode(runCancelCmd(args[0], actor, principal, root, reason, force, auditOnly))
		},
	}
	cmd.Flags().StringVar(&actor, "actor", "", "actor for the commit trailer")
	cmd.Flags().StringVar(&principal, "principal", "", "the human/<id> the actor is acting on behalf of (required when --actor is non-human; gates the verb through the I2.5 allow-rule)")
	cmd.Flags().StringVar(&root, "root", "", "consumer repo root")
	cmd.Flags().StringVar(&reason, "reason", "", "free-form prose explaining why; lands in the commit body, surfaces in `aiwf history`")
	cmd.Flags().BoolVar(&force, "force", false, "record an audit trailer even when the verb's existing checks would normally allow it (requires --reason)")
	cmd.Flags().BoolVar(&auditOnly, "audit-only", false, "record an audit-trail commit without mutating files; entity must already be at the kind's terminal-cancel target (requires --reason; mutex with --force; G24 recovery path)")
	return cmd
}

func runCancelCmd(id, actor, principal, root, reason string, force, auditOnly bool) int {
	if force && auditOnly {
		fmt.Fprintln(os.Stderr, "aiwf cancel: --force and --audit-only cannot coexist (force makes a transition; audit-only records one that already happened)")
		return exitUsage
	}
	if (force || auditOnly) && strings.TrimSpace(reason) == "" {
		gateFlag := "--force"
		if auditOnly {
			gateFlag = "--audit-only"
		}
		fmt.Fprintf(os.Stderr, "aiwf cancel: --reason \"...\" is required when %s is set (non-empty after trim)\n", gateFlag)
		return exitUsage
	}

	rootDir, err := resolveRoot(root)
	if err != nil {
		fmt.Fprintf(os.Stderr, "aiwf cancel: %v\n", err)
		return exitUsage
	}
	actorStr, err := resolveActor(actor, rootDir)
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
		Principal:         strings.TrimSpace(principal),
		VerbKind:          verb.VerbAct,
		TargetID:          id,
		IsTerminalPromote: !entity.IsCompositeID(id), // cancel always lands on a kind's terminal-cancel target
	}
	if auditOnly {
		result, vErr := verb.CancelAuditOnly(ctx, tr, id, actorStr, reason)
		return decorateAndFinish(ctx, rootDir, "aiwf cancel", tr, result, vErr, pctx)
	}
	result, vErr := verb.Cancel(ctx, tr, id, actorStr, reason, force)
	return decorateAndFinish(ctx, rootDir, "aiwf cancel", tr, result, vErr, pctx)
}

// newRenameCmd builds `aiwf rename <id> <new-slug>`.
func newRenameCmd() *cobra.Command {
	var (
		actor     string
		principal string
		root      string
	)
	cmd := &cobra.Command{
		Use:           "rename <id> <new-slug>",
		Short:         "Rename the file/dir slug; id preserved",
		Args:          cobra.ExactArgs(2),
		SilenceErrors: true,
		SilenceUsage:  true,
		RunE: func(c *cobra.Command, args []string) error {
			return wrapExitCode(runRenameCmd(args[0], args[1], actor, principal, root))
		},
	}
	cmd.Flags().StringVar(&actor, "actor", "", "actor for the commit trailer")
	cmd.Flags().StringVar(&principal, "principal", "", "the human/<id> the actor is acting on behalf of (required when --actor is non-human; gates the verb through the I2.5 allow-rule)")
	cmd.Flags().StringVar(&root, "root", "", "consumer repo root")
	return cmd
}

func runRenameCmd(id, newSlug, actor, principal, root string) int {
	rootDir, err := resolveRoot(root)
	if err != nil {
		fmt.Fprintf(os.Stderr, "aiwf rename: %v\n", err)
		return exitUsage
	}
	actorStr, err := resolveActor(actor, rootDir)
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
		Principal: strings.TrimSpace(principal),
		VerbKind:  verb.VerbAct,
		TargetID:  id,
	}
	return decorateAndFinish(ctx, rootDir, "aiwf rename", tr, result, err, pctx)
}

// newMoveCmd builds `aiwf move <M-id> --epic <E-id>`: relocates a
// milestone to a different epic in one commit.
func newMoveCmd() *cobra.Command {
	var (
		actor     string
		principal string
		root      string
		epic      string
	)
	cmd := &cobra.Command{
		Use:           "move <M-id> --epic <E-id>",
		Short:         "Move a milestone to a different epic; id preserved",
		Args:          cobra.ExactArgs(1),
		SilenceErrors: true,
		SilenceUsage:  true,
		RunE: func(c *cobra.Command, args []string) error {
			if epic == "" {
				fmt.Fprintln(os.Stderr, "aiwf move: --epic <E-id> is required")
				return &exitError{code: exitUsage}
			}
			return wrapExitCode(runMoveCmd(args[0], epic, actor, principal, root))
		},
	}
	cmd.Flags().StringVar(&actor, "actor", "", "actor for the commit trailer")
	cmd.Flags().StringVar(&principal, "principal", "", "the human/<id> the actor is acting on behalf of (required when --actor is non-human; gates the verb through the I2.5 allow-rule)")
	cmd.Flags().StringVar(&root, "root", "", "consumer repo root")
	cmd.Flags().StringVar(&epic, "epic", "", "target epic id (e.g., E-04)")
	return cmd
}

func runMoveCmd(id, epic, actor, principal, root string) int {
	rootDir, err := resolveRoot(root)
	if err != nil {
		fmt.Fprintf(os.Stderr, "aiwf move: %v\n", err)
		return exitUsage
	}
	actorStr, err := resolveActor(actor, rootDir)
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
	result, err := verb.Move(ctx, tr, id, epic, actorStr)
	pctx := provenanceContext{
		Actor:      actorStr,
		Principal:  strings.TrimSpace(principal),
		VerbKind:   verb.VerbMove,
		TargetID:   epic,
		MoveSource: moveSource,
	}
	return decorateAndFinish(ctx, rootDir, "aiwf move", tr, result, err, pctx)
}

// newReallocateCmd builds `aiwf reallocate <id-or-path>`.
func newReallocateCmd() *cobra.Command {
	var (
		actor     string
		principal string
		root      string
	)
	cmd := &cobra.Command{
		Use:           "reallocate <id-or-path>",
		Short:         "Renumber the entity; rewrite refs in others",
		Args:          cobra.ExactArgs(1),
		SilenceErrors: true,
		SilenceUsage:  true,
		RunE: func(c *cobra.Command, args []string) error {
			return wrapExitCode(runReallocateCmd(args[0], actor, principal, root))
		},
	}
	cmd.Flags().StringVar(&actor, "actor", "", "actor for the commit trailer")
	cmd.Flags().StringVar(&principal, "principal", "", "the human/<id> the actor is acting on behalf of (required when --actor is non-human; gates the verb through the I2.5 allow-rule)")
	cmd.Flags().StringVar(&root, "root", "", "consumer repo root")
	return cmd
}

func runReallocateCmd(target, actor, principal, root string) int {
	rootDir, err := resolveRoot(root)
	if err != nil {
		fmt.Fprintf(os.Stderr, "aiwf reallocate: %v\n", err)
		return exitUsage
	}
	actorStr, err := resolveActor(actor, rootDir)
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
		Principal: strings.TrimSpace(principal),
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
