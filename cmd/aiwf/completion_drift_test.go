package main

import (
	"sort"
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// TestPolicy_FlagsHaveCompletion is the drift-prevention chokepoint
// behind the auto-completion design principle in CLAUDE.md: every
// value-taking flag in the Cobra command tree must either have an
// explicit completion function bound (`RegisterFlagCompletionFunc`)
// or appear in the curated opt-out list below. A flag added without
// completion wiring fails CI here, satisfying M-054 AC-3.
//
// Why the test lives in cmd/aiwf/ rather than internal/policies/
// (where the analogous policies live):
//
// The Cobra command tree is constructed by newRootCmd() and is not
// reachable from a sibling package without a circular import (the
// policies package would have to import cmd/aiwf which itself depends
// on its package-private state). Putting the test here means it
// executes against the actual production tree and stays in step with
// it for free.
//
// Opt-out categories (see optOutFlags below):
//   - Path-shaped: file/directory completion is Cobra's default when
//     no func is bound; explicit opt-out makes that intent visible.
//   - Free-form text: --reason, --title, --tests carry user prose
//     or structured tokens with no closed set worth enumerating.
//   - Identity: --actor / --principal are role/email-shaped strings;
//     git-config-derived completion is YAGNI for the PoC.
//   - Reserved: --scope on render is documented as not-yet-implemented.
//
// Boolean flags are auto-skipped (no value to suggest); the test does
// not check them.
func TestPolicy_FlagsHaveCompletion(t *testing.T) {
	root := newRootCmd()

	// optOutFlags name (cmd-path, flag-name) pairs that intentionally
	// have no completion function registered. The cmd-path side is
	// optional: an empty string opts the flag out across every command
	// where it appears (the more common case for shared flags like
	// --root and --actor).
	optOutFlags := map[flagKey]string{
		// Path-shaped (file completion is the default Cobra behavior).
		{flag: "root"}:                         "consumer repo path; default file completion is correct",
		{flag: "body-file"}:                    "filesystem path with stdin sentinel '-'",
		{flag: "out"}:                          "filesystem directory path",
		{flag: "from"}:                         "filesystem path",
		{flag: "validator"}:                    "validator name; closed set is not yet authoritative",
		{flag: "schema"}:                       "filesystem path",
		{flag: "fixtures"}:                     "filesystem directory path",
		{cmd: "aiwf upgrade", flag: "version"}: "semver tag string; no closed set",

		// Free-form text.
		{flag: "reason"}: "free-form prose",
		{flag: "title"}:  "free-form entity title",
		{flag: "tests"}:  "structured 'pass=N fail=N skip=N total=N' grammar; no closed set",
		{flag: "since"}:  "git ref string",

		// Commit SHAs (no closed set; user enumerates via git log).
		{flag: "by-commit"}: "comma-separated commit SHAs; no closed set worth enumerating",

		// Identity (could complete from git history, but YAGNI for the PoC).
		{flag: "actor"}:     "role/identifier; free-form identity string",
		{flag: "principal"}: "human/<id>; free-form identity string",

		// Authorize verb-specific flags (passthrough-shaped today; will
		// migrate as part of the broader Cobra adoption — until then
		// they don't have RegisterFlagCompletionFunc bindings).
		{cmd: "aiwf authorize", flag: "to"}:     "agent name; closed set is principal × agent × scope, not yet enumerated",
		{cmd: "aiwf authorize", flag: "pause"}:  "free-form pause-reason text",
		{cmd: "aiwf authorize", flag: "resume"}: "free-form resume-reason text",

		// Render's reserved flag is documented as not-yet-implemented.
		{cmd: "aiwf render", flag: "scope"}: "reserved (not yet implemented)",

		// `aiwf completion` is Cobra-generated; its --no-descriptions
		// flag is internal to the completion-script generator.
		{cmd: "aiwf completion bash", flag: "no-descriptions"}:       "Cobra completion script generator; out of E-14 scope",
		{cmd: "aiwf completion zsh", flag: "no-descriptions"}:        "Cobra completion script generator; out of E-14 scope",
		{cmd: "aiwf completion fish", flag: "no-descriptions"}:       "Cobra completion script generator; out of E-14 scope",
		{cmd: "aiwf completion powershell", flag: "no-descriptions"}: "Cobra completion script generator; out of E-14 scope",
	}

	var failures []string
	walkCommands(root, func(cmd *cobra.Command) {
		cmd.Flags().VisitAll(func(f *pflag.Flag) {
			if f.Value.Type() == "bool" {
				return
			}
			if _, ok := optOutFlags[flagKey{cmd: cmd.CommandPath(), flag: f.Name}]; ok {
				return
			}
			if _, ok := optOutFlags[flagKey{flag: f.Name}]; ok {
				return
			}
			// Cobra resolves completion funcs from the command's local
			// map; inherited persistent flags fall back to the parent.
			if hasFlagCompletion(cmd, f.Name) {
				return
			}
			failures = append(failures, cmd.CommandPath()+" --"+f.Name)
		})
	})

	if len(failures) > 0 {
		sort.Strings(failures)
		t.Errorf("flags missing completion wiring (E-14 / M-054):\n  %s\n\n"+
			"Either bind a completion function via cmd.RegisterFlagCompletionFunc(...)\n"+
			"(static via cobra.FixedCompletions, dynamic via completeEntityIDFlag),\n"+
			"or add an entry to optOutFlags in completion_drift_test.go with a one-line rationale.",
			joinFailures(failures))
	}
}

// TestPolicy_PositionalsHaveCompletion mirrors the flag-completion
// drift test for positional args (verbs that accept ids/kinds via
// `<id>` rather than `--id`). Every command whose Args validator
// permits positionals must declare a ValidArgsFunction or appear in
// the opt-out list. M-054 AC-4: dynamic-completion-required cases
// covered.
func TestPolicy_PositionalsHaveCompletion(t *testing.T) {
	root := newRootCmd()

	// Commands intentionally without a ValidArgsFunction. The cmd-path
	// is the full Cobra command path (`aiwf <verb> [<sub>]`).
	optOutPositional := map[string]string{
		"aiwf":                       "root command; no positional args of its own",
		"aiwf check":                 "no positional args",
		"aiwf doctor":                "no positional args",
		"aiwf init":                  "no positional args",
		"aiwf update":                "no positional args",
		"aiwf upgrade":               "no positional args",
		"aiwf version":               "no positional args",
		"aiwf render":                "no positional args (subcommand or --format=html)",
		"aiwf render roadmap":        "no positional args",
		"aiwf render help":           "hidden help alias; no positional args",
		"aiwf import":                "<manifest> is a filesystem path; default file completion is correct",
		"aiwf completion":            "Cobra completion script generator; out of E-14 scope",
		"aiwf completion bash":       "Cobra completion script generator; out of E-14 scope",
		"aiwf completion zsh":        "Cobra completion script generator; out of E-14 scope",
		"aiwf completion fish":       "Cobra completion script generator; out of E-14 scope",
		"aiwf completion powershell": "Cobra completion script generator; out of E-14 scope",
		"aiwf help":                  "Cobra-default help command; positional is the verb name (auto-completed)",

		// Verbs not yet migrated to native Cobra (still passthrough-shaped):
		// these do not have ValidArgsFunction set because their argument
		// parsing happens inside the legacy handler. They will gain
		// completion when they migrate.
		"aiwf show":      "passthrough verb; not yet migrated to native Cobra",
		"aiwf status":    "passthrough verb; not yet migrated to native Cobra",
		"aiwf whoami":    "passthrough verb; not yet migrated to native Cobra",
		"aiwf authorize": "passthrough verb; not yet migrated to native Cobra",
		"aiwf contract":  "passthrough verb; not yet migrated to native Cobra",
	}

	var failures []string
	walkCommands(root, func(cmd *cobra.Command) {
		// Skip if this command has subcommands and no Args validator
		// — Cobra dispatches to children, args don't apply.
		if !cmd.Runnable() && cmd.HasSubCommands() {
			return
		}
		if _, ok := optOutPositional[cmd.CommandPath()]; ok {
			return
		}
		if cmd.ValidArgsFunction == nil && len(cmd.ValidArgs) == 0 {
			failures = append(failures, cmd.CommandPath())
		}
	})

	if len(failures) > 0 {
		sort.Strings(failures)
		t.Errorf("commands with positional args but no ValidArgsFunction / ValidArgs (E-14 / M-054):\n  %s\n\n"+
			"Either set cmd.ValidArgsFunction (static or dynamic), or add an entry\n"+
			"to optOutPositional in completion_drift_test.go with a one-line rationale.",
			joinFailures(failures))
	}
}

// flagKey selects an entry in optOutFlags. cmd is the full Cobra
// command path (`aiwf <verb> [<sub>]`); empty matches any command.
type flagKey struct {
	cmd  string
	flag string
}

// hasFlagCompletion reports whether cmd has a completion function
// registered for the named flag. Cobra exposes GetFlagCompletionFunc
// on the *Command (returns (func, bool)) since v1.10; the wrapper
// keeps the call-site readable.
func hasFlagCompletion(cmd *cobra.Command, name string) bool {
	_, ok := cmd.GetFlagCompletionFunc(name)
	return ok
}

// walkCommands invokes fn against every command in the tree rooted
// at root, including root itself. Order is depth-first preorder.
func walkCommands(root *cobra.Command, fn func(*cobra.Command)) {
	fn(root)
	for _, child := range root.Commands() {
		walkCommands(child, fn)
	}
}

// joinFailures joins failure entries with the indentation the
// surrounding Errorf format expects, for cleaner test output.
func joinFailures(failures []string) string {
	var s string
	for i, f := range failures {
		if i > 0 {
			s += "\n  "
		}
		s += f
	}
	return s
}
