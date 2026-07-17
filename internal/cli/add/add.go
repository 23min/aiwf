// Package add implements the `aiwf add` verb and its `aiwf add ac`
// subcommand (per-verb subpackage of M-0115; cmd/aiwf/main.go's
// newRootCmd wires it via NewCmd). Both verbs share the package so
// the Cobra subcommand wiring (`add ac` as a child of `add`) and the
// PersistentFlag-sharing pattern remain intact.
package add

import (
	"bytes"
	"context"
	"log/slog"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/23min/aiwf/internal/areamatch"
	"github.com/23min/aiwf/internal/cli/cliutil"
	"github.com/23min/aiwf/internal/entity"
	"github.com/23min/aiwf/internal/gitops"
	"github.com/23min/aiwf/internal/logger"
	"github.com/23min/aiwf/internal/tree"
	"github.com/23min/aiwf/internal/verb"
)

// NewCmd builds `aiwf add <kind> --title "..." [kind-specific flags]`
// and the `aiwf add ac <milestone-id> --title "..."` sub-shape. ACs are
// modeled as a Cobra subcommand of add (matching their composite-id
// status as sub-elements of a milestone, not a kind in the schema
// sense). For the six top-level kinds, args[0] is the kind and the
// runtime validates kind-vs-flag relevance — same shape as pre-Cobra.
func NewCmd(correlationID string) *cobra.Command {
	var (
		titles        []string
		actor         string
		principal     string
		root          string
		epicID        string
		tddPolicy     string
		dependsOn     string
		discoveredIn  string
		area          string
		priority      string
		pathHint      string
		relatesTo     string
		linkedADRs    string
		bindValidator string
		bindSchema    string
		bindFixtures  string
		bodyFile      string
		bodyText      string
		force         bool
		reason        string
		fetch         bool
		out           *cliutil.OutputFormat
	)
	cmd := &cobra.Command{
		Use:   "add <kind> [...]",
		Short: "Create a new entity of the given kind",
		Example: `  # Create a top-level epic
  aiwf add epic --title "Foundations and aiwf check"

  # Create a milestone under an epic (--tdd is required: required|advisory|none)
  aiwf add milestone --epic E-01 --tdd required --title "Bootstrap Cobra"

  # Create a contract atomically wired to a validator
  aiwf add contract --linked-adr ADR-0001 --title "Render envelope" \
    --validator render --schema schemas/render.cue --fixtures fixtures/render`,
		Args:          cobra.MinimumNArgs(1),
		SilenceErrors: true,
		SilenceUsage:  true,
		RunE: func(c *cobra.Command, args []string) error {
			if len(args) > 1 {
				cliutil.Errorf("aiwf add: unexpected args after kind %q: %v\n", args[0], args[1:])
				return cliutil.WrapExitCode(cliutil.ExitUsage)
			}
			kindArg := args[0]
			k, ok := cliutil.ParseKind(kindArg)
			if !ok {
				cliutil.Errorf("aiwf add: unknown kind %q\n", kindArg)
				return cliutil.WrapExitCode(cliutil.ExitUsage)
			}
			if len(titles) > 1 {
				cliutil.Errorf("aiwf add: --title may not be repeated for kind %q (only `aiwf add ac` accepts a repeated --title for batched creation)\n", kindArg)
				return cliutil.WrapExitCode(cliutil.ExitUsage)
			}
			title := ""
			if len(titles) == 1 {
				title = titles[0]
			}
			return cliutil.WrapExitCode(Run(k, title, actor, principal, root,
				epicID, tddPolicy, dependsOn, discoveredIn, area, priority, pathHint, relatesTo, linkedADRs,
				bindValidator, bindSchema, bindFixtures, bodyFile, bodyText, reason, fetch, force, *out))
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
	cmd.Flags().StringVar(&tddPolicy, "tdd", "", "milestone TDD policy: required|advisory|none — required at creation time for kind=milestone (G-055 layer 1)")
	cmd.Flags().StringVar(&dependsOn, "depends-on", "", "comma-separated milestone ids the new milestone depends on (milestone only); each id must resolve to an existing milestone (M-076)")
	cmd.Flags().StringVar(&discoveredIn, "discovered-in", "", "id of milestone or epic where the gap was discovered (gap only)")
	cmd.Flags().StringVar(&area, "area", "", "workstream area tag (root kinds only); validated against aiwf.yaml: areas.members; a gap with --discovered-in derives it when omitted (E-0043)")
	cmd.Flags().StringVar(&priority, "priority", "", "priority level (gap/decision only): urgent, high, medium, low (G-0078, E-0066)")
	cmd.Flags().StringVar(&pathHint, "path-hint", "", "repo-relative path hint (root kinds only); when --area is omitted and the hint falls under exactly one declared area's paths, derive area from it via the areamatch SSOT (E-0044, M-0182)")
	cmd.Flags().StringVar(&relatesTo, "relates-to", "", "comma-separated ids the decision relates to (decision only)")
	cmd.Flags().StringVar(&linkedADRs, "linked-adr", "", "comma-separated ADR ids motivating the contract (contract only)")
	cmd.Flags().StringVar(&bindValidator, "validator", "", "validator name (contract only; if set, --schema and --fixtures are also required and the binding is added atomically)")
	cmd.Flags().StringVar(&bindSchema, "schema", "", "repo-relative path to the schema (contract only; pairs with --validator and --fixtures)")
	cmd.Flags().StringVar(&bindFixtures, "fixtures", "", "repo-relative path to the fixtures-tree root (contract only; pairs with --validator and --schema)")
	cmd.Flags().StringVar(&bodyFile, "body-file", "", `path to a file whose content becomes the entity body, in the same atomic commit as the frontmatter (use "-" to read from stdin); replaces the per-kind default template; the file must contain body content only — leading "---" is refused; mutually exclusive with --body`)
	cmd.Flags().StringVar(&bodyText, "body", "", `inline text that becomes the entity body, in the same atomic commit as the frontmatter; replaces the per-kind default template; must not begin with a "---" frontmatter delimiter; mutually exclusive with --body-file`)
	cmd.Flags().BoolVar(&force, "force", false, "bypass the born-complete-kind empty-body gate (gap/decision/adr/contract only — G-0326: these kinds have no draft phase, so an empty body is refused at creation); requires --reason; inert on epic/milestone (no gate to bypass there)")
	cmd.Flags().StringVar(&reason, "reason", "", `sovereign-override justification recorded in the "aiwf-force:" commit trailer; required (non-empty after trim) when --force is set`)
	cmd.Flags().BoolVar(&fetch, "fetch", false, "before allocating the id, best-effort `git fetch --all` to refresh every remote-tracking ref, so the id is computed against the freshest published view across all branches (not just trunk); a fetch failure (offline, unreachable remote) degrades to local-only allocation with a warning and never blocks the add (M-0214)")
	out = cliutil.AddFormatFlags(cmd)
	out.CorrelationID = correlationID

	cmd.ValidArgsFunction = func(_ *cobra.Command, args []string, _ string) ([]string, cobra.ShellCompDirective) {
		if len(args) > 0 {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		// `ac` is a Cobra subcommand and gets surfaced automatically;
		// only the six top-level kinds need explicit listing here.
		return cliutil.AllKindNames(), cobra.ShellCompDirectiveNoFileComp
	}
	_ = cmd.RegisterFlagCompletionFunc("epic", cliutil.CompleteEntityIDFlag(entity.KindEpic))
	_ = cmd.RegisterFlagCompletionFunc("tdd", cobra.FixedCompletions(entity.AllowedTDDPolicies(), cobra.ShellCompDirectiveNoFileComp))
	_ = cmd.RegisterFlagCompletionFunc("depends-on", cliutil.CompleteEntityIDFlag(entity.KindMilestone))
	_ = cmd.RegisterFlagCompletionFunc("discovered-in", cliutil.CompleteEntityIDFlag(""))
	_ = cmd.RegisterFlagCompletionFunc("area", cliutil.CompleteAreaFlag())
	_ = cmd.RegisterFlagCompletionFunc("priority", cobra.FixedCompletions(entity.AllowedPriorityLevels(), cobra.ShellCompDirectiveNoFileComp))
	_ = cmd.RegisterFlagCompletionFunc("relates-to", cliutil.CompleteEntityIDFlag(""))
	_ = cmd.RegisterFlagCompletionFunc("linked-adr", cliutil.CompleteEntityIDFlag(entity.KindADR))

	cmd.AddCommand(newACCmd(&titles, &actor, &principal, &root, correlationID))
	return cmd
}

// Run executes `aiwf add <kind>`. Returns one of the cliutil.Exit* codes.
func Run(k entity.Kind, title, actor, principal, root,
	epicID, tddPolicy, dependsOn, discoveredIn, area, priority, pathHint, relatesTo, linkedADRs,
	bindValidator, bindSchema, bindFixtures, bodyFile, bodyText, reason string, fetch, force bool, out cliutil.OutputFormat,
) (code int) {
	// G-0326: --body and --body-file are mutually exclusive ride-along
	// body sources; --force requires a non-empty --reason (mirrors
	// `aiwf promote --force --reason`). Both are pure usage-shape
	// checks, independent of the tree, so they run before any
	// root-resolution or disk work.
	if bodyFile != "" && bodyText != "" {
		cliutil.Errorln("aiwf add: --body and --body-file are mutually exclusive; pass one or the other")
		return cliutil.ExitUsage
	}
	if force && strings.TrimSpace(reason) == "" {
		cliutil.Errorln("aiwf add: --reason \"...\" is required when --force is set (non-empty after trim)")
		return cliutil.ExitUsage
	}

	rootDir, err := cliutil.ResolveRoot(root)
	if err != nil { //coverage:ignore cliutil.ResolveRoot only fails on missing aiwf.yaml + non-existent --root path
		cliutil.Errorf("aiwf add: %v\n", err)
		return cliutil.ExitUsage
	}
	actorStr, err := cliutil.ResolveActor(actor, rootDir)
	if err != nil {
		cliutil.Errorf("aiwf add: %v\n", err)
		return cliutil.ExitUsage
	}

	ctx := context.Background()

	// M-0249: diagnostic-logging wiring, mirroring cancel.Run's own
	// M-0238/AC-5 pattern. entity is unknown at bind time — add
	// allocates the id, it doesn't take one — so this binds with an
	// empty entity field; the JSON envelope's metadata.entity_id
	// (populated on every mutating verb) is where a human cross-
	// references the allocated id against this run_id.
	diagLog, closeDiagLog := cliutil.ResolveLogger(rootDir, os.Getenv)
	defer func() { _ = closeDiagLog() }()
	if diagLog.Enabled(ctx, slog.LevelInfo) {
		runID := out.CorrelationID
		if runID == "" {
			runID = logger.NewRunID()
		}
		diagLog = logger.WithVerb(diagLog, "add", "", actorStr, runID)
	}
	var sha string
	defer func() { cliutil.EmitVerbOutcome(diagLog, "verb", code, sha) }()

	release, rc := cliutil.AcquireRepoLock(rootDir, "aiwf add", out)
	if release == nil {
		return rc
	}
	defer release()

	// M-0214: opt-in best-effort refresh of every remote-tracking ref
	// (git fetch --all) before allocation, so the broadened remote-refs
	// scan computes max against the freshest published view across all
	// branches. A failure (offline, unreachable remote) degrades to
	// local-only allocation with a warning — never blocks the add; a
	// no-remote repo is a clean no-op (git exits 0, no warning). The
	// fetch must land before LoadTreeWithTrunk so the refreshed refs are
	// the ones read into the allocator's view.
	if fetch {
		if ferr := gitops.FetchAll(ctx, rootDir); ferr != nil {
			cliutil.Errorf("aiwf add: --fetch: %v; allocating against the local view\n", ferr)
		}
	}
	tr, _, err := cliutil.LoadTreeWithTrunk(ctx, rootDir)
	if err != nil {
		cliutil.Errorf("aiwf add: loading tree: %v\n", err)
		return cliutil.ExitInternal
	}

	// E-0043 / M-0173: resolve the area write path. An explicit --area is
	// validated at write time against the declared set (root kinds only;
	// kind=milestone is rejected by the verb with a clear flag-vs-kind
	// error, so we skip the member check there and let the verb speak).
	// A gap with --discovered-in and no explicit --area derives its area
	// from the discovered-in entity's effective area — an epic carries it
	// directly, a milestone target is a two-hop derivation through its
	// parent epic (ResolvedAreaByID). Explicit --area always wins.
	resolvedArea := area
	if area != "" {
		if entity.CarriesOwnArea(k) {
			if rc := validateAreaMember(rootDir, area); rc != cliutil.ExitOK {
				return rc
			}
			// AC-5: an explicit --area always wins, but report a --path-hint
			// that unambiguously points elsewhere — a cheap at-add mistag
			// signal — without overriding the explicit choice.
			if pathHint != "" {
				warnAreaHintConflict(rootDir, area, pathHint)
			}
		}
	} else {
		// M-0182: with --area omitted, derive from a single unambiguous
		// --path-hint (root kinds only) through the areamatch SSOT. A hint
		// matching zero or several areas derives nothing (deriveAreaFromHint
		// prints the suggestion); an explicit --area above always wins, so
		// derivation never overwrites it.
		if pathHint != "" {
			if entity.CarriesOwnArea(k) {
				resolvedArea = deriveAreaFromHint(rootDir, pathHint)
			} else {
				// A milestone's area derives from its parent epic, so a path
				// hint has nothing to set. Note it rather than silently
				// ignoring an explicitly-passed flag (the AC-7 principle).
				cliutil.Errorln("aiwf add: --path-hint ignored — a milestone's area derives from its parent epic, not a path hint")
			}
		}
		// E-0043: a gap with --discovered-in derives from the source entity's
		// effective area — a fallback when --path-hint derived nothing.
		if resolvedArea == "" && k == entity.KindGap && discoveredIn != "" {
			resolvedArea = tr.ResolvedAreaByID(discoveredIn)
		}
	}

	// M-0178: under `aiwf.yaml: areas.required: true`, a self-tagging root
	// kind whose resolved area is empty is refused at creation — fail-fast
	// before any entity is written, rather than waiting for the next push's
	// area-required check. A milestone derives its area from its parent epic
	// and is never directly tagged, so it is exempt (the verb already
	// rejects --area on a milestone). A gap whose --discovered-in derived a
	// non-empty area above is unaffected — only a genuinely empty resolved
	// area trips the refusal.
	if resolvedArea == "" && entity.CarriesOwnArea(k) && cliutil.ConfiguredAreaRequired(rootDir) {
		members := cliutil.ConfiguredAreaMembers(rootDir)
		cliutil.Errorf("aiwf add: aiwf.yaml: areas.required is set — %s requires an --area; declared: %s\n", k, strings.Join(members, ", "))
		return cliutil.ExitUsage
	}

	opts := verb.AddOptions{
		EpicID:         epicID,
		TDD:            tddPolicy,
		DiscoveredIn:   discoveredIn,
		Area:           resolvedArea,
		Priority:       priority,
		BindValidator:  bindValidator,
		BindSchema:     bindSchema,
		BindFixtures:   bindFixtures,
		TitleMaxLength: cliutil.ConfiguredTitleMaxLength(rootDir),
		Force:          force,
		Reason:         reason,
	}
	opts.RelatesTo = cliutil.SplitCommaList(relatesTo)
	opts.LinkedADRs = cliutil.SplitCommaList(linkedADRs)
	opts.DependsOn = cliutil.SplitCommaList(dependsOn)

	switch {
	case bodyFile != "":
		body, readErr := cliutil.ReadBodyFile(bodyFile)
		if readErr != nil {
			cliutil.Errorf("aiwf add: %v\n", readErr)
			return cliutil.ExitUsage
		}
		opts.BodyOverride = body
	case bodyText != "":
		opts.BodyOverride = []byte(bodyText)
	}

	if k == entity.KindContract && bindValidator != "" {
		doc, contracts, loadErr := cliutil.LoadContractsDoc(rootDir)
		if loadErr != nil {
			cliutil.Errorf("aiwf add: %v\n", loadErr)
			return cliutil.ExitUsage
		}
		opts.AiwfDoc = doc
		opts.AiwfContracts = contracts
		opts.RepoRoot = rootDir
	}

	result, err := verb.Add(ctx, tr, k, title, actorStr, opts)
	pctx := cliutil.ProvenanceContext{
		Actor:        actorStr,
		Principal:    strings.TrimSpace(principal),
		VerbKind:     verb.VerbCreate,
		CreationRefs: addCreationRefs(k, opts),
	}
	code, sha = cliutil.DecorateAndFinish(ctx, rootDir, "aiwf add", tr, result, err, pctx, out)
	return code
}

// validateAreaMember enforces the M-0173/AC-2 write-time rule: an
// explicit --area must be a member of the declared `aiwf.yaml:
// areas.members` set (the M-0171 accessor — the same closed set the
// M-0172 area-unknown check reads). Returns cliutil.ExitOK when the
// value is declared; otherwise prints a usage error naming the offending
// value (and, when a block exists, the declared set) and returns
// cliutil.ExitUsage so the caller aborts before any entity is created.
// An absent areas block is its own rejection — the field is inert until
// a block is declared (M-0171), so an explicit --area is a usage error.
func validateAreaMember(rootDir, area string) int {
	members := cliutil.ConfiguredAreaMembers(rootDir)
	// The no-block guard stays AHEAD of the value check, deliberately: the
	// `area` field is inert until an areas block is declared (M-0171), so
	// even the reserved `global` sentinel is a usage error with no block
	// (M-0184/AC-4). This is the one write path where global is not a
	// recognized value — every other surface routes through
	// entity.IsValidAreaValue, which accepts global regardless of the
	// declared set.
	if len(members) == 0 {
		cliutil.Errorf("aiwf add: --area %q given but no `areas` block is declared in aiwf.yaml; declare areas.members or omit --area\n", area)
		return cliutil.ExitUsage
	}
	// With a block declared, the reserved `global` sentinel or any declared
	// member is valid — the SSOT predicate, not a parallel `== global`.
	if entity.IsValidAreaValue(area, members) {
		return cliutil.ExitOK
	}
	cliutil.Errorf("aiwf add: --area %q is not a declared area; declared: %s\n", area, strings.Join(members, ", "))
	return cliutil.ExitUsage
}

// deriveAreaFromHint maps a repo-relative --path-hint to a declared area via
// the areamatch SSOT (M-0182). It returns the area name when the hint falls
// under exactly one declared area's `paths:` globs, and "" otherwise — no
// declared paths (inert), no match, or an ambiguous multi-area match; the
// suggestion output for the zero/multi/inert cases lands with AC-6/AC-7. A
// malformed glob (already rejected at config load by areamatch.Validate)
// collapses to "" here too.
func deriveAreaFromHint(rootDir, pathHint string) string {
	areas := configuredAreaPaths(rootDir)
	if len(areas) == 0 {
		// AC-7: no oracle — no declared area carries a paths: glob. Inert, but
		// noted so an explicitly-passed hint does not silently do nothing.
		cliutil.Errorf("aiwf add: --path-hint %q ignored — no declared area has a paths: glob to match against\n", pathHint)
		return ""
	}
	matched, err := areamatch.Derive(areas, normalizeHint(rootDir, pathHint))
	if err != nil { //coverage:ignore unreachable via the public path: area globs are validated at config load (areamatch.Validate via config.Areas.validate), so Derive never sees a malformed glob here; kept as defense-in-depth degrading to no-derivation
		cliutil.Errorf("aiwf add: --path-hint derivation skipped: %v\n", err)
		return ""
	}
	switch len(matched) {
	case 1:
		return matched[0]
	case 0:
		// AC-6: paths are declared, but none claim the hint. Describe the hint
		// outcome ("no area derived"), not the entity's final state — a gap may
		// still be tagged by the --discovered-in fallback in Run.
		cliutil.Errorf("aiwf add: --path-hint %q matches no declared area's paths; no area derived (pass --area to tag explicitly)\n", pathHint)
	default:
		// AC-6: several areas claim the hint — ambiguous, so derive nothing.
		cliutil.Errorf("aiwf add: --path-hint %q is ambiguous (claimed by: %s); no area derived — pass --area to choose\n", pathHint, strings.Join(matched, ", "))
	}
	return ""
}

// warnAreaHintConflict implements AC-5: when both --area and --path-hint are
// given, the explicit --area wins (this never changes resolvedArea), but if the
// hint unambiguously derives a DIFFERENT area, report the disagreement — the
// cheapest possible at-add mistag-prevention signal. Silent when the hint
// agrees, is ambiguous, matches nothing, or has no oracle: the operator chose
// --area deliberately, so only a clear single-area conflict is worth a word.
func warnAreaHintConflict(rootDir, area, pathHint string) {
	matched, err := areamatch.Derive(configuredAreaPaths(rootDir), normalizeHint(rootDir, pathHint))
	if err == nil && len(matched) == 1 && matched[0] != area {
		cliutil.Errorf("aiwf add: note: --area %q overrides --path-hint %q, which points to area %q\n", area, pathHint, matched[0])
	}
}

// normalizeHint makes a user-supplied --path-hint comparable to the declared,
// repo-relative area globs (M-0182, second-review hardening). An absolute path
// under the repo root is made relative to it first — the LLM, this milestone's
// primary user, usually carries absolute paths — then ./, ../, and trailing-
// slash segments are collapsed via path.Clean. The '..' collapse happens
// BEFORE glob matching, so a hint like "projects/app-a/../app-b/x" resolves to
// app-b and cannot lexically derive a confidently-wrong area. A path outside
// the repo relativizes to a "../"-prefixed form that matches no repo-relative
// glob — a correct zero-match.
func normalizeHint(rootDir, pathHint string) string {
	h := pathHint
	if filepath.IsAbs(h) {
		if rel, err := filepath.Rel(rootDir, h); err == nil {
			h = rel
		}
	}
	return path.Clean(filepath.ToSlash(h))
}

// configuredAreaPaths projects the declared members into the name → globs map
// areamatch.Derive consumes, dropping members that declare no `paths:` (they
// offer no oracle to match against).
func configuredAreaPaths(rootDir string) map[string][]string {
	members := cliutil.ConfiguredAreaMembersFull(rootDir)
	areas := make(map[string][]string, len(members))
	for _, m := range members {
		if len(m.Paths) > 0 {
			areas[m.Name] = m.Paths
		}
	}
	return areas
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
	refs = append(refs, opts.DependsOn...)
	_ = k // reserved for future kind-specific ref derivation
	return refs
}

// newACCmd builds `aiwf add ac <milestone-id> --title "..." [--title
// "..."] [--body-file <path>] [--body-file <path>]`. ACs are sub-elements
// (composite id M-NNN/AC-N), not a kind in the schema sense, so they're
// modeled as a child Cobra command. The pointers to the parent's flag
// variables let one --title slice be shared between kinds and ac (a
// typical pattern with cobra child cmds). --body-file is a separate
// repeatable flag local to the ac subcommand: positional pairing with
// --title (the Nth --body-file populates the body of the Nth AC).
func newACCmd(titles *[]string, actor, principal, root *string, correlationID string) *cobra.Command {
	var (
		tests     string
		bodyFiles []string
		out       *cliutil.OutputFormat
	)
	cmd := &cobra.Command{
		Use:   "ac <milestone-id>",
		Short: "Add one or more acceptance criteria to a milestone",
		Example: `  # Add a single AC
  aiwf add ac M-007 --title "rename preserves the entity id"

  # Add multiple ACs in one atomic commit
  aiwf add ac M-007 \
    --title "verb writes exactly one commit" \
    --title "exit codes preserved"

  # Add an AC with body content from a file (M-067)
  aiwf add ac M-007 --title "rename preserves id" --body-file ./ac1-body.md`,
		Args:          cobra.ExactArgs(1),
		SilenceErrors: true,
		SilenceUsage:  true,
		RunE: func(c *cobra.Command, args []string) error {
			return cliutil.WrapExitCode(runAC(args[0], *titles, bodyFiles, *actor, *principal, *root, tests, *out))
		},
	}
	cmd.Flags().StringVar(&tests, "tests", "", `optional test metrics for the seeded red phase (only valid when parent milestone is tdd: required and a single AC is being added); format: "pass=N fail=N skip=N total=N" — keys must be one of pass/fail/skip/total, integers non-negative`)
	cmd.Flags().StringArrayVar(&bodyFiles, "body-file", nil, `path to a file whose content becomes the AC body section under "### AC-N — <title>" (use "-" to read from stdin; only valid with single --title); positionally paired with --title — the Nth --body-file populates the Nth AC; the file must contain body content only — leading "---" is refused`)
	out = cliutil.AddFormatFlags(cmd)
	out.CorrelationID = correlationID
	cmd.ValidArgsFunction = cliutil.CompleteEntityIDArg(entity.KindMilestone, 0)
	return cmd
}

func runAC(parentID string, titles, bodyFiles []string, actor, principal, root, tests string, out cliutil.OutputFormat) int {
	if len(titles) == 0 {
		cliutil.Errorln("aiwf add ac: --title \"...\" is required (pass --title once per AC; repeat for batch)")
		return cliutil.ExitUsage
	}
	// M-067/AC-3: when --body-file is provided at all, per-flag
	// counts must match — the Nth --body-file pairs positionally
	// with the Nth --title. Refuse before file reads, lock, or
	// id allocation so the operator gets a clean usage error.
	if len(bodyFiles) > 0 && len(bodyFiles) != len(titles) {
		cliutil.Errorf(
			"aiwf add ac: got %d titles, %d body files — counts must match "+
				"(positional pairing: the Nth --body-file populates the Nth --title's body; "+
				"equal counts required). To create ACs without bodies, omit --body-file entirely.\n",
			len(titles), len(bodyFiles))
		return cliutil.ExitUsage
	}
	// M-067/AC-5: --body-file - is only valid with a single
	// --title. Stdin is one stream and cannot be split
	// positionally — silently routing it to "the first AC" would
	// surprise the operator. Refuse before reading any --body-file
	// so a piped operator doesn't lose their input.
	if len(titles) > 1 {
		for i, p := range bodyFiles {
			if p == "-" {
				cliutil.Errorf(
					"aiwf add ac: --body-file[%d] -: stdin (--body-file -) is only valid with a single --title (got %d titles); stdin is one stream and cannot be split positionally — use files for multi-AC invocations\n",
					i, len(titles))
				return cliutil.ExitUsage
			}
		}
	}
	metrics, err := cliutil.ParseTestsFlag(tests, "aiwf add ac")
	if err != nil {
		return cliutil.ExitUsage
	}

	var bodies [][]byte
	if len(bodyFiles) > 0 {
		bodies = make([][]byte, len(bodyFiles))
		for i, path := range bodyFiles {
			b, readErr := cliutil.ReadBodyFile(path)
			if readErr != nil {
				cliutil.Errorf("aiwf add ac: --body-file[%d] %s: %v\n", i, path, readErr)
				return cliutil.ExitUsage
			}
			// M-067/AC-4: refuse body files with leading `---`
			// frontmatter — same rule as the whole-entity --body-file
			// path (internal/verb/common.go:validateUserBodyBytes).
			// The AC body is appended after a heading the verb owns,
			// so an embedded frontmatter block would land in the
			// wrong place and silently break document structure.
			trimmed := bytes.TrimLeft(b, " \t\r\n")
			if bytes.HasPrefix(trimmed, []byte("---\n")) || bytes.HasPrefix(trimmed, []byte("---\r\n")) {
				cliutil.Errorf(
					"aiwf add ac: --body-file[%d] %s: body content begins with a frontmatter delimiter (---); pass body content only, not a full markdown file with its own frontmatter\n",
					i, path)
				return cliutil.ExitUsage
			}
			bodies[i] = b
		}
	}

	rootDir, err := cliutil.ResolveRoot(root)
	if err != nil { //coverage:ignore cliutil.ResolveRoot only fails on missing aiwf.yaml + non-existent --root path
		cliutil.Errorf("aiwf add ac: %v\n", err)
		return cliutil.ExitUsage
	}
	actorStr, err := cliutil.ResolveActor(actor, rootDir)
	if err != nil {
		cliutil.Errorf("aiwf add ac: %v\n", err)
		return cliutil.ExitUsage
	}

	release, rc := cliutil.AcquireRepoLock(rootDir, "aiwf add ac", out)
	if release == nil {
		return rc
	}
	defer release()

	ctx := context.Background()
	tr, _, err := tree.Load(ctx, rootDir)
	if err != nil { //coverage:ignore tree.Load errors only on filesystem IO failure (e.g. a permission fault) or context cancellation; malformed entities surface as load findings, not an error here.
		cliutil.Errorf("aiwf add ac: loading tree: %v\n", err)
		return cliutil.ExitInternal
	}
	result, err := verb.AddACBatch(ctx, tr, parentID, titles, bodies, actorStr, metrics)
	// An AC is a sub-element of its parent milestone — its sole
	// "outbound reference" for scope reachability is the parent id.
	pctx := cliutil.ProvenanceContext{
		Actor:        actorStr,
		Principal:    strings.TrimSpace(principal),
		VerbKind:     verb.VerbCreate,
		CreationRefs: []string{parentID},
	}
	code, _ := cliutil.DecorateAndFinish(ctx, rootDir, "aiwf add ac", tr, result, err, pctx, out)
	return code
}
