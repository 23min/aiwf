// Package authorize implements the `aiwf authorize ` verb (per-verb subpackage of M-0116;
// cmd/aiwf/main.go's newRootCmd wires it via NewCmd).
package authorize

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"

	"github.com/23min/aiwf/internal/branchparse"
	"github.com/23min/aiwf/internal/cli/cliutil"
	"github.com/23min/aiwf/internal/tree"
	"github.com/23min/aiwf/internal/verb"
)

// NewCmd builds `aiwf authorize <id>` in its three modes:
//
//	aiwf authorize <id> --to <agent> [--reason "..."] [--force]
//	aiwf authorize <id> --pause "<reason>"
//	aiwf authorize <id> --resume "<reason>"
//
// Exactly one of --to / --pause / --resume must be set. The dispatcher
// resolves the actor (must be human/), locks the repo, loads the
// existing scopes for the entity from git log (so --pause / --resume
// can find their target), and hands off to verb.Authorize.
//
// Per docs/pocv3/design/provenance-model.md §"The aiwf authorize verb"
// the `--reason` argument has a different role per mode:
//   - --to: optional; the rationale the human writes when opening a scope.
//   - --pause / --resume: required; the argument to the flag is itself
//     the reason (e.g. `--pause "blocked by E-09"`). Passing both
//     `--pause "..."` and `--reason "..."` is a usage error.
func NewCmd() *cobra.Command {
	var (
		actor  string
		root   string
		to     string
		pause  string
		resume string
		reason string
		branch string
		force  bool
		out    *cliutil.OutputFormat
	)
	cmd := &cobra.Command{
		Use:   "authorize <id>",
		Short: "Open / pause / resume an autonomous-work scope on an entity",
		Example: `  # Delegate autonomous work on an epic to ai/claude
  aiwf authorize E-14 --to ai/claude --reason "delegated cobra+completion epic"

  # Pause the most-recently-opened active scope
  aiwf authorize E-14 --pause "blocked on review feedback"

  # Resume the most-recently-paused scope
  aiwf authorize E-14 --resume "review feedback addressed"`,
		Args:          cobra.ExactArgs(1),
		SilenceErrors: true,
		SilenceUsage:  true,
		RunE: func(c *cobra.Command, args []string) error {
			return cliutil.WrapExitCode(Run(args[0], actor, root, to, pause, resume, reason, branch, force, *out))
		},
	}
	cmd.Flags().StringVar(&actor, "actor", "", "actor for the commit trailer (default: derived from git config user.email; must be human/...)")
	cmd.Flags().StringVar(&root, "root", "", "consumer repo root")
	cmd.Flags().StringVar(&to, "to", "", "agent the scope authorizes (e.g. ai/claude); opens a new scope")
	cmd.Flags().StringVar(&pause, "pause", "", "pause the most-recently-opened active scope on <id>; the argument is the reason")
	cmd.Flags().StringVar(&resume, "resume", "", "resume the most-recently-paused scope on <id>; the argument is the reason")
	cmd.Flags().StringVar(&reason, "reason", "", "rationale text for --to (optional) / --force (required); ignored by --pause and --resume (their argument is the reason)")
	cmd.Flags().StringVar(&branch, "branch", "", "ritual branch the scope is bound to (ADR-0010); when set, the authorize commit carries an aiwf-branch: trailer with this value. From `main` or a ritual-shape current branch (epic/milestone/patch), naming a ritual-shape future branch is accepted — the step-7 pattern of aiwfx-start-epic (M-0104/AC-4) or step-4 of aiwfx-start-milestone (M-0105/AC-6). The named branch is cut by a later step of the ritual.")
	cmd.Flags().BoolVar(&force, "force", false, "open a fresh scope on a terminal scope-entity (requires --reason)")
	out = cliutil.AddFormatFlags(cmd)
	cmd.ValidArgsFunction = cliutil.CompleteEntityIDArg("", 0)
	// M-0102 / AC-6: --branch completion returns local branch names
	// matching the ADR-0010 ritual shape — epic/E-NNNN-..., milestone/
	// M-NNNN-..., patch/[Gg]-NNNN-... — filtered via
	// branchparse.ParseEntityFromBranch. Non-ritual branches (main,
	// fix/*, chore/*, etc.) are deliberately omitted so completion is
	// the discoverability surface for the convention itself.
	_ = cmd.RegisterFlagCompletionFunc("branch", completeBranchFlag)
	return cmd
}

