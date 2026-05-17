// Package add implements the `aiwf add` verb and its `aiwf add ac`
// subcommand (per-verb subpackage of M-0115; cmd/aiwf/main.go's
// newRootCmd wires it via NewCmd). Both verbs share the package so
// the Cobra subcommand wiring (`add ac` as a child of `add`) and the
// PersistentFlag-sharing pattern remain intact.
package add

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/23min/aiwf/internal/cli/cliutil"
	"github.com/23min/aiwf/internal/entity"
	"github.com/23min/aiwf/internal/tree"
	"github.com/23min/aiwf/internal/verb"
)

// NewCmd builds `aiwf add <kind> --title "..." [kind-specific flags]`
// and the `aiwf add ac <milestone-id> --title "..."` sub-shape. ACs are
// modeled as a Cobra subcommand of add (matching their composite-id
// status as sub-elements of a milestone, not a kind in the schema
// sense). For the six top-level kinds, args[0] is the kind and the
// runtime validates kind-vs-flag relevance — same shape as pre-Cobra.
func NewCmd() *cobra.Command {
	var (
		titles        []string
		actor         string
		principal     string
		root          string
		epicID        string
		tddPolicy     string
		dependsOn     string
		discoveredIn  string
		relatesTo     string
		linkedADRs    string
		bindValidator string
		bindSchema    string
		bindFixtures  string
		bodyFile      string
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
				fmt.Fprintf(os.Stderr, "aiwf add: unexpected args after kind %q: %v\n", args[0], args[1:])
				return cliutil.WrapExitCode(cliutil.ExitUsage)
			}
			kindArg := args[0]
			k, ok := cliutil.ParseKind(kindArg)
			if !ok {
				fmt.Fprintf(os.Stderr, "aiwf add: unknown kind %q\n", kindArg)
				return cliutil.WrapExitCode(cliutil.ExitUsage)
			}
			if len(titles) > 1 {
				fmt.Fprintf(os.Stderr, "aiwf add: --title may not be repeated for kind %q (only `aiwf add ac` accepts a repeated --title for batched creation)\n", kindArg)
				return cliutil.WrapExitCode(cliutil.ExitUsage)
			}
			title := ""
			if len(titles) == 1 {
				title = titles[0]
			}
			return cliutil.WrapExitCode(Run(k, title, actor, principal, root,
				epicID, tddPolicy, dependsOn, discoveredIn, relatesTo, linkedADRs,
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
	cmd.Flags().StringVar(&tddPolicy, "tdd", "", "milestone TDD policy: required|advisory|none — required at creation time for kind=milestone (G-055 layer 1)")
	cmd.Flags().StringVar(&dependsOn, "depends-on", "", "comma-separated milestone ids the new milestone depends on (milestone only); each id must resolve to an existing milestone (M-076)")
	cmd.Flags().StringVar(&discoveredIn, "discovered-in", "", "id of milestone or epic where the gap was discovered (gap only)")
	cmd.Flags().StringVar(&relatesTo, "relates-to", "", "comma-separated ids the decision relates to (decision only)")
	cmd.Flags().StringVar(&linkedADRs, "linked-adr", "", "comma-separated ADR ids motivating the contract (contract only)")
	cmd.Flags().StringVar(&bindValidator, "validator", "", "validator name (contract only; if set, --schema and --fixtures are also required and the binding is added atomically)")
	cmd.Flags().StringVar(&bindSchema, "schema", "", "repo-relative path to the schema (contract only; pairs with --validator and --fixtures)")
	cmd.Flags().StringVar(&bindFixtures, "fixtures", "", "repo-relative path to the fixtures-tree root (contract only; pairs with --validator and --schema)")
	cmd.Flags().StringVar(&bodyFile, "body-file", "", `path to a file whose content becomes the entity body, in the same atomic commit as the frontmatter (use "-" to read from stdin); replaces the per-kind default template; the file must contain body content only — leading "---" is refused`)

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
	_ = cmd.RegisterFlagCompletionFunc("relates-to", cliutil.CompleteEntityIDFlag(""))
	_ = cmd.RegisterFlagCompletionFunc("linked-adr", cliutil.CompleteEntityIDFlag(entity.KindADR))

	cmd.AddCommand(newACCmd(&titles, &actor, &principal, &root))
	return cmd
}

// Run executes `aiwf add <kind>`. Returns one of the cliutil.Exit* codes.
func Run(k entity.Kind, title, actor, principal, root,
	epicID, tddPolicy, dependsOn, discoveredIn, relatesTo, linkedADRs,
	bindValidator, bindSchema, bindFixtures, bodyFile string,
) int {
	rootDir, err := cliutil.ResolveRoot(root)
	if err != nil {
		fmt.Fprintf(os.Stderr, "aiwf add: %v\n", err)
		return cliutil.ExitUsage
	}
	actorStr, err := cliutil.ResolveActor(actor, rootDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "aiwf add: %v\n", err)
		return cliutil.ExitUsage
	}

	release, rc := cliutil.AcquireRepoLock(rootDir, "aiwf add")
	if release == nil {
		return rc
	}
	defer release()

	ctx := context.Background()
	tr, _, err := cliutil.LoadTreeWithTrunk(ctx, rootDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "aiwf add: loading tree: %v\n", err)
		return cliutil.ExitInternal
	}

	opts := verb.AddOptions{
		EpicID:         epicID,
		TDD:            tddPolicy,
		DiscoveredIn:   discoveredIn,
		BindValidator:  bindValidator,
		BindSchema:     bindSchema,
		BindFixtures:   bindFixtures,
		TitleMaxLength: cliutil.ConfiguredTitleMaxLength(rootDir),
	}
	opts.RelatesTo = cliutil.SplitCommaList(relatesTo)
	opts.LinkedADRs = cliutil.SplitCommaList(linkedADRs)
	opts.DependsOn = cliutil.SplitCommaList(dependsOn)

	if bodyFile != "" {
		body, readErr := cliutil.ReadBodyFile(bodyFile)
		if readErr != nil {
			fmt.Fprintf(os.Stderr, "aiwf add: %v\n", readErr)
			return cliutil.ExitUsage
		}
		opts.BodyOverride = body
	}

	if k == entity.KindContract && bindValidator != "" {
		doc, contracts, loadErr := cliutil.LoadContractsDoc(rootDir)
		if loadErr != nil {
			fmt.Fprintf(os.Stderr, "aiwf add: %v\n", loadErr)
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
	return cliutil.DecorateAndFinish(ctx, rootDir, "aiwf add", tr, result, err, pctx)
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
func newACCmd(titles *[]string, actor, principal, root *string) *cobra.Command {
	var (
		tests     string
		bodyFiles []string
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
			return cliutil.WrapExitCode(runAC(args[0], *titles, bodyFiles, *actor, *principal, *root, tests))
		},
	}
	cmd.Flags().StringVar(&tests, "tests", "", `optional test metrics for the seeded red phase (only valid when parent milestone is tdd: required and a single AC is being added); format: "pass=N fail=N skip=N total=N" — keys must be one of pass/fail/skip/total, integers non-negative`)
	cmd.Flags().StringArrayVar(&bodyFiles, "body-file", nil, `path to a file whose content becomes the AC body section under "### AC-N — <title>" (use "-" to read from stdin; only valid with single --title); positionally paired with --title — the Nth --body-file populates the Nth AC; the file must contain body content only — leading "---" is refused`)
	cmd.ValidArgsFunction = cliutil.CompleteEntityIDArg(entity.KindMilestone, 0)
	return cmd
}

func runAC(parentID string, titles, bodyFiles []string, actor, principal, root, tests string) int {
	if len(titles) == 0 {
		fmt.Fprintln(os.Stderr, "aiwf add ac: --title \"...\" is required (pass --title once per AC; repeat for batch)")
		return cliutil.ExitUsage
	}
	// M-067/AC-3: when --body-file is provided at all, per-flag
	// counts must match — the Nth --body-file pairs positionally
	// with the Nth --title. Refuse before file reads, lock, or
	// id allocation so the operator gets a clean usage error.
	if len(bodyFiles) > 0 && len(bodyFiles) != len(titles) {
		fmt.Fprintf(os.Stderr,
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
				fmt.Fprintf(os.Stderr,
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
				fmt.Fprintf(os.Stderr, "aiwf add ac: --body-file[%d] %s: %v\n", i, path, readErr)
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
				fmt.Fprintf(os.Stderr,
					"aiwf add ac: --body-file[%d] %s: body content begins with a frontmatter delimiter (---); pass body content only, not a full markdown file with its own frontmatter\n",
					i, path)
				return cliutil.ExitUsage
			}
			bodies[i] = b
		}
	}

	rootDir, err := cliutil.ResolveRoot(root)
	if err != nil {
		fmt.Fprintf(os.Stderr, "aiwf add ac: %v\n", err)
		return cliutil.ExitUsage
	}
	actorStr, err := cliutil.ResolveActor(actor, rootDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "aiwf add ac: %v\n", err)
		return cliutil.ExitUsage
	}

	release, rc := cliutil.AcquireRepoLock(rootDir, "aiwf add ac")
	if release == nil {
		return rc
	}
	defer release()

	ctx := context.Background()
	tr, _, err := tree.Load(ctx, rootDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "aiwf add ac: loading tree: %v\n", err)
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
	return cliutil.DecorateAndFinish(ctx, rootDir, "aiwf add ac", tr, result, err, pctx)
}