// completeBranchFlag is the cobra completion function for --branch.
// Runs against the user's current working directory at completion
// time. Best-effort: any git failure collapses to an empty list so
// the shell falls through to its default (no suggestions).
func completeBranchFlag(_ *cobra.Command, _ []string, _ string) ([]string, cobra.ShellCompDirective) {
	return ritualLocalBranches("."), cobra.ShellCompDirectiveNoFileComp
}

// ritualLocalBranches returns local branch names from rootDir whose
// shape matches the ADR-0010 ritual grammar via
// branchparse.ParseEntityFromBranch. Returns nil on git failure or
// when the repo has no matching branches.
func ritualLocalBranches(rootDir string) []string {
	cmd := exec.Command("git", "for-each-ref", "refs/heads/", "--format=%(refname:short)")
	cmd.Dir = rootDir
	out, err := cmd.Output()
	if err != nil {
		return nil
	}
	var ritual []string
	for _, line := range strings.Split(string(out), "\n") {
		name := strings.TrimSpace(line)
		if name == "" {
			continue
		}
		if branchparse.ParseEntityFromBranch(name) != "" {
			ritual = append(ritual, name)
		}
	}
	return ritual
}

// currentBranch returns the short name of the currently-checked-out
// branch in rootDir, or empty if HEAD is detached, not a branch, or
// git fails. Used by the M-0103 AI-target preflight in verb.Authorize
// to detect implicit ritual-branch context.
func currentBranch(rootDir string) string {
	cmd := exec.Command("git", "symbolic-ref", "--short", "HEAD")
	cmd.Dir = rootDir
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

// branchExists reports whether refs/heads/<branch> resolves in
// rootDir's repo. Used by the M-0103 AI-target preflight to
// distinguish "no --branch passed" from "--branch <name> typo'd".
// Empty branch always reports false (no name to check).
func branchExists(rootDir, branch string) bool {
	if strings.TrimSpace(branch) == "" {
		return false
	}
	cmd := exec.Command("git", "show-ref", "--verify", "--quiet", "refs/heads/"+branch)
	cmd.Dir = rootDir
	return cmd.Run() == nil
}

// branchTipSHA returns the 40-char lowercase hex SHA of the
// branch's current tip via `git rev-parse refs/heads/<branch>`
// (M-0161/AC-6 / G-0206). Caller MUST gate on branchExists
// — calling with an unresolvable ref returns "" silently, but
// the verb-side TrailerBranchSHA validator would reject any
// malformed value, so an empty return here keeps the trailer
// absent. The trailer staying absent is the future-branch
// carve-out's correct behavior.
func branchTipSHA(rootDir, branch string) string {
	if strings.TrimSpace(branch) == "" {
		return ""
	}
	cmd := exec.Command("git", "rev-parse", "refs/heads/"+branch)
	cmd.Dir = rootDir
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

// Run executes `aiwf authorize`. Returns one of the cliutil.Exit* codes.
func Run(id, actor, root, to, pause, resume, reason, branch string, force bool, out cliutil.OutputFormat) int {
	modes := 0
	if to != "" {
		modes++
	}
	if pause != "" {
		modes++
	}
	if resume != "" {
		modes++
	}
	if modes != 1 {
		fmt.Fprintln(os.Stderr, "aiwf authorize: pick exactly one of --to <agent>, --pause \"<reason>\", or --resume \"<reason>\"")
		return cliutil.ExitUsage
	}
	// `--reason` is meaningful only with --to (and --to --force). For
	// --pause / --resume the flag value IS the reason; a separate
	// --reason would be ambiguous.
	if (pause != "" || resume != "") && reason != "" {
		fmt.Fprintln(os.Stderr, "aiwf authorize: --reason is not used with --pause / --resume; the argument to --pause/--resume is itself the reason")
		return cliutil.ExitUsage
	}
	if force && to == "" {
		fmt.Fprintln(os.Stderr, "aiwf authorize: --force is only meaningful with --to (overrides terminal-scope-entity refusal)")
		return cliutil.ExitUsage
	}
	// M-0102: --branch binds a fresh scope to a ritual branch; reusing
	// an existing scope (--pause / --resume) inherits the original
	// binding. Silently ignoring --branch in those modes is a usability
	// footgun, so refuse the combination upfront — matches the
	// existing --reason + --pause/--resume gate above.
	if (pause != "" || resume != "") && branch != "" {
		fmt.Fprintln(os.Stderr, "aiwf authorize: --branch is only meaningful with --to (binds a fresh scope to a ritual branch); --pause / --resume reuse the opening scope's branch")
		return cliutil.ExitUsage
	}
	if force && strings.TrimSpace(reason) == "" {
		fmt.Fprintln(os.Stderr, "aiwf authorize: --force requires --reason \"...\" (non-empty after trim)")
		return cliutil.ExitUsage
	}

	rootDir, err := cliutil.ResolveRoot(root)
	if err != nil {
		fmt.Fprintf(os.Stderr, "aiwf authorize: %v\n", err)
		return cliutil.ExitUsage
	}
	actorStr, err := cliutil.ResolveActor(actor, rootDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "aiwf authorize: %v\n", err)
		return cliutil.ExitUsage
	}

	release, rc := cliutil.AcquireRepoLock(rootDir, "aiwf authorize")
	if release == nil {
		return rc
	}
	defer release()

	ctx := context.Background()
	tr, _, err := tree.Load(ctx, rootDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "aiwf authorize: loading tree: %v\n", err)
		return cliutil.ExitInternal
	}

	opts := verb.AuthorizeOptions{}
	// M-0103 structural invariant: opts.Agent is populated ONLY in the
	// `case to != ""` arm (i.e., only for AuthorizeOpen). Pause and
	// resume modes never carry an Agent value here, which is the second
	// of the two gates protecting pause/resume from the AI-target
	// preflight (the first being the preflight's location inside
	// authorizeOpen). A refactor that filled opts.Agent for pause/
	// resume — e.g., to thread scope context into transitional
	// commits — would, in combination with a verb-side leak of the
	// preflight to non-Open modes, regress AC-7. The combined
	// regression is caught by the AC-7 cli-seam test
	// (TestRunAuthorize_PauseResume_NonRitualBranch_Accepts).
	switch {
	case to != "":
		opts.Mode = verb.AuthorizeOpen
		opts.Agent = to
		opts.Reason = reason
		opts.Branch = branch
		opts.Force = force
		// M-0103 preflight inputs: only the AuthorizeOpen path on an
		// ai/* target consumes them; --pause / --resume modes ignore
		// them entirely. Computed once per invocation; if git fails
		// the verb interprets the empty CurrentBranch + false
		// BranchExists as "no ritual context detected" and refuses
		// when the gate fires.
		opts.CurrentBranch = currentBranch(rootDir)
		opts.BranchExists = branchExists(rootDir, branch)
		// M-0161/AC-6 (G-0206): plumb the bound branch's tip SHA
		// so the verb can record aiwf-branch-sha:. Resolved iff
		// Branch exists (BranchExists path); empty for the
		// future-branch carve-out keeps the trailer absent and
		// preserves the existing name-only behavior for that
		// case.
		if opts.BranchExists {
			opts.BranchSHA = branchTipSHA(rootDir, branch)
		}
		// M-0161/AC-1 (G-0200): plumb the configured trunk
		// short-name into the verb so the carve-out's "trunk +
		// ritual --branch" predicate honors the configured trunk
		// rather than the literal "main". cliutil reads aiwf.yaml
		// and derives via Config.TrunkBranchShortName().
		opts.TrunkShort = cliutil.ConfiguredTrunkBranchShortName(rootDir)
	case pause != "":
		opts.Mode = verb.AuthorizePause
		opts.Reason = pause
	case resume != "":
		opts.Mode = verb.AuthorizeResume
		opts.Reason = resume
	}
	if opts.Mode == verb.AuthorizePause || opts.Mode == verb.AuthorizeResume {
		scopes, scopesErr := cliutil.LoadEntityScopes(ctx, rootDir, id)
		if scopesErr != nil {
			fmt.Fprintf(os.Stderr, "aiwf authorize: %v\n", scopesErr)
			return cliutil.ExitInternal
		}
		opts.Scopes = scopes
	}

	result, vErr := verb.Authorize(ctx, tr, id, actorStr, opts)
	return cliutil.FinishVerb(ctx, rootDir, "aiwf authorize", result, vErr, out)
}
